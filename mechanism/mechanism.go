package mech

import (
	"errors"
	"reflect"
	"sync/atomic"

	"github.com/netrack/netrack/database"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism/injector"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
)

var (
	// ErrMechanismNotRegistered is returned on
	// not registered mechanism operations.
	ErrMechanismNotRegistered = errors.New(
		"MechanismManager: mechanism not registered")

	// ErrMechanismAlreadyEnabled is returned on enabling
	// of already enabled mechanism.
	ErrMechanismAlreadyEnabled = errors.New(
		"MechanismManager: mechanism already enabled")

	// ErrMechanismAlreadyActivated is returned on
	// activating of already activated mechanism.
	ErrMechanismAlreadyActivated = errors.New(
		"MechanismManager: mechanism already activated")

	// ErrMechanismAlreadyDisabled is returned on
	// disabling of already disabled mechanism.
	ErrMechanismAlreadyDisabled = errors.New(
		"MechanismManager: mechanism already disabled")
)

// Proto is protocol string alias
type Proto int

// MechanismContext is a context, that shared among mechanisms
// enabled for a particular device. It is a placeholder for
// mechanism driver context and mechanism drivers for a single switch.
type MechanismContext struct {
	// OpenFlow switch instance.
	Switch Switch

	// Pipe to connect mechanism drivers.
	Func rpc.ProcCaller

	// OpenFlow multiplexer handler.
	Mux *of.ServeMux

	// Route mechanism manager.
	Routing RouteMechanismManager

	// Extention mechanism manager.
	Extension *ExtensionMechanismManager

	Managers injector.Injector
}

// Mechanism describes switch drivers
type Mechanism interface {
	// Enable performs mechanism initialization.
	Enable(*MechanismContext)

	// Enabled returns true if Enable called before.
	Enabled() bool

	// Activate called after switch handshake procedure
	// completion.
	Activate()

	// Activated returns true if Activate called before.
	Activated() bool

	// Disable removes installed flows from the switch
	// and performs all other necessary clean-up operations.
	Disable()
}

// BaseMechanism implements thread-safe methods of
// Mechanism interface.
type BaseMechanism struct {
	enabled   int64
	activated int64

	// MechanismContext instance
	C *MechanismContext
}

// Enable implements Mechanism interface.
func (m *BaseMechanism) Enable(c *MechanismContext) {
	m.C = c
	atomic.CompareAndSwapInt64(&m.enabled, 0, 1)
}

// Enabled implements Mechanism interface.
func (m *BaseMechanism) Enabled() bool {
	return atomic.LoadInt64(&m.enabled) == 1
}

// Active implements Mechanism interface.
func (m *BaseMechanism) Activate() {
	atomic.CompareAndSwapInt64(&m.activated, 0, 1)
}

// Activated implements Mechanism interface.
func (m *BaseMechanism) Activated() bool {
	return atomic.LoadInt64(&m.activated) == 1
}

// Disable implements Mechanism interface.
func (m *BaseMechanism) Disable() {
	atomic.StoreInt64(&m.enabled, 0)
}

// MechanismMap describes map for mechanism type.
type MechanismMap interface {
	// Get returns Mechanism by registered name.
	Get(string) (Mechanism, bool)

	// Set saves Mechanism under specified name.
	Set(string, Mechanism)

	// Iter call specified function for each element of map.
	Iter(func(string, Mechanism) bool)
}

// BaseMechanismManager manages networking
// mechanisms using drivers.
type MechanismManager interface {
	// Mechanism returns registered mechanism by specified name.
	Mechanism(string, Mechanism) error

	// Enable performs initialization of registered mechanisms.
	Enable(*MechanismContext)

	// EnableByName performs intialization of specified mechanism.
	EnableByName(string, *MechanismContext) error

	// Activate activates registered mechanisms.
	Activate()

	// ActivateByName activates scpecified mechanism.
	ActivateByName(string) error

	// Disable releases resources for registered mechanisms.
	Disable()

	// DisableByName releases resources of specified mechanism.
	DisableByName(string) error
}

// BaseMechanismManager implements MechanismManager interface.
type BaseMechanismManager struct {
	Datapath string

	Mechanisms MechanismMap

	enabled   int64
	activated int64
}

func (m *BaseMechanismManager) Mechanism(name string, mech Mechanism) (err error) {
	mechanism, ok := m.Mechanisms.Get(name)
	if !ok {
		log.ErrorLog("mechanism/MECHANISM",
			"Failed to find requested mechanism")
		return ErrMechanismNotRegistered
	}

	defer func() {
		recovered := recover()
		if recovered == nil {
			return
		}

		err = ErrMechanismNotRegistered

		if recoveredErr, ok := recovered.(error); ok {
			err = recoveredErr
		}
	}()

	firstValue := reflect.ValueOf(mech)
	// Receiver should be a pointer and can be changed.
	if firstValue.Kind() != reflect.Ptr || !firstValue.Elem().CanSet() {
		return ErrMechanismNotRegistered
	}

	secondValue := reflect.ValueOf(mechanism)
	// Values shoud be the same type.
	if firstValue.Type() != secondValue.Type() {
		return ErrMechanismNotRegistered
	}

	firstValue.Elem().Set(secondValue.Elem())
	return nil
}

func (m *BaseMechanismManager) Context(model db.Model, context interface{}) error {
	err := db.Read(model, m.Datapath, context)
	if err != nil {
		log.ErrorLog("mechanism/CONTEXT",
			"Failed to read mechanism configuration from the database: ", err)
		return err
	}

	return nil
}

func (m *BaseMechanismManager) Create(model db.Model, context interface{}, fn func() error) (err error) {
	err = db.Transaction(func(p db.ModelPersister) error {
		// Lock to make consistent configuration
		err := p.Lock(model, m.Datapath, context)

		// Create a new record in a dabase for a new switch
		if err != nil {
			log.InfoLogf("mechanism/CREATE_CONFIG",
				"Creating %s mechanism config for %s switch", model, m.Datapath)

			err = p.Create(model, map[string]string{"id": m.Datapath})
			if err != nil {
				log.ErrorLogf("mechanism/CREATE_CONFIG",
					"Failed to create %s mechanism config for %s switch", model, err)
			}

			return err
		}

		log.InfoLogf("mechanism/CREATE_CONFIG",
			"Restoring %s mechanism configuration for %s", model, m.Datapath)

		if err = fn(); err != nil {
			log.ErrorLogf("mechanism/CREATE_CONFIG",
				"Failed to restore %s mechanism configuration", model)
		}

		return err
	})

	if err != nil {
		log.ErrorLog("mechanism/CREATE_CONFIG",
			"Failed to create mechanism config: ", err)
	}

	return err
}

func (m *BaseMechanismManager) Update(model db.Model, context interface{}, fn func() error) (err error) {
	err = db.Transaction(func(p db.ModelPersister) error {
		if err = db.Lock(model, m.Datapath, context); err != nil {
			log.ErrorLog("mechanism/UPDATE_CONFIG_DB_LOCK",
				"Failed to lock mechanism config: ", err)
			return err
		}

		// fn can update context value
		if err = fn(); err != nil {
			log.ErrorLog("mechanism/UPDATE_CONFIG",
				"Failed to update mechanism config: ", err)
			return err
		}

		if err = db.Update(model, m.Datapath, context); err != nil {
			log.ErrorLog("mechanism/UPDATE_CONFIG_DB_UPDATE",
				"Failed to update %s mechanism config: ", model, err)
		}

		return err
	})

	if err != nil {
		log.ErrorLog("mechanism/UPDATE_CONFIG",
			"Failed to update mechanism config: ", err)
	}

	return err
}

// Enable enables all registered mechanisms
func (m *BaseMechanismManager) Enable(c *MechanismContext) {
	atomic.CompareAndSwapInt64(&m.enabled, 0, 1)

	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		mechanism.Enable(c)
		return true
	})
}

// EnableByName enables mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already enabled.
func (m *BaseMechanismManager) EnableByName(name string, c *MechanismContext) error {
	mechanism, ok := m.Mechanisms.Get(name)
	if !ok {
		return ErrMechanismNotRegistered
	}

	if mechanism.Enabled() {
		return ErrMechanismAlreadyEnabled
	}

	atomic.CompareAndSwapInt64(&m.enabled, 0, 1)
	mechanism.Enable(c)
	return nil
}

func (m *BaseMechanismManager) Enabled() bool {
	return atomic.LoadInt64(&m.enabled) == 1
}

// Activate activates all registered mechanisms
func (m *BaseMechanismManager) Activate() {
	atomic.CompareAndSwapInt64(&m.activated, 0, 1)

	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		mechanism.Activate()
		return true
	})
}

// ActivateByName activates mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already activated.
func (m *BaseMechanismManager) ActivateByName(name string) error {
	mechanism, ok := m.Mechanisms.Get(name)
	if !ok {
		return ErrMechanismNotRegistered
	}

	if mechanism.Activated() {
		return ErrMechanismAlreadyActivated
	}

	atomic.CompareAndSwapInt64(&m.activated, 0, 1)
	mechanism.Activate()
	return nil
}

func (m *BaseMechanismManager) Activated() bool {
	return atomic.LoadInt64(&m.activated) == 1
}

// Disable disables all registered mechanisms
func (m *BaseMechanismManager) Disable() {
	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		mechanism.Disable()
		return true
	})
}

// DisableByName disables mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already disabled.
func (m *BaseMechanismManager) DisableByName(name string) error {
	mechanism, ok := m.Mechanisms.Get(name)
	if !ok {
		return ErrMechanismNotRegistered
	}

	if !mechanism.Enabled() {
		return ErrMechanismAlreadyDisabled
	}

	mechanism.Disable()
	return nil
}

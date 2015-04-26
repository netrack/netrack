package mech

import (
	"errors"
	"sync/atomic"

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

	// Link layer mechanism manager.
	Link LinkMechanismManager

	// Network layer mechanism manager.
	Network NetworkMechanismManager

	// Extention mechanism manager.
	Extension *ExtensionMechanismManager
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
	Mechanism(string) (Mechanism, error)

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
	Mechanisms MechanismMap
}

// Mechanism implements MechanismManager interface.
func (m *BaseMechanismManager) Mechanism(name string) (Mechanism, error) {
	if mechanism, ok := m.Mechanisms.Get(name); ok {
		return mechanism, nil
	}

	return nil, ErrMechanismNotRegistered
}

// Enable enables all registered mechanisms
func (m *BaseMechanismManager) Enable(c *MechanismContext) {
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

	mechanism.Enable(c)
	return nil
}

// Activate activates all registered mechanisms
func (m *BaseMechanismManager) Activate() {
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

	mechanism.Activate()
	return nil
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

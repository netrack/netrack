package mech

import (
	"sync/atomic"

	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
)

// MechanismDriverContext is an context, that shared among
// mechanisms enabled for a particular device.
type MechanismDriverContext struct {
	// OpenFlow switch instance.
	Switch Switch

	// Pipe to connect mechanism drivers.
	Func rpc.ProcCaller

	// OpenFlow multiplexer handler.
	Mux *of.ServeMux
}

// MechanismDriver describes switch drivers
type MechanismDriver interface {
	// Enable performs driver initialization.
	Enable(*MechanismDriverContext)

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

// BaseMechanismDriver implements thread-safe methods of
// MechanismDriver interface.
type BaseMechanismDriver struct {
	enabled   int64
	activated int64

	// MechanismDriverContext instance
	C *MechanismDriverContext
}

// Enable implements MechanismDriver interface.
func (m *BaseMechanismDriver) Enable(c *MechanismDriverContext) {
	m.C = c
	atomic.CompareAndSwapInt64(&m.enabled, 0, 1)
}

// Enabled implements MechanismDriver interface.
func (m *BaseMechanismDriver) Enabled() bool {
	return atomic.LoadInt64(&m.enabled) == 1
}

// Active implements MechanismDriver interface.
func (m *BaseMechanismDriver) Activate() {
	atomic.CompareAndSwapInt64(&m.activated, 0, 1)
}

// Activated implements MechanismDriver interface.
func (m *BaseMechanismDriver) Activated() bool {
	return atomic.LoadInt64(&m.activated) == 1
}

// Disable implements MechanismDriver interface.
func (m *BaseMechanismDriver) Disable() {
	atomic.StoreInt64(&m.enabled, 0)
}

// MechanismDriverConstructor is a generic
// constructor for mechanism drivers.
type MechanismDriverConstructor interface {
	// New creates a new MechanismDriver instance.
	New() MechanismDriver
}

// MechanismDriverCostructorFunc is a function adapter for
// MechanismDriverCostructor.
type MechanismDriverConstructorFunc func() MechanismDriver

// New implements MechanismDriverConstructor interface.
func (fn MechanismDriverConstructorFunc) New() MechanismDriver {
	return fn()
}

var mechanisms = make(map[string]MechanismDriverConstructor)

// RegisterMechanism makes a mechanism available by provided name
func RegisterMechanismDriver(name string, mechanism MechanismDriverConstructor) {
	if mechanism == nil {
		log.FatalLog("driver/REGISTER_MECHANISM",
			"Failed to register nil driver for: ", name)
	}

	if _, dup := mechanisms[name]; dup {
		log.FatalLog("driver/REGISTER_DUPLICATE",
			"Falied to register duplicate driver for: ", name)
	}

	mechanisms[name] = mechanism
}

// MechanismDriverByName returns MechanismDriverConstructor
// registered for specified name, nil will be returned when
// there is no required constructor.
func MechanismDriverByName(name string) MechanismDriverConstructor {
	return mechanisms[name]
}

// MechanismDriverNameList returns names of registered mechanism drivers.
func MechanismDriverNameList() []string {
	var names []string

	for name := range mechanisms {
		names = append(names, name)
	}

	return names
}

// MechanismDriverList returns list of registered
// mechanism drivers constructors.
func MechanismDriverList() []MechanismDriverConstructor {
	var list []MechanismDriverConstructor

	for _, driver := range mechanisms {
		list = append(list, driver)
	}

	return list
}

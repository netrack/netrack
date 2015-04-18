package mech

import (
	"sync/atomic"

	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
)

// MechanismContext is a context, that shared among
// mechanisms enabled for a particular device.
type MechanismContext struct {
	// OpenFlow switch instance.
	Switch Switch

	// Pipe to connect mechanism drivers.
	Func rpc.ProcCaller

	// OpenFlow multiplexer handler.
	Mux *of.ServeMux
}

// Mechanism describes switch drivers
type Mechanism interface {
	// Enable performs driver initialization.
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

// MechanismConstructor is a generic
// constructor for mechanism drivers.
type MechanismConstructor interface {
	// New creates a new Mechanism instance.
	New() Mechanism
}

// MechanismCostructorFunc is a function adapter for
// MechanismCostructor.
type MechanismConstructorFunc func() Mechanism

// New implements MechanismConstructor interface.
func (fn MechanismConstructorFunc) New() Mechanism {
	return fn()
}

var mechanisms = make(map[string]MechanismConstructor)

// RegisterMechanism makes a mechanism available by provided name
func RegisterMechanism(name string, mechanism MechanismConstructor) {
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

// MechanismByName returns MechanismConstructor
// registered for specified name, nil will be returned when
// there is no required constructor.
func MechanismByName(name string) MechanismConstructor {
	return mechanisms[name]
}

// MechanismNameList returns names of registered mechanism drivers.
func MechanismNameList() []string {
	var names []string

	for name := range mechanisms {
		names = append(names, name)
	}

	return names
}

// MechanismList returns list of registered
// mechanism drivers constructors.
func MechanismList() []MechanismConstructor {
	var list []MechanismConstructor

	for _, driver := range mechanisms {
		list = append(list, driver)
	}

	return list
}

package mech

import (
	"github.com/netrack/netrack/logging"
	"github.com/netrack/openflow"
)

// SwitchContext is a placeholder for mechanism driver context
// and mechanism drivers for a single switch.
type SwitchContext struct {
	// Mechanism driver context.
	*MechanismContext

	// Link layer mechanism manager.
	Links *LinkMechanismManager

	// Network layer mechanism manager.
	Networks *NetworkMechanismManager

	// Extention mechanism manager.
	Extensions *ExtensionMechanismManager
}

// SwichPort describes switch ports.
type SwitchPort interface {
	// Name returns name of the switch port.
	Name() string

	// Number returns number of the port in a switch.
	Number() uint32

	// Link returns link layer resources.
	//Link() *LinkContext

	// Network returns network layer resources.
	//Network() *NetworkContext
}

// Switch describes instance of openflow device
type Switch interface {
	// ID returns switch datapath identifier.
	ID() string

	// Boot performs version negotiation and initial switch
	// configuration on specified openflow connection. The
	// very next step on Boot call is to send ofp_hello message back.
	Boot(of.OFPConn) error

	// Conn returns connection to OpenFlow switch.
	Conn() of.OFPConn

	// Name return name of the switch local port,
	// which can be interpreted as switch name.
	Name() string

	// AllocateTables reserves first available table. Error
	// will be returned if all tables are aquired.
	AllocateTable() (int, error)

	// ReleaseTable makes table available for other mechanisms.
	ReleaseTable(int)

	// PortList returns list of ports registered in a switch.
	PortList() []SwitchPort

	// PortByName returns port instance by specified port name,
	// an error will returned if port not found.
	PortByName(string) (SwitchPort, error)

	// PortByNumber returns port instance by specified port number,
	// an error will returned if port not found.
	PortByNumber(uint32) (SwitchPort, error)
}

// SwitchConstructor is a generic constructor for switches.
type SwitchConstructor interface {
	// New creates a new Switch instance.
	New() Switch
}

// SwitchConstructorFunc is a function adapted for SwitchConstructor.
type SwitchConstructorFunc func() Switch

// New implements SwitchConstructor interface.
func (fn SwitchConstructorFunc) New() Switch {
	return fn()
}

var switches = make(map[string]SwitchConstructor)

// RegisterSwitch makes a switch available by provided version.
func RegisterSwitch(version string, s SwitchConstructor) {
	if s == nil {
		log.FatalLog("switch/REGISTER_SWITCH",
			"Failed to register nil switch for: ", version)
	}

	if _, dup := switches[version]; dup {
		log.FatalLog("switch/REGISTER_DUPLICATE",
			"Failed to register duplicate switch for: ", version)
	}

	switches[version] = s
}

// SwitchByVersion returns switch constructor registered for specified version,
// nil will be returned when there is no required switch constructor.
func SwitchByVersion(v string) SwitchConstructor {
	return switches[v]
}

// SwitchVersionList returns versions of registered switches.
func SwitchVersionList() []string {
	var versions []string

	for version := range switches {
		versions = append(versions, version)
	}

	return versions
}

// SwitchList returns list of registered switches constructors.
func SwitchList() []SwitchConstructor {
	var list []SwitchConstructor

	for _, s := range switches {
		list = append(list, s)
	}

	return list
}

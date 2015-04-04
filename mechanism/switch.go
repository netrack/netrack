package mech

import (
	"github.com/netrack/netrack/log"
	"github.com/netrack/openflow"
)

// Switch describes instance of openflow device
type Switch interface {
	// Boot performs version negotiation and intial switch
	// configuration on specified openflow connection.
	Boot(of.OFPConn) error

	// Conn returns connection to OpenFlow switch.
	Conn() of.OFPConn

	// ID returns switch datapath identifier.
	ID() string

	// Name return name of the switch local port,
	// which can be interpreted as switch name.
	Name() string

	// PortNameList returns names of ports available in a system
	PortNameList() []string

	// PortName returns name of the specified port, an error will be
	// returned if port does not exist in a system.
	PortName(int) (string, error)

	// PortHWAddrList returns list of hardware addresses
	// of ports available in a system.
	PortHWAddrList() [][]byte

	// PortHWAddr returns hardware address of the specified port,
	// an error will be returned if port does not exist in a system.
	PortHWAddr(int) ([]byte, error)
}

// SwitchConstructor is a generic constructor for switches.
type SwitchConstructor interface {
	// New creates a new Switch instance.
	New() Switch
}

// SwitchConstructorFunc is a function adapted for SwitchConstructor.
type SwitchConstructorFunc func() Switch

func (fn SwitchConstructorFunc) New() Switch {
	return fn()
}

var switches map[string]SwitchConstructor

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

// SwitchVersionList returns versions of registered switches.
func SwitchVersionList() []string {
	var versions []string

	for version := range switches {
		versions = append(versions, version)
	}

	return version
}

// SwitchList returns list of registered switches constructors.
func SwitchList() []SwitchConstructor {
	var list []SwitchConstructor

	for _, s := range switches {
		list = append(list, s)
	}

	return list
}

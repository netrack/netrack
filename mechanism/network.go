package mech

import (
	"io"
)

const (
	// IPv4 protocol
	NetworkProtoIPv4 NetworkProto = "IPv4"

	// IPv6 protocol
	NetworkProtoIPv6 NetworkProto = "IPv6"
)

// NetworkProto is network layer protocol.
type NetworkProto string

// NetworkAddr represents a L3 address.
type NetworkAddr interface {
	// String returns string form of address.
	String() string
}

// NetworkContext wraps network resources and provides
// methods for accessing other network intformation.
type NetworkContext struct {
	// Network layer address
	Addr NetworkAddr
}

// NetworkMechanism is the interface implemented by an object
// that can handle OSI L3 network types.
type NetworkMechanism interface {
	Mechanism

	// Proto returns network layer encapsulation protocol.
	Proto() NetworkProto

	// Network returns context of the type mechanism.
	Network() *NetworkContext

	// UpdateNetwork is called for all changes to network state.
	UpdateNetwork(*NetworkContext) error

	// Encapsulate wraps reader data with network layer packet.
	//Encapsulate(io.Reader) (io.Reader, error)
}

// NetworkMechanismContructor is a generic
// constructor for network type mechanisms.
type NetworkMechanismContructor interface {
	// New returns a new NetworkMechanism instance.
	New() NetworkMechanism
}

// NetworkMechanismConstructorFunc is a function adapter for
// NetworkMechanismConstructor.
type NetworkTypeConstructorFunc func() NewtorkType

func (fn NetworkMechanismContructorFunc) New() NetworkMechanism {
	return fn()
}

var networks = make(map[string]NetworkMechanism)

func RegisterNetworkMechanism(name string, mechanism NetworkMechanismContructor) {
	if mechanism == nil {
		log.FatalLog("mechanism/REGISTER_NETWORK_MECHANISM",
			"Failed to register nil network mechanism for: ", name)
	}

	if _, dup := networks[name]; dup {
		log.FatalLog("mechanism/REGISTER_NETWORK_MECHANISM",
			"Falied to register duplicate network mechanism for: ", name)
	}

	networks[name] = mechanism
}

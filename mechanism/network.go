package mech

import (
	"errors"

	"github.com/netrack/netrack/logging"
)

// NetworkAddr represents a L3 address.
type NetworkAddr interface {
	// String returns string form of address.
	String() string

	// Bytes returns raw address representation.
	Bytes() []byte

	// Mask return network layer address mask.
	Mask() []byte
}

// NetworkContext wraps network resources and provides
// methods for accessing other network intformation.
type NetworkContext struct {
	// Network layer address.
	Addr NetworkAddr
}

// NetworkPacket describes OSI L3 PDU.
type NetworkPacket interface {
	// DstAddr returns packet destination address.
	DstAddr() NetworkAddr

	// SrcAddr returns packet source address.
	SrcAddr() NetworkAddr

	// Proto returns payload protocol type.
	Proto() Proto

	// Len returns length of the protocol header.
	Len() int64

	// ContentLen returns length of the payload.
	ContentLen() int64
}

// NetworkDriver describes types that handles
// network layer protocol.
type NetworkDriver interface {
	// ParseAddr returns NetworkAddr from string.
	ParseAddr(string) (NetworkAddr, error)

	// Addr returns network layer address of specified link layer address.
	//Addr(laddr LinkAddr) (NetworkAddr, error)

	// Decapsulate removes network layer header from the packet.
	//Decapsulate(io.Reader) (NetworkPacket, error)
}

// BaseNetworkDriver implements NetworkDriver interface.
type BaseNetworkDriver struct{}

// ParseAddr implements NetworkDriver interface.
func (d *BaseNetworkDriver) ParseAddr(string) (NetworkAddr, error) {
	return nil, errors.New("BaseNetworkDriver: not implemented")
}

// Addr implements NetworkDriver interface.
func (d *BaseNetworkDriver) Addr(laddr LinkAddr) (NetworkAddr, error) {
	return nil, errors.New("BaseNetworkDriver: not implemented")
}

// NetworkMechanism is the interface implemented by an object
// that handles OSI network layer resources.
type NetworkMechanism interface {
	Mechanism

	// UpdateNetwork is called for all changes to network state.
	UpdateNetwork(*NetworkContext) error

	// DeleteNetwork erases all allocated resources.
	DeleteNetwork(*NetworkContext) error
}

// BaseNetworkMechanism implements NetworkMechanism interface.
type BaseNetworkMechanism struct {
	BaseMechanism
}

// UpdateNetwork implements NetworkMechanism interface.
func (m *BaseNetworkMechanism) UpdateNetwork(c *NetworkContext) error {
	return nil
}

// DeleteNetwork implements NetworkMechanism interface.
func (m *BaseNetworkMechanism) DeleteNetwork(c *NetworkContext) error {
	return nil
}

// NetworkMechanismConstructor is a generic
// constructor for network type mechanisms.
type NetworkMechanismConstructor interface {
	// New returns a new NetworkMechanism instance.
	New() NetworkMechanism
}

// NetworkMechanismConstructorFunc is a function adapter for
// NetworkMechanismConstructor.
type NetworkMechanismConstructorFunc func() NetworkMechanism

func (fn NetworkMechanismConstructorFunc) New() NetworkMechanism {
	return fn()
}

// NetworkDriverConstructor is a generic
// constructor for network drivers.
type NetworkDriverConstructor interface {
	// New returns a new NetwordDriver instance.
	New() NetworkDriver
}

// NetworkDriverConstructorFunc is a function adapter for
// NetworkDriverConstructor.
type NetworkDriverConstructorFunc func() NetworkDriver

func (fn NetworkDriverConstructorFunc) New() NetworkDriver {
	return fn()
}

var networks = make(map[string]NetworkMechanismConstructor)

// RegisterNetworkMechanism registers a new network layer mechanism
// under specified name.
func RegisterNetworkMechanism(name string, constructor NetworkMechanismConstructor) {
	if constructor == nil {
		log.FatalLog("network/REGISTER_NETWORK_MECHANISM",
			"Failed to register nil network constructor for: ", name)
	}

	if _, dup := networks[name]; dup {
		log.FatalLog("network/REGISTER_NETWORK_MECHANISM",
			"Falied to register duplicate network constructor for: ", name)
	}

	networks[name] = constructor
}

var networkDrivers = make(map[string]NetworkDriverConstructor)

// RegisterNetworkDriver registers a new network layer driver
// under specified name.
func RegisterNetworkDriver(name string, constructor NetworkDriverConstructor) {
	if constructor == nil {
		log.FatalLog("network/REGISTER_NETWORK_DRIVER",
			"Failed to register nil driver constructor for: ", name)
	}

	if _, dup := networkDrivers[name]; dup {
		log.FatalLog("network/REGISTER_NETWORK_DRIVER",
			"Falied to register duplicate driver constructor for: ", name)
	}

	networkDrivers[name] = constructor
}

// NetworkMechanisms returns map of registered network layer mechanisms
func NetworkMechanisms() NetworkMechanismMap {
	nmap := make(NetworkMechanismMap)

	for name, constructor := range networks {
		nmap.Set(name, constructor.New())
	}

	return nmap
}

// NetworkMechanismMap implements MechanismMap interface.
type NetworkMechanismMap map[string]NetworkMechanism

// Get returns Mechanism by specified name.
func (m NetworkMechanismMap) Get(s string) (Mechanism, bool) {
	mechanism, ok := m[s]
	return mechanism, ok
}

// Set registers Mechanis under specified name.
func (m NetworkMechanismMap) Set(s string, mechanism Mechanism) {
	nmechanism, ok := mechanism.(NetworkMechanism)
	if !ok {
		log.ErrorLog("network/SET_MECHANISM",
			"Failed to cast to network layer mechanism")
		return
	}

	m[s] = nmechanism
}

// Iter calls specified function for all registered mechcanisms.
func (m NetworkMechanismMap) Iter(fn func(string, Mechanism) bool) {
	for s, mechanism := range m {
		fn(s, mechanism)
	}
}

// NetworkMechanismManager manages link layer mechanisms.
type NetworkMechanismManager interface {
	// Base mechanism manager interface.
	MechanismManager

	// Network driver interface.
	NetworkDriver

	// UpdateNetwork forwards call to all registered mechanisms.
	UpdateNetwork(*NetworkContext) error

	// DeleteNetwork forwards call to all registered mechanisms.
	DeleteNetwork(*NetworkContext) error
}

// BaseNetworkMechanismManager implements NetworkMechanismManager interface.
type BaseNetworkMechanismManager struct {
	// Base mechanism manager instance.
	BaseMechanismManager

	// Network layer driver.
	Driver NetworkDriver
}

// ParseAddr parses string as network layer address.
func (m *BaseNetworkMechanismManager) ParseAddr(s string) (NetworkAddr, error) {
	return m.Driver.ParseAddr(s)
}

// Iter calls specified function for all registered link layer mechanisms.
func (m *BaseNetworkMechanismManager) Iter(fn func(NetworkMechanism) bool) {
	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		nmechanism, ok := mechanism.(NetworkMechanism)
		if !ok {
			log.ErrorLog("network/ITERATE",
				"Failed to cast mechanism to network layer mechanism.")
			return true
		}

		return fn(nmechanism)
	})
}

type networkMechanismFunc func(NetworkMechanism, *NetworkContext) error

func (m *BaseNetworkMechanismManager) do(context *NetworkContext, fn networkMechanismFunc) (err error) {
	m.Iter(func(mechanism NetworkMechanism) bool {
		// Forward request only to activated mechanisms.
		if !mechanism.Activated() {
			return true
		}

		if err = fn(mechanism, context); err != nil {
			log.ErrorLog("network/ALTER_NETWORK",
				"Failed to alter network layer mechanism: ", err)
			return false
		}

		return true
	})

	return
}

// UpdateNetwork calls corresponding method for activated mechanisms.
func (m *BaseNetworkMechanismManager) UpdateNetwork(context *NetworkContext) error {
	return m.do(context, NetworkMechanism.UpdateNetwork)
}

// DeleteNetwork calls corresponding method for activated mechanisms.
func (m *BaseNetworkMechanismManager) DeleteNetwork(context *NetworkContext) error {
	return m.do(context, NetworkMechanism.DeleteNetwork)
}

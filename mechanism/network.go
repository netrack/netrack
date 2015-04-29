package mech

import (
	"errors"
	"io"
	"sync"

	"github.com/netrack/netrack/database"
	"github.com/netrack/netrack/ioutil"
	"github.com/netrack/netrack/logging"
)

const (
	// NetworkModel is a database table name (networks)
	NetworkModel db.Model = "network"
)

func init() {
	// Register model in a database to make it available
	db.Register(NetworkModel)
}

var (
	// ErrNetworkNotRegistered is returned on
	// not registered driver operations.
	ErrNetworkNotRegistered = errors.New(
		"NetworkManager: network driver not registered")

	// ErrNewtorkNotIntialized is returned on
	// accessing not intialized network driver.
	ErrNetworkNotInitialized = errors.New(
		"NetworkManager: network driver not intialized")
)

// NetworkAddr represents a L3 address.
type NetworkAddr interface {
	// String returns string form of address.
	String() string

	// Contains reports whether the network includes address.
	Contains(NetworkAddr) bool

	// Bytes returns raw address representation.
	Bytes() []byte

	// Mask return network layer address mask.
	Mask() []byte
}

// NetworkPort represents network layer
// port abstraction.
type NetworkPort struct {
	// Network layer address string.
	Addr string `json:"address"`

	// Switch port number.
	Port uint32 `json:"port"`
}

type NetworkManagerContext struct {
	// Datapath identifier. This one is necessary
	// only for database storing.
	Datapath string `json:"id"`

	// Network driver name.
	Driver string `json:"driver"`

	// List of ports to modify.
	Ports []NetworkPort `json:"ports"`
}

// Port searchs for a specified port number.
func (c *NetworkManagerContext) Port(p uint32) NetworkPort {
	for _, port := range c.Ports {
		if port.Port == p {
			return port
		}
	}

	return NetworkPort{}
}

// SetPort updates ports with specified one.
func (c *NetworkManagerContext) SetPort(p NetworkPort) {
	for i, port := range c.Ports {
		if port.Port == p.Port {
			c.Ports[i] = p
			return
		}
	}

	c.Ports = append(c.Ports, p)
}

// DelPort remove specified port from port list.
func (c *NetworkManagerContext) DelPort(p NetworkPort) {
	for i, port := range c.Ports {
		if port.Port == p.Port {
			c.Ports = append(c.Ports[:i], c.Ports[i+1:]...)
			return
		}
	}
}

// NetworkContext wraps network resources and provides
// methods for accessing other network intformation.
type NetworkContext struct {
	// Network layer address.
	Addr NetworkAddr

	// Switch port number.
	Port uint32
}

// NetworkPacket describes OSI L3 PDU.
type NetworkPacket struct {
	// Packet destination address.
	DstAddr NetworkAddr

	// Packet source address.
	SrcAddr NetworkAddr

	// Payload protocol type.
	Proto Proto

	// Length of the protocol header.
	Len int64

	// Length of the payload.
	ContentLen int64

	// Payload data. This field should not be nil on write
	// operations.
	Payload io.Reader
}

// NetworkPacketReader describes types, that can read network layer packets.
type NetworkPacketReader interface {
	// ReadPacket reads network layer packets.
	ReadPacket(io.Reader) (*NetworkPacket, error)
}

// MakeNetworkReaderFrom is a helper to transform NetworkPacketReader to io.ReaderFrom.
func MakeNetworkReaderFrom(rp NetworkPacketReader, p *NetworkPacket) io.ReaderFrom {
	return ioutil.ReaderFromFunc(func(r io.Reader) (int64, error) {
		packet, err := rp.ReadPacket(r)
		if err != nil {
			return 0, err
		}

		*p = *packet
		return packet.Len, nil
	})
}

// NetworkPacketWriter describes types, taht can write network layer packets.
type NetworkPacketWriter interface {
	// WritePacket writers network layer packets.
	WritePacket(io.Writer, *NetworkPacket) error
}

// MakeNetworkWriterTo is a helper to transform NetworkPacketWriter type to io.WriterTo.
func MakeNetworkWriterTo(wp NetworkPacketWriter, p *NetworkPacket) io.WriterTo {
	return ioutil.WriterToFunc(func(w io.Writer) (int64, error) {
		return p.Len, wp.WritePacket(w, p)
	})
}

// NetworkDriver describes types that handles
// network layer protocol.
type NetworkDriver interface {
	// Name returns driver name.
	Name() string

	// ParseAddr returns network layer address from string.
	ParseAddr(string) (NetworkAddr, error)

	// CreateAddr returns a new NetworkAddr
	CreateAddr([]byte, []byte) NetworkAddr

	// Addr returns network layer address of specified switch port.
	Addr(uint32) (NetworkAddr, error)

	// UpdateAddr updates switch port network layer address.
	UpdateAddr(uint32, NetworkAddr) error

	// Reads network layer packets.
	NetworkPacketReader

	// Writes network layer packets.
	NetworkPacketWriter
}

// BaseNetworkDriver implements NetworkDriver interface.
type BaseNetworkDriver struct{}

// ParseAddr implements NetworkDriver interface.
func (d *BaseNetworkDriver) ParseAddr(string) (NetworkAddr, error) {
	return nil, errors.New("BaseNetworkDriver: not implemented")
}

// Addr implements NetworkDriver interface.
func (d *BaseNetworkDriver) Addr(LinkAddr) (NetworkAddr, error) {
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

type CompositeNetworkMechanism struct {
	BaseMechanism

	Mechanisms []NetworkMechanism
}

func (m *CompositeNetworkMechanism) do(fn func(NetworkMechanism) error) error {
	for _, mechanism := range m.Mechanisms {
		if err := fn(mechanism); err != nil {
			return err
		}
	}

	return nil
}

func (m *CompositeNetworkMechanism) Enable(context *MechanismContext) {
	m.do(func(mechanism NetworkMechanism) error {
		mechanism.Enable(context)
		return nil
	})
}

func (m *CompositeNetworkMechanism) Activate() {
	m.do(func(mechanism NetworkMechanism) error {
		mechanism.Activate()
		return nil
	})
}

func (m *CompositeNetworkMechanism) Disable() {
	m.do(func(mechanism NetworkMechanism) error {
		mechanism.Disable()
		return nil
	})
}

func (m *CompositeNetworkMechanism) UpdateNetwork(context *NetworkContext) error {
	return m.do(func(mechanism NetworkMechanism) error {
		return mechanism.UpdateNetwork(context)
	})
}

func (m *CompositeNetworkMechanism) DeleteNetwork(context *NetworkContext) error {
	return m.do(func(mechanism NetworkMechanism) error {
		return mechanism.DeleteNetwork(context)
	})
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

func NetworkMechanismCompositeConstructor(constuctors ...NetworkMechanismConstructor) NetworkMechanismConstructor {
	return NetworkMechanismConstructorFunc(func() NetworkMechanism {
		var mechanisms []NetworkMechanism
		for _, constructor := range networks {
			mechanisms = append(mechanisms, constructor.New())
		}

		return &CompositeNetworkMechanism{Mechanisms: mechanisms}
	})
}

// NetworkMechanisms returns map of registered network layer mechanisms
func NetworkMechanisms() NetworkMechanismMap {
	nmap := make(NetworkMechanismMap)

	for name, constructor := range networks {
		nmap.Set(name, constructor.New())
	}

	return nmap
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

// NetworkDriver returns map of registered network layer drivers instances.
func NetworkDrivers() map[string]NetworkDriver {
	nmap := make(map[string]NetworkDriver)

	for name, constructor := range networkDrivers {
		nmap[name] = constructor.New()
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

	// NetworkDriver returns active network layer driver instance.
	Driver() (NetworkDriver, error)

	// Context returns network context.
	Context(*NetworkManagerContext) error

	// CreateNetwork allocates necessary resources and restores
	// persisted state.
	CreateNetwork() error

	// UpdateNetwork forwards call to all registered mechanisms.
	UpdateNetwork(*NetworkManagerContext) error

	// DeleteNetwork forwards call to all registered mechanisms.
	DeleteNetwork(*NetworkManagerContext) error
}

// BaseNetworkMechanismManager implements NetworkMechanismManager interface.
type BaseNetworkMechanismManager struct {
	// Base mechanism manager instance.
	BaseMechanismManager

	// List of available network drivers.
	Drivers map[string]NetworkDriver

	// Activated network layer driver.
	drv NetworkDriver

	// Lock for drv member.
	lock sync.RWMutex
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

func (m *BaseNetworkMechanismManager) do(fn networkMechanismFunc, context *NetworkContext) (err error) {
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

func (m *BaseNetworkMechanismManager) driver(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// If driver already activated.
	if m.drv != nil && m.drv.Name() == name {
		return nil
	}

	// Search for a new driver.
	drv, ok := m.Drivers[name]
	if !ok {
		log.ErrorLog("network/NETWORK_DRIVER",
			"Requested network driver not found: ", name)
		return ErrNetworkNotRegistered
	}

	m.drv = drv
	return nil
}

// Driver returns active network layer driver.
func (m *BaseNetworkMechanismManager) Driver() (NetworkDriver, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.drv == nil {
		log.ErrorLog("network/NETWORK_DRIVER",
			"Network layer driver is not itialized")
		return nil, ErrNetworkNotInitialized
	}

	return m.drv, nil
}

// Context returns network context of specified switch port.
func (m *BaseNetworkMechanismManager) Context(context *NetworkManagerContext) error {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.drv == nil {
		log.ErrorLog("network/CONTEXT",
			"Network layer driver is not initialized")
		return ErrNetworkNotInitialized
	}

	return m.BaseMechanismManager.Context(NetworkModel, context)
}

func (m *BaseNetworkMechanismManager) CreateNetwork() error {
	context := new(NetworkManagerContext)

	return m.BaseMechanismManager.Create(NetworkModel, context, func() error {
		// Nothing to do.
		if context.Driver == "" {
			return nil
		}

		// Restore state of the persited switch
		if err := m.driver(context.Driver); err != nil {
			return err
		}

		m.lock.RLock()
		defer m.lock.RUnlock()

		for _, port := range context.Ports {
			addr, err := m.drv.ParseAddr(port.Addr)
			if err != nil {
				log.ErrorLog("network/UPDATE_NETWORK",
					"Failed to parse network layer address: ", err)
				return err
			}

			if err = m.drv.UpdateAddr(port.Port, addr); err != nil {
				log.ErrorLog("network/UPDATE_NETWORK",
					"Failed to update port network layer address: ", err)
				return err
			}

			err = m.do(NetworkMechanism.UpdateNetwork, &NetworkContext{
				Addr: addr,
				Port: port.Port,
			})

			if err != nil {
				log.ErrorLog("network/CREATE_NETWORK",
					"Failed to create network for port: ", port.Port)
				return err
			}
		}

		return nil
	})
}

// UpdateNetwork calls corresponding method for activated mechanisms.
func (m *BaseNetworkMechanismManager) UpdateNetwork(context *NetworkManagerContext) (err error) {
	network := new(NetworkManagerContext)

	if err = m.driver(context.Driver); err != nil {
		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.BaseMechanismManager.Update(NetworkModel, network, func() error {
		network.Driver = context.Driver

		for _, port := range context.Ports {
			addr, err := m.drv.ParseAddr(port.Addr)
			if err != nil {
				log.ErrorLog("network/UPDATE_NETWORK",
					"Failed to parse network layer address: ", err)
				return err
			}

			if err = m.drv.UpdateAddr(port.Port, addr); err != nil {
				log.ErrorLog("network/UPDATE_NETWORK",
					"Failed to update port network layer address: ", err)
				return err
			}

			err = m.do(NetworkMechanism.UpdateNetwork, &NetworkContext{
				Addr: addr,
				Port: port.Port,
			})

			if err != nil {
				log.ErrorLog("network/UPDATE_NETWORK",
					"Failed to update network configuration: ", err)
				return err
			}

			// Update port configuration
			network.SetPort(port)
		}

		return nil
	})

}

// DeleteNetwork calls corresponding method for activated mechanisms.
func (m *BaseNetworkMechanismManager) DeleteNetwork(context *NetworkManagerContext) (err error) {
	network := new(NetworkManagerContext)

	if err = m.driver(context.Driver); err != nil {
		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.BaseMechanismManager.Update(NetworkModel, network, func() error {
		for _, port := range context.Ports {
			err = m.do(NetworkMechanism.DeleteNetwork, &NetworkContext{
				Port: port.Port,
			})

			if err != nil {
				log.ErrorLog("network/DELETE_NETWORK",
					"Failed to delete network configuration: ", err)
				return err
			}

			// Update port configuration
			network.DelPort(port)
		}

		//TODO: delete address from the driver

		return nil
	})
}

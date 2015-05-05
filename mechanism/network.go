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

	// Register network mechanism manager as link layer mechanism
	constructor := LinkMechanismConstructorFunc(func() LinkMechanism {
		return NewNetworkMechanismManager()
	})

	RegisterLinkMechanism("inet", constructor)
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
	// Network layer driver.
	NetworkDriver NetworkDriver

	// Link layer driver.
	LinkDriver LinkDriver

	// Network layer address.
	NetworkAddr NetworkAddr

	// Link layer address.
	LinkAddr LinkAddr

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

	// DeleteAddr delets address associated with port.
	DeleteAddr(uint32) error

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
	// Base mechanism interface.
	Mechanism

	// CreateNetworkPreCommit
	CreateNetworkPreCommit(*NetworkContext) error

	// CreateNetworkPostCommit
	CreateNetworkPostCommit() error

	// UpdateNetwork is called for all changes to network state.
	UpdateNetworkPreCommit(*NetworkContext) error

	// UpdateNetworkPostCommit
	UpdateNetworkPostCommit(*NetworkContext) error

	// DeleteNetwork erases all allocated resources.
	DeleteNetworkPreCommit(*NetworkContext) error

	// DeleteNetworkPostCommit
	DeleteNetworkPostCommit() error
}

// BaseNetworkMechanism implements NetworkMechanism interface.
type BaseNetworkMechanism struct {
	BaseMechanism
}

// UpdateNetworkPreCommit implements NetworkMechanism interface.
func (m *BaseNetworkMechanism) CreateNetworkPreCommit(*NetworkContext) error {
	return nil
}

// CreateNetworkPostCommit implements NetworkMechanism interface.
func (m *BaseNetworkMechanism) CreateNetworkPostCommit() error {
	return nil
}

// UpdateNetworkPreCommit implements NetworkMechanism interface.
func (m *BaseNetworkMechanism) UpdateNetworkPreCommit(*NetworkContext) error {
	return nil
}

// UpdateNetworkPostCommit implements NetworkMechanism interface.
func (m *BaseNetworkMechanism) UpdateNetworkPostCommit(*NetworkContext) error {
	return nil
}

// DeleteNetworkPreCommit implements NetworkMechanism interface.
func (m *BaseNetworkMechanism) DeleteNetworkPreCommit(c *NetworkContext) error {
	return nil
}

// DeleteNetworkPostCommit implements NetworkMechanism interface.
func (m *BaseNetworkMechanism) DeleteNetworkPostCommit() error {
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
	Context() (*NetworkManagerContext, error)

	// UpdateNetwork forwards call to all registered mechanisms.
	UpdateNetwork(*NetworkManagerContext) error

	// DeleteNetwork forwards call to all registered mechanisms.
	DeleteNetwork(*NetworkManagerContext) error
}

func NetworkDrv(context *MechanismContext) (NetworkDriver, error) {
	var network NetworkMechanismManager
	if err := context.Managers.Obtain(&network); err != nil {
		log.ErrorLog("mechanism/NETWORK_DRIVER",
			"Failed obtain network layer manager: ", err)
		return nil, err
	}

	nldriver, err := network.Driver()
	if err != nil {
		log.ErrorLog("mechanism/NETWORK_DRIVER",
			"Network layer driver is not initialized: ", err)
		return nil, err
	}

	return nldriver, nil
}

// networkMechanismManager implements NetworkMechanismManager interface.
type networkMechanismManager struct {
	// Base mechanism manager instance.
	BaseMechanismManager

	// List of available network drivers.
	drivers map[string]NetworkDriver

	// Activated network layer driver.
	drv     NetworkDriver
	drvLock sync.RWMutex

	// Link layer driver
	lldrv     LinkDriver
	lldrvLock sync.RWMutex

	// Lock for drv member.
	lock sync.RWMutex
}

func NewNetworkMechanismManager() *networkMechanismManager {
	return &networkMechanismManager{
		drivers: NetworkDrivers(),
	}
}

func (m *networkMechanismManager) Enable(c *MechanismContext) {
	m.BaseMechanismManager = BaseMechanismManager{
		Datapath:   c.Switch.ID(),
		Mechanisms: NetworkMechanisms(),
		activated:  0,
		enabled:    0,
	}

	m.BaseMechanismManager.Enable(c)
	c.Managers.Bind(new(NetworkMechanismManager), m)
}

func (m *networkMechanismManager) Activate() {
	m.BaseMechanismManager.Activate()

	network := new(NetworkManagerContext)

	// Create empty configuration
	err := m.BaseMechanismManager.Create(
		NetworkModel, network, func() error { return nil },
	)

	if err != nil {
		log.ErrorLog("network/ACTIVATE_HOOK",
			"Failed to create empty network configuration: ", err)
		return
	}

	log.DebugLog("network/ACTIVATE_HOOK",
		"Network mechanism manager activated")
}

func (m *networkMechanismManager) CreateLinkPreCommit(context *LinkContext) error {
	log.DebugLog("network/CREATE_LINK_HOOK",
		"Got request to create link")

	// Persist link layer driver
	m.SetLinkDriver(context.Driver)
	return nil
}

func (m *networkMechanismManager) CreateLinkPostCommit() error {
	log.DebugLog("network/CREATE_LINK_POSTCOMMIT",
		"Got create link postcommit request")

	err := m.CreateNetwork()
	if err != nil {
		log.ErrorLog("network/CREATE_LINK_POSTCOMMIT",
			"Failed to restore network configuration: ", err)
	}

	return err
}

// UpdateLink implements LinkMechanism interface, so network manager
// can react on link changese and forward requests to the network layer mechanims.
// Generally, on changing link layer address, network mechanisms should
// react appropriately.
func (m *networkMechanismManager) UpdateLinkPreCommit(context *LinkContext) error {
	log.DebugLog("network/UPDATE_LINK_PRECOMMIT",
		"Got update link precommit request")

	// Persist link layer driver
	m.SetLinkDriver(context.Driver)

	// Since that hook is for preparation puroposes,
	// we are not going to update persisted configuration
	// of the network layer. Instead, remote flows from the switch

	// If network layer address is not associated with
	// specified port, there is nothing to do then.

	networkContext, err := m.readNetworkContext(context.Port)
	if err != nil {
		return nil
	}

	err = m.do(func(nlmech NetworkMechanism) error {
		return nlmech.UpdateNetworkPreCommit(networkContext)
	})

	if err != nil {
		log.ErrorLog("network/UPDATE_LINK_PRECOMMIT",
			"Network update pre-commit failed: ", err)
	}

	return err
}

func (m *networkMechanismManager) UpdateLinkPostCommit(context *LinkContext) error {
	log.DebugLog("network/UPDATE_LINK_POSTCOMMIT",
		"Got update link postcommit request")

	// Persist link layer driver
	m.SetLinkDriver(context.Driver)

	networkContext, err := m.readNetworkContext(context.Port)
	if err != nil {
		return nil
	}

	err = m.do(func(nlmech NetworkMechanism) error {
		return nlmech.UpdateNetworkPostCommit(networkContext)
	})

	if err != nil {
		log.ErrorLog("network/UPDATE_LINK_POSTCOMMIT",
			"Network update post-commit failed: ", err)
	}

	return err
}

// DeleteLink implements LinkMechanism interface. On link layer deletion
// network layer should be turned off either.
func (m *networkMechanismManager) DeleteLinkPreCommit(context *LinkContext) error {
	log.DebugLog("network/DELETE_LINK_PRECOMMIT",
		"Got request to delete link")

	// Probably network layer just not initialized.
	networkContext, err := m.readNetworkContext(context.Port)
	if err != nil {
		return nil
	}

	// Delete network layer configuration
	err = m.DeleteNetwork(&NetworkManagerContext{
		Datapath: m.Datapath, Ports: []NetworkPort{{
			Addr: networkContext.NetworkAddr.String(),
			Port: networkContext.Port,
		}},
	})

	if err != nil {
		log.ErrorLog("network/DELETE_LINK_PRECOMMIT",
			"Failed to update network configuration: ", err)
	}

	return err
}

func (m *networkMechanismManager) DeleteLinkPostCommit() error {
	log.DebugLog("network/DELETE_LINK_POSTCOMMIT",
		"Got delete link postcommit request")
	return nil
}

func (m *networkMechanismManager) SetLinkDriver(lldriver LinkDriver) {
	m.lldrvLock.Lock()
	defer m.lldrvLock.Unlock()

	m.lldrv = lldriver
}

func (m *networkMechanismManager) LinkDriver() (LinkDriver, error) {
	m.lldrvLock.RLock()
	defer m.lldrvLock.RUnlock()

	if m.lldrv == nil {
		log.ErrorLog("network/LINK_DRIVER",
			"Link layer driver is not initialized")
		return nil, ErrLinkNotInitialized
	}

	return m.lldrv, nil
}

// Iter calls specified function for all registered link layer mechanisms.
func (m *networkMechanismManager) Iter(fn func(NetworkMechanism) bool) {
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

func (m *networkMechanismManager) do(fn func(NetworkMechanism) error) (err error) {
	m.Iter(func(mechanism NetworkMechanism) bool {
		// Forward request only to activated mechanisms.
		if !mechanism.Activated() {
			return true
		}

		if err = fn(mechanism); err != nil {
			log.ErrorLog("network/ALTER_NETWORK",
				"Failed to alter network layer mechanism: ", err)
			return false
		}

		return true
	})

	return
}

func (m *networkMechanismManager) SetDriver(name string) error {
	m.drvLock.Lock()
	defer m.drvLock.Unlock()

	// If driver already activated.
	if m.drv != nil && m.drv.Name() == name {
		return nil
	}

	// Search for a new driver.
	drv, ok := m.drivers[name]
	if !ok {
		log.ErrorLog("network/NETWORK_DRIVER",
			"Requested network driver not found: ", name)
		return ErrNetworkNotRegistered
	}

	m.drv = drv
	return nil
}

// Driver returns active network layer driver.
func (m *networkMechanismManager) Driver() (NetworkDriver, error) {
	m.drvLock.RLock()
	defer m.drvLock.RUnlock()

	if m.drv == nil {
		log.ErrorLog("network/NETWORK_DRIVER",
			"Network layer driver is not itialized")
		return nil, ErrNetworkNotInitialized
	}

	return m.drv, nil
}

// Context returns network context of specified switch port.
func (m *networkMechanismManager) Context() (*NetworkManagerContext, error) {
	context := new(NetworkManagerContext)
	err := m.BaseMechanismManager.Context(NetworkModel, context)
	if err != nil {
		log.ErrorLog("network/CONTEXT",
			"Failed to retrieve persisted configuration: ", err)
		return nil, err
	}

	return context, nil
}

func (m *networkMechanismManager) readNetworkContext(port uint32) (*NetworkContext, error) {
	lldriver, err := m.LinkDriver()
	if err != nil {
		return nil, err
	}

	nldriver, err := m.Driver()
	if err != nil {
		return nil, err
	}

	lladdr, err := lldriver.Addr(port)
	if err != nil {
		return nil, err
	}

	// If network layer address is not associated with
	// specified port, there is nothing to do then.
	nladdr, err := nldriver.Addr(port)
	if err != nil {
		return nil, err
	}

	context := &NetworkContext{
		NetworkAddr:   nladdr,
		NetworkDriver: nldriver,
		LinkAddr:      lladdr,
		LinkDriver:    lldriver,
		Port:          port,
	}

	return context, nil
}

func (m *networkMechanismManager) updateNetworkContext(port NetworkPort) (*NetworkContext, error) {
	lldriver, err := m.LinkDriver()
	if err != nil {
		return nil, err
	}

	nldriver, err := m.Driver()
	if err != nil {
		return nil, err
	}

	lladdr, err := lldriver.Addr(port.Port)
	if err != nil {
		log.ErrorLog("network/NETWORK_CONTEXT",
			"Link layer address is not associated with port: ", port.Port)
		return nil, err
	}

	nladdr, err := nldriver.ParseAddr(port.Addr)
	if err != nil {
		log.ErrorLog("network/NETWORK_CONTEXT",
			"Failed to parse network layer address: ", err)
		return nil, err
	}

	if err = nldriver.UpdateAddr(port.Port, nladdr); err != nil {
		log.ErrorLog("network/NETWORK_CONTEXT",
			"Failed to update network layer driver: ", err)
		return nil, err
	}

	context := &NetworkContext{
		NetworkAddr:   nladdr,
		NetworkDriver: nldriver,
		LinkAddr:      lladdr,
		LinkDriver:    lldriver,
		Port:          port.Port,
	}

	return context, nil
}

// CreateNetwork loads persisted network layer configuration.
func (m *networkMechanismManager) CreateNetwork() error {
	network := new(NetworkManagerContext)

	create := func(fn func() error) error {
		return m.BaseMechanismManager.Create(
			NetworkModel, network, fn,
		)
	}

	alter := func(port NetworkPort) error {
		networkContext, err := m.updateNetworkContext(port)
		if err != nil {
			return err
		}

		// Broadcast request to all activated network mechanisms
		err = m.do(func(nlmech NetworkMechanism) error {
			return nlmech.CreateNetworkPreCommit(networkContext)
		})

		if err != nil {
			log.ErrorLog("network/CREATE_NETWORK",
				"Failed to create network layer configuration: ", port.Port)
		}

		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return create(func() error {
		// If driver equals to empty string, that mechanism was not
		// previously activated, so there is nothing to configure
		if network.Driver == "" {
			return nil
		}

		// Restore state of the persited switch
		err := m.SetDriver(network.Driver)
		if err != nil {
			return err
		}

		for _, port := range network.Ports {
			if err := alter(port); err != nil {
				log.ErrorLog("network/CREATE_NETWORK",
					"Network create pre-commit failed: ", err)
				return err
			}
		}

		err = m.do(func(nlmech NetworkMechanism) error {
			return nlmech.CreateNetworkPostCommit()
		})

		if err != nil {
			log.ErrorLog("network/CREATE_NETWORK",
				"Network create post-commit failed: ", err)
		}

		return err
	})
}

// UpdateNetwork calls corresponding method for activated mechanisms.
func (m *networkMechanismManager) UpdateNetwork(context *NetworkManagerContext) (err error) {
	network := new(NetworkManagerContext)

	update := func(fn func() error) error {
		return m.BaseMechanismManager.Update(
			NetworkModel, network, fn,
		)
	}

	pre := func(port NetworkPort) error {
		networkContext, err := m.updateNetworkContext(port)
		if err != nil {
			return err
		}

		err = m.do(func(nlmech NetworkMechanism) error {
			return nlmech.UpdateNetworkPreCommit(networkContext)
		})

		if err != nil {
			log.ErrorLog("network/UPDATE_NETWORK_PRECOMMIT",
				"Failed to parse ")
		}

		return err
	}

	post := func(port NetworkPort) error {
		networkContext, err := m.updateNetworkContext(port)
		if err != nil {
			return err
		}

		err = m.do(func(nlmech NetworkMechanism) error {
			return nlmech.UpdateNetworkPostCommit(networkContext)
		})

		if err != nil {
			log.ErrorLog("network/UPDATE_NETWORK_POSTCOMMIT",
				"Failed to update network configuration: ", err)
		}

		return err
	}

	return update(func() error {
		_, err := m.LinkDriver()
		if err != nil {
			return err
		}

		_, err = m.Driver()
		if err != nil && len(network.Ports) != 0 {
			log.ErrorLog("network/UPDATE_NETWORK",
				"Driver not found, but ports need to be updated")
			return ErrNetworkNotInitialized
		}

		for _, port := range context.Ports {
			// Get previous port coniguration
			port = network.Port(port.Port)
			blank := NetworkPort{}

			// Skip blank ports
			if port == blank {
				continue
			}

			// Precommit changes
			if err := pre(port); err != nil {
				log.ErrorLog("network/UPDATE_NETWORK",
					"Network update pre-commit failed: ", err)
				return err
			}
		}

		// Set new driver
		err = m.SetDriver(context.Driver)
		if err != nil {
			return err
		}

		// Update driver configuration
		network.Driver = context.Driver

		for _, port := range context.Ports {
			if err := post(port); err != nil {
				log.ErrorLog("network/UPDATE_NETWORK",
					"Network update post-commit failed: ", err)
				return err
			}

			// Update port configuration
			network.SetPort(port)
		}

		return nil
	})
}

// DeleteNetwork calls corresponding method for activated mechanisms.
func (m *networkMechanismManager) DeleteNetwork(context *NetworkManagerContext) (err error) {
	// Create new instance, that whould be readed from
	// the database
	network := new(NetworkManagerContext)

	lldriver, err := m.LinkDriver()
	if err != nil {
		return err
	}

	nldriver, err := m.Driver()
	if err != nil {
		return err
	}

	update := func(fn func() error) error {
		return m.BaseMechanismManager.Update(
			NetworkModel, network, fn,
		)
	}

	alter := func(port NetworkPort) error {
		lladdr, err := lldriver.Addr(port.Port)
		if err != nil {
			log.ErrorLog("network/DELETE_NETWORK",
				"Link layer address is not assigned to port: ", err)
			return err
		}

		// Since, DELETE request does not necessary contains
		// network layer address to delete, first, request
		// driver for port configuration.
		nladdr, err := nldriver.Addr(port.Port)
		if err != nil {
			log.ErrorLog("network/DELETE_NETWORK",
				"Network layer address is not assigned to port: ", err)
			return err
		}

		// Forward event to activated mechanisms.
		err = m.do(func(nlmech NetworkMechanism) error {
			return nlmech.DeleteNetworkPreCommit(&NetworkContext{
				NetworkDriver: nldriver,
				NetworkAddr:   nladdr,
				LinkDriver:    lldriver,
				LinkAddr:      lladdr,
				Port:          port.Port,
			})
		})

		if err != nil {
			log.ErrorLog("network/DELETE_NETWORK",
				"Failed to delete network configuration: ", err)
			return err
		}

		return nldriver.DeleteAddr(port.Port)
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return update(func() error {
		for _, port := range context.Ports {
			if err := alter(port); err != nil {
				log.ErrorLog("network/DELETE_NETWORK",
					"Network delete pre-commit failed: ", err)
				return err
			}

			// Update network layer port configuration.
			// This updated instance will be stored in a database.
			network.DelPort(port)

		}

		err := m.do(func(nlmech NetworkMechanism) error {
			return nlmech.DeleteNetworkPostCommit()
		})

		if err != nil {
			log.ErrorLog("network/DELETE_NETWORK",
				"Network delete post-commit failed: ", err)
		}

		return err
	})
}

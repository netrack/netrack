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
	// LinkModel is a database table name (networks)
	LinkModel db.Model = "link"
)

func init() {
	// Register model in a database to make it available
	db.Register(LinkModel)
}

var (
	// ErrLinkNotRegistered is returned on
	// accessing not intialized link driver.
	ErrLinkNotRegistered = errors.New(
		"LinkManager: link driver not registered")

	// ErrNewtorkNotIntialized is returned on
	// accessing not intialized link driver.
	ErrLinkNotInitialized = errors.New(
		"LinkManager: link driver not intialized")
)

const (
	// Full-Duplex system, allows communication in both directions.
	LinkModeFullDuplex LinkMode = "FULL-DUPLEX"

	// Half-Duplex system provides communication in both
	// directions, but only one direction at a time.
	LinkModeHalfDuplex LinkMode = "HALF-DUPLEX"
)

// LinkMode is a link communication mode.
type LinkMode string

// LinkAddr represents a L2 address.
type LinkAddr interface {
	// String returns string form of address.
	String() string

	// Bytes returns raw address representation.
	Bytes() []byte
}

// LinkPort represents link layer
// port abstraction
type LinkPort struct {
	// Link layer address string.
	Addr string `json:"address"`

	// Switch port number
	Port uint32 `json:"port"`
}

type LinkManagerContext struct {
	// Datapath identifier. This one is necessary
	// only for database storing.
	Datapath string `json:"id"`

	// Link driver name.
	Driver string `json:"driver"`

	// List of ports to modify.
	Ports []LinkPort `json:"ports"`
}

// Port searchs for a specified port number.
func (c *LinkManagerContext) Port(p uint32) LinkPort {
	for _, port := range c.Ports {
		if port.Port == p {
			return port
		}
	}

	return LinkPort{}
}

// SetPort updates ports with specified one.
func (c *LinkManagerContext) SetPort(p LinkPort) {
	for i, port := range c.Ports {
		if port.Port == p.Port {
			c.Ports[i] = p
			return
		}
	}

	c.Ports = append(c.Ports, p)
}

// DelPort remove specified port from port list.
func (c *LinkManagerContext) DelPort(p LinkPort) {
	for i, port := range c.Ports {
		if port.Port == p.Port {
			c.Ports = append(c.Ports[:i], c.Ports[i+1:]...)
			return
		}
	}
}

// LinkContext wraps link layer resources and provides
// methods for accessing other link information.
type LinkContext struct {
	// Link layer address.
	Addr LinkAddr

	// Switch port number.
	Port uint32

	// Link communication mode.
	Mode LinkMode

	// Link bandwidth.
	Bandwidth int
}

// LinkFrame describes OSI L2 PDU.
type LinkFrame struct {
	// DstAddr is a frame destination address.
	DstAddr LinkAddr

	// SrcAddr is a frame source address.
	SrcAddr LinkAddr

	// Proto is payload protocol type. Value of
	// Proto depends on link layer encapsulation. In
	// case of ethernet Proto returns IANA ethernet types.
	Proto Proto

	// Len is a header length.
	Len int64
}

// LinkReaderFrom describes types, that can read link layer frames.
type LinkFrameReader interface {
	// ReadFrame reads link layer frames.
	ReadFrame(io.Reader) (*LinkFrame, error)
}

// MakeLinkReaderFrom is a helper to transform LinkFrameReader type to io.ReaderFrom
func MakeLinkReaderFrom(rf LinkFrameReader, f *LinkFrame) io.ReaderFrom {
	return ioutil.ReaderFromFunc(func(r io.Reader) (int64, error) {
		frame, err := rf.ReadFrame(r)
		if err != nil {
			return 0, err
		}

		*f = *frame
		return frame.Len, nil
	})
}

// LinkFramWriter describes types, that can write link layer header.
type LinkFrameWriter interface {
	// WriteFrame write link layer header.
	WriteFrame(io.Writer, *LinkFrame) error
}

// MakeLinkWriterTo is a helper to transform LinkFrameWriter type to io.WriterTo
func MakeLinkWriterTo(wf LinkFrameWriter, f *LinkFrame) io.WriterTo {
	return ioutil.WriterToFunc(func(w io.Writer) (int64, error) {
		return f.Len, wf.WriteFrame(w, f)
	})
}

// LinkDriver describes types that handles
// link layer protocols.
type LinkDriver interface {
	// Name returns driver name.
	Name() string

	// ParseAddr returns link layer address from string.
	ParseAddr(s string) (LinkAddr, error)

	// CreateAddr returns a new LinkAddr
	CreateAddr([]byte) LinkAddr

	// Addr returns link layer address of specified port.
	Addr(portNo uint32) (LinkAddr, error)

	// UpdateAddr updates switch port link layer address.
	UpdateAddr(uint32, LinkAddr) error

	// Reads link layer frames.
	LinkFrameReader

	// Writes link layer frames.
	LinkFrameWriter
}

// BaseLinkDriver implements LinkDriver interface.
type BaseLinkDriver struct{}

// ParseAddr implements LinkDriver interface.
func (d *BaseLinkDriver) ParseAddr(string) (LinkAddr, error) {
	return nil, errors.New("BaseLinkDriver: not implemented")
}

// Addr implements LinkDriver interface.
func (d *BaseLinkDriver) Addr(uint32) (LinkAddr, error) {
	return nil, errors.New("BaseLinkDriver: not implemented")
}

// LinkMechanism is the interface implemented by an object
// that can handle OSI L2 network types.
type LinkMechanism interface {
	Mechanism

	// UpdateLink is called for all changes to link state.
	UpdateLink(*LinkContext) error

	// DeleteLink erases all allocated resources.
	DeleteLink(*LinkContext) error
}

// BaseLinkMechanism implements LinkMechanism interface.
type BaseLinkMechanism struct {
	BaseMechanism
}

// CreateLink implements LinkMechanism interface.
func (m *BaseLinkMechanism) CreateLink(c *LinkContext) error {
	return nil
}

// DeleteLink implements LinkMechanism interface.
func (m *BaseLinkMechanism) DeleteLink(c *LinkContext) error {
	return nil
}

// LinkMechanismConstructor is a genereic
// constructor for data link type mechanisms.
type LinkMechanismConstructor interface {
	// New returns a new LinkMechanism instance.
	New() LinkMechanism
}

// LinkMechanismConstructorFunc is a function adapter for
// LinkMechanismConstructor.
type LinkMechanismConstructorFunc func() LinkMechanism

func (fn LinkMechanismConstructorFunc) New() LinkMechanism {
	return fn()
}

var links = make(map[string]LinkMechanismConstructor)

// RegisterLinkMechanism registers a new link layer mechanism
// under specified name.
func RegisterLinkMechanism(name string, ctor LinkMechanismConstructor) {
	if ctor == nil {
		log.FatalLog("link/REGISTER_LINK_MECHANISM",
			"Failed to register nil link constructor for: ", name)
	}

	if _, dup := links[name]; dup {
		log.FatalLog("link/REGISTER_LINK_MECHANISM",
			"Falied to register duplicate link constructor for: ", name)
	}

	links[name] = ctor
}

// LinkDriverConstructor is a generic
// constructor for network drivers.
type LinkDriverConstructor interface {
	// New returns a new NetwordDriver instance.
	New() LinkDriver
}

// LinkDriverConstructorFunc is a function adapter for
// LinkDriverConstructor.
type LinkDriverConstructorFunc func() LinkDriver

func (fn LinkDriverConstructorFunc) New() LinkDriver {
	return fn()
}

var linkDrivers = make(map[string]LinkDriverConstructor)

// RegisterLinkDriver registers a new link layer driver
// under specified name.
func RegisterLinkDriver(name string, constructor LinkDriverConstructor) {
	if constructor == nil {
		log.FatalLog("link/REGISTER_LINK_DRIVER",
			"Failed to register nil driver constructor for: ", name)
	}

	if _, dup := linkDrivers[name]; dup {
		log.FatalLog("link/REGISTER_LINK_DRIVER",
			"Falied to register duplicate driver constructor for: ", name)
	}

	linkDrivers[name] = constructor
}

// LinkDriver returns map of registered network layer drivers instances.
func LinkDrivers() map[string]LinkDriver {
	nmap := make(map[string]LinkDriver)

	for name, constructor := range linkDrivers {
		nmap[name] = constructor.New()
	}

	return nmap
}

// LinkMechanisms retruns instances of registered mechanisms.
func LinkMechanisms() LinkMechanismMap {
	lmap := make(LinkMechanismMap)

	for name, constructor := range links {
		lmap.Set(name, constructor.New())
	}

	return lmap
}

// LinkMechanismMap implements MechanismMap interface.
type LinkMechanismMap map[string]LinkMechanism

// Get returns Mechanism by specified name.
func (m LinkMechanismMap) Get(s string) (Mechanism, bool) {
	mechanism, ok := m[s]
	return mechanism, ok
}

// Set registers mechanism under specified name.
func (m LinkMechanismMap) Set(s string, mechanism Mechanism) {
	lmechanism, ok := mechanism.(LinkMechanism)
	if !ok {
		log.ErrorLog("link/SET_MECHANISM",
			"Failed to cast to link layer mechanism")
		return
	}

	m[s] = lmechanism
}

// Iter calls specified function for all registered mechanisms.
func (m LinkMechanismMap) Iter(fn func(string, Mechanism) bool) {
	for s, mechanism := range m {
		fn(s, mechanism)
	}
}

// LinkMechanismManager manages link layer mechanisms.
type LinkMechanismManager interface {
	// Base mechanism manager.
	MechanismManager

	// LinkDriver returns active link layer driver.
	Driver() (LinkDriver, error)

	// Context returns link context.
	Context() (*LinkManagerContext, error)

	// CreateLink allocates necessary resources and restores
	// persisted state.
	CreateLink() error

	// UpdateLink forwards call to all registered mechanisms.
	UpdateLink(*LinkManagerContext) error

	// DeleteLink forwards call to all registered mechanisms.
	DeleteLink(*LinkManagerContext) error
}

// BaseLinkMechcanismManager implements LinkMechanismManager.
type BaseLinkMechanismManager struct {
	// Datapath identifier of the service switch.
	Datapath string

	// Base mechanism manager instance.
	BaseMechanismManager

	// List of available link drivers.
	Drivers map[string]LinkDriver

	// Activated link layer driver.
	drv LinkDriver

	// Lock for drv member.
	lock sync.RWMutex
}

// Iter calls specified function for all registered link layer mechanisms.
func (m *BaseLinkMechanismManager) Iter(fn func(LinkMechanism) bool) {
	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		lmechanism, ok := mechanism.(LinkMechanism)
		if !ok {
			log.ErrorLog("link/ITERATE",
				"Failed to cast mechanism to link layer mechanism.")
			return true
		}

		return fn(lmechanism)
	})
}

type linkMechanismFunc func(LinkMechanism, *LinkContext) error

func (m *BaseLinkMechanismManager) do(fn linkMechanismFunc, context *LinkContext) (err error) {
	m.Iter(func(mechanism LinkMechanism) bool {
		// Forward request only to activated mechanisms.
		if !mechanism.Activated() {
			return true
		}

		if err = fn(mechanism, context); err != nil {
			log.ErrorLog("link/ALTER_LINK",
				"Failed to alter link layer mechanism: ", err)
			return false
		}

		return true
	})

	return
}

func (m *BaseLinkMechanismManager) driver(name string) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// If driver already activated.
	if m.drv != nil && m.drv.Name() == name {
		return nil
	}

	// Search for a new driver.
	drv, ok := m.Drivers[name]
	if !ok {
		log.ErrorLog("link/LINK_DRIVER",
			"Requested link driver not found: ", name)
		return ErrLinkNotRegistered
	}

	m.drv = drv
	return nil
}

// Driver implements LinkMechanismManager interface.
func (m *BaseLinkMechanismManager) Driver() (LinkDriver, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.drv == nil {
		log.ErrorLog("link/LINK_DRIVER",
			"Link layer driver is not initialized")
		return nil, ErrLinkNotInitialized
	}

	return m.drv, nil
}

// Context returns link context of specified switch port.
func (m *BaseLinkMechanismManager) Context() (*LinkManagerContext, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if m.drv == nil {
		log.ErrorLog("link/CONTEXT",
			"Link layer driver is not initialized")
		return &LinkManagerContext{}, ErrLinkNotInitialized
	}

	var context LinkManagerContext
	err := db.Read(LinkModel, m.Datapath, &context)
	if err != nil {
		log.ErrorLog("link/CONTEXT",
			"Failed to read link configuration from the database: ", err)
		return nil, err
	}

	return &context, nil
}

func (m *BaseLinkMechanismManager) CreateLink() error {
	var context LinkManagerContext

	err := db.Transaction(func(p db.ModelPersister) error {
		// Lock to make consistent configuration
		err := p.Lock(LinkModel, m.Datapath, &context)

		// Create a new record in a dabase for a new switch
		if err != nil {
			log.InfoLog("link/CREATE_LINK",
				"Creating link configuration for: ", m.Datapath)

			err = p.Create(LinkModel, &LinkManagerContext{Datapath: m.Datapath})
			if err != nil {
				log.ErrorLog("link/CREATE_LINK",
					"Failed to create link configuration: ", err)
			}

			return err
		}

		log.InfoLog("link/CREATE_LINK",
			"Restoring link configuration for: ", m.Datapath)

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
				log.ErrorLog("link/UPDATE_LINK",
					"Failed to parse link layer address: ", err)
				return err
			}

			if err = m.drv.UpdateAddr(port.Port, addr); err != nil {
				log.ErrorLog("link/UPDATE_LINK",
					"Failed to update port link layer address: ", err)
				return err
			}

			err = m.do(LinkMechanism.UpdateLink, &LinkContext{
				Addr: addr,
				Port: port.Port,
			})

			if err != nil {
				log.ErrorLog("link/CREATE_LINK",
					"Failed to create link for port: ", port.Port)
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.ErrorLog("link/CREATE_LINK",
			"Failed to create link records in a database: ", err)
	}

	return err
}

// UpdateLink calls corresponding method for activated mechanisms.
func (m *BaseLinkMechanismManager) UpdateLink(context *LinkManagerContext) (err error) {
	if err = m.driver(context.Driver); err != nil {
		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	ports := context.Ports

	for _, port := range ports {
		addr, err := m.drv.ParseAddr(port.Addr)
		if err != nil {
			log.ErrorLog("link/UPDATE_LINK",
				"Failed to parse link layer address: ", err)
			return err
		}

		if err = m.drv.UpdateAddr(port.Port, addr); err != nil {
			log.ErrorLog("link/UPDATE_LINK",
				"Failed to update port link layer address: ", err)
			return err
		}

		err = m.do(LinkMechanism.UpdateLink, &LinkContext{
			Addr: addr,
			Port: port.Port,
		})

		if err != nil {
			log.ErrorLog("link/UPDATE_LINK",
				"Failed to update link configuration: ", err)
			return err
		}
	}

	// Update link configuration in a database.
	err = db.Transaction(func(p db.ModelPersister) error {
		var oldcontext LinkManagerContext

		if err = db.Lock(LinkModel, m.Datapath, &oldcontext); err != nil {
			log.ErrorLog("link/UPDATE_LINK_DB_LOCK",
				"Failed to lock record: ", err)
			return err
		}

		// Save previous port configuration
		context.Ports = oldcontext.Ports

		// Update port configuration
		for _, port := range ports {
			context.SetPort(port)
		}

		if err = db.Update(LinkModel, m.Datapath, context); err != nil {
			log.ErrorLog("link/UPDATE_LINK_DB_UPDATE",
				"Failed to update record: ", err)
		}

		return err
	})

	if err != nil {
		log.ErrorLog("link/UPDATE_LINK",
			"Failed to create link records in a database: ", err)
	}

	return err
}

// DeleteLink calls corresponding method for activated mechanisms.
func (m *BaseLinkMechanismManager) DeleteLink(context *LinkManagerContext) (err error) {
	if err = m.driver(context.Driver); err != nil {
		return err
	}

	ports := context.Ports

	for _, port := range ports {
		err = m.do(LinkMechanism.DeleteLink, &LinkContext{
			Port: port.Port,
		})

		if err != nil {
			log.ErrorLog("link/DELETE_LINK",
				"Failed to delete link configuration: ", err)
			return err
		}
	}

	//TODO: delete address from the driver

	// Update link configuration in a database.
	err = db.Transaction(func(p db.ModelPersister) error {
		if err = db.Lock(LinkModel, m.Datapath, context); err != nil {
			log.ErrorLog("link/DELETE_LINK_DB_LOCK",
				"Failed to lock record: ", err)
			return err
		}

		// Update port configuration
		for _, port := range ports {
			context.DelPort(port)
		}

		if err = db.Update(LinkModel, m.Datapath, context); err != nil {
			log.ErrorLog("link/DELETE_LINK_DB_DELETE",
				"Failed to update record: ", err)
		}

		return err
	})

	if err != nil {
		log.ErrorLog("link/DELETE_LINK",
			"Failed to delete link records from a database: ", err)
	}

	return err
}

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

	// Link layer driver
	Driver LinkDriver

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

	// DeleteAddr deletes link layer address associated
	// with specified port.
	DeleteAddr(uint32) error

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

	// CreateLink is called upon link creation
	CreateLinkPreCommit(*LinkContext) error

	CreateLinkPostCommit() error

	// UpdateLink is called for all changes to link state.
	UpdateLinkPreCommit(*LinkContext) error

	UpdateLinkPostCommit(*LinkContext) error

	// DeleteLink erases all allocated resources.
	DeleteLinkPreCommit(*LinkContext) error

	DeleteLinkPostCommit() error
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
	// Base mechanism manager interface.
	MechanismManager

	// LinkDriver returns active link layer driver.
	Driver() (LinkDriver, error)

	// Context returns link context.
	Context() (*LinkManagerContext, error)

	// UpdateLink forwards call to all registered mechanisms.
	UpdateLink(*LinkManagerContext) error

	// DeleteLink forwards call to all registered mechanisms.
	DeleteLink(*LinkManagerContext) error
}

func LinkDrv(context *MechanismContext) (LinkDriver, error) {
	var link LinkMechanismManager
	if err := context.Managers.Obtain(&link); err != nil {
		log.ErrorLog("mechanism/LINK_DRIVER",
			"Failed obtain link layer manager: ", err)
		return nil, err
	}

	lldriver, err := link.Driver()
	if err != nil {
		log.ErrorLog("mechanism/LINK_DRIVER",
			"Link layer driver is not initialized: ", err)
		return nil, err
	}

	return lldriver, nil
}

// BaseLinkMechcanismManager implements LinkMechanismManager.
type linkMechanismManager struct {
	// Datapath identifier of the service switch.
	Datapath string

	// Base mechanism manager instance.
	BaseMechanismManager

	// List of available link drivers.
	drivers map[string]LinkDriver

	// Activated link layer driver.
	drv     LinkDriver
	drvLock sync.RWMutex

	// Lock for drv member.
	lock sync.RWMutex
}

func NewLinkMechanismManager() *linkMechanismManager {
	return &linkMechanismManager{
		drivers: LinkDrivers(),
	}
}

func (m *linkMechanismManager) Enable(c *MechanismContext) {
	m.BaseMechanismManager = BaseMechanismManager{
		Datapath:   c.Switch.ID(),
		Mechanisms: LinkMechanisms(),
		activated:  0,
		enabled:    0,
	}

	m.BaseMechanismManager.Enable(c)
	c.Managers.Bind(new(LinkMechanismManager), m)
}

func (m *linkMechanismManager) Activate() {
	// Activate registered mechanisms
	m.BaseMechanismManager.Activate()

	link := new(LinkManagerContext)
	err := m.BaseMechanismManager.Create(
		LinkModel, link, func() error { return nil },
	)

	if err != nil {
		log.ErrorLog("link/ACTIVATE_HOOK",
			"Failed create empty link configuration: ", err)
		return
	}

	log.DebugLog("link/ACTIVATE_HOOK",
		"Link mechanism manager activated")
}

// Iter calls specified function for all registered link layer mechanisms.
func (m *linkMechanismManager) Iter(fn func(LinkMechanism) bool) {
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

func (m *linkMechanismManager) do(fn func(LinkMechanism) error) (err error) {
	m.Iter(func(mechanism LinkMechanism) bool {
		// Forward request only to activated mechanisms.
		if !mechanism.Activated() {
			return true
		}

		if err = fn(mechanism); err != nil {
			log.ErrorLog("link/ALTER_LINK",
				"Failed to alter link layer mechanism: ", err)
			return false
		}

		return true
	})

	return
}

func (m *linkMechanismManager) SetDriver(name string) error {
	m.drvLock.Lock()
	defer m.drvLock.Unlock()

	// If driver already activated.
	if m.drv != nil && m.drv.Name() == name {
		return nil
	}

	// Search for a new driver.
	drv, ok := m.drivers[name]
	if !ok {
		log.ErrorLog("link/SET_LINK_DRIVER",
			"Requested link driver not found: ", name)
		return ErrLinkNotRegistered
	}

	m.drv = drv
	return nil
}

// Driver implements LinkMechanismManager interface.
func (m *linkMechanismManager) Driver() (LinkDriver, error) {
	m.drvLock.RLock()
	defer m.drvLock.RUnlock()

	if m.drv == nil {
		log.ErrorLog("link/LINK_DRIVER",
			"Link layer driver is not initialized")
		return nil, ErrLinkNotInitialized
	}

	return m.drv, nil
}

// Context returns link context of specified switch port.
func (m *linkMechanismManager) Context() (*LinkManagerContext, error) {
	if _, err := m.Driver(); err != nil {
		log.ErrorLog("link/CONTEXT",
			"Link layer driver is not initialized")
		return nil, err
	}

	context := new(LinkManagerContext)
	err := m.BaseMechanismManager.Context(LinkModel, context)
	if err != nil {
		log.ErrorLog("link/CONTEXT",
			"Failed to retrieve persisted configuration: ", err)
		return nil, err
	}

	return context, nil
}

func (m *linkMechanismManager) linkContext(port LinkPort) (*LinkContext, error) {
	lldriver, err := m.Driver()
	if err != nil {
		return nil, err
	}

	lladdr, err := lldriver.ParseAddr(port.Addr)
	if err != nil {
		log.ErrorLog("link/LINK_CONTEXT",
			"Failed to parse link layer address: ", err)
		return nil, err
	}

	if err = lldriver.UpdateAddr(port.Port, lladdr); err != nil {
		log.ErrorLog("link/LINK_CONTEXT",
			"Failed to update link layer driver: ", err)
		return nil, err
	}

	linkContext := &LinkContext{
		Addr:   lladdr,
		Port:   port.Port,
		Driver: lldriver,
	}

	return linkContext, nil
}

func (m *linkMechanismManager) CreateLink() error {
	link := new(LinkManagerContext)

	create := func(fn func() error) error {
		return m.BaseMechanismManager.Create(
			LinkModel, link, fn,
		)
	}

	alter := func(port LinkPort) error {
		linkContext, err := m.linkContext(port)
		if err != nil {
			return err
		}

		err = m.do(func(llmech LinkMechanism) error {
			return llmech.CreateLinkPreCommit(linkContext)
		})

		if err != nil {
			log.ErrorLog("link/CREATE_LINK",
				"Failed to create link for port: ", port.Port)
		}

		return err
	}

	return create(func() error {
		// Nothing to do.
		if link.Driver == "" {
			return nil
		}

		log.DebugLog("link/CREATE_LINK",
			"Restoring link layer")

		// Restore state of the persited switch
		err := m.SetDriver(link.Driver)
		if err != nil {
			return err
		}

		m.lock.RLock()
		defer m.lock.RUnlock()

		// Fire create pre-commit event
		for _, port := range link.Ports {
			if err = alter(port); err != nil {
				log.ErrorLog("link/CREATE_LINK",
					"Link layer create pre-commit failed: ", err)
				return err
			}
		}

		// Fire create-post-commit event
		err = m.do(func(llmech LinkMechanism) error {
			return llmech.CreateLinkPostCommit()
		})

		log.ErrorLog("link/CREATE_LINK",
			"Link layer create post-commit failed: ", err)

		return err
	})
}

// UpdateLink calls corresponding method for activated mechanisms.
func (m *linkMechanismManager) UpdateLink(context *LinkManagerContext) (err error) {
	link := new(LinkManagerContext)

	update := func(fn func() error) error {
		return m.BaseMechanismManager.Update(
			LinkModel, link, fn,
		)
	}

	pre := func(port LinkPort) error {
		linkContext, err := m.linkContext(port)
		if err != nil {
			return err
		}

		err = m.do(func(llmech LinkMechanism) error {
			return llmech.UpdateLinkPreCommit(linkContext)
		})

		if err != nil {
			log.ErrorLog("link/UPDATE_LINK_PRECOMMIT",
				"Failed to update link configuration: ", err)
		}

		return err
	}

	post := func(port LinkPort) error {
		linkContext, err := m.linkContext(port)
		if err != nil {
			return err
		}

		err = m.do(func(llmech LinkMechanism) error {
			return llmech.UpdateLinkPostCommit(linkContext)
		})

		if err != nil {
			log.ErrorLog("link/UPDATE_LINK_POSTCOMMIT",
				"Failed to update link configuration: ", err)
		}

		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return update(func() error {
		_, err := m.Driver()
		if err != nil && len(link.Ports) != 0 {
			log.ErrorLog("link/UPDATE_LINK",
				"Driver not found, but ports need to be updated")
			return ErrLinkNotInitialized
		}

		for _, port := range context.Ports {
			port = link.Port(port.Port)
			blank := LinkPort{}

			// Skip blank ports
			if port == blank {
				continue
			}

			// Update pre-commit
			if err := pre(port); err != nil {
				log.ErrorLog("link/UPDATE_LINK",
					"Link layer delete pre-commit failed: ", err)
				return err
			}
		}

		// Set new link layer driver
		err = m.SetDriver(context.Driver)
		if err != nil {
			return err
		}

		// Update driver configuration
		link.Driver = context.Driver

		for _, port := range context.Ports {
			// Update  post-commit
			if err := post(port); err != nil {
				log.ErrorLog("link/UPDATE_LINK",
					"Link layer update pre-commit failed: ", err)
				return err
			}

			// Update link layer port configuration
			link.SetPort(port)
		}

		return nil
	})
}

// DeleteLink calls corresponding method for activated mechanisms.
func (m *linkMechanismManager) DeleteLink(context *LinkManagerContext) (err error) {
	link := new(LinkManagerContext)

	update := func(fn func() error) error {
		return m.BaseMechanismManager.Update(
			LinkModel, link, fn,
		)
	}

	alter := func(lldriver LinkDriver, port LinkPort) error {
		addr, err := lldriver.Addr(port.Port)
		if err != nil {
			log.ErrorLog("link/DELETE_LINK",
				"Link layer address is not assigned to port: ", err)
			return err
		}

		// Remove association from the driver
		defer lldriver.DeleteAddr(port.Port)

		// Forward event to activated mechanisms
		err = m.do(func(llmech LinkMechanism) error {
			return llmech.DeleteLinkPreCommit(&LinkContext{
				Addr:   addr,
				Port:   port.Port,
				Driver: lldriver,
			})
		})

		if err != nil {
			log.ErrorLog("link/DELETE_LINK",
				"Failed to delete link configuration: ", err)
		}

		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return update(func() error {
		lldriver, err := m.Driver()
		if err != nil {
			return err
		}

		for _, port := range context.Ports {
			if err := alter(lldriver, port); err != nil {
				log.ErrorLog("link/DELETE_LINK",
					"Link delete pre-commit failed: ", err)
				return err
			}

			// Update link layer port configuration.
			link.DelPort(port)
		}

		err = m.do(func(llmech LinkMechanism) error {
			return llmech.DeleteLinkPostCommit()
		})

		if err != nil {
			log.ErrorLog("link/DELETE_LINK",
				"Link delete post-commit failed: ", err)
		}

		return nil
	})
}

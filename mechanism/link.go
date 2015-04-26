package mech

import (
	"errors"
	"io"

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

// LinkContext wraps link layer resources and provides
// methods for accessing other link information.
type LinkContext struct {
	// Link layer address.
	Addr LinkAddr

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
	// ParseAddr returns link layer address from string.
	ParseAddr(s string) (LinkAddr, error)

	// Addr returns link layer address of specified port.
	Addr(portNo uint32) (LinkAddr, error)

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

	// UpdateLink forwards call to all registered mechanisms.
	UpdateLink(*LinkContext) error

	// DeleteLink forwards call to all registered mechanisms.
	DeleteLink(*LinkContext) error
}

// BaseLinkMechcanismManager implements LinkMechanismManager.
type BaseLinkMechanismManager struct {
	// Base mechanism manager instance.
	BaseMechanismManager

	// Link layer driver.
	Drv LinkDriver
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

func (m *BaseLinkMechanismManager) do(context *LinkContext, fn linkMechanismFunc) (err error) {
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

// Driver implements LinkMechanismManager interface.
func (m *BaseLinkMechanismManager) Driver() (LinkDriver, error) {
	if m.Drv == nil {
		log.ErrorLog("link/LINK_DRIVER",
			"Link layer driver is not initialized")
		return nil, ErrLinkNotInitialized
	}

	return m.Drv, nil
}

// UpdateLink calls corresponding method for activated mechanisms.
func (m *BaseLinkMechanismManager) UpdateLink(context *LinkContext) (err error) {
	return m.do(context, LinkMechanism.UpdateLink)
}

// DeleteLink calls corresponding method for activated mechanisms.
func (m *BaseLinkMechanismManager) DeleteLink(context *LinkContext) (err error) {
	return m.do(context, LinkMechanism.DeleteLink)
}

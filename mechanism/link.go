package mech

import (
	"github.com/netrack/netrack/logging"
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

const (
	// High-Level Data Link Control protocol (ISO 13239)
	LinkProtoHDLC LinkProto = "HDLC"

	// Point-to-Point protocol (RFC 1661)
	LinkProtoPPP LinkProto = "PPP"

	// Ethernet protocol
	LinkProtoEthernet LinkProto = "ETHERNET"
)

// LinkProto is a data link layer protocol.
type LinkProto string

// LinkAddr represents a L2 address.
type LinkAddr interface {
	// String returns string form of address.
	String() string
}

// LinkContext wraps link layer resources and provides
// methods for accessing other link information.
type LinkContext interface {
	// Link layer address.
	Addr() LinkAddr

	// Link communication mode.
	Mode() LinkMode

	// Link bandwidth.
	Bandwidth() int
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
type LinkMechanismManager struct {
	// Base mechanism manager
	MechanismManager

	// Link layer context
	context LinkContext
}

// Context returns link layer context
func (m *LinkMechanismManager) Context() LinkContext {
	return m.context
}

// Iter calls specified function for all registered link layer mechanisms.
func (m *LinkMechanismManager) Iter(fn func(LinkMechanism) bool) {
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

func (m *LinkMechanismManager) do(context *LinkContext, fn linkMechanismFunc) (err error) {
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

// UpdateLink calls corresponding method for activated mechanisms.
func (m *LinkMechanismManager) UpdateLink(context *LinkContext) (err error) {
	return m.do(context, LinkMechanism.UpdateLink)
}

// DeleteLink calls corresponding method for activated mechanisms.
func (m *LinkMechanismManager) DeleteLink(context *LinkContext) (err error) {
	return m.do(context, LinkMechanism.DeleteLink)
}

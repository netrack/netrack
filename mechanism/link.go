package mech

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
type LinkContext struct {
	// Link layer address.
	Addr LinkAddr

	// Link communication mode.
	Mode LinkMode

	// Link bandwidth.
	Bandwidth int
}

// LinkMechanism is the interface implemented by an object
// that can handle OSI L2 network types.
type LinkMechanism interface {
	Mechanism

	// Proto returns link layer encapsulation protocol.
	Proto() LinkProto

	// Addr returns link layer address (hardware address).
	Addr() LinkAddr

	SetAddr(LinkAddr) error

	// UpdateLink is called for all changes to link state.
	UpdateLink(*LinkContext) error

	// Encapsulate wraps reader data with link layer frame.
	//Encapsulate(io.Reader) (io.Reader, error)
}

// LinkMechanismConstructor is a genereic
// constructor for data link type mechanisms.
type LinkMechanismContructor interface {
	// New returns a new LinkMechanism instance.
	New() LinkMechanism
}

// LinkMechanismConstructorFunc is a function adapter for
// LinkMechanismConstructor.
type LinkMechanismContructorFunc func() LinkMechanism

func (fn LinkMechanismContructorFunc) New() LinkMechanism {
	return fn()
}

var links = make(LinkMechanismMap)

func RegisterLinkMechanism(name string, mechanism LinkMechanismContructor) {
	if mechanism == nil {
		log.FatalLog("mechanism/REGISTER_LINK_MECHANISM",
			"Failed to register nil link mechanism for: ", name)
	}

	if _, dup := links[name]; dup {
		log.FatalLog("mechanism/REGISTER_LINK_MECHANISM",
			"Falied to register duplicate link mechanism for: ", name)
	}

	links[name] = mechanism
}

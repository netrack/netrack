package mech

// ExtensionMechanism is the interface implemented by an object
// that can provide additional functionality.
type ExtensionMechanism interface {
	Mechanism
}

// ExtensionMechanismContructor is a generic
// constructor for network type mechanisms.
type ExtensionMechanismContructor interface {
	// New returns a new ExtensionMechanism instance.
	New() ExtensionMechanism
}

// ExtensionMechanismConstructorFunc is a function adapter for
// ExtensionMechanismConstructor.
type ExtensionTypeConstructorFunc func() NewtorkType

func (fn ExtensionMechanismContructorFunc) New() ExtensionMechanism {
	return fn()
}

var extentions = make(map[string]ExtensionMechanismContructor)

func RegisterExtensionMechanism(name string, mechanism ExtensionMechanismContructor) {
	if mechanism == nil {
		log.FatalLog("mechanism/REGISTER_EXTENSION_MECHANISM",
			"Failed to register nil extension mechanism for: ", name)
	}

	if _, dup := networks[name]; dup {
		log.FatalLog("mechanism/REGISTER_NETWORK_MECHANISM",
			"Falied to register duplicate extension mechanism for: ", name)
	}

	extensions[name] = mechanism
}

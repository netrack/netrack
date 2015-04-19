package mech

import (
	"github.com/netrack/netrack/logging"
)

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
type ExtensionMechanismConstructorFunc func() ExtensionMechanism

func (fn ExtensionMechanismConstructorFunc) New() ExtensionMechanism {
	return fn()
}

// ExtensionMechanismMap implements MechanismMap interface.
type ExtensionMechanismMap map[string]ExtensionMechanism

// Get returns Mechanism by specified name.
func (m ExtensionMechanismMap) Get(s string) (Mechanism, bool) {
	mechanism, ok := m[s]
	return mechanism, ok
}

// Set registers mechanism under specified name.
func (m ExtensionMechanismMap) Set(s string, mechanism Mechanism) {
	m[s] = mechanism
}

// Iter calls specified function for all registered mechanisms.
func (m ExtensionMechanismMap) Iter(fn func(string, Mechanism) bool) {
	for s, mechanism := range m {
		fn(s, mechanism)
	}
}

var extensions = make(map[string]ExtensionMechanismContructor)

// RegisterExtensionMechanism registers a new extension mechanism
// under specified name.
func RegisterExtensionMechanism(name string, ctor ExtensionMechanismContructor) {
	if ctor == nil {
		log.FatalLog("extension/REGISTER_EXTENSION_MECHANISM",
			"Failed to register nil extension constructor for: ", name)
	}

	if _, dup := networks[name]; dup {
		log.FatalLog("extension/REGISTER_EXTENSION_MECHANISM",
			"Falied to register duplicate extension constructor for: ", name)
	}

	extensions[name] = ctor
}

// ExtensionMechanisms returns map of registered extension mechanisms.
func ExtensionMechanisms() ExtensionMechanismMap {
	emap := make(ExtensionMechanismMap)

	for name, constructor := range extensions {
		emap.Set(name, constructor.New())
	}

	return emap
}

// ExtensionManager manages extension mechanisms.
type ExtensionMechanismManager struct {
	// Base mechanism manager.
	MechanismManager
}

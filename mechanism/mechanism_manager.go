package mech

import (
	"errors"
)

var (
	// ErrMechanismNotRegistered is returned on
	// not registered mechanism operations.
	ErrMechanismNotRegistered = errors.New(
		"MechanismManager: mechanism not registered")

	// ErrMechanismAlreadyEnabled is returned on enabling
	// of already enabled mechanism.
	ErrMechanismAlreadyEnabled = errors.New(
		"MechanismManager: mechanism already enabled")

	// ErrMechanismAlreadyActivated is returned on
	// activating of already activated mechanism.
	ErrMechanismAlreadyActivated = errors.New(
		"MechanismManager: mechanism already activated")

	// ErrMechanismAlreadyDisabled is returned on
	// disabling of already disabled mechanism.
	ErrMechanismAlreadyDisabled = errors.New(
		"MechanismManager: mechanism already disabled")
)

// MechanismMap describes map for mechanism type.
type MechanismMap interface {
	// Get returns Mechanism by registered name.
	Get(string) (Mechanism, bool)

	// Set saves Mechanism under specified name.
	Set(string, Mechanism)

	// Iter call specified function for each element of map.
	Iter(func(string, Mechansim) bool)
}

// MechanismManager manages networking
// mechanisms using drivers.
type MechanismManager struct {
	Mechanisms MechansimMap
}

// Enable enables all registered mechanisms
func (m *MechanismManager) Enable(c *MechanismContext) {
	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		mechanism.Enable(c)
		return true
	})
}

// EnableByName enables mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already enabled.
func (m *MechanismManager) EnableByName(name string, c *MechanismContext) error {
	mechanism, ok := m.Mechanisms.Get(name)
	if !ok {
		return ErrMechanismNotRegistered
	}

	if mechanism.Enabled() {
		return ErrMechanismAlreadyEnabled
	}

	mechanism.Enable(c)
	return nil
}

// Activate activates all registered mechanisms
func (m *MechanismManager) Activate() {
	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		mechanism.Activate()
		return true
	})
}

// ActivateByName activates mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already activated.
func (m *MechanismManager) ActivateByName(name string) error {
	mechanism, ok := m.Mechanisms.Get(name)
	if !ok {
		return ErrMechanismNotRegistered
	}

	if mechanism.Activated() {
		return ErrMechanismAlreadyActivated
	}

	mechanism.Activate()
	return nil
}

// Disable disables all registered mechanisms
func (m *MechanismManager) Disable() {
	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		mechanism.Disable()
		return true
	})
}

// DisableByName disables mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already disabled.
func (m *MechanismManager) DisableByName(name string) error {
	mechanism, ok := m.Mechanisms.Get(name)
	if !ok {
		return ErrMechanismNotRegistered
	}

	if !mechanism.Enabled() {
		return ErrMechanismAlreadyDisabled
	}

	mechanism.Disable()
	return nil
}

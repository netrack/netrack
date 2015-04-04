package mech

import (
	"errors"
)

var (
	// ErrMechanismNotRegistered is returned on not registered mechanism operations.
	ErrMechanismNotRegistered = errors.New("MechanismDriverManager: mechanism not registered")

	// ErrMechanismAlreadyEnabled is returned on enabling of already enabled mechanism.
	ErrMechanismAlreadyEnabled = errors.New("MechanismDriverManager: mechanism already enabled")

	// ErrMechanismAlreadyActivated is returned on activating of already activated mechanism.
	ErrMechanismAlreadyActivated = errors.New("MechanismDriverManager: mechanism already activated")

	// ErrMechanismAlreadyDisabled is returned on disabling of already disabled mechanism.
	ErrMechanismAlreadyDisabled = errors.New("MechanismDriverManager: mechanism already disabled")
)

// MechanismDriverManager manages networking
// mechanisms using drivers.
type MechanismDriverManager struct {
	Mechanisms map[string]MechanismDriver
}

// Enable enables all registered mechanisms
func (m *MechanismDriverManager) Enable(c *MechanismDriverContext) {
	for _, mechanism := range m.Mechanisms {
		mechanism.Enable(c)
	}
}

// EnableByName enables mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already enabled.
func (m *MechanismDriverManager) EnableByName(name string, c *MechanismDriverContext) error {
	mechanism, ok := m.Mechanisms[name]
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
func (m *MechanismDriverManager) Activate() {
	for _, mechanism := range m.Mechanisms {
		mechanism.Activate()
	}
}

// ActivateByName activates mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already activated.
func (m *MechanismDriverManager) ActivateByName(name string) error {
	mechanism, ok := m.Mechanisms[name]
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
func (m *MechanismDriverManager) Disable() {
	for _, mechanism := range m.Mechanisms {
		mechanism.Disable()
	}
}

// DisableByName disables mechanism driver by specified name,
// error will be returned, when mechanism was not registered
// or specified mechanism already disabled.
func (m *MechanismDriverManager) DisableByName(name string) error {
	mechanism, ok := m.Mechanisms[name]
	if !ok {
		return ErrMechanismNotRegistered
	}

	if !mechanism.Enabled() {
		return ErrMechanismAlreadyDisabled
	}

	mechanism.Disable()
	return nil

}

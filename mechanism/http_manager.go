package mech

// HTTPDriverManager manages registered HTTP drivers.
type HTTPDriverManager struct {
	// List of registered drivers.
	drivers []HTTPDriver
}

// Enables performs intialization of registered drivers.
func (m *HTTPDriverManager) Enable(c *HTTPDriverContext) {
	if m.drivers != nil {
		// Drivers are enabled, nothing to do.
		return
	}

	constructors := HTTPDriverList()
	for _, constructor := range constructors {
		driver := constructor.New()
		driver.Enable(c)

		m.drivers = append(m.drivers, driver)
	}
}

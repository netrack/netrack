package mech

import (
	"github.com/netrack/netrack/httputil"
)

// HTTPDriverContext is a context, that shared among HTTP drivers.
type HTTPDriverContext struct {
	// HTTP multiplexer handler.
	Mux *httputil.ServeMux

	// SwitchManager instance.
	SwitchManager *SwitchManager
}

// HTTPDriver describes HTTP control handlers.
type HTTPDriver interface {
	// Enable performs initial driver intialization, like
	// registering HTTP handlers to multiplexer.
	Enable(*HTTPDriverContext)
}

// BaseHTTPDriver implements basic methods of HTTPDriver interface.
type BaseHTTPDriver struct {
	// HTTPDriverContext instance
	C *HTTPDriverContext
}

// Enable implements HTTPDriver interface.
func (m *BaseHTTPDriver) Enable(c *HTTPDriverContext) {
	m.C = c
}

// HTTPDriverConstructor is a generic
// constructor for http handlers.
type HTTPDriverContructor interface {
	// New creates a new HTTPDriver instance
	New() HTTPDriver
}

// HTTPDriverConstructorFunc is a function adapter for
// HTTPDriverConstructor
type HTTPDriverConstructorFunc func() HTTPDriver

// New implements HTTPDriverConstructor interface.
func (fn HTTPDriverConstructorFunc) New() HTTPDriver {
	return fn()
}

var handlers []HTTPDriverContructor

// RegisterHTTPDriver makes HTTP driver available.
func RegisterHTTPDriver(c HTTPDriverContructor) {
	handlers = append(handlers, c)
}

// HTTPDriverList returns list of registered
// HTTP drivers constructors.
func HTTPDriverList() []HTTPDriverContructor {
	drivers := make([]HTTPDriverContructor, len(handlers))
	copy(drivers, handlers)
	return drivers
}

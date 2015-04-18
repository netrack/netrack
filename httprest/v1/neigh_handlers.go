package httprest

import (
	"net/http"

	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	// Register management HTTP interface.
	constructor := mech.HTTPDriverConstructorFunc(NewNeighHandler)
	mech.RegisterHTTPDriver(constructor)
}

// NeighHandler provides HTTP API for management
// of IPv4 neighbour table (ARP cache).
type NeighHandler struct {
	// Base HTTP driver instance.
	mech.BaseHTTPDriver
}

// NewNeighHandler creates a new instance of NeightMgmt type.
func NewNeighHandler() mech.HTTPDriver {
	return &NeighHandler{}
}

func (h *NeighHandler) Enable(c *mech.HTTPDriverContext) {
	h.BaseHTTPDriver.Enable(c)

	h.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ipv4/neigh", h.createHandler)
	h.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ipv4/neigh", h.indexHandler)
	h.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ipv4/neigh", h.destroyHandler)

	log.InfoLog("neigh_mgmt/ENABLE_HOOK",
		"Neight management enabled")
}

func (h *NeighHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
}

func (h *NeighHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
}

func (h *NeighHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
}

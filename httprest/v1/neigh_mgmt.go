package httprest

import (
	"net/http"

	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	// Register management HTTP interface.
	constructor := mech.HTTPDriverConstructorFunc(NewNeighMgmt)
	mech.RegisterHTTPDriver(constructor)
}

// NeighMgmt provides HTTP API for management
// of IPv4 neighbour table (ARP cache).
type NeighMgmt struct {
	// Base HTTP driver instance.
	mech.BaseHTTPDriver
}

// NewNeighMgmt creates a new instance of NeightMgmt type.
func NewNeighMgmt() mech.HTTPDriver {
	return &NeighMgmt{}
}

func (m *NeighMgmt) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ipv4/neigh", m.createHandler)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ipv4/neigh", m.indexHandler)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ipv4/neigh", m.destroyHandler)

	log.InfoLog("neigh_mgmt/ENABLE_HOOK",
		"Neight management enabled")
}

func (m *NeighMgmt) createHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *NeighMgmt) indexHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *NeighMgmt) destroyHandler(rw http.ResponseWriter, r *http.Request) {
}

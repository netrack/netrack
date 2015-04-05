package rest

import (
	"github.com/netrack/netrack/mechanism"
)

type NeighMgmt struct {
	mech.BaseHTTPDriver
}

func (m *NeighMgmt) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ip/neigh", nil)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ip/neigh", nil)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ip/neigh", nil)
}

package rest

import (
	"github.com/netrack/netrack/mechanism"
)

type RouteMgmt struct {
	mech.BaseHTTPDriver
}

func (m *RouteMgmt) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ip/route", nil)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ip/route", nil)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ip/route", nil)
}

package httprest

import (
	"net/http"

	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
)

type RouteMgmt struct {
	mech.BaseHTTPDriver
}

func NewRouteMgmt() mech.HTTPDriver {
	return &RouteMgmt{}
}

func (m *RouteMgmt) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ipv4/route", nil)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ipv4/route", nil)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ipv4/route", nil)

	log.InfoLog("address_mgmt/ENABLE",
		"Route management enabled")
}

func (m *RouteMgmt) putHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *RouteMgmt) getHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *RouteMgmt) deleteHandler(rw http.ResponseWriter, r *http.Request) {
}

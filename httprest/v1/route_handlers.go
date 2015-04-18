package httprest

import (
	"net/http"

	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
)

type RouteHandler struct {
	mech.BaseHTTPDriver
}

func NewRouteHandler() mech.HTTPDriver {
	return &RouteHandler{}
}

func (m *RouteHandler) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ipv4/route", nil)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ipv4/route", nil)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ipv4/route", nil)

	log.InfoLog("route_mgmt/ENABLE_HOOK",
		"Route management enabled")
}

func (m *RouteHandler) putHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *RouteHandler) getHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *RouteHandler) deleteHandler(rw http.ResponseWriter, r *http.Request) {
}

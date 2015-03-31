package rest

import (
	"fmt"
	"net/http"

	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
)

type RouteMgmt struct {
	C *mech.HTTPContext
}

func (m *RouteMgmt) Intialize(c *mech.HTTPContext) {
	m.C = c

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ip/route", nil)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ip/route", nil)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ip/route", nil)
}

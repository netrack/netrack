package rest

import (
	"net/http"

	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	constructor := mech.HTTPDriverConstructorFunc(NewAddressMgmt)
	mech.RegisterHTTPDriver(constructor)
}

// IP protocol address management
type AddressMgmt struct {
	mech.BaseHTTPDriver
}

func NewAddressMgmt() mech.HTTPDriver {
	return &AddressMgmt{}
}

func (m *AddressMgmt) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ip/address", m.indexHandler)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ip/address", m.showHandler)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ip/address", m.deleteHandler)

	log.InfoLog("address_mgmt/ENABLE",
		"IP address management enabled")
}

func (m *AddressMgmt) indexHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *AddressMgmt) showHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *AddressMgmt) deleteHandler(rw http.ResponseWriter, r *http.Request) {
}

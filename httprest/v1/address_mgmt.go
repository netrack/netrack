package httprest

import (
	"net/http"

	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	constructor := mech.HTTPDriverConstructorFunc(NewAddressMgmt)
	mech.RegisterHTTPDriver(constructor)
}

type IPv4Model struct {
	Addr string `json:"address"`
	Netw string `json:"network"`
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

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/interfaces/{interface}/ipv4/address", m.createHandler)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/interfaces/{interface}/ipv4/address", m.showHandler)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/interfaces/{interface}/ipv4/address", m.destroyHandler)

	log.InfoLog("address_mgmt/ENABLE",
		"IP address management enabled")
}

func (m *AddressMgmt) createHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *AddressMgmt) showHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *AddressMgmt) destroyHandler(rw http.ResponseWriter, r *http.Request) {
}

package httprest

import (
	"net/http"

	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	constructor := mech.HTTPDriverConstructorFunc(NewNetworkMechanismHandler)
	mech.RegisterHTTPDriver(constructor)
}

type NetworkMechanismHandler struct {
	// Base HTTP driver
	mech.BaseHTTPDriver
}

func NewNetworkMechanismHandler() mech.HTTPDriver {
	return &NetworkMechanismHandler{}
}

func (h *NetworkMechanismHandler) Enable(c *mech.HTTPDriverContext) {
	h.BaseHTTPDriver.Enable(c)

	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/network/mechanisms", h.indexHandler)
	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/network/mechanisms/{mechanism}", h.showHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/network/mechanisms/{mechanism}/enable", h.enableHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/network/mechanisms/{mechanism}/disable", h.disableHandler)

	log.InfoLog("network_mechanism_handlers/ENABLE_NOOK",
		"Network mechanism handlers enabled")
}

func (h *NetworkMechanismHandler) manager(fn func(interface{}) error) (mech.MechanismManager, error) {
	var lnetwork mech.NetworkMechanismManager

	if err := fn(&lnetwork); err != nil {
		return nil, err
	}

	return lnetwork, nil
}

func (h *NetworkMechanismHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_mechanism_handlers/INDEX_HANDLER",
		"Got request to list network layer mechanisms")

	handler := MechanismHandler{h.C, h.manager}
	handler.indexHandler(rw, r)
}

func (h *NetworkMechanismHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_mechanism_handlers/SHOW_HANDLER",
		"Got request to show network layer mechanism")

	handler := MechanismHandler{h.C, h.manager}
	handler.showHandler(rw, r)
}

func (h *NetworkMechanismHandler) enableHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_mechanism_handlers/ENABLE_HANDLER",
		"Got request to enable network layer mechanism")

	handler := MechanismHandler{h.C, h.manager}
	handler.enableHandler(rw, r)
}

func (h *NetworkMechanismHandler) disableHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_mechanism_handlers/DISABLE_HANDLER",
		"Got request to disable network layer mechanism")

	handler := MechanismHandler{h.C, h.manager}
	handler.disableHandler(rw, r)
}

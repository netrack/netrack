package httprest

import (
	"net/http"

	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	constructor := mech.HTTPDriverConstructorFunc(NewLinkMechanismHandler)
	mech.RegisterHTTPDriver(constructor)
}

type LinkMechanismHandler struct {
	// Base HTTP driver
	mech.BaseHTTPDriver
}

func NewLinkMechanismHandler() mech.HTTPDriver {
	return &LinkMechanismHandler{}
}

func (h *LinkMechanismHandler) Enable(c *mech.HTTPDriverContext) {
	h.BaseHTTPDriver.Enable(c)

	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/link/mechanisms", h.indexHandler)
	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/link/mechanisms/{mechanism}", h.showHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/link/mechanisms/{mechanism}/enable", h.enableHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/link/mechanisms/{mechanism}/disable", h.disableHandler)

	log.InfoLog("link_mechanism_handlers/ENABLE_NOOK",
		"Link mechanism handlers enabled")
}

func (h *LinkMechanismHandler) manager(fn func(interface{}) error) (mech.MechanismManager, error) {
	var llink mech.LinkMechanismManager

	if err := fn(&llink); err != nil {
		return nil, err
	}

	return llink, nil
}

func (h *LinkMechanismHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_mechanism_handlers/INDEX_HANDLER",
		"Got request to list link layer mechanisms")

	handler := MechanismHandler{h.C, h.manager}
	handler.indexHandler(rw, r)
}

func (h *LinkMechanismHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_mechanism_handlers/SHOW_HANDLER",
		"Got request to show link layer mechanism")

	handler := MechanismHandler{h.C, h.manager}
	handler.showHandler(rw, r)
}

func (h *LinkMechanismHandler) enableHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_mechanism_handlers/ENABLE_HANDLER",
		"Got request to enable link layer mechanism")

	handler := MechanismHandler{h.C, h.manager}
	handler.enableHandler(rw, r)
}

func (h *LinkMechanismHandler) disableHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_mechanism_handlers/DISABLE_HANDLER",
		"Got request to disable link layer mechanism")

	handler := MechanismHandler{h.C, h.manager}
	handler.disableHandler(rw, r)
}

package httprest

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/netrack/netrack/httprest/format"
	"github.com/netrack/netrack/httprest/v1/models"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	// Register interface management HTTP API driver.
	constructor := mech.HTTPDriverConstructorFunc(NewInterfaceHandler)
	mech.RegisterHTTPDriver(constructor)
}

// InterfaceHandler provides HTTP API for management of switch ports.
type InterfaceHandler struct {
	// Base HTTP driver instance.
	mech.BaseHTTPDriver
}

// NewInterfaceHandler create a new instance of InterfaceHandler type.
func NewInterfaceHandler() mech.HTTPDriver {
	return &InterfaceHandler{}
}

func (h *InterfaceHandler) Enable(c *mech.HTTPDriverContext) {
	h.BaseHTTPDriver.Enable(c)

	h.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/interfaces", h.indexHandler)
	h.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/interfaces/{interface}", h.showHandler)

	log.InfoLog("iface_mgmt/ENABLE_HOOK",
		"Interface management enabled")
}

// indexHandler returns list of interfaces of specified switch.
func (h *InterfaceHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	// log.InfoLog("iface_mgmt/INDEX_HANDLER",
	// 	"Got request to list interfaces")

	// f, err := format.Format(r.Header.Get(httputil.HeaderAccept))
	// if err != nil {
	// 	log.ErrorLog("iface_mgmt/INDEX_HANDLER",
	// 		"Failed to select Accept formatter for request: ", err)

	// 	formats := strings.Join(format.FormatNameList(), ", ")

	// 	f.Write(rw, r, models.Error{fmt.Sprintf("only '%s' are supported", formats)})
	// 	rw.WriteHeader(http.StatusNotAcceptable)
	// 	return
	// }

	// dpid := httputil.Param(r, "dpid")
	// log.DebugLog("iface_mgmt/INDEX_HANDLER",
	// 	"Request list interfaces of: ", dpid)

	// c, err := h.C.SwitchManager.SwitchContextByID(dpid)
	// if err != nil {
	// 	log.ErrorLog("iface_mgmt/INDEX_HANDLER",
	// 		"Failed to find requested datapath: ", err)

	// 	f.Write(rw, r, models.Error{fmt.Sprintf("switch '%s' not found", dpid)})
	// 	rw.WriteHeader(http.StatusNotFound)
	// 	return
	// }

	// if err = f.Write(rw, r, c.Context.Switch.PortNameList()); err != nil {
	// 	log.ErrorLog("iface_mgmt/INDEX_HANDLER",
	// 		"Failed to write list of interface names: ", err)

	// 	rw.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
}

// showHandler returns description of the specified switch interface.
func (h *InterfaceHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
}

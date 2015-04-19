package httprest

import (
	"fmt"
	//"net"
	"net/http"
	//"strings"

	//"github.com/netrack/netrack/httprest/format"
	"github.com/netrack/netrack/httprest/v1/models"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	//"github.com/netrack/netrack/mechanism/rpc"
)

func init() {
	// Register address management HTTP API driver.
	constructor := mech.HTTPDriverConstructorFunc(NewNetworkHandler)
	mech.RegisterHTTPDriver(constructor)
}

// L3 protocol address management
type NetworkHandler struct {
	// Base HTTP driver instance.
	mech.BaseHTTPDriver
}

// NewNetworkHandler creates a new instance of NetworkHandler type.
func NewNetworkHandler() mech.HTTPDriver {
	return &NetworkHandler{}
}

// Enable implements HTTPDriver interface.
func (h *NetworkHandler) Enable(c *mech.HTTPDriverContext) {
	h.BaseHTTPDriver.Enable(c)

	h.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/interfaces/l3/address", h.indexHandler)
	h.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/interfaces/{interface}/l3/address", h.showHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/interfaces/{interface}/l3/address", h.createHandler)
	h.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/interfaces/{interface}/l3/address", h.destroyHandler)

	log.InfoLog("network_handlers/ENABLE_HOOK",
		"IP address management enabled")
}

func (h *NetworkHandler) context(rw http.ResponseWriter, r *http.Request) (*mech.SwitchContext, mech.SwitchPort, error) {
	log.InfoLog("network_handlers/SWICH_PORT",
		"Got request to handle L3 address")

	dpid := httputil.Param(r, "dpid")
	iface := httputil.Param(r, "interface")

	f := WriteFormat(r)

	log.DebugLogf("network_handlers/CONTEXT",
		"Request handle L3 address of: %s dev %s", dpid, iface)

	context, err := h.C.SwitchManager.SwitchContext(dpid)
	if err != nil {
		log.ErrorLog("network_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		f.Write(rw, r, models.Error{text})
		rw.WriteHeader(http.StatusNotFound)

		return nil, nil, fmt.Errorf(text)
	}

	port, err := context.Switch.PortByName(iface)
	if err != nil {
		log.ErrorLog("network_handlers/CONTEXT",
			"Failed to find requested interface: ", iface)

		text := fmt.Sprintf("switch '%s' does not have '%s' interface", dpid, iface)

		f.Write(rw, r, models.Error{text})
		rw.WriteHeader(http.StatusNotFound)

		return nil, nil, fmt.Errorf(text)
	}

	return context, port, nil
}

func (h *NetworkHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
}

func (h *NetworkHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/CREATE_HANDLER",
		"Got request to create L3 address")

	rf, wf := Format(r)

	switchContext, switchPort, err := h.context(rw, r)
	if err != nil {
		return
	}

	var requestAddr models.NetworkAddr
	if err = rf.Read(rw, r, &requestAddr); err != nil {
		log.ErrorLog("network_handlers/CREATE_HANDLER",
			"Failed to read request body: ", err)

		wf.Write(rw, r, models.Error{"failed to read request body"})
		rw.WriteHeader(http.StatusBadRequest)

		return
	}

	networkDrv := switchContext.Networks.NetworkDriver()
	networkAddr, err := networkDrv.ParseAddr(requestAddr.Addr)
	if err != nil {
		log.ErrorLog("network_handlers/CREATE_HANDLER",
			"Failed to parse request address: ", err)

		wf.Write(rw, r, models.Error{"failed to parse request"})
		rw.WriteHeader(http.StatusBadRequest)

		return
	}

	networkContext := mech.NetworkContext{
		Addr: networkAddr,
		Port: switchPort.Name(),
	}

	err = switchContext.Networks.UpdateNetwork(networkContext)
	if err != nil {
		log.ErrorLog("network_handlers/CREATE_HANDLER",
			"Failed to createa a new L3 address: ", err)

		wf.Write(rw, r, models.Error{"failed create a new L3 address"})
		rw.WriteHeader(http.StatusConflict)

		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (h *NetworkHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/SHOW_HANDLER",
		"Got request to show L3 address")

	// rf, wf := Format(r)

	// port, err := h.switchPort(rw, r)
	// if err != nil {
	// 	return
	// }

	// network := driver.Network()
	// if network.Addr == "" {
	// 	log.ErrorLog("network_handlers/SHOW_HANDLER",
	// 		"Failed to find L3 protocol on interface: ", dpid)

	// 	text := fmt.Sprintf("protocol '%s' enabled, but address "+
	// 		"is not assigned to '%s' interface", network.Proto, iface)

	// 	wf.Write(rw, r, models.Error{text})
	// 	rw.WriteHeader(http.StatusNotFound)

	// 	return
	// }

	// // Return L3 address
	// wf.Write(rw, r, models.NetworkAddr{
	// 	Type: string(driver.Proto),
	// 	Addr: network.Addr.String(),
	// })

	// rw.WriteHeader(http.StatusOK)
}

func (h *NetworkHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/DESTROY_HANDLER",
		"Got request to destroy L3 address")

	//rf, wf := Format(r)

	// port, err := h.switchPort(rw, r)
	// if err != nil {
	// 	return
	// }

	// driver := port.NetworkDriver()
	// driver.DeteleNework(mech.NetworkContext{})

	// rw.WriteHeader(http.StatusOK)
}

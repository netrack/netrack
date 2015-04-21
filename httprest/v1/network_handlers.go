package httprest

import (
	"fmt"
	"net/http"

	"github.com/netrack/netrack/httprest/v1/models"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
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

	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/interfaces/network", h.indexHandler)
	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/interfaces/{interface}/network", h.showHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/interfaces/{interface}/network", h.createHandler)
	h.C.Mux.HandleFunc("DELETE", "/v1/datapaths/{dpid}/interfaces/{interface}/network", h.destroyHandler)

	log.InfoLog("network_handlers/ENABLE_HOOK",
		"IP address management enabled")
}

func (h *NetworkHandler) context(rw http.ResponseWriter, r *http.Request) (*mech.MechanismContext, *mech.SwitchPort, error) {
	log.InfoLog("network_handlers/CONTEXT",
		"Got request to handle L3 address")

	dpid := httputil.Param(r, "dpid")
	iface := httputil.Param(r, "interface")

	f := WriteFormat(r)

	log.DebugLogf("network_handlers/CONTEXT",
		"Request handle L3 address of: %s dev %s", dpid, iface)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("network_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		rw.WriteHeader(http.StatusNotFound)
		f.Write(rw, r, models.Error{text})
		return nil, nil, fmt.Errorf(text)
	}

	port, err := context.Switch.PortByName(iface)
	if err != nil {
		log.ErrorLog("network_handlers/CONTEXT",
			"Failed to find requested interface: ", iface)

		text := fmt.Sprintf("switch '%s' does not have '%s' interface", dpid, iface)

		rw.WriteHeader(http.StatusNotFound)
		f.Write(rw, r, models.Error{text})
		return nil, nil, fmt.Errorf(text)
	}

	return context, port, nil
}

func (h *NetworkHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/INDEX_HANDLER",
		"Got request to list network layer addresses")

	dpid := httputil.Param(r, "dpid")
	wf := WriteFormat(r)

	log.DebugLogf("network_handlers/INDEX_HANDLER",
		"Request list networks of: %s", dpid)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("network_handlers/INDEX_HANDLER",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		rw.WriteHeader(http.StatusNotFound)
		wf.Write(rw, r, models.Error{text})
		return
	}

	var networks []models.Network

	for _, port := range context.Switch.PortList() {
		network, _ := context.Network.Context(port.Number)

		networks = append(networks, models.Network{
			Encapsulation: models.NullString(network.Driver),
			Addr:          models.NullString(network.Addr),
			InterfaceName: port.Name,
			Interface:     port.Number,
		})
	}

	rw.WriteHeader(http.StatusOK)
	wf.Write(rw, r, networks)
}

func (h *NetworkHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/CREATE_HANDLER",
		"Got request to create network layer address")

	rf, wf := Format(r)

	switchContext, switchPort, err := h.context(rw, r)
	if err != nil {
		return
	}

	var network models.Network
	if err = rf.Read(rw, r, &network); err != nil {
		log.ErrorLog("network_handlers/CREATE_HANDLER",
			"Failed to read request body: ", err)

		rw.WriteHeader(http.StatusBadRequest)
		wf.Write(rw, r, models.Error{"failed to read request body"})
		return
	}

	context := &mech.NetworkManagerContext{
		Addr:   network.Addr.String(),
		Driver: network.Encapsulation.String(),
		Port:   switchPort.Number,
	}

	if err = switchContext.Network.UpdateNetwork(context); err != nil {
		log.ErrorLog("network_handlers/CREATE_HANDLER",
			"Failed to createa a new L3 address: ", err)

		rw.WriteHeader(http.StatusConflict)
		wf.Write(rw, r, models.Error{"failed update network"})
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (h *NetworkHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/SHOW_HANDLER",
		"Got request to show L3 address")

	wf := WriteFormat(r)

	context, switchPort, err := h.context(rw, r)
	if err != nil {
		return
	}

	network, _ := context.Network.Context(switchPort.Number)

	// Return interface network data.
	rw.WriteHeader(http.StatusOK)
	wf.Write(rw, r, models.Network{
		Encapsulation: models.NullString(network.Driver),
		Addr:          models.NullString(network.Addr),
		InterfaceName: switchPort.Name,
		Interface:     switchPort.Number,
	})
}

func (h *NetworkHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/DESTROY_HANDLER",
		"Got request to destroy L3 address")

	// driver.DeteleNework(mech.NetworkManagerContext{})
}

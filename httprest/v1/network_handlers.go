package httprest

import (
	"fmt"
	"net/http"

	"github.com/netrack/netrack/httprest/format"
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

type NetworkHandlerContext struct {
	// Back-end context
	Mech *mech.MechanismContext

	// Network layer manager instance.
	Network mech.NetworkMechanismManager

	// Network layer manager context.
	NetworkContext *mech.NetworkManagerContext

	// SwitchPort instance
	Port *mech.SwitchPort

	// Write formatter
	W format.WriteFormatter

	// Read formatter
	R format.ReadFormatter
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

	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/network/interfaces", h.indexHandler)
	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/network/interfaces/{interface}", h.showHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/network/interfaces/{interface}", h.createHandler)
	h.C.Mux.HandleFunc("DELETE", "/v1/datapaths/{dpid}/network/interfaces/{interface}", h.destroyHandler)

	log.InfoLog("network_handlers/ENABLE_HOOK",
		"Network layer handlers enabled")
}

func (h *NetworkHandler) context(rw http.ResponseWriter, r *http.Request) (*NetworkHandlerContext, error) {
	log.InfoLog("network_handlers/CONTEXT",
		"Got request to handle L3 address")

	dpid := httputil.Param(r, "dpid")
	iface := httputil.Param(r, "interface")

	rf, wf := Format(r)

	log.DebugLogf("network_handlers/CONTEXT",
		"Request handle L3 address of: %s dev %s", dpid, iface)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("network_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		wf.Write(rw, models.Error{text}, http.StatusNotFound)
		return nil, fmt.Errorf(text)
	}

	var port *mech.SwitchPort

	if iface != "" {
		port, err = context.Switch.PortByName(iface)
		if err != nil {
			log.ErrorLog("network_handlers/CONTEXT",
				"Failed to find requested interface: ", iface)

			text := fmt.Sprintf("switch '%s' does not have '%s' interface", dpid, iface)

			wf.Write(rw, models.Error{text}, http.StatusNotFound)
			return nil, fmt.Errorf(text)
		}
	}

	var nnetwork mech.NetworkMechanismManager
	if err := context.Managers.Obtain(&nnetwork); err != nil {
		log.ErrorLog("network_handlers/NETWORK_LAYER_MANAGER",
			"Failed to obtain network layer manager: ", err)

		text := fmt.Sprintf("network layer manager is dead")
		wf.Write(rw, models.Error{text}, http.StatusInternalServerError)
		return nil, err
	}

	networkContext, err := nnetwork.Context()
	if err != nil {
		log.ErrorLog("network_handlers/LINK_LAYER_CONTEXT",
			"Failed to get link layer context: ", err)

		text := fmt.Sprintf("network layer context inaccessible")
		wf.Write(rw, models.Error{text}, http.StatusConflict)
		return nil, err
	}

	ctx := &NetworkHandlerContext{
		NetworkContext: networkContext,
		Mech:           context,
		Network:        nnetwork,
		Port:           port,
		W:              wf,
		R:              rf,
	}

	return ctx, nil
}

func (h *NetworkHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/INDEX_HANDLER",
		"Got request to list network layer addresses")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	networkModels := make([]models.Network, 0)

	for _, switchPort := range context.Mech.Switch.PortList() {
		networkPort := context.NetworkContext.Port(switchPort.Number)

		networkModels = append(networkModels, models.Network{
			Encapsulation: models.NullString(context.NetworkContext.Driver),
			Addr:          models.NullString(networkPort.Addr),
			InterfaceName: switchPort.Name,
			Interface:     switchPort.Number,
		})
	}

	context.W.Write(rw, networkModels, http.StatusOK)
}

func (h *NetworkHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/CREATE_HANDLER",
		"Got request to create network layer address")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	var networkModel models.Network
	if err = context.R.Read(r, &networkModel); err != nil {
		log.ErrorLog("network_handlers/CREATE_HANDLER",
			"Failed to read request body: ", err)

		body := models.Error{"failed to read request body"}
		context.W.Write(rw, body, http.StatusBadRequest)
		return
	}

	port := mech.NetworkPort{
		Addr: networkModel.Addr.String(),
		Port: context.Port.Number,
	}

	networkContext := &mech.NetworkManagerContext{
		Datapath: context.Mech.Switch.ID(),
		Driver:   networkModel.Encapsulation.String(),
		Ports:    []mech.NetworkPort{port},
	}

	if err = context.Network.UpdateNetwork(networkContext); err != nil {
		log.ErrorLog("network_handlers/CREATE_HANDLER",
			"Failed to createa a new L3 address: ", err)

		body := models.Error{"failed update network"}
		context.W.Write(rw, body, http.StatusConflict)
		return
	}

	context.W.Write(rw, nil, http.StatusOK)
}

func (h *NetworkHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/SHOW_HANDLER",
		"Got request to show L3 address")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	networkContext := context.NetworkContext
	networkPort := networkContext.Port(context.Port.Number)

	// Return interface network data.
	body := models.Network{
		Encapsulation: models.NullString(networkContext.Driver),
		Addr:          models.NullString(networkPort.Addr),
		InterfaceName: context.Port.Name,
		Interface:     context.Port.Number,
	}

	context.W.Write(rw, body, http.StatusOK)
}

func (h *NetworkHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("network_handlers/DESTROY_HANDLER",
		"Got request to destroy L3 address")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	networkContext := &mech.NetworkManagerContext{
		Datapath: context.Mech.Switch.ID(),
		Ports:    []mech.NetworkPort{{Port: context.Port.Number}},
	}

	if err = context.Network.DeleteNetwork(networkContext); err != nil {
		log.ErrorLog("network_handlers/DELETE_HANDLER",
			"Failed to delete network layer address: ", err)

		body := models.Error{"failed update network"}
		context.W.Write(rw, body, http.StatusConflict)
		return
	}

	context.W.Write(rw, nil, http.StatusOK)
}

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
	// Register link management HTTP API driver.
	constructor := mech.HTTPDriverConstructorFunc(NewLinkHandler)
	mech.RegisterHTTPDriver(constructor)
}

// L2 protocol address management
type LinkHandler struct {
	// Base HTTP driver instance.
	mech.BaseHTTPDriver
}

// NewLinkHandler creates a new instance of LinkHandler type.
func NewLinkHandler() mech.HTTPDriver {
	return &LinkHandler{}
}

// Enable implements HTTPDriver interface.
func (h *LinkHandler) Enable(c *mech.HTTPDriverContext) {
	h.BaseHTTPDriver.Enable(c)

	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/interfaces/links", h.indexHandler)
	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/interfaces/{interface}/link", h.showHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/interfaces/{interface}/link", h.createHandler)
	h.C.Mux.HandleFunc("DELETE", "/v1/datapaths/{dpid}/interfaces/{interface}/link", h.destroyHandler)

	log.InfoLog("link_handlers/ENABLE_HOOK",
		"Link layer handlers enabled")
}

func (h *LinkHandler) context(rw http.ResponseWriter, r *http.Request) (*mech.MechanismContext, *mech.SwitchPort, error) {
	log.InfoLog("link_handlers/CONTEXT",
		"Got request to handle L2 address")

	dpid := httputil.Param(r, "dpid")
	iface := httputil.Param(r, "interface")

	f := WriteFormat(r)

	log.DebugLogf("link_handlers/CONTEXT",
		"Request handle L2 address of: %s dev %s", dpid, iface)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("link_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		f.Write(rw, models.Error{text}, http.StatusNotFound)
		return nil, nil, fmt.Errorf(text)
	}

	port, err := context.Switch.PortByName(iface)
	if err != nil {
		log.ErrorLog("link_handlers/CONTEXT",
			"Failed to find requested interface: ", iface)

		text := fmt.Sprintf("switch '%s' does not have '%s' interface", dpid, iface)

		f.Write(rw, models.Error{text}, http.StatusNotFound)
		return nil, nil, fmt.Errorf(text)
	}

	return context, port, nil
}

func (h *LinkHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_handlers/INDEX_HANDLER",
		"Got request to list link layer addresses")

	dpid := httputil.Param(r, "dpid")
	wf := WriteFormat(r)

	log.DebugLogf("link_handlers/INDEX_HANDLER",
		"Request list links of: %s", dpid)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("link_handlers/INDEX_HANDLER",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		wf.Write(rw, models.Error{text}, http.StatusNotFound)
		return
	}

	var links []models.Link
	var link mech.LinkManagerContext

	if err := context.Link.Context(&link); err != nil {
		log.ErrorLog("link_handlers/INDEX_HANDLER",
			"Failed to retrieve link context: ", err)

		text := fmt.Sprintf("failed to access database")

		wf.Write(rw, models.Error{text}, http.StatusNotFound)
		return
	}

	for _, switchPort := range context.Switch.PortList() {
		linkPort := link.Port(switchPort.Number)

		links = append(links, models.Link{
			Encapsulation: models.NullString(link.Driver),
			Addr:          models.NullString(linkPort.Addr),
			InterfaceName: switchPort.Name,
			Interface:     switchPort.Number,
		})
	}

	wf.Write(rw, links, http.StatusOK)
}

func (h *LinkHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_handlers/CREATE_HANDLER",
		"Got request to create link layer address")

	rf, wf := Format(r)

	switchContext, switchPort, err := h.context(rw, r)
	if err != nil {
		return
	}

	var link models.Link
	if err = rf.Read(r, &link); err != nil {
		log.ErrorLog("link_handlers/CREATE_HANDLER",
			"Failed to read request body: ", err)

		body := models.Error{"failed to read request body"}
		wf.Write(rw, body, http.StatusBadRequest)
		return
	}

	context := &mech.LinkManagerContext{
		Datapath: switchContext.Switch.ID(),
		Driver:   link.Encapsulation.String(),
		Ports: []mech.LinkPort{
			{link.Addr.String(), switchPort.Number},
		},
	}

	if err = switchContext.Link.UpdateLink(context); err != nil {
		log.ErrorLog("link_handlers/CREATE_HANDLER",
			"Failed to createa a new L2 address: ", err)

		body := models.Error{"failed update link"}
		wf.Write(rw, body, http.StatusConflict)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (h *LinkHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_handlers/SHOW_HANDLER",
		"Got request to show L2 address")

	wf := WriteFormat(r)

	context, switchPort, err := h.context(rw, r)
	if err != nil {
		return
	}

	var link mech.LinkManagerContext
	context.Link.Context(&link)
	linkPort := link.Port(switchPort.Number)

	body := models.Link{
		Encapsulation: models.NullString(link.Driver),
		Addr:          models.NullString(linkPort.Addr),
		InterfaceName: switchPort.Name,
		Interface:     switchPort.Number,
	}

	// Return interface link data.
	wf.Write(rw, body, http.StatusOK)
}

func (h *LinkHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_handlers/DESTROY_HANDLER",
		"Got request to destroy L2 address")

	// driver.DeteleNework(mech.LinkManagerContext{})
}

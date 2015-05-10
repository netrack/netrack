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
	// Register link management HTTP API driver.
	constructor := mech.HTTPDriverConstructorFunc(NewLinkHandler)
	mech.RegisterHTTPDriver(constructor)
}

type LinkHandlerContext struct {
	// Back-end context.
	Mech *mech.MechanismContext

	// Link layer manager instance.
	Link mech.LinkMechanismManager

	// Link layer maanger context.
	LinkContext *mech.LinkManagerContext

	// SwitchPort instance
	Port *mech.SwitchPort

	// WriteFormatter, to write data in
	// requested format.
	W format.WriteFormatter

	// ReadFormatter equal to nil if request
	// does not contain body.
	R format.ReadFormatter
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

	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/link/interfaces", h.indexHandler)
	h.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/link/interfaces/{interface}", h.showHandler)
	h.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/link/interfaces/{interface}", h.createHandler)
	h.C.Mux.HandleFunc("DELETE", "/v1/datapaths/{dpid}/link/interfaces/{interface}", h.destroyHandler)

	log.InfoLog("link_handlers/ENABLE_HOOK",
		"Link layer handlers enabled")
}

func (h *LinkHandler) context(rw http.ResponseWriter, r *http.Request) (*LinkHandlerContext, error) {
	log.InfoLog("link_handlers/CONTEXT",
		"Got request to handle L2 address")

	dpid := httputil.Param(r, "dpid")
	iface := httputil.Param(r, "interface")

	rf, wf := Format(r)

	log.DebugLogf("link_handlers/CONTEXT",
		"Request handle L2 address of: %s dev %s", dpid, iface)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("link_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		wf.Write(rw, models.Error{text}, http.StatusNotFound)
		return nil, fmt.Errorf(text)
	}

	var port *mech.SwitchPort

	if iface != "" {
		port, err = context.Switch.PortByName(iface)
		if err != nil {
			log.ErrorLog("link_handlers/CONTEXT",
				"Failed to find requested interface: ", iface)

			text := fmt.Sprintf("switch '%s' does not have '%s' interface", dpid, iface)

			wf.Write(rw, models.Error{text}, http.StatusNotFound)
			return nil, fmt.Errorf(text)
		}
	}

	var llink mech.LinkMechanismManager
	if err := context.Managers.Obtain(&llink); err != nil {
		log.ErrorLog("link_handlers/LINK_LAYER_MANAGER",
			"Failed to obtain link layer manager: ", err)

		text := fmt.Sprintf("link layer manager is dead")
		wf.Write(rw, models.Error{text}, http.StatusInternalServerError)
		return nil, err
	}

	linkContext, err := llink.Context()
	if err != nil {
		log.ErrorLog("link_handlers/LINK_LAYER_CONTEXT",
			"Failed to get link layer context: ", err)

		text := fmt.Sprintf("link layer context inaccessible")
		wf.Write(rw, models.Error{text}, http.StatusConflict)
		return nil, err
	}

	ctx := &LinkHandlerContext{
		LinkContext: linkContext,
		Link:        llink,
		Mech:        context,
		Port:        port,
		W:           wf,
		R:           rf,
	}

	return ctx, nil
}

func (h *LinkHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_handlers/INDEX_HANDLER",
		"Got request to list link layer addresses")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	linkModels := make([]models.Link, 0)

	for _, switchPort := range context.Mech.Switch.PortList() {
		linkPort := context.LinkContext.Port(switchPort.Number)

		linkModels = append(linkModels, models.Link{
			Encapsulation: models.NullString(context.LinkContext.Driver),
			Addr:          models.NullString(linkPort.Addr),
			State:         models.NullString(switchPort.State),
			Config:        models.NullString(switchPort.Config),
			Features:      models.NullString(switchPort.Features),
			InterfaceName: switchPort.Name,
			Interface:     switchPort.Number,
		})
	}

	context.W.Write(rw, linkModels, http.StatusOK)
}

func (h *LinkHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_handlers/CREATE_HANDLER",
		"Got request to create link layer address")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	var linkModel models.Link

	if err = context.R.Read(r, &linkModel); err != nil {
		log.ErrorLog("link_handlers/CREATE_HANDLER",
			"Failed to read request body: ", err)

		body := models.Error{"failed to read request body"}
		context.W.Write(rw, body, http.StatusBadRequest)
		return
	}

	port := mech.LinkPort{
		Addr: linkModel.Addr.String(),
		Port: context.Port.Number,
	}

	linkContext := &mech.LinkManagerContext{
		Datapath: context.Mech.Switch.ID(),
		Driver:   linkModel.Encapsulation.String(),
		Ports:    []mech.LinkPort{port},
	}

	if err = context.Link.UpdateLink(linkContext); err != nil {
		log.ErrorLog("link_handlers/CREATE_HANDLER",
			"Failed to createa a new L2 address: ", err)

		body := models.Error{"failed update link"}
		context.W.Write(rw, body, http.StatusConflict)
		return
	}

	context.W.Write(rw, nil, http.StatusOK)
}

func (h *LinkHandler) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_handlers/SHOW_HANDLER",
		"Got request to show L2 address")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	linkContext := context.LinkContext
	linkPort := linkContext.Port(context.Port.Number)

	body := models.Link{
		Encapsulation: models.NullString(linkContext.Driver),
		Addr:          models.NullString(linkPort.Addr),
		State:         models.NullString(context.Port.State),
		Config:        models.NullString(context.Port.Config),
		Features:      models.NullString(context.Port.Features),
		InterfaceName: context.Port.Name,
		Interface:     context.Port.Number,
	}

	// Return interface link data.
	context.W.Write(rw, body, http.StatusOK)
}

func (h *LinkHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("link_handlers/DESTROY_HANDLER",
		"Got request to destroy L2 address")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	linkContext := &mech.LinkManagerContext{
		Datapath: context.Mech.Switch.ID(),
		Ports:    []mech.LinkPort{{Port: context.Port.Number}},
	}

	if err = context.Link.DeleteLink(linkContext); err != nil {
		log.ErrorLog("link_handlers/DELETE_HANDLER",
			"Failed to delete link layer address: ", err)

		body := models.Error{"failed update link"}
		context.W.Write(rw, body, http.StatusConflict)
		return
	}

	context.W.Write(rw, nil, http.StatusOK)
}

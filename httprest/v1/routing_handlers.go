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
	constructor := mech.HTTPDriverConstructorFunc(NewRoutingHandler)
	mech.RegisterHTTPDriver(constructor)
}

type RoutingHandlerContext struct {
	// Back-end context
	Mech *mech.MechanismContext

	// Routing mechanism manager
	Routing mech.RoutingMechanismManager

	// Routing context
	RoutingContext *mech.RoutingManagerContext

	// Write formatter
	W format.WriteFormatter

	// Read formatter
	R format.ReadFormatter
}

type RoutingHandler struct {
	mech.BaseHTTPDriver
}

func NewRoutingHandler() mech.HTTPDriver {
	return &RoutingHandler{}
}

func (m *RoutingHandler) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/routes", m.indexHandler)
	m.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/routes", m.createHandler)
	m.C.Mux.HandleFunc("DELETE", "/v1/datapaths/{dpid}/routes", m.destroyHandler)

	log.InfoLog("routing_handlers/ENABLE_HOOK",
		"Route handlers enabled")
}

func (h *RoutingHandler) context(rw http.ResponseWriter, r *http.Request) (*RoutingHandlerContext, error) {
	log.InfoLog("routing_handlers/CONTEXT",
		"Got request to handle routes")

	dpid := httputil.Param(r, "dpid")

	rf, wf := Format(r)

	log.DebugLog("routing_handlers/CONTEXT",
		"Request handle routes of: ", dpid)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("routing_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		wf.Write(rw, models.Error{text}, http.StatusNotFound)
		return nil, fmt.Errorf(text)
	}

	var routing mech.RoutingMechanismManager
	if err := context.Managers.Obtain(&routing); err != nil {
		log.ErrorLog("routing_handlers/ROUTING_MANAGER",
			"Failed to obtain routing layer manager: ", err)

		text := fmt.Sprintf("routing manager is dead")
		wf.Write(rw, models.Error{text}, http.StatusInternalServerError)
		return nil, err
	}

	routingContext, err := routing.Context()
	if err != nil {
		log.ErrorLog("routing_handlers/ROUTING_CONTEXT",
			"Failed to get routing context: ", err)

		text := fmt.Sprintf("routing context inaccessible")
		wf.Write(rw, models.Error{text}, http.StatusConflict)
		return nil, err
	}

	ctx := &RoutingHandlerContext{
		Mech:           context,
		Routing:        routing,
		RoutingContext: routingContext,
		W:              wf,
		R:              rf,
	}

	return ctx, nil
}

func (h *RoutingHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("routing_handlers/INDEX_HANDLER",
		"Got request to list routing table")

	context, err := h.context(rw, r)
	if err != nil {
		return
	}

	routingContext := context.RoutingContext
	routeModels := make([]models.Route, 0)

	for _, route := range routingContext.Routes {
		switchPort, _ := context.Mech.Switch.PortByNumber(route.Port)

		routeModels = append(routeModels, models.Route{
			Type:          route.Type,
			Network:       route.Network,
			NextHop:       route.NextHop,
			Interface:     switchPort.Number,
			InterfaceName: switchPort.Name,
		})
	}

	context.W.Write(rw, routeModels, http.StatusOK)
}

func (h *RoutingHandler) alter(rw http.ResponseWriter, r *http.Request) (*RoutingHandlerContext, error) {
	log.InfoLog("routing_handlers/ALTER_ROUTES",
		"Got request to alter routes")

	context, err := h.context(rw, r)
	if err != nil {
		return nil, err
	}

	var routeModels []models.Route
	if err = context.R.Read(r, &routeModels); err != nil {
		log.ErrorLog("routing_handlers/ALTER_ROUTES",
			"Failed to read request body: ", err)

		body := models.Error{"failed to read request body"}
		context.W.Write(rw, body, http.StatusBadRequest)
		return nil, err
	}

	routingContext := &mech.RoutingManagerContext{
		Datapath: context.Mech.Switch.ID(),
	}

	for _, route := range routeModels {
		switchPort, err := context.Mech.Switch.PortByName(route.InterfaceName)
		if err != nil {
			log.ErrorLog("routing_handlers/ALTER_ROUTES",
				"Failed to find requested interface: ", err)

			text := fmt.Sprintf("interface '%s' not found", route.InterfaceName)

			context.W.Write(rw, models.Error{text}, http.StatusConflict)
			return nil, err
		}

		routingContext.Routes = append(routingContext.Routes, &mech.Route{
			Type:    string(mech.StaticRoute),
			Network: route.Network,
			NextHop: route.NextHop,
			Port:    switchPort.Number,
		})
	}

	// Save new routing context
	context.RoutingContext = routingContext
	return context, nil
}

func (h *RoutingHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("routing_handlers/CREATE_HANDLER",
		"Got request to create routes")

	context, err := h.alter(rw, r)
	if err != nil {
		return
	}

	if err = context.Routing.UpdateRoutes(context.RoutingContext); err != nil {
		log.ErrorLog("routing_handlers/CREATE_HANDLER",
			"Failed to create routes: ", err)

		body := models.Error{"Failed update routing table"}
		context.W.Write(rw, body, http.StatusConflict)
		return
	}

	context.W.Write(rw, nil, http.StatusOK)
}

func (h *RoutingHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("routing_handlers/DESTROY_HANDLER",
		"Got request to destroy routes")

	context, err := h.alter(rw, r)
	if err != nil {
		return
	}

	if err = context.Routing.DeleteRoutes(context.RoutingContext); err != nil {
		log.ErrorLog("routing_handlers/DESTROY_HANDLER",
			"Failed to destroy routes: ", err)

		body := models.Error{"Failed update routing table"}
		context.W.Write(rw, body, http.StatusConflict)
		return
	}

	context.W.Write(rw, nil, http.StatusOK)
}

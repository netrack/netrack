package httprest

import (
	"fmt"
	"net/http"

	"github.com/netrack/netrack/httprest/v1/models"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/mechutil"
)

func init() {
	// Register address management HTTP API driver.
	constructor := mech.HTTPDriverConstructorFunc(NewRoutingHandler)
	mech.RegisterHTTPDriver(constructor)
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

func (h *RoutingHandler) context(rw http.ResponseWriter, r *http.Request) (*mech.MechanismContext, error) {
	log.InfoLog("routing_handlers/CONTEXT",
		"Got request to handle routes")

	dpid := httputil.Param(r, "dpid")

	f := WriteFormat(r)

	log.DebugLog("routing_handlers/CONTEXT",
		"Request handle routes of: ", dpid)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("routing_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		f.Write(rw, models.Error{text}, http.StatusNotFound)
		return nil, fmt.Errorf(text)
	}

	return context, nil
}

func (h *RoutingHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("routing_handlers/INDEX_HANDLER",
		"Got request to list routing table")

	wf := WriteFormat(r)

	switchContext, err := h.context(rw, r)
	if err != nil {
		return
	}

	routes := make([]models.Route, 0)
	var routeContext mech.RouteManagerContext
	switchContext.Routing.Context(&routeContext)

	for _, route := range routeContext.Routes {
		switchPort, _ := switchContext.Switch.PortByNumber(route.Port)

		routes = append(routes, models.Route{
			Type:          route.Type,
			Network:       route.Network,
			NextHop:       route.NextHop,
			Interface:     switchPort.Number,
			InterfaceName: switchPort.Name,
		})
	}

	wf.Write(rw, routes, http.StatusOK)
}

func (h *RoutingHandler) alter(rw http.ResponseWriter, r *http.Request) (*mech.MechanismContext, *mech.RouteManagerContext, error) {
	log.InfoLog("routing_handlers/ALTER_ROUTES",
		"Got request to alter routes")

	rf, wf := Format(r)

	switchContext, err := h.context(rw, r)
	if err != nil {
		return nil, nil, err
	}

	var routes []models.Route
	if err = rf.Read(r, &routes); err != nil {
		log.ErrorLog("routing_handlers/ALTER_ROUTES",
			"Failed to read request body: ", err)

		body := models.Error{"failed to read request body"}
		wf.Write(rw, body, http.StatusBadRequest)
		return nil, nil, err
	}

	context := &mech.RouteManagerContext{
		Datapath: switchContext.Switch.ID(),
	}

	for _, route := range routes {
		port, err := switchContext.Switch.PortByName(route.InterfaceName)
		if err != nil {
			log.ErrorLog("routing_handlers/ALTER_ROUTES",
				"Failed to find requested interface: ", err)

			text := fmt.Sprintf("interface '%s' not found", route.InterfaceName)

			wf.Write(rw, models.Error{text}, http.StatusConflict)
			return nil, nil, err
		}

		context.Routes = append(context.Routes, &mech.RouteContext{
			Type:    string(mechutil.StaticRoute),
			Network: route.Network,
			NextHop: route.NextHop,
			Port:    port.Number,
		})
	}

	return switchContext, context, nil
}

func (h *RoutingHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("routing_handlers/CREATE_HANDLER",
		"Got request to create routes")

	switchContext, context, err := h.alter(rw, r)
	if err != nil {
		return
	}

	wf := WriteFormat(r)

	if err = switchContext.Routing.UpdateRoutes(context); err != nil {
		log.ErrorLog("routing_handlers/CREATE_HANDLER",
			"Failed to create routes: ", err)

		body := models.Error{"Failed update routing table"}
		wf.Write(rw, body, http.StatusConflict)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (h *RoutingHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("routing_handlers/DESTROY_HANDLER",
		"Got request to destroy routes")

	switchContext, context, err := h.alter(rw, r)
	if err != nil {
		return
	}

	wf := WriteFormat(r)

	if err = switchContext.Routing.DeleteRoutes(context); err != nil {
		log.ErrorLog("routing_handlers/DESTROY_HANDLER",
			"Failed to destroy routes: ", err)

		body := models.Error{"Failed update routing table"}
		wf.Write(rw, body, http.StatusConflict)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

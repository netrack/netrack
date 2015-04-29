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
	constructor := mech.HTTPDriverConstructorFunc(NewRouteHandler)
	mech.RegisterHTTPDriver(constructor)
}

type RouteHandler struct {
	mech.BaseHTTPDriver
}

func NewRouteHandler() mech.HTTPDriver {
	return &RouteHandler{}
}

func (m *RouteHandler) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("GET", "/v1/datapaths/{dpid}/routes", m.indexHandler)
	m.C.Mux.HandleFunc("PUT", "/v1/datapaths/{dpid}/routes", m.createHandler)
	m.C.Mux.HandleFunc("DELETE", "/v1/datapaths/{dpid}/routes", m.destroyHandler)

	log.InfoLog("route_handlers/ENABLE_HOOK",
		"Route handlers enabled")
}

func (h *RouteHandler) context(rw http.ResponseWriter, r *http.Request) (*mech.MechanismContext, error) {
	log.InfoLog("route_handlers/CONTEXT",
		"Got request to handle routes")

	dpid := httputil.Param(r, "dpid")

	f := WriteFormat(r)

	log.DebugLog("route_handlers/CONTEXT",
		"Request handle routes of: ", dpid)

	context, err := h.C.SwitchManager.Context(dpid)
	if err != nil {
		log.ErrorLog("route_handlers/CONTEXT",
			"Failed to find requested datapath: ", err)

		text := fmt.Sprintf("switch '%s' not found", dpid)

		rw.WriteHeader(http.StatusNotFound)
		f.Write(rw, r, models.Error{text})
		return nil, fmt.Errorf(text)
	}

	return context, nil
}

func (h *RouteHandler) indexHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("route_handlers/INDEX_HANDLER",
		"Got request to list routing table")

	wf := WriteFormat(r)

	switchContext, err := h.context(rw, r)
	if err != nil {
		return
	}

	var routes []models.Route
	var routeContext mech.RouteManagerContext
	switchContext.Route.Context(&routeContext)

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

	rw.WriteHeader(http.StatusOK)
	wf.Write(rw, r, routes)
}

func (h *RouteHandler) createHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("route_handlers/CREATE_HANDLER",
		"Got request to create routes")

	rf, wf := Format(r)

	switchContext, err := h.context(rw, r)
	if err != nil {
		return
	}

	var routes []models.Route
	if err = rf.Read(rw, r, &routes); err != nil {
		log.ErrorLog("route_handlers/CREATE_HANDLER",
			"Failed to read request body: ", err)

		rw.WriteHeader(http.StatusBadRequest)
		wf.Write(rw, r, models.Error{"failed to read request body"})
		return
	}

	context := &mech.RouteManagerContext{
		Datapath: switchContext.Switch.ID(),
	}

	for _, route := range routes {
		port, err := switchContext.Switch.PortByName(route.InterfaceName)
		if err != nil {
			log.ErrorLog("route_handlers/CREATE_HANDLER",
				"Failed to find requested interface: ", err)

			text := fmt.Sprintf("interface '%s' not found", route.InterfaceName)

			rw.WriteHeader(http.StatusConflict)
			wf.Write(rw, r, models.Error{text})
			return
		}

		context.Routes = append(context.Routes, &mech.RouteContext{
			Network: route.Network,
			NextHop: route.NextHop,
			Port:    port.Number,
		})
	}

	if err = switchContext.Route.UpdateRoutes(context); err != nil {
		log.ErrorLog("route_handlers/CREATE_ROUTE",
			"Failed to create routes: ", err)

		rw.WriteHeader(http.StatusConflict)
		wf.Write(rw, r, models.Error{"Failed update routing table"})
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (h *RouteHandler) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("route_handlers/DESTROY_HANDLER",
		"Got request to destroy routes")
}

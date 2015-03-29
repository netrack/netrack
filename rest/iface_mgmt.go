package rest

import (
	"fmt"
	"net/http"

	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
)

type IFaceMgmt struct {
	C *mech.HTTPContext
}

func (m *IFaceMgmt) Initialize(c *mech.HTTPContext) {
	m.C = c

	m.C.Mux.HandleFunc("GET", "/dps/{dpid}/ifaces", m.indexHandler)
	m.C.Mux.HandleFunc("GET", "/dps/{dpid}/ifaces/{iface}", m.showHandler)
}

func (m *IFaceMgmt) indexHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *IFaceMgmt) showHandler(rw http.ResponseWriter, r *http.Request) {
	dpid := httputil.Param(r, "dpid")

	caller, err := m.C.R.Call(rpc.T_DATAPATH, dpid)
	if err != nil {
		fmt.Println(err)
	}

	ports, err := caller.(rpc.ProcCaller).Call(rpc.T_DATAPATH_PORTS, nil)
	fmt.Println(ports, err)
}

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
	var caller rpc.ProcCaller
	dpid := httputil.Param(r, "dpid")

	err := m.C.R.Call(rpc.T_DATAPATH, rpc.StringParam(dpid), rpc.ProcCallerResult(&caller))
	if err != nil {
		fmt.Println(err)
		return
	}

	var ports []string
	err = caller.Call(rpc.T_DATAPATH_PORT_NAMES, nil, rpc.StringSliceResult(&ports))
	fmt.Println(ports, err)
}

package rest

import (
	"fmt"
	"net/http"

	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
)

// IP protocol address management
type AddressMgmt struct {
	C *mech.HTTPContext
}

func (m *AddressMgmt) Initialize(c *mech.HTTPContext) {
	m.C = c

	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/ip/address", m.indexHandler)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/ip/address", m.showHandler)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/ip/address", m.deleteHandler)
}

func (m *AddressMgmt) indexHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *AddressMgmt) showHandler(rw http.ResponseWriter, r *http.Request) {
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

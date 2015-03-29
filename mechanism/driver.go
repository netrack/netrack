package mech

import (
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
)

type OFPContext struct {
	R    rpc.ProcCaller
	Conn of.OFPConn
	Mux  *of.ServeMux
}

type OFPDriver interface {
	Initialize(*OFPContext)
}

type HTTPContext struct {
	R   rpc.ProcCaller
	Mux *httputil.ServeMux
}

type HTTPDriver interface {
	Initialize(*HTTPContext)
}

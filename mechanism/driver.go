package mech

import (
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
)

type Context struct {
	R    rpc.RPCaller
	Conn of.OFPConn
	Mux  *of.ServeMux
}

type Driver interface {
	Initialize(*Context)
}

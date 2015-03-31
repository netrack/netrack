package mech

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
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

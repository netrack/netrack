package mech

import (
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type Switch struct {
	ID   string
	Conn of.OFPConn
	Drv  []Driver
	C    *Context
}

func (d *Switch) Boot() {
	d.C = &Context{rpc.New(), d.Conn, of.NewServeMux()}
	d.C.Mux.HandleFunc(of.T_HELLO, d.helloHandler)
	d.C.Mux.HandleFunc(of.T_ECHO_REQUEST, d.echoHandler)

	for _, drv := range d.Drv {
		drv.Initialize(d.C)
	}

	go d.serve()
}

func (d *Switch) serve() {
	for {
		r, err := d.Conn.Receive()
		if err != nil {
			return
		}

		rw := &of.Response{Conn: d.Conn}
		go d.C.Mux.Serve(rw, r)
	}
}

func (d *Switch) helloHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_HELLO)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.WriteHeader()
}

func (d *Switch) echoHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.WriteHeader()
}

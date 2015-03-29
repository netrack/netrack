package mech

import (
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
)

type Switch struct {
	Conn of.OFPConn
	Drv  []OFPDriver
	C    *OFPContext
}

func (d *Switch) Boot() {
	d.C = &OFPContext{rpc.New(), d.Conn, of.NewServeMux()}
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

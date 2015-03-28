package controller

import (
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
)

type C struct {
	Addr string
	Drv  []mech.Driver
}

func (c *C) ListenAndServe() {
	var conn of.OFPConn
	l, err := of.Listen("tcp", c.Addr)
	if err != nil {
		return
	}

	for {
		conn, err = l.AcceptOFP()
		if err != nil {
			return
		}

		device := mech.Switch{Conn: conn, Drv: c.Drv}
		go device.Boot()
	}
}

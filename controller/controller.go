package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
)

type C struct {
	Addr    string
	OFPDrv  []mech.OFPDriver
	HTTPDrv []mech.HTTPDriver

	devices []*mech.Switch
	httpc   *mech.HTTPContext
}

func (c *C) ListenAndServe() {
	c.serveHTTP()
	c.serveOFP()
}

func (c *C) serveOFP() {
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

		device := &mech.Switch{Conn: conn, Drv: c.OFPDrv}
		c.devices = append(c.devices, device)
		go device.Boot()
	}
}

func (c *C) serveHTTP() {
	mux := httputil.NewServeMux()
	c.httpc = &mech.HTTPContext{rpc.New(), mux}
	c.httpc.R.RegisterFunc(rpc.T_DATAPATH, c.datapath)

	for _, drv := range c.HTTPDrv {
		drv.Initialize(c.httpc)
	}

	s := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		log.Fatal(s.ListenAndServe())
	}()
}

func (c *C) datapath(param rpc.Param, result rpc.Result) error {
	var id, dpid string

	if err := param.Obtain(&id); err != nil {
		return nil
	}

	for _, device := range c.devices {
		err := device.C.R.Call(rpc.T_DATAPATH_ID, nil, rpc.StringResult(&dpid))
		if err != nil {
			return err
		}

		fmt.Println(dpid, id)
		if dpid == id {
			return result.Return(device.C.R)
		}
	}

	return errors.New("datapath not found")
}

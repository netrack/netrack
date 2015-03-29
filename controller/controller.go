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
	c.httpc.R.RegisterFunc(rpc.T_DATAPATH, c.dpCaller)

	for _, drv := range c.HTTPDrv {
		drv.Initialize(c.httpc)
	}

	s := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		log.Fatal(s.ListenAndServe())
	}()
}

func (c *C) dpCaller(param interface{}) (interface{}, error) {
	id, ok := param.(string)
	if !ok {
		return nil, errors.New("unexpected value for string")
	}

	return c.dp(id)
}

func (c *C) dp(id string) (rpc.ProcCaller, error) {
	for _, device := range c.devices {
		dpid, err := rpc.String(device.C.R.Call(rpc.T_DATAPATH_ID, nil))
		if err != nil {
			continue
		}

		fmt.Println(dpid, id)
		if dpid == id {
			return device.C.R, nil
		}
	}

	return nil, errors.New("dp not found")
}

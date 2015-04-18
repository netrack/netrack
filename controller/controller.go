package controller

import (
	"net/http"

	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
)

// C represents OpenFlow controller.
type C struct {
	Addr string

	// switchManager manages switch connections.
	switchManager mech.SwitchManager

	// httpManager manages HTTP drivers.
	httpManager mech.HTTPDriverManager
}

func (c *C) ListenAndServe() {
	c.initializeHTTPDrivers()
	c.initializeSwitches()
}

func (c *C) initializeSwitches() {
	l, err := of.Listen("tcp", c.Addr)
	if err != nil {
		log.ErrorLog("controller/LISTEN_AND_SERVE_OFP_ERR",
			"Failed to serve OFP: ", err)
		return
	}

	for {
		conn, err := l.AcceptOFP()
		if err != nil {
			log.ErrorLog("controller/ACCEPT_OFP_CONN_ERR",
				"Failed to accept OFP connection: ", err)
			return
		}

		go func() {
			if err := c.switchManager.CreateSwitch(conn); err != nil {
				log.ErrorLog("controller/CREATE_SWITCH_ERR",
					"Failed to create a new switch: ", err)
			}
		}()
	}
}

func (c *C) initializeHTTPDrivers() {
	context := &mech.HTTPDriverContext{
		Mux:           httputil.NewServeMux(),
		SwitchManager: &c.switchManager,
	}

	// activate registered HTTP drivers.
	c.httpManager.Enable(context)

	//FIXME: make address configurable
	s := &http.Server{Addr: ":8080", Handler: context.Mux}

	// Start serving.
	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.ErrorLog("controller/LISTEN_AND_SERVE_HTTP_ERR",
				"Failed to serve HTTP: ", err)
		}
	}()
}

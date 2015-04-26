package controller

import (
	"net/http"
	"net/url"

	"github.com/netrack/netrack/config"
	"github.com/netrack/netrack/database"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
)

// C represents OpenFlow controller.
type C struct {
	// Controller configuration.
	Config *config.Config

	// switchManager manages switch connections.
	switchManager mech.SwitchManager

	// httpManager manages HTTP drivers.
	httpManager mech.HTTPDriverManager
}

func (c *C) ListenAndServe() {
	c.initializeDatabase()
	c.initializeHTTPDrivers()
	c.initializeSwitches()
}

func (c *C) initializeDatabase() {
	persister, err := db.Open(c.Config.ConnString())
	if err != nil {
		log.FatalLog("controller/INTIALIZE_DATABASE",
			"Failed to open database connection: ", err)
	}

	db.DefaultDB = persister
}

func (c *C) initializeSwitches() {
	u, err := url.Parse(c.Config.OFPEndpoint)
	if err != nil {
		log.FatalLog("controller/PARSE_OFP_ADDRESS_ERR",
			"Failed to parse openflow_endpoint parameter: ", err)
	}

	log.DebugLogf("controller/INITIALIZE_SWITCHES",
		"Starting serving OFP at: %s://%s", u.Scheme, u.Host)

	l, err := of.Listen(u.Scheme, u.Host)
	if err != nil {
		log.FatalLog("controller/LISTEN_AND_SERVE_OFP_ERR",
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
	u, err := url.Parse(c.Config.APIEndpoint)
	if err != nil {
		log.FatalLog("constrollers/PARSE_HTTP_ADDRESS_ERR",
			"Failed to parse api_endpoint parameter: ", err)
	}

	context := &mech.HTTPDriverContext{
		Mux:           httputil.NewServeMux(),
		SwitchManager: &c.switchManager,
	}

	// Activate registered HTTP drivers.
	c.httpManager.Enable(context)

	// Create HTTP Server.
	s := &http.Server{Addr: u.Host, Handler: context.Mux}

	log.DebugLogf("controller/INTIALIZE_HTTP",
		"Starting service HTTP at: http://%s", u.Host)

	// Start serving.
	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.FatalLog("controller/LISTEN_AND_SERVE_HTTP_ERR",
				"Failed to serve HTTP: ", err)
		}
	}()
}

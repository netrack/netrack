package ofp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type OFPMech struct {
	C        *mech.MechanismDriverContext
	ports    []ofp.Port
	features ofp.SwitchFeatures
}

func (m *OFPMech) Initialize(c *mech.MechanismDriverContext) {
	m.C = c

	m.C.Mux.HandleFunc(of.T_HELLO, m.helloHandler)
	m.C.Mux.HandleFunc(of.T_ECHO_REQUEST, m.echoHandler)
	m.C.Mux.HandleFunc(of.T_FEATURES_REPLY, m.featuresHandler)
	m.C.Mux.HandleFunc(of.T_MULTIPART_REPLY, m.multipartHandler)

	log.InfoLog("ofp/INIT_DONE", "OFP mechanism successfully initialized")
}

func (m *OFPMech) helloHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_HELLO)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	if err := rw.WriteHeader(); err != nil {
		log.ErrorLog("ofp/HELLO_SEND_HELLO",
			"Failed to send ofp_hello message: ", err)
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_FEATURES_REQUEST)
	if err := rw.WriteHeader(); err != nil {
		log.ErrorLog("ofp/HELLO_SEND_FEATURES_REQUEST",
			"Failed to send ofp_features_request: ", err)
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_MULTIPART_REQUEST)
	desc := ofp.MultipartRequest{Type: ofp.MP_PORT_DESC}
	desc.WriteTo(rw)
	if err := rw.WriteHeader(); err != nil {
		log.ErrorLog("ofp/HELLO_SEND_MULTIPART_REQUEST",
			"Failed to send ofp_multipart_request: ", err)
	}
}

func (m *OFPMech) echoHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	if err := rw.WriteHeader(); err != nil {
		log.ErrorLog("ofp/ECHO_SEND_ECHO_REPLY",
			"Failed to send ofp_echo_reply: ", err)
	}
}

func (m *OFPMech) featuresHandler(rw of.ResponseWriter, r *of.Request) {
	if _, err := m.features.ReadFrom(r.Body); err != nil {
		log.ErrorLog("ofp/FEATURES_READ_ERR",
			"Failed to read ofp_features_reply: ", err)
	}
}

func (m *OFPMech) multipartHandler(rw of.ResponseWriter, r *of.Request) {
	var reply ofp.MultipartReply
	reply.ReadFrom(r.Body)

	var ports ofp.Ports
	if _, err := ports.ReadFrom(r.Body); err != nil {
		log.ErrorLog("ofp/MULTIPART_READ_ERR",
			"Failed to read next ofp_port value: ", err)
		return
	}

	for _, p := range ports {
		if p.PortNo == ofp.P_LOCAL {
			continue
		}

		m.ports = append(m.ports, p)
	}
}

package ofp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type OFPMech struct {
	C        *mech.OFPContext
	ports    []ofp.Port
	features ofp.SwitchFeatures
}

func (m *OFPMech) Initialize(c *mech.OFPContext) {
	m.C = c

	m.C.R.RegisterFunc(rpc.T_DATAPATH_PORTS, m.datapathPorts)
	m.C.R.RegisterFunc(rpc.T_DATAPATH_ID, m.datapathIdentifier)

	m.C.Mux.HandleFunc(of.T_HELLO, m.helloHandler)
	m.C.Mux.HandleFunc(of.T_ECHO_REQUEST, m.echoHandler)
	m.C.Mux.HandleFunc(of.T_FEATURES_REPLY, m.featuresHandler)
	m.C.Mux.HandleFunc(of.T_MULTIPART_REPLY, m.multipartHandler)

	log.InfoLog("ofp/INIT_DONE", "OFP mechanism successfully initialized")
}

func (m *OFPMech) helloHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_HELLO)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.WriteHeader()

	rw.Header().Set(of.TypeHeaderKey, of.T_FEATURES_REQUEST)
	rw.WriteHeader()

	rw.Header().Set(of.TypeHeaderKey, of.T_MULTIPART_REQUEST)
	desc := ofp.MultipartRequest{Type: ofp.MP_PORT_DESC}
	desc.WriteTo(rw)
	rw.WriteHeader()
}

func (m *OFPMech) echoHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.WriteHeader()
}

func (m *OFPMech) featuresHandler(rw of.ResponseWriter, r *of.Request) {
	m.features.ReadFrom(r.Body)
}

func (m *OFPMech) multipartHandler(rw of.ResponseWriter, r *of.Request) {
	var reply ofp.MultipartReply
	reply.ReadFrom(r.Body)

	var ports ofp.Ports
	_, err := ports.ReadFrom(r.Body)
	if err != nil {
		return
	}

	for _, p := range ports {
		if p.PortNo == ofp.P_LOCAL {
			continue
		}

		m.ports = append(m.ports, p)
	}
}

func (m *OFPMech) datapathPorts(param rpc.Param, result rpc.Result) error {
	var ports []string

	for _, p := range m.ports {
		ports = append(ports, string(p.Name))
	}

	return result.Return(ports)
}

func (m *OFPMech) datapathIdentifier(param rpc.Param, result rpc.Result) error {
	var b bytes.Buffer

	err := binary.Write(&b, binary.BigEndian, m.features.DatapathID)
	if err != nil {
		return err
	}

	id := fmt.Sprintf("%x", b.Bytes())
	var parts []string

	for i := 0; i < len(id); i += 2 {
		parts = append(parts, string(id[i:i+2]))
	}

	return result.Return(strings.Join(parts, "-"))
}

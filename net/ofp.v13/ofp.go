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
	C        *mech.OFPContext
	ports    []ofp.Port
	features ofp.SwitchFeatures
}

func (m *OFPMech) Initialize(c *mech.OFPContext) {
	m.C = c

	m.C.R.RegisterFunc(rpc.T_DATAPATH_PORT_NAMES, m.datapathPortNames)
	m.C.R.RegisterFunc(rpc.T_DATAPATH_PORT_HWADDR, m.datapathPortHWAddr)
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

func (m *OFPMech) datapathPortNames(param rpc.Param, result rpc.Result) error {
	var ports []string

	for _, p := range m.ports {
		ports = append(ports, string(p.Name))
	}

	return result.Return(ports)
}

func (m *OFPMech) datapathPortHWAddr(param rpc.Param, result rpc.Result) error {
	var portNo uint16

	if err := param.Obtain(&portNo); err != nil {
		log.ErrorLog("ofp/DATAPATH_PORT_HWADDR_ERR",
			"Failed to obtain port number from param: ", err)
		return err
	}

	for _, port := range m.ports {
		if uint16(port.PortNo) == portNo {
			log.DebugLog("ofp/DATAPATH_PORT_HWADDR",
				"Found port hardware address: ", port.HWAddr)
			return result.Return([]byte(port.HWAddr))
		}
	}

	log.DebugLog("ofp/DATAPATH_PORT_HWADDR",
		"Requested port not found: ", portNo)
	return errors.New("ofp: port not found")
}

func (m *OFPMech) datapathIdentifier(param rpc.Param, result rpc.Result) error {
	var b bytes.Buffer

	err := binary.Write(&b, binary.BigEndian, m.features.DatapathID)
	if err != nil {
		log.ErrorLog("ofp/DATAPATH_ID_WRITE_ERR",
			"Failed serialize datapath identifier: ", err)
		return err
	}

	id := fmt.Sprintf("%x", b.Bytes())
	var parts []string

	for i := 0; i < len(id); i += 2 {
		parts = append(parts, string(id[i:i+2]))
	}

	return result.Return(strings.Join(parts, "-"))
}

package ofp

import (
	"fmt"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type OFPMech struct {
	C *mech.Context
}

func (m *OFPMech) Initialize(c *mech.Context) {
	m.C = c

	m.C.Mux.HandleFunc(of.T_HELLO, m.helloHandler)
	m.C.Mux.HandleFunc(of.T_ECHO_REQUEST, m.echoHandler)
	m.C.Mux.HandleFunc(of.T_MULTIPART_REPLY, m.multipartHandler)
}

func (m *OFPMech) helloHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_HELLO)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.WriteHeader()

	rw.Header().Set(of.TypeHeaderKey, of.T_MULTIPART_REQUEST)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)

	desc := ofp.MultipartRequest{Type: ofp.MP_PORT_DESC}
	desc.WriteTo(rw)
	rw.WriteHeader()
}

func (m *OFPMech) echoHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.WriteHeader()
}

func (m *OFPMech) multipartHandler(rw of.ResponseWriter, r *of.Request) {
	var reply ofp.MultipartReply
	reply.ReadFrom(r.Body)

	var ports ofp.Ports
	ports.ReadFrom(r.Body)

	for _, p := range ports {
		fmt.Println(string(p.Name), p.HWAddr)
	}
}

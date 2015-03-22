package ip

import (
	"bytes"
	"net"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/net/l3"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type NeighTable struct {
	//
}

type ARPServer struct {
	HWAddr net.HardwareAddr
	IPAddr net.IP
}

func (s *ARPServer) Hello(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_FLOW_MOD)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)

	// Catch all ARP requests
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_ARP), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ARP_OP, of.Bytes(l3.ARPOT_REQUEST), nil},
	}}

	// Move all such packets to controller
	instr := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS,
		ofp.Actions{ofp.ActionOutput{ofp.P_CONTROLLER, ofp.CML_NO_BUFFER}},
	}}

	fmod := &ofp.FlowMod{
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Match:        match,
		Instructions: instr,
	}

	fmod.WriteTo(rw)
	rw.WriteHeader()
}

func (s *ARPServer) PacketIn(rw of.ResponseWriter, r *of.Request) {
	var p ofp.PacketIn
	p.ReadFrom(r.Body)

	var eth l2.EthernetII
	if eth.ReadFrom(r.Body); eth.EthType != iana.ETHT_ARP {
		return
	}

	var arp l3.ARP
	if arp.ReadFrom(r.Body); arp.Operation != l3.ARPOT_REQUEST {
		return
	}

	if !bytes.Equal(arp.ProtoDst, s.IPAddr) {
		return
	}

	eth = l2.EthernetII{eth.HWSrc, s.HWAddr, iana.ETHT_ARP}
	arp = l3.ARP{l3.ARPT_ETHERNET, iana.ETHT_IPV4, l3.ARPOT_REPLY,
		s.HWAddr,
		s.IPAddr,
		arp.HWSrc,
		arp.ProtoSrc,
	}

	pout := ofp.PacketOut{BufferID: ofp.NO_BUFFER,
		InPort:  p.Match.Field(ofp.XMT_OFB_IN_PORT).Value.PortNo(),
		Actions: ofp.Actions{ofp.ActionOutput{ofp.P_IN_PORT, 0}},
	}

	pout.WriteTo(rw)
	eth.WriteTo(rw)
	arp.WriteTo(rw)

	rw.Header().Set(of.TypeHeaderKey, of.T_PACKET_OUT)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.WriteHeader()
}

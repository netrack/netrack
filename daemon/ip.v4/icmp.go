package ip

import (
	"net"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/net/l3"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type ICMPServer struct {
	HWAddr net.HardwareAddr
	IPAddr net.IP
}

func (s *ICMPServer) Hello(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_FLOW_MOD)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)

	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, of.Bytes(s.IPAddr), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IP_PROTO, of.Bytes(iana.IP_PROTO_ICMP), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ICMPV4_TYPE, of.Bytes(l3.ICMPT_ECHO_REQUEST), nil},
	}}

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

func (s *ICMPServer) PacketIn(rw of.ResponseWriter, r *of.Request) {
	var p ofp.PacketIn
	p.ReadFrom(r.Body)

	var eth l2.EthernetII
	if eth.ReadFrom(r.Body); eth.EthType != iana.ETHT_IPV4 {
		return
	}

	var ip l3.IPv4
	if ip.ReadFrom(r.Body); ip.Proto != iana.IP_PROTO_ICMP {
		return
	}

	icmp := l3.ICMPEcho{Data: make([]byte, ip.Len-l3.IPv4HeaderLen-l3.ICMPHeaderLen)}
	if icmp.ReadFrom(r.Body); icmp.Type != l3.ICMPT_ECHO_REQUEST {
		return
	}

	icmp.Type = l3.ICMPT_ECHO_REPLY
	payload, _ := of.NewReader(&icmp)

	eth = l2.EthernetII{eth.HWSrc, s.HWAddr, iana.ETHT_IPV4}
	ip = l3.IPv4{Src: s.IPAddr, Dst: ip.Src, Proto: iana.IP_PROTO_ICMP, Payload: payload}

	pout := ofp.PacketOut{BufferID: ofp.NO_BUFFER,
		InPort:  p.Match.Field(ofp.XMT_OFB_IN_PORT).Value.PortNo(),
		Actions: ofp.Actions{ofp.ActionOutput{ofp.P_IN_PORT, 0}},
	}

	pout.WriteTo(rw)
	eth.WriteTo(rw)
	ip.WriteTo(rw)

	rw.Header().Set(of.TypeHeaderKey, of.T_PACKET_OUT)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.WriteHeader()
}

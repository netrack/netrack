package ip

import (
	"net"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/net/l3"
	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	//"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type NeighTable struct {
	//
}

type ARPMech struct {
	C      *mech.OFPContext
	HWAddr net.HardwareAddr
	IPAddr net.IP
}

func (m *ARPMech) Initialize(c *mech.OFPContext) {
	m.C = c

	//TODO: HWAddr from datapath
	m.HWAddr = net.HardwareAddr{0, 0, 0, 0, 0, 254}

	//m.C.R.RegisterFunc(rpc.T_ARP_RESOLVE, m.resolveCaller)

	m.C.Mux.HandleFunc(of.T_HELLO, m.Hello)
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.PacketIn)

	log.InfoLog("arp/INIT_DONE", "ARP mechanism successfully initialized")
}

func (m *ARPMech) Hello(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_FLOW_MOD)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)

	// Catch all ARP requests
	// TODO: make them prettier
	// ofp.MatchEtherType(iana.ETHT_ARP),
	// ofp.MatchARPOperation(l3.ARPOT_REQUEST),
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

func (m *ARPMech) PacketIn(rw of.ResponseWriter, r *of.Request) {
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

	//m.Handle(netutil.ARPHandler(m.arpHook))
	//m.Handle(netutil.CompositeHandler(
	//netutil.IPv4Handler(nil),
	//netutil.ICMPHandler(nil),
	//))

	//m.Serve(r.Body)

	//if !bytem.Equal(arp.ProtoDst, m.IPAddr) {
	//return
	//}

	eth = l2.EthernetII{eth.HWSrc, m.HWAddr, iana.ETHT_ARP}
	arp = l3.ARP{l3.ARPT_ETHERNET, iana.ETHT_IPV4, l3.ARPOT_REPLY,
		m.HWAddr,
		arp.ProtoDst,
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

//func (m *ARPMech) resolveCaller(param interface{}) (interface{}, error) {
//ipaddr, err := rpc.IPAddr(param, nil)
//if err != nil {
//return nil, err
//}

//return m.Resolve(ipaddr)
//}

//func (m *ARPMech) Resolve(net.IP) (net.HardwareAddr, error) {
////r := of.NewRequest(of.T_PACKET_OUT, &ofp.PacketOut{
////BufferID: ofp.NO_BUFFER,
////InPort:   ofp.P_FLOOD,
////Actions:  ofp.Actions{},
////})

////m.C.Conn.Send(r)
////m.C.Conn.Flush()

//return nil, nil
//}

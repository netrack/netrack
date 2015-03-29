package ip

import (
	"net"
	"sync"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/net/l3"
	"github.com/netrack/net/netutil"
	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type NeighEntry struct {
	Addr  net.IP
	Iface ofp.PortNo
	//Timestamp
}

type NeighTable struct {
	neighs []NeighEntry
	lock   sync.RWMutex
}

type ARPMech struct {
	C      *mech.OFPContext
	IPAddr net.IP
	T      NeighTable
}

func (m *ARPMech) Initialize(c *mech.OFPContext) {
	m.C = c

	m.C.R.RegisterFunc(rpc.T_ARP_RESOLVE, m.resolveIPAddr)

	m.C.Mux.HandleFunc(of.T_HELLO, m.Hello)
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.PacketIn)

	log.InfoLog("arp/INIT_DONE", "ARP mechanism successfully initialized")
}

func (m *ARPMech) Hello(rw of.ResponseWriter, r *of.Request) {
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

	var srcHWAddr []byte
	portNo := p.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()

	err := m.C.R.Call(rpc.T_DATAPATH_PORT_HWADDR,
		rpc.UInt16Param(uint16(portNo)),
		rpc.ByteSliceResult(&srcHWAddr))

	if err != nil {
		log.ErrorLog("arp/ARP_REQUEST_PORT_HWADDR_ERR",
			"Failed to find port hardware address: ", err)
		return
	}

	eth = l2.EthernetII{eth.HWSrc, net.HardwareAddr(srcHWAddr), iana.ETHT_ARP}
	arp = l3.ARP{l3.ARPT_ETHERNET, iana.ETHT_IPV4, l3.ARPOT_REPLY,
		net.HardwareAddr(srcHWAddr),
		arp.ProtoDst,
		arp.HWSrc,
		arp.ProtoSrc,
	}

	pout := ofp.PacketOut{BufferID: ofp.NO_BUFFER,
		InPort:  p.Match.Field(ofp.XMT_OFB_IN_PORT).Value.PortNo(),
		Actions: ofp.Actions{ofp.ActionOutput{ofp.P_IN_PORT, 0}},
	}

	_, err = netutil.WriteAllTo(rw, &pout, &eth, &arp)
	if err != nil {
		log.ErrorLog("arp/ARP_REQUEST_WRITE_ERR",
			"Failed to write ARP response: ", err)
		return
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_PACKET_OUT)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	if err := rw.WriteHeader(); err != nil {
		log.ErrorLog("arp/ARP_REQUEST_SEND_ERR",
			"Failed to send ARP response: ", err)
	}
}

func (m *ARPMech) resolveIPAddr(param rpc.Param, result rpc.Result) error {
	//r := of.NewRequest(of.T_PACKET_OUT, &ofp.PacketOut{
	//BufferID: ofp.NO_BUFFER,
	//InPort:   ofp.P_FLOOD,
	//Actions:  ofp.Actions{},
	//})

	//m.C.Conn.Send(r)
	//m.C.Conn.Flush()

	return nil
}

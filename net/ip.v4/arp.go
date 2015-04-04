package ip

import (
	"net"
	"sync"
	"time"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/net/l3"
	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func init() {
	constructor := mech.MechanismDriverCostructorFunc(NewARPMechanism)
	mech.RegisterMechanismDriver("arp-mechanism", constructor)
}

const TableARP ofp.Table = 2

type NeighEntry struct {
	IPAddr []byte
	HWAddr []byte
	Port   ofp.PortNo
	Time   time.Time
}

type NeighTable struct {
	neighs []NeighEntry
	lock   sync.RWMutex
}

func (t *NeighTable) Lookup(ipaddr []byte) ([]byte, error) {
	return nil, nil
}

type ARPMechanism struct {
	mech.BaseMechanismDriver
	T NeighTable
}

func NewARPMechanism() mech.MechanismDriver {
	return &ARPMechanism{}
}

// Enable implements MechanismDriver interface
func (m *ARPMechanism) Enable(c *mech.MechanismDriverContext) {
	m.BaseMechanismDriver.Enable(c)

	m.C.Func.RegisterFunc(rpc.T_ARP_RESOLVE, m.resolveCaller)

	m.C.Mux.HandleFunc(of.T_HELLO, m.helloHandler)
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.packetHandler)

	log.InfoLog("arp/INIT_DONE", "ARP mechanism successfully initialized")
}

// Activate implements MechanismDriver interface
func (m *ARPMechanism) Activate() {
	m.BaseMechanismDriver.Activate()

	var xid uint32
	err := m.C.Func.Call(rpc.T_OFP_TRANSACTION, nil,
		rpc.UInt32Result(&xid))

	if err != nil {
		log.ErrorLog("arp/HELLO_HANDLER",
			"Failed to retrieve a new transaction identifier: ", err)
		return
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_FLOW_MOD)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	rw.Header().Set(of.XIDHeaderKey, xid)

	//var hwDstAddr []byte
	//err = m.BaseMechanismDriver.C.Func.Call(rpc.T_OFP_PORT_HWADDR,
	//rpc.

	for _, hwaddr := range append([][]byte{l2.HWBcast}, hwDstAddr) {
		// Catch all ARP requests
		match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
			ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_ARP), nil},
			ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_DST, hwaddr, nil},
			ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ARP_OP, of.Bytes(l3.ARPOT_REQUEST), nil},
		}}

		// Move data to controller
		instr := ofp.Instructions{ofp.InstructionActions{
			ofp.ActionOutput{ofp.P_CONTROLLER, ofp.CML_NO_BUFFER},
		}}

		flowmod := &ofp.FlowMod{
			Table:        TableARP,
			Command:      ofp.FC_ADD,
			BufferID:     ofp.NO_BUFFER,
			Match:        match,
			Instructions: instr,
		}

		if _, err = fmod.WriteTo(rw); err != nil {
			log.ErrorLog("arp/HELLO_HANDLER",
				"Failed to write ofp_flow_mod message: ", err)
			return
		}

	}

	if err = rw.WriteHeader(); err != nil {
		log.ErrorLog("arp/HELLO_HANDLER",
			"Failed to write flow modifications: ", err)
	}
}

func (m *ARPMechanism) Disable() {
	m.BaseMechanismDriver.Disable()
}

func (m *ARPMechanism) packetHandler(rw of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn
	if _, err := packet.ReadFrom(r.Body); err != nil {
		log.DebugLog("arp/ARP_PACKET_HANDLER",
			"Failed to read ofp_packet_in message: ", err)
		return
	}

	if packet.TableID != TableARP {
		log.DebugLog("arp/PACKET_HANDLER",
			"Received packet from wrong table")
		return
	}

	var pdu2 l2.EthernetII
	var pdu3 l3.ARP

	if _, err = of.ReadAllFrom(r.Body, &pdu2, &pdu3); err != nil {
		log.ErrorLog("arp/ARP_PACKET_HANDLER",
			"Failed to read packet: ", err)
		return
	}

	if _, err = eth.ReadFrom(r.Body); err != nil {
		log.ErrorLog("arp/ARP_PACKET_HANDLER")
		return
	}

	var arp l3.ARP
	if arp.ReadFrom(r.Body); arp.Operation != l3.ARPOT_REQUEST {
		return
	}

	var srcHWAddr []byte
	portNo := p.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()

	err := m.C.Func.Call(rpc.T_OFP_PORT_HWADDR,
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

	_, err = of.WriteAllTo(rw, &pout, &eth, &arp)
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

func (m *ARPMechanism) resolveCaller(param rpc.Param, result rpc.Result) error {
	var srcIPAddr, dstIPAddr []byte
	var portNo uint16

	if err := param.Obtain(&srcIPAddr, &dstIPAddr, &portNo); err != nil {
		log.ErrorLog("arp/ARP_RESOLVE",
			"Failed to obtain argument: ", err)
		return err
	}

	// Check if entry already in a neigh table
	dstHWAddr, err := m.T.Lookup(dstIPAddr)
	if err == nil {
		return result.Return(dstHWAddr)
	}

	var srcHWAddr []byte
	err = m.C.Func.Call(rpc.T_OFP_PORT_HWADDR,
		rpc.UInt16Param(),
		rpc.ByteSliceResult(&srcHWaddr))

	if err != nil {
		log.ErrorLog("arp/ARP_RESOLVE",
			"Failed to fetch port hardware address: ", err)
		return err
	}

	// Start long process of discovery
	eth := l2.EthernetII{
		net.HardwareAddr(l2.HWBcast),
		net.HardwareAddr(net.srcHWAddr),
		iana.ETHT_ARP,
	}

	arp := l3.ARP{
		HWType:    l2.ARPT_ETHERNET,
		ProtoType: iana.ETHT_IPV4,
		Operation: l2.ARPOT_REQUEST,
		HWSrc:     net.HardwareAddr(srcHWAddr),
		ProtoSrc:  net.IP(srcIPAddr),
		HWDst:     net.HardwareAddr(l2.HWUnspec),
		ProtoDst:  net.IP(dstIPAddr),
	}

	packet := ofp.PacketOut{
		BufferID: ofp.NO_BUFFER,
		InPort:   ofp.P_FLOOD,
		Actions:  ofp.Actions{},
	}

	r, err := of.NewRequest(of.T_PACKET_OUT, of.NewReader(&packet, &eth, &arp))
	if err != nil {
		log.ErrorLog("arp/ARP_RESOLVE",
			"Failed to craete a new ARP request: ", err)
		return err
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("arp/ARP_RESOLVE",
			"Failed to send an ARP request: ", err)
		return err
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("arp/ARP_RESOLVE",
			"Failed to flush data to connection: ", err)
		return err
	}

	return nil
}

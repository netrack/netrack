package ip

import (
	//"bytes"
	"net"
	"sync"
	"time"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/net/l3"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewARPMechanism)
	mech.RegisterNetworkMechanism("ARP~RFC826", constructor)
}

type NeighEntry struct {
	NetworkAddr mech.NetworkAddr
	LinkAddr    mech.LinkAddr
	Port        uint32
	Time        time.Time
}

type NeighTable struct {
	neighs map[string][]NeighEntry
	lock   sync.RWMutex
}

func NewNeighTable() *NeighTable {
	neighs := make(map[string][]NeighEntry)
	return &NeighTable{neighs: neighs}
}

func (t *NeighTable) Populate(entry NeighEntry) error {
	t.lock.RLock()

	entries, ok := t.neighs[entry.NetworkAddr.String()]
	if !ok {
		defer t.lock.RUnlock()

		entries = append(entries, entry)
		t.neighs[entry.NetworkAddr.String()] = entries

		return nil
	}

	// Start search of matching entry
	//for _, e := range entries {
	//	netwEq := bytes.Equal(e.NetworkAddr.Bytes(), entry.NetworkAddr.Bytes())
	//	linkEq := bytes.Equal(e.LinkAddr.Bytes(), entry.LinkAddr.Bytes())
	//	portEq := e.Port == entry.Port

	//	//
	//}

	return nil
}

// ARPMechanism handles ARP requests to the networks,
// associated with switch ports.
type ARPMechanism struct {
	mech.BaseNetworkMechanism

	T *NeighTable

	// Table number allocated for the mechanism.
	tableNo int
}

func NewARPMechanism() mech.NetworkMechanism {
	return &ARPMechanism{T: NewNeighTable()}
}

// Enable implements Mechanism interface
func (m *ARPMechanism) Enable(c *mech.MechanismContext) {
	m.BaseMechanism.Enable(c)

	// Handle incoming ARP requests.
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.packetInHandler)

	log.InfoLog("arp/ENABLE_HOOK", "Mechanism ARP enabled")
}

// Activate implements Mechanism interface
func (m *ARPMechanism) Activate() {
	m.BaseMechanism.Activate()

	// Allocate table for handling arp protocol.
	tableNo, err := m.C.Switch.AllocateTable()
	if err != nil {
		log.ErrorLog("arp/ACTIVATE_HOOK",
			"Failed to allocate a new table: ", err)
		return
	}

	m.tableNo = tableNo

	log.DebugLog("arp/ACTIVATE_HOOK",
		"Allocated table: ", tableNo)

	// Match packets of ARP protocol.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_ARP), nil},
	}}

	// Move all packets to allocated matching table for ARP packets.
	instructions := ofp.Instructions{ofp.InstructionGotoTable{ofp.Table(m.tableNo)}}

	// Insert flow into 0 table.
	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Priority:     20,
		Match:        match,
		Instructions: instructions,
	}))

	if err != nil {
		log.ErrorLog("arp/ACTIVATE_HOOK",
			"Failed to create ofp_flow_mod request: ", err)
		return
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("arp/ACTIVATE_HOOK",
			"Failed to send request: ", err)
		return
	}

	// Create black-hole rule.
	r, err = of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		TableID:  ofp.Table(m.tableNo),
		Command:  ofp.FC_ADD,
		BufferID: ofp.NO_BUFFER,
		Match:    ofp.Match{ofp.MT_OXM, nil},
	}))

	if err != nil {
		log.ErrorLog("arp/ACTIVATE_HOOK",
			"Failed to create ofp_flow_mod request: ", err)
		return
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("arp/ACTIVATE_HOOK",
			"Failed to send request: ", err)
		return
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("arp/ACTIVATE_HOOK",
			"Failed to flush requests: ", err)
	}
}

func (m *ARPMechanism) Disable() {
	m.BaseMechanism.Disable()
}

func (m *ARPMechanism) UpdateNetwork(c *mech.NetworkContext) error {
	// Match broadcast ARP requests to resolve updated address.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_ARP), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ARP_OP, of.Bytes(l3.ARPOT_REQUEST), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ARP_THA, l2.HWUnspec, nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ARP_TPA, c.Addr.Bytes(), c.Addr.Mask()},
	}}

	// Send all such packets to controller
	// TODO: figure out if openflow allows flip packet fields.
	actions := ofp.Actions{
		ofp.ActionOutput{ofp.PortNo(ofp.P_CONTROLLER), ofp.CML_NO_BUFFER},
	}

	// Apply actions to packet
	instructions := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS, actions,
	}}

	// Insert flow into ARP-allocated flow table.
	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:      ofp.FC_ADD,
		TableID:      ofp.Table(m.tableNo),
		BufferID:     ofp.NO_BUFFER,
		Priority:     2, // Use non-zero priority
		Match:        match,
		Instructions: instructions,
	}))

	if err != nil {
		log.ErrorLog("arp/UPDATE_NETWORK",
			"Failed to create ofp_flow_mod request: ", err)
		return err
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("arp/UPDATE_NETWORK",
			"Failed to send request: ", err)
		return err
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("arp/UPDATE_NETWORK",
			"Failed to flush requests: ", err)
	}

	return err
}

func (m *ARPMechanism) packetInHandler(rw of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn
	var pdu2 mech.LinkFrame
	var pdu3 l3.ARP

	var err error

	lldriver, err := m.C.Link.Driver()
	if err != nil {
		log.InfoLog("arp/PACKET_IN_HANDLER",
			"Link layer driver is not intialized: ", err)
		return
	}

	// Assume, that all packets are ARP protocol messages.
	reader := mech.MakeLinkReaderFrom(lldriver, &pdu2)
	if _, err = of.ReadAllFrom(r.Body, &packet, reader, &pdu3); err != nil {
		log.ErrorLog("arp/PACKET_IN_HANDLER",
			"Failed to read packet: ", err)
		return
	}

	if int(packet.TableID) != m.tableNo {
		return
	}

	log.DebugLog("arp/PACKET_IN_HANDLER",
		"Got ARP request to resolve: ", pdu3.ProtoDst)

	// Use that port as egress to send response.
	portNo := packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()

	// Get link layer address associated with egress port.
	lladdr, err := lldriver.Addr(portNo)
	if err != nil {
		log.ErrorLogf("arp/PACKET_IN_HANDLER",
			"Failed to resolve port '%s' hardware address: '%s'", portNo, err)
		return
	}

	// Build link layer PDU.
	pdu2 = mech.LinkFrame{pdu2.SrcAddr, lladdr, mech.Proto(iana.ETHT_ARP), 0}

	// Build ARP response message.
	pdu3 = l3.ARP{l3.ARPT_ETHERNET, iana.ETHT_IPV4, l3.ARPOT_REPLY,
		net.HardwareAddr(lladdr.Bytes()),
		pdu3.ProtoDst,
		pdu3.HWSrc,
		pdu3.ProtoSrc,
	}

	packetOut := ofp.PacketOut{BufferID: ofp.NO_BUFFER,
		InPort:  packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.PortNo(),
		Actions: ofp.Actions{ofp.ActionOutput{ofp.P_IN_PORT, 0}},
	}

	llwriter := mech.MakeLinkWriterTo(lldriver, &pdu2)
	if _, err = of.WriteAllTo(rw, &packetOut, llwriter, &pdu3); err != nil {
		log.ErrorLog("arp/ARP_REQUEST_WRITE_ERR",
			"Failed to write ARP response: ", err)

		return
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_PACKET_OUT)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	if err = rw.WriteHeader(); err != nil {
		log.ErrorLog("arp/ARP_REQUEST_SEND_ERR",
			"Failed to send ARP response: ", err)
	}
}

func (m *ARPMechanism) resolveCaller(param rpc.Param, result rpc.Result) error {
	//var srcIPAddr, dstIPAddr []byte
	//var portNo uint16

	//if err := param.Obtain(&srcIPAddr, &dstIPAddr, &portNo); err != nil {
	//log.ErrorLog("arp/ARP_RESOLVE",
	//"Failed to obtain argument: ", err)
	//return err
	//}

	//// Check if entry already in a neigh table
	//dstHWAddr, err := m.T.Lookup(dstIPAddr)
	//if err == nil {
	//return result.Return(dstHWAddr)
	//}

	//var srcHWAddr []byte
	//err = m.C.Func.Call(rpc.T_OFP_PORT_HWADDR,
	//rpc.UInt16Param(),
	//rpc.ByteSliceResult(&srcHWaddr))

	//if err != nil {
	//log.ErrorLog("arp/ARP_RESOLVE",
	//"Failed to fetch port hardware address: ", err)
	//return err
	//}

	//// Start long process of discovery
	//eth := l2.EthernetII{
	//net.HardwareAddr(l2.HWBcast),
	//net.HardwareAddr(net.srcHWAddr),
	//iana.ETHT_ARP,
	//}

	//arp := l3.ARP{
	//HWType:    l2.ARPT_ETHERNET,
	//ProtoType: iana.ETHT_IPV4,
	//Operation: l2.ARPOT_REQUEST,
	//HWSrc:     net.HardwareAddr(srcHWAddr),
	//ProtoSrc:  net.IP(srcIPAddr),
	//HWDst:     net.HardwareAddr(l2.HWUnspec),
	//ProtoDst:  net.IP(dstIPAddr),
	//}

	//packet := ofp.PacketOut{
	//BufferID: ofp.NO_BUFFER,
	//InPort:   ofp.P_FLOOD,
	//Actions:  ofp.Actions{},
	//}

	//r, err := of.NewRequest(of.T_PACKET_OUT, of.NewReader(&packet, &eth, &arp))
	//if err != nil {
	//log.ErrorLog("arp/ARP_RESOLVE",
	//"Failed to craete a new ARP request: ", err)
	//return err
	//}

	//if err = m.C.Switch.Conn().Send(r); err != nil {
	//log.ErrorLog("arp/ARP_RESOLVE",
	//"Failed to send an ARP request: ", err)
	//return err
	//}

	//if err = m.C.Switch.Conn().Flush(); err != nil {
	//log.ErrorLog("arp/ARP_RESOLVE",
	//"Failed to flush data to connection: ", err)
	//return err
	//}

	//return nil
	return nil
}

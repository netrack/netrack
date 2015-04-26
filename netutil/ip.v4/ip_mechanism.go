package ip

import (
	"github.com/netrack/net/iana"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

const IPv4MechanismName = "IPv4#RFC791"

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewIPMechanism)
	mech.RegisterNetworkMechanism(IPv4MechanismName, constructor)
}

type IPMechanism struct {
	mech.BaseNetworkMechanism

	// IPv4 routing table instance.
	T RoutingTable

	// Table number allocated for the mechanism.
	tableNo int
}

func NewIPMechanism() mech.NetworkMechanism {
	return &IPMechanism{}
}

// Enable implements Mechanism interface
func (m *IPMechanism) Enable(c *mech.MechanismContext) {
	m.BaseNetworkMechanism.Enable(c)

	// Handle incoming IPv4 packets.
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.packetInHandler)

	log.InfoLog("ipv4/ENABLE_HOOK",
		"Mechanism IP enabled")
}

// Activate implements Mechanism interface
func (m *IPMechanism) Activate() {
	m.BaseNetworkMechanism.Activate()

	// Allocate table for handling ipv4 protocol.
	tableNo, err := m.C.Switch.AllocateTable()
	if err != nil {
		log.ErrorLog("ipv4/ACTIVATE_HOOK",
			"Failed to allocate a new table: ", err)

		return
	}

	m.tableNo = tableNo

	log.DebugLog("ipv4/ACTIVATE_HOOK",
		"Allocated table: ", tableNo)

	// Match packets of IPv4 protocol.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
	}}

	// Move all packets to allocated matching table for IPv4 packets.
	instructions := ofp.Instructions{ofp.InstructionGotoTable{ofp.Table(m.tableNo)}}

	// Insert flow into 0 table.
	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Priority:     10,
		Match:        match,
		Instructions: instructions,
	}))

	if err != nil {
		log.ErrorLog("ipv4/ACTIVATE_HOOK",
			"Failed to create ofp_flow_mod request: ", err)

		return
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("ipv4/ACTIVATE_HOOK",
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
		log.ErrorLog("ipv4/ACTIVATE_HOOK",
			"Failed to create ofp_flow_mod request: ", err)

		return
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("ipv4/ACTIVATE_HOOK",
			"Failed to send request: ", err)

		return
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("ipv4/ACTIVATE_HOOK",
			"Failed to flush requests: ", err)
	}
}

// Disable implements Mechanism interface
func (m *IPMechanism) Disable() {
	m.BaseNetworkMechanism.Disable()
	// pass
}

func (m *IPMechanism) UpdateNetwork(context *mech.NetworkContext) error {
	//lldriver, err := m.C.Link.Driver()
	//if err != nil {
	//log.ErrorLog("ip/UPDATE_NETWORK",
	//"Network layer driver is not intialized: ", err)
	//return err
	//}

	// Get link layer address associated with ingress port.
	//lladdr, err := lldriver.Addr(context.Port)
	//if err != nil {
	//log.ErrorLogf("ip/UPDATE_NETWORK",
	//"Failed to resolve port '%s' hardware address: '%s'", context.Port, err)
	//return err
	//}

	// Match IPv4 packets of specified subnetwork.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		//TODO: should be valid for all switch ports
		//ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_DST, lladdr.Bytes(), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, context.Addr.Bytes(), context.Addr.Mask()},
	}}

	// Send all such packets to controller.
	instruction := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS, ofp.Actions{
			ofp.ActionOutput{ofp.P_CONTROLLER, ofp.CML_NO_BUFFER},
		},
	}}

	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:      ofp.FC_ADD,
		TableID:      ofp.Table(m.tableNo),
		BufferID:     ofp.NO_BUFFER,
		Priority:     10,
		Match:        match,
		Instructions: instruction,
	}))

	if err != nil {
		log.ErrorLog("ip/UPDATE_NETWORK",
			"Failed to create new ofp_flow_mod request: ", err)
		return err
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("ip/UPDATE_NETWORK",
			"Failed to send ofp_flow_mode request: ", err)
		return err
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("ip/UPDATE_NETWORK",
			"Failed to flush requests: ", err)
	}

	return err
}

func (m *IPMechanism) DeleteNetwork(context *mech.NetworkContext) error {
	return nil
}

func (m *IPMechanism) packetInHandler(rw of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn
	var pdu2 mech.LinkFrame
	var pdu3 mech.NetworkPacket

	// TODO: differ ofp_packet_in messages by cookies.
	lldriver, err := m.C.Link.Driver()
	if err != nil {
		log.InfoLog("ip/PACKET_IN_HANDLER",
			"Link layer driver is not initialized: ", err)
		return
	}

	nldriver, err := m.C.Network.Driver()
	if err != nil {
		log.InfoLog("ip/PACKET_IN_HANDLER",
			"Network layer driver is not intialized: ", err)
		return
	}

	llreader := mech.MakeLinkReaderFrom(lldriver, &pdu2)
	nlreader := mech.MakeNetworkReaderFrom(nldriver, &pdu3)

	if _, err = of.ReadAllFrom(r.Body, &packet, llreader, nlreader); err != nil {
		log.ErrorLog("ip/PACKET_IN_HANDLER",
			"Failed to read packet: ", err)
		return
	}

	if int(packet.TableID) != m.tableNo {
		return
	}

	netwMech, err := m.C.Network.Mechanism(ARPMechanismName)
	if err != nil {
		log.ErrorLog("ip/PACKET_IN_HANDLER",
			"ARP network mechanism is not found: ", err)
	}

	arpMech, ok := netwMech.(*ARPMechanism)
	if !ok {
		log.ErrorLog("ip/PACKET_IN_HANDLER",
			"Failed to find ARP mechanism")
		return
	}

	_, err = arpMech.ResolveFunc(pdu3.DstAddr, uint32(2))
	if err != nil {
		log.ErrorLog("ip/PACKET_IN_HANDLER",
			"Failed to resolve hardware address: ", err)
	}

	//netw, err := m.T.Lookup(pduL3.Dst)
	//if err != nil {
	//log.DebugLog("ip/PACKET_HANDLER",
	//"Failed to find route: ", err)

	////TODO: Send ofp_packet_out ICMP destination network unreachable
	//return
	//}

	//TODO: Send ARP query to resolve next hop ip address to hw address

	//portNo := packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()
	//match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
	//ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
	//ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, of.Bytes(netw.IP), of.Bytes(netw.Mask)},
	//}}

	//var srcHWAddr []byte

	//// Get switch port source hardware address
	//srcHWAddr, err := m.C.Switch.PortHWAddr(int(portNo))
	//if err != nil {
	//log.ErrorLog("ip/PACKET_HANDLER",
	//"Failed to retrieve port hardware address: ", err)
	//return err
	//}

	//dstHWAddr := net.HardwareAddr{0, 0, 0, 0, 0, byte(portNo)}

	//// Change source and destination hardware addresses
	//dsthw := ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_DST, of.Bytes(dstHWAddr), nil}
	//srchw := ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_SRC, of.Bytes(srcHWAddr), nil}

	//instr := ofp.Instructions{ofp.InstructionActions{
	//ofp.IT_APPLY_ACTIONS,
	//ofp.Actions{
	//ofp.ActionSetField{dsthw},
	//ofp.ActionSetField{srchw},
	//ofp.Action{ofp.AT_DEC_NW_TTL},
	//ofp.ActionOutput{ofp.PortNo(portNo), 0},
	//},
	//}}

	//// TODO: priority based on administrative distance
	//// TODO: set expire timeout
	//r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
	//TableID:      TableIPv4,
	//Priority:     1,
	//Command:      ofp.FC_ADD,
	//BufferID:     ofp.NO_BUFFER,
	//Match:        match,
	//Instructions: instr,
	//}))

	//if err != nil {
	//log.ErrorLog("ip/PACKET_HANDLER",
	//"Failed to create new request: ", err)
	//return
	//}

	//if err = m.C.Switch.Conn().Send(r); err != nil {
	//log.Errorlog("ip/PACKET_HANDLER",
	//"Failed to write request: ", err)
	//return
	//}

	//if err = m.C.Switch.Conn().Flush(); err != nil {
	//log.ErrorLog("ip/PACKET_HANDLER",
	//"Failed to send ofp_flow_mode message: ", err)
	//}
}

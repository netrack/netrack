package ip

import (
	"github.com/netrack/net/iana"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/mechutil"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
	"github.com/netrack/openflow/ofp.v13/ofputil"
)

const IPv4MechanismName = "IPv4#RFC791"

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewIPMechanism)
	//constructor := mech.ComposuteNetworkMechanismConstructor(
	//mech.NetworkMechanismConstructorFunc(NewIPMechanism),
	//mech.NetworkMechanismConstructorFunc(NewICMPMechanism),
	//mech.NetworkMechanismConstructorFunc(NewARPMechanism),
	//)

	mech.RegisterNetworkMechanism(IPv4MechanismName, constructor)
}

type IPMechanism struct {
	mech.BaseNetworkMechanism

	cookies *of.CookieFilter

	// IPv4 routing table instance.
	routeTable mechutil.RoutingTable

	// Table number allocated for the mechanism.
	tableNo int
}

func NewIPMechanism() mech.NetworkMechanism {
	return &IPMechanism{
		cookies: of.NewCookieFilter(),
	}
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

	// Operate on PacketIn messages
	m.cookies.Baker = ofputil.PacketInBaker()

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

	flowMod := ofp.FlowMod{
		Command:      ofp.FC_ADD,
		TableID:      ofp.Table(m.tableNo),
		BufferID:     ofp.NO_BUFFER,
		Priority:     10,
		Match:        match,
		Instructions: instruction,
	}

	// Move ip packets to ipPacketHandler
	m.cookies.FilterFunc(&flowMod, m.ipPacketHandler)

	// Update routing table with new address
	m.routeTable.Populate(mechutil.RouteEntry{
		Type:    mechutil.ConnectedRoute,
		Network: context.Addr,
		Port:    context.Port,
	})

	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&flowMod))
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
	m.cookies.Serve(rw, r)
}

func (m *IPMechanism) ipPacketHandler(rw of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn
	var pdu2 mech.LinkFrame
	var pdu3 mech.NetworkPacket

	lldriver, err := m.C.Link.Driver()
	if err != nil {
		log.InfoLog("ip/IP_PACKET_HANDLER_LLDRIVER",
			"Link layer driver is not initialized: ", err)
		return
	}

	nldriver, err := m.C.Network.Driver()
	if err != nil {
		log.InfoLog("ip/IP_PACKET_HANDLER",
			"Network layer driver is not intialized: ", err)
		return
	}

	llreader := mech.MakeLinkReaderFrom(lldriver, &pdu2)
	nlreader := mech.MakeNetworkReaderFrom(nldriver, &pdu3)

	if _, err = of.ReadAllFrom(r.Body, &packet, llreader, nlreader); err != nil {
		log.ErrorLog("ip/IP_PACKET_HANDLER",
			"Failed to read packet: ", err)
		return
	}

	log.DebugLog("ip/IP_PACKET_HANDLER",
		"Got ip packet to: ", pdu3.DstAddr)

	route, ok := m.routeTable.Lookup(pdu3.DstAddr)
	if !ok {
		log.DebugLogf("ip/IP_PACKET_HANDLER",
			"Route to %s not found", pdu3.DstAddr)
		return
	}

	// Search for link layer address of egress port.
	srcAddr, err := lldriver.Addr(route.Port)
	if err != nil {
		log.ErrorLog("ip/PACKET_IN_HANDLER",
			"Failed to retrieve port link layer address: ", err)
		return
	}

	var arpMech ARPMechanism
	err = m.C.Network.Mechanism(ARPMechanismName, &arpMech)
	if err != nil {
		log.ErrorLog("ip/IP_PACKET_HANDLER",
			"ARP network mechanism is not found: ", err)
		return
	}

	dstAddr, err := arpMech.ResolveFunc(pdu3.DstAddr, route.Port)
	if err != nil {
		log.ErrorLog("ip/IP_PACKET_HANDLER",
			"Failed to resolve link layer address: ", err)
		return
	}

	log.DebugLog("ip/IP_PACKET_HANDLER",
		"Resolved link layer address: ", dstAddr)

	// Create permanent rule for discovered address.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, pdu3.DstAddr.Bytes(), nil},
	}}

	// Change source and destination link layer addresses
	setDst := ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_DST, dstAddr.Bytes(), nil}
	setSrc := ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_SRC, srcAddr.Bytes(), nil}

	instructions := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS,
		ofp.Actions{
			ofp.ActionSetField{setDst},
			ofp.ActionSetField{setSrc},
			ofp.Action{ofp.AT_DEC_NW_TTL},
			ofp.ActionOutput{ofp.PortNo(route.Port), 0},
		},
	}}

	// TODO: set expire timeout
	flowMod := ofp.FlowMod{
		Command:      ofp.FC_ADD,
		TableID:      ofp.Table(m.tableNo),
		BufferID:     ofp.NO_BUFFER,
		Priority:     20,
		Match:        match,
		Instructions: instructions,
	}

	_, err = of.WriteAllTo(rw, &flowMod)
	if err != nil {
		log.ErrorLog("ip/IP_PACKET_HANDLER",
			"Failed to write response: ", err)
		return
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_FLOW_MOD)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	if err = rw.WriteHeader(); err != nil {
		log.ErrorLog("ip/IP_PACKET_HANDLER",
			"Failed to send ICMP-REPLY response: ", err)
	}
}

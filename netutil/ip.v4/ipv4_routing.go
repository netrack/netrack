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

const IPv4RoutingName = "ipv4"

func init() {
	constructor := mech.RouteMechanismConstructorFunc(NewIPv4Routing)
	mech.RegisterRouteMechanism(IPv4RoutingName, constructor)
}

type IPv4Routing struct {
	mech.BaseRouteMechanism

	cookies *of.CookieFilter

	// IPv4 routing table instance.
	routeTable *mechutil.RoutingTable

	// Table number allocated for the mechanism.
	tableNo int
}

func NewIPv4Routing() mech.RouteMechanism {
	return &IPv4Routing{
		cookies:    of.NewCookieFilter(),
		routeTable: mechutil.NewRoutingTable(),
	}
}

// Enable implements Mechanism interface
func (m *IPv4Routing) Enable(c *mech.MechanismContext) {
	m.BaseRouteMechanism.Enable(c)

	// Handle incoming IPv4 packets.
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.packetInHandler)

	log.InfoLog("routing/ENABLE_HOOK",
		"IPv4 routing enabled")
}

func (m *IPv4Routing) Activate() {
	m.BaseRouteMechanism.Activate()

	// Operate on PacketIn messages
	m.cookies.Baker = ofputil.PacketInBaker()

	// Allocate table for handling ipv4 protocol.
	tableNo, err := m.C.Switch.AllocateTable()
	if err != nil {
		log.ErrorLog("routing/ACTIVATE_HOOK",
			"Failed to allocate a new table: ", err)
		return
	}

	m.tableNo = tableNo

	log.DebugLog("routing/ACTIVATE_HOOK",
		"Allocated table: ", tableNo)

	// Match packets of IPv4 protocol.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
	}}

	// Move all packets to allocated matching table for IPv4 packets.
	instructions := ofp.Instructions{ofp.InstructionGotoTable{ofp.Table(m.tableNo)}}

	// Insert flow into 0 table.
	flowModGoto, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Priority:     10,
		Match:        match,
		Instructions: instructions,
	}))

	if err != nil {
		log.ErrorLog("routing/ACTIVATE_HOOK",
			"Failed to create ofp_flow_mod request: ", err)

		return
	}

	err = of.Send(m.C.Switch.Conn(),
		// Flush flows from table before using it.
		ofputil.TableFlush(ofp.Table(m.tableNo)),
		// Create black-hole rule for non-matching packets.
		ofputil.FlowDrop(ofp.Table(m.tableNo)),
		// Redirect all ARP requests to allocated table to process.
		flowModGoto,
	)

	if err != nil {
		log.ErrorLog("routing/ACTIVATE_HOOK",
			"Failed to send requests: ", err)
	}
}

func (m *IPv4Routing) UpdateRoute(context *mech.RouteContext) error {
	nldriver, err := m.C.Network.Driver()
	if err != nil {
		log.InfoLog("routing/CREATE_ROUTE",
			"Network layer driver is not intialized: ", err)
		return err
	}

	network, err := nldriver.ParseAddr(context.Network)
	if err != nil {
		log.ErrorLog("routing/CREATE_ROUTE",
			"Failed to parse network string: ", err)
		return err
	}

	nextHop, err := nldriver.ParseAddr(context.NextHop)
	if context.NextHop != "" && err != nil {
		log.ErrorLog("routing/CREATE_ROUTE",
			"Failed to parse next-hop string: ", err)
		return err
	}

	// Match IPv4 packets of specified route.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, network.Bytes(), network.Mask()},
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
		Priority:     15,
		Match:        match,
		Instructions: instruction,
	}

	// Move ip packets to ipPacketHandler
	m.cookies.FilterFunc(&flowMod, m.ipPacketHandler)

	// Update routing table with new address
	m.routeTable.Populate(mechutil.RouteEntry{
		Type:    mechutil.RouteType(context.Type),
		Network: network,
		NextHop: nextHop,
		Port:    context.Port,
	})

	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&flowMod))
	if err != nil {
		log.ErrorLog("routing/UPDATE_ROUTES",
			"Failed to create new ofp_flow_mod request: ", err)
		return err
	}

	if err = of.Send(m.C.Switch.Conn(), r); err != nil {
		log.ErrorLog("routing/UPDATE_ROUTES",
			"Failed to send ofp_flow_mode request: ", err)
	}

	return err
}

func (m *IPv4Routing) DeleteRoute(context *mech.RouteContext) error {
	log.DebugLog("routing/DELETE_ROUTE",
		"Got delete route request")

	nldriver, err := m.C.Network.Driver()
	if err != nil {
		log.InfoLog("routing/DELETE_ROUTE",
			"Network layer driver is not intialized: ", err)
		return err
	}

	network, err := nldriver.ParseAddr(context.Network)
	if err != nil {
		log.ErrorLog("routing/DELETE_ROUTE",
			"Failed to parse network string: ", err)
		return err
	}

	nextHop, err := nldriver.ParseAddr(context.NextHop)
	if context.NextHop != "" && err != nil {
		log.ErrorLog("routing/DELETE_ROUTE",
			"Failed to parse next-hop string: ", err)
		return err
	}

	// Update routing table with new address
	evicted := m.routeTable.Evict(mechutil.RouteEntry{
		Network: network,
		NextHop: nextHop,
		Port:    context.Port,
	})

	if !evicted {
		log.ErrorLog("routing/DELETE_ROUTE",
			"Failed to delete specified route: ", network)
		return nil
	}

	// Match IPv4 packets of specified route.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, network.Bytes(), network.Mask()},
	}}

	err = of.Send(m.C.Switch.Conn(),
		ofputil.FlowFlush(ofp.Table(m.tableNo), match),
	)

	if err != nil {
		log.ErrorLog("routing/DELETE_ROUTES",
			"Failed to send requests: ", err)
	}

	return err
}

func (m *IPv4Routing) packetInHandler(rw of.ResponseWriter, r *of.Request) {
	m.cookies.Serve(rw, r)
}

func (m *IPv4Routing) ipPacketHandler(rw of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn
	var pdu2 mech.LinkFrame
	var pdu3 mech.NetworkPacket

	lldriver, err := m.C.Link.Driver()
	if err != nil {
		log.InfoLog("routing/IP_PACKET_HANDLER_LLDRIVER",
			"Link layer driver is not initialized: ", err)
		return
	}

	nldriver, err := m.C.Network.Driver()
	if err != nil {
		log.InfoLog("routing/IP_PACKET_HANDLER",
			"Network layer driver is not intialized: ", err)
		return
	}

	llreader := mech.MakeLinkReaderFrom(lldriver, &pdu2)
	nlreader := mech.MakeNetworkReaderFrom(nldriver, &pdu3)

	if _, err = of.ReadAllFrom(r.Body, &packet, llreader, nlreader); err != nil {
		log.ErrorLog("routing/IP_PACKET_HANDLER",
			"Failed to read packet: ", err)
		return
	}

	log.DebugLog("routing/IP_PACKET_HANDLER",
		"Got ip packet to: ", pdu3.DstAddr)

	route, ok := m.routeTable.Lookup(pdu3.DstAddr)
	if !ok {
		log.DebugLogf("routing/IP_PACKET_HANDLER",
			"Route to %s not found", pdu3.DstAddr)
		return
	}

	// Search for link layer address of egress port.
	srcAddr, err := lldriver.Addr(route.Port)
	if err != nil {
		log.ErrorLog("routing/PACKET_IN_HANDLER",
			"Failed to retrieve port link layer address: ", err)
		return
	}

	var arpMech ARPMechanism
	err = m.C.Network.Mechanism(ARPMechanismName, &arpMech)
	if err != nil {
		log.ErrorLog("routing/IP_PACKET_HANDLER",
			"ARP network mechanism is not found: ", err)
		return
	}

	netwAddr := route.NextHop
	if netwAddr == nil {
		netwAddr = pdu3.DstAddr
	}

	dstAddr, err := arpMech.ResolveFunc(netwAddr, route.Port)
	if err != nil {
		log.ErrorLog("routing/IP_PACKET_HANDLER",
			"Failed to resolve link layer address: ", err)
		return
	}

	log.DebugLog("routing/IP_PACKET_HANDLER",
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
		Priority:     25,
		Match:        match,
		Instructions: instructions,
	}

	_, err = of.WriteAllTo(rw, &flowMod)
	if err != nil {
		log.ErrorLog("routing/IP_PACKET_HANDLER",
			"Failed to write response: ", err)
		return
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_FLOW_MOD)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	if err = rw.WriteHeader(); err != nil {
		log.ErrorLog("routing/IP_PACKET_HANDLER",
			"Failed to send ICMP-REPLY response: ", err)
	}
}

package ip

import (
	"errors"
	"net"
	"sort"
	"sync"

	"github.com/netrack/net/iana"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewIPMechanism)
	mech.RegisterMechanismDriver(mech.NetworkProtoIPv4, constructor)
}

const (
	StaticRoute    RouteType = "S"
	LocalRoute     RouteType = "L"
	ConnectedRoute RouteType = "C"
	EIGRPRoute     RouteType = "D"
	OSPFRoute      RouteType = "O"
	RIPRoute       RouteType = "R"
)

var distanceMap = map[RouteType]int{
	StaticRoute:    0,
	ConnectedRoute: 1,
	EIGRPRoute:     90,
	OSPFRoute:      110,
	RIPRoute:       120,
}

func routeToDistance(r RouteType) (int, error) {
	distance, ok := distanceMap[r]
	if !ok {
		return 0, errors.New("ip: unsupported route type")
	}

	return distance, nil
}

const TableIPv4 ofp.Table = 4

type RouteType string

type RouteEntry struct {
	Type     RouteType
	Net      net.IPNet
	NextHop  net.IP
	Distance int
	//Metric
	//Timestamp
	Port ofp.PortNo
}

type RoutingTable struct {
	routes []RouteEntry
	lock   sync.RWMutex
}

func (t *RoutingTable) Append(entry RouteEntry) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.routes = append(t.routes, entry)
	sort.Sort(t)

	return nil
}

func (t *RoutingTable) Lookup(ipaddr net.IP) (net.IPNet, error) {
	return net.IPNet{}, nil
}

func (t *RoutingTable) Len() int {
	return len(t.routes)
}

func (t *RoutingTable) Less(i, j int) bool {
	if t.routes[i].Distance < t.routes[j].Distance {
		return true
	}

	// Compare metric
	//if r.routes[i].Metric < r.routes[j].Metric {
	//}

	return false
}

func (t *RoutingTable) Swap(i, j int) {
	t.routes[i], t.routes[j] = t.routes[j], t.routes[i]
}

type IPMechanism struct {
	mech.BaseMechanismDriver

	// IPv4 routing table instance.
	T RoutingTable

	// Table number allocated for the mechanism.
	tableNo int
}

func NewIPMechanism() mech.MechanismDriver {
	return &IPMechanism{}
}

// Enable implements MechanismDriver interface
func (m *IPMechanism) Enable(c *mech.MechanismDriverContext) {
	m.BaseMechanismDriver.Enable(c)

	log.InfoLog("ipv4/ENABLE_HOOK",
		"Mechanism IP enabled")
}

// Activate implements MechanismDriver interface
func (m *IPMechanism) Activate() {
	m.BaseMechanismDriver.Activate()

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

// Disable implements MechanismDriver interface
func (m *IPMechanism) Disable() {
	m.BaseMechanismDriver.Disable()
	// pass
}

// getAddressFunc returnes IPv4 address and network mask in a result
// variable, error will be returned if neither port exists, nor IPv4
// address assigned to required port.
func (m *IPMechanism) getAddressFunc(param rpc.Param, result rpc.Result) error {
	return nil
}

func (m *IPMechanism) addAddressFunc(param rpc.Param, result rpc.Result) error {
	return nil
}

func (m *IPMechanism) deleteAddressFunc(param rpc.Param, result rpc.Result) error {
	return nil
}

func (m *IPMechanism) addRouteFunc(entry RouteEntry) error {
	//_, netw, _ := net.ParseCIDR(s)

	if err := m.T.Append(entry); err != nil {
		log.ErrorLog("ip/ADD_ROUTE_ERR",
			"Failed to add a new route: ", err)
		return err
	}

	return nil
}

func (m *IPMechanism) packetHandler(rw *of.ResponseWriter, r *of.Request) {
	//var packet ofp.PacketIn

	//if _, err := packet.ReadFrom(r.Body); err != nil {
	//log.ErrorLog("ip/PACKET_HANDLER",
	//"Failed to read ofp_packet_in message: ", err)
	//return
	//}

	//if packet.TableID != TableIPv4 {
	//log.DebugLog("ip/PACKET_HANDLER",
	//"Received packet from wrong table")
	//return
	//}

	//var pduL2 l2.EthernetII
	//var pduL3 l3.IPv4

	//if _, err := of.ReadAllFrom(r.Body, &pduL2, &pduL3); err != nil {
	//log.ErrorLog("ip/PACKET_HANDLER",
	//"Failed to unmarshal arrived packet: ", err)
	//return
	//}

	//netw, err := m.T.Lookup(pduL3.Dst)
	//if err != nil {
	//log.DebugLog("ip/PACKET_HANDLER",
	//"Failed to find route: ", err)

	////TODO: Send ofp_packet_out ICMP destination network unreachable
	//return
	//}

	////TODO: Send ARP query to resolve next hop ip address to hw address

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
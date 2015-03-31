package ip

import (
	"errors"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/netrack/net/iana"
	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

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
}

func (t *RoutingTable) Lookup(ipaddr net.IP) (net.IPNet, error) {
	return nil, nil
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

type IPMech struct {
	C *mech.OFPContext
	T RoutingTable
}

func (m *IPMech) Initialize(c *mech.OFPContext) {
	m.C = c

	//m.C.Mux.HandleFunc(of.T_HELLO, m.helloHandler)

	log.InfoLog("ip/INIT_DONE", "IP mechanism successfully intialized")
}

func (m *IPMech) helloHandler(rw of.ResponseWriter, r *of.Request) {
	go func() {
		time.Sleep(time.Second * 7)
		//m.AddRoute(RouteEntry{StaticRoute, })
	}()
}

func (m *IPMech) addRoute(entry RouteEntry) error {
	//_, netw, _ := net.ParseCIDR(s)

	if err := m.T.Append(entry); err != nil {
		log.ErrorLog("ip/ADD_ROUTE_ERR",
			"Failed to add a new route: ", err)
		return err
	}

	return nil
}

func (m *IPMech) packetHandler(rw *of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn

	if _, err := packet.ReadFrom(r.Body); err != nil {
		log.ErrorLog("ip/PACKET_HANDLER",
			"Failed to read ofp_packet_in message: ", err)
		return
	}

	if packet.TableID != TableIPv4 {
		log.DebugLog("ip/PACKET_HANDLER",
			"Received packet from wrong table")
		return
	}

	var pduL2 l2.EthernetII
	var pduL3 l3.IPv4

	if _, err := of.ReadAllFrom(r.Body, &pduL2, &pduL3); err != nil {
		log.ErrorLog("ip/PACKET_HANDLER",
			"Failed to unmarshal arrived packet: ", err)
		return
	}

	netw, err := m.T.Lookup(pduL3.Dst)
	if err != nil {
		log.DebugLog("ip/PACKET_HANDLER",
			"Failed to find route: ", err)

		//TODO: Send ofp_packet_out ICMP destination network unreachable
		return
	}

	//TODO: Send ARP query to resolve next hop ip address to hw address

	portNo := packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, of.Bytes(netw.IP), of.Bytes(netw.Mask)},
	}}

	var srcHWAddr []byte

	// Get switch port source hardware address
	err := m.C.R.Call(rpc.T_OFP_PORT_HWADDR,
		rpc.UInt16Param(uint16(portNo)),
		rpc.ByteSliceResult(&srcHWAddr))

	if err != nil {
		log.ErrorLog("ip/PACKET_HANDLER",
			"Failed to retrieve port hardware address: ", err)
		return err
	}

	dstHWAddr := net.HardwareAddr{0, 0, 0, 0, 0, byte(portNo)}

	// Change source and destination hardware addresses
	dsthw := ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_DST, of.Bytes(dstHWAddr), nil}
	srchw := ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_SRC, of.Bytes(srcHWAddr), nil}

	instr := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS,
		ofp.Actions{
			ofp.ActionSetField{dsthw},
			ofp.ActionSetField{srchw},
			ofp.Action{ofp.AT_DEC_NW_TTL},
			ofp.ActionOutput{ofp.PortNo(portNo), 0},
		},
	}}

	// TODO: priority based on administrative distance
	// TODO: set expire timeout
	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		TableID:      TableIPv4,
		Priority:     1,
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Match:        match,
		Instructions: instr,
	}))

	if err != nil {
		log.ErrorLog("ip/PACKET_HANDLER",
			"Failed to create new request: ", err)
		return
	}

	if err = m.C.Conn.Send(r); err != nil {
		log.Errorlog("ip/PACKET_HANDLER",
			"Failed to write request: ", err)
		return
	}

	if err = m.C.Conn.Flush(); err != nil {
		log.ErrorLog("ip/PACKET_HANDLER",
			"Failed to send ofp_flow_mode message: ", err)
	}
}

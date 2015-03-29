package ip

import (
	"net"
	"sync"

	"github.com/netrack/net/iana"
	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

const (
	STATIC_ROUTE RouteType = "S"
)

const (
	IPV4_TABLE_ID ofp.Table = 0
)

type RouteType string

type Route struct {
	Type    RouteType
	Net     net.IPNet
	NextHop net.IPAddr
	//Distance
	//Metric
	//Timestamp
	Port ofp.PortNo
}

type RoutingTable struct {
	routes []Route
	lock   sync.RWMutex
}

type IPMech struct {
	C *mech.OFPContext
	T RoutingTable
}

func (m *IPMech) Initialize(c *mech.OFPContext) {
	m.C = c

	m.C.Mux.HandleFunc(of.T_HELLO, m.helloHandler)

	log.InfoLog("ip/INIT_DONE", "IP mechanism successfully intialized")
}

func (m *IPMech) helloHandler(rw of.ResponseWriter, r *of.Request) {
	m.Add("10.0.1.1/24", 1)
	m.Add("10.0.2.1/24", 2)
	m.Add("10.0.3.1/24", 3)
}

func (m *IPMech) Add(s string, portNo ofp.PortNo) {
	_, netw, _ := net.ParseCIDR(s)
	hwaddr := net.HardwareAddr{0, 0, 0, 0, 0, byte(portNo)}

	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, of.Bytes(netw.IP), of.Bytes(netw.Mask)},
	}}

	//hwaddr, err := m.C.RPC.Call(rpc.ARP_LOOKUP, netw.IP)

	fields := []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_DST, of.Bytes(hwaddr), nil},
	}

	instr := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS,
		ofp.Actions{
			ofp.ActionSetField{fields},
			ofp.Action{ofp.AT_DEC_NW_TTL},
			ofp.ActionOutput{portNo, 0},
		},
	}}

	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		//TableID:      IPV4_TABLE_ID,
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Match:        match,
		Instructions: instr,
	}))

	if err != nil {
		return
	}

	m.C.Conn.Send(r)
	m.C.Conn.Flush()
}

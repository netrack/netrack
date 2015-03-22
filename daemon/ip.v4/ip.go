package ip

import (
	"net"

	"github.com/netrack/net/iana"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

type RouteEntry struct {
	//Source RouteSrouce
	//Network
	//NextHop
	//Distance
	//Metric
	//Timestamp
	//IFace
}

type RoutingTable struct {
}

type Router struct {
	Table RoutingTable
}

func (router *Router) AddRoute(rw of.ResponseWriter, s string, portNo ofp.PortNo) {
	rw.Header().Set(of.TypeHeaderKey, of.T_FLOW_MOD)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	_, netw, _ := net.ParseCIDR(s)

	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, of.Bytes(netw.IP), of.Bytes(netw.Mask)},
	}}

	instr := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS,
		ofp.Actions{
			ofp.Action{ofp.AT_DEC_NW_TTL},
			ofp.ActionOutput{portNo, 0},
			ofp.ActionOutput{ofp.P_CONTROLLER, ofp.CML_NO_BUFFER},
		},
	}}

	// TODO: Add SET_FIELD action to replace destination HWAddr

	fmod := &ofp.FlowMod{
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Match:        match,
		Instructions: instr,
	}

	fmod.WriteTo(rw)
	rw.WriteHeader()
}

func (router *Router) Hello(rw of.ResponseWriter, r *of.Request) {
	router.AddRoute(rw, "10.0.1.1/24", 1)
	router.AddRoute(rw, "10.0.2.1/24", 2)
	router.AddRoute(rw, "10.0.3.1/24", 3)
}

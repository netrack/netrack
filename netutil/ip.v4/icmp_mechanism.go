package ip

import (
	//"net"

	"github.com/netrack/net/iana"
	//"github.com/netrack/net/l2"
	//"github.com/netrack/net/l3"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewICMPMechanism)
	mech.RegisterNetworkMechanism("ICMP", constructor)
}

type ICMPMechanism struct {
	mech.BaseNetworkMechanism

	tableNo int
}

func NewICMPMechanism() mech.NetworkMechanism {
	return &ICMPMechanism{}
}

func (m *ICMPMechanism) Enable(c *mech.MechanismContext) {
	m.BaseNetworkMechanism.Enable(c)

	log.InfoLog("icmp/ENABLE_HOOK",
		"Mechanism ICMP mechanism enabled")
}

func (m *ICMPMechanism) Activate() {
	m.BaseNetworkMechanism.Activate()

	// Allocate table for handling icmp protocol.
	tableNo, err := m.C.Switch.AllocateTable()
	if err != nil {
		log.ErrorLog("icmp/ACTIVATE_HOOK",
			"Failed to allocate a new table: ", err)

		return
	}

	m.tableNo = tableNo

	log.DebugLog("icmp/ACTIVATE_HOOK",
		"Allocated table: ", tableNo)

	// Match packets of ICMP protocol.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IP_PROTO, of.Bytes(iana.IP_PROTO_ICMP), nil},
	}}

	// Move all packets to allocated matching table for ICMP packets.
	instructions := ofp.Instructions{ofp.InstructionGotoTable{ofp.Table(m.tableNo)}}

	// Insert flow into 0 table.
	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Priority:     15,
		Match:        match,
		Instructions: instructions,
	}))

	if err != nil {
		log.ErrorLog("icmp/ACTIVATE_HOOK",
			"Failed to create ofp_flow_mod request: ", err)

		return
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("icmp/ACTIVATE_HOOK",
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
		log.ErrorLog("icmp/ACTIVATE_HOOK",
			"Failed to create ofp_flow_mod request: ", err)

		return
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("icmp/ACTIVATE_HOOK",
			"Failed to send request: ", err)

		return
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("icmp/ACTIVATE_HOOK",
			"Failed to flush requests: ", err)
	}
}

func (m *ICMPMechanism) UpdateNetwork(context *mech.NetworkContext) error {
	return nil
}

func (m *ICMPMechanism) DeleteNetwork(context *mech.NetworkContext) error {
	return nil
}

func (m *ICMPMechanism) packetInHandler(rw of.ResponseWriter, r *of.Request) {
	//var p ofp.PacketIn
	//p.ReadFrom(r.Body)

	//var eth l2.EthernetII
	//if eth.ReadFrom(r.Body); eth.EthType != iana.ETHT_IPV4 {
	//return
	//}

	//var ip l3.IPv4
	//if ip.ReadFrom(r.Body); ip.Proto != iana.IP_PROTO_ICMP {
	//return
	//}

	//icmp := l3.ICMPEcho{Data: make([]byte, ip.Len-l3.IPv4HeaderLen-l3.ICMPHeaderLen)}
	//if icmp.ReadFrom(r.Body); icmp.Type != l3.ICMPT_ECHO_REQUEST {
	//return
	//}

	//icmp.Type = l3.ICMPT_ECHO_REPLY
	//payload := of.NewReader(&icmp)

	//var hwaddr []byte
	//portNo := p.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()

	//err := m.BaseNetworkMechanism.C.Func.Call(rpc.T_OFP_PORT_HWADDR,
	//rpc.UInt16Param(uint16(portNo)),
	//rpc.ByteSliceResult(&hwaddr))

	//if err != nil {
	//log.ErrorLog("icmp/PACKET_IN_HWADDR_ERR",
	//"Failed to retrieve port hardware address: ", err)
	//return
	//}

	//eth = l2.EthernetII{eth.HWSrc, net.HardwareAddr(hwaddr), iana.ETHT_IPV4}
	//ip = l3.IPv4{Src: ip.Dst, Dst: ip.Src, Proto: iana.IP_PROTO_ICMP, Payload: payload}

	//pout := ofp.PacketOut{BufferID: ofp.NO_BUFFER,
	//InPort:  ofp.P_CONTROLLER,
	//Actions: ofp.Actions{ofp.ActionOutput{ofp.P_IN_PORT, 0}},
	//}

	//_, err = of.WriteAllTo(rw, &pout, &eth, &ip)
	//if err != nil {
	//log.ErrorLog("icmp/PACKET_IN_WRITE_ERR",
	//"Failed to write response: ", err)
	//return
	//}

	//rw.Header().Set(of.TypeHeaderKey, of.T_PACKET_OUT)
	//rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	//rw.WriteHeader()
}

func (m *ICMPMechanism) Add(param rpc.Param, result rpc.Result) error {
	//var ipaddr []byte
	//var portNo uint16

	//if err := param.Obtain(&ipaddr, &portNo); err != nil {
	//log.ErrorLog("icmp/ADD_ICMP_SERVER_PARAM_ERR",
	//"Failed to obtain parameters: ", err)
	//return err
	//}

	//var hwaddr []byte
	//err := m.BaseNetworkMechanism.C.Func.Call(rpc.T_OFP_PORT_HWADDR,
	//rpc.UInt16Param(portNo),
	//rpc.ByteSliceResult(&hwaddr))

	//if err != nil {
	//log.ErrorLog("icmp/ADD_ICMP_SERVER_HWADDR_ERR",
	//"Failed to return port hardware address: ", err)
	//return err
	//}

	////TODO:
	//ipaddr[3] = 254

	//match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
	//ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
	//ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_DST, hwaddr, nil},
	//ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, ipaddr, nil},
	//ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IP_PROTO, of.Bytes(iana.IP_PROTO_ICMP), nil},
	//ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ICMPV4_TYPE, of.Bytes(l3.ICMPT_ECHO_REQUEST), nil},
	//}}

	//instr := ofp.Instructions{ofp.InstructionActions{
	//ofp.IT_APPLY_ACTIONS,
	//ofp.Actions{ofp.ActionOutput{ofp.P_CONTROLLER, ofp.CML_NO_BUFFER}},
	//}}

	//req, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
	//Command:      ofp.FC_ADD,
	//BufferID:     ofp.NO_BUFFER,
	//Match:        match,
	//Instructions: instr,
	//}))

	//if err != nil {
	//log.ErrorLog("icmp/ADD_ICMP_SERVER_REQUEST_ERR",
	//"Failed to create a new ofp_flow_mod request: ", err)
	//}

	//if err = m.BaseNetworkMechanism.C.Conn.Send(req); err != nil {
	//log.ErrorLogf("icmp/ADD_ICMP_SERVER_SEND_ERR",
	//"Failed to send ofp_flow_mod request:", err)
	//}

	//return err
	return nil
}

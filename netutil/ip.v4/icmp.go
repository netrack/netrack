package ip

import (
	"github.com/netrack/net/iana"
	"github.com/netrack/net/l3"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
	"github.com/netrack/openflow/ofp.v13/ofputil"
)

const ICMPMechanismName = "icmp"

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewICMPMechanism)
	mech.RegisterNetworkMechanism(ICMPMechanismName, constructor)
}

func EchoRequest(ipaddr []byte) ofp.Match {
	// Match ICMP echo-request messages to created network address.
	return ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		//TODO: make it available for all link layer addresses.
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, ipaddr, nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IP_PROTO, of.Bytes(iana.IP_PROTO_ICMP), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ICMPV4_TYPE, of.Bytes(l3.ICMPT_ECHO_REQUEST), nil},
	}}
}

type ICMPMechanism struct {
	mech.BaseNetworkMechanism

	// Handle request based on cookie value.
	cookies *of.CookieFilter
}

func NewICMPMechanism() mech.NetworkMechanism {
	return &ICMPMechanism{
		cookies: of.NewCookieFilter(),
	}
}

func (m *ICMPMechanism) Enable(c *mech.MechanismContext) {
	m.BaseNetworkMechanism.Enable(c)

	// Handle incoming ICMP requests.
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.packetInHandler)
	m.C.Mux.HandleFunc(of.T_FLOW_REMOVED, m.flowRemovedHandler)

	log.InfoLog("icmp/ENABLE_HOOK", "Mechanism ICMP enabled")
}

func (m *ICMPMechanism) Activate() {
	m.BaseNetworkMechanism.Activate()

	// Operate on PacketIn messages
	m.cookies.Baker = ofputil.PacketInBaker()
}

func (m *ICMPMechanism) UpdateNetwork(context *mech.NetworkContext) error {
	log.DebugLog("icmp/UPDATE_NETWORK",
		"Got update network request")

	// Send ICMP message to the controller
	instructions := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS,
		ofp.Actions{ofp.ActionOutput{ofp.P_CONTROLLER, ofp.CML_NO_BUFFER}},
	}}

	// Insert flow into ICMP-allocated table.
	flowMod := ofp.FlowMod{
		Command:  ofp.FC_ADD,
		BufferID: ofp.NO_BUFFER,
		// Notify controller, when flow removed
		Flags:        ofp.FF_SEND_FLOW_REM,
		Priority:     30, // Use non-zero priority
		Match:        EchoRequest(context.Addr.Bytes()),
		Instructions: instructions,
	}

	// Assign cookie to FlowMod message, and
	// redirect such requests to icmpEchoHandler
	m.cookies.FilterFunc(&flowMod, m.icmpEchoHandler)

	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&flowMod))
	if err != nil {
		log.ErrorLog("icmp/UPDATE_NETWORK",
			"Failed to create a new ofp_flow_mod request: ", err)
		return err
	}

	if err = of.Send(m.C.Switch.Conn(), r); err != nil {
		log.ErrorLogf("icmp/UPDATE_NETWORK",
			"Failed to send request:", err)
	}

	return err
}

func (m *ICMPMechanism) DeleteNetwork(context *mech.NetworkContext) error {
	// Flush ICMP flow for specified address (if any).
	err := of.Send(m.C.Switch.Conn(), ofputil.FlowFlush(
		0, EchoRequest(context.Addr.Bytes()),
	))

	if err != nil {
		log.ErrorLog("icmp/DELETE_NETWORK",
			"Failed to send requests: ", err)
	}

	return err
}

func (m *ICMPMechanism) packetInHandler(rw of.ResponseWriter, r *of.Request) {
	m.cookies.Serve(rw, r)
}

func (m *ICMPMechanism) flowRemovedHandler(rw of.ResponseWriter, r *of.Request) {
	var flowRemoved ofp.FlowRemoved

	_, err := of.ReadAllFrom(r.Body, &flowRemoved)
	if err != nil {
		return
	}

	m.cookies.Release(&flowRemoved)
}

func (m *ICMPMechanism) icmpEchoHandler(rw of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn
	var pdu2 mech.LinkFrame
	var pdu3 mech.NetworkPacket

	// TODO: differ ofp_packet_in messages by cookies.
	lldriver, err := m.C.Link.Driver()
	if err != nil {
		log.InfoLog("icmp/PACKET_IN_HANDLER",
			"Link layer driver is not initialized: ", err)
		return
	}

	nldriver, err := m.C.Network.Driver()
	if err != nil {
		log.InfoLog("icmp/PACKET_IN_HANDLER",
			"Network layer driver is not intialized: ", err)
		return
	}

	llreader := mech.MakeLinkReaderFrom(lldriver, &pdu2)
	nlreader := mech.MakeNetworkReaderFrom(nldriver, &pdu3)

	if _, err = of.ReadAllFrom(r.Body, &packet, llreader, nlreader); err != nil {
		log.ErrorLog("icmp/PACKET_IN_HANDLER",
			"Failed to read packet: ", err)
		return
	}

	// Read icmp echo-request message
	icmp := l3.ICMPEcho{Data: make([]byte, pdu3.ContentLen-l3.ICMPHeaderLen)}
	if _, err = of.ReadAllFrom(r.Body, &icmp); err != nil {
		log.ErrorLog("icmp/PACKET_IN_HANDLER",
			"Failed to read ICMP message: ", err)
		return
	}

	log.DebugLog("icmp/ECHO_REQUEST_HANDLER",
		"Got ICMP echo-request: %s -> %s", pdu3.SrcAddr, pdu3.DstAddr)

	// Get port number from match fields.
	portNo := packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()

	// Search for link layer address of egress port.
	lladdr, err := lldriver.Addr(portNo)
	if err != nil {
		log.ErrorLog("icmp/PACKET_IN_HWADDR_ERR",
			"Failed to retrieve port hardware address: ", err)
		return
	}

	// Build link layer PDU.
	pdu2 = mech.LinkFrame{pdu2.SrcAddr, lladdr, mech.Proto(iana.ETHT_IPV4), 0}

	// Send echo-reply message.
	icmp.Type = l3.ICMPT_ECHO_REPLY

	// Build network layer PDU.
	pdu3 = mech.NetworkPacket{
		DstAddr: pdu3.SrcAddr,
		SrcAddr: pdu3.DstAddr,
		Proto:   pdu3.Proto,
		Payload: of.NewReader(&icmp),
	}

	packetOut := ofp.PacketOut{BufferID: ofp.NO_BUFFER,
		InPort:  packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.PortNo(),
		Actions: ofp.Actions{ofp.ActionOutput{ofp.P_IN_PORT, 0}},
	}

	llwriter := mech.MakeLinkWriterTo(lldriver, &pdu2)
	nlwriter := mech.MakeNetworkWriterTo(nldriver, &pdu3)

	_, err = of.WriteAllTo(rw, &packetOut, llwriter, nlwriter)
	if err != nil {
		log.ErrorLog("icmp/PACKET_IN_HANDLER",
			"Failed to write response: ", err)
		return
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_PACKET_OUT)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	if err = rw.WriteHeader(); err != nil {
		log.ErrorLog("icmp/PACKET_IN_HANDLER",
			"Failed to send ICMP-REPLY response: ", err)
	}
}

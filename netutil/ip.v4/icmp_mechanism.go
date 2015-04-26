package ip

import (
	"github.com/netrack/net/iana"
	"github.com/netrack/net/l3"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

const ICMPMechanismName = "ICMP#RFC792"

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewICMPMechanism)
	mech.RegisterNetworkMechanism(ICMPMechanismName, constructor)
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

	// Handle incoming ICMP requests.
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.packetInHandler)

	log.InfoLog("icmp/ENABLE_HOOK", "Mechanism ICMP enabled")
}

func (m *ICMPMechanism) UpdateNetwork(context *mech.NetworkContext) error {
	lldriver, err := m.C.Link.Driver()
	if err != nil {
		log.ErrorLog("icmp/UPDATE_NETWORK",
			"Link layer driver is not intialized: ", err)
		return err
	}

	// Get link layer address associated with ingress port.
	lladdr, err := lldriver.Addr(context.Port)
	if err != nil {
		log.ErrorLogf("icmp/UPDATE_NETWORK",
			"Failed to resolve port '%s' hardware address: '%s'", context.Port, err)
		return err
	}

	// Match ICMP echo-request messages to created network address.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_IPV4), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_DST, lladdr.Bytes(), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IPV4_DST, context.Addr.Bytes(), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_IP_PROTO, of.Bytes(iana.IP_PROTO_ICMP), nil},
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ICMPV4_TYPE, of.Bytes(l3.ICMPT_ECHO_REQUEST), nil},
	}}

	// Send ICMP message to the controller
	instructions := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS,
		ofp.Actions{ofp.ActionOutput{ofp.P_CONTROLLER, ofp.CML_NO_BUFFER}},
	}}

	// Insert flow into ICMP-allocated table.
	req, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:      ofp.FC_ADD,
		TableID:      ofp.Table(m.tableNo),
		BufferID:     ofp.NO_BUFFER,
		Priority:     2, // Use non-zero priority
		Match:        match,
		Instructions: instructions,
	}))

	if err != nil {
		log.ErrorLog("icmp/UPDATE_NETWORK",
			"Failed to create a new ofp_flow_mod request: ", err)
	}

	if err = m.C.Switch.Conn().Send(req); err != nil {
		log.ErrorLogf("icmp/UPDATE_NETWORK",
			"Failed to send ofp_flow_mod request:", err)
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("icmp/UPDATE_NETWORK",
			"Failed to flush requests: ", err)
	}

	return err
}

func (m *ICMPMechanism) DeleteNetwork(context *mech.NetworkContext) error {
	return nil
}

func (m *ICMPMechanism) packetInHandler(rw of.ResponseWriter, r *of.Request) {
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

	if int(packet.TableID) != m.tableNo {
		return
	}

	// Read icmp echo-request message
	icmp := l3.ICMPEcho{Data: make([]byte, pdu3.ContentLen-l3.ICMPHeaderLen)}
	if _, err = of.ReadAllFrom(r.Body, &icmp); err != nil {
		log.ErrorLog("icmp/PACKET_IN_HANDLER",
			"Failed to read ICMP message: ", err)
		return
	}

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

package ip

import (
	"net"
	"sync"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/net/l3"
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/mechutil"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
	"github.com/netrack/openflow/ofp.v13/ofputil"
)

const ARPMechanismName = "arp"

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewARPMechanism)
	mech.RegisterNetworkMechanism(ARPMechanismName, constructor)
}

// ARPMechanism handles ARP requests to the networks,
// associated with switch ports.
type ARPMechanism struct {
	mech.BaseNetworkMechanism

	// Handle request based on cookie value.
	cookies *of.CookieFilter

	// ARP table
	neighTable *mechutil.NeighTable

	// Table number allocated for the mechanism.
	tableNo int

	requests map[string][]chan bool
	lock     sync.Mutex
}

func NewARPMechanism() mech.NetworkMechanism {
	return &ARPMechanism{
		cookies:    of.NewCookieFilter(),
		requests:   make(map[string][]chan bool),
		neighTable: mechutil.NewNeighTable(),
	}
}

func (m *ARPMechanism) createRequest(nladdr mech.NetworkAddr) <-chan bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	log.DebugLog("arp/CREATE_REQUEST",
		"Create request for: ", nladdr)

	waitCh := make(chan bool)
	channels := m.requests[nladdr.String()]

	channels = append(channels, waitCh)
	m.requests[nladdr.String()] = channels

	return waitCh
}

func (m *ARPMechanism) releaseRequest(nladdr mech.NetworkAddr) {
	m.lock.Lock()
	defer m.lock.Unlock()

	log.DebugLog("arp/RELEASE_REQUEST",
		"Release requests for: ", nladdr)

	// Broadcast response to waiters
	for _, channel := range m.requests[nladdr.String()] {
		// To prevent enclosing of the variable
		ch := channel

		go func() {
			defer close(ch)
			ch <- true
		}()
	}

	delete(m.requests, nladdr.String())
}

// Enable implements Mechanism interface
func (m *ARPMechanism) Enable(c *mech.MechanismContext) {
	m.BaseMechanism.Enable(c)

	// Register resolve function by function address.
	m.C.Func.RegisterFunc((*ARPMechanism).Lookup, resolveFuncWrapper)

	// Handle incoming ARP requests.
	m.C.Mux.HandleFunc(of.T_PACKET_IN, m.packetInHandler)

	// Handle removed flows notifications
	m.C.Mux.HandleFunc(of.T_FLOW_REMOVED, m.flowRemovedHandler)

	log.InfoLog("arp/ENABLE_HOOK", "Mechanism ARP enabled")
}

// Activate implements Mechanism interface
func (m *ARPMechanism) Activate() {
	m.BaseMechanism.Activate()

	// Operate on PacketIn messages
	m.cookies.Baker = ofputil.PacketInBaker()

	// Allocate table for handling arp protocol.
	tableNo, err := m.C.Switch.AllocateTable()
	if err != nil {
		log.ErrorLog("arp/ACTIVATE_HOOK",
			"Failed to allocate a new table: ", err)
		return
	}

	m.tableNo = tableNo

	log.DebugLog("arp/ACTIVATE_HOOK",
		"Allocated table: ", tableNo)

	// Match packets of ARP protocol.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofp.OXM{ofp.XMC_OPENFLOW_BASIC, ofp.XMT_OFB_ETH_TYPE, of.Bytes(iana.ETHT_ARP), nil},
	}}

	// Move all packets to allocated matching table for ARP packets.
	instructions := ofp.Instructions{ofp.InstructionGotoTable{ofp.Table(m.tableNo)}}

	// Insert flow into 0 table.
	flowModGoto, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:      ofp.FC_ADD,
		BufferID:     ofp.NO_BUFFER,
		Priority:     20,
		Match:        match,
		Instructions: instructions,
	}))

	if err != nil {
		log.ErrorLog("arp/ACTIVATE_HOOK",
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
		log.ErrorLog("arp/ACTIVATE_HOOK",
			"Failed to send requests: ", err)
	}
}

func (m *ARPMechanism) Disable() {
	m.BaseMechanism.Disable()
}

func (m *ARPMechanism) CreateNetworkPreCommit(context *mech.NetworkContext) error {
	return m.UpdateNetworkPostCommit(context)
}

func (m *ARPMechanism) UpdateNetworkPreCommit(context *mech.NetworkContext) error {
	return m.DeleteNetworkPreCommit(context)
}

func (m *ARPMechanism) UpdateNetworkPostCommit(context *mech.NetworkContext) error {
	log.DebugLog("arp/UPDATE_NETWORK_POSTCOMMIT",
		"Got update network request")

	// Match broadcast ARP requests to resolve updated address.
	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofputil.EthType(uint16(iana.ETHT_ARP), nil),
		ofputil.ARPOpType(uint16(l3.ARPOT_REQUEST), nil),
		ofputil.ARPTargetHWAddr(l2.HWUnspec, nil),
		ofputil.ARPTargetProtoAddr(context.NetworkAddr.Bytes(), nil),
	}}

	// Send all such packets to controller
	// TODO: figure out if openflow allows flip packet fields.
	actions := ofp.Actions{
		ofp.ActionOutput{ofp.PortNo(ofp.P_CONTROLLER), ofp.CML_NO_BUFFER},
	}

	// Apply actions to packet
	instructions := ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS, actions,
	}}

	flowMod := ofp.FlowMod{
		Command: ofp.FC_ADD,
		TableID: ofp.Table(m.tableNo),
		// Notify controller, when flow removed
		Flags:    ofp.FF_SEND_FLOW_REM,
		BufferID: ofp.NO_BUFFER,
		// Use non-zero priority
		Priority:     2,
		Match:        match,
		Instructions: instructions,
	}

	// Assign cookie to FlowMod message, and
	// redirect such requests to arpRequestHandler
	m.cookies.FilterFunc(&flowMod, m.arpRequestHandler)

	// Insert flow into ARP-allocated flow table.
	arpRequest, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&flowMod))
	if err != nil {
		log.ErrorLog("arp/UPDATE_NETWORK_POSTCOMMIT",
			"Failed to send ARP request message: ", err)
		return err
	}

	// Match direct messages to receive ARP responses.
	match = ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofputil.EthType(uint16(iana.ETHT_ARP), nil),
		ofputil.EthDstAddr(context.LinkAddr.Bytes(), nil),
		ofputil.ARPOpType(uint16(l3.ARPOT_REPLY), nil),
		ofputil.ARPTargetProtoAddr(context.NetworkAddr.Bytes(), nil),
	}}

	// Send all such packets to controller
	actions = ofp.Actions{
		ofp.ActionOutput{ofp.PortNo(ofp.P_CONTROLLER), ofp.CML_NO_BUFFER},
	}

	// Apply actions to packet
	instructions = ofp.Instructions{ofp.InstructionActions{
		ofp.IT_APPLY_ACTIONS, actions,
	}}

	flowMod = ofp.FlowMod{
		Command: ofp.FC_ADD,
		TableID: ofp.Table(m.tableNo),
		// Notify controller, when flow removed
		Flags:    ofp.FF_SEND_FLOW_REM,
		BufferID: ofp.NO_BUFFER,
		// Use non-zero priority
		Priority:     3,
		Match:        match,
		Instructions: instructions,
	}

	m.cookies.FilterFunc(&flowMod, m.arpReplyHandler)

	// Insert flow into ARP-allocated flow table.
	arpReply, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&flowMod))
	if err != nil {
		log.ErrorLog("arp/UPDATE_NETWORK_POSTCOMMIT",
			"Failed to send ARP reply message: ", err)
		return err
	}

	err = of.Send(m.C.Switch.Conn(),
		arpRequest,
		arpReply,
	)

	if err != nil {
		log.ErrorLog("arp/UPDATE_NETWORK_POSTCOMMIT",
			"Failed to send requests: ", err)
	}

	return err
}

func (m *ARPMechanism) DeleteNetworkPreCommit(context *mech.NetworkContext) error {
	log.DebugLog("arp/DELETE_NETWORK_PRECOMMIT",
		"Got delete network request")

	match := ofp.Match{ofp.MT_OXM, []ofp.OXM{
		ofputil.EthType(uint16(iana.ETHT_ARP), nil),
		ofputil.ARPTargetProtoAddr(context.NetworkAddr.Bytes(), nil),
	}}

	err := of.Send(m.C.Switch.Conn(),
		ofputil.FlowFlush(ofp.Table(m.tableNo), match),
	)

	if err != nil {
		log.ErrorLog("arp/DELETE_NETWORK_PRECOMMIT",
			"Failed to remove installed ARP flows: ", err)
	}

	return err
}

func (m *ARPMechanism) packetInHandler(rw of.ResponseWriter, r *of.Request) {
	// Serve message based on PacketIn cookies.
	m.cookies.Serve(rw, r)
}

func (m *ARPMechanism) flowRemovedHandler(rw of.ResponseWriter, r *of.Request) {
	var flowRemoved ofp.FlowRemoved

	_, err := of.ReadAllFrom(r.Body, &flowRemoved)
	if err != nil {
		return
	}

	m.cookies.Release(&flowRemoved)
}

func (m *ARPMechanism) arpRequestHandler(rw of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn
	var pdu2 mech.LinkFrame
	var pdu3 l3.ARP

	log.InfoLog("arp/ARP_REQUEST_HANDLER",
		"Got ARP requets handler")

	lldriver, err := mech.LinkDrv(m.C)
	if err != nil {
		return
	}

	nldriver, err := mech.NetworkDrv(m.C)
	if err != nil {
		return
	}

	// Assume, that all packets are ARP protocol messages.
	reader := mech.MakeLinkReaderFrom(lldriver, &pdu2)
	if _, err = of.ReadAllFrom(r.Body, &packet, reader, &pdu3); err != nil {
		log.ErrorLog("arp/ARP_REQUEST_HANDLER",
			"Failed to read packet: ", err)
		return
	}

	log.DebugLog("arp/ARP_REQUEST_HANDLER",
		"Got ARP request to resolve: ", pdu3.ProtoDst)

	// Use that port as egress to send response.
	portNo := packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()

	// Get link layer address associated with egress port.
	lladdr, err := lldriver.Addr(portNo)
	if err != nil {
		log.ErrorLogf("arp/ARP_REQUEST_HANDLER",
			"Failed to resolve port '%s' hardware address: '%s'", portNo, err)
		return
	}

	// Update neighbor table with a new lladdr
	m.neighTable.Populate(mechutil.NeighEntry{
		NetworkAddr: nldriver.CreateAddr(pdu3.ProtoSrc, nil),
		LinkAddr:    pdu2.SrcAddr,
		Port:        portNo,
	})

	log.DebugLogf("arp/ARP_REQUEST_HANDLER",
		"Resolve network layer address %s -> %s", pdu3.ProtoDst, lladdr)

	// Build link layer PDU.
	pdu2 = mech.LinkFrame{pdu2.SrcAddr, lladdr, mech.Proto(iana.ETHT_ARP), 0}

	// Build ARP response message.
	pdu3 = l3.ARP{l3.ARPT_ETHERNET, iana.ETHT_IPV4, l3.ARPOT_REPLY,
		net.HardwareAddr(lladdr.Bytes()),
		pdu3.ProtoDst,
		pdu3.HWSrc,
		pdu3.ProtoSrc,
	}

	packetOut := ofp.PacketOut{BufferID: ofp.NO_BUFFER,
		InPort:  packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.PortNo(),
		Actions: ofp.Actions{ofp.ActionOutput{ofp.P_IN_PORT, 0}},
	}

	llwriter := mech.MakeLinkWriterTo(lldriver, &pdu2)
	if _, err = of.WriteAllTo(rw, &packetOut, llwriter, &pdu3); err != nil {
		log.ErrorLog("arp/ARP_REQUEST_WRITE_ERR",
			"Failed to write ARP response: ", err)

		return
	}

	rw.Header().Set(of.TypeHeaderKey, of.T_PACKET_OUT)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
	if err = rw.WriteHeader(); err != nil {
		log.ErrorLog("arp/ARP_REQUEST_SEND_ERR",
			"Failed to send ARP response: ", err)
	}
}

func (m *ARPMechanism) arpReplyHandler(rw of.ResponseWriter, r *of.Request) {
	var packet ofp.PacketIn
	var pdu2 mech.LinkFrame
	var pdu3 l3.ARP

	log.InfoLog("arp/ARP_REPLY_HANDLER",
		"Got ARP reply message")

	lldriver, err := mech.LinkDrv(m.C)
	if err != nil {
		return
	}

	nldriver, err := mech.NetworkDrv(m.C)
	if err != nil {
		return
	}

	// Assume, that all packets are ARP protocol messages.
	reader := mech.MakeLinkReaderFrom(lldriver, &pdu2)
	if _, err = of.ReadAllFrom(r.Body, &packet, reader, &pdu3); err != nil {
		log.ErrorLog("arp/ARP_REPLY_HANDLER",
			"Failed to read packet: ", err)
		return
	}

	log.DebugLogf("arp/ARP_REPLY_HANDLER",
		"Resolve network layer address %s -> %s", pdu3.ProtoSrc, pdu3.HWSrc)

	portNo := packet.Match.Field(ofp.XMT_OFB_IN_PORT).Value.UInt32()
	nladdr := nldriver.CreateAddr(pdu3.ProtoSrc, nil)

	m.neighTable.Populate(mechutil.NeighEntry{
		NetworkAddr: nladdr,
		LinkAddr:    pdu2.SrcAddr,
		Port:        portNo,
	})

	m.releaseRequest(nladdr)
}

// Wrapper of ARPMechanism.Lookup
func resolveFuncWrapper(param rpc.Param, result rpc.Result) (err error) {
	var arpMech ARPMechanism
	var nladdr mech.NetworkAddr
	var port uint32

	if err = param.Obtain(&arpMech, &nladdr, &port); err != nil {
		log.ErrorLog("arp/ARP_LOOKUP_WRAPPER",
			"Failed to obtain arguments: ", err)
		return err
	}

	var lladdr mech.LinkAddr
	if lladdr, err = arpMech.Lookup(nladdr, port); err == nil {
		return result.Return(lladdr)
	}

	return err
}

func (m *ARPMechanism) Lookup(addr mech.NetworkAddr, port uint32) (mech.LinkAddr, error) {
	log.DebugLog("arp/ARP_LOOKUP",
		"Got requests to lookup address: ", addr)

	if neigh, ok := m.neighTable.Lookup(addr); ok {
		// Success, table hit.
		return neigh.LinkAddr, nil
	}

	lldriver, err := mech.LinkDrv(m.C)
	if err != nil {
		return nil, err
	}

	nldriver, err := mech.NetworkDrv(m.C)
	if err != nil {
		return nil, err
	}

	// Get link layer address associated with egress port.
	lladdr, err := lldriver.Addr(port)
	if err != nil {
		log.ErrorLogf("arp/ARP_LOOKUP",
			"Failed to resolve port '%d' hardware address: '%s'", port, err)
		return nil, err
	}

	// Get network layer address associated with egress port.
	nladdr, err := nldriver.Addr(port)
	if err != nil {
		log.ErrorLogf("arp/ARP_LOOKUP",
			"Failed to resolve port '%d' network address: '%s'", port, err)
		return nil, err
	}

	// Start long process of discovery
	//TODO: HWType and ProtoType should return driver
	arp := l3.ARP{
		HWType:    l3.ARPT_ETHERNET,
		ProtoType: iana.ETHT_IPV4,
		Operation: l3.ARPOT_REQUEST,
		HWSrc:     lladdr.Bytes(),
		ProtoSrc:  nladdr.Bytes(),
		ProtoDst:  addr.Bytes(),
	}

	packetOut := ofp.PacketOut{
		BufferID: ofp.NO_BUFFER,
		InPort:   ofp.P_CONTROLLER,
		Actions:  ofp.Actions{ofp.ActionOutput{ofp.PortNo(port), 0}},
	}

	llbcast := lldriver.CreateAddr(l2.HWBcast)
	llwriter := mech.MakeLinkWriterTo(lldriver, &mech.LinkFrame{
		llbcast, lladdr, mech.Proto(iana.ETHT_ARP), 0,
	})

	r, err := of.NewRequest(of.T_PACKET_OUT, of.NewReader(&packetOut, llwriter, &arp))
	if err != nil {
		log.ErrorLog("arp/ARP_LOOKUP",
			"Failed to create a new ofp_packet_out request: ", err)
		return nil, err
	}

	// Create waiter for specified network address
	wait := m.createRequest(addr)

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("arp/ARP_LOOKUP",
			"Failed to send an ARP request: ", err)
		return nil, err
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("arp/ARP_LOOKUP",
			"Failed to flush data to connection: ", err)
		return nil, err
	}

	//TODO: create timeout waiter
	// Wait for response
	<-wait

	neigh, _ := m.neighTable.Lookup(addr)
	return neigh.LinkAddr, nil
}

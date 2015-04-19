package ofp13

import (
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func init() {
	constructor := mech.ExtensionMechanismConstructorFunc(NewOFPMechanism)
	mech.RegisterExtensionMechanism("ofp1.3-mechanism", constructor)
}

type OFPMechanism struct {
	mech.BaseMechanism
}

// NewOFPMechanism creates new instance of OFPMechanism type.
func NewOFPMechanism() mech.ExtensionMechanism {
	return &OFPMechanism{}
}

// Enable implements Mechanism interface.
func (m *OFPMechanism) Enable(c *mech.MechanismContext) {
	m.BaseMechanism.Enable(c)

	m.C.Mux.HandleFunc(of.T_ECHO_REQUEST, m.echoHandler)

	log.InfoLog("ofp/ENABLE_HOOK",
		"Mechanism ofp1.3 enabled")
}

// Activate implements Mechanism interface.
func (m *OFPMechanism) Activate() {
	m.BaseMechanism.Activate()

	// Write black-hole rule with the lowest priority.
	// This rule prevents flooding of the controller with
	// dumb ofp_packet_in messages.
	r, err := of.NewRequest(of.T_FLOW_MOD, of.NewReader(&ofp.FlowMod{
		Command:  ofp.FC_ADD,
		BufferID: ofp.NO_BUFFER,
		Match:    ofp.Match{ofp.MT_OXM, nil},
	}))

	if err != nil {
		log.ErrorLog("ofp1.3/ACTIVATE_HOOK",
			"Failed to create new ofp_flow_mod request: ", err)

		return
	}

	if err = m.C.Switch.Conn().Send(r); err != nil {
		log.ErrorLog("ofp1.3/ACTIVATE_HOOK",
			"Failed to write request: ", err)

		return
	}

	if err = m.C.Switch.Conn().Flush(); err != nil {
		log.ErrorLog("ofp1.3/ACTIVATE_HOOK",
			"Failed to flush request: ", err)
	}
}

func (m *OFPMechanism) echoHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)

	if err := rw.WriteHeader(); err != nil {
		log.ErrorLog("ofp1.3/ECHO_SEND_ECHO_REPLY",
			"Failed to send ofp_echo_reply: ", err)
	}
}

package ofp13

import (
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func init() {
	constructor := mech.ExtensionMechanismConstructorFunc(NewOFPMechanism)
	mech.RegisterExtensionMechanism("ofp-1.3", constructor)
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
}

func (m *OFPMechanism) echoHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)

	if err := rw.WriteHeader(); err != nil {
		log.ErrorLog("ofp1.3/ECHO_SEND_ECHO_REPLY",
			"Failed to send ofp_echo_reply: ", err)
	}
}

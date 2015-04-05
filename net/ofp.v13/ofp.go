package ofp

import (
	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func init() {
	constructor := mech.MechanismDriverConstructorFunc(NewOFPMechanism)
	mech.RegisterMechanismDriver("ofp1.3-mechanism", constructor)
}

type OFPMechanism struct {
	mech.BaseMechanismDriver
}

func NewOFPMechanism() mech.MechanismDriver {
	return &OFPMechanism{}
}

func (m *OFPMechanism) Enable(c *mech.MechanismDriverContext) {
	m.BaseMechanismDriver.Enable(c)

	m.C.Mux.HandleFunc(of.T_ECHO_REQUEST, m.echoHandler)
	log.InfoLog("ofp/ENABLE", "Mechanism ofp1.3 enabled")
}

func (m *OFPMechanism) echoHandler(rw of.ResponseWriter, r *of.Request) {
	rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)
	rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)

	if err := rw.WriteHeader(); err != nil {
		log.ErrorLog("ofp/ECHO_SEND_ECHO_REPLY",
			"Failed to send ofp_echo_reply: ", err)
	}
}

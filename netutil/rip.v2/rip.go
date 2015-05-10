package rip

import (
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
)

type RIPMechanism struct {
	mech.BaseMechanismDriver
}

func (m *RIPMech) Initialize(c *mech.MechanismDriverContext) {
}

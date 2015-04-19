package eth

import (
	"github.com/netrack/netrack/mechansim"
)

func init() {
	constructor := mech.LinkMechanismConstructorFunc(NewEthernetMechanism)
	mech.RegisterMechanism(mech.LinkProtoEthernet, constructor)
}

type EthernetMechanism struct {
	mech.BaseLinkMechanism
}

func NewEthernetMechanism() mech.LinkMechanism {
	return EthernetMechanism{}
}

func (m *EthernetMechanism) CreateLink(context *mech.LinkContext) error {
	return nil
}

func (m *EthernetMechanism) UpdateLink(context *mech.LinkContext) error {
	return nil
}

func (m *EthernetMechanism) DeleteLink(context *mech.LinkContext) error {
	return nil
}

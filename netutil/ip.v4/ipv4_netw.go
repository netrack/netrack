package ip

import (
	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/netrack/mechanism/mechutil"
)

const IPv4MechanismName = "IPv4#RFC791"

func init() {
	constructor := mech.NetworkMechanismConstructorFunc(NewIPMechanism)
	mech.RegisterNetworkMechanism(IPv4MechanismName, constructor)
}

type IPMechanism struct {
	mech.BaseNetworkMechanism
}

func NewIPMechanism() mech.NetworkMechanism {
	return &IPMechanism{}
}

// Enable implements Mechanism interface
func (m *IPMechanism) Enable(c *mech.MechanismContext) {
	m.BaseNetworkMechanism.Enable(c)

	log.InfoLog("ipv4/ENABLE_HOOK",
		"Mechanism IP enabled")
}

// Activate implements Mechanism interface
func (m *IPMechanism) Activate() {
	m.BaseNetworkMechanism.Activate()
	// pass
}

// Disable implements Mechanism interface
func (m *IPMechanism) Disable() {
	m.BaseNetworkMechanism.Disable()
	// pass
}

func (m *IPMechanism) UpdateNetwork(context *mech.NetworkContext) error {
	ipv4Routing := new(IPv4Routing)

	err := m.C.Routing.Mechanism(IPv4RoutingName, ipv4Routing)
	if err != nil {
		log.ErrorLog("ipv4/UPDATE_NETWORK",
			"IPv4 routing mechanism is not found: ", err)
		return err
	}

	// Update routing table with new address
	return ipv4Routing.UpdateRoute(&mech.RouteContext{
		Type:    string(mechutil.ConnectedRoute),
		Network: context.Addr.String(),
		Port:    context.Port,
	})
}

func (m *IPMechanism) DeleteNetwork(context *mech.NetworkContext) error {
	ipv4Routing := new(IPv4Routing)

	err := m.C.Routing.Mechanism(IPv4RoutingName, ipv4Routing)
	if err != nil {
		log.ErrorLog("ipv4/UPDATE_NETWORK",
			"IPv4 routing mechanism is not found: ", err)
		return err
	}

	// Delete route from the routing table.
	return ipv4Routing.DeleteRoute(&mech.RouteContext{
		Type:    string(mechutil.ConnectedRoute),
		Network: context.Addr.String(),
		Port:    context.Port,
	})
}

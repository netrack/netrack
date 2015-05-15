package models

// Link is a JSON representation of link layer configuration.
type Link struct {
	// Link layer encapsulation protocol (HDLC, PPP, Ethernet)
	Encapsulation nullString `json:"encapsulation"`

	// Link layer address data
	Addr nullString `json:"address"`

	// Port state
	State nullString `json:"state"`

	// Port configuration
	Config nullString `json:"config"`

	// Port features
	Features nullString `json:"features"`

	// Switch port number.
	Interface uint32 `json:"interface,omitempty"`

	// Switch port name.
	InterfaceName string `json:"interface_name,omitempty"`
}

// Network is a JSON representation of network layer configuration.
type Network struct {
	// Network layer encapsulation protocol (IPv4, IPv6, etc.)
	Encapsulation nullString `json:"encapsulation"`

	// Network layer address data
	Addr nullString `json:"address"`

	// Switch port number.
	Interface uint32 `json:"interface,omitempty"`

	// Switch port name.
	InterfaceName string `json:"interface_name,omitempty"`
}

// Route is a JSON representation of route configuration.
type Route struct {
	// Route type (static, local, rip)
	Type string `json:"type,omitempty"`

	// Next hop address
	NextHop string `json:"via"`

	// Network in a CIDR notation
	Network string `json:"network"`

	// Switch port number.
	Interface uint32 `json:"interface,omitempty"`

	// Switch port name.
	InterfaceName string `json:"interface_name"`
}

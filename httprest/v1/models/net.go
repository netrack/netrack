package models

// Link is a JSON representation of link layer configuration.
type Link struct {
	LineProtocol string `json:"line_protocol"` //DOWN, UP

	// Link layer encapsulation protocol (HDLC, PPP, Ethernet)
	Encapsulation nullString `json:"encapsulation"`

	// Link layer address data
	Addr nullString `json:"address"`
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

package models

// LinkAddr is a JSON representation of L2 address.
type LinkAddr struct {
	// L2 encapsulation protocol (HDLC, PPP, Ethernet)
	Type string `json:"type"`

	// L2 address data
	Addr string `json:"address"`
}

// NetworkAddr is a JSON representation of L3 address.
type NetworkAddr struct {
	// L3 encapsulation protocol (IPv4, IPv6, etc.)
	Type string `json:"type"`

	// L3 address data
	Addr string `json:"address"`
}

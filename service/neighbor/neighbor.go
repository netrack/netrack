package neighbor

import (
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/internal/iana"
	"golang.org/x/net/ipv6"
)

const icmp6 = "ip6:58"

type NeighborAdvertisement struct {
	Target net.IP
}

func (na *NeighborAdvertisement) Len(proto int) int {
	if na == nil {
		return 0
	}

	return 4 + len(na.Target)
}

func (na *NeighborAdvertisement) Marshal(proto int) ([]byte, error) {
	b := make([]byte, 4+len(na.Target))
	copy(b[4:], na.Target)
	return b, nil
}

type NeighborSolicitation struct {
	Target net.IP
}

func (ns *NeighborSolicitation) Len(proto int) int {
	if ns == nil {
		return 0
	}

	return 4 + len(ns.Target)
}

func (ns *NeighborSolicitation) Marshal(proto int) ([]byte, error) {
	b := make([]byte, 4+len(ns.Target))
	copy(b[4:], ns.Target)
	return b, nil
}

type neighbor struct {
	// connection list
	conns []net.PacketConn
	// interval between solicitations
	interval time.Duration
	// list of parsed IPv6 addresses
	ipaddrs []*net.IPAddr

	neigh map[string]bool

	stopCh  chan bool
	neighCh chan string
}

func New(config *Config) (*neighbor, error) {
	node := &neighbor{
		neigh:   make(map[string]bool),
		neighCh: make(chan string),
		stopCh:  make(chan bool, len(config.AdvertisementZones)+1),
	}

	raddr := net.ParseIP(config.AdvertisementGroup)
	if raddr == nil {
		text := "invalid IPv6 address"
		return nil, &net.ParseError{text, config.AdvertisementGroup}
	}

	for _, zone := range config.AdvertisementZones {
		ipaddr := &net.IPAddr{IP: raddr, Zone: zone}
		node.ipaddrs = append(node.ipaddrs, ipaddr)
	}

	return node, nil
}

func (n *neighbor) Start() {
	err := n.init()
	if err != nil {
		return
	}

	go n.cache()

	for i := range n.conns {
		go n.listen(n.conns[i], n.ipaddrs[i])
	}
}

func (n *neighbor) init() error {
	if len(n.conns) != 0 {
		return nil
	}

	var conns []net.PacketConn

	cleanup := func() {
		for _, c := range conns {
			c.Close()
		}
	}

	for _, ipaddr := range n.ipaddrs {
		conn, err := icmp.ListenPacket(icmp6, ipaddr.String())
		if err != nil {
			defer cleanup()
			return err
		}

		conns = append(conns, conn)
	}

	n.conns = conns
	return nil
}

func (n *neighbor) cache() {
	for {
		select {
		case addr := <-n.neighCh:
			n.neigh[addr] = true
		case <-n.stopCh:
			return
		}
	}
}

func (n *neighbor) listen(conn net.PacketConn, ipaddr *net.IPAddr) error {
	buf := make([]byte, 1500)

	for {
		nn, raddr, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}

		ns, err := icmp.ParseMessage(iana.ProtocolIPv6ICMP, buf[:nn])
		if err != nil {
			return err
		}

		if ns.Type != ipv6.ICMPTypeNeighborAdvertisement {
			continue
		}

		n.neighCh <- raddr.String()

		select {
		case <-n.stopCh:
			return nil
		default:
		}
	}
}

func (n *neighbor) Stop() {
	go func() {
		n.stopCh <- true
		for range n.conns {
			n.stopCh <- true
		}
	}()
}

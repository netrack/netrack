package neighbor

import (
	"bytes"
	"net"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

var neighborConfig = Config{
	AdvertisementGroup: "ff02::a",
	AdvertisementZones: []string{"eth0"},
}

type dummyConn struct {
	addr net.Addr
	rbuf bytes.Buffer
	wbuf bytes.Buffer
}

func (c *dummyConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, err := c.rbuf.Read(b)
	return n, c.addr, err
}

func (c *dummyConn) WriteTo(b []byte, a net.Addr) (int, error) {
	return c.wbuf.Write(b)
}

func (c *dummyConn) Close() error {
	return nil
}

func (c *dummyConn) LocalAddr() net.Addr {
	return c.addr
}

func (c *dummyConn) RemoteAddr() net.Addr {
	return c.addr
}

func (c *dummyConn) SetDeadline(_ time.Time) error {
	return nil
}

func (c *dummyConn) SetReadDeadline(_ time.Time) error {
	return nil
}

func (c *dummyConn) SetWriteDeadline(_ time.Time) error {
	return nil
}

func TestNeighborSolicitation(t *testing.T) {
	numconn := 20

	neighbor := &neighbor{
		neigh:   make(map[string]bool),
		neighCh: make(chan string, numconn),
		stopCh:  make(chan bool, numconn),
		conns:   make([]net.PacketConn, numconn),
		ipaddrs: make([]*net.IPAddr, numconn),
	}

	addr := net.ParseIP("fe80::1")
	b := make([]byte, 4+len(addr))
	copy(b[4:], addr)

	na := icmp.Message{}
	na.Type = ipv6.ICMPTypeNeighborAdvertisement
	na.Body = &NeighborAdvertisement{Target: addr}

	pkg, _ := na.Marshal(nil)

	for i := range neighbor.conns {
		ipaddr := &net.IPAddr{IP: addr, Zone: "en0"}
		conn := &dummyConn{addr: ipaddr, rbuf: *bytes.NewBuffer(pkg)}

		neighbor.conns[i] = conn
		neighbor.ipaddrs[i] = ipaddr
	}

	for i, c := range neighbor.conns {
		go neighbor.listen(c, neighbor.ipaddrs[i])
	}

	defer neighbor.Stop()

	for i := 0; i < numconn; i++ {
		addr := <-neighbor.neighCh
		if addr != "fe80::1%en0" {
			t.Fatal("Failed to return right address:", addr)
		}
	}
}

func TestNeighbor(t *testing.T) {
	neighbor, err := New(&neighborConfig)
	if err != nil {
		t.Fatal("Failed to create a new neighbor:", err)
	}

	err = neighbor.init()
	if err != nil {
		t.Fatal("Failed to initialize neighbor:", err)
	}

	if len(neighbor.conns) != 1 {
		t.Fatal("Failed to initialize connections:", err)
	}

	go neighbor.cache()
	neighbor.neighCh <- "fe80::13"
	neighbor.stopCh <- true

	_, ok := neighbor.neigh["fe80::13"]
	if !ok {
		t.Fatalf("Failed to store received address")
	}
}

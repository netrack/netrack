package neighbor

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"

	"github.com/netrack/netrack/storage"
)

var neighborConfig = Config{
	Group: "ff02::a",
	Zones: []string{"eth0"},
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

	var wg sync.WaitGroup
	wg.Add(1)

	trigger := storage.TriggerFunc(func(v interface{}) error {
		defer wg.Done()
		return nil
	})

	storage.Hook(storage.NeighAddrType, trigger)

	for i, c := range neighbor.conns {
		go neighbor.listen(c, neighbor.ipaddrs[i])
	}

	defer neighbor.Stop()

	wg.Wait()
}

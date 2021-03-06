package drivers

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l3"
	"github.com/netrack/netrack/mechanism"
)

const IPv4DriverName = "ipv4"

var IPv4HostMask = net.IPMask{255, 255, 255, 255}

var (
	IPv4AddrErr = errors.New(
		"ipv4: there is no network layer address associated with this port")
)

func init() {
	constructor := mech.NetworkDriverConstructorFunc(NewIPv4Driver)
	mech.RegisterNetworkDriver(IPv4DriverName, constructor)
}

type IPv4Mask struct {
	mask net.IPMask
}

func (m *IPv4Mask) Len() int {
	ones, _ := m.mask.Size()
	return ones
}

func (m *IPv4Mask) Bytes() []byte {
	return []byte(m.mask)
}

type IPv4Addr struct {
	ip   net.IP
	mask net.IPMask
}

func (a *IPv4Addr) String() string {
	ones, _ := a.mask.Size()
	return fmt.Sprintf("%s/%d", a.ip, ones)
}

func (s *IPv4Addr) Contains(nladdr mech.NetworkAddr) bool {
	network := net.IPNet{s.ip, s.mask}
	return network.Contains(net.IP(nladdr.Bytes()))
}

func (a *IPv4Addr) Bytes() []byte {
	return []byte(a.ip.To4())
}

func (a *IPv4Addr) Mask() mech.NetworkMask {
	return &IPv4Mask{a.mask}
}

type IPv4Driver struct {
	mech.BaseNetworkDriver

	// Mapping of network addresses to switch ports.
	addrs map[uint32]mech.NetworkAddr
	lock  sync.RWMutex
}

func NewIPv4Driver() mech.NetworkDriver {
	return &IPv4Driver{
		addrs: make(map[uint32]mech.NetworkAddr),
	}
}

func (d *IPv4Driver) Name() string {
	return IPv4DriverName
}

func (d *IPv4Driver) ParseAddr(s string) (mech.NetworkAddr, error) {
	ip, netw, err := net.ParseCIDR(s)
	if err != nil {
		if ip = net.ParseIP(s); ip == nil {
			return nil, err
		}

		netw = &net.IPNet{nil, IPv4HostMask}
	}

	return &IPv4Addr{ip, netw.Mask}, nil
}

func (d *IPv4Driver) CreateAddr(addr []byte, mask []byte) mech.NetworkAddr {
	if mask == nil {
		mask = IPv4HostMask
	}

	return &IPv4Addr{addr, mask}
}

func (d *IPv4Driver) Addr(port uint32) (mech.NetworkAddr, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	if addr, ok := d.addrs[port]; ok {
		return addr, nil
	}

	return nil, IPv4AddrErr
}

func (d *IPv4Driver) UpdateAddr(port uint32, addr mech.NetworkAddr) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.addrs[port] = addr
	return nil
}

func (d *IPv4Driver) DeleteAddr(port uint32) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if _, ok := d.addrs[port]; !ok {
		return IPv4AddrErr
	}

	delete(d.addrs, port)
	return nil
}

func (d *IPv4Driver) ReadPacket(r io.Reader) (*mech.NetworkPacket, error) {
	var ipv4 l3.IPv4

	_, err := ipv4.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	packet := &mech.NetworkPacket{
		DstAddr:    &IPv4Addr{ipv4.Dst, IPv4HostMask},
		SrcAddr:    &IPv4Addr{ipv4.Src, IPv4HostMask},
		Proto:      mech.Proto(ipv4.Proto),
		Len:        int64(l3.IPv4HeaderLen),
		ContentLen: int64(ipv4.Len - l3.IPv4HeaderLen),
	}

	return packet, nil
}

func (d *IPv4Driver) WritePacket(w io.Writer, p *mech.NetworkPacket) error {
	ipv4 := l3.IPv4{
		Dst:     p.DstAddr.Bytes(),
		Src:     p.SrcAddr.Bytes(),
		Proto:   iana.IPProto(p.Proto),
		Payload: p.Payload,
	}

	_, err := ipv4.WriteTo(w)
	return err
}

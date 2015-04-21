package drivers

import (
	"fmt"
	"net"

	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
)

const IPv4DriverName = "IPv4"

func init() {
	constructor := mech.NetworkDriverConstructorFunc(NewIPv4Driver)
	mech.RegisterNetworkDriver(IPv4DriverName, constructor)
}

type IPv4Addr struct {
	ip   net.IP
	mask net.IPMask
}

func (a *IPv4Addr) String() string {
	ones, _ := a.mask.Size()
	return fmt.Sprintf("%s/%d", a.ip, ones)
}

func (a *IPv4Addr) Bytes() []byte {
	return []byte(a.ip.To4())
}

func (a *IPv4Addr) Mask() []byte {
	return []byte(a.mask)
}

type IPv4Driver struct {
	mech.BaseNetworkDriver

	// Mapping of network addresses to switch ports.
	addrs map[uint32]mech.NetworkAddr
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
	ip, net, err := net.ParseCIDR(s)
	if err != nil {
		log.ErrorLog("ipv4_driver/PARSE_ADDRESS",
			"Failed to parse CIDR: ", err)
		return nil, err
	}

	return &IPv4Addr{ip, net.Mask}, nil
}

func (d *IPv4Driver) Addr(port uint32) (mech.NetworkAddr, error) {
	if addr, ok := d.addrs[port]; ok {
		return addr, nil
	}

	text := "There is no network address associated with port: '%d'"
	log.ErrorLog("ipv4_driver/ADDRESS", fmt.Sprintf(text, port))

	return nil, fmt.Errorf(text, port)
}

func (d *IPv4Driver) UpdateAddr(port uint32, addr mech.NetworkAddr) error {
	d.addrs[port] = addr
	return nil
}

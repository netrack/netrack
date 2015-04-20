package drivers

import (
	"fmt"
	"net"

	"github.com/netrack/netrack/mechanism"
)

func init() {
	constructor := mech.NetworkDriverConstructorFunc(NewIPv4Driver)
	mech.RegisterNetworkDriver("IPv4", constructor)
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
}

func NewIPv4Driver() mech.NetworkDriver {
	return &IPv4Driver{}
}

func (d *IPv4Driver) ParseAddr(s string) (mech.NetworkAddr, error) {
	ip, net, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}

	fmt.Println("Parsed addr:", ip, net)
	return &IPv4Addr{ip, net.Mask}, nil
}

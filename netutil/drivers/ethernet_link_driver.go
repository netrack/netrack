package drivers

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/netrack/mechanism"
)

const EthernetDriverName = "ieee-802.3"

func init() {
	constructor := mech.LinkDriverConstructorFunc(NewEthernetLinkDriver)
	mech.RegisterLinkDriver(EthernetDriverName, constructor)
}

// Hardware address
type EthernetAddr []byte

func (a EthernetAddr) String() string {
	addr := fmt.Sprintf("%x", []byte(a))
	var parts []string

	for i := 0; i < len(addr); i += 4 {
		parts = append(parts, string(addr[i:i+4]))
	}

	return strings.Join(parts, ".")
}

func (a EthernetAddr) Bytes() []byte {
	return []byte(a)
}

type EthernetLinkDriver struct {
	mech.BaseLinkDriver

	// Mapping of link addresses to switch ports.
	addrs map[uint32]mech.LinkAddr
}

func NewEthernetLinkDriver() mech.LinkDriver {
	return &EthernetLinkDriver{
		addrs: make(map[uint32]mech.LinkAddr),
	}
}

func (d *EthernetLinkDriver) Name() string {
	return EthernetDriverName
}

func (d *EthernetLinkDriver) CreateAddr(addr []byte) mech.LinkAddr {
	return EthernetAddr(addr)
}

func (d *EthernetLinkDriver) ParseAddr(s string) (mech.LinkAddr, error) {
	hwaddr, err := net.ParseMAC(s)
	if err != nil {
	}

	return EthernetAddr(hwaddr), nil
}

func (d *EthernetLinkDriver) UpdateAddr(port uint32, addr mech.LinkAddr) error {
	d.addrs[port] = addr
	return nil
}

func (d *EthernetLinkDriver) Addr(port uint32) (mech.LinkAddr, error) {
	if addr, ok := d.addrs[port]; ok {
		return addr, nil
	}

	text := "There is no link address associated with port: '%d'"
	return nil, fmt.Errorf(text, port)
}

func (d *EthernetLinkDriver) ReadFrame(r io.Reader) (*mech.LinkFrame, error) {
	var eth l2.EthernetII

	_, err := eth.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	length := int64(len(eth.HWSrc)+len(eth.HWDst)) + 2
	frame := &mech.LinkFrame{
		DstAddr: EthernetAddr(eth.HWDst),
		SrcAddr: EthernetAddr(eth.HWSrc),
		Proto:   mech.Proto(eth.EthType),
		Len:     length,
	}

	return frame, nil
}

func (d *EthernetLinkDriver) WriteFrame(w io.Writer, f *mech.LinkFrame) error {
	eth := l2.EthernetII{f.DstAddr.Bytes(), f.SrcAddr.Bytes(), iana.EthType(f.Proto)}
	_, err := eth.WriteTo(w)
	return err
}

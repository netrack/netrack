package drivers

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/netrack/mechanism"
)

const EthernetDriverName = "ieee-802.3"

func init() {
	constructor := mech.LinkDriverConstructorFunc(NewEthernetLinkDriver)
	mech.RegisterLinkDriver(EthernetDriverName, constructor)
}

var (
	EthernetAddrErr = errors.New(
		"ieee-802.3: there is no link layer address associated with this port")
)

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
	lock  sync.RWMutex
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

func (d *EthernetLinkDriver) DeleteAddr(port uint32) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if _, ok := d.addrs[port]; !ok {
		return EthernetAddrErr
	}

	delete(d.addrs, port)
	return nil
}

func (d *EthernetLinkDriver) ParseAddr(s string) (mech.LinkAddr, error) {
	hwaddr, err := net.ParseMAC(s)
	if err != nil {
		return nil, err
	}

	return EthernetAddr(hwaddr), nil
}

func (d *EthernetLinkDriver) UpdateAddr(port uint32, addr mech.LinkAddr) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.addrs[port] = addr
	return nil
}

func (d *EthernetLinkDriver) Addr(port uint32) (mech.LinkAddr, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	if addr, ok := d.addrs[port]; ok {
		return addr, nil
	}

	return nil, EthernetAddrErr
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

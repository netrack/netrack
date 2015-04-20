package drivers

import (
	"fmt"
	"io"
	"strings"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	constructor := mech.LinkDriverConstructorFunc(NewEthernetLinkDriver)
	mech.RegisterLinkDriver("ethernet-link-driver", constructor)
}

type EthernetFrame struct {
	frame l2.EthernetII
}

func (f *EthernetFrame) DstAddr() mech.LinkAddr {
	return EthernetAddr(f.frame.HWDst)
}

func (f *EthernetFrame) SrcAddr() mech.LinkAddr {
	return EthernetAddr(f.frame.HWSrc)
}

func (f *EthernetFrame) Proto() mech.Proto {
	return mech.Proto(f.frame.EthType)
}

func (f *EthernetFrame) Len() int64 {
	return int64(len(f.frame.HWSrc)+len(f.frame.HWDst)) + 2
}

// Hardware address
type EthernetAddr []byte

func (a EthernetAddr) String() string {
	addr := fmt.Sprintf("%x", a)
	var parts []string

	for i := 0; i < len(addr); i += 2 {
		parts = append(parts, string(addr[i:i+2]))
	}

	return strings.Join(parts, ":")
}

func (a EthernetAddr) Bytes() []byte {
	return []byte(a)
}

type EthernetLinkDriver struct {
	mech.BaseLinkDriver
}

func NewEthernetLinkDriver() mech.LinkDriver {
	return &EthernetLinkDriver{}
}

func (d *EthernetLinkDriver) ParseAddr(s string) (mech.LinkAddr, error) {
	return nil, nil
}

func (d *EthernetLinkDriver) Addr(portNo uint32) (mech.LinkAddr, error) {
	addr := append([]byte{0x00, 0x50, 0x56, 0x00, 0x00}, byte(portNo)&0xff)
	return EthernetAddr(addr), nil
}

func (d *EthernetLinkDriver) ReadFrame(r io.Reader) (mech.LinkFrame, error) {
	var eth l2.EthernetII

	_, err := eth.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	return &EthernetFrame{eth}, nil
}

func (d *EthernetLinkDriver) WriteFrame(w io.Writer, f mech.LinkFrame) error {
	eth := l2.EthernetII{f.DstAddr().Bytes(), f.SrcAddr().Bytes(), iana.EthType(f.Proto())}
	_, err := eth.WriteTo(w)
	return err
}

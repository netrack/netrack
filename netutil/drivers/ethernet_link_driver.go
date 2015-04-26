package drivers

import (
	"fmt"
	"io"
	"strings"

	"github.com/netrack/net/iana"
	"github.com/netrack/net/l2"
	"github.com/netrack/netrack/mechanism"
)

const EthernetDriverName = "ETHERNET-II#802.3"

func init() {
	constructor := mech.LinkDriverConstructorFunc(NewEthernetLinkDriver)
	mech.RegisterLinkDriver(EthernetDriverName, constructor)
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

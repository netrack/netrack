package ofp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/netrack/netrack/log"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func init() {
	constructor := mech.SwitchConstructor(NewSwitch)
	mech.RegisterSwitch("OFP/1.3", constructor)
}

// Switch handles connections with OpenFlow 1.3 switches
type Switch struct {
	// connection to openflow switch
	conn of.OFPConn

	// description of switch ports
	ports []ofp.Port

	// list of switch features
	features ofp.SwitchFeatures
}

func NewSwitch() mech.Switch {
	return &Switch{}
}

// Boot implements Switch interface
func (s *Switch) Boot(c of.OFPConn) error {
}

// Conn implements Switch interface
func (s *Switch) Conn() of.OFPConn {
	return s.conn
}

// ID implements Switch interface
func (s *Switch) ID() string {
	var b bytes.Buffer

	err := binary.Write(&b, binary.BigEndian, s.features.DatapathID)
	if err != nil {
		log.ErrorLog("switch/SWITCH_ID_ERR",
			"Failed serialize datapath identifier: ", err)

		return ""
	}

	id := fmt.Sprintf("%x", b.Bytes())
	var parts []string

	for i := 0; i < len(id); i += 2 {
		parts = append(parts, string(id[i:i+2]))
	}

	return strings.Join(parts, ":")
}

// Name implements Switch interface
func (s *Switch) Name() string {
	for _, port := range s.ports {
		if port.portNo == ofp.P_LOCAL {
			log.DebugLog("switch/SWITCH_NAME",
				"Found local port name: ", string(port.Name))

			return string(port.Name)
		}
	}

	log.DebugLog("switch/SWITCH_NAME",
		"Failed to find switch local port")

	return ""
}

// PortNameList implements Switch interface
func (s *Switch) PortNameList() []string {
	var names []string

	for _, port := range m.ports {
		if port.PortNo != ofp.P_LOCAL {
			names = append(names, string(port.Name))
		}
	}

	return names
}

// PortName implements Switch interface
func (s *Switch) PortName(p int) (string, error) {
	portNo := uint32(p)

	for _, port := range s.ports {
		if port.PortNo == portNo && port.portNo != ofp.P_LOCAL {
			log.DebugLog("switch/PORT_NAME",
				"Found port name: ", string(port.Name))

			return string(port.Name), nil
		}
	}

	log.DebugLog("switch/PORT_NAME_ERR",
		"Requested port not found: ", portNo)

	return errors.New("switch: port does not exist")
}

// PortHWAddrList implements Switch interface
func (s *Switch) PortHWAddrList() [][]byte {
	var hwaddrs [][]byte

	for _, port := range m.ports {
		hwaddrs = append(hwaddrs, []byte(port.HWAddr))
	}

	return hwaddrs
}

// PortHWAddr implements Switch interface
func (s *Switch) PortHWAddr(p int) ([]byte, error) {
	portNo := uint32(p)

	for _, port := range s.ports {
		if port.PortNo == portNo && port.portNo != ofp.P_LOCAL {
			log.DebugLog("switch/PORT_HWADDR",
				"Found port hardware address: ", port.HWAddr)

			return []byte(port.HWAddr), nil
		}
	}

	log.DebugLog("switch/PORT_HWADDR_ERR",
		"Requested port not found: ", portNo)

	return errors.New("switch: port does not exist")
}

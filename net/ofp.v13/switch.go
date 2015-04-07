package ofp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

var (
	// ErrTablesAllocate is returned when all tables are allocated.
	ErrTableAllocate = errors.New("Switch: all tables are allocated")
)

func init() {
	constructor := mech.SwitchConstructorFunc(NewSwitch)
	mech.RegisterSwitch("OFP/1.3", constructor)
}

// Switch handles connections with OpenFlow 1.3 switches
type Switch struct {
	// Connection to openflow switch.
	conn of.OFPConn

	// Description of switch ports.
	ports ofp.Ports

	// List of switch features.
	features ofp.SwitchFeatures

	// Switch tables allocation.
	tables []int

	// Lock for tables.
	lock sync.Mutex
}

// NewSwitch returns new instance of a Switch.
func NewSwitch() mech.Switch {
	return &Switch{}
}

// Boot implements Switch interface
func (s *Switch) Boot(c of.OFPConn) error {
	// Save connection instance
	s.conn = c

	// Send ofp_hello message to complete handshake.
	r, err := of.NewRequest(of.T_HELLO, nil)
	if err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to create OpenFlow request: ", err)

		return err
	}

	if err = c.Send(r); err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to send ofp_hello message:", err)

		return err
	}

	// Send ofp_features_request to retrieve datapath id.
	r, err = of.NewRequest(of.T_FEATURES_REQUEST, nil)
	if err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to create OpenFlow request: ", err)

		return err
	}

	if err = c.Send(r); err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to send ofp_features_request message:", err)

		return err
	}

	// Send ofp_multipart_request to retrieve port descriptions.
	body := of.NewReader(&ofp.MultipartRequest{Type: ofp.MP_PORT_DESC})
	r, err = of.NewRequest(of.T_MULTIPART_REQUEST, body)
	if err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to create OpenFlow request: ", err)

		return err
	}

	if err = c.Send(r); err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to send ofp_multipart_request message: ", err)

		return err
	}

	if err = c.Flush(); err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to flush requests to switch: ", err)

		return err
	}

	errCh, doneCh := make(chan error, 2), make(chan bool, 2)

	done := func() {
		doneCh <- true
	}

	echoHandler := func(rw of.ResponseWriter, r *of.Request) {
		rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
		rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)

		if err := rw.WriteHeader(); err != nil {
			log.ErrorLog("ofp/ECHO_SEND_ECHO_REPLY",
				"Failed to send ofp_echo_reply: ", err)
		}
	}

	featuresHandler := func(rw of.ResponseWriter, r *of.Request) {
		defer done()

		if _, err := s.features.ReadFrom(r.Body); err != nil {
			log.ErrorLog("ofp/FEATURES_READ_ERR",
				"Failed to read ofp_features_reply: ", err)

			errCh <- err
		}
	}

	multipartHandler := func(rw of.ResponseWriter, r *of.Request) {
		var packet ofp.MultipartReply

		defer done()

		if _, err := of.ReadAllFrom(r.Body, &packet); err != nil {
			log.ErrorLog("switch/SWITCH_BOOT_ERR",
				"Failed to read ofp_multipart_reply message: ", err)

			errCh <- err
			return
		}

		if _, err := of.ReadAllFrom(r.Body, &s.ports); err != nil {
			log.ErrorLog("switch/SWITCH_BOOT_ERR",
				"Failed to read ofp_port values: ", err)

			errCh <- err
			return
		}
	}

	mux := of.NewServeMux()
	mux.HandleFunc(of.T_FEATURES_REPLY, featuresHandler)
	mux.HandleFunc(of.T_MULTIPART_REPLY, multipartHandler)
	mux.HandleFunc(of.T_ECHO_REQUEST, echoHandler)

	exitCh := make(chan bool)

	run := func(count int, fn func() error) {
		var done int

		for {
			select {
			case <-doneCh:
				if done++; done == count {
					exitCh <- true
					return
				}

			default:
				if err := fn(); err != nil {
					errCh <- err
					return
				}
			}
		}
	}

	go run(2, func() error {
		r, err := c.Receive()
		if err != nil {
			log.ErrorLog("switch/SWITCH_BOOT_ERR",
				"Failed receive next OpenFlow message: ", err)

			return err
		}

		mux.Serve(&of.Response{Conn: c}, r)
		return nil
	})

	select {
	case err := <-errCh:
		return err

	case <-exitCh:
		return nil
	}

	return nil
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

// AllocateTable implements Switch interface.
func (s *Switch) AllocateTable() (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Make lazy intialization
	if s.tables == nil {
		for i := 0; i < int(s.features.NumTables); i++ {
			s.tables = append(s.tables, i)
		}
	}

	// The first table is reserved for protocol matching.
	if len(s.tables) < 2 {
		return 0, ErrTableAllocate
	}

	tableNo := s.tables[1]
	s.tables = append(s.tables[:1], s.tables[2:]...)

	return tableNo, nil
}

// ReleaseTable implements Switch interface.
func (s *Switch) ReleaseTable(tableNo int) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Make at least an effort to protect from bad releases.
	if len(s.tables) == int(s.features.NumTables) {
		return
	}

	s.tables = append(s.tables, tableNo)
}

// Name implements Switch interface
func (s *Switch) Name() string {
	for _, port := range s.ports {
		if port.PortNo == ofp.P_LOCAL {
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

	for _, port := range s.ports {
		if port.PortNo != ofp.P_LOCAL {
			name := strings.TrimRight(string(port.Name), "\u0000")
			names = append(names, name)
		}
	}

	return names
}

// PortHWAddrList implements Switch interface
func (s *Switch) PortHWAddrList() [][]byte {
	var hwaddrs [][]byte

	for _, port := range s.ports {
		hwaddrs = append(hwaddrs, []byte(port.HWAddr))
	}

	return hwaddrs
}

// PortNo implements Switch interface
func (s *Switch) PortNo(name string) (int, error) {
	for _, port := range s.ports {
		portName := strings.TrimRight(string(port.Name), "\u0000")
		if portName == name && port.PortNo != ofp.P_LOCAL {
			log.DebugLog("switch/PORT_NAME",
				"Found port number: ", port.PortNo)

			return int(port.PortNo), nil
		}
	}

	log.DebugLog("switch/PORT_NAME_ERR",
		"Requested port not found: ", name)

	return 0, errors.New("switch: port does not exist")
}

// PortHWAddr implements Switch interface
func (s *Switch) PortHWAddr(p int) ([]byte, error) {
	portNo := ofp.PortNo(p)

	for _, port := range s.ports {
		if port.PortNo == portNo && port.PortNo != ofp.P_LOCAL {
			log.DebugLog("switch/PORT_HWADDR",
				"Found port hardware address: ", port.HWAddr)

			return []byte(port.HWAddr), nil
		}
	}

	log.DebugLog("switch/PORT_HWADDR_ERR",
		"Requested port not found: ", portNo)

	return nil, errors.New("switch: port does not exist")
}

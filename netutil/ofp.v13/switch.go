package ofp13

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
	"github.com/netrack/openflow/ofp.v13/ofputil"
)

var (
	// ErrTableAllocate is returned when all tables are allocated.
	ErrTableAllocate = errors.New("Switch: all tables are allocated")
)

func init() {
	constructor := mech.SwitchConstructorFunc(NewSwitch)
	mech.RegisterSwitch("OFP/1.3", constructor)
}

func SwitchPort(port ofp.Port) *mech.SwitchPort {
	return &mech.SwitchPort{
		Name:     strings.TrimRight(string(port.Name), "\u0000"),
		Number:   uint32(port.PortNo),
		Config:   port.Config.String(),
		State:    port.State.String(),
		Features: port.Curr.String(),
	}
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
	ofpHello, err := of.NewRequest(of.T_HELLO, nil)
	if err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to create OpenFlow Hello request: ", err)
		return err
	}

	// Send ofp_features_request to retrieve datapath id.
	ofpFeatures, err := of.NewRequest(of.T_FEATURES_REQUEST, nil)
	if err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to create OpenFlow Features request: ", err)
		return err
	}

	// Send ofp_multipart_request to retrieve port descriptions.
	body := of.NewReader(&ofp.MultipartRequest{Type: ofp.MP_PORT_DESC})
	ofpMultipart, err := of.NewRequest(of.T_MULTIPART_REQUEST, body)
	if err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_ERR",
			"Failed to create OpenFlow Port Description request: ", err)
		return err
	}

	err = of.Send(c,
		ofpHello,
		ofpFeatures,
		ofpMultipart,
		// Clear 0 table first
		ofputil.TableFlush(0),
		// Write black-hole rule with the lowest priority.
		// This rule prevents flooding of the controller with
		// dumb ofp_packet_in messages.
		ofputil.FlowDrop(0),
	)

	if err != nil {
		log.ErrorLog("switch/SWITCH_BOOT_SEND_ERR",
			"Failed to send handshake messages: ", err)
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
func (s *Switch) Name() (name string) {
	s.PortIter(func(port *mech.SwitchPort) (ok bool) {
		if ok = port.Number != uint32(ofp.P_LOCAL); !ok {
			log.DebugLog("switch/SWITCH_NAME",
				"Found local port name: ", port.Name)

			name = port.Name
		}

		return
	})

	if name == "" {
		log.ErrorLog("switch/SWITCH_NAME",
			"Failed to find switch local port")
	}

	return
}

// PortIter calls specified function for all registered ports.
func (s *Switch) PortIter(fn func(*mech.SwitchPort) bool) {
	for _, port := range s.ports {
		if !fn(SwitchPort(port)) {
			return
		}
	}
}

// PortList implements Switch interface
func (s *Switch) PortList() []*mech.SwitchPort {
	var ports []*mech.SwitchPort

	s.PortIter(func(port *mech.SwitchPort) bool {
		if port.Number != uint32(ofp.P_LOCAL) {
			ports = append(ports, port)
		}

		return true
	})

	return ports
}

// PortByName implements Switch interface
func (s *Switch) PortByName(name string) (p *mech.SwitchPort, err error) {
	err = errors.New("switch: port does not exist")

	s.PortIter(func(port *mech.SwitchPort) (ok bool) {
		if ok = port.Name != name; !ok {
			p, err = port, nil

			log.DebugLog("switch/PORT_BY_NAME",
				"Found port by name: ", name)
		}

		return
	})

	if err != nil {
		log.ErrorLog("switch/SWITCH_BY_NAME",
			"Failed to find switch by name: ", name)
	}

	return
}

// PortByNumber implements Switch interface
func (s *Switch) PortByNumber(number uint32) (p *mech.SwitchPort, err error) {
	err = errors.New("switch: port does not exist")

	s.PortIter(func(port *mech.SwitchPort) (ok bool) {
		if ok = port.Number != number; !ok {
			p, err = port, nil

			log.DebugLog("switch/PORT_BY_NUMBER",
				"Found port by number: ", number)
		}

		return
	})

	if err != nil {
		log.ErrorLog("switch/SWITCH_BY_NUMBER",
			"Failed to find switch by number: ", number)
	}

	return
}

package mech

import (
	"errors"
	"sync"

	"github.com/netrack/netrack/logging"
	"github.com/netrack/netrack/mechanism/rpc"
	"github.com/netrack/openflow"
)

var (
	// ErrSwitchNotFound is returned when switch is not managed by SwitchManager.
	ErrSwitchNotFound = errors.New("SwitchManager: switch not found")
)

// SwitchManager manages switch connections and mechanism
// drivers associated with each switch.
type SwitchManager struct {
	// List of serving switches
	entries map[string]*MechanismContext

	// Lock for entries list
	lock sync.RWMutex
}

// CreateSwitch searches available switch implementation
// for requested version and initialize switch boot process.
func (m *SwitchManager) CreateSwitch(conn of.OFPConn) error {
	// Make lazy intialization
	if m.entries == nil {
		m.entries = make(map[string]*MechanismContext)
	}

	// Read ofp_hello message to get protocol version
	r, err := conn.Receive()
	if err != nil {
		log.ErrorLog("switch_manager/CREATE_SWITCH",
			"Failed to read ofp_hello message from connection: ", err)
		return err
	}

	constructor := SwitchByVersion(r.Proto)
	if constructor == nil {
		log.ErrorLog("switch_manager/CREATE_SWITCH",
			"Unknown protocol version: ", r.Proto)

		return errors.New("SwitchManager: Unknown protocol version")
	}

	// Create a new switch instance
	sw := constructor.New()

	log.DebugLog("switch_manager/CREATE_SWITCH",
		"Booting switch...")

	// Wait for switch boot up
	if err = sw.Boot(conn); err != nil {
		log.ErrorLog("switch_manager/CREATE_SWITCH",
			"Failed to boot switch: ", err)
		return err
	}

	log.DebugLog("switch_manager/CREATE_SWITCH",
		"Switch successfully booted for ", r.Proto)

	// FIXME: should be configured through REST api.
	var ldrv LinkDriver
	for _, driver := range linkDrivers {
		ldrv = driver.New()
		break
	}

	// Create mechanism managers
	linkManager := &BaseLinkMechanismManager{
		BaseMechanismManager{LinkMechanisms()}, ldrv,
	}

	// FIXME: should be configured through REST api.
	var ndrv NetworkDriver
	for _, driver := range networkDrivers {
		ndrv = driver.New()
		break
	}

	networkManager := &BaseNetworkMechanismManager{
		BaseMechanismManager{NetworkMechanisms()}, ndrv,
	}

	extensionManager := &ExtensionMechanismManager{
		BaseMechanismManager{ExtensionMechanisms()},
	}

	// Create a new mechanism driver context
	context := &MechanismContext{
		Switch:    sw,
		Func:      rpc.New(),
		Mux:       of.NewServeMux(),
		Link:      linkManager,
		Network:   networkManager,
		Extension: extensionManager,
	}

	linkManager.Enable(context)
	networkManager.Enable(context)
	extensionManager.Enable(context)

	// Since switch already booted, activate drivers
	// TODO: make this configurable (or deactivate all by default)
	linkManager.Activate()
	networkManager.Activate()
	extensionManager.Activate()

	log.DebugLog("switch_manager/CREATE_SWITCH",
		"Switch successfully created")

	m.lock.Lock()
	defer m.lock.Unlock()

	m.entries[context.Switch.ID()] = context

	// Serve can delete context from entries list,
	// so call it after adding context to entries list.
	go m.serve(context)

	return nil
}

// SwitchContext returns switch context of managing switch,
// ErrSwitchNotFound returned when switch is not managed by SwitchManager.
func (m *SwitchManager) Context(dpid string) (*MechanismContext, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	context, ok := m.entries[dpid]
	if !ok {
		log.DebugLog("switch_manager/SWITCH_CONTEXT_BY_ID",
			"Failed to find switch for: ", dpid)

		return nil, ErrSwitchNotFound
	}

	log.DebugLog("switch_manager/SWITCH_CONTEXT_BY_ID",
		"Found switch context for: ", dpid)

	return context, nil
}

func (m *SwitchManager) serve(c *MechanismContext) {
	conn := c.Switch.Conn()

	for {
		r, err := conn.Receive()
		if err != nil {
			log.ErrorLog("switch_manager/SWITCH_SERVE_ERR",
				"Failed to receive next OpenFlow message: ", err)

			m.lock.Lock()
			defer m.lock.Unlock()

			delete(m.entries, c.Switch.ID())

			log.InfoLogf("switch_manager/SWITCH_SERVE",
				"Switch %s deleted", c.Switch.ID())

			return
		}

		go c.Mux.Serve(&of.Response{Conn: conn}, r)
	}
}

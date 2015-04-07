package httprest

import (
	"net"
	"net/http"

	"github.com/netrack/netrack/httprest/format"
	"github.com/netrack/netrack/httprest/v1/models"
	"github.com/netrack/netrack/httputil"
	"github.com/netrack/netrack/log"
	"github.com/netrack/netrack/mechanism"
)

func init() {
	// Register address management HTTP API driver.
	constructor := mech.HTTPDriverConstructorFunc(NewAddressMgmt)
	mech.RegisterHTTPDriver(constructor)
}

// IP protocol address management
type AddressMgmt struct {
	// Base HTTP driver instance.
	mech.BaseHTTPDriver
}

// NewAddressMgmt creates a new instance of AddressMgmt type.
func NewAddressMgmt() mech.HTTPDriver {
	return &AddressMgmt{}
}

// Enable implements HTTPDriver interface.
func (m *AddressMgmt) Enable(c *mech.HTTPDriverContext) {
	m.BaseHTTPDriver.Enable(c)

	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/interfaces/ipv4/address", m.indexHandler)
	m.C.Mux.HandleFunc("GET", "/v1/switches/{dpid}/interfaces/{interface}/ipv4/address", m.showHandler)
	m.C.Mux.HandleFunc("PUT", "/v1/switches/{dpid}/interfaces/{interface}/ipv4/address", m.createHandler)
	m.C.Mux.HandleFunc("DELETE", "/v1/switches/{dpid}/interfaces/{interface}/ipv4/address", m.destroyHandler)

	log.InfoLog("address_mgmt/ENABLE_HOOK",
		"IP address management enabled")
}

func (m *AddressMgmt) indexHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *AddressMgmt) createHandler(rw http.ResponseWriter, r *http.Request) {
}

func (m *AddressMgmt) showHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("address_mgmt/SHOW_HANDLER",
		"Got request to show IPv4 address")

	f, err := format.Format(r.Header.Get(httputil.HeaderAccept))
	if err != nil {
		log.ErrorLog("address_mgmt/SHOW_HANDLER",
			"Failed to select Accept formatter for request: ", err)

		formats := strings.Join(format.FormatNameList(), ", ")

		f.Write(rw, r, models.Error{fmt.Sprintf("only '%s' are supported", formats)})
		rw.WriteHeader(http.StatusNotAcceptable)
		return
	}

	dpid := httputil.Param(r, "dpid")
	iface := httputil.Param(r, "interface")

	log.DebugLogf("address_mgmt/SHOW_HANDLER",
		"Request show IPv4 address of: %s dev %s", dpid.iface)

	c, err := m.C.SwitchManager.SwitchContextByID(dpid)
	if err != nil {
		log.ErrorLog("address_mgmt/SHOW_HANDLER",
			"Failed to find requested datapath: ", err)

		f.Write(rw, r, models.Error{fmt.Sprintf("switch '%s' not found", dpid)})
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	portNo, err := c.Context.Switch.PortNo(iface)
	if err != nil {
		log.ErrorLog("address_mgmt/SHOW_HANDER",
			"Failed to find requested interface: ", iface)

		text := fmt.Sprintf("switch '%s' do not have '%s' interface", dpid, iface)
		f.Write(rw, r, models.Error{text})
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	var addr, mask []byte

	// Retrieve IPv4 address and network mask assigned to the port.
	err = c.Context.Func.Call(rpc.IPv4GetAddressFunc,
		rpc.Uint32Param(portNo),
		rpc.CompositeResult(
			rpc.ByteSliceResult(&ip),
			rpc.ByteSliceResult(&mask),
		))

	if err != nil {
		log.ErrorLog("address_mgmt/SHOW_HANDLER",
			"Failed to find IPv4 address of the interface: ", dpid)

		text := fmt.Sprintf("IPv4 address is not assigned to '%s' interface", iface)
		f.Write(rw, r, models.Error{text})
		rw.WriteHeader(http.StatusConflict)
	}

	ipaddr, ipmask := net.IP(addr), net.IPMask(mask)
	bits, _ := ipmask.Size()

	// Return IPv4 address in a CIDR notation
	f.Write(rw, r, fmt.Sprintf("%s/%d", ipaddr, bits))
	rw.WriteHeader(http.StatusOK)
}

func (m *AddressMgmt) destroyHandler(rw http.ResponseWriter, r *http.Request) {
	log.InfoLog("address_mgmt/DESTROY_HANDLER",
		"Got request to destroy IPv4 address")

	dpid := httputil.Param(r, "dpid")
	iface := httputil.Param(r, "interface")

	log.DebugLog("address_mgmt/DESTROY_HANDLER",
		"Request delete IPv4 address of: %s dev %s", dpid, iface)

	c, err := m.C.SwitchManager.SwitchContextByID(dpid)
	if err != nil {
		log.ErrorLog("address_mgmt/DESTROY_HANDLER",
			"Failed to find requested datapath: ", err)

		f.Write(rw, r, models.Error{fmt.Sprintf("switch '%s' not found", dpid)})
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	portNo, err := c.Context.Switch.PortNo(iface)
	if err != nil {
		log.ErrorLog("address_mgmt/DESTROY_HANDER",
			"Failed to find requested interface: ", iface)

		text := fmt.Sprintf("switch '%s' do not have '%s' interface", dpid, iface)
		f.Write(rw, r, models.Error{text})
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// Delete IPv4 address from the interface.
	err = c.Context.Func.Call(rpc.IPv4DeleteAddressFunc,
		rpc.Uint32Param(uint32(portNo)), nil)

	if err != nil {
		log.ErrorLog("address_mgmt/DESTROY_HANDLER",
			"Failed to delete IPv4 address from the interface: ", iface)

		text := fmt.Sprintf("IPv4 address was not deleted from '%s' interface", iface)
		f.Write(rw, r, models.Error{text})
		rw.WriteHeader(http.StatusConflict)
	}

	rw.WriteHeader(http.StatusOK)
}

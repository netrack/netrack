package main

import (
	"net"

	"github.com/netrack/netrack/daemon/ip.v4"
	"github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp.v13"
)

func main() {
	l3addr := net.IP{10, 0, 0, 254}
	l2addr := net.HardwareAddr{2, 0, 0, 0, 0, 254}

	icmpd := ip.ICMPServer{l2addr, l3addr}
	arpd := ip.ARPServer{l2addr, l3addr}

	router := ip.Router{}

	of.HandleFunc(of.T_HELLO, func(rw of.ResponseWriter, r *of.Request) {
		rw.Header().Set(of.TypeHeaderKey, of.T_HELLO)
		rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
		rw.WriteHeader()
	})

	of.HandleFunc(of.T_ECHO_REQUEST, func(rw of.ResponseWriter, r *of.Request) {
		rw.Header().Set(of.TypeHeaderKey, of.T_ECHO_REPLY)
		rw.Header().Set(of.VersionHeaderKey, ofp.VERSION)
		rw.WriteHeader()
	})

	of.HandleFunc(of.T_HELLO, router.Hello)
	of.HandleFunc(of.T_HELLO, arpd.Hello)
	of.HandleFunc(of.T_HELLO, icmpd.Hello)

	//of.HandleFunc(of.T_PACKET_IN, icmpd.PacketIn)
	of.HandleFunc(of.T_PACKET_IN, arpd.PacketIn)

	of.ListenAndServe()
}

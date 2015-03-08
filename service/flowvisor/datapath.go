package flowvisor

import (
	"net"

	of "github.com/netrack/openflow"
)

type Datapath struct {
	ID   string
	conn *of.Conn
}

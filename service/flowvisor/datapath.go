package flowvisor

import (
	"net"
)

type Datapath struct {
	conn net.Conn
}

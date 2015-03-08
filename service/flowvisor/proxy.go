package flowvisor

import (
	"net"

	"github.com/netrack/netrack/service/redis"
	"github.com/netrack/openflow"
)

type proxy struct {
	dps  map[string]Datapath
	addr *net.TCPAddr

	stopCh chan bool
}

func NewProxy(config *Config) (*proxy, error) {
	addr := fmt.Sprintf("%s:%d",
		config.ControllerAddress,
		config.ControllerPort)

	tcpaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	p = &proxy{
		dps:    make(map[string]Datapath),
		stopCh: make(chan bool),
		addr:   tcpaddr,
	}

	return p, nil
}

func (p *proxy) accept() (chan<- *of.Conn, chan<- error) {
	cchan := make(chan *of.Conn)
	echan := make(chan error)

	go func() {
		oflistener, err := of.Listen("tcp", p.addr.String())
		if err != nil {
			echan <- err
			return
		}

		for {
			ofconn, err := oflistener.Accept()
			if err != nil {
				echan <- err
				return
			}

			cchan <- ofconn
		}
	}()

	return cchan, echan
}

func (p *proxy) serve(ofconn *of.Conn) {
	cchan, echan := p.acceptAsync()

	select {
	case ofconn := <-cchan:
		go p.handle(ofconn)
	case <-echan:
		return
	}
}

func (p *proxy) handle(ofconn *of.Conn) {
}

func echoHandler(ofconn *of.Conn, r *of.Request) error {
	req, err := of.NewRequest(of.T_ECHO_REPLY, nil)
	if err != nil {
		return err
	}

	return ofconn.Send(req)
}

func helloHandler(ofconn *of.Conn, r *of.Request) error {
	req, err := of.NewRequest(of.T_HELLO, nil)
	if err != nil {
		return err
	}

	return ofconn.Send(req)
}

func defaultHandle(ofcinn *of.Conn, r *of.Request) error {
	// send message to actual controller
}

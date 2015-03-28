package rpc

import (
	"errors"
	"net"
)

var (
	ErrEmptyReturn = errors.New("rpc: return list is empty")
)

func HWAddr(v interface{}, err error) (net.HardwareAddr, error) {
	if err != nil {
		return nil, err
	}

	addr, ok := v.(net.HardwareAddr)
	if !ok {
		return nil, errors.New("rpc: unexpected type for net.HardwareAddr")
	}

	return addr, nil
}

func IPAddr(v interface{}, err error) (net.IP, error) {
	if err != nil {
		return nil, err
	}

	addr, ok := v.(net.IP)
	if !ok {
		return nil, errors.New("rpc: unexpected type for net.IPAddr")
	}

	return addr, nil
}

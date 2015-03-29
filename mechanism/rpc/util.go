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

func StringSlice(v interface{}, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}

	s, ok := v.([]string)
	if !ok {
		return nil, errors.New("rpc: unexpected type for []string")
	}

	return s, nil
}

func String(v interface{}, err error) (string, error) {
	if err != nil {
		return "", err
	}

	s, ok := v.(string)
	if !ok {
		return "", errors.New("rpc: unexpected type for string")
	}

	return s, nil
}

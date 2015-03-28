package rpc

import (
	"errors"
	"sync"
)

const (
	T_ARP_RESOLVE Type = iota
	T_IPV4_ADD_ROUTE
	T_IPV4_DELETE_ROUTE
)

type Type int

type Caller interface {
	Call(param interface{}) (interface{}, error)
}

type CallerFunc func(param interface{}) (interface{}, error)

func (fn CallerFunc) Call(param interface{}) (interface{}, error) {
	return fn(param)
}

type RPCaller interface {
	Register(Type, Caller) error
	RegisterFunc(Type, func(interface{}) (interface{}, error)) error
	Call(Type, interface{}) (interface{}, error)
}

func New() RPCaller {
	return &rpCaller{methods: make(map[Type]Caller)}
}

type rpCaller struct {
	methods map[Type]Caller
	lock    sync.RWMutex
}

func (r *rpCaller) Register(t Type, c Caller) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, dup := r.methods[t]; dup {
		return errors.New("rpc: multiple registrations")
	}

	if c == nil {
		return errors.New("rpc: nil caller")
	}

	r.methods[t] = c
	return nil
}

func (r *rpCaller) RegisterFunc(t Type, fn func(interface{}) (interface{}, error)) error {
	return r.Register(t, CallerFunc(fn))
}

func (r *rpCaller) Call(t Type, param interface{}) (interface{}, error) {
	caller, ok := r.methods[t]
	if !ok {
		return nil, errors.New("rpc: caller not registered")
	}

	return caller.Call(param)
}

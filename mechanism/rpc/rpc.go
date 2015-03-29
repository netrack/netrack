package rpc

import (
	"errors"
	"sync"
)

const (
	T_ARP_RESOLVE Type = iota
	T_IPV4_ADD_ROUTE
	T_IPV4_DELETE_ROUTE
	T_DATAPATH_PORTS
	T_DATAPATH_ID
	T_DATAPATH
)

type Type int

type Param interface {
	Obtain(...interface{}) error
}

type Result interface {
	Return(...interface{}) error
}

type ParamFunc func(...interface{}) error

func (fn ParamFunc) Obtain(args ...interface{}) error {
	return fn(args...)
}

type ResultFunc func(...interface{}) error

func (fn ResultFunc) Return(args ...interface{}) error {
	return fn(args...)
}

type Caller interface {
	Call(param Param, result Result) error
}

type CallerFunc func(param Param, result Result) error

func (fn CallerFunc) Call(param Param, result Result) error {
	return fn(param, result)
}

type ProcCaller interface {
	Register(Type, Caller) error
	RegisterFunc(Type, CallerFunc) error
	Call(Type, Param, Result) error
}

func New() ProcCaller {
	return &procCaller{methods: make(map[Type]Caller)}
}

type procCaller struct {
	methods map[Type]Caller
	lock    sync.RWMutex
}

func (c *procCaller) Register(t Type, caller Caller) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if _, dup := c.methods[t]; dup {
		return errors.New("rpc: multiple registrations")
	}

	if caller == nil {
		return errors.New("rpc: nil caller")
	}

	c.methods[t] = caller
	return nil
}

func (c *procCaller) RegisterFunc(t Type, fn CallerFunc) error {
	return c.Register(t, CallerFunc(fn))
}

func (c *procCaller) Call(t Type, param Param, result Result) error {
	caller, ok := c.methods[t]
	if !ok {
		return errors.New("rpc: caller not registered")
	}

	return caller.Call(param, result)
}

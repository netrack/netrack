package rpc

import (
	"errors"
	"sync"
)

// Type represents name of calling function.
type Type string

// Param describes types that could be passed as
// parameters in a function call method.
type Param interface {
	// Obtain returns passed parameters.
	Obtain(...interface{}) error
}

// Result describes types that could be returned
// as a result of function call.
type Result interface {
	// Return returns function result.
	Return(...interface{}) error
}

// ParamFunc is a function adapter for Param interface.
type ParamFunc func(...interface{}) error

// Obtain implements Param interface.
func (fn ParamFunc) Obtain(args ...interface{}) error {
	return fn(args...)
}

// ResultFunc is a function adapter for Result interface.
type ResultFunc func(...interface{}) error

// ResultFunc implements Result interface.
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

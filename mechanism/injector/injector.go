package injector

import (
	"errors"
	"reflect"
	"sync"
)

var (
	ErrNoType = errors.New("injector: trying to access unbound type")
)

type TypeReflect map[reflect.Type]reflect.Value

func TypeOf(iface interface{}) (t reflect.Type) {
	t = reflect.TypeOf(iface)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t
}

type Injector interface {
	Bind(typ interface{}, impl interface{})
	Unbind(typ interface{})
	Obtain(ptr interface{}) error
}

type injector struct {
	bindings TypeReflect
	lock     sync.RWMutex
}

func New() Injector {
	return &injector{bindings: make(TypeReflect)}
}

func (i *injector) Bind(typ interface{}, impl interface{}) {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.bindings[TypeOf(typ)] = reflect.ValueOf(impl)
}

func (i *injector) Unbind(typ interface{}) {
	i.lock.Lock()
	defer i.lock.Unlock()

	delete(i.bindings, reflect.TypeOf(typ))
}

func (i *injector) Obtain(ptr interface{}) error {
	i.lock.RLock()
	defer i.lock.RUnlock()

	value, ok := i.bindings[TypeOf(ptr)]
	if !ok {
		return ErrNoType
	}

	reflect.ValueOf(ptr).Elem().Set(value)
	return nil
}

package eventlet

import (
	"errors"
)

var (
	ErrTypeLoad = errors.New("eventlet: Load Type is wrong")
)

type Type string

type Loader interface {
	Load(interface{}) error
}

type LoaderFunc func(interface{}) error

func (fn LoaderFunc) Load(v interface{}) error {
	return fn(v)
}

type Event interface {
	Type() Type
	LoadTo(Loader) error
}

type StringEvent struct {
	EventType Type
	Body      string
}

func (e StringEvent) Type() Type {
	return e.EventType
}

func (e StringEvent) LoadTo(l Loader) error {
	return l.Load(e.Body)
}

func StringLoader(s *string) Loader {
	return LoaderFunc(func(v interface{}) error {
		str, ok := v.(string)
		if !ok {
			return ErrTypeLoad
		}

		*s = str
		return nil
	})
}

func IntLoader(i *int) Loader {
	return LoaderFunc(func(v interface{}) error {
		n, ok := v.(int)
		if !ok {
			return ErrTypeLoad
		}

		*i = n
		return nil
	})
}

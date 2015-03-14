package storage

import (
	"errors"
	"io"
)

var (
	ErrTypeNotDeclared = errors.New("storage: Type is not declared")
	ErrTypeLoad        = errors.New("storage: Load Type is wrong")
)

type Loader interface {
	Load(interface{}) error
}

type LoaderFunc func(interface{}) error

func (fn LoaderFunc) Load(v interface{}) error {
	return fn(v)
}

type Trigger interface {
	Trigger(interface{}) error
}

type TriggerFunc func(interface{}) error

func (fn TriggerFunc) Trigger(v interface{}) error {
	return fn(v)
}

type Persister interface {
	io.Closer

	// Add the specified value to the set stored at Type. Specified values
	// that are already a member of this set are ignored. If Type does not exist,
	// an error will be returned.
	Put(Type, interface{}) error

	// Returns all the values of the set stored at Type. If Type does not exists,
	// an error will be returned.
	Get(Type, Loader) error

	// Subscribes the client to the specified Type. If Type does not exists,
	// an error will be returned.
	Hook(Type, Trigger) error
}

var types = make(map[Type]bool)

// Declare makes a new type available.
func Declare(t Type) {
	types[t] = true
}

type Result interface {
}

type Elements interface {
	Next() bool
	LoadTo(Loader) error
}

type Statement interface {
	io.Closer

	Exec(args ...interface{}) (Result, error)
	Query(args ...interface{}) (Elements, error)
}

type stmt struct {
	putStmt  Statement
	getStmt  Statement
	hookStmt Statement
}

func (s *stmt) Close() (err error) {
	err = s.putStmt.Close()
	if err != nil {
		return
	}

	err = s.getStmt.Close()
	if err != nil {
		return
	}

	return s.hookStmt.Close()
}

type persister struct {
	stmts map[Type]*stmt
}

func NewPersister() *persister {
	return &persister{prepareStmt()}
}

func (p *persister) recordToStmt(r Type) (*stmt, error) {
	s, ok := p.stmts[r]
	if !ok {
		return nil, ErrTypeNotDeclared
	}

	return s, nil
}

func (p *persister) Put(r Type, v interface{}) error {
	stmt, err := p.recordToStmt(r)
	if err != nil {
		return err
	}

	_, err = stmt.putStmt.Exec(r, v)
	return err
}

func (p *persister) Get(r Type, l Loader) error {
	stmt, err := p.recordToStmt(r)
	if err != nil {
		return err
	}

	elements, err := stmt.getStmt.Query(r)
	for elements.Next() {
		err := elements.LoadTo(l)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *persister) Hook(r Type, t Trigger) error {
	stmt, err := p.recordToStmt(r)
	if err != nil {
		return err
	}

	_, err = stmt.hookStmt.Exec(r, t)
	return err
}

func (p *persister) Close() (e error) {
	for _, stmt := range p.stmts {
		err := stmt.Close()
		if err != nil {
			e = err
		}
	}

	return
}

var DefaultPersister = NewPersister()

func Put(r Type, v interface{}) error {
	return DefaultPersister.Put(r, v)
}

func Get(r Type, l Loader) error {
	return DefaultPersister.Get(r, l)
}

func Hook(r Type, t Trigger) error {
	return DefaultPersister.Hook(r, t)
}

func Close() error {
	return DefaultPersister.Close()
}

func StringSliceLoader(s *[]string) Loader {
	*s = []string{}

	return LoaderFunc(func(v interface{}) error {
		str, ok := v.(string)
		if !ok {
			return ErrTypeLoad
		}

		*s = append(*s, str)
		return nil
	})
}

func IntSliceLoader(s *[]int) Loader {
	*s = []int{}

	return LoaderFunc(func(v interface{}) error {
		n, ok := v.(int)
		if !ok {
			return ErrTypeLoad
		}

		*s = append(*s, n)
		return nil
	})
}

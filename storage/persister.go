package storage

import (
	"errors"
	"io"
)

type Result interface {
}

type Elements interface {
	Next() bool
	LoadTo(Loader) error
}

type Stmt interface {
	io.Closer

	Exec(args ...interface{}) (Result, error)
	Query(args ...interface{}) (Elements, error)
}

type Loader interface {
	Load(interface{}) error
}

type Dumper interface {
	Dump() (interface{}, error)
}

type LoaderFunc func(interface{}) error

func (fn LoaderFunc) Scan(v interface{}) error {
	return fn(v)
}

type Persister interface {
	io.Closer

	Put(Type, Dumper) error
	Get(Type, Loader) error

	Hook(Type) (chan<- bool, error)
}

type stmt struct {
	putStmt  Stmt
	getStmt  Stmt
	hookStmt Stmt
}

type perister struct {
	stmts map[Type]*stmt
}

func (p *persister) recordToStmt(r Type) (*stmt, error) {
	s, ok := p.stmts[r]
	if !ok {
		return nil, fmt.Errorf("storage: record is not present")
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

func (p *persister) Get(r Type, s Loader) error {
	stmt, err := p.recordToStmt(r)
	if err != nil {
		return err
	}

	elements, err := stmt.getStmt.Query(r, v)
	for elements.Next() {
		err := elements.ScanTo(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *persister) Hoot(r Type) error {
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

func Open(config *Config) (Persister, error) {
	stmts, err := prepareStmt(config)
	if err != nil {
		return nil, err
	}

	return &persister{stmts}, nil
}

var DefaultPersister Persister

func Put(r Type, v interface{}) error {
	return DefaultPersister.Put(r, v)
}

func Get(r Type, s Loader) error {
	return DefaultPersister.Get(r, s)
}

func Close() error {
	return DefaultPersister.Close()
}

var ErrType = errors.New("storage: wrong type")

func StringSliceLoader(s []string) Loader {
	return LoaderFunc(func(v interface{}) error {
		str, ok := v.(string)
		if !ok {
			return ErrType
		}

		append(s, str)
		return nil
	})
}

func IntSliceLoader(s []int) Loader {
	return LoaderFunc(func(v interface{}) {
		n, ok := v.(int)
		if !ok {
			return ErrType
		}

		append(s, n)
		return nil
	})
}

package storage

import (
	"errors"
	"sync"
)

type dsValue struct {
	records   map[interface{}]bool
	recordsMu sync.RWMutex

	triggers   []Trigger
	triggersMu sync.RWMutex
}

func (v *dsValue) hook(t Trigger) {
	v.triggersMu.Lock()
	defer v.triggersMu.Unlock()
	v.triggers = append(v.triggers, t)
}

func (v *dsValue) put(val interface{}) {
	v.recordsMu.Lock()

	if _, ok := v.records[val]; ok {
		v.recordsMu.Unlock()
		return
	}

	v.records[val] = true
	v.recordsMu.Unlock()

	v.triggersMu.RLock()
	defer v.triggersMu.RUnlock()

	for _, trigger := range v.triggers {
		go trigger.Trigger(val)
	}
}

func (v *dsValue) get() (s []interface{}) {
	v.recordsMu.RLock()
	defer v.recordsMu.RUnlock()

	for record := range v.records {
		s = append(s, record)
	}

	return
}

type ds map[Type]*dsValue

func (s ds) put(r Type, v interface{}) error {
	value, ok := s[r]
	if !ok {
		return ErrTypeNotDeclared
	}

	value.put(v)
	return nil
}

func (s ds) get(r Type) ([]interface{}, error) {
	value, ok := s[r]
	if !ok {
		return nil, ErrTypeNotDeclared
	}

	return value.get(), nil
}

func (s ds) hook(r Type, t Trigger) error {
	value, ok := s[r]
	if !ok {
		return ErrTypeNotDeclared
	}

	value.hook(t)
	return nil
}

type dsElem struct {
	records []interface{}
	current int
}

func (e *dsElem) elem(pos int) (interface{}, error) {
	if pos >= len(e.records) {
		return nil, errors.New("storage: Elements are closed")
	}

	return e.records[pos], nil
}

func (e *dsElem) Next() bool {
	_, err := e.elem(e.current)
	e.current += 1
	return err == nil
}

func (e *dsElem) LoadTo(l Loader) error {
	if e.current == 0 {
		return errors.New("storage: LoadTo called without calling Next")
	}

	elem, err := e.elem(e.current - 1)
	if err != nil {
		return err
	}

	return l.Load(elem)
}

type dsStmt struct {
	ds    *ds
	exec  func(*ds, ...interface{}) (Result, error)
	query func(*ds, ...interface{}) (Elements, error)
}

func (s *dsStmt) Exec(args ...interface{}) (Result, error) {
	return s.exec(s.ds, args...)
}

func (s *dsStmt) Query(args ...interface{}) (Elements, error) {
	return s.query(s.ds, args...)
}

func (s *dsStmt) Close() error {
	return nil
}

func putExec(ds *ds, args ...interface{}) (Result, error) {
	err := ds.put(args[0].(Type), args[1].(interface{}))
	return nil, err
}

func hookExec(ds *ds, args ...interface{}) (Result, error) {
	err := ds.hook(args[0].(Type), args[1].(Trigger))
	return nil, err
}

func getQuery(ds *ds, args ...interface{}) (Elements, error) {
	records, err := ds.get(args[0].(Type))
	if err != nil {
		return nil, err
	}

	return &dsElem{records, 0}, nil
}

func nopExec(*ds, ...interface{}) (Result, error) {
	return nil, errors.New("storage: statement is not prepared")
}

func nopQuery(*ds, ...interface{}) (Elements, error) {
	return nil, errors.New("storage: statement is not prepared")
}

func prepareStmt() map[Type]*stmt {
	stmts := make(map[Type]*stmt)
	ds := make(ds)

	putStmt := &dsStmt{&ds, putExec, nopQuery}
	getStmt := &dsStmt{&ds, nopExec, getQuery}
	hookStmt := &dsStmt{&ds, hookExec, nopQuery}

	for t := range types {
		records := make(map[interface{}]bool)
		ds[t] = &dsValue{records: records}
		stmts[t] = &stmt{putStmt, getStmt, hookStmt}
	}

	return stmts
}

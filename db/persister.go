package db

import (
	"github.com/netrack/netrack/db/record"
)

type Result interface {
	ElementsAffected() (int64, error)
}

type Elements interface {
	Close() error
	Next() bool
	Scan(dest ...interface{}) error
}

type Stmt interface {
	Close() error
	Exec(args ...interface{}) (Result, error)
	Query(args ...interface{}) (Elements, error)
}

type Persister interface {
	Add(record.Type, interface{}) error
	Get(record.Type, interface{}) error
	Prepare(query string) (Stmt, error)
	Close() error
}

type vDB struct {
}

func (db *vDB) Add(r record.Type, v interface{}) error {
	return nil
}

var DefaultDB Persister

func Add(r record.Type, v interface{}) error {
	return DefaultDB.Add(r, v)
}

func Get(r record.Type, v interface{}) error {
	return DefaultDB.Get(r, v)
}

func Close() error {
	return DefaultDB.Close()
}

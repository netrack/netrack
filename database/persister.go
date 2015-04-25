package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
)

const (
	// FakeModel just for testing purposes.
	FakeModel Model = iota

	// NetworkModel represents network layer database model
	NetworkModel

	// LinkModel represents link layer database model
	LinkModel
)

type Model int

var ErrNoRows = sql.ErrNoRows

type Persister interface {
	io.Closer

	Create(Model, interface{}) error
	Update(Model, string, interface{}) error
	Read(Model, string, interface{}) error
	Delete(Model, string) error
}

type sqlStmt struct {
	createStmt *sql.Stmt
	updateStmt *sql.Stmt
	readStmt   *sql.Stmt
	deleteStmt *sql.Stmt
}

type sqlDB struct {
	sqldb *sql.DB
	stmts map[Model]*sqlStmt
}

func Open(connstr string) (*sqlDB, error) {
	db, err := sql.Open(driverName, connstr)
	if err != nil {
		return nil, err
	}

	stmts, err := prepareStmts(db)
	if err != nil {
		return nil, err
	}

	return &sqlDB{db, stmts}, nil
}

func (db *sqlDB) modelToStmt(m Model) (*sqlStmt, error) {
	stmt, ok := db.stmts[m]
	if !ok {
		return nil, fmt.Errorf("db: model is not present")
	}

	return stmt, nil
}

func (db *sqlDB) editRecord(m interface{}, stmt *sql.Stmt, args ...interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	r, err := stmt.Exec(append([]interface{}{b}, args...)...)
	if err != nil {
		return err
	}

	if affected, _ := r.RowsAffected(); affected == 0 {
		return ErrNoRows
	}

	return nil
}

func (db *sqlDB) readRecord(m interface{}, stmt *sql.Stmt, args ...interface{}) error {
	var b []byte

	// Row is always non-nil
	err := stmt.QueryRow(args...).Scan(&b)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, m)
}

// Create makes a new record in the database
func (db *sqlDB) Create(m Model, s interface{}) error {
	stmt, err := db.modelToStmt(m)
	if err != nil {
		return err
	}

	return db.editRecord(s, stmt.createStmt)
}

// Update replaces existing record in the database by the provided
func (db *sqlDB) Update(m Model, id string, s interface{}) error {
	stmt, err := db.modelToStmt(m)
	if err != nil {
		return err
	}

	return db.editRecord(s, stmt.updateStmt, id)
}

// Read retrieves record from the database by specified id
func (db *sqlDB) Read(m Model, id string, s interface{}) error {
	stmt, err := db.modelToStmt(m)
	if err != nil {
		return err
	}

	return db.readRecord(s, stmt.readStmt, id)
}

// Delete removes record from the database by specified id
func (db *sqlDB) Delete(m Model, id string) error {
	stmt, err := db.modelToStmt(m)
	if err != nil {
		return err
	}

	r, err := stmt.deleteStmt.Exec(id)
	if err != nil {
		return err
	}

	if affected, _ := r.RowsAffected(); affected == 0 {
		return ErrNoRows
	}

	return nil
}

func (db *sqlDB) Close() error {
	return db.sqldb.Close()
}

var DefaultDB Persister

func Create(m Model, s interface{}) error {
	return DefaultDB.Create(m, s)
}

func Update(m Model, id string, s interface{}) error {
	return DefaultDB.Update(m, id, s)
}

func Read(m Model, id string, s interface{}) error {
	return DefaultDB.Read(m, id, s)
}

func Delete(m Model, id string) error {
	return DefaultDB.Delete(m, id)
}

func Close() error {
	return DefaultDB.Close()
}

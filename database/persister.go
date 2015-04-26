package db

import (
	"database/sql"
	"encoding/json"
	"io"
)

const (
	// FakeModel just for testing purposes.
	FakeModel Model = "fake"
)

type Model string

var models = map[Model]bool{FakeModel: true}

func Register(m Model) {
	models[m] = true
}

var ErrNoRows = sql.ErrNoRows

type Stmt struct {
	Lock   *sql.Stmt
	Create *sql.Stmt
	Update *sql.Stmt
	Read   *sql.Stmt
	Delete *sql.Stmt
}

type Statementer interface {
	Stmt(Model) (*Stmt, error)
}

type StatementerFunc func(Model) (*Stmt, error)

func (fn StatementerFunc) Stmt(m Model) (*Stmt, error) {
	return fn(m)
}

type Transactor interface {
	// Transaction invokes specified function inside database transaction.
	Transaction(func(ModelPersister) error) error
}

type ModelPersister interface {
	// Lock locks and reads record to interface.
	// This has effect only inside transaction.
	Lock(Model, string, interface{}) error

	// Create creates a new record in a table, specified in a Model paramter.
	Create(Model, interface{}) error

	// Update updates a record in a table, specified in a interface{} parameter.
	Update(Model, string, interface{}) error

	// Read reads single record specified by ID.
	Read(Model, string, interface{}) error

	// Delete deletes single record specified by ID.
	Delete(Model, string) error
}

type Persister interface {
	io.Closer

	Transactor
	ModelPersister
}

type sqlDB struct {
	sqldb *sql.DB
	stmts Statementer
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

	// Do not read if not necessary.
	if m == nil {
		return nil
	}

	return json.Unmarshal(b, m)
}

// Lock reads record from the database for update.
func (db *sqlDB) Lock(m Model, id string, s interface{}) error {
	stmt, err := db.stmts.Stmt(m)
	if err != nil {
		return err
	}

	return db.readRecord(s, stmt.Read, id)
}

// Create makes a new record in the database
func (db *sqlDB) Create(m Model, s interface{}) error {
	stmt, err := db.stmts.Stmt(m)
	if err != nil {
		return err
	}

	return db.editRecord(s, stmt.Create)
}

// Update replaces existing record in the database by the provided
func (db *sqlDB) Update(m Model, id string, s interface{}) error {
	stmt, err := db.stmts.Stmt(m)
	if err != nil {
		return err
	}

	return db.editRecord(s, stmt.Update, id)
}

// Read retrieves record from the database by specified id
func (db *sqlDB) Read(m Model, id string, s interface{}) error {
	stmt, err := db.stmts.Stmt(m)
	if err != nil {
		return err
	}

	return db.readRecord(s, stmt.Read, id)
}

// Delete removes record from the database by specified id
func (db *sqlDB) Delete(m Model, id string) error {
	stmt, err := db.stmts.Stmt(m)
	if err != nil {
		return err
	}

	r, err := stmt.Delete.Exec(id)
	if err != nil {
		return err
	}

	if affected, _ := r.RowsAffected(); affected == 0 {
		return ErrNoRows
	}

	return nil
}

func (db *sqlDB) Transaction(fn func(ModelPersister) error) error {
	tx, err := db.sqldb.Begin()
	if err != nil {
		return err
	}

	// Create transaction statementer
	stmt := StatementerFunc(func(m Model) (*Stmt, error) {
		stmt, err := db.stmts.Stmt(m)
		if err != nil {
			return nil, err
		}

		return &Stmt{
			Lock:   tx.Stmt(stmt.Lock),
			Create: tx.Stmt(stmt.Create),
			Update: tx.Stmt(stmt.Update),
			Read:   tx.Stmt(stmt.Read),
			Delete: tx.Stmt(stmt.Delete),
		}, nil
	})

	if err = fn(&sqlDB{db.sqldb, stmt}); err != nil {
		// Rollback transaction on failure.
		defer tx.Rollback()

		return err
	}

	// Or commit on success.
	return tx.Commit()
}

func (db *sqlDB) Close() error {
	return db.sqldb.Close()
}

var DefaultDB Persister

func Lock(m Model, id string, s interface{}) error {
	return DefaultDB.Lock(m, id, s)
}

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

func Transaction(fn func(ModelPersister) error) error {
	return DefaultDB.Transaction(fn)
}

func Close() error {
	return DefaultDB.Close()
}

package db

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
)

const (
	driverName = "postgres"
)

type pgStmt map[Model]*Stmt

func (s pgStmt) Stmt(m Model) (*Stmt, error) {
	if stmts, ok := s[m]; ok {
		return stmts, nil
	}

	return nil, errors.New("db: model in not presetnt")
}

func prepareHelper(db *sql.DB, resource string, idx ...string) (*Stmt, error) {
	var buf bytes.Buffer

	for _, column := range idx {
		buf.WriteString(fmt.Sprintf("->'%s'", column))
	}

	// Prepare statement for INSERT operations
	query := fmt.Sprintf("INSERT INTO %ss (%s) VALUES ($1)", resource, resource)
	createStmt, err := db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("db: failed to prepare sql INSERT statement: '%s'", err)
	}

	// Prepare statement for UPDATE operations
	query = fmt.Sprintf("UPDATE %ss SET %s = ($1) WHERE %s%s->>'id' = ($2)",
		resource, resource, resource, buf.String())

	updateStmt, err := db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("db: failed to prepare sql UPDATE statement: '%s'", err)
	}

	// Prepare statement for SELECT FOR UPDATE operations
	query = fmt.Sprintf("SELECT %s FROM %ss WHERE %s%s->>'id' = ($1) FOR UPDATE",
		resource, resource, resource, buf.String())

	lockStmt, err := db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("db: failed to prepare sql SELECT FOR UPDATE statement: '%s'", err)
	}

	// Prepare statement for SELECT operations
	query = fmt.Sprintf("SELECT %s FROM %ss WHERE %s%s->>'id' = ($1)",
		resource, resource, resource, buf.String())

	readStmt, err := db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("db: failed to prepare sql SELECT statement: '%s'", err)
	}

	// Prepare statement for DELETE operations
	query = fmt.Sprintf("DELETE FROM %ss WHERE %s%s->>'id' = ($1)",
		resource, resource, buf.String())

	deleteStmt, err := db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("db: failed to prepare sql DELETE statement: '%s'", err)
	}

	return &Stmt{lockStmt, createStmt, updateStmt, readStmt, deleteStmt}, nil
}

func prepareStmts(db *sql.DB) (Statementer, error) {
	pgStmt := make(pgStmt)

	for model := range models {
		stmt, err := prepareHelper(db, string(model))
		if err != nil {
			return nil, err
		}

		pgStmt[model] = stmt
	}

	return pgStmt, nil
}

// TruncateTables erases data from tables.
func TruncateTables(db *sql.DB) {
	for model := range models {
		db.Exec(fmt.Sprintf("DELETE FROM %ss", string(model)))
	}
}

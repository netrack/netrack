package db

import (
	"bytes"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

const (
	driverName = "postgres"
)

func prepareHelper(db *sql.DB, resource string, idx ...string) (*sqlStmt, error) {
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

	return &sqlStmt{createStmt, updateStmt, readStmt, deleteStmt}, nil
}

func prepareStmts(db *sql.DB) (map[Model]*sqlStmt, error) {
	fakeStmt, err := prepareHelper(db, "fake")
	if err != nil {
		return nil, err
	}

	linkStmt, err := prepareHelper(db, "link")
	if err != nil {
		return nil, err
	}

	networkStmt, err := prepareHelper(db, "network")
	if err != nil {
		return nil, err
	}

	return map[Model]*sqlStmt{
		FakeModel:    fakeStmt,
		LinkModel:    linkStmt,
		NetworkModel: networkStmt,
	}, nil
}

// TruncateTables erases data from tables.
func TruncateTables(db *sql.DB) {
	db.Exec("DELETE FROM fakes")
	db.Exec("DELETE FROM links")
	db.Exec("DELETE FROM networks")
}

package db

import (
	"reflect"
	"testing"

	"github.com/netrack/netrack/test"
)

func TestDBCreate(t *testing.T) {
	withDb(t, func() {
		record := map[string]interface{}{"id": "1", "column": 12345}
		err := DefaultDB.Create(FakeModel, record)
		if err != nil {
			t.Fatalf("Failed to create a new record: '%s'", err)
		}
	})
}

func TestDBRead(t *testing.T) {
	withDb(t, func() {
		record := map[string]interface{}{"id": "2", "column": "12345"}
		err := DefaultDB.Create(FakeModel, record)
		if err != nil {
			t.Fatalf("Failed to create a new record: '%s'", err)
		}

		var testrecord map[string]interface{}
		err = DefaultDB.Read(FakeModel, "2", &testrecord)
		if err != nil {
			t.Fatalf("Failed to read a record: '%s'", err)
		}

		if !reflect.DeepEqual(record, testrecord) {
			t.Fatalf("Records are not equal")
		}

		err = DefaultDB.Read(FakeModel, "3", &testrecord)
		if err != ErrNoRows {
			t.Fatalf("Failed to repot error on invalid id")
		}
	})
}

func TestDBUpdate(t *testing.T) {
	withDb(t, func() {
		record := map[string]interface{}{"id": "3", "column": "12345"}
		err := DefaultDB.Create(FakeModel, record)
		if err != nil {
			t.Fatalf("Failed to create a new record: '%s'", err)
		}

		record = map[string]interface{}{"id": "3", "column": "01234"}
		err = DefaultDB.Update(FakeModel, "3", record)
		if err != nil {
			t.Fatalf("Failed to update record: '%s'", err)
		}

		var testrecord map[string]interface{}
		err = DefaultDB.Read(FakeModel, "3", &testrecord)
		if err != nil {
			t.Fatalf("Failed to read updated record: '%s'", err)
		}

		if !reflect.DeepEqual(record, testrecord) {
			t.Fatalf("Records are not equal")
		}

		err = DefaultDB.Update(FakeModel, "4", record)
		if err != ErrNoRows {
			t.Fatalf("Failed to report error on invalid id", err)
		}
	})
}

func TestDBDelete(t *testing.T) {
	withDb(t, func() {
		record := map[string]interface{}{"id": "4", "column": "12345"}
		err := DefaultDB.Create(FakeModel, record)
		if err != nil {
			t.Fatalf("Failed to create a new record: '%s'", err)
		}

		err = DefaultDB.Delete(FakeModel, "4")
		if err != nil {
			t.Fatalf("Failed to remove record: '%s'", err)
		}

		var testrecord map[string]interface{}
		err = DefaultDB.Read(FakeModel, "4", &testrecord)
		if err != ErrNoRows {
			t.Fatalf("Failed to report error on invalid id")
		}

		err = DefaultDB.Delete(FakeModel, "5")
		if err != ErrNoRows {
			t.Fatalf("Failed to report error on invalid id", err)
		}
	})
}

func withDb(t *testing.T, fn func()) {
	config, err := test.Config()
	if err != nil {
		t.Fatalf("Failed to load configuration: '%s'", err)
	}

	db, err := Open(config.ConnString())
	if err != nil {
		t.Fatalf("Failed to open database connection: '%s'", err)
	}

	defer db.Close()

	TruncateTables(db.sqldb)
	defer TruncateTables(db.sqldb)

	DefaultDB = db
	fn()
}

package storage

import (
	"testing"
)

const TestRecord = "TEST-RECORD"

func TestRedisAdd(t *testing.T) {
	withDb(func() {
		err := Add(TestRecord, "somevalue")
		if err != nil {
			t.Fatal("Failed to add value to a set:", err)
		}
	})
}

func TestRedisGet(t *testing.T) {
	withDb(func() {
		err := Add(TestRecord, "somevalue")
		if err != nil {
			t.Fatal("Failed to add value to a set:", err)
		}

		var v []string
		err = Get(TestRecord, &v)
		if err != nil {
			t.Fatal("Failde to return data:", err)
		}
	})
}

func withDb(fn func()) {
	rdb := Open(&Config{Addr: "127.0.0.1", Port: 6379})

	DefaultDB = rdb
	defer DefaultDB.Close()

	conn := rdb.pool.Get()
	defer conn.Close()
	defer conn.Send(FlushAllCommand)

	fn()
}

package storage

import (
	"fmt"
	"sync"
	"testing"
)

func TestPersister(t *testing.T) {
	p := NewPersister()
	var addrs []string

	err := p.Get(NeighAddrType, StringSliceLoader(&addrs))
	if err != ErrTypeNotDeclared {
		t.Fatal("Failed to report not declared type error:", err)
	}

	err = p.Put(NeighAddrType, "2001:db8:cafe::1")
	if err != ErrTypeNotDeclared {
		t.Fatal("Failed to report not declared type error:", err)
	}

	err = p.Hook(NeighAddrType, TriggerFunc(func(interface{}) error {
		return nil
	}))

	if err != ErrTypeNotDeclared {
		t.Fatal("Failed to report not declared type error:", err)
	}

	err = p.Close()
	if err != nil {
		t.Fatal("Failed to close persister:", err)
	}
}

func TestPersisterPut(t *testing.T) {
	types = make(map[Type]bool)
	Declare(NeighAddrType)

	p := NewPersister()
	var addrs []string

	err := p.Get(NeighAddrType, StringSliceLoader(&addrs))
	if err != nil {
		t.Fatal("Failed to retrieve records:", err)
	}

	if len(addrs) != 0 {
		t.Fatal("Failed to return empty set:", err)
	}

	err = p.Put(NeighAddrType, "2001:db8:cafe::1")
	if err != nil {
		t.Fatal("Failed to put a new record:", err)
	}

	err = p.Get(NeighAddrType, StringSliceLoader(&addrs))
	if err != nil {
		t.Fatal("Failed to retrieve records:", err)
	}

	if len(addrs) != 1 {
		t.Fatal("Failed to return single record, returned:", len(addrs))
	}

	if addrs[0] != "2001:db8:cafe::1" {
		t.Fatal("Failed to return posted value:", addrs[0])
	}
}

func TestPersisterPutMultiple(t *testing.T) {
	types = make(map[Type]bool)
	Declare(NeighAddrType)

	p := NewPersister()
	var addrs []string

	for i := 0; i < 10000; i++ {
		err := p.Put(NeighAddrType, "2001:acad:1::1")
		if err != nil {
			t.Fatal("Failed to put a new record:", err)
		}
	}

	err := p.Get(NeighAddrType, StringSliceLoader(&addrs))
	if err != nil {
		t.Fatal("Failed to retrieve records:", err)
	}

	if len(addrs) != 1 {
		t.Fatal("Failed to return single record, returned:", len(addrs))
	}

	if addrs[0] != "2001:acad:1::1" {
		t.Fatal("Failed to return posted value:", addrs[0])
	}

	for i := 0; i < 10000; i++ {
		err := p.Put(NeighAddrType, fmt.Sprintf("%d", i))
		if err != nil {
			t.Fatal("Failed to put a new record:", err)
		}
	}

	err = p.Get(NeighAddrType, StringSliceLoader(&addrs))
	if err != nil {
		t.Fatal("Failed to retrieve records:", err)
	}

	if len(addrs) != 10001 {
		t.Fatal("Failed to return valid quantity of records, returned:", len(addrs))
	}
}

func TestPersisterHook(t *testing.T) {
	types = make(map[Type]bool)
	Declare(NeighAddrType)

	p := NewPersister()

	var wg sync.WaitGroup
	wg.Add(1)

	err := p.Hook(NeighAddrType, TriggerFunc(func(v interface{}) error {
		defer wg.Done()

		if v != "2001:acad:1::1" {
			t.Fatal("Failed to return posted value:", v)
		}

		return nil
	}))

	if err != nil {
		t.Fatal("Failed to register trigger:", err)
	}

	for i := 0; i < 10000; i++ {
		err := p.Put(NeighAddrType, "2001:acad:1::1")
		if err != nil {
			t.Fatal("Failed to put a new record:", err)
		}
	}

	wg.Wait()
}

func TestPeristerHookMultiple(t *testing.T) {
	types = make(map[Type]bool)
	Declare(NeighAddrType)

	p := NewPersister()

	var wg sync.WaitGroup
	wg.Add(10000)

	err := p.Hook(NeighAddrType, TriggerFunc(func(v interface{}) error {
		defer wg.Done()
		return nil
	}))

	if err != nil {
		t.Fatal("Failed to register trigger:", err)
	}

	for i := 0; i < 10000; i++ {
		err := p.Put(NeighAddrType, fmt.Sprintf("%d", i))
		if err != nil {
			t.Fatal("Failed to put a new record:", err)
		}
	}

	wg.Wait()
}

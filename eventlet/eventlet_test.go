package eventlet

import (
	"fmt"
	"sync"
	"testing"
)

func TestSpawnerTell(t *testing.T) {
	var wg sync.WaitGroup
	var addr string

	s := New()

	wg.Add(1)

	s.Hook("addr", HandlerFunc(func(e Event) error {
		defer wg.Done()

		err := e.LoadTo(StringLoader(&addr))
		if err != nil {
			t.Fatal("Failed to load received string:", err)
		}

		return nil
	}))

	s.Tell(&StringEvent{"addr", "2001:db8:cafe::1"})

	wg.Wait()

	if addr != "2001:db8:cafe::1" {
		t.Fatal("Failed to return posted value:", addr)
	}
}

func TestSpawnerTellMultiple(t *testing.T) {
	var wg sync.WaitGroup
	var lock sync.Mutex
	var addrs []string

	s, num := New(), 10000
	wg.Add(num)

	for i := 0; i < num; i++ {
		s.Hook("addr", HandlerFunc(func(e Event) error {
			defer wg.Done()

			var addr string
			err := e.LoadTo(StringLoader(&addr))
			if err != nil {
				t.Fatal("Failed to load received string:", err)
			}

			lock.Lock()
			defer lock.Unlock()
			addrs = append(addrs, addr)

			return nil
		}))
	}

	s.Tell(&StringEvent{"addr", "2001:db8:cafe::2"})

	wg.Wait()

	if len(addrs) != num {
		t.Fatal("Failed to post num events, got:", len(addrs))
	}

	for i := 0; i < num; i++ {
		if addrs[i] != "2001:db8:cafe::2" {
			t.Fatal("Failed to return posted value:", addrs[i])
		}
	}
}

func TestSpawnerHookMultiple(t *testing.T) {
	var wg sync.WaitGroup
	var lock sync.Mutex

	addrs := make(map[string]bool)
	s, num := New(), 10000

	wg.Add(num)

	for i := 0; i < num; i++ {
		s.Hook(Type(fmt.Sprintf("%d", i)), HandlerFunc(func(e Event) error {
			defer wg.Done()

			var addr string
			err := e.LoadTo(StringLoader(&addr))
			if err != nil {
				t.Fatal("Failed to load received string:", err)
			}

			lock.Lock()
			defer lock.Unlock()
			addrs[addr] = true

			return nil
		}))
	}

	for i := 0; i < num; i++ {
		s.Tell(&StringEvent{Type(fmt.Sprintf("%d", i)), fmt.Sprintf("%d", i)})
	}

	wg.Wait()

	if len(addrs) != num {
		t.Fatal("Failed to return 10000 posted events, got:", len(addrs))
	}
}

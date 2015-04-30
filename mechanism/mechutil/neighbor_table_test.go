package mechutil

import (
	"testing"

	"github.com/netrack/netrack/mechanism"
)

type LinkAddr string

func (lladdr LinkAddr) String() string {
	return string(lladdr)
}

func (lladdr LinkAddr) Bytes() []byte {
	return []byte(lladdr)
}

type NetworkAddr string

func (nladdr NetworkAddr) Contains(mech.NetworkAddr) bool {
	return false
}

func (nladdr NetworkAddr) String() string {
	return string(nladdr)
}

func (nladdr NetworkAddr) Bytes() []byte {
	return []byte(nladdr)
}

func (nladdr NetworkAddr) Mask() []byte {
	return nil
}

func TestNeighTable(t *testing.T) {
	table := NewNeighTable()

	table.Populate(NeighEntry{
		NetworkAddr: NetworkAddr("1.1.1.1"),
		LinkAddr:    LinkAddr("2-2-2-2"),
		Port:        42,
	})

	table.Populate(NeighEntry{
		NetworkAddr: NetworkAddr("1.1.1.1"),
		LinkAddr:    LinkAddr("2-2-2-2"),
		Port:        43,
	})

	neigh, ok := table.Lookup(NetworkAddr("1.1.1.1"))
	if !ok {
		t.Fatal("Failed to return neighbor entry")
	}

	if neigh.Port != 42 {
		t.Fatal("Failed to return right neighbor instance:", neigh.Port)
	}
}

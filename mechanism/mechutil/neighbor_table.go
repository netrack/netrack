package mechutil

import (
	"bytes"
	"sync"
	"time"

	"github.com/netrack/netrack/mechanism"
)

type NeighEntry struct {
	NetworkAddr mech.NetworkAddr
	LinkAddr    mech.LinkAddr
	Port        uint32
	Time        time.Time
}

type NeighTable struct {
	neighs map[string]NeighEntry
	lock   sync.RWMutex
}

func NewNeighTable() *NeighTable {
	neighs := make(map[string]NeighEntry)
	return &NeighTable{neighs: neighs}
}

func (t *NeighTable) Populate(entry NeighEntry) {
	t.lock.Lock()
	defer t.lock.Unlock()

	entry.Time = time.Now()

	nladdr := entry.NetworkAddr.String()
	neighEntry, ok := t.neighs[nladdr]
	// If network address is the first one
	if !ok {
		t.neighs[nladdr] = entry
		return
	}

	// Entry already in the table
	if bytes.Equal(neighEntry.LinkAddr.Bytes(), entry.LinkAddr.Bytes()) {
		return
	}

	// Add a new entry in a table
	t.neighs[nladdr] = entry
}

func (t *NeighTable) List() []NeighEntry {
	var entries []NeighEntry
	for _, entry := range t.neighs {
		entries = append(entries, entry)
	}

	return entries
}

func (t *NeighTable) Lookup(nladdr mech.NetworkAddr) (NeighEntry, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if entry, ok := t.neighs[nladdr.String()]; ok {
		return entry, true
	}

	return NeighEntry{}, false
}

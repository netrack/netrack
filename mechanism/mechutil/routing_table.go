package mechutil

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/netrack/netrack/mechanism"
)

var distanceMap = map[mech.RouteType]int{
	mech.StaticRoute:    0,
	mech.LocalRoute:     0,
	mech.ConnectedRoute: 1,
	mech.EIGRPRoute:     90,
	mech.OSPFRoute:      110,
	mech.RIPRoute:       120,
}

func routeToDistance(r mech.RouteType) (int, error) {
	distance, ok := distanceMap[r]
	if !ok {
		return 0, errors.New("route: unsupported route type")
	}

	return distance, nil
}

type RouteEntry struct {
	Type      mech.RouteType
	Network   mech.NetworkAddr
	NextHop   mech.NetworkAddr
	Distance  int
	Metric    int
	Timestamp time.Time
	Port      uint32
}

func (e *RouteEntry) Equal(entry *RouteEntry) bool {
	if !bytes.Equal(e.Network.Bytes(), entry.Network.Bytes()) {
		return false
	}

	if !bytes.Equal(e.Network.Mask().Bytes(), entry.Network.Mask().Bytes()) {
		return false
	}

	if e.Port != entry.Port {
		return false
	}

	return true
}

type RoutingTable struct {
	routes []RouteEntry
	lock   sync.RWMutex
}

func NewRoutingTable() *RoutingTable {
	return &RoutingTable{}
}

func (t *RoutingTable) Populate(entry RouteEntry) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	distance, err := routeToDistance(entry.Type)
	if err != nil {
		return err
	}

	entry.Timestamp = time.Now()
	entry.Distance = distance

	t.routes = append(t.routes, entry)

	return nil
}

func (t *RoutingTable) Evict(entry RouteEntry) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	for i, e := range t.routes {
		if !e.Equal(&entry) {
			continue
		}

		t.routes = append(t.routes[i:], t.routes[i+1:]...)
		return true
	}

	return false
}

func (t *RoutingTable) Lookup(nladdr mech.NetworkAddr) (RouteEntry, bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var candidate *RouteEntry

	for _, entry := range t.routes {
		if entry.Network.Contains(nladdr) {
			e := entry

			if candidate == nil {
				candidate = &e
				continue
			}

			if candidate.Network.Mask().Len() < entry.Network.Mask().Len() {
				candidate = &e
				continue
			}

			if candidate.Distance > entry.Distance {
				candidate = &e
				continue
			}
		}
	}

	if candidate == nil {
		return RouteEntry{}, false
	}

	return *candidate, true
}

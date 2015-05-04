package mechutil

import (
	"bytes"
	"errors"
	//"sort"
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

	if !bytes.Equal(e.Network.Mask(), entry.Network.Mask()) {
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

	entry.Timestamp = time.Now()

	t.routes = append(t.routes, entry)
	//sort.Sort(t)

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

	for _, entry := range t.routes {
		if entry.Network.Contains(nladdr) {
			return entry, true
		}
	}

	return RouteEntry{}, false
}

func (t *RoutingTable) Len() int {
	return len(t.routes)
}

func (t *RoutingTable) Less(i, j int) bool {
	routei, routej := t.routes[i], t.routes[j]

	// That is dumb, but okay for the begining
	if routej.Network.Contains(routei.Network) {
		return true
	}

	if routei.Distance < routej.Distance {
		return true
	}

	// Compare metric
	if routei.Metric < routej.Metric {
		return true
	}

	return false
}

func (t *RoutingTable) Swap(i, j int) {
	t.routes[i], t.routes[j] = t.routes[j], t.routes[i]
}

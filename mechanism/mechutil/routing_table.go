package mechutil

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/netrack/netrack/mechanism"
)

const (
	StaticRoute    RouteType = "S"
	LocalRoute     RouteType = "L"
	ConnectedRoute RouteType = "C"
	EIGRPRoute     RouteType = "D"
	OSPFRoute      RouteType = "O"
	RIPRoute       RouteType = "R"
)

var distanceMap = map[RouteType]int{
	StaticRoute:    0,
	LocalRoute:     0,
	ConnectedRoute: 1,
	EIGRPRoute:     90,
	OSPFRoute:      110,
	RIPRoute:       120,
}

func routeToDistance(r RouteType) (int, error) {
	distance, ok := distanceMap[r]
	if !ok {
		return 0, errors.New("route: unsupported route type")
	}

	return distance, nil
}

type RouteType string

type RouteEntry struct {
	Type      RouteType
	Network   mech.NetworkAddr
	NextHop   mech.NetworkAddr
	Distance  int
	Metric    int
	Timestamp time.Time
	Port      uint32
}

type RoutingTable struct {
	routes []RouteEntry
	lock   sync.RWMutex
}

func (t *RoutingTable) Populate(entry RouteEntry) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	entry.Timestamp = time.Now()

	t.routes = append(t.routes, entry)
	sort.Sort(t)

	return nil
}

func (t *RoutingTable) Evict(entry RouteEntry) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	if entry.Network == nil {
		return false
	}

	for i, e := range t.routes {
		if e.Network.String() != entry.Network.String() {
			continue
		}

		if e.Port != entry.Port {
			continue
		}

		if (e.NextHop != nil) != (entry.Network != nil) {
			continue
		}

		checkNextHop := e.NextHop != nil && entry.NextHop != nil
		if checkNextHop && e.NextHop.String() != entry.NextHop.String() {
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

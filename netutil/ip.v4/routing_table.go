package ip

import (
	"errors"
	"net"
	"sort"
	"sync"
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
	ConnectedRoute: 1,
	EIGRPRoute:     90,
	OSPFRoute:      110,
	RIPRoute:       120,
}

func routeToDistance(r RouteType) (int, error) {
	distance, ok := distanceMap[r]
	if !ok {
		return 0, errors.New("ip: unsupported route type")
	}

	return distance, nil
}

type RouteType string

type RouteEntry struct {
	Type     RouteType
	Net      net.IPNet
	NextHop  net.IP
	Distance int
	//Metric
	//Timestamp
	//Port ofp.PortNo
}

type RoutingTable struct {
	routes []RouteEntry
	lock   sync.RWMutex
}

func (t *RoutingTable) Append(entry RouteEntry) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.routes = append(t.routes, entry)
	sort.Sort(t)

	return nil
}

func (t *RoutingTable) Lookup(ipaddr net.IP) (net.IPNet, error) {
	return net.IPNet{}, nil
}

func (t *RoutingTable) Len() int {
	return len(t.routes)
}

func (t *RoutingTable) Less(i, j int) bool {
	if t.routes[i].Distance < t.routes[j].Distance {
		return true
	}

	// Compare metric
	//if r.routes[i].Metric < r.routes[j].Metric {
	//}

	return false
}

func (t *RoutingTable) Swap(i, j int) {
	t.routes[i], t.routes[j] = t.routes[j], t.routes[i]
}

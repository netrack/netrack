package mech

import (
	"sync"

	"github.com/netrack/netrack/database"
	"github.com/netrack/netrack/logging"
)

const (
	// RouteModel is a database table name (routes)
	RouteModel db.Model = "route"
)

const (
	StaticRoute    RouteType = "static"
	LocalRoute     RouteType = "local"
	ConnectedRoute RouteType = "connected"
	EIGRPRoute     RouteType = "eigrp"
	OSPFRoute      RouteType = "ospf"
	RIPRoute       RouteType = "rip"
)

func init() {
	// Register model in a database to make it available
	db.Register(RouteModel)

	// Register routing mechanism manager as network layer mechanism
	constructor := NetworkMechanismConstructorFunc(func() NetworkMechanism {
		return NewRoutingMechanismManager()
	})

	RegisterNetworkMechanism("routing", constructor)
}

type RouteType string

type RoutingContext struct {
	Type    RouteType
	Network NetworkAddr
	NextHop NetworkAddr
	Driver  NetworkDriver
	Port    uint32
}

type Route struct {
	Type    string `json:"type"`
	Network string `json:"network"`
	NextHop string `json:"nexthop"`
	Port    uint32 `json:"port"`
}

func (c *Route) Equals(rc *Route) bool {
	return c.Network == rc.Network && c.NextHop == rc.NextHop
}

type RoutingMechanism interface {
	Mechanism

	// UpdateRoute is called for all changes to route state.
	UpdateRoute(*RoutingContext) error

	// DeleteRoute erases all allocated resources.
	DeleteRoute(*RoutingContext) error
}

// BaseRoutingMechanism implements RoutingMechanism interface.
type BaseRoutingMechanism struct {
	BaseMechanism
}

// UpdateRoute implements RoutingMechanism interface.
func (m *BaseRoutingMechanism) UpdateRoute(context *RoutingContext) error {
	return nil
}

// DeleteRoute implements RoutingMechanism interface.
func (m *BaseRoutingMechanism) DeleteRoute(context *RoutingContext) error {
	return nil
}

type RoutingManagerContext struct {
	Datapath string   `json:"id"`
	Routes   []*Route `json:"routes"`
}

func (c *RoutingManagerContext) SetRoute(r *Route) {
	for _, route := range c.Routes {
		if route.Equals(r) {
			return
		}
	}

	c.Routes = append(c.Routes, r)
}

func (c *RoutingManagerContext) DelRoute(r *Route) {
	for i, route := range c.Routes {
		if route.Equals(r) {
			c.Routes = append(c.Routes[:i], c.Routes[i+1:]...)
			return
		}
	}
}

// RoutingMechanismConstructor is a genereic
// constructor for data route type mechanisms.
type RoutingMechanismConstructor interface {
	// New returns a new RoutingMechanism instance.
	New() RoutingMechanism
}

// RoutingMechanismConstructorFunc is a function adapter for
// RoutingMechanismConstructor.
type RoutingMechanismConstructorFunc func() RoutingMechanism

func (fn RoutingMechanismConstructorFunc) New() RoutingMechanism {
	return fn()
}

var routes = make(map[string]RoutingMechanismConstructor)

// RegisterRoutingMechanism registers a new route layer mechanism
// under specified name.
func RegisterRoutingMechanism(name string, ctor RoutingMechanismConstructor) {
	if ctor == nil {
		log.FatalLog("route/REGISTER_ROUTE_MECHANISM",
			"Failed to register nil route constructor for: ", name)
	}

	if _, dup := routes[name]; dup {
		log.FatalLog("route/REGISTER_ROUTE_MECHANISM",
			"Falied to register duplicate route constructor for: ", name)
	}

	routes[name] = ctor
}

// RoutingMechanisms retruns instances of registered mechanisms.
func RoutingMechanisms() RoutingMechanismMap {
	lmap := make(RoutingMechanismMap)

	for name, constructor := range routes {
		lmap.Set(name, constructor.New())
	}

	return lmap
}

// RoutingMechanismMap implements MechanismMap interface.
type RoutingMechanismMap map[string]RoutingMechanism

// Get returns Mechanism by specified name.
func (m RoutingMechanismMap) Get(s string) (Mechanism, bool) {
	mechanism, ok := m[s]
	return mechanism, ok
}

// Set registers mechanism under specified name.
func (m RoutingMechanismMap) Set(s string, mechanism Mechanism) {
	rmechanism, ok := mechanism.(RoutingMechanism)
	if !ok {
		log.ErrorLog("route/SET_MECHANISM",
			"Failed to cast to route layer mechanism")
		return
	}

	m[s] = rmechanism
}

// Iter calls specified function for all registered mechanisms.
func (m RoutingMechanismMap) Iter(fn func(string, Mechanism) bool) {
	for s, mechanism := range m {
		fn(s, mechanism)
	}
}

type RoutingMechanismManager interface {
	// Base mechanism manager interface
	MechanismManager

	// Context return route context.
	Context() (*RoutingManagerContext, error)

	// UpdateRoutes forwards call to all registered mechanisms.
	UpdateRoutes(*RoutingManagerContext) error

	// DeleteRoutes forwards call to all registered mechanisms.
	DeleteRoutes(*RoutingManagerContext) error
}

type routingMechanismManager struct {
	BaseMechanismManager

	// Network layer driver.
	nldrv     NetworkDriver
	nldrvLock sync.RWMutex

	lock sync.RWMutex
}

func NewRoutingMechanismManager() *routingMechanismManager {
	return &routingMechanismManager{}
}

func (m *routingMechanismManager) Enable(c *MechanismContext) {
	m.BaseMechanismManager = BaseMechanismManager{
		Datapath:   c.Switch.ID(),
		Mechanisms: RoutingMechanisms(),
		activated:  0,
		enabled:    0,
	}

	m.BaseMechanismManager.Enable(c)
	c.Managers.Bind(new(RoutingMechanismManager), m)
}

func (m *routingMechanismManager) Activate() {
	m.BaseMechanismManager.Activate()

	routing := new(RoutingManagerContext)

	err := m.BaseMechanismManager.Create(
		RouteModel, routing, func() error { return nil },
	)

	if err != nil {
		log.ErrorLog("routing/ACTIVATE_HOOK",
			"Failed to create empty routing configuration")
		return
	}

	log.DebugLog("routing/ACTIVATE_HOOK",
		"Routing mechanism manager activated")
}

func (m *routingMechanismManager) SetNetworkDriver(nldriver NetworkDriver) {
	m.nldrvLock.Lock()
	defer m.nldrvLock.Unlock()

	m.nldrv = nldriver
}

func (m *routingMechanismManager) NetworkDriver() (NetworkDriver, error) {
	m.nldrvLock.RLock()
	defer m.nldrvLock.RUnlock()

	if m.nldrv == nil {
		log.ErrorLog("routing/NETWORK_DRIVER",
			"Network layer driver is not intialized")
		return nil, ErrNetworkNotInitialized
	}

	return m.nldrv, nil
}

func (m *routingMechanismManager) CreateNetworkPreCommit(context *NetworkContext) error {
	log.DebugLog("routing/CREATE_NETWORK_PRECOMMIT",
		"Got create precommit request")

	// Persist network layer driver
	m.SetNetworkDriver(context.NetworkDriver)
	return nil
}

func (m *routingMechanismManager) CreateNetworkPostCommit() error {
	log.DebugLog("routing/CREATE_NETWORK_POSTCOMMIT",
		"Got create postcommit request")

	err := m.CreateRoutes()
	if err != nil {
		log.ErrorLog("routing/CREATE_NETWORK_POSTCOMMIT",
			"Failed to restore routing configuration: ", err)
	}

	return err
}

func (m *routingMechanismManager) UpdateNetworkPreCommit(context *NetworkContext) error {
	log.DebugLog("routing/UPDATE_NETWORK_PRECOMMIT",
		"Got update precommit request")

	// Persist network layer driver
	m.SetNetworkDriver(context.NetworkDriver)

	err := m.DeleteRoutes(&RoutingManagerContext{
		Datapath: m.Datapath, Routes: []*Route{{
			Type:    string(ConnectedRoute),
			Network: context.NetworkAddr.String(),
			Port:    context.Port,
		}},
	})

	if err != nil {
		log.ErrorLog("routing/UPDATE_NETWORK_PRECOMMIT",
			"Failed update routing configuration: ", err)
	}

	return err
}

func (m *routingMechanismManager) UpdateNetworkPostCommit(context *NetworkContext) error {
	log.DebugLog("routing/UPDATE_NETWORK_POSTCOMMIT",
		"Got update network postcommit request")

	// Persist network layer driver
	m.SetNetworkDriver(context.NetworkDriver)

	err := m.UpdateRoutes(&RoutingManagerContext{
		Datapath: m.Datapath, Routes: []*Route{{
			Type:    string(ConnectedRoute),
			Network: context.NetworkAddr.String(),
			Port:    context.Port,
		}},
	})

	if err != nil {
		log.ErrorLog("routing/UPDATE_NETORK_POSTCOMMIT",
			"Failed update routing configuration: ", err)
	}

	return err
}

func (m *routingMechanismManager) DeleteNetworkPreCommit(context *NetworkContext) error {
	log.DebugLog("routing/DELETE_NETWORK_PRECOMMIT",
		"Got delete network precommit request")

	err := m.DeleteRoutes(&RoutingManagerContext{
		Datapath: m.Datapath, Routes: []*Route{{
			Type:    string(ConnectedRoute),
			Network: context.NetworkAddr.String(),
			Port:    context.Port,
		}},
	})

	if err != nil {
		log.ErrorLog("routing/DELETE_NETWORK_PRECOMMIT",
			"Failed update routing configuration: ", err)
	}

	return err
}

func (m *routingMechanismManager) DeleteNetworkPostCommit() error {
	log.DebugLog("routing/DELETE_NETWORK_PRECOMMIT",
		"Got delete network postcommit request")
	return nil
}

// Iter calls specified function for all registered link layer mechanisms.
func (m *routingMechanismManager) Iter(fn func(RoutingMechanism) bool) {
	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		rmechanism, ok := mechanism.(RoutingMechanism)
		if !ok {
			log.ErrorLog("route/ITERATE",
				"Failed to cast mechanism to route mechanism.")
			return true
		}

		return fn(rmechanism)
	})
}

type routeMechanismFunc func(RoutingMechanism, *RoutingContext) error

func (m *routingMechanismManager) do(fn routeMechanismFunc, context *RoutingContext) (err error) {
	callback := func(mechanism RoutingMechanism) bool {
		if !mechanism.Activated() {
			return true
		}

		err = fn(mechanism, context)
		if err != nil {
			log.ErrorLog("route/ALTER_ROUTE",
				"Failed to alter route mechanism: ", err)
			return false
		}

		return true
	}

	m.Iter(callback)

	return
}

func (m *routingMechanismManager) Context() (*RoutingManagerContext, error) {
	context := new(RoutingManagerContext)
	err := m.BaseMechanismManager.Context(RouteModel, context)
	if err != nil {
		log.ErrorLog("routing/CONTEXT",
			"Failed to retrieve persisted configuration: ", err)
	}

	return context, nil
}

func (m *routingMechanismManager) CreateRoutes() error {
	routing := new(RoutingManagerContext)

	create := func(fn func() error) error {
		return m.BaseMechanismManager.Create(
			RouteModel, routing, fn,
		)
	}

	alter := func(route *Route) error {
		routingContext, err := m.routingContext(route)
		if err != nil {
			return err
		}

		err = m.do(RoutingMechanism.UpdateRoute, routingContext)
		if err != nil {
			log.ErrorLog("routing/CREATE_ROUTES",
				"Failed to create routing configuration: ", err)
		}

		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return create(func() error {
		for _, route := range routing.Routes {
			if err := alter(route); err != nil {
				return err
			}
		}

		return nil
	})
}

func (m *routingMechanismManager) routingContext(route *Route) (*RoutingContext, error) {
	nldriver, err := m.NetworkDriver()
	if err != nil {
		return nil, err
	}

	networkAddr, err := nldriver.ParseAddr(route.Network)
	if err != nil {
		log.ErrorLog("routing/ROUTING_CONTEXT",
			"Failed to parse network address: ", err)
		return nil, err
	}

	nextHopAddr, err := nldriver.ParseAddr(route.NextHop)
	if route.NextHop != "" && err != nil {
		log.ErrorLog("routing/ROUTING_CONTEXT",
			"Failed to parse next-hop address: ", err)
		return nil, err
	}

	context := &RoutingContext{
		Type:    RouteType(route.Type),
		Network: networkAddr,
		NextHop: nextHopAddr,
		Driver:  nldriver,
		Port:    route.Port,
	}

	return context, nil
}

func (m *routingMechanismManager) UpdateRoutes(context *RoutingManagerContext) error {
	routing := new(RoutingManagerContext)

	update := func(fn func() error) error {
		return m.BaseMechanismManager.Update(
			RouteModel, routing, fn,
		)
	}

	alter := func(route *Route) error {
		routingContext, err := m.routingContext(route)
		if err != nil {
			return err
		}

		err = m.do(RoutingMechanism.UpdateRoute, routingContext)
		if err != nil {
			log.ErrorLog("routing/UPDATE_ROUTES",
				"Failed to update routing configuration: ", err)
		}

		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return update(func() error {
		for _, route := range context.Routes {
			if err := alter(route); err != nil {
				return err
			}

			routing.SetRoute(route)
		}

		return nil
	})
}

func (m *routingMechanismManager) DeleteRoutes(context *RoutingManagerContext) error {
	routes := new(RoutingManagerContext)

	update := func(fn func() error) error {
		return m.BaseMechanismManager.Update(
			RouteModel, routes, fn,
		)
	}

	alter := func(route *Route) error {
		routingContext, err := m.routingContext(route)
		if err != nil {
			return err
		}

		err = m.do(RoutingMechanism.DeleteRoute, routingContext)
		if err != nil {
			log.ErrorLog("routing/DELETE_ROUTES",
				"Failed to update routing configuration: ", err)
		}

		return err
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	return update(func() error {
		for _, route := range context.Routes {
			if err := alter(route); err != nil {
				return err
			}

			routes.DelRoute(route)
		}

		return nil
	})
}

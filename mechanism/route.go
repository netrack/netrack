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

func init() {
	// Register model in a database to make it available
	db.Register(RouteModel)
}

type RouteContext struct {
	Type    string `json:"type"`
	Network string `json:"network"`
	NextHop string `json:"nexthop"`
	Port    uint32 `json:"port"`
}

func (c *RouteContext) Equals(rc *RouteContext) bool {
	return c.Network == rc.Network && c.NextHop == rc.NextHop
}

type RouteMechanism interface {
	Mechanism

	UpdateRoute(*RouteContext) error

	DeleteRoute(*RouteContext) error
}

type BaseRouteMechanism struct {
	BaseMechanism
}

func (m *BaseRouteMechanism) UpdateRoute(context *RouteContext) error {
	return nil
}

func (m *BaseRouteMechanism) DeleteRoute(context *RouteContext) error {
	return nil
}

type RouteManagerContext struct {
	Datapath string          `json:"id"`
	Routes   []*RouteContext `json:"routes"`
}

func (c *RouteManagerContext) SetRoute(r *RouteContext) {
	for _, route := range c.Routes {
		if route.Equals(r) {
			return
		}
	}

	c.Routes = append(c.Routes, r)
}

func (c *RouteManagerContext) DelRoute(r *RouteContext) {
	for i, route := range c.Routes {
		if route.Equals(r) {
			c.Routes = append(c.Routes[:i], c.Routes[i+1:]...)
			return
		}
	}
}

// RouteMechanismConstructor is a genereic
// constructor for data route type mechanisms.
type RouteMechanismConstructor interface {
	// New returns a new RouteMechanism instance.
	New() RouteMechanism
}

// RouteMechanismConstructorFunc is a function adapter for
// RouteMechanismConstructor.
type RouteMechanismConstructorFunc func() RouteMechanism

func (fn RouteMechanismConstructorFunc) New() RouteMechanism {
	return fn()
}

var routes = make(map[string]RouteMechanismConstructor)

// RegisterRouteMechanism registers a new route layer mechanism
// under specified name.
func RegisterRouteMechanism(name string, ctor RouteMechanismConstructor) {
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

// RouteMechanisms retruns instances of registered mechanisms.
func RouteMechanisms() RouteMechanismMap {
	lmap := make(RouteMechanismMap)

	for name, constructor := range routes {
		lmap.Set(name, constructor.New())
	}

	return lmap
}

// RouteMechanismMap implements MechanismMap interface.
type RouteMechanismMap map[string]RouteMechanism

// Get returns Mechanism by specified name.
func (m RouteMechanismMap) Get(s string) (Mechanism, bool) {
	mechanism, ok := m[s]
	return mechanism, ok
}

// Set registers mechanism under specified name.
func (m RouteMechanismMap) Set(s string, mechanism Mechanism) {
	rmechanism, ok := mechanism.(RouteMechanism)
	if !ok {
		log.ErrorLog("route/SET_MECHANISM",
			"Failed to cast to route layer mechanism")
		return
	}

	m[s] = rmechanism
}

// Iter calls specified function for all registered mechanisms.
func (m RouteMechanismMap) Iter(fn func(string, Mechanism) bool) {
	for s, mechanism := range m {
		fn(s, mechanism)
	}
}

type RouteMechanismManager interface {
	MechanismManager

	// Context return route context.
	Context(*RouteManagerContext) error

	CreateRoutes() error

	UpdateRoutes(*RouteManagerContext) error

	DeleteRoutes(*RouteManagerContext) error
}

type BaseRouteMechanismManager struct {
	BaseMechanismManager

	lock sync.RWMutex
}

// Iter calls specified function for all registered link layer mechanisms.
func (m *BaseRouteMechanismManager) Iter(fn func(RouteMechanism) bool) {
	m.Mechanisms.Iter(func(_ string, mechanism Mechanism) bool {
		rmechanism, ok := mechanism.(RouteMechanism)
		if !ok {
			log.ErrorLog("route/ITERATE",
				"Failed to cast mechanism to route mechanism.")
			return true
		}

		return fn(rmechanism)
	})
}

type routeMechanismFunc func(RouteMechanism, *RouteContext) error

func (m *BaseRouteMechanismManager) do(fn routeMechanismFunc, context *RouteContext) (err error) {
	m.Iter(func(mechanism RouteMechanism) bool {
		if !mechanism.Activated() {
			return true
		}

		if err = fn(mechanism, context); err != nil {
			log.ErrorLog("route/ALTER_ROUTE",
				"Failed to alter route mechanism: ", err)
			return false
		}

		return true
	})

	return
}

func (m *BaseRouteMechanismManager) Context(context *RouteManagerContext) error {
	return m.BaseMechanismManager.Context(RouteModel, context)
}

func (m *BaseRouteMechanismManager) CreateRoutes() error {
	context := new(RouteManagerContext)

	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.BaseMechanismManager.Create(RouteModel, context, func() error {
		for _, route := range context.Routes {
			err := m.do(RouteMechanism.UpdateRoute, route)

			if err != nil {
				log.ErrorLog("route/CREATE_ROUTES",
					"Failed to create routes configuration: ", err)
				return err
			}
		}

		return nil
	})
}

func (m *BaseRouteMechanismManager) UpdateRoutes(context *RouteManagerContext) error {
	routes := new(RouteManagerContext)

	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.BaseMechanismManager.Update(RouteModel, routes, func() error {
		for _, route := range context.Routes {
			err := m.do(RouteMechanism.UpdateRoute, route)

			if err != nil {
				log.ErrorLog("route/UPDATE_ROUTE",
					"Failed to update routes configuration: ", err)
				return err
			}

			routes.SetRoute(route)
		}

		return nil
	})
}

func (m *BaseRouteMechanismManager) DeleteRoutes(context *RouteManagerContext) error {
	routes := new(RouteManagerContext)

	m.lock.RLock()
	defer m.lock.RUnlock()

	update := func(fn func() error) error {
		return m.BaseMechanismManager.Update(
			RouteModel, routes, fn,
		)
	}

	alter := func(route *RouteContext) error {
		err := m.do(RouteMechanism.DeleteRoute, route)

		if err != nil {
			log.ErrorLog("route/DELETE_ROUTE",
				"Failed to delete routes configuration: ", err)
			return err
		}

		routes.DelRoute(route)
		return nil
	}

	return update(func() error {
		for _, route := range context.Routes {
			if err := alter(route); err != nil {
				return err
			}
		}

		return nil
	})
}

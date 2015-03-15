package eventlet

import (
	"sync"
)

type Handler interface {
	Handle(e Event) error
}

type HandlerFunc func(Event) error

func (fn HandlerFunc) Handle(e Event) error {
	return fn(e)
}

type Spawner interface {
	// Tell posts the event to subscribers.
	Tell(e Event)

	// Hook subscribes the client to the specified event Type.
	Hook(t Type, h Handler)
}

func New() *eventSpawner {
	return &eventSpawner{
		hooks: make(map[Type]*eventHook),
	}
}

type eventHook struct {
	handlers []Handler
	lock     sync.RWMutex
}

func (h *eventHook) attach(handler Handler) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.handlers = append(h.handlers, handler)
}

func (h *eventHook) notify(event Event) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	for _, handler := range h.handlers {
		go handler.Handle(event)
	}
}

type eventSpawner struct {
	hooks map[Type]*eventHook
	lock  sync.RWMutex
}

func (p *eventSpawner) hook(t Type) *eventHook {
	p.lock.RLock()
	hook, ok := p.hooks[t]
	p.lock.RUnlock()

	if ok {
		return hook
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	hook, ok = p.hooks[t]
	if ok {
		return hook
	}

	p.hooks[t] = &eventHook{}
	return p.hooks[t]
}

func (p *eventSpawner) Tell(e Event) {
	go p.hook(e.Type()).notify(e)
}

func (p *eventSpawner) Hook(t Type, h Handler) {
	p.hook(t).attach(h)
}

var DefaultSpawner = New()

func Tell(e Event) {
	DefaultSpawner.Tell(e)
}

func Hook(t Type, h Handler) {
	DefaultSpawner.Hook(t, h)
}

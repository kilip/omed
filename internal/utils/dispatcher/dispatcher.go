package dispatcher

import (
	"sort"
	"sync"

	"github.com/kilip/omed/internal/contracts"
)

type ListenerInfo struct {
	Priority int
	Listener contracts.Listener
}

type Dispatcher struct {
	listeners map[string][]ListenerInfo
	mu        sync.RWMutex
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		listeners: make(map[string][]ListenerInfo),
	}
}

func (d *Dispatcher) ListenP(eventName string, listener contracts.Listener, priority int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.listeners[eventName] = append(d.listeners[eventName], ListenerInfo{
		Priority: priority,
		Listener: listener,
	})

	sort.Slice(d.listeners[eventName], func(i, j int) bool {
		return d.listeners[eventName][i].Priority < d.listeners[eventName][j].Priority
	})
}

func (d *Dispatcher) Listen(eventName string, listener contracts.Listener) {
	d.ListenP(eventName, listener, 99999)
}

func (d *Dispatcher) Dispatch(event contracts.Event) {
	eventName := event.GetName()
	d.mu.RLock()
	listeners, exists := d.listeners[eventName]
	d.mu.RUnlock()

	if !exists {
		return
	}

	for _, l := range listeners {
		l.Listener(event, d)
	}
}

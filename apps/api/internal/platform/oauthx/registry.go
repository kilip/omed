package oauthx

import (
	"sort"
	"sync"
)

// Registry manages registered OAuth drivers.
type Registry struct {
	mu      sync.RWMutex
	drivers map[string]Driver
}

// NewRegistry creates a new Registry and registers the given drivers.
func NewRegistry(drivers ...Driver) *Registry {
	r := &Registry{
		drivers: make(map[string]Driver, len(drivers)),
	}
	for _, d := range drivers {
		r.Register(d)
	}
	return r
}

// Register registers or replaces a driver by its ID.
func (r *Registry) Register(d Driver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drivers[d.ID()] = d
}

// Get returns the driver registered for the given ID, or ErrUnsupportedProvider if not found.
func (r *Registry) Get(id string) (Driver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.drivers[id]
	if !ok {
		return nil, ErrUnsupportedProvider
	}
	return d, nil
}

// IDs returns a sorted list of registered driver IDs.
func (r *Registry) IDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.drivers))
	for id := range r.drivers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

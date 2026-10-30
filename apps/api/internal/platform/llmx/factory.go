package llmx

import "sync"

var (
	registryMu sync.RWMutex
	registry   = make(map[Kind]Factory)
)

// Register registers a factory for a given provider kind.
func Register(kind Kind, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[kind] = factory
}

// New creates an Adapter for the specified Config.
// If cfg.HTTPClient is nil, New sets it using newHTTPClient(cfg.AllowPrivateURLs).
// Returns ErrUnsupportedKind if no factory has been registered for cfg.Kind.
func New(cfg Config) (Adapter, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = newHTTPClient(cfg.AllowPrivateURLs)
	}

	registryMu.RLock()
	factory, ok := registry[cfg.Kind]
	registryMu.RUnlock()

	if !ok {
		return nil, ErrUnsupportedKind
	}

	return factory(cfg)
}

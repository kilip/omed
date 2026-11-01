package llmx

import (
	"context"
	"errors"
	"strings"
	"sync"
)

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

// ListModels queries the provider for available model identifiers.
func ListModels(ctx context.Context, cfg Config) ([]string, error) {
	adapter, err := New(cfg)
	if err != nil {
		return nil, sanitizeError(err, cfg.APIKey)
	}

	lister, ok := adapter.(ModelLister)
	if !ok {
		return nil, ErrUnsupportedKind
	}

	models, err := lister.ListModels(ctx)
	if err != nil {
		return nil, sanitizeError(err, cfg.APIKey)
	}

	return models, nil
}

func sanitizeError(err error, apiKey string) error {
	if err == nil || apiKey == "" {
		return err
	}
	if strings.Contains(err.Error(), apiKey) {
		return errors.New(strings.ReplaceAll(err.Error(), apiKey, "[REDACTED]"))
	}
	return err
}

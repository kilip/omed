package mailx

import (
	"context"
	"sync"
)

// MemoryDriver stores sent messages in-memory for testing and local development.
type MemoryDriver struct {
	mu       sync.RWMutex
	messages []Message
	closed   bool
}

// NewMemoryDriver constructs a fresh MemoryDriver.
func NewMemoryDriver() *MemoryDriver {
	return &MemoryDriver{
		messages: make([]Message, 0),
	}
}

// Send records the given message in the in-memory log.
func (d *MemoryDriver) Send(ctx context.Context, msg Message) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	d.messages = append(d.messages, msg)
	return nil
}

// SentMessages returns a copy of all messages transmitted so far.
func (d *MemoryDriver) SentMessages() []Message {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cp := make([]Message, len(d.messages))
	copy(cp, d.messages)
	return cp
}

// LastMessage returns the most recently sent message, or nil if none.
func (d *MemoryDriver) LastMessage() *Message {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.messages) == 0 {
		return nil
	}
	msg := d.messages[len(d.messages)-1]
	return &msg
}

// Reset clears all recorded messages.
func (d *MemoryDriver) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messages = make([]Message, 0)
}

// Close marks the driver as closed.
func (d *MemoryDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}

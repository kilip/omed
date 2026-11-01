package llmx

import (
	"context"
	"iter"
	"sync"
)

func init() {
	Register(KindMemory, func(cfg Config) (Adapter, error) {
		return NewMemory(), nil
	})
}

var (
	_ Adapter     = (*Memory)(nil)
	_ ModelLister = (*Memory)(nil)
)

// Memory implements Adapter by returning scripted responses in order and recording received requests.
type Memory struct {
	mu       sync.Mutex
	replies  []Response
	requests []Request
}

// NewMemory creates a new Memory adapter initialized with the given scripted responses.
func NewMemory(replies ...Response) *Memory {
	reps := make([]Response, len(replies))
	copy(reps, replies)
	return &Memory{
		replies: reps,
	}
}

// Requests returns a copy of all requests received by the adapter.
func (m *Memory) Requests() []Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	reqs := make([]Request, len(m.requests))
	copy(reqs, m.requests)
	return reqs
}

// Chat returns the next scripted response in order, or an empty Response{} if exhausted.
func (m *Memory) Chat(ctx context.Context, req Request) (*Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = append(m.requests, req)

	if len(m.replies) == 0 {
		return &Response{}, nil
	}

	resp := m.replies[0]
	m.replies = m.replies[1:]
	return &resp, nil
}

// Stream yields EventText from the reply's first text part or message, then EventDone with Usage.
func (m *Memory) Stream(ctx context.Context, req Request) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		m.mu.Lock()
		m.requests = append(m.requests, req)
		var resp Response
		if len(m.replies) > 0 {
			resp = m.replies[0]
			m.replies = m.replies[1:]
		}
		m.mu.Unlock()

		if ctx.Err() != nil {
			yield(Event{}, ctx.Err())
			return
		}

		var text string
		for _, p := range resp.Message.Parts {
			if p.Text != "" {
				text = p.Text
				break
			}
		}

		if text != "" || len(resp.Message.Parts) > 0 {
			if !yield(Event{Type: EventText, Text: text}, nil) {
				return
			}
		}

		yield(Event{
			Type:  EventDone,
			Usage: &resp.Usage,
		}, nil)
	}
}

// Capabilities returns the default capabilities for KindOpenAICompat.
func (m *Memory) Capabilities() Capabilities {
	return DefaultCapabilities(KindOpenAICompat)
}

// ListModels returns a static list of available memory models.
func (m *Memory) ListModels(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{"memory-default", "memory-fast"}, nil
}

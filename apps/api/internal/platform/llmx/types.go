package llmx

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
)

// Kind identifies the LLM provider kind.
type Kind string

const (
	KindOpenAICompat Kind = "openai_compat"
	KindAnthropic    Kind = "anthropic"
	KindGemini       Kind = "gemini"
	KindMemory       Kind = "memory"
)

// Role defines the role of a message participant.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Image represents binary image data with a MIME type.
type Image struct {
	MIME string
	Data []byte
}

// Part is a chunk of content in a message (text or image).
type Part struct {
	Text  string
	Image *Image
}

// ToolCall represents a tool invocation produced by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Message is a chat message exchanged with the model.
type Message struct {
	Role       Role
	Parts      []Part
	ToolCalls  []ToolCall
	ToolCallID string
}

// Tool describes a callable function tool provided to the model.
type Tool struct {
	Name        string
	Description string
	Schema      json.RawMessage
}

// Request holds parameters for a model invocation.
type Request struct {
	Model          string
	Messages       []Message
	Tools          []Tool
	ResponseSchema json.RawMessage
	MaxTokens      int
	Temperature    *float64
}

// Usage reports token consumption for an invocation.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Response represents the output of a model invocation.
type Response struct {
	Message    Message
	Usage      Usage
	StopReason string
}

// EventType indicates the type of a streaming event.
type EventType int

const (
	EventText EventType = iota
	EventToolCall
	EventDone
)

// Event is an item emitted during streaming.
type Event struct {
	Type     EventType
	Text     string
	ToolCall *ToolCall
	Usage    *Usage
}

// Capabilities indicates features supported by a provider or adapter.
type Capabilities struct {
	Vision           bool
	ToolCalling      bool
	StructuredOutput bool
	Streaming        bool
}

// Adapter defines the uniform interface for LLM drivers.
type Adapter interface {
	Chat(ctx context.Context, req Request) (*Response, error)
	Stream(ctx context.Context, req Request) iter.Seq2[Event, error]
	Capabilities() Capabilities
}

// ModelLister defines an optional interface for adapters capable of listing models.
type ModelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// Config provides configuration for constructing an Adapter.
type Config struct {
	Kind             Kind
	BaseURL          string
	Model            string
	APIKey           string
	Capabilities     *Capabilities
	AllowPrivateURLs bool
	HTTPClient       *http.Client
}

// Factory constructs an Adapter from a Config.
type Factory func(Config) (Adapter, error)

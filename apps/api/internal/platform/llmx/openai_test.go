package llmx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAI(t *testing.T) {
	t.Run("TestChatMapsRequestAndResponse", testOpenAIChatMapsRequestAndResponse)
	t.Run("TestStreamYieldsTextDeltasThenDone", testOpenAIStreamYieldsTextDeltasThenDone)
	t.Run("TestStreamToolCallDeltasAssembleOneToolCall", testOpenAIStreamToolCallDeltasAssembleOneToolCall)
	t.Run("TestErrorMapping", testOpenAIErrorMapping)
	t.Run("TestErrorBodyNeverContainsAPIKey", testOpenAIErrorBodyNeverContainsAPIKey)
	t.Run("TestNoRedirectFollow", testOpenAINoRedirectFollow)
	t.Run("TestCapabilitiesOverride", testOpenAICapabilitiesOverride)
}

func testOpenAIChatMapsRequestAndResponse(t *testing.T) {
	var gotBody map[string]any
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(b, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-1",
			"object": "chat.completion",
			"created": 1700000000,
			"model": "gpt-4o-mini",
			"choices": [{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Hello back",
					"tool_calls": [{
						"id": "call_abc",
						"type": "function",
						"function": {
							"name": "lookup",
							"arguments": "{\"q\":\"test\"}"
						}
					}]
				},
				"finish_reason": "tool_calls"
			}],
			"usage": {
				"prompt_tokens": 12,
				"completion_tokens": 7,
				"total_tokens": 19
			}
		}`))
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindOpenAICompat,
		BaseURL:          srv.URL,
		Model:            "gpt-4o-mini",
		APIKey:           "sk-test-openai-key",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	temp := 0.25
	resp, err := adapter.Chat(context.Background(), Request{
		Model: "gpt-4o-mini",
		Messages: []Message{
			{Role: RoleSystem, Parts: []Part{{Text: "Be concise"}}},
			{
				Role: RoleUser,
				Parts: []Part{
					{Text: "What is in this image?"},
					{Image: &Image{MIME: "image/png", Data: []byte{0x89, 0x50, 0x4e, 0x47}}},
				},
			},
		},
		Tools: []Tool{
			{
				Name:        "lookup",
				Description: "Look up item",
				Schema:      json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`),
			},
		},
		ResponseSchema: json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}}}`),
		MaxTokens:      256,
		Temperature:    &temp,
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	if gotAuth != "Bearer sk-test-openai-key" {
		t.Errorf("Authorization = %q, want Bearer sk-test-openai-key", gotAuth)
	}
	if gotBody["model"] != "gpt-4o-mini" {
		t.Errorf("model = %v, want gpt-4o-mini", gotBody["model"])
	}
	msgs, ok := gotBody["messages"].([]any)
	if !ok || len(msgs) != 2 {
		t.Fatalf("messages = %#v, want len 2", gotBody["messages"])
	}
	firstMsg := msgs[0].(map[string]any)
	if firstMsg["role"] != "system" || firstMsg["content"] != "Be concise" {
		t.Errorf("system message = %#v", firstMsg)
	}
	secondMsg := msgs[1].(map[string]any)
	contentParts, ok := secondMsg["content"].([]any)
	if !ok || len(contentParts) != 2 {
		t.Fatalf("user content parts = %#v, want len 2", secondMsg["content"])
	}
	imgPart := contentParts[1].(map[string]any)
	if imgPart["type"] != "image_url" {
		t.Errorf("second part type = %v, want image_url", imgPart["type"])
	}
	tools, ok := gotBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want len 1", gotBody["tools"])
	}
	if gotBody["response_format"] == nil {
		t.Errorf("expected response_format to be set in request body")
	}

	if len(resp.Message.Parts) != 1 || resp.Message.Parts[0].Text != "Hello back" {
		t.Errorf("resp.Message.Parts = %#v, want Hello back", resp.Message.Parts)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].ID != "call_abc" || resp.Message.ToolCalls[0].Name != "lookup" || string(resp.Message.ToolCalls[0].Arguments) != `{"q":"test"}` {
		t.Errorf("resp.Message.ToolCalls = %#v", resp.Message.ToolCalls)
	}
	if resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 7 {
		t.Errorf("resp.Usage = %#v, want {12, 7}", resp.Usage)
	}
	if resp.StopReason != "tool_calls" {
		t.Errorf("resp.StopReason = %q, want tool_calls", resp.StopReason)
	}
}

func testOpenAIStreamYieldsTextDeltasThenDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hel"}}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"lo"}}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","choices":[],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", c)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindOpenAICompat,
		BaseURL:          srv.URL,
		Model:            "gpt-4o-mini",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var events []Event
	for ev, err := range adapter.Stream(context.Background(), Request{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "Hi"}}}},
	}) {
		if err != nil {
			t.Fatalf("stream error: %v", err)
		}
		events = append(events, ev)
	}

	if len(events) != 3 {
		t.Fatalf("events len = %d (%#v), want 3", len(events), events)
	}
	if events[0].Type != EventText || events[0].Text != "Hel" {
		t.Errorf("events[0] = %#v", events[0])
	}
	if events[1].Type != EventText || events[1].Text != "lo" {
		t.Errorf("events[1] = %#v", events[1])
	}
	if events[2].Type != EventDone || events[2].Usage == nil || events[2].Usage.InputTokens != 5 || events[2].Usage.OutputTokens != 2 {
		t.Errorf("events[2] = %#v", events[2])
	}
}

func testOpenAIStreamToolCallDeltasAssembleOneToolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":"}}]}}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Denpasar\"}"}}]}}]}`,
			`data: [DONE]`,
		}
		for _, c := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", c)
		}
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindOpenAICompat,
		BaseURL:          srv.URL,
		Model:            "gpt-4o-mini",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var toolEvents []Event
	for ev, err := range adapter.Stream(context.Background(), Request{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "weather"}}}},
	}) {
		if err != nil {
			t.Fatalf("stream error: %v", err)
		}
		if ev.Type == EventToolCall {
			toolEvents = append(toolEvents, ev)
		}
	}

	if len(toolEvents) != 1 || toolEvents[0].ToolCall == nil {
		t.Fatalf("toolEvents = %#v, want 1 tool call", toolEvents)
	}
	tc := toolEvents[0].ToolCall
	if tc.ID != "call_1" || tc.Name != "get_weather" || string(tc.Arguments) != `{"city":"Denpasar"}` {
		t.Errorf("assembled tool call = %#v", tc)
	}
}

func testOpenAIErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{401, ErrAuth},
		{404, ErrModelNotFound},
		{429, ErrRateLimit},
		{500, ErrUnavailable},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"error":{"message":"failed","type":"err"}}`))
		}))
		adapter, err := New(Config{
			Kind:             KindOpenAICompat,
			BaseURL:          srv.URL,
			Model:            "gpt-4o-mini",
			APIKey:           "sk-test",
			AllowPrivateURLs: true,
		})
		if err != nil {
			srv.Close()
			t.Fatalf("New error: %v", err)
		}
		_, err = adapter.Chat(context.Background(), Request{
			Model:    "gpt-4o-mini",
			Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
		})
		srv.Close()
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: err = %v, want %v", tc.status, err, tc.want)
		}
	}
}

func testOpenAIErrorBodyNeverContainsAPIKey(t *testing.T) {
	secret := "sk-super-secret-openai-9999"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"invalid key %s"}}`, secret)
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindOpenAICompat,
		BaseURL:          srv.URL,
		Model:            "gpt-4o-mini",
		APIKey:           secret,
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	_, err = adapter.Chat(context.Background(), Request{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error leaked secret: %v", err)
	}
}

func testOpenAINoRedirectFollow(t *testing.T) {
	followed := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		followed = true
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindOpenAICompat,
		BaseURL:          srv.URL,
		Model:            "gpt-4o-mini",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	_, err = adapter.Chat(context.Background(), Request{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
	})
	if err == nil {
		t.Fatalf("expected error on redirect, got nil")
	}
	if followed {
		t.Errorf("redirect was followed")
	}
}

func testOpenAICapabilitiesOverride(t *testing.T) {
	override := &Capabilities{
		Vision:           false,
		ToolCalling:      true,
		StructuredOutput: false,
		Streaming:        true,
	}
	adapter, err := New(Config{
		Kind:             KindOpenAICompat,
		APIKey:           "sk-test",
		Capabilities:     override,
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if got := adapter.Capabilities(); got != *override {
		t.Errorf("Capabilities() = %#v, want %#v", got, *override)
	}
}

package llmx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestAnthropic(t *testing.T) {
	t.Run("TestChatMapsRequestAndResponse", testAnthropicChatMapsRequestAndResponse)
	t.Run("TestStreamYieldsTextDeltasThenDone", testAnthropicStreamYieldsTextDeltasThenDone)
	t.Run("TestStreamToolCallDeltasAssembleOneToolCall", testAnthropicStreamToolCallDeltasAssembleOneToolCall)
	t.Run("TestErrorMapping", testAnthropicErrorMapping)
	t.Run("TestErrorBodyNeverContainsAPIKey", testAnthropicErrorBodyNeverContainsAPIKey)
	t.Run("TestNoRedirectFollow", testAnthropicNoRedirectFollow)
	t.Run("TestCapabilitiesOverride", testAnthropicCapabilitiesOverride)
	t.Run("TestListModels", testAnthropicListModels)
	t.Run("TestListModelsError", testAnthropicListModelsError)
}

func testAnthropicChatMapsRequestAndResponse(t *testing.T) {
	var gotBody map[string]any
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-Api-Key")
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(b, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "msg-1",
			"type": "message",
			"role": "assistant",
			"model": "claude-3-5-sonnet-20241022",
			"content": [
				{
					"type": "text",
					"text": "Hello back"
				},
				{
					"type": "tool_use",
					"id": "call_abc",
					"name": "lookup",
					"input": {"q":"test"}
				}
			],
			"stop_reason": "tool_use",
			"usage": {
				"input_tokens": 12,
				"output_tokens": 7
			}
		}`))
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindAnthropic,
		BaseURL:          srv.URL,
		Model:            "claude-3-5-sonnet-20241022",
		APIKey:           "sk-test-anthropic-key",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	temp := 0.25
	resp, err := adapter.Chat(context.Background(), Request{
		Model: "claude-3-5-sonnet-20241022",
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

	if gotAuth != "sk-test-anthropic-key" {
		t.Errorf("X-Api-Key = %q, want sk-test-anthropic-key", gotAuth)
	}
	if gotBody["model"] != "claude-3-5-sonnet-20241022" {
		t.Errorf("model = %v, want claude-3-5-sonnet-20241022", gotBody["model"])
	}

	// Verify system prompt in Anthropic request
	sysArr, ok := gotBody["system"].([]any)
	if !ok || len(sysArr) != 1 {
		t.Fatalf("system = %#v, want slice of len 1", gotBody["system"])
	}
	firstSys := sysArr[0].(map[string]any)
	if firstSys["text"] != "Be concise" {
		t.Errorf("system text = %#v, want 'Be concise'", firstSys["text"])
	}

	// Verify user messages and image
	msgs, ok := gotBody["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("messages = %#v, want len 1", gotBody["messages"])
	}
	userMsg := msgs[0].(map[string]any)
	if userMsg["role"] != "user" {
		t.Errorf("user message role = %v, want user", userMsg["role"])
	}
	contentParts, ok := userMsg["content"].([]any)
	if !ok || len(contentParts) != 2 {
		t.Fatalf("user content parts = %#v, want len 2", userMsg["content"])
	}
	txtPart := contentParts[0].(map[string]any)
	if txtPart["type"] != "text" || txtPart["text"] != "What is in this image?" {
		t.Errorf("first part = %#v", txtPart)
	}
	imgPart := contentParts[1].(map[string]any)
	if imgPart["type"] != "image" {
		t.Errorf("second part type = %v, want image", imgPart["type"])
	}
	src, ok := imgPart["source"].(map[string]any)
	if !ok || src["type"] != "base64" || src["media_type"] != "image/png" {
		t.Errorf("image source = %#v", src)
	}

	// Verify tools
	tools, ok := gotBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want len 1", gotBody["tools"])
	}
	tool0 := tools[0].(map[string]any)
	if tool0["name"] != "lookup" || tool0["description"] != "Look up item" {
		t.Errorf("tool = %#v", tool0)
	}

	// Verify response schema / output format
	outCfg, ok := gotBody["output_config"].(map[string]any)
	if !ok {
		t.Fatalf("expected output_config in request body: %#v", gotBody)
	}
	format, ok := outCfg["format"].(map[string]any)
	if !ok || format["type"] != "json_schema" {
		t.Errorf("expected format.type = json_schema in output_config: %#v", outCfg)
	}

	// Verify decoded response
	if len(resp.Message.Parts) != 1 || resp.Message.Parts[0].Text != "Hello back" {
		t.Errorf("resp.Message.Parts = %#v, want Hello back", resp.Message.Parts)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].ID != "call_abc" || resp.Message.ToolCalls[0].Name != "lookup" || string(resp.Message.ToolCalls[0].Arguments) != `{"q":"test"}` {
		t.Errorf("resp.Message.ToolCalls = %#v", resp.Message.ToolCalls)
	}
	if resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 7 {
		t.Errorf("resp.Usage = %#v, want {12, 7}", resp.Usage)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("resp.StopReason = %q, want tool_use", resp.StopReason)
	}
}

func testAnthropicStreamYieldsTextDeltasThenDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-3-5-sonnet-20241022\",\"usage\":{\"input_tokens\":5,\"output_tokens\":0}}}",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hel\"}}",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"lo\"}}",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":2}}",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}",
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
		Kind:             KindAnthropic,
		BaseURL:          srv.URL,
		Model:            "claude-3-5-sonnet-20241022",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var events []Event
	for ev, err := range adapter.Stream(context.Background(), Request{
		Model:    "claude-3-5-sonnet-20241022",
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

func testAnthropicStreamToolCallDeltasAssembleOneToolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-3-5-sonnet-20241022\",\"usage\":{\"input_tokens\":4,\"output_tokens\":0}}}",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"call_1\",\"name\":\"get_weather\"}}",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"city\\\":\"}}",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"\\\"Denpasar\\\"}\"}}",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":6}}",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}",
		}
		for _, c := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", c)
		}
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindAnthropic,
		BaseURL:          srv.URL,
		Model:            "claude-3-5-sonnet-20241022",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var toolEvents []Event
	for ev, err := range adapter.Stream(context.Background(), Request{
		Model:    "claude-3-5-sonnet-20241022",
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

func testAnthropicErrorMapping(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"failed"}}`))
		}))
		adapter, err := New(Config{
			Kind:             KindAnthropic,
			BaseURL:          srv.URL,
			Model:            "claude-3-5-sonnet-20241022",
			APIKey:           "sk-test",
			AllowPrivateURLs: true,
		})
		if err != nil {
			srv.Close()
			t.Fatalf("New error: %v", err)
		}
		_, err = adapter.Chat(context.Background(), Request{
			Model:    "claude-3-5-sonnet-20241022",
			Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
		})
		srv.Close()
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: err = %v, want %v", tc.status, err, tc.want)
		}
	}
}

func testAnthropicErrorBodyNeverContainsAPIKey(t *testing.T) {
	secret := "sk-super-secret-anthropic-9999"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid key %s"}}`, secret)
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindAnthropic,
		BaseURL:          srv.URL,
		Model:            "claude-3-5-sonnet-20241022",
		APIKey:           secret,
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	_, err = adapter.Chat(context.Background(), Request{
		Model:    "claude-3-5-sonnet-20241022",
		Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error leaked secret: %v", err)
	}
}

func testAnthropicNoRedirectFollow(t *testing.T) {
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
		Kind:             KindAnthropic,
		BaseURL:          srv.URL,
		Model:            "claude-3-5-sonnet-20241022",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	_, err = adapter.Chat(context.Background(), Request{
		Model:    "claude-3-5-sonnet-20241022",
		Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
	})
	if err == nil {
		t.Fatalf("expected error on redirect, got nil")
	}
	if followed {
		t.Errorf("redirect was followed")
	}
}

func testAnthropicCapabilitiesOverride(t *testing.T) {
	override := &Capabilities{
		Vision:           false,
		ToolCalling:      true,
		StructuredOutput: false,
		Streaming:        true,
	}
	adapter, err := New(Config{
		Kind:             KindAnthropic,
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

func testAnthropicListModels(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-Api-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"id": "claude-3-5-sonnet-20241022",
					"display_name": "Claude 3.5 Sonnet",
					"type": "model",
					"created_at": "2024-10-22T00:00:00Z"
				},
				{
					"id": "claude-3-opus-20240229",
					"display_name": "Claude 3 Opus",
					"type": "model",
					"created_at": "2024-02-29T00:00:00Z"
				},
				{
					"id": "claude-3-5-haiku-20241022",
					"display_name": "Claude 3.5 Haiku",
					"type": "model",
					"created_at": "2024-10-22T00:00:00Z"
				}
			],
			"has_more": false
		}`))
	}))
	defer srv.Close()

	cfg := Config{
		Kind:             KindAnthropic,
		BaseURL:          srv.URL,
		APIKey:           "sk-test-anthropic-key",
		AllowPrivateURLs: true,
	}

	models, err := ListModels(context.Background(), cfg)
	if err != nil {
		t.Fatalf("ListModels error: %v", err)
	}

	if gotAuth != "sk-test-anthropic-key" {
		t.Errorf("X-Api-Key = %q, want sk-test-anthropic-key", gotAuth)
	}

	want := []string{"claude-3-5-haiku-20241022", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229"}
	if !slices.Equal(models, want) {
		t.Errorf("ListModels = %v, want %v", models, want)
	}
}

func testAnthropicListModelsError(t *testing.T) {
	secret := "sk-super-secret-anthropic-key-999"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid key %s"}}`, secret)
	}))
	defer srv.Close()

	cfg := Config{
		Kind:             KindAnthropic,
		BaseURL:          srv.URL,
		APIKey:           secret,
		AllowPrivateURLs: true,
	}

	_, err := ListModels(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrAuth) {
		t.Errorf("err = %v, want ErrAuth", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error leaked secret key: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Errorf("error missing [REDACTED]: %v", err)
	}
}

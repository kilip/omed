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

func TestGemini(t *testing.T) {
	t.Run("TestChatMapsRequestAndResponse", testGeminiChatMapsRequestAndResponse)
	t.Run("TestStreamYieldsTextDeltasThenDone", testGeminiStreamYieldsTextDeltasThenDone)
	t.Run("TestStreamToolCallDeltasAssembleOneToolCall", testGeminiStreamToolCallDeltasAssembleOneToolCall)
	t.Run("TestErrorMapping", testGeminiErrorMapping)
	t.Run("TestErrorBodyNeverContainsAPIKey", testGeminiErrorBodyNeverContainsAPIKey)
	t.Run("TestNoRedirectFollow", testGeminiNoRedirectFollow)
	t.Run("TestCapabilitiesOverride", testGeminiCapabilitiesOverride)
}

func testGeminiChatMapsRequestAndResponse(t *testing.T) {
	var gotBody map[string]any
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("x-goog-api-key")
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(b, &gotBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {
					"role": "model",
					"parts": [
						{"text": "Hello back"},
						{
							"functionCall": {
								"id": "call_abc",
								"name": "lookup",
								"args": {"q": "test"}
							}
						}
					]
				},
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 12,
				"candidatesTokenCount": 7,
				"totalTokenCount": 19
			}
		}`))
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindGemini,
		BaseURL:          srv.URL,
		Model:            "gemini-2.5-flash",
		APIKey:           "sk-test-gemini-key",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	temp := 0.25
	resp, err := adapter.Chat(context.Background(), Request{
		Model: "gemini-2.5-flash",
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

	if gotAuth != "sk-test-gemini-key" {
		t.Errorf("x-goog-api-key = %q, want sk-test-gemini-key", gotAuth)
	}

	// Verify system instruction in Gemini request
	sysInst, ok := gotBody["systemInstruction"].(map[string]any)
	if !ok {
		t.Fatalf("systemInstruction missing or not map: %#v", gotBody["systemInstruction"])
	}
	sysParts, ok := sysInst["parts"].([]any)
	if !ok || len(sysParts) != 1 {
		t.Fatalf("systemInstruction parts = %#v, want len 1", sysInst["parts"])
	}
	firstSysPart := sysParts[0].(map[string]any)
	if firstSysPart["text"] != "Be concise" {
		t.Errorf("system text = %#v, want 'Be concise'", firstSysPart["text"])
	}

	// Verify contents
	contents, ok := gotBody["contents"].([]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents = %#v, want len 1", gotBody["contents"])
	}
	userContent := contents[0].(map[string]any)
	if userContent["role"] != "user" {
		t.Errorf("content role = %v, want user", userContent["role"])
	}
	userParts, ok := userContent["parts"].([]any)
	if !ok || len(userParts) != 2 {
		t.Fatalf("user parts = %#v, want len 2", userContent["parts"])
	}
	textPart := userParts[0].(map[string]any)
	if textPart["text"] != "What is in this image?" {
		t.Errorf("textPart = %#v", textPart)
	}
	imgPart := userParts[1].(map[string]any)
	if imgPart["inlineData"] == nil {
		t.Errorf("imgPart missing inlineData: %#v", imgPart)
	}

	// Verify tools
	tools, ok := gotBody["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want len 1", gotBody["tools"])
	}
	toolMap := tools[0].(map[string]any)
	decls, ok := toolMap["functionDeclarations"].([]any)
	if !ok || len(decls) != 1 {
		t.Fatalf("functionDeclarations = %#v, want len 1", toolMap["functionDeclarations"])
	}
	firstDecl := decls[0].(map[string]any)
	if firstDecl["name"] != "lookup" || firstDecl["description"] != "Look up item" {
		t.Errorf("firstDecl = %#v", firstDecl)
	}

	// Verify response schema in generationConfig
	genCfg, ok := gotBody["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig missing: %#v", gotBody)
	}
	if genCfg["responseSchema"] == nil && genCfg["responseJsonSchema"] == nil {
		t.Errorf("expected responseSchema or responseJsonSchema in generationConfig: %#v", genCfg)
	}
	if genCfg["responseMimeType"] != "application/json" {
		t.Errorf("responseMimeType = %v, want application/json", genCfg["responseMimeType"])
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
	if resp.StopReason != "STOP" {
		t.Errorf("resp.StopReason = %q, want STOP", resp.StopReason)
	}
}

func testGeminiStreamYieldsTextDeltasThenDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		chunks := []string{
			`data: {"candidates":[{"content":{"parts":[{"text":"Hel"}]}}]}`,
			`data: {"candidates":[{"content":{"parts":[{"text":"lo"}]}}]}`,
			`data: {"candidates":[{"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2}}`,
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
		Kind:             KindGemini,
		BaseURL:          srv.URL,
		Model:            "gemini-2.5-flash",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var events []Event
	for ev, err := range adapter.Stream(context.Background(), Request{
		Model:    "gemini-2.5-flash",
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

func testGeminiStreamToolCallDeltasAssembleOneToolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		chunks := []string{
			`data: {"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","id":"call_1"}}]}}]}`,
			`data: {"candidates":[{"content":{"parts":[{"functionCall":{"id":"call_1","args":{"city":"Denpasar"}}}]}}]}`,
			`data: {"candidates":[{"finishReason":"STOP"}]}`,
		}
		for _, c := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", c)
		}
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindGemini,
		BaseURL:          srv.URL,
		Model:            "gemini-2.5-flash",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	var toolEvents []Event
	for ev, err := range adapter.Stream(context.Background(), Request{
		Model:    "gemini-2.5-flash",
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

func testGeminiErrorMapping(t *testing.T) {
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
			_, _ = fmt.Fprintf(w, `{"error":{"code":%d,"message":"failed","status":"ERROR"}}`, tc.status)
		}))
		adapter, err := New(Config{
			Kind:             KindGemini,
			BaseURL:          srv.URL,
			Model:            "gemini-2.5-flash",
			APIKey:           "sk-test",
			AllowPrivateURLs: true,
		})
		if err != nil {
			srv.Close()
			t.Fatalf("New error: %v", err)
		}
		_, err = adapter.Chat(context.Background(), Request{
			Model:    "gemini-2.5-flash",
			Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
		})
		srv.Close()
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: err = %v, want %v", tc.status, err, tc.want)
		}
	}
}

func testGeminiErrorBodyNeverContainsAPIKey(t *testing.T) {
	secret := "sk-super-secret-gemini-9999"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"error":{"code":401,"message":"invalid key %s","status":"UNAUTHENTICATED"}}`, secret)
	}))
	defer srv.Close()

	adapter, err := New(Config{
		Kind:             KindGemini,
		BaseURL:          srv.URL,
		Model:            "gemini-2.5-flash",
		APIKey:           secret,
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	_, err = adapter.Chat(context.Background(), Request{
		Model:    "gemini-2.5-flash",
		Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("error leaked secret: %v", err)
	}
}

func testGeminiNoRedirectFollow(t *testing.T) {
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
		Kind:             KindGemini,
		BaseURL:          srv.URL,
		Model:            "gemini-2.5-flash",
		APIKey:           "sk-test",
		AllowPrivateURLs: true,
	})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	_, err = adapter.Chat(context.Background(), Request{
		Model:    "gemini-2.5-flash",
		Messages: []Message{{Role: RoleUser, Parts: []Part{{Text: "hi"}}}},
	})
	if err == nil {
		t.Fatalf("expected error on redirect, got nil")
	}
	if followed {
		t.Errorf("redirect was followed")
	}
}

func testGeminiCapabilitiesOverride(t *testing.T) {
	override := &Capabilities{
		Vision:           false,
		ToolCalling:      true,
		StructuredOutput: false,
		Streaming:        true,
	}
	adapter, err := New(Config{
		Kind:             KindGemini,
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

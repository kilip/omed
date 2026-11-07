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

func TestGemini(t *testing.T) {
	t.Run("TestChatMapsRequestAndResponse", testGeminiChatMapsRequestAndResponse)
	t.Run("TestStreamYieldsTextDeltasThenDone", testGeminiStreamYieldsTextDeltasThenDone)
	t.Run("TestStreamToolCallDeltasAssembleOneToolCall", testGeminiStreamToolCallDeltasAssembleOneToolCall)
	t.Run("TestErrorMapping", testGeminiErrorMapping)
	t.Run("TestErrorBodyNeverContainsAPIKey", testGeminiErrorBodyNeverContainsAPIKey)
	t.Run("TestNoRedirectFollow", testGeminiNoRedirectFollow)
	t.Run("TestCapabilitiesOverride", testGeminiCapabilitiesOverride)
	t.Run("TestListModels", testGeminiListModels)
	t.Run("TestListModelsError", testGeminiListModelsError)
	t.Run("TestDocumentPart", testGeminiDocumentPart)
	t.Run("TestToolResponseSerialization", testGeminiToolResponseSerialization)
	t.Run("TestToolResponseFallbackLookup", testGeminiToolResponseFallbackLookup)
	t.Run("TestToolResponseConsecutiveBatching", testGeminiToolResponseConsecutiveBatching)
	t.Run("TestBuildParamsFunctionResponseName", testGeminiBuildParamsFunctionResponseName)
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

func testGeminiListModels(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("x-goog-api-key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"models": [
				{
					"name": "models/gemini-2.5-flash",
					"displayName": "Gemini 2.5 Flash"
				},
				{
					"name": "models/gemini-1.5-pro",
					"displayName": "Gemini 1.5 Pro"
				},
				{
					"name": "gemini-1.0-pro",
					"displayName": "Gemini 1.0 Pro"
				}
			]
		}`))
	}))
	defer srv.Close()

	cfg := Config{
		Kind:             KindGemini,
		BaseURL:          srv.URL,
		APIKey:           "sk-test-gemini-key",
		AllowPrivateURLs: true,
	}

	models, err := ListModels(context.Background(), cfg)
	if err != nil {
		t.Fatalf("ListModels error: %v", err)
	}

	if gotAuth != "sk-test-gemini-key" {
		t.Errorf("x-goog-api-key = %q, want sk-test-gemini-key", gotAuth)
	}

	want := []string{"gemini-1.0-pro", "gemini-1.5-pro", "gemini-2.5-flash"}
	if !slices.Equal(models, want) {
		t.Errorf("ListModels = %v, want %v", models, want)
	}
}

func testGeminiListModelsError(t *testing.T) {
	secret := "sk-super-secret-gemini-key-999"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprintf(w, `{"error":{"code":401,"message":"API key not valid: %s","status":"UNAUTHENTICATED"}}`, secret)
	}))
	defer srv.Close()

	cfg := Config{
		Kind:             KindGemini,
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

func testGeminiDocumentPart(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
						{"text": "Document received"}
					]
				},
				"finishReason": "STOP"
			}],
			"usageMetadata": {
				"promptTokenCount": 50,
				"candidatesTokenCount": 5,
				"totalTokenCount": 55
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

	pdfData := []byte("%PDF-1.4 gemini document test")
	_, err = adapter.Chat(context.Background(), Request{
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{
				Role: RoleUser,
				Parts: []Part{
					{Text: "Please analyze this doc"},
					{Document: &Document{MIME: "application/pdf", Data: pdfData}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	contents, ok := gotBody["contents"].([]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents = %#v, want len 1", gotBody["contents"])
	}
	userContent := contents[0].(map[string]any)
	userParts, ok := userContent["parts"].([]any)
	if !ok || len(userParts) != 2 {
		t.Fatalf("user parts = %#v, want len 2", userContent["parts"])
	}

	docPart, ok := userParts[1].(map[string]any)
	if !ok {
		t.Fatalf("docPart not a map: %#v", userParts[1])
	}
	inlineData, ok := docPart["inlineData"].(map[string]any)
	if !ok {
		t.Fatalf("inlineData missing or not map: %#v", docPart)
	}
	if inlineData["mimeType"] != "application/pdf" {
		t.Errorf("mimeType = %v, want application/pdf", inlineData["mimeType"])
	}
	expectedB64 := "JVBERi0xLjQgZ2VtaW5pIGRvY3VtZW50IHRlc3Q="
	if inlineData["data"] != expectedB64 {
		t.Errorf("data = %v, want %v", inlineData["data"], expectedB64)
	}
}

func testGeminiToolResponseSerialization(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					"parts": [{"text": "Tool output acknowledged"}]
				},
				"finishReason": "STOP"
			}]
		}`))
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
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{Role: RoleUser, Parts: []Part{{Text: "What is the weather?"}}},
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "call_weather_1", Name: "get_weather", Arguments: json.RawMessage(`{"city":"Jakarta"}`)},
				},
			},
			{
				Role:       RoleTool,
				ToolCallID: "call_weather_1",
				ToolName:   "get_weather",
				Parts:      []Part{{Text: `{"temperature":30,"condition":"sunny"}`}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	contents, ok := gotBody["contents"].([]any)
	if !ok || len(contents) != 3 {
		t.Fatalf("contents = %#v, want len 3", gotBody["contents"])
	}

	// Turn 0: user text
	userTurn := contents[0].(map[string]any)
	if userTurn["role"] != "user" {
		t.Errorf("contents[0].role = %v, want user", userTurn["role"])
	}

	// Turn 1: model tool call
	modelTurn := contents[1].(map[string]any)
	if modelTurn["role"] != "model" {
		t.Errorf("contents[1].role = %v, want model", modelTurn["role"])
	}

	// Turn 2: tool response
	toolTurn := contents[2].(map[string]any)
	if toolTurn["role"] != "user" {
		t.Errorf("contents[2].role = %v, want user", toolTurn["role"])
	}
	parts, ok := toolTurn["parts"].([]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("toolTurn parts = %#v, want len 1", toolTurn["parts"])
	}
	partMap, ok := parts[0].(map[string]any)
	if !ok {
		t.Fatalf("parts[0] not map: %#v", parts[0])
	}
	fnResp, ok := partMap["functionResponse"].(map[string]any)
	if !ok {
		t.Fatalf("part missing functionResponse: %#v", partMap)
	}
	if fnResp["name"] != "get_weather" {
		t.Errorf("functionResponse.name = %v, want get_weather", fnResp["name"])
	}
	if fnResp["id"] != "call_weather_1" {
		t.Errorf("functionResponse.id = %v, want call_weather_1", fnResp["id"])
	}
	respMap, ok := fnResp["response"].(map[string]any)
	if !ok || respMap["condition"] != "sunny" {
		t.Errorf("functionResponse.response = %#v, want condition=sunny", fnResp["response"])
	}
}

func testGeminiToolResponseFallbackLookup(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					"parts": [{"text": "Analysis complete"}]
				},
				"finishReason": "STOP"
			}]
		}`))
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

	// ToolName is intentionally empty on RoleTool message to test fallback lookup
	_, err = adapter.Chat(context.Background(), Request{
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{Role: RoleUser, Parts: []Part{{Text: "Analyze stock"}}},
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "call_stock_99", Name: "query_financials", Arguments: json.RawMessage(`{"ticker":"BBCA"}`)},
				},
			},
			{
				Role:       RoleTool,
				ToolCallID: "call_stock_99",
				ToolName:   "", // Left empty to verify fallback lookup
				Parts:      []Part{{Text: `{"price":10000}`}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	contents, ok := gotBody["contents"].([]any)
	if !ok || len(contents) != 3 {
		t.Fatalf("contents = %#v, want len 3", gotBody["contents"])
	}
	toolTurn := contents[2].(map[string]any)
	parts, ok := toolTurn["parts"].([]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("toolTurn parts = %#v, want len 1", toolTurn["parts"])
	}
	partMap := parts[0].(map[string]any)
	fnResp, ok := partMap["functionResponse"].(map[string]any)
	if !ok {
		t.Fatalf("missing functionResponse: %#v", partMap)
	}
	if fnResp["name"] != "query_financials" {
		t.Errorf("fallback functionResponse.name = %v, want query_financials", fnResp["name"])
	}
	if fnResp["id"] != "call_stock_99" {
		t.Errorf("functionResponse.id = %v, want call_stock_99", fnResp["id"])
	}
}

func testGeminiToolResponseConsecutiveBatching(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					"parts": [{"text": "All tools processed"}]
				},
				"finishReason": "STOP"
			}]
		}`))
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
		Model: "gemini-2.5-flash",
		Messages: []Message{
			{Role: RoleUser, Parts: []Part{{Text: "Get user and orders"}}},
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "call_u1", Name: "fetch_user", Arguments: json.RawMessage(`{"id":1}`)},
					{ID: "call_o2", Name: "fetch_orders", Arguments: json.RawMessage(`{"user_id":1}`)},
				},
			},
			{
				Role:       RoleTool,
				ToolCallID: "call_u1",
				ToolName:   "fetch_user",
				Parts:      []Part{{Text: `{"name":"Alice"}`}},
			},
			{
				Role:       RoleTool,
				ToolCallID: "call_o2",
				ToolName:   "fetch_orders",
				Parts:      []Part{{Text: `{"orders":[101,102]}`}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	contents, ok := gotBody["contents"].([]any)
	if !ok || len(contents) != 3 {
		t.Fatalf("contents length = %d, want 3 (user, model, and single batched user turn for tools)", len(contents))
	}

	toolContent := contents[2].(map[string]any)
	if toolContent["role"] != "user" {
		t.Errorf("batched tool turn role = %v, want user", toolContent["role"])
	}
	parts, ok := toolContent["parts"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("batched tool parts len = %d, want 2", len(parts))
	}

	part1 := parts[0].(map[string]any)["functionResponse"].(map[string]any)
	if part1["name"] != "fetch_user" || part1["id"] != "call_u1" {
		t.Errorf("part1 = %#v, want fetch_user/call_u1", part1)
	}

	part2 := parts[1].(map[string]any)["functionResponse"].(map[string]any)
	if part2["name"] != "fetch_orders" || part2["id"] != "call_o2" {
		t.Errorf("part2 = %#v, want fetch_orders/call_o2", part2)
	}
}

func testGeminiBuildParamsFunctionResponseName(t *testing.T) {
	a := &geminiAdapter{model: "gemini-2.5-flash"}
	_, contents, _ := a.buildParams(Request{
		Messages: []Message{
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "call_x", Name: "my_tool"},
				},
			},
			{
				Role:       RoleTool,
				ToolCallID: "call_x",
				// ToolName is blank, should resolve from call_x
				Parts: []Part{{Text: "raw plain output"}},
			},
		},
	})

	if len(contents) != 2 {
		t.Fatalf("contents len = %d, want 2", len(contents))
	}
	toolParts := contents[1].Parts
	if len(toolParts) != 1 || toolParts[0].FunctionResponse == nil {
		t.Fatalf("expected 1 FunctionResponse part, got %#v", toolParts)
	}
	fnResp := toolParts[0].FunctionResponse
	if fnResp.Name != "my_tool" {
		t.Errorf("FunctionResponse.Name = %q, want my_tool", fnResp.Name)
	}
	if fnResp.ID != "call_x" {
		t.Errorf("FunctionResponse.ID = %q, want call_x", fnResp.ID)
	}
	if fnResp.Response["output"] != "raw plain output" {
		t.Errorf("FunctionResponse.Response[output] = %v, want 'raw plain output'", fnResp.Response["output"])
	}
}

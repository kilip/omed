package llmx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"

	"google.golang.org/genai"
)

func init() {
	Register(KindGemini, newGemini)
}

type geminiAdapter struct {
	client       *genai.Client
	model        string
	apiKey       string
	capabilities Capabilities
}

func newGemini(cfg Config) (Adapter, error) {
	if cfg.BaseURL != "" {
		if err := ValidateBaseURL(cfg.BaseURL, cfg.AllowPrivateURLs); err != nil {
			return nil, err
		}
	}

	client := cfg.HTTPClient
	if client == nil {
		client = newHTTPClient(cfg.AllowPrivateURLs)
	}

	genClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:     cfg.APIKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: client,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: cfg.BaseURL,
		},
	})
	if err != nil {
		return nil, err
	}

	caps := MergeCapabilities(DefaultCapabilities(KindGemini), cfg.Capabilities)

	return &geminiAdapter{
		client:       genClient,
		model:        cfg.Model,
		apiKey:       cfg.APIKey,
		capabilities: caps,
	}, nil
}

func (a *geminiAdapter) Capabilities() Capabilities {
	return a.capabilities
}

func (a *geminiAdapter) wrapError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		body := []byte(apiErr.Message)
		if len(body) == 0 {
			body = []byte(apiErr.Error())
		}
		return MapHTTPError(apiErr.Code, body, a.apiKey)
	}
	var apiErrPtr *genai.APIError
	if errors.As(err, &apiErrPtr) {
		body := []byte(apiErrPtr.Message)
		if len(body) == 0 {
			body = []byte(apiErrPtr.Error())
		}
		return MapHTTPError(apiErrPtr.Code, body, a.apiKey)
	}
	if a.apiKey != "" && strings.Contains(err.Error(), a.apiKey) {
		return errors.New(strings.ReplaceAll(err.Error(), a.apiKey, "[REDACTED]"))
	}
	return err
}

func (a *geminiAdapter) buildParams(req Request) (string, []*genai.Content, *genai.GenerateContentConfig) {
	model := req.Model
	if model == "" {
		model = a.model
	}

	genConfig := &genai.GenerateContentConfig{}

	if req.MaxTokens > 0 {
		genConfig.MaxOutputTokens = int32(req.MaxTokens)
	}
	if req.Temperature != nil {
		t := float32(*req.Temperature)
		genConfig.Temperature = &t
	}

	if len(req.ResponseSchema) > 0 {
		genConfig.ResponseMIMEType = "application/json"
		var schemaMap map[string]any
		if err := json.Unmarshal(req.ResponseSchema, &schemaMap); err == nil {
			genConfig.ResponseJsonSchema = schemaMap
		}
	}

	if len(req.Tools) > 0 {
		var decls []*genai.FunctionDeclaration
		for _, t := range req.Tools {
			decl := &genai.FunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
			}
			if len(t.Schema) > 0 {
				var paramsMap map[string]any
				if err := json.Unmarshal(t.Schema, &paramsMap); err == nil {
					decl.ParametersJsonSchema = paramsMap
				}
			}
			decls = append(decls, decl)
		}
		genConfig.Tools = []*genai.Tool{
			{FunctionDeclarations: decls},
		}
	}

	var systemParts []*genai.Part
	var contents []*genai.Content

	for _, m := range req.Messages {
		switch m.Role {
		case RoleSystem:
			for _, p := range m.Parts {
				if p.Text != "" {
					systemParts = append(systemParts, genai.NewPartFromText(p.Text))
				}
			}

		case RoleUser:
			var parts []*genai.Part
			for _, p := range m.Parts {
				if p.Text != "" {
					parts = append(parts, genai.NewPartFromText(p.Text))
				}
				if p.Image != nil {
					parts = append(parts, genai.NewPartFromBytes(p.Image.Data, p.Image.MIME))
				}
			}
			if len(parts) > 0 {
				contents = append(contents, &genai.Content{
					Role:  "user",
					Parts: parts,
				})
			}

		case RoleAssistant:
			var parts []*genai.Part
			for _, p := range m.Parts {
				if p.Text != "" {
					parts = append(parts, genai.NewPartFromText(p.Text))
				}
			}
			for _, tc := range m.ToolCalls {
				var argsMap map[string]any
				if len(tc.Arguments) > 0 {
					_ = json.Unmarshal(tc.Arguments, &argsMap)
				}
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						ID:   tc.ID,
						Name: tc.Name,
						Args: argsMap,
					},
				})
			}
			if len(parts) > 0 {
				contents = append(contents, &genai.Content{
					Role:  "model",
					Parts: parts,
				})
			}

		case RoleTool:
			var sb strings.Builder
			for _, p := range m.Parts {
				sb.WriteString(p.Text)
			}
			var respMap map[string]any
			if err := json.Unmarshal([]byte(sb.String()), &respMap); err != nil {
				respMap = map[string]any{"output": sb.String()}
			}
			contents = append(contents, &genai.Content{
				Role: "user",
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							ID:       m.ToolCallID,
							Response: respMap,
						},
					},
				},
			})
		}
	}

	if len(systemParts) > 0 {
		genConfig.SystemInstruction = &genai.Content{
			Parts: systemParts,
		}
	}

	return model, contents, genConfig
}

func (a *geminiAdapter) Chat(ctx context.Context, req Request) (*Response, error) {
	model, contents, genConfig := a.buildParams(req)

	resp, err := a.client.Models.GenerateContent(ctx, model, contents, genConfig)
	if err != nil {
		return nil, a.wrapError(err)
	}

	res := Response{
		Message: Message{
			Role: RoleAssistant,
		},
	}

	if resp.UsageMetadata != nil {
		res.Usage = Usage{
			InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
			OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		}
	}

	if len(resp.Candidates) > 0 {
		cand := resp.Candidates[0]
		res.StopReason = string(cand.FinishReason)
		if cand.Content != nil {
			for _, p := range cand.Content.Parts {
				if p.Text != "" {
					res.Message.Parts = append(res.Message.Parts, Part{Text: p.Text})
				}
				if p.FunctionCall != nil {
					var rawArgs []byte
					if p.FunctionCall.Args != nil {
						rawArgs, _ = json.Marshal(p.FunctionCall.Args)
					} else {
						rawArgs = []byte("{}")
					}
					res.Message.ToolCalls = append(res.Message.ToolCalls, ToolCall{
						ID:        p.FunctionCall.ID,
						Name:      p.FunctionCall.Name,
						Arguments: json.RawMessage(rawArgs),
					})
				}
			}
		}
	}

	return &res, nil
}

func (a *geminiAdapter) Stream(ctx context.Context, req Request) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		model, contents, genConfig := a.buildParams(req)

		type toolCallAccum struct {
			id   string
			name string
			args map[string]any
		}
		var toolCallOrder []string
		toolCalls := make(map[string]*toolCallAccum)
		var usage *Usage

		for resp, err := range a.client.Models.GenerateContentStream(ctx, model, contents, genConfig) {
			if err != nil {
				yield(Event{}, a.wrapError(err))
				return
			}
			if resp == nil {
				continue
			}

			if resp.UsageMetadata != nil {
				usage = &Usage{
					InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
					OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
				}
			}

			for _, cand := range resp.Candidates {
				if cand.Content == nil {
					continue
				}
				for _, part := range cand.Content.Parts {
					if part.Text != "" {
						if !yield(Event{Type: EventText, Text: part.Text}, nil) {
							return
						}
					}
					if part.FunctionCall != nil {
						fc := part.FunctionCall
						key := fc.ID
						if key == "" {
							key = fc.Name
						}
						if key == "" {
							key = fmt.Sprintf("call_%d", len(toolCallOrder))
						}
						accum, exists := toolCalls[key]
						if !exists {
							accum = &toolCallAccum{
								args: make(map[string]any),
							}
							toolCalls[key] = accum
							toolCallOrder = append(toolCallOrder, key)
						}
						if fc.ID != "" {
							accum.id = fc.ID
						}
						if fc.Name != "" {
							accum.name = fc.Name
						}
						for k, v := range fc.Args {
							accum.args[k] = v
						}
					}
				}
			}
		}

		for _, key := range toolCallOrder {
			accum := toolCalls[key]
			var rawArgs []byte
			if len(accum.args) > 0 {
				rawArgs, _ = json.Marshal(accum.args)
			} else {
				rawArgs = []byte("{}")
			}
			tc := &ToolCall{
				ID:        accum.id,
				Name:      accum.name,
				Arguments: json.RawMessage(rawArgs),
			}
			if !yield(Event{Type: EventToolCall, ToolCall: tc}, nil) {
				return
			}
		}

		yield(Event{
			Type:  EventDone,
			Usage: usage,
		}, nil)
	}
}

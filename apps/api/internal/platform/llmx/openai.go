package llmx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
	"github.com/openai/openai-go/shared/constant"
)

func init() {
	Register(KindOpenAICompat, newOpenAI)
}

type openAIAdapter struct {
	client       openai.Client
	model        string
	apiKey       string
	capabilities Capabilities
}

func newOpenAI(cfg Config) (Adapter, error) {
	if cfg.BaseURL != "" {
		if err := ValidateBaseURL(cfg.BaseURL, cfg.AllowPrivateURLs); err != nil {
			return nil, err
		}
	}

	client := cfg.HTTPClient
	if client == nil {
		client = newHTTPClient(cfg.AllowPrivateURLs)
	}

	opts := []option.RequestOption{
		option.WithHTTPClient(client),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}

	caps := MergeCapabilities(DefaultCapabilities(KindOpenAICompat), cfg.Capabilities)

	return &openAIAdapter{
		client:       openai.NewClient(opts...),
		model:        cfg.Model,
		apiKey:       cfg.APIKey,
		capabilities: caps,
	}, nil
}

func (a *openAIAdapter) Capabilities() Capabilities {
	return a.capabilities
}

func (a *openAIAdapter) wrapError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		body := []byte(apiErr.RawJSON())
		if len(body) == 0 && apiErr.Message != "" {
			body = []byte(apiErr.Message)
		}
		return MapHTTPError(apiErr.StatusCode, body, a.apiKey)
	}
	if a.apiKey != "" && strings.Contains(err.Error(), a.apiKey) {
		return errors.New(strings.ReplaceAll(err.Error(), a.apiKey, "[REDACTED]"))
	}
	return err
}

func (a *openAIAdapter) buildParams(req Request) openai.ChatCompletionNewParams {
	model := req.Model
	if model == "" {
		model = a.model
	}

	params := openai.ChatCompletionNewParams{
		Model: shared.ChatModel(model),
	}

	if req.MaxTokens > 0 {
		params.MaxTokens = param.NewOpt(int64(req.MaxTokens))
	}
	if req.Temperature != nil {
		params.Temperature = param.NewOpt(*req.Temperature)
	}

	for _, m := range req.Messages {
		switch m.Role {
		case RoleSystem:
			var sb strings.Builder
			for _, p := range m.Parts {
				sb.WriteString(p.Text)
			}
			params.Messages = append(params.Messages, openai.SystemMessage(sb.String()))

		case RoleUser:
			hasImage := false
			for _, p := range m.Parts {
				if p.Image != nil {
					hasImage = true
					break
				}
			}
			if hasImage || len(m.Parts) > 1 {
				var parts []openai.ChatCompletionContentPartUnionParam
				for _, p := range m.Parts {
					if p.Text != "" {
						parts = append(parts, openai.TextContentPart(p.Text))
					}
					if p.Image != nil {
						b64 := base64.StdEncoding.EncodeToString(p.Image.Data)
						dataURL := fmt.Sprintf("data:%s;base64,%s", p.Image.MIME, b64)
						parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
							URL: dataURL,
						}))
					}
				}
				params.Messages = append(params.Messages, openai.UserMessage(parts))
			} else if len(m.Parts) == 1 {
				params.Messages = append(params.Messages, openai.UserMessage(m.Parts[0].Text))
			} else {
				params.Messages = append(params.Messages, openai.UserMessage(""))
			}

		case RoleAssistant:
			var sb strings.Builder
			for _, p := range m.Parts {
				sb.WriteString(p.Text)
			}
			if len(m.ToolCalls) > 0 {
				var tcParams []openai.ChatCompletionMessageToolCallParam
				for _, tc := range m.ToolCalls {
					tcParams = append(tcParams, openai.ChatCompletionMessageToolCallParam{
						ID: tc.ID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tc.Name,
							Arguments: string(tc.Arguments),
						},
					})
				}
				assistantMsg := openai.ChatCompletionAssistantMessageParam{
					Role:      constant.Assistant("assistant"),
					ToolCalls: tcParams,
				}
				if sb.Len() > 0 {
					assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: param.NewOpt(sb.String()),
					}
				}
				params.Messages = append(params.Messages, openai.ChatCompletionMessageParamUnion{
					OfAssistant: &assistantMsg,
				})
			} else {
				params.Messages = append(params.Messages, openai.AssistantMessage(sb.String()))
			}

		case RoleTool:
			var sb strings.Builder
			for _, p := range m.Parts {
				sb.WriteString(p.Text)
			}
			params.Messages = append(params.Messages, openai.ToolMessage(sb.String(), m.ToolCallID))
		}
	}

	for _, t := range req.Tools {
		var parameters shared.FunctionParameters
		if len(t.Schema) > 0 {
			_ = json.Unmarshal(t.Schema, &parameters)
		}
		toolParam := openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:       t.Name,
				Parameters: parameters,
			},
		}
		if t.Description != "" {
			toolParam.Function.Description = param.NewOpt(t.Description)
		}
		params.Tools = append(params.Tools, toolParam)
	}

	if len(req.ResponseSchema) > 0 {
		var schema any
		if err := json.Unmarshal(req.ResponseSchema, &schema); err != nil {
			schema = req.ResponseSchema
		}
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:   "response",
					Schema: schema,
				},
			},
		}
	}

	return params
}

func (a *openAIAdapter) Chat(ctx context.Context, req Request) (*Response, error) {
	params := a.buildParams(req)
	cc, err := a.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, a.wrapError(err)
	}

	var res Response
	res.Usage = Usage{
		InputTokens:  int(cc.Usage.PromptTokens),
		OutputTokens: int(cc.Usage.CompletionTokens),
	}
	if len(cc.Choices) > 0 {
		choice := cc.Choices[0]
		res.StopReason = choice.FinishReason

		msg := Message{
			Role: RoleAssistant,
		}
		if choice.Message.Content != "" {
			msg.Parts = append(msg.Parts, Part{Text: choice.Message.Content})
		}
		for _, tc := range choice.Message.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: json.RawMessage(tc.Function.Arguments),
			})
		}
		res.Message = msg
	}
	return &res, nil
}

func (a *openAIAdapter) Stream(ctx context.Context, req Request) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		params := a.buildParams(req)
		params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		}

		stream := a.client.Chat.Completions.NewStreaming(ctx, params)
		defer func() { _ = stream.Close() }()

		type toolCallAccum struct {
			index int64
			id    string
			name  string
			args  strings.Builder
		}
		var toolCallOrder []int64
		toolCalls := make(map[int64]*toolCallAccum)
		var usage *Usage

		for stream.Next() {
			chunk := stream.Current()
			if chunk.Usage.TotalTokens > 0 || chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
				usage = &Usage{
					InputTokens:  int(chunk.Usage.PromptTokens),
					OutputTokens: int(chunk.Usage.CompletionTokens),
				}
			}

			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					if !yield(Event{Type: EventText, Text: choice.Delta.Content}, nil) {
						return
					}
				}
				for _, tc := range choice.Delta.ToolCalls {
					idx := tc.Index
					accum, exists := toolCalls[idx]
					if !exists {
						accum = &toolCallAccum{index: idx}
						toolCalls[idx] = accum
						toolCallOrder = append(toolCallOrder, idx)
					}
					if tc.ID != "" {
						accum.id = tc.ID
					}
					if tc.Function.Name != "" {
						accum.name = tc.Function.Name
					}
					if tc.Function.Arguments != "" {
						accum.args.WriteString(tc.Function.Arguments)
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			yield(Event{}, a.wrapError(err))
			return
		}

		for _, idx := range toolCallOrder {
			accum := toolCalls[idx]
			tc := &ToolCall{
				ID:        accum.id,
				Name:      accum.name,
				Arguments: json.RawMessage(accum.args.String()),
			}
			if !yield(Event{Type: EventToolCall, ToolCall: tc}, nil) {
				return
			}
		}

		yield(Event{Type: EventDone, Usage: usage}, nil)
	}
}

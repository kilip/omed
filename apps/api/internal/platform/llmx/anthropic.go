package llmx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"iter"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

func init() {
	Register(KindAnthropic, newAnthropic)
}

type anthropicAdapter struct {
	client       anthropic.Client
	model        string
	apiKey       string
	capabilities Capabilities
}

func newAnthropic(cfg Config) (Adapter, error) {
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

	caps := MergeCapabilities(DefaultCapabilities(KindAnthropic), cfg.Capabilities)

	return &anthropicAdapter{
		client:       anthropic.NewClient(opts...),
		model:        cfg.Model,
		apiKey:       cfg.APIKey,
		capabilities: caps,
	}, nil
}

func (a *anthropicAdapter) Capabilities() Capabilities {
	return a.capabilities
}

func (a *anthropicAdapter) wrapError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		body := []byte(apiErr.RawJSON())
		if len(body) == 0 && apiErr.Error() != "" {
			body = []byte(apiErr.Error())
		}
		return MapHTTPError(apiErr.StatusCode, body, a.apiKey)
	}
	if a.apiKey != "" && strings.Contains(err.Error(), a.apiKey) {
		return errors.New(strings.ReplaceAll(err.Error(), a.apiKey, "[REDACTED]"))
	}
	return err
}

func (a *anthropicAdapter) buildParams(req Request) anthropic.MessageNewParams {
	model := req.Model
	if model == "" {
		model = a.model
	}

	maxTokens := int64(req.MaxTokens)
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
	}

	if req.Temperature != nil {
		params.Temperature = param.NewOpt(*req.Temperature)
	}

	for _, m := range req.Messages {
		switch m.Role {
		case RoleSystem:
			for _, p := range m.Parts {
				if p.Text != "" {
					params.System = append(params.System, anthropic.TextBlockParam{
						Text: p.Text,
					})
				}
			}

		case RoleUser:
			var blocks []anthropic.ContentBlockParamUnion
			for _, p := range m.Parts {
				if p.Text != "" {
					blocks = append(blocks, anthropic.NewTextBlock(p.Text))
				}
				if p.Image != nil {
					b64 := base64.StdEncoding.EncodeToString(p.Image.Data)
					blocks = append(blocks, anthropic.NewImageBlockBase64(p.Image.MIME, b64))
				}
			}
			params.Messages = append(params.Messages, anthropic.NewUserMessage(blocks...))

		case RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			for _, p := range m.Parts {
				if p.Text != "" {
					blocks = append(blocks, anthropic.NewTextBlock(p.Text))
				}
			}
			for _, tc := range m.ToolCalls {
				var input any
				if len(tc.Arguments) > 0 {
					_ = json.Unmarshal(tc.Arguments, &input)
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
			}
			params.Messages = append(params.Messages, anthropic.NewAssistantMessage(blocks...))

		case RoleTool:
			var sb strings.Builder
			for _, p := range m.Parts {
				sb.WriteString(p.Text)
			}
			params.Messages = append(params.Messages, anthropic.NewUserMessage(
				anthropic.NewToolResultBlock(m.ToolCallID, sb.String(), false),
			))
		}
	}

	for _, t := range req.Tools {
		var inputSchema anthropic.ToolInputSchemaParam
		if len(t.Schema) > 0 {
			_ = json.Unmarshal(t.Schema, &inputSchema)
		}
		tp := anthropic.ToolParam{
			Name:        t.Name,
			InputSchema: inputSchema,
		}
		if t.Description != "" {
			tp.Description = param.NewOpt(t.Description)
		}
		params.Tools = append(params.Tools, anthropic.ToolUnionParam{OfTool: &tp})
	}

	if len(req.ResponseSchema) > 0 {
		var schema map[string]any
		if err := json.Unmarshal(req.ResponseSchema, &schema); err == nil {
			params.OutputConfig = anthropic.OutputConfigParam{
				Format: anthropic.JSONOutputFormatParam{
					Schema: schema,
				},
			}
		}
	}

	return params
}

func (a *anthropicAdapter) Chat(ctx context.Context, req Request) (*Response, error) {
	params := a.buildParams(req)
	msg, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return nil, a.wrapError(err)
	}

	var res Response
	res.Usage = Usage{
		InputTokens:  int(msg.Usage.InputTokens),
		OutputTokens: int(msg.Usage.OutputTokens),
	}
	res.StopReason = string(msg.StopReason)

	resMsg := Message{
		Role: RoleAssistant,
	}
	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				resMsg.Parts = append(resMsg.Parts, Part{Text: block.Text})
			}
		case "tool_use":
			rawArgs := block.Input
			if len(rawArgs) == 0 {
				rawArgs = json.RawMessage("{}")
			}
			resMsg.ToolCalls = append(resMsg.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: rawArgs,
			})
		}
	}
	res.Message = resMsg

	return &res, nil
}

func (a *anthropicAdapter) Stream(ctx context.Context, req Request) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		params := a.buildParams(req)
		stream := a.client.Messages.NewStreaming(ctx, params)
		defer func() { _ = stream.Close() }()

		type toolCallAccum struct {
			index int64
			id    string
			name  string
			args  strings.Builder
		}
		var toolCallOrder []int64
		toolCalls := make(map[int64]*toolCallAccum)
		var usage Usage

		for stream.Next() {
			evt := stream.Current()
			switch evt.Type {
			case "message_start":
				if evt.Message.Usage.InputTokens > 0 {
					usage.InputTokens = int(evt.Message.Usage.InputTokens)
				}
				if evt.Message.Usage.OutputTokens > 0 {
					usage.OutputTokens = int(evt.Message.Usage.OutputTokens)
				}

			case "content_block_start":
				if evt.ContentBlock.Type == "tool_use" {
					idx := evt.Index
					accum, exists := toolCalls[idx]
					if !exists {
						accum = &toolCallAccum{index: idx}
						toolCalls[idx] = accum
						toolCallOrder = append(toolCallOrder, idx)
					}
					if evt.ContentBlock.ID != "" {
						accum.id = evt.ContentBlock.ID
					}
					if evt.ContentBlock.Name != "" {
						accum.name = evt.ContentBlock.Name
					}
				}

			case "content_block_delta":
				switch evt.Delta.Type {
				case "text_delta":
					if evt.Delta.Text != "" {
						if !yield(Event{Type: EventText, Text: evt.Delta.Text}, nil) {
							return
						}
					}
				case "input_json_delta":
					if evt.Delta.PartialJSON != "" {
						idx := evt.Index
						accum, exists := toolCalls[idx]
						if !exists {
							accum = &toolCallAccum{index: idx}
							toolCalls[idx] = accum
							toolCallOrder = append(toolCallOrder, idx)
						}
						accum.args.WriteString(evt.Delta.PartialJSON)
					}
				}

			case "message_delta":
				if evt.Usage.OutputTokens > 0 {
					usage.OutputTokens = int(evt.Usage.OutputTokens)
				}
				if evt.Usage.InputTokens > 0 {
					usage.InputTokens = int(evt.Usage.InputTokens)
				}
			}
		}

		if err := stream.Err(); err != nil {
			yield(Event{}, a.wrapError(err))
			return
		}

		for _, idx := range toolCallOrder {
			accum := toolCalls[idx]
			args := accum.args.String()
			if args == "" {
				args = "{}"
			}
			tc := &ToolCall{
				ID:        accum.id,
				Name:      accum.name,
				Arguments: json.RawMessage(args),
			}
			if !yield(Event{Type: EventToolCall, ToolCall: tc}, nil) {
				return
			}
		}

		yield(Event{
			Type:  EventDone,
			Usage: &usage,
		}, nil)
	}
}

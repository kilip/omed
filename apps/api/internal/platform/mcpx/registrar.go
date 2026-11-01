package mcpx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/authz"
)

// Config configures the MCP registrar.
type Config struct {
	Authorizer     *authz.Authorizer
	Resolver       authz.MembershipResolver
	UserID         func(*mcp.CallToolRequest) (int64, bool)
	Logger         *slog.Logger
	MaxResultBytes int
}

// Registrar manages tool registration and authorization for the MCP server.
type Registrar struct {
	cfg          Config
	server       *mcp.Server
	knownObjects map[string]bool
}

func defaultUserID(req *mcp.CallToolRequest) (int64, bool) {
	if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil || req.Extra.TokenInfo.UserID == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(req.Extra.TokenInfo.UserID, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// NewRegistrar creates a new tool registrar with the given configuration.
func NewRegistrar(cfg Config) *Registrar {
	if cfg.UserID == nil {
		cfg.UserID = defaultUserID
	}
	if cfg.MaxResultBytes <= 0 {
		cfg.MaxResultBytes = DefaultMaxResultBytes
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	knownObjects := make(map[string]bool)
	if cfg.Authorizer != nil {
		for _, role := range cfg.Authorizer.Roles() {
			for _, perm := range role.Permissions {
				if perm.Object != "" && !strings.Contains(perm.Object, "*") {
					knownObjects[perm.Object] = true
				}
			}
		}
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "omed",
		Version: "1.0.0",
	}, nil)

	return &Registrar{
		cfg:          cfg,
		server:       server,
		knownObjects: knownObjects,
	}
}

// Server returns the underlying MCP server instance.
func (r *Registrar) Server() *mcp.Server {
	return r.server
}

func (r *Registrar) validateSpec(spec ToolSpec) {
	if len(spec.Permissions) == 0 {
		panic(fmt.Sprintf("mcpx: tool %q: must declare at least one permission", spec.Name))
	}
	for _, perm := range spec.Permissions {
		if !r.knownObjects[perm.Object] {
			panic(fmt.Sprintf("mcpx: tool %q: unknown permission object %q", spec.Name, perm.Object))
		}
		mod, _, _ := strings.Cut(perm.Object, ":")
		if mod != "" && !strings.HasPrefix(spec.Name, mod+"_") {
			panic(fmt.Sprintf("mcpx: tool %q: name must start with module prefix %q", spec.Name, mod+"_"))
		}
	}
}

// AddTool registers a workspace-scoped tool with permission enforcement, error mapping, and result size cap.
func AddTool[In WorkspaceScoped, Out any](r *Registrar, spec ToolSpec, h func(ctx context.Context, in In) (Out, error)) {
	r.validateSpec(spec)

	tool := &mcp.Tool{
		Name:        spec.Name,
		Description: spec.Description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: spec.ReadOnly,
		},
	}

	mcp.AddTool[In, Out](r.server, tool, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var zero Out

		wid := in.GetWorkspaceID()
		if wid <= 0 {
			return nil, zero, newValidationError("workspace_id is required and must be a positive integer")
		}

		userID, ok := r.cfg.UserID(req)
		if !ok || userID <= 0 {
			return nil, zero, formatToolError(ctx, base.ErrUnauthorized, r.cfg.Logger)
		}

		m, err := r.cfg.Resolver.ResolveMembership(ctx, userID, wid)
		if err != nil {
			return nil, zero, formatToolError(ctx, err, r.cfg.Logger)
		}

		ctx = authz.ContextWithMembership(ctx, m)

		for _, perm := range spec.Permissions {
			if !r.cfg.Authorizer.Can(ctx, perm.Object, perm.Action) {
				return nil, zero, formatToolError(ctx, base.ErrForbidden, r.cfg.Logger)
			}
		}

		out, err := h(ctx, in)
		if err != nil {
			return nil, zero, formatToolError(ctx, err, r.cfg.Logger)
		}

		maxBytes := r.cfg.MaxResultBytes
		if maxBytes <= 0 {
			maxBytes = DefaultMaxResultBytes
		}

		data, err := json.Marshal(out)
		if err != nil {
			return nil, zero, formatToolError(ctx, err, r.cfg.Logger)
		}

		var shrinkable Shrinkable
		if s, ok := any(out).(Shrinkable); ok {
			shrinkable = s
		} else if s, ok := any(&out).(Shrinkable); ok {
			shrinkable = s
		}

		if len(data) > maxBytes && shrinkable != nil {
			for len(data) > maxBytes && shrinkable.Shrink() {
				data, err = json.Marshal(out)
				if err != nil {
					return nil, zero, formatToolError(ctx, err, r.cfg.Logger)
				}
			}
		}

		if len(data) > maxBytes {
			return nil, zero, formatToolError(ctx, ErrResultTooLarge, r.cfg.Logger)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(data)},
			},
		}, out, nil
	})
}

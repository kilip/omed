package mcpx

// Permission represents an object-action permission pair required by a tool.
type Permission struct {
	Object string `json:"object"`
	Action string `json:"action"`
}

// ToolSpec defines the registration metadata and permission requirements for an MCP tool.
type ToolSpec struct {
	Name        string
	Description string
	ReadOnly    bool
	Permissions []Permission
}

// WorkspaceScoped is implemented by tool input structs to provide the target workspace ID.
type WorkspaceScoped interface {
	GetWorkspaceID() int64
}

// Shrinkable is implemented on pointer receivers to progressively reduce output size.
type Shrinkable interface {
	Shrink() bool
}

// DefaultMaxResultBytes is the default cap for serialized MCP tool results (64 KiB).
const DefaultMaxResultBytes = 64 << 10

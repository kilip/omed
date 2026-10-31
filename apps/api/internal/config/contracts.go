package config

import "github.com/kilip/omed/api/internal/platform/mcpx"

// Module defines the contract for self-contained application modules.
// Modules register their routes, services, and event handlers during the boot phase.
type Module interface {
	Boot() error
}

// HasPreBoot is an optional interface that modules can implement
// to execute tasks before the main Boot phase.
type HasPreBoot interface {
	PreBoot() error
}

// HasMCP is an optional interface that modules can implement
// to register tools with the MCP server.
type HasMCP interface {
	RegisterMCP(r *mcpx.Registrar) error
}

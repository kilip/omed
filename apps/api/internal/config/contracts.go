package config

// Module defines the contract for self-contained application modules.
// Modules register their routes, services, and event handlers during the boot phase.
type Module interface {
	// Boot initializes and binds the module to the application runtime state.
	Boot() error
}

// HasPreBoot is an optional lifecycle hook for modules that require setup
// steps before the primary boot phase (e.g., registering middlewares or early schemas).
type HasPreBoot interface {
	// PreBoot executes early initialization before any module's Boot method is invoked.
	PreBoot() error
}

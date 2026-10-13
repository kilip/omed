package config

var (
	InitBaseMiddleware = initBaseMiddleware
	InitFirewall       = initFirewall
	CheckJWKS          = checkJWKS
)

// NewWithModules creates an App instance with explicit modules and state for testing.
func NewWithModules(state State, modules []Module) App {
	return App{
		state:   state,
		modules: modules,
	}
}

// Modules returns the registered modules for testing assertions.
func (a *App) Modules() []Module {
	return a.modules
}

// State returns the App's runtime state for testing assertions.
func (a *App) State() State {
	return a.state
}

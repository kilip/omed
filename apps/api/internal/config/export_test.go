package config

var (
	InitBaseMiddleware = initBaseMiddleware
	InitFirewall       = initFirewall
)

// SetLogFatalf overrides logFatalf for tests and returns a restore function.
func SetLogFatalf(fn func(format string, v ...any)) func() {
	orig := logFatalf
	logFatalf = fn
	return func() {
		logFatalf = orig
	}
}

// NewWithModules creates an Omed instance with explicit modules and state for testing.
func NewWithModules(state State, modules []Module) Omed {
	return Omed{
		state:   state,
		modules: modules,
	}
}

// Modules returns the registered modules for testing assertions.
func (a *Omed) Modules() []Module {
	return a.modules
}

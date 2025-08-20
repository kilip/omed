package contracts

type Event interface {
	GetName() string
}

type Listener func(event Event, dispatcher Dispatcher)

type Dispatcher interface {

	// Listen to event
	Listen(string, Listener)

	// ListenP listen to event with priority
	ListenP(string, Listener, int)

	// Dispatch start to dispatch an event
	Dispatch(Event)
}

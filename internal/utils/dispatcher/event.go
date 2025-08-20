package dispatcher

type GenericEvent struct {
	Name string
}

func NewEvent(name string) *GenericEvent {
	return &GenericEvent{Name: name}
}

func (e *GenericEvent) GetName() string {
	return e.Name
}

package dispatcher_test

import (
	"testing"

	"github.com/kilip/omed/internal/contracts"
	"github.com/kilip/omed/internal/infra/dispatcher"
	"github.com/stretchr/testify/assert"
)

type UserCreatedEvent struct {
	Username string
	Email    string
}

func (e *UserCreatedEvent) GetName() string {
	return "user.created"
}

func onUserCreated(e contracts.Event, d contracts.Dispatcher) {
	evt := e.(*UserCreatedEvent)

	evt.Username = "John Doe"
	evt.Email = "john.doe@example.com"
}

func TestListen(t *testing.T) {

	d := dispatcher.NewDispatcher()
	d.Listen("user.created", onUserCreated)

	event := &UserCreatedEvent{
		Username: "john",
		Email:    "john@example.com",
	}
	d.Dispatch(event)
	assert.Equal(t, "John Doe", event.Username)
}

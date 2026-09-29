package test

import (
	"testing"

	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToValue(t *testing.T) {
	auth := shared.AuthenticatedUser{
		ID:            shared.GenerateID(),
		Name:          "Test User",
		WorkspaceID:   shared.GenerateID(),
		WorkspaceName: "Test Team",
	}

	var snapshot model.UserSnapshot
	require.NoError(t, shared.ToValue(auth, &snapshot))
	assert.Equal(t, auth.ID, snapshot.ID)
}

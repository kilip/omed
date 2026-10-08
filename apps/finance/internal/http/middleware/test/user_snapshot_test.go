package test

import (
	"context"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/testutil"
)

type UserSnapshotTest struct {
	testutil.ApiTestSuite[core.AuthenticatedUser]
}

func handleTestPing(c fiber.Ctx) error {
	return http.OK(c, core.UserFromContext(c))
}

func (m *UserSnapshotTest) SetupSuite() {
	state := testutil.GetState()
	state.FiberApp.Get("/ping", handleTestPing)
}

func (m *UserSnapshotTest) SetupTest() {
	m.ApiTestSuite.SetupTest()
}

func (m *UserSnapshotTest) TestSnapshot() {
	m.Request("/ping", fiber.MethodGet, nil)
	m.OK()

	resp := m.GetResponse()
	m.Equal(m.User.Name, resp.Data.Name)

	state := testutil.GetState()
	ctx := context.Background()

	u, err := state.EntClient.User.Get(ctx, m.User.ID)
	m.NoError(err)
	m.Equal(m.User.Name, u.Name)

	ws, err := state.EntClient.Workspace.Get(ctx, m.User.WorkspaceID)
	m.NoError(err)
	m.Equal(m.User.WorkspaceName, ws.Name)
}

func TestAuthSnapshotSuite(t *testing.T) {
	suite.Run(t, new(UserSnapshotTest))
}

package test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/testutil"
)

type AuthSnapshotTest struct {
	testutil.ApiTestSuite[shared.AuthenticatedUser]
}

func handleTestPing(c fiber.Ctx) error {
	return http.OK(c, shared.UserFromContext(c))
}

func (m *AuthSnapshotTest) SetupTest() {
	m.ApiTestSuite.SetupTest()
	state := testutil.GetState()
	api := state.Api
	api.Get("/ping", handleTestPing)
}

func (m *AuthSnapshotTest) TestSnapshot() {
	m.Request("/ping", fiber.MethodGet, nil)
	m.OK()

	resp := m.GetResponse()
	m.Equal(m.User.Name, resp.Data.Name)
}

func TestAuthSnapshotSuite(t *testing.T) {
	suite.Run(t, new(AuthSnapshotTest))
}

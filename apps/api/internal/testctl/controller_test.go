package testctl_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/testctl"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type TestControllerSuite struct {
	suite.Suite
	client *ent.Client
	app    *fiber.App
}

func (s *TestControllerSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:testctl_test?mode=memory&cache=shared&_fk=1")
	s.app = fiber.New()
	ctl := testctl.NewController(s.client, "token")
	ctl.RegisterRoutes(s.app)
}

func (s *TestControllerSuite) TearDownSuite() {
	_ = s.client.Close()
}

func (s *TestControllerSuite) TestLifecycle() {
	// 1. Ensure user
	userReq := map[string]string{
		"name":  "E2E Tester",
		"email": "tester@omed.test",
	}
	body, _ := json.Marshal(userReq)
	req := httptest.NewRequest(http.MethodPost, "/test/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var userRes struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&userRes)
	s.Require().True(userRes.ID > 0)
	s.Require().Equal("tester@omed.test", userRes.Email)

	// 2. Create session
	sessReq := map[string]int64{"userId": userRes.ID}
	body, _ = json.Marshal(sessReq)
	req = httptest.NewRequest(http.MethodPost, "/test/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = s.app.Test(req)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var sessRes struct {
		Token  string `json:"token"`
		Cookie string `json:"cookie"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&sessRes)
	s.Require().NotEmpty(sessRes.Token)
	s.Require().Contains(sessRes.Cookie, "token=")

	// 3. Create workspace
	wsReq := map[string]any{
		"name":     "Tester Workspace",
		"ownerId":  userRes.ID,
		"personal": true,
	}
	body, _ = json.Marshal(wsReq)
	req = httptest.NewRequest(http.MethodPost, "/test/workspaces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = s.app.Test(req)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var wsRes struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		OwnerID  int64  `json:"ownerId"`
		Personal bool   `json:"personal"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wsRes)
	s.Require().True(wsRes.ID > 0)
	s.Require().True(wsRes.Personal)

	// 4. Add member
	memberReq := map[string]any{
		"workspaceId": wsRes.ID,
		"userId":      userRes.ID,
		"roles":       []string{"admin"},
	}
	body, _ = json.Marshal(memberReq)
	req = httptest.NewRequest(http.MethodPost, "/test/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = s.app.Test(req)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	// 5. Delete user
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/test/users/%d", userRes.ID), nil)
	resp, err = s.app.Test(delReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	// 6. Reset
	resetReq := httptest.NewRequest(http.MethodPost, "/test/reset", nil)
	resp, err = s.app.Test(resetReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	// 7. Flush coverage
	flushReq := httptest.NewRequest(http.MethodPost, "/test/coverage/flush", nil)
	resp, err = s.app.Test(flushReq)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode)
}

func TestController(t *testing.T) {
	suite.Run(t, new(TestControllerSuite))
}


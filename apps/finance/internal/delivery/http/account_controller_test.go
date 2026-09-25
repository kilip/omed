package http_test

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/testutil"
	"github.com/stretchr/testify/suite"
)

type AccountSuite struct {
	testutil.EndpointSuite
}

func TestAccountSuite(t *testing.T) {
	suite.Run(t, new(AccountSuite))
}

func (s *AccountSuite) createPayload() model.AccountResponse {
	return model.AccountResponse{
		Code:          "1000",
		Name:          "Cash",
		Type:          model.AccountTypeAsset,
		DetailType:    model.AccountDetailTypeCash,
		NormalBalance: model.AccountNormalBalanceDebit,
		Currency:      "USD",
	}
}

func (s *AccountSuite) createAccount() model.AccountResponse {
	token := s.Token(testutil.RoleFinance)
	body := testutil.JSONBody(s.T(), s.createPayload())
	resp := s.Do(fiber.MethodPost, "/accounts", token, body)
	s.Equal(fiber.StatusCreated, resp.StatusCode)

	var env model.Envelope[model.AccountResponse]
	testutil.DecodeEnvelope(s.T(), resp, &env)

	return env.Data
}

// --- POST /accounts ---

func (s *AccountSuite) TestCreate_MissingToken() {
	// no Authorization header at all -> jwtware's ErrJWTMissingOrMalformed -> 400
	body := testutil.JSONBody(s.T(), s.createPayload())
	resp := s.Do(fiber.MethodPost, "/accounts", "", body)

	s.Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountSuite) TestCreate_InvalidToken() {
	// token present but signature doesn't verify -> 401
	body := testutil.JSONBody(s.T(), s.createPayload())
	resp := s.Do(fiber.MethodPost, "/accounts", "not-a-valid-jwt", body)

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AccountSuite) TestCreate_Forbidden_ReadOnlyRole() {
	// "member" only has "accounts read" in policy.csv
	token := s.Token(testutil.RoleMember)
	body := testutil.JSONBody(s.T(), s.createPayload())

	resp := s.Do(fiber.MethodPost, "/accounts", token, body)

	s.Equal(fiber.StatusForbidden, resp.StatusCode)
}

func (s *AccountSuite) TestCreate_Success() {
	token := s.Token(testutil.RoleFinance)
	body := testutil.JSONBody(s.T(), s.createPayload())

	resp := s.Do(fiber.MethodPost, "/accounts", token, body)
	s.Require().Equal(fiber.StatusCreated, resp.StatusCode)

	var env model.Envelope[model.AccountResponse]
	testutil.DecodeEnvelope(s.T(), resp, &env)
	s.True(env.Success)
	s.NotEqual(uuid.Nil, env.Data.ID)
	s.Equal("1000", env.Data.Code)
	s.Equal("Cash", env.Data.Name)
	s.Equal(model.AccountTypeAsset, env.Data.Type)
}

// -- PUT /accounts/{id} --
func (s *AccountSuite) TestUpdate_Success() {
	token := s.Token(testutil.RoleFinance)
	body := testutil.JSONBody(s.T(), s.createPayload())
	resp := s.Do(fiber.MethodPost, "/accounts", token, body)
	s.Equal(fiber.StatusCreated, resp.StatusCode)

	var env model.Envelope[model.AccountResponse]
	testutil.DecodeEnvelope(s.T(), resp, &env)

	created := env.Data
	created.Name = "Updated Name"
	body = testutil.JSONBody(s.T(), created)
	resp = s.Do(fiber.MethodPut, fmt.Sprintf("/accounts/%s", created.ID), token, body)
	s.Equal(fiber.StatusOK, resp.StatusCode)
	testutil.DecodeEnvelope(s.T(), resp, &env)

	s.Equal(created.Name, env.Data.Name)
}

// --- DELETE /accounts/{id}
func (s *AccountSuite) TestDelete_Success() {
	token := s.Token(testutil.RoleFinance)
	created := s.createAccount()
	resp := s.Do(fiber.MethodDelete, fmt.Sprintf("/accounts/%v", created.ID), token, nil)

	s.Equal(fiber.StatusNoContent, resp.StatusCode)
}

// --- GET /accounts/{id}
func (s *AccountSuite) TestGetByID_Success() {
	account := s.createAccount()
	token := s.Token(testutil.RoleMember)
	resp := s.Do(fiber.MethodGet, fmt.Sprintf("/accounts/%s", account.ID), token, nil)

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

// --- GET /accounts ---

func (s *AccountSuite) TestList_MissingToken() {
	resp := s.Do(fiber.MethodGet, "/accounts", "", nil)

	s.Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AccountSuite) TestList_Forbidden_UnknownRole() {
	// role not present in policy.csv at all
	token := s.Token(testutil.Role("guest"))
	body := testutil.JSONBody(s.T(), model.AccountFilter{})

	resp := s.Do(fiber.MethodGet, "/accounts", token, body)

	s.Equal(fiber.StatusForbidden, resp.StatusCode)
}

func (s *AccountSuite) TestList_Success() {
	s.createAccount()

	token := s.Token(testutil.RoleFinance)
	body := testutil.JSONBody(s.T(), model.AccountFilter{})
	resp := s.Do(fiber.MethodGet, "/accounts", token, body)

	s.Equal(fiber.StatusOK, resp.StatusCode)

	var env model.Envelope[[]*model.AccountResponse]
	testutil.DecodeEnvelope(s.T(), resp, &env)
	s.Len(env.Data, 1)
}

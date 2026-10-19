package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type AccountApiSuite struct {
	testutil.ApiTestSuite[model.AccountResponse]
}

func ptr[T any](v T) *T {
	return &v
}

func (s *AccountApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *AccountApiSuite) getListResponse() httpx.WebResponse[[]model.AccountResponse] {
	var res httpx.WebResponse[[]model.AccountResponse]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&res))
	return res
}

func (s *AccountApiSuite) Test_Create_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:        "1000",
		Name:        "Cash and Bank",
		Type:        domain.AccountTypeAsset,
		Description: "Operating cash",
		IsActive:    ptr(true),
		IsPostable:  ptr(true),
	})
	s.Created()

	res := s.GetResponse()
	s.Equal("1000", res.Data.Code)
	s.Equal("Cash and Bank", res.Data.Name)
	s.Equal(domain.AccountTypeAsset, res.Data.Type)
	s.Equal(domain.NormalBalanceDebit, res.Data.NormalBalance)
	s.True(res.Data.IsPostable)
	s.True(res.Data.IsActive)
	s.Nil(res.Data.AuditMeta.DeletedAt)
}

func (s *AccountApiSuite) Test_Create_DefaultValues() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code: "1002",
		Name: "Testing Defaults",
		Type: domain.AccountTypeAsset,
	})
	s.Created()

	res := s.GetResponse()
	s.Equal("1002", res.Data.Code)
	s.True(res.Data.IsActive, "expected IsActive to default to true")
	s.True(res.Data.IsPostable, "expected IsPostable to default to true")
}

func (s *AccountApiSuite) Test_GetByID_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code: "1010",
		Name: "Petty Cash",
		Type: domain.AccountTypeAsset,
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()

	res := s.GetResponse()
	s.Equal(created.ID, res.Data.ID)
	s.Equal("1010", res.Data.Code)
}

func (s *AccountApiSuite) Test_Search_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code: "1100",
		Name: "Accounts Receivable",
		Type: domain.AccountTypeAsset,
	})
	s.Created()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code: "2100",
		Name: "Accounts Payable",
		Type: domain.AccountTypeLiability,
	})
	s.Created()

	// Search returns all accounts in workspace without filtering
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/search", ws.ID), fiber.MethodPost, model.SearchAccountRequest{})
	s.OK()

	listRes := s.getListResponse()
	s.Len(listRes.Data, 2)
	s.Equal("1100", listRes.Data[0].Code)
	s.Equal("2100", listRes.Data[1].Code)
}

func (s *AccountApiSuite) Test_Update_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:       "3000",
		Name:       "Common Stock",
		Type:       domain.AccountTypeEquity,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdateAccountRequest{
		Name:       "Capital Stock",
		IsPostable: ptr(false),
	})
	s.OK()

	res := s.GetResponse()
	s.Equal("Capital Stock", res.Data.Name)
	s.False(res.Data.IsPostable)
	s.True(res.Data.IsActive, "IsActive should remain untouched")
}

func (s *AccountApiSuite) Test_Update_PreserveFlagsWhenNotSpecified() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code: "3001",
		Name: "Retained Earnings",
		Type: domain.AccountTypeEquity,
	})
	s.Created()
	created := s.GetResponse().Data
	s.True(created.IsActive)
	s.True(created.IsPostable)

	// Update only name
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdateAccountRequest{
		Name: "Updated Retained Earnings",
	})
	s.OK()

	res := s.GetResponse()
	s.Equal("Updated Retained Earnings", res.Data.Name)
	s.True(res.Data.IsActive, "IsActive should remain true")
	s.True(res.Data.IsPostable, "IsPostable should remain true")
}

func (s *AccountApiSuite) Test_SoftDelete_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:       "5000",
		Name:       "Office Supplies",
		Type:       domain.AccountTypeExpense,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	})
	s.Created()
	created := s.GetResponse().Data

	// Delete
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	// Subsequent GetByID returns 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *AccountApiSuite) Test_SoftDelete_BlockedIfChildrenExist() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:       "6000",
		Name:       "Administrative Expense",
		Type:       domain.AccountTypeExpense,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	})
	s.Created()
	root := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:       "6001",
		Name:       "Salaries Expense",
		Type:       domain.AccountTypeExpense,
		ParentID:   root.ID,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	})
	s.Created()

	// Deleting root fails with 409 Conflict
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, root.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *AccountApiSuite) Test_RBAC_Viewer_Accountant_Admin() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	// Create an account as owner
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:       "7000",
		Name:       "Interest Income",
		Type:       domain.AccountTypeRevenue,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	})
	s.Created()
	acc := s.GetResponse().Data

	// 1. Viewer member
	viewer := s.AsUser("viewer@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	// Viewer can read
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, acc.ID), fiber.MethodGet, nil)
	s.OK()

	// Viewer cannot create -> 403 Forbidden
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:       "7001",
		Name:       "Dividend Income",
		Type:       domain.AccountTypeRevenue,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	})
	s.AssertStatus(fiber.StatusForbidden)

	// 2. Accountant member
	accountant := s.AsUser("accountant@example.com", "Accountant User")
	s.AddMember(ws.ID, accountant.ID, domain.RoleAccountant)

	// Accountant can create
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:       "7002",
		Name:       "Dividend Income",
		Type:       domain.AccountTypeRevenue,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	})
	s.Created()
	createdAcc := s.GetResponse().Data

	// Accountant cannot delete -> 403 Forbidden
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, createdAcc.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)

	// 3. Admin member
	admin := s.AsUser("admin@example.com", "Admin User")
	s.AddMember(ws.ID, admin.ID, domain.RoleAdmin)

	// Admin can delete -> 200 OK
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d", ws.ID, createdAcc.ID), fiber.MethodDelete, nil)
	s.OK()
}

func (s *AccountApiSuite) Test_Seed_Accounts() {
	ws := s.ensureWorkspace("Clean Workspace", s.User.ID)

	// 1. Invalid template / lang -> 400 Bad Request
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/seed", ws.ID), fiber.MethodPost, model.SeedAccountsRequest{
		Template: "invalid",
		Lang:     "id",
	})
	s.AssertStatus(fiber.StatusBadRequest)

	// 2. Valid freelancer id template -> 201 Created
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/seed", ws.ID), fiber.MethodPost, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "id",
	})
	s.Created()

	var seedRes httpx.WebResponse[model.SeedAccountsResponse]
	err := json.NewDecoder(s.HttpResponse.Body).Decode(&seedRes)
	s.Require().NoError(err)
	s.Greater(seedRes.Data.Count, 40)

	// 3. Seeding again on populated workspace -> 409 Conflict
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/seed", ws.ID), fiber.MethodPost, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "id",
	})
	s.AssertStatus(fiber.StatusConflict)

	// 4. Verify search returns the accounts
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/search", ws.ID), fiber.MethodPost, model.SearchAccountRequest{})
	s.OK()
	listRes := s.getListResponse()
	s.Len(listRes.Data, seedRes.Data.Count)
}

func TestAccountApiSuite(t *testing.T) {
	suite.Run(t, new(AccountApiSuite))
}

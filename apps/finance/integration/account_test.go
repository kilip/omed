package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/util"
	"github.com/kilip/omed/finance/testutil"
)

type AccountSuite struct {
	testutil.ApiTestSuite[model.AccountResponse]
}

func (s *AccountSuite) SetupTest() {
	s.ApiTestSuite.SetupTest()
}

func (s *AccountSuite) TearDownTest() {
	if s.User != nil {
		ctx := core.ContextWithUser(context.Background(), *s.User)
		_, _ = testutil.GetState().EntClient.Account.Delete().Exec(ctx)
	}
}

func (s *AccountSuite) createAccount(req model.CreateAccountRequest) model.AccountResponse {
	s.T().Helper()
	s.Request("/accounts", http.MethodPost, req)
	s.Created()
	resp := s.GetResponse()
	return resp.Data
}

func (s *AccountSuite) decodeErrorResponse() model.ErrorResponse {
	s.T().Helper()
	var errResp model.ErrorResponse
	defer s.HttpResponse.Body.Close()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&errResp))
	return errResp
}

// ---------------------------------------------------------------------------
// List Accounts Tests (GET /accounts)
// ---------------------------------------------------------------------------

func (s *AccountSuite) TestList_Empty() {
	s.Request("/accounts", http.MethodGet, nil)
	s.OK()
	resp := s.PagedResponse()
	s.Empty(resp.Data)
}

func (s *AccountSuite) TestList_Success() {
	acc1 := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Utama",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})
	acc2 := s.createAccount(model.CreateAccountRequest{
		Code:     "2001",
		Name:     "Hutang Usaha",
		Type:     model.AccountTypeLiability,
		Currency: "IDR",
	})

	s.Request("/accounts", http.MethodGet, nil)
	s.OK()
	resp := s.PagedResponse()
	s.Len(resp.Data, 2)

	codes := []string{resp.Data[0].Code, resp.Data[1].Code}
	s.Contains(codes, acc1.Code)
	s.Contains(codes, acc2.Code)
}

func (s *AccountSuite) TestList_FilterByType() {
	s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})
	s.createAccount(model.CreateAccountRequest{
		Code:     "2001",
		Name:     "Hutang",
		Type:     model.AccountTypeLiability,
		Currency: "IDR",
	})

	s.Request("/accounts?type=asset", http.MethodGet, nil)
	s.OK()
	resp := s.PagedResponse()
	s.Len(resp.Data, 1)
	s.Equal("1001", resp.Data[0].Code)
	s.Equal(model.AccountTypeAsset, resp.Data[0].Type)

	s.Request("/accounts?type=liability", http.MethodGet, nil)
	s.OK()
	resp = s.PagedResponse()
	s.Len(resp.Data, 1)
	s.Equal("2001", resp.Data[0].Code)
	s.Equal(model.AccountTypeLiability, resp.Data[0].Type)
}

func (s *AccountSuite) TestList_FilterByStatus() {
	acc1 := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Aktif",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})
	acc2 := s.createAccount(model.CreateAccountRequest{
		Code:     "1002",
		Name:     "Kas Lama",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	archivedStatus := model.AccountStatusArchived
	s.Request(fmt.Sprintf("/accounts/%s", acc2.ID), http.MethodPut, model.UpdateAccountRequest{
		Status: &archivedStatus,
	})
	s.OK()

	s.Request("/accounts?status=active", http.MethodGet, nil)
	s.OK()
	activeList := s.PagedResponse()
	s.Len(activeList.Data, 1)
	s.Equal(acc1.ID, activeList.Data[0].ID)

	s.Request("/accounts?status=archived", http.MethodGet, nil)
	s.OK()
	archivedList := s.PagedResponse()
	s.Len(archivedList.Data, 1)
	s.Equal(acc2.ID, archivedList.Data[0].ID)
}

func (s *AccountSuite) TestList_InvalidFilter() {
	s.Request("/accounts?type=invalid_type", http.MethodGet, nil)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)

	s.Request("/accounts?status=invalid_status", http.MethodGet, nil)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp = s.decodeErrorResponse()
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)
}

func (s *AccountSuite) TestList_WorkspaceIsolation() {
	acc1 := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Workspace 1",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	// Switch to a new workspace user
	s.User = &core.AuthenticatedUser{
		Name:           "User Two",
		ID:             util.GenerateID(),
		WorkspaceID:    util.GenerateID(),
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
		WorkspaceName:  "Workspace Two",
	}

	s.Request("/accounts", http.MethodGet, nil)
	s.OK()
	resp := s.PagedResponse()
	s.Empty(resp.Data)

	// Create account in workspace 2
	acc2 := s.createAccount(model.CreateAccountRequest{
		Code:     "1002",
		Name:     "Kas Workspace 2",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	s.Request("/accounts", http.MethodGet, nil)
	s.OK()
	resp = s.PagedResponse()
	s.Len(resp.Data, 1)
	s.Equal(acc2.ID, resp.Data[0].ID)
	s.NotEqual(acc1.ID, resp.Data[0].ID)
}

func (s *AccountSuite) TestList_Forbidden() {
	s.User.WorkspaceRoles = []core.WorkspaceRole{"guest"}
	s.Request("/accounts", http.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *AccountSuite) TestList_Unauthorized() {
	s.RequestWithToken("/accounts", http.MethodGet, nil, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Create Account Tests (POST /accounts)
// ---------------------------------------------------------------------------

func (s *AccountSuite) TestCreate_Success() {
	desc := "Kas Operasional Utama"
	req := model.CreateAccountRequest{
		Code:        "1001",
		Name:        "Kas Utama",
		Description: &desc,
		Type:        model.AccountTypeAsset,
		Currency:    "IDR",
	}

	s.Request("/accounts", http.MethodPost, req)
	s.Created()
	resp := s.GetResponse()

	s.NotEqual(uuid.Nil, resp.Data.ID)
	s.Equal(s.User.WorkspaceID, resp.Data.WorkspaceID)
	s.Equal(req.Code, resp.Data.Code)
	s.Equal(req.Name, resp.Data.Name)
	s.Require().NotNil(resp.Data.Description)
	s.Equal(desc, *resp.Data.Description)
	s.Equal(req.Type, resp.Data.Type)
	s.Equal(req.Currency, resp.Data.Currency)
	s.Equal(model.AccountStatusActive, resp.Data.Status)
	s.Nil(resp.Data.ParentID)
	s.Equal(s.User.ID, resp.Data.CreatedBy)
	s.Equal(s.User.ID, resp.Data.UpdatedBy)
}

func (s *AccountSuite) TestCreate_WithoutDescription() {
	req := model.CreateAccountRequest{
		Code:     "1002",
		Name:     "Kas Kecil",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}

	s.Request("/accounts", http.MethodPost, req)
	s.Created()
	resp := s.GetResponse()

	s.Nil(resp.Data.Description)
	s.Equal("1002", resp.Data.Code)
}

func (s *AccountSuite) TestCreate_WithParentID() {
	parent := s.createAccount(model.CreateAccountRequest{
		Code:     "1000",
		Name:     "Aset Lancar",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	child := s.createAccount(model.CreateAccountRequest{
		Code:     "1010",
		Name:     "Kas Bank",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
		ParentID: &parent.ID,
	})

	s.Require().NotNil(child.ParentID)
	s.Equal(parent.ID, *child.ParentID)
}

func (s *AccountSuite) TestCreate_DuplicateCode() {
	req := model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Utama",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}

	s.createAccount(req)

	// Second create with duplicate code should fail
	s.Request("/accounts", http.MethodPost, req)
	s.AssertStatus(fiber.StatusInternalServerError)
}

func (s *AccountSuite) TestCreate_ValidationError() {
	// Missing required fields
	reqMissing := model.CreateAccountRequest{}
	s.Request("/accounts", http.MethodPost, reqMissing)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)

	// Invalid type
	reqInvalidType := model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas",
		Type:     "invalid_type",
		Currency: "IDR",
	}
	s.Request("/accounts", http.MethodPost, reqInvalidType)
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Invalid currency length / format
	reqInvalidCurrency := model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas",
		Type:     model.AccountTypeAsset,
		Currency: "idr", // lowercase
	}
	s.Request("/accounts", http.MethodPost, reqInvalidCurrency)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *AccountSuite) TestCreate_Forbidden() {
	s.User.WorkspaceRoles = []core.WorkspaceRole{core.WorkspaceRoleMember}
	s.Request("/accounts", http.MethodPost, model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *AccountSuite) TestCreate_Unauthorized() {
	s.RequestWithToken("/accounts", http.MethodPost, model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Get Account By ID Tests (GET /accounts/:id)
// ---------------------------------------------------------------------------

func (s *AccountSuite) TestGetByID_Success() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1101",
		Name:     "Bank BCA",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodGet, nil)
	s.OK()
	resp := s.GetResponse()

	s.Equal(created.ID, resp.Data.ID)
	s.Equal(created.Code, resp.Data.Code)
	s.Equal(created.Name, resp.Data.Name)
	s.Equal(created.Type, resp.Data.Type)
}

func (s *AccountSuite) TestGetByID_NotFound() {
	randomID := uuid.New()
	s.Request(fmt.Sprintf("/accounts/%s", randomID), http.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
	errResp := s.decodeErrorResponse()
	s.Equal("NOT_FOUND", errResp.Error.Code)
}

func (s *AccountSuite) TestGetByID_InvalidID() {
	s.Request("/accounts/invalid-uuid-format", http.MethodGet, nil)
	s.AssertStatus(fiber.StatusBadRequest)
	errResp := s.decodeErrorResponse()
	s.Equal("INVALID_ID", errResp.Error.Code)
}

func (s *AccountSuite) TestGetByID_WorkspaceIsolation() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1101",
		Name:     "Bank BCA",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	// User in different workspace
	s.User = &core.AuthenticatedUser{
		Name:           "User Two",
		ID:             util.GenerateID(),
		WorkspaceID:    util.GenerateID(),
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
		WorkspaceName:  "Workspace Two",
	}

	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *AccountSuite) TestGetByID_Forbidden() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1101",
		Name:     "Bank BCA",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	s.User.WorkspaceRoles = []core.WorkspaceRole{"guest"}
	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *AccountSuite) TestGetByID_Unauthorized() {
	s.RequestWithToken(fmt.Sprintf("/accounts/%s", uuid.New()), http.MethodGet, nil, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Update Account Tests (PUT /accounts/:id)
// ---------------------------------------------------------------------------

func (s *AccountSuite) TestUpdate_Success() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Asli",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	newName := "Kas Diperbarui"
	newDesc := "Deskripsi Baru"
	newStatus := model.AccountStatusArchived
	updateReq := model.UpdateAccountRequest{
		Name:        &newName,
		Description: &newDesc,
		Status:      &newStatus,
	}

	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodPut, updateReq)
	s.OK()
	resp := s.GetResponse()

	s.Equal(created.ID, resp.Data.ID)
	s.Equal(newName, resp.Data.Name)
	s.Require().NotNil(resp.Data.Description)
	s.Equal(newDesc, *resp.Data.Description)
	s.Equal(newStatus, resp.Data.Status)
	// Code, Type, Currency cannot be changed
	s.Equal(created.Code, resp.Data.Code)
	s.Equal(created.Type, resp.Data.Type)
	s.Equal(created.Currency, resp.Data.Currency)
}

func (s *AccountSuite) TestUpdate_ParentID() {
	parent := s.createAccount(model.CreateAccountRequest{
		Code:     "1000",
		Name:     "Induk",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})
	child := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Anak",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})
	s.Nil(child.ParentID)

	updateReq := model.UpdateAccountRequest{
		ParentID: &parent.ID,
	}

	s.Request(fmt.Sprintf("/accounts/%s", child.ID), http.MethodPut, updateReq)
	s.OK()
	resp := s.GetResponse()

	s.Require().NotNil(resp.Data.ParentID)
	s.Equal(parent.ID, *resp.Data.ParentID)
}

func (s *AccountSuite) TestUpdate_NotFound() {
	newName := "Baru"
	s.Request(fmt.Sprintf("/accounts/%s", uuid.New()), http.MethodPut, model.UpdateAccountRequest{
		Name: &newName,
	})
	s.AssertStatus(fiber.StatusNotFound)
	errResp := s.decodeErrorResponse()
	s.Equal("NOT_FOUND", errResp.Error.Code)
}

func (s *AccountSuite) TestUpdate_InvalidID() {
	newName := "Baru"
	s.Request("/accounts/invalid-uuid", http.MethodPut, model.UpdateAccountRequest{
		Name: &newName,
	})
	s.AssertStatus(fiber.StatusBadRequest)
	errResp := s.decodeErrorResponse()
	s.Equal("INVALID_ID", errResp.Error.Code)
}

func (s *AccountSuite) TestUpdate_WorkspaceIsolation() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Workspace 1",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	// Switch workspace
	s.User = &core.AuthenticatedUser{
		Name:           "User Two",
		ID:             util.GenerateID(),
		WorkspaceID:    util.GenerateID(),
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
		WorkspaceName:  "Workspace Two",
	}

	newName := "Hacked Name"
	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodPut, model.UpdateAccountRequest{
		Name: &newName,
	})
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *AccountSuite) TestUpdate_ValidationError() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	invalidStatus := model.AccountStatus("invalid_status")
	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodPut, model.UpdateAccountRequest{
		Status: &invalidStatus,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)
}

func (s *AccountSuite) TestUpdate_Forbidden() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	s.User.WorkspaceRoles = []core.WorkspaceRole{core.WorkspaceRoleMember}
	newName := "New Name"
	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodPut, model.UpdateAccountRequest{
		Name: &newName,
	})
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *AccountSuite) TestUpdate_Unauthorized() {
	newName := "New Name"
	s.RequestWithToken(fmt.Sprintf("/accounts/%s", uuid.New()), http.MethodPut, model.UpdateAccountRequest{
		Name: &newName,
	}, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Delete Account Tests (DELETE /accounts/:id)
// ---------------------------------------------------------------------------

func (s *AccountSuite) TestDelete_Success() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Hapus",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodDelete, nil)
	s.NoContent()

	// Verify account is deleted
	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *AccountSuite) TestDelete_NotFound() {
	s.Request(fmt.Sprintf("/accounts/%s", uuid.New()), http.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)
	errResp := s.decodeErrorResponse()
	s.Equal("NOT_FOUND", errResp.Error.Code)
}

func (s *AccountSuite) TestDelete_InvalidID() {
	s.Request("/accounts/invalid-uuid", http.MethodDelete, nil)
	s.AssertStatus(fiber.StatusBadRequest)
	errResp := s.decodeErrorResponse()
	s.Equal("INVALID_ID", errResp.Error.Code)
}

func (s *AccountSuite) TestDelete_WorkspaceIsolation() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Workspace 1",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	// Switch workspace
	s.User = &core.AuthenticatedUser{
		Name:           "User Two",
		ID:             util.GenerateID(),
		WorkspaceID:    util.GenerateID(),
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
		WorkspaceName:  "Workspace Two",
	}

	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *AccountSuite) TestDelete_Forbidden() {
	created := s.createAccount(model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})

	s.User.WorkspaceRoles = []core.WorkspaceRole{core.WorkspaceRoleMember}
	s.Request(fmt.Sprintf("/accounts/%s", created.ID), http.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *AccountSuite) TestDelete_Unauthorized() {
	s.RequestWithToken(fmt.Sprintf("/accounts/%s", uuid.New()), http.MethodDelete, nil, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Seed Accounts Tests (POST /accounts/seed)
// ---------------------------------------------------------------------------

func (s *AccountSuite) TestSeed_Success() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}

	s.Request("/accounts/seed", http.MethodPost, req)
	s.Created()
	resp := s.PagedResponse()

	s.NotEmpty(resp.Data)
	for _, acc := range resp.Data {
		s.NotEqual(uuid.Nil, acc.ID)
		s.Equal(s.User.WorkspaceID, acc.WorkspaceID)
		s.Equal("IDR", acc.Currency)
	}

	// Verify querying list returns all seeded accounts
	s.Request("/accounts", http.MethodGet, nil)
	s.OK()
	listResp := s.PagedResponse()
	s.Equal(len(resp.Data), len(listResp.Data))
}

func (s *AccountSuite) TestSeed_DefaultCurrency() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "en",
		Currency: "",
		Force:    false,
	}

	s.Request("/accounts/seed", http.MethodPost, req)
	s.Created()
	resp := s.PagedResponse()

	s.NotEmpty(resp.Data)
	for _, acc := range resp.Data {
		s.Equal("IDR", acc.Currency)
	}
}

func (s *AccountSuite) TestSeed_CustomCurrency() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "en",
		Currency: "USD",
		Force:    false,
	}

	s.Request("/accounts/seed", http.MethodPost, req)
	s.Created()
	resp := s.PagedResponse()

	s.NotEmpty(resp.Data)
	for _, acc := range resp.Data {
		s.Equal("USD", acc.Currency)
	}
}

func (s *AccountSuite) TestSeed_AlreadySeeded_ForceFalse() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}

	s.Request("/accounts/seed", http.MethodPost, req)
	s.Created()

	// Seed again with Force: false
	s.Request("/accounts/seed", http.MethodPost, req)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("ACCOUNTS_NOT_EMPTY", errResp.Error.Code)
}

func (s *AccountSuite) TestSeed_AlreadySeeded_ForceTrue() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}

	s.Request("/accounts/seed", http.MethodPost, req)
	s.Created()
	resp1 := s.PagedResponse()

	// Seed again with Force: true
	reqForce := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
		Force:    true,
	}
	s.Request("/accounts/seed", http.MethodPost, reqForce)
	s.Created()
	resp2 := s.PagedResponse()

	s.Equal(len(resp1.Data), len(resp2.Data))
}

func (s *AccountSuite) TestSeed_ValidationError() {
	// Missing required profile and lang
	req := model.SeedAccountRequest{}
	s.Request("/accounts/seed", http.MethodPost, req)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)
}

func (s *AccountSuite) TestSeed_InvalidProfile() {
	req := model.SeedAccountRequest{
		Profile:  "nonexistent_profile",
		Lang:     "id",
		Currency: "IDR",
	}
	s.Request("/accounts/seed", http.MethodPost, req)
	s.AssertStatus(fiber.StatusInternalServerError)
	errResp := s.decodeErrorResponse()
	s.Equal("INTERNAL", errResp.Error.Code)
}

func (s *AccountSuite) TestSeed_Forbidden() {
	s.User.WorkspaceRoles = []core.WorkspaceRole{core.WorkspaceRoleMember}
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
	}
	s.Request("/accounts/seed", http.MethodPost, req)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *AccountSuite) TestSeed_Unauthorized() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
	}
	s.RequestWithToken("/accounts/seed", http.MethodPost, req, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

func TestAccountSuite(t *testing.T) {
	suite.Run(t, new(AccountSuite))
}

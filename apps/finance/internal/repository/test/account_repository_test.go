package test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/config"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/repository"
)

type AccountRepositorySuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.AccountRepository
	ctx    context.Context
	user   core.AuthenticatedUser
}

func (s *AccountRepositorySuite) SetupSuite() {
	dbName := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", uuid.New().String())
	client, err := ent.Open("sqlite3", dbName)
	s.Require().NoError(err)

	err = client.Schema.Create(context.Background())
	s.Require().NoError(err)

	config.ConfigureDBClient(client)
	s.client = client

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s.repo = repository.NewAccountRepository(client, logger)
}

func (s *AccountRepositorySuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *AccountRepositorySuite) createUser(u core.AuthenticatedUser) {
	_, err := s.client.User.Create().
		SetID(u.ID).
		SetName(u.Name).
		SetAvatar("").
		Save(context.Background())
	s.Require().NoError(err)
}

func (s *AccountRepositorySuite) SetupTest() {
	s.user = core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "Test User",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Test Workspace",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.ctx = core.ContextWithUser(context.Background(), s.user)
	s.createUser(s.user)
}

func (s *AccountRepositorySuite) TearDownTest() {
	_, _ = s.client.Account.Delete().Exec(s.ctx)
	_, _ = s.client.User.Delete().Exec(context.Background())
}

// ---------------------------------------------------------------------------
// Create Tests
// ---------------------------------------------------------------------------

func (s *AccountRepositorySuite) TestCreate_Success() {
	desc := "Kas Operasional Utama"
	req := model.CreateAccountRequest{
		Code:        "1001",
		Name:        "Kas Utama",
		Description: &desc,
		Type:        model.AccountTypeAsset,
		Currency:    "IDR",
	}

	res, err := s.repo.Create(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	s.NotEqual(uuid.Nil, res.ID)
	s.Equal(s.user.WorkspaceID, res.WorkspaceID)
	s.Equal(req.Code, res.Code)
	s.Equal(req.Name, res.Name)
	s.Require().NotNil(res.Description)
	s.Equal(desc, *res.Description)
	s.Equal(req.Type, res.Type)
	s.Equal(req.Currency, res.Currency)
	s.Equal(model.AccountStatusActive, res.Status)
	s.Nil(res.ParentID)
	s.Equal(s.user.ID, res.CreatedBy)
	s.Equal(s.user.ID, res.UpdatedBy)
	s.False(res.CreatedAt.IsZero())
	s.False(res.UpdatedAt.IsZero())

	// Verify account can be retrieved by ID
	fetched, err := s.repo.GetByID(s.ctx, res.ID)
	s.Require().NoError(err)
	s.Equal(res.ID, fetched.ID)
	s.Equal(res.Code, fetched.Code)
}

func (s *AccountRepositorySuite) TestCreate_WithoutDescription() {
	req := model.CreateAccountRequest{
		Code:        "1002",
		Name:        "Kas Kecil",
		Description: nil,
		Type:        model.AccountTypeAsset,
		Currency:    "IDR",
	}

	res, err := s.repo.Create(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Nil(res.Description)
}

func (s *AccountRepositorySuite) TestCreate_WithParentID() {
	parentReq := model.CreateAccountRequest{
		Code:     "1000",
		Name:     "Aset Lancar",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}
	parent, err := s.repo.Create(s.ctx, parentReq)
	s.Require().NoError(err)

	childReq := model.CreateAccountRequest{
		Code:     "1010",
		Name:     "Kas Bank",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
		ParentID: &parent.ID,
	}
	child, err := s.repo.Create(s.ctx, childReq)
	s.Require().NoError(err)
	s.Require().NotNil(child)
	s.Require().NotNil(child.ParentID)
	s.Equal(parent.ID, *child.ParentID)
}

func (s *AccountRepositorySuite) TestCreate_DuplicateCodeSameWorkspace() {
	req := model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Utama",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}

	res, err := s.repo.Create(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	// Create again with duplicate code in same workspace
	dupRes, err := s.repo.Create(s.ctx, req)
	s.Error(err)
	s.Nil(dupRes)
}

func (s *AccountRepositorySuite) TestCreate_SameCodeDifferentWorkspace() {
	req := model.CreateAccountRequest{
		Code:     "1001",
		Name:     "Kas Utama",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}

	res1, err := s.repo.Create(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(res1)

	// User in second workspace
	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)
	defer func() {
		_, _ = s.client.Account.Delete().Exec(ctx2)
	}()

	res2, err := s.repo.Create(ctx2, req)
	s.Require().NoError(err)
	s.Require().NotNil(res2)
	s.NotEqual(res1.ID, res2.ID)
	s.Equal(user2.WorkspaceID, res2.WorkspaceID)
}

func (s *AccountRepositorySuite) TestCreate_ValidationError() {
	// Empty code
	reqEmptyCode := model.CreateAccountRequest{
		Code:     "",
		Name:     "Akun Tanpa Kode",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}
	res, err := s.repo.Create(s.ctx, reqEmptyCode)
	s.Error(err)
	s.Nil(res)

	// Empty name
	reqEmptyName := model.CreateAccountRequest{
		Code:     "9999",
		Name:     "",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}
	res, err = s.repo.Create(s.ctx, reqEmptyName)
	s.Error(err)
	s.Nil(res)
}

// ---------------------------------------------------------------------------
// GetByID Tests
// ---------------------------------------------------------------------------

func (s *AccountRepositorySuite) TestGetByID_Success() {
	created, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code:     "1101",
		Name:     "Bank Mandiri",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})
	s.Require().NoError(err)

	res, err := s.repo.GetByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(created.ID, res.ID)
	s.Equal(created.Code, res.Code)
	s.Equal(created.Name, res.Name)
	s.Equal(created.Type, res.Type)
}

func (s *AccountRepositorySuite) TestGetByID_NotFound() {
	res, err := s.repo.GetByID(s.ctx, uuid.New())
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

func (s *AccountRepositorySuite) TestGetByID_OtherWorkspace() {
	created, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code:     "1101",
		Name:     "Bank Mandiri",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)

	// User2 should not be able to get account from workspace 1
	res, err := s.repo.GetByID(ctx2, created.ID)
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

// ---------------------------------------------------------------------------
// List Tests
// ---------------------------------------------------------------------------

func (s *AccountRepositorySuite) TestList_Empty() {
	res, err := s.repo.List(s.ctx, model.ListAccountRequest{})
	s.Require().NoError(err)
	s.Empty(res)
}

func (s *AccountRepositorySuite) TestList_All() {
	_, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "2001", Name: "Hutang Usaha", Type: model.AccountTypeLiability, Currency: "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "5001", Name: "Beban Gaji", Type: model.AccountTypeExpense, Currency: "IDR",
	})
	s.Require().NoError(err)

	res, err := s.repo.List(s.ctx, model.ListAccountRequest{})
	s.Require().NoError(err)
	s.Len(res, 3)
}

func (s *AccountRepositorySuite) TestList_FilterByType() {
	_, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1002", Name: "Bank", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "2001", Name: "Hutang", Type: model.AccountTypeLiability, Currency: "IDR",
	})
	s.Require().NoError(err)

	assets, err := s.repo.List(s.ctx, model.ListAccountRequest{Type: model.AccountTypeAsset})
	s.Require().NoError(err)
	s.Len(assets, 2)
	for _, acc := range assets {
		s.Equal(model.AccountTypeAsset, acc.Type)
	}

	liabilities, err := s.repo.List(s.ctx, model.ListAccountRequest{Type: model.AccountTypeLiability})
	s.Require().NoError(err)
	s.Len(liabilities, 1)
	s.Equal("2001", liabilities[0].Code)
}

func (s *AccountRepositorySuite) TestList_FilterByStatus() {
	acc1, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas Aktif", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	acc2, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1002", Name: "Kas Lama", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	// Archive acc2
	archivedStatus := model.AccountStatusArchived
	_, err = s.repo.Update(s.ctx, acc2.ID, model.UpdateAccountRequest{
		Status: &archivedStatus,
	})
	s.Require().NoError(err)

	activeList, err := s.repo.List(s.ctx, model.ListAccountRequest{Status: model.AccountStatusActive})
	s.Require().NoError(err)
	s.Len(activeList, 1)
	s.Equal(acc1.ID, activeList[0].ID)

	archivedList, err := s.repo.List(s.ctx, model.ListAccountRequest{Status: model.AccountStatusArchived})
	s.Require().NoError(err)
	s.Len(archivedList, 1)
	s.Equal(acc2.ID, archivedList[0].ID)
}

func (s *AccountRepositorySuite) TestList_CombinedFilter() {
	// Create active asset
	_, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	// Create archived asset
	acc2, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1002", Name: "Bank Lama", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)
	archivedStatus := model.AccountStatusArchived
	_, err = s.repo.Update(s.ctx, acc2.ID, model.UpdateAccountRequest{Status: &archivedStatus})
	s.Require().NoError(err)

	// Create active liability
	_, err = s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "2001", Name: "Hutang", Type: model.AccountTypeLiability, Currency: "IDR",
	})
	s.Require().NoError(err)

	res, err := s.repo.List(s.ctx, model.ListAccountRequest{
		Type:   model.AccountTypeAsset,
		Status: model.AccountStatusActive,
	})
	s.Require().NoError(err)
	s.Len(res, 1)
	s.Equal("1001", res[0].Code)
}

func (s *AccountRepositorySuite) TestList_WorkspaceIsolation() {
	_, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas Workspace 1", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)
	defer func() {
		_, _ = s.client.Account.Delete().Exec(ctx2)
	}()

	_, err = s.repo.Create(ctx2, model.CreateAccountRequest{
		Code: "1002", Name: "Kas Workspace 2", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	list1, err := s.repo.List(s.ctx, model.ListAccountRequest{})
	s.Require().NoError(err)
	s.Len(list1, 1)
	s.Equal("1001", list1[0].Code)

	list2, err := s.repo.List(ctx2, model.ListAccountRequest{})
	s.Require().NoError(err)
	s.Len(list2, 1)
	s.Equal("1002", list2[0].Code)
}

// ---------------------------------------------------------------------------
// Update Tests
// ---------------------------------------------------------------------------

func (s *AccountRepositorySuite) TestUpdate_Success() {
	created, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas Asli", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	newName := "Kas Diperbarui"
	newDesc := "Deskripsi baru"
	updated, err := s.repo.Update(s.ctx, created.ID, model.UpdateAccountRequest{
		Name:        &newName,
		Description: &newDesc,
	})
	s.Require().NoError(err)
	s.Require().NotNil(updated)
	s.Equal(newName, updated.Name)
	s.Require().NotNil(updated.Description)
	s.Equal(newDesc, *updated.Description)

	// Verify persistence in DB
	fetched, err := s.repo.GetByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(newName, fetched.Name)
	s.Require().NotNil(fetched.Description)
	s.Equal(newDesc, *fetched.Description)
}

func (s *AccountRepositorySuite) TestUpdate_Status() {
	created, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)
	s.Equal(model.AccountStatusActive, created.Status)

	archived := model.AccountStatusArchived
	updated, err := s.repo.Update(s.ctx, created.ID, model.UpdateAccountRequest{
		Status: &archived,
	})
	s.Require().NoError(err)
	s.Equal(model.AccountStatusArchived, updated.Status)
}

func (s *AccountRepositorySuite) TestUpdate_ParentID() {
	parent, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1000", Name: "Induk Akun", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	child, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Anak Akun", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)
	s.Nil(child.ParentID)

	updated, err := s.repo.Update(s.ctx, child.ID, model.UpdateAccountRequest{
		ParentID: &parent.ID,
	})
	s.Require().NoError(err)
	s.Require().NotNil(updated.ParentID)
	s.Equal(parent.ID, *updated.ParentID)
}

func (s *AccountRepositorySuite) TestUpdate_NotFound() {
	newName := "Nama Baru"
	res, err := s.repo.Update(s.ctx, uuid.New(), model.UpdateAccountRequest{
		Name: &newName,
	})
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

func (s *AccountRepositorySuite) TestUpdate_OtherWorkspace() {
	created, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas Workspace 1", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)

	newName := "Hacked Name"
	res, err := s.repo.Update(ctx2, created.ID, model.UpdateAccountRequest{
		Name: &newName,
	})
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

// ---------------------------------------------------------------------------
// Delete Tests
// ---------------------------------------------------------------------------

func (s *AccountRepositorySuite) TestDelete_Success() {
	created, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas Hapus", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	err = s.repo.Delete(s.ctx, created.ID)
	s.Require().NoError(err)

	// Fetch should return not found
	res, err := s.repo.GetByID(s.ctx, created.ID)
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

func (s *AccountRepositorySuite) TestDelete_NotFound() {
	err := s.repo.Delete(s.ctx, uuid.New())
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
}

func (s *AccountRepositorySuite) TestDelete_OtherWorkspace() {
	created, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)

	// User2 cannot delete account from Workspace 1
	err = s.repo.Delete(ctx2, created.ID)
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))

	// Account in Workspace 1 should still exist
	fetched, err := s.repo.GetByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Require().NotNil(fetched)
}

// ---------------------------------------------------------------------------
// Count Tests
// ---------------------------------------------------------------------------

func (s *AccountRepositorySuite) TestCount_Empty() {
	cnt, err := s.repo.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(0, cnt)
}

func (s *AccountRepositorySuite) TestCount_WithRecords() {
	_, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas 1", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1002", Name: "Kas 2", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	cnt, err := s.repo.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(2, cnt)
}

func (s *AccountRepositorySuite) TestCount_WorkspaceIsolation() {
	_, err := s.repo.Create(s.ctx, model.CreateAccountRequest{
		Code: "1001", Name: "Kas 1", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)
	defer func() {
		_, _ = s.client.Account.Delete().Exec(ctx2)
	}()

	_, err = s.repo.Create(ctx2, model.CreateAccountRequest{
		Code: "1002", Name: "Kas 2", Type: model.AccountTypeAsset, Currency: "IDR",
	})
	s.Require().NoError(err)

	cnt1, err := s.repo.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(1, cnt1)

	cnt2, err := s.repo.Count(ctx2)
	s.Require().NoError(err)
	s.Equal(1, cnt2)
}

// ---------------------------------------------------------------------------
// Seed Tests
// ---------------------------------------------------------------------------

func (s *AccountRepositorySuite) TestSeed_Success() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}

	res, err := s.repo.Seed(s.ctx, req)
	s.Require().NoError(err)
	s.NotEmpty(res)

	// Verify all seeded accounts belong to the user's workspace and have currency IDR
	codeMap := make(map[string]model.AccountResponse)
	for _, acc := range res {
		s.NotEqual(uuid.Nil, acc.ID)
		s.Equal(s.user.WorkspaceID, acc.WorkspaceID)
		s.Equal("IDR", acc.Currency)
		codeMap[acc.Code] = acc
	}

	// Verify hierarchical relationships:
	// In freelancer.id.json, account 1100 has parentCode 1000
	acc1000, exists := codeMap["1000"]
	s.True(exists, "account 1000 should exist in seed")
	s.Nil(acc1000.ParentID, "account 1000 should have no parent")

	acc1100, exists := codeMap["1100"]
	s.True(exists, "account 1100 should exist in seed")
	s.Require().NotNil(acc1100.ParentID, "account 1100 should have a parent")
	s.Equal(acc1000.ID, *acc1100.ParentID, "account 1100 parent should be account 1000")

	// Total count in database should match seeded count
	cnt, err := s.repo.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(len(res), cnt)
}

func (s *AccountRepositorySuite) TestSeed_DefaultCurrency() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "en",
		Currency: "", // Empty currency should default to IDR
		Force:    false,
	}

	res, err := s.repo.Seed(s.ctx, req)
	s.Require().NoError(err)
	s.NotEmpty(res)

	for _, acc := range res {
		s.Equal("IDR", acc.Currency)
	}
}

func (s *AccountRepositorySuite) TestSeed_CustomCurrency() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "en",
		Currency: "USD",
		Force:    false,
	}

	res, err := s.repo.Seed(s.ctx, req)
	s.Require().NoError(err)
	s.NotEmpty(res)

	for _, acc := range res {
		s.Equal("USD", acc.Currency)
	}
}

func (s *AccountRepositorySuite) TestSeed_ForceTrue_OverwritesExisting() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}

	res1, err := s.repo.Seed(s.ctx, req)
	s.Require().NoError(err)
	s.NotEmpty(res1)

	// Seeding again with Force: true should succeed
	reqForce := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
		Force:    true,
	}
	res2, err := s.repo.Seed(s.ctx, reqForce)
	s.Require().NoError(err)
	s.Equal(len(res1), len(res2))

	cnt, err := s.repo.Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(len(res2), cnt)
}

func (s *AccountRepositorySuite) TestSeed_ForceFalse_FailsWhenAlreadySeeded() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}

	_, err := s.repo.Seed(s.ctx, req)
	s.Require().NoError(err)

	// Second seed with Force: false will fail because code unique constraint violates
	res2, err := s.repo.Seed(s.ctx, req)
	s.Error(err)
	s.Nil(res2)
}

func (s *AccountRepositorySuite) TestSeed_InvalidProfile() {
	req := model.SeedAccountRequest{
		Profile:  "nonexistent_profile",
		Lang:     "id",
		Currency: "IDR",
	}

	res, err := s.repo.Seed(s.ctx, req)
	s.Error(err)
	s.Nil(res)
}

func (s *AccountRepositorySuite) TestSeed_InvalidLang() {
	req := model.SeedAccountRequest{
		Profile:  "freelancer",
		Lang:     "nonexistent_lang",
		Currency: "IDR",
	}

	res, err := s.repo.Seed(s.ctx, req)
	s.Error(err)
	s.Nil(res)
}

func TestAccountRepositorySuite(t *testing.T) {
	suite.Run(t, new(AccountRepositorySuite))
}

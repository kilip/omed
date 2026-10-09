package test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/service"
	"github.com/kilip/omed/finance/internal/service/test/mocks"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=../account_service.go -destination=mocks/mock_account_service.go -package=mocks

type AccountServiceSuite struct {
	suite.Suite
	ctrl         *gomock.Controller
	mockAccounts *mocks.MockAccountRepository
	mockEntries  *mocks.MockAccountEntryRepository
	service      service.AccountService
	ctx          context.Context
}

func (s *AccountServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockAccounts = mocks.NewMockAccountRepository(s.ctrl)
	s.mockEntries = mocks.NewMockAccountEntryRepository(s.ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s.service = service.NewAccountService(s.mockAccounts, s.mockEntries, logger)
	s.ctx = context.Background()
}

func (s *AccountServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *AccountServiceSuite) TestList_Success() {
	req := model.ListAccountRequest{
		Type:   model.AccountTypeAsset,
		Status: model.AccountStatusActive,
	}
	expected := []model.AccountResponse{
		{
			ID:        uuid.New(),
			Code:      "1000",
			Name:      "Kas",
			Type:      model.AccountTypeAsset,
			Currency:  "IDR",
			Status:    model.AccountStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	s.mockAccounts.EXPECT().List(s.ctx, req).Return(expected, nil)

	res, err := s.service.List(s.ctx, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *AccountServiceSuite) TestList_Error() {
	req := model.ListAccountRequest{}
	expectedErr := errors.New("database error")

	s.mockAccounts.EXPECT().List(s.ctx, req).Return(nil, expectedErr)

	res, err := s.service.List(s.ctx, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestGetByID_Success() {
	id := uuid.New()
	expected := &model.AccountResponse{
		ID:        id,
		Code:      "1001",
		Name:      "Kas Operasional",
		Type:      model.AccountTypeAsset,
		Currency:  "IDR",
		Status:    model.AccountStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.mockAccounts.EXPECT().GetByID(s.ctx, id).Return(expected, nil)

	res, err := s.service.GetByID(s.ctx, id)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *AccountServiceSuite) TestGetByID_Error() {
	id := uuid.New()

	s.mockAccounts.EXPECT().GetByID(s.ctx, id).Return(nil, core.ErrItemNotFound)

	res, err := s.service.GetByID(s.ctx, id)
	s.Error(err)
	s.ErrorIs(err, core.ErrItemNotFound)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestCreate_Success() {
	desc := "Akun Kas Utama"
	req := model.CreateAccountRequest{
		Code:        "1000",
		Name:        "Kas Utama",
		Description: &desc,
		Type:        model.AccountTypeAsset,
		Currency:    "IDR",
	}
	expected := &model.AccountResponse{
		ID:          uuid.New(),
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Currency:    req.Currency,
		Status:      model.AccountStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.mockAccounts.EXPECT().Create(s.ctx, req).Return(expected, nil)

	res, err := s.service.Create(s.ctx, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *AccountServiceSuite) TestCreate_Error() {
	req := model.CreateAccountRequest{
		Code:     "1000",
		Name:     "Kas Utama",
		Type:     model.AccountTypeAsset,
		Currency: "IDR",
	}
	expectedErr := errors.New("failed to create account")

	s.mockAccounts.EXPECT().Create(s.ctx, req).Return(nil, expectedErr)

	res, err := s.service.Create(s.ctx, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestUpdate_Success() {
	id := uuid.New()
	name := "Updated Kas Operasional"
	status := model.AccountStatusArchived
	req := model.UpdateAccountRequest{
		Name:   &name,
		Status: &status,
	}
	expected := &model.AccountResponse{
		ID:        id,
		Code:      "1001",
		Name:      name,
		Type:      model.AccountTypeAsset,
		Currency:  "IDR",
		Status:    status,
		UpdatedAt: time.Now(),
	}

	s.mockAccounts.EXPECT().Update(s.ctx, id, req).Return(expected, nil)

	res, err := s.service.Update(s.ctx, id, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *AccountServiceSuite) TestUpdate_Error() {
	id := uuid.New()
	name := "Updated Name"
	req := model.UpdateAccountRequest{
		Name: &name,
	}
	expectedErr := errors.New("update error")

	s.mockAccounts.EXPECT().Update(s.ctx, id, req).Return(nil, expectedErr)

	res, err := s.service.Update(s.ctx, id, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestDelete_Success() {
	id := uuid.New()

	s.mockAccounts.EXPECT().Delete(s.ctx, id).Return(nil)

	err := s.service.Delete(s.ctx, id)
	s.NoError(err)
}

func (s *AccountServiceSuite) TestDelete_Error() {
	id := uuid.New()
	expectedErr := errors.New("delete error")

	s.mockAccounts.EXPECT().Delete(s.ctx, id).Return(expectedErr)

	err := s.service.Delete(s.ctx, id)
	s.Error(err)
	s.Equal(expectedErr, err)
}

func (s *AccountServiceSuite) TestSeed_EntriesCountError() {
	req := model.SeedAccountRequest{
		Profile:  "standard",
		Lang:     "id",
		Currency: "IDR",
	}
	expectedErr := errors.New("entry count query failed")

	s.mockEntries.EXPECT().Count(s.ctx).Return(0, expectedErr)

	res, err := s.service.Seed(s.ctx, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestSeed_EntriesNotEmpty() {
	req := model.SeedAccountRequest{
		Profile:  "standard",
		Lang:     "id",
		Currency: "IDR",
	}

	s.mockEntries.EXPECT().Count(s.ctx).Return(5, nil)

	res, err := s.service.Seed(s.ctx, req)
	s.Error(err)
	s.ErrorIs(err, core.ErrEntriesNotEmpty)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestSeed_AccountsCountError() {
	req := model.SeedAccountRequest{
		Profile:  "standard",
		Lang:     "id",
		Currency: "IDR",
	}
	expectedErr := errors.New("account count query failed")

	s.mockEntries.EXPECT().Count(s.ctx).Return(0, nil)
	s.mockAccounts.EXPECT().Count(s.ctx).Return(0, expectedErr)

	res, err := s.service.Seed(s.ctx, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestSeed_AccountsNotEmpty_ForceFalse() {
	req := model.SeedAccountRequest{
		Profile:  "standard",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}

	s.mockEntries.EXPECT().Count(s.ctx).Return(0, nil)
	s.mockAccounts.EXPECT().Count(s.ctx).Return(10, nil)

	res, err := s.service.Seed(s.ctx, req)
	s.Error(err)
	s.ErrorIs(err, core.ErrAccountsNotEmpty)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestSeed_AccountsNotEmpty_ForceTrue_Success() {
	req := model.SeedAccountRequest{
		Profile:  "standard",
		Lang:     "id",
		Currency: "IDR",
		Force:    true,
	}
	expected := []model.AccountResponse{
		{
			ID:       uuid.New(),
			Code:     "1000",
			Name:     "Kas",
			Type:     model.AccountTypeAsset,
			Currency: "IDR",
			Status:   model.AccountStatusActive,
		},
	}

	s.mockEntries.EXPECT().Count(s.ctx).Return(0, nil)
	s.mockAccounts.EXPECT().Count(s.ctx).Return(10, nil)
	s.mockAccounts.EXPECT().Seed(s.ctx, req).Return(expected, nil)

	res, err := s.service.Seed(s.ctx, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *AccountServiceSuite) TestSeed_AccountsEmpty_ForceFalse_Success() {
	req := model.SeedAccountRequest{
		Profile:  "standard",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}
	expected := []model.AccountResponse{
		{
			ID:       uuid.New(),
			Code:     "1000",
			Name:     "Kas",
			Type:     model.AccountTypeAsset,
			Currency: "IDR",
			Status:   model.AccountStatusActive,
		},
	}

	s.mockEntries.EXPECT().Count(s.ctx).Return(0, nil)
	s.mockAccounts.EXPECT().Count(s.ctx).Return(0, nil)
	s.mockAccounts.EXPECT().Seed(s.ctx, req).Return(expected, nil)

	res, err := s.service.Seed(s.ctx, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *AccountServiceSuite) TestSeed_AccountsSeedError() {
	req := model.SeedAccountRequest{
		Profile:  "standard",
		Lang:     "id",
		Currency: "IDR",
		Force:    false,
	}
	expectedErr := errors.New("seed execution failed")

	s.mockEntries.EXPECT().Count(s.ctx).Return(0, nil)
	s.mockAccounts.EXPECT().Count(s.ctx).Return(0, nil)
	s.mockAccounts.EXPECT().Seed(s.ctx, req).Return(nil, expectedErr)

	res, err := s.service.Seed(s.ctx, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

func (s *AccountServiceSuite) TestGetSeedTemplates_Success() {
	templates, err := s.service.GetSeedTemplates(s.ctx)
	s.NoError(err)
	s.NotEmpty(templates)
	s.Equal("freelancer", templates[0].ID)
	s.Contains(templates[0].Languages, "en")
	s.Contains(templates[0].Languages, "id")
}

func (s *AccountServiceSuite) TestGetSeedPreview_Success() {
	s.mockAccounts.EXPECT().Count(s.ctx).Return(3, nil)
	s.mockEntries.EXPECT().Count(s.ctx).Return(0, nil)

	preview, err := s.service.GetSeedPreview(s.ctx, "freelancer", "en")
	s.NoError(err)
	s.NotNil(preview)
	s.Equal("freelancer", preview.Profile)
	s.Equal("en", preview.Lang)
	s.Equal(3, preview.AccountCount)
	s.Equal(0, preview.EntryCount)
	s.NotEmpty(preview.Accounts)
	s.Equal("1000", preview.Accounts[0].Code)
}

func (s *AccountServiceSuite) TestGetSeedPreview_InvalidProfile() {
	preview, err := s.service.GetSeedPreview(s.ctx, "nonexistent", "en")
	s.Error(err)
	s.Nil(preview)
}

func TestAccountServiceSuite(t *testing.T) {
	suite.Run(t, new(AccountServiceSuite))
}


package test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/service"
	"github.com/kilip/omed/finance/internal/shared"
)

type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) List(ctx context.Context, req model.ListAccountRequest) ([]model.Account, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]model.Account), args.Error(1)
}

func (m *MockAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountRepository) Create(ctx context.Context, req model.CreateAccountRequest) (*model.Account, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.Account, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockAccountRepository) Seed(ctx context.Context, req model.SeedAccountRequest) ([]model.Account, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]model.Account), args.Error(1)
}

type MockAccountEntryRepository struct {
	mock.Mock
}

func (m *MockAccountEntryRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

type AccountServiceSuite struct {
	suite.Suite
	accountRepo *MockAccountRepository
	entryRepo   *MockAccountEntryRepository
	svc         service.AccountService
}

func (s *AccountServiceSuite) SetupTest() {
	s.accountRepo = new(MockAccountRepository)
	s.entryRepo = new(MockAccountEntryRepository)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s.svc = service.NewAccountService(s.accountRepo, s.entryRepo, logger)
}

func (s *AccountServiceSuite) TestSeed_EntriesNotEmpty() {
	ctx := context.Background()
	req := model.SeedAccountRequest{Profile: "freelancer", Lang: "en", Force: true}

	s.entryRepo.On("Count", ctx).Return(3, nil)

	res, err := s.svc.Seed(ctx, req)
	assert.ErrorIs(s.T(), err, shared.ErrEntriesNotEmpty)
	assert.Nil(s.T(), res)
	s.entryRepo.AssertExpectations(s.T())
}

func (s *AccountServiceSuite) TestSeed_AccountsNotEmpty_WithoutForce() {
	ctx := context.Background()
	req := model.SeedAccountRequest{Profile: "freelancer", Lang: "en", Force: false}

	s.entryRepo.On("Count", ctx).Return(0, nil)
	s.accountRepo.On("Count", ctx).Return(5, nil)

	res, err := s.svc.Seed(ctx, req)
	assert.ErrorIs(s.T(), err, shared.ErrAccountsNotEmpty)
	assert.Nil(s.T(), res)
	s.entryRepo.AssertExpectations(s.T())
	s.accountRepo.AssertExpectations(s.T())
}

func (s *AccountServiceSuite) TestSeed_AccountsNotEmpty_WithForce() {
	ctx := context.Background()
	req := model.SeedAccountRequest{Profile: "freelancer", Lang: "en", Force: true}
	expected := []model.Account{{ID: uuid.New(), Code: "1000", Name: "Assets"}}

	s.entryRepo.On("Count", ctx).Return(0, nil)
	s.accountRepo.On("Count", ctx).Return(5, nil)
	s.accountRepo.On("Seed", ctx, req).Return(expected, nil)

	res, err := s.svc.Seed(ctx, req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expected, res)
	s.entryRepo.AssertExpectations(s.T())
	s.accountRepo.AssertExpectations(s.T())
}

func (s *AccountServiceSuite) TestSeed_EmptyAccounts_Success() {
	ctx := context.Background()
	req := model.SeedAccountRequest{Profile: "freelancer", Lang: "en", Force: false}
	expected := []model.Account{{ID: uuid.New(), Code: "1000", Name: "Assets"}}

	s.entryRepo.On("Count", ctx).Return(0, nil)
	s.accountRepo.On("Count", ctx).Return(0, nil)
	s.accountRepo.On("Seed", ctx, req).Return(expected, nil)

	res, err := s.svc.Seed(ctx, req)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expected, res)
	s.entryRepo.AssertExpectations(s.T())
	s.accountRepo.AssertExpectations(s.T())
}

func TestAccountServiceSuite(t *testing.T) {
	suite.Run(t, new(AccountServiceSuite))
}

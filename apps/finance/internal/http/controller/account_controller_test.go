package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kilip/omed/finance/internal/http/controller"
	"github.com/kilip/omed/finance/internal/model"
)

type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) List(ctx context.Context, req model.ListAccountRequest) ([]model.Account, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]model.Account), args.Error(1)
}

func (m *MockAccountService) GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountService) Create(ctx context.Context, req model.CreateAccountRequest) (*model.Account, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountService) Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.Account, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Account), args.Error(1)
}

func (m *MockAccountService) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountService) Seed(ctx context.Context, req model.SeedAccountRequest) ([]model.Account, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]model.Account), args.Error(1)
}

func TestAccountController_Seed(t *testing.T) {
	t.Run("success seeding", func(t *testing.T) {
		app := fiber.New()

		mockSvc := new(MockAccountService)
		c := controller.NewAccountController(mockSvc)

		// Register route directly without RequirePermission middleware for unit test
		app.Post("/accounts/seed", c.Seed)

		expectedAccounts := []model.Account{
			{
				ID:   uuid.New(),
				Code: "1000",
				Name: "Assets",
				Type: model.AccountTypeAsset,
			},
		}

		mockSvc.On("Seed", mock.Anything, model.SeedAccountRequest{
			Profile: "freelancer",
			Lang:    "en",
		}).Return(expectedAccounts, nil)

		body, _ := json.Marshal(model.SeedAccountRequest{
			Profile: "freelancer",
			Lang:    "en",
		})
		req := httptest.NewRequest("POST", "/accounts/seed", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var webResp model.WebResponse[[]model.Account]
		err = json.NewDecoder(resp.Body).Decode(&webResp)
		assert.NoError(t, err)
		assert.Len(t, webResp.Data, 1)
		if len(webResp.Data) > 0 {
			assert.Equal(t, "1000", webResp.Data[0].Code)
		}
		mockSvc.AssertExpectations(t)
	})
}

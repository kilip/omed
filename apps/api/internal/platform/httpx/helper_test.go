package httpx_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/suite"
)

type HelperSuite struct {
	suite.Suite
}

type dummyEntity struct {
	ID   int64
	Name string
}

type dummyUUIDEntity struct {
	ID   uuid.UUID
	Name string
}

type dummyReq struct {
	Name string `json:"name"`
}

type dummyRes struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type dummySearchReq struct {
	Query string `json:"query"`
}

func toDummyRes(e dummyEntity) dummyRes {
	return dummyRes{
		ID:   fmt.Sprintf("%d", e.ID),
		Name: e.Name,
	}
}

func toDummyUUIDRes(e dummyUUIDEntity) dummyRes {
	return dummyRes{
		ID:   e.ID.String(),
		Name: e.Name,
	}
}

// ======================== DoGetByID Tests ========================

func (s *HelperSuite) Test_DoGetByID_Int64_Success() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handler := func(ctx context.Context, id int64) (dummyEntity, error) {
		s.Equal(int64(42), id)
		s.NotNil(ctx)
		return dummyEntity{ID: id, Name: "Found Item"}, nil
	}

	app.Get("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoGetByID(c, handler, toDummyRes)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/items/42", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NoError(capturedErr)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res httpx.WebResponse[dummyRes]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&res))
	s.Equal("42", res.Data.ID)
	s.Equal("Found Item", res.Data.Name)
}

func (s *HelperSuite) Test_DoGetByID_UUID_Success() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	targetUUID := uuid.New()
	handler := func(ctx context.Context, id uuid.UUID) (dummyUUIDEntity, error) {
		s.Equal(targetUUID, id)
		return dummyUUIDEntity{ID: id, Name: "UUID Item"}, nil
	}

	app.Get("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoGetByID(c, handler, toDummyUUIDRes)
	})

	req := httptest.NewRequest(fiber.MethodGet, fmt.Sprintf("/items/%s", targetUUID.String()), nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NoError(capturedErr)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res httpx.WebResponse[dummyRes]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&res))
	s.Equal(targetUUID.String(), res.Data.ID)
	s.Equal("UUID Item", res.Data.Name)
}

func (s *HelperSuite) Test_DoGetByID_InvalidID() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handlerCalled := false
	handler := func(ctx context.Context, id int64) (dummyEntity, error) {
		handlerCalled = true
		return dummyEntity{}, nil
	}

	app.Get("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoGetByID(c, handler, toDummyRes)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/items/not-a-number", nil)
	_, err := app.Test(req)
	s.Require().NoError(err)

	s.False(handlerCalled)
	s.ErrorIs(capturedErr, base.ErrInvalidID)
}

func (s *HelperSuite) Test_DoGetByID_HandlerError() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	expectedErr := base.ErrNotFound
	handler := func(ctx context.Context, id int64) (dummyEntity, error) {
		return dummyEntity{}, expectedErr
	}

	app.Get("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoGetByID(c, handler, toDummyRes)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/items/42", nil)
	_, err := app.Test(req)
	s.Require().NoError(err)

	s.ErrorIs(capturedErr, expectedErr)
}

// ======================== DoCreate Tests ========================

func (s *HelperSuite) Test_DoCreate_Success() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handler := func(ctx context.Context, req dummyReq) (dummyEntity, error) {
		s.Equal("Created Name", req.Name)
		return dummyEntity{ID: 100, Name: req.Name}, nil
	}

	app.Post("/items", func(c fiber.Ctx) error {
		return httpx.DoCreate(c, handler, toDummyRes)
	})

	reqBody := `{"name":"Created Name"}`
	req := httptest.NewRequest(fiber.MethodPost, "/items", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NoError(capturedErr)
	s.Equal(fiber.StatusCreated, resp.StatusCode)

	var res httpx.WebResponse[dummyRes]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&res))
	s.Equal("100", res.Data.ID)
	s.Equal("Created Name", res.Data.Name)
}

func (s *HelperSuite) Test_DoCreate_BindError() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handlerCalled := false
	handler := func(ctx context.Context, req dummyReq) (dummyEntity, error) {
		handlerCalled = true
		return dummyEntity{}, nil
	}

	app.Post("/items", func(c fiber.Ctx) error {
		return httpx.DoCreate(c, handler, toDummyRes)
	})

	reqBody := `{"name": invalid-json`
	req := httptest.NewRequest(fiber.MethodPost, "/items", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req)
	s.Require().NoError(err)

	s.False(handlerCalled)
	s.Error(capturedErr)
}

func (s *HelperSuite) Test_DoCreate_HandlerError() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	expectedErr := errors.New("cannot create item")
	handler := func(ctx context.Context, req dummyReq) (dummyEntity, error) {
		return dummyEntity{}, expectedErr
	}

	app.Post("/items", func(c fiber.Ctx) error {
		return httpx.DoCreate(c, handler, toDummyRes)
	})

	reqBody := `{"name":"Any Name"}`
	req := httptest.NewRequest(fiber.MethodPost, "/items", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req)
	s.Require().NoError(err)

	s.ErrorIs(capturedErr, expectedErr)
}

// ======================== DoUpdate Tests ========================

func (s *HelperSuite) Test_DoUpdate_Int64_Success() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handler := func(ctx context.Context, id int64, req dummyReq) (dummyEntity, error) {
		s.Equal(int64(55), id)
		s.Equal("Updated Name", req.Name)
		return dummyEntity{ID: id, Name: req.Name}, nil
	}

	app.Put("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoUpdate(c, handler, toDummyRes)
	})

	reqBody := `{"name":"Updated Name"}`
	req := httptest.NewRequest(fiber.MethodPut, "/items/55", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NoError(capturedErr)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res httpx.WebResponse[dummyRes]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&res))
	s.Equal("55", res.Data.ID)
	s.Equal("Updated Name", res.Data.Name)
}

func (s *HelperSuite) Test_DoUpdate_UUID_Success() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	targetUUID := uuid.New()
	handler := func(ctx context.Context, id uuid.UUID, req dummyReq) (dummyUUIDEntity, error) {
		s.Equal(targetUUID, id)
		s.Equal("Updated UUID", req.Name)
		return dummyUUIDEntity{ID: id, Name: req.Name}, nil
	}

	app.Put("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoUpdate(c, handler, toDummyUUIDRes)
	})

	reqBody := `{"name":"Updated UUID"}`
	req := httptest.NewRequest(fiber.MethodPut, fmt.Sprintf("/items/%s", targetUUID.String()), strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NoError(capturedErr)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res httpx.WebResponse[dummyRes]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&res))
	s.Equal(targetUUID.String(), res.Data.ID)
	s.Equal("Updated UUID", res.Data.Name)
}

func (s *HelperSuite) Test_DoUpdate_InvalidID() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handlerCalled := false
	handler := func(ctx context.Context, id int64, req dummyReq) (dummyEntity, error) {
		handlerCalled = true
		return dummyEntity{}, nil
	}

	app.Put("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoUpdate(c, handler, toDummyRes)
	})

	reqBody := `{"name":"Updated Name"}`
	req := httptest.NewRequest(fiber.MethodPut, "/items/not-a-number", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req)
	s.Require().NoError(err)

	s.False(handlerCalled)
	s.ErrorIs(capturedErr, base.ErrInvalidID)
}

func (s *HelperSuite) Test_DoUpdate_BindError() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handlerCalled := false
	handler := func(ctx context.Context, id int64, req dummyReq) (dummyEntity, error) {
		handlerCalled = true
		return dummyEntity{}, nil
	}

	app.Put("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoUpdate(c, handler, toDummyRes)
	})

	reqBody := `{"name": invalid-json`
	req := httptest.NewRequest(fiber.MethodPut, "/items/10", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req)
	s.Require().NoError(err)

	s.False(handlerCalled)
	s.Error(capturedErr)
}

func (s *HelperSuite) Test_DoUpdate_HandlerError() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	expectedErr := errors.New("update conflict")
	handler := func(ctx context.Context, id int64, req dummyReq) (dummyEntity, error) {
		return dummyEntity{}, expectedErr
	}

	app.Put("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoUpdate(c, handler, toDummyRes)
	})

	reqBody := `{"name":"New Name"}`
	req := httptest.NewRequest(fiber.MethodPut, "/items/10", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req)
	s.Require().NoError(err)

	s.ErrorIs(capturedErr, expectedErr)
}

// ======================== DoDelete Tests ========================

func (s *HelperSuite) Test_DoDelete_Int64_Success() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	var deletedID int64
	handler := func(ctx context.Context, id int64) error {
		deletedID = id
		return cSendNoContentHelper(ctx)
	}

	app.Delete("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoDelete(c, handler)
	})

	req := httptest.NewRequest(fiber.MethodDelete, "/items/77", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NoError(capturedErr)
	s.Equal(int64(77), deletedID)
	s.Equal(fiber.StatusNoContent, resp.StatusCode)
}

func (s *HelperSuite) Test_DoDelete_UUID_Success() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	targetUUID := uuid.New()
	var deletedID uuid.UUID
	handler := func(ctx context.Context, id uuid.UUID) error {
		deletedID = id
		return nil
	}

	app.Delete("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoDelete(c, handler)
	})

	req := httptest.NewRequest(fiber.MethodDelete, fmt.Sprintf("/items/%s", targetUUID.String()), nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NoError(capturedErr)
	s.Equal(targetUUID, deletedID)
	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *HelperSuite) Test_DoDelete_InvalidID() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handlerCalled := false
	handler := func(ctx context.Context, id int64) error {
		handlerCalled = true
		return nil
	}

	app.Delete("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoDelete(c, handler)
	})

	req := httptest.NewRequest(fiber.MethodDelete, "/items/not-a-number", nil)
	_, err := app.Test(req)
	s.Require().NoError(err)

	s.False(handlerCalled)
	s.ErrorIs(capturedErr, base.ErrInvalidID)
}

func (s *HelperSuite) Test_DoDelete_HandlerError() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	expectedErr := errors.New("cannot delete item")
	handler := func(ctx context.Context, id int64) error {
		return expectedErr
	}

	app.Delete("/items/:id", func(c fiber.Ctx) error {
		return httpx.DoDelete(c, handler)
	})

	req := httptest.NewRequest(fiber.MethodDelete, "/items/88", nil)
	_, err := app.Test(req)
	s.Require().NoError(err)

	s.ErrorIs(capturedErr, expectedErr)
}

// ======================== DoSearch Tests ========================

func (s *HelperSuite) Test_DoSearch_Success() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handler := func(ctx context.Context, req dummySearchReq) ([]dummyEntity, string, error) {
		s.Equal("query-text", req.Query)
		items := []dummyEntity{
			{ID: 1, Name: "Result 1"},
			{ID: 2, Name: "Result 2"},
		}
		return items, "cursor-next-page", nil
	}

	app.Post("/items/search", func(c fiber.Ctx) error {
		return httpx.DoSearch(c, handler, toDummyRes)
	})

	reqBody := `{"query":"query-text"}`
	req := httptest.NewRequest(fiber.MethodPost, "/items/search", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NoError(capturedErr)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res httpx.WebResponse[[]dummyRes]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&res))
	s.Require().Len(res.Data, 2)
	s.Equal("1", res.Data[0].ID)
	s.Equal("Result 1", res.Data[0].Name)
	s.Equal("2", res.Data[1].ID)
	s.Equal("Result 2", res.Data[1].Name)
	s.Equal("cursor-next-page", res.Meta.Cursor)
}

func (s *HelperSuite) Test_DoSearch_BindError() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	handlerCalled := false
	handler := func(ctx context.Context, req dummySearchReq) ([]dummyEntity, string, error) {
		handlerCalled = true
		return nil, "", nil
	}

	app.Post("/items/search", func(c fiber.Ctx) error {
		return httpx.DoSearch(c, handler, toDummyRes)
	})

	reqBody := `{"query": invalid-json`
	req := httptest.NewRequest(fiber.MethodPost, "/items/search", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req)
	s.Require().NoError(err)

	s.False(handlerCalled)
	s.Error(capturedErr)
}

func (s *HelperSuite) Test_DoSearch_HandlerError() {
	var capturedErr error
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			capturedErr = err
			return fiber.DefaultErrorHandler(c, err)
		},
	})

	expectedErr := errors.New("search failed")
	handler := func(ctx context.Context, req dummySearchReq) ([]dummyEntity, string, error) {
		return nil, "", expectedErr
	}

	app.Post("/items/search", func(c fiber.Ctx) error {
		return httpx.DoSearch(c, handler, toDummyRes)
	})

	reqBody := `{"query":"query-text"}`
	req := httptest.NewRequest(fiber.MethodPost, "/items/search", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	_, err := app.Test(req)
	s.Require().NoError(err)

	s.ErrorIs(capturedErr, expectedErr)
}

func cSendNoContentHelper(ctx context.Context) error {
	if c, ok := ctx.(fiber.Ctx); ok {
		return c.SendStatus(fiber.StatusNoContent)
	}
	return nil
}

func TestHelperSuite(t *testing.T) {
	suite.Run(t, &HelperSuite{})
}

package config_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/config"
	"github.com/stretchr/testify/suite"
)

type AppSuite struct {
	suite.Suite
	mockAuth *testAuthMock
	logger   *slog.Logger
}

func (s *AppSuite) SetupSuite() {
	s.mockAuth = newTestAuthMock(s.T())
	s.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (s *AppSuite) TearDownSuite() {
	if s.mockAuth != nil && s.mockAuth.server != nil {
		s.mockAuth.server.Close()
	}
}

type trackingModule struct {
	name         string
	preBootErr   error
	bootErr      error
	orderTracker *[]string
}

func (m *trackingModule) PreBoot() error {
	if m.orderTracker != nil {
		*m.orderTracker = append(*m.orderTracker, m.name+":preboot")
	}
	return m.preBootErr
}

func (m *trackingModule) Boot() error {
	if m.orderTracker != nil {
		*m.orderTracker = append(*m.orderTracker, m.name+":boot")
	}
	return m.bootErr
}

type bootOnlyModule struct {
	name         string
	bootErr      error
	orderTracker *[]string
}

func (m *bootOnlyModule) Boot() error {
	if m.orderTracker != nil {
		*m.orderTracker = append(*m.orderTracker, m.name+":boot")
	}
	return m.bootErr
}

func (s *AppSuite) Test_New() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config: config.Config{
			JWKSUrl: s.mockAuth.url,
			AuthUrl: "http://auth.local",
		},
	}

	app := config.New(state)
	s.NotNil(app.State().Authz)
	s.Equal(state.FiberApp, app.State().FiberApp)
	s.Equal(state.Logger, app.State().Logger)
	s.Equal(state.Config, app.State().Config)
	s.Require().Len(app.Modules(), 1)
}

func (s *AppSuite) Test_Boot_DefaultApp_WithBaseMiddleware() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config: config.Config{
			JWKSUrl: s.mockAuth.url,
			AuthUrl: "http://auth.local",
		},
	}

	app := config.New(state)
	err := app.Boot()
	s.Require().NoError(err)

	// Since requestid middleware was registered by initBaseMiddleware,
	// incoming requests receive an X-Request-Id header in response.
	req := httptest.NewRequest(fiber.MethodGet, "/internal/any", nil)
	resp, err := fApp.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.NotEmpty(resp.Header.Get("X-Request-Id"))
}

func (s *AppSuite) Test_Boot_LifecycleOrder() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config: config.Config{
			JWKSUrl: s.mockAuth.url,
			AuthUrl: "http://auth.local",
		},
	}

	var executionOrder []string
	m1 := &trackingModule{name: "m1", orderTracker: &executionOrder}
	m2 := &bootOnlyModule{name: "m2", orderTracker: &executionOrder}
	m3 := &trackingModule{name: "m3", orderTracker: &executionOrder}

	app := config.NewWithModules(state, []config.Module{m1, m2, m3})
	err := app.Boot()
	s.Require().NoError(err)

	expectedOrder := []string{
		"m1:preboot",
		"m3:preboot",
		"m1:boot",
		"m2:boot",
		"m3:boot",
	}
	s.Equal(expectedOrder, executionOrder)
}

func (s *AppSuite) Test_Boot_PreBootError() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config: config.Config{
			JWKSUrl: s.mockAuth.url,
			AuthUrl: "http://auth.local",
		},
	}

	errPreBoot := errors.New("preboot failed")
	m1 := &trackingModule{name: "m1", preBootErr: errPreBoot}
	m2 := &trackingModule{name: "m2"}

	app := config.NewWithModules(state, []config.Module{m1, m2})
	err := app.Boot()
	s.Require().Error(err)
	s.True(errors.Is(err, errPreBoot))
}

func (s *AppSuite) Test_Boot_BootError() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config: config.Config{
			JWKSUrl: s.mockAuth.url,
			AuthUrl: "http://auth.local",
		},
	}

	errBoot := errors.New("boot failed")
	m1 := &trackingModule{name: "m1"}
	m2 := &trackingModule{name: "m2", bootErr: errBoot}

	app := config.NewWithModules(state, []config.Module{m1, m2})
	err := app.Boot()
	s.Require().Error(err)
	s.True(errors.Is(err, errBoot))
}

func (s *AppSuite) Test_Boot_PreBootAndBootErrorsJoined() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config: config.Config{
			JWKSUrl: s.mockAuth.url,
			AuthUrl: "http://auth.local",
		},
	}

	errPreBoot := errors.New("preboot failed")
	errBoot := errors.New("boot failed")
	m1 := &trackingModule{name: "m1", preBootErr: errPreBoot, bootErr: errBoot}

	app := config.NewWithModules(state, []config.Module{m1})
	err := app.Boot()
	s.Require().Error(err)
	s.True(errors.Is(err, errPreBoot))
	s.True(errors.Is(err, errBoot))
}

func (s *AppSuite) Test_CheckJWKS_Success() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := config.CheckJWKS(ctx, srv.URL, s.logger)
	s.Require().NoError(err)
}

func (s *AppSuite) Test_CheckJWKS_RetryAndContextCancelled() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := config.CheckJWKS(ctx, srv.URL, s.logger)
	s.Require().Error(err)
	s.True(errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled))
}

func (s *AppSuite) Test_WaitForJWKS_Success() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Config{JWKSUrl: srv.URL}
	s.NotPanics(func() {
		config.WaitForJWKS(cfg, s.logger)
	})
}

func (s *AppSuite) Test_Run_GracefulShutdown() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config: config.Config{
			JWKSUrl: s.mockAuth.url,
			AuthUrl: "http://auth.local",
		},
	}

	app := config.New(state)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(150*time.Millisecond, cancel)

	err := app.Run(ctx)
	s.NoError(err)
}

func TestAppSuite(t *testing.T) {
	suite.Run(t, new(AppSuite))
}

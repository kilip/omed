package config_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type AppSuite struct {
	suite.Suite
	logger *slog.Logger
	entCli *ent.Client
}

func (s *AppSuite) SetupSuite() {
	s.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	s.entCli = testutil.CreateTestDB("file:config_app_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
}

func (s *AppSuite) TearDownSuite() {
	if s.entCli != nil {
		_ = s.entCli.Close()
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
		EntCli:   s.entCli,
		Config: config.Config{},
	}

	app := config.New(state)
	s.NotNil(app.State().Authz)
	s.Equal(state.FiberApp, app.State().FiberApp)
	s.Equal(state.Logger, app.State().Logger)
	s.Equal(state.Config, app.State().Config)
	s.Require().Len(app.Modules(), 3)
}

func (s *AppSuite) Test_Boot_DefaultApp_WithBaseMiddleware() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   s.entCli,
		Config: config.Config{},
	}

	app := config.New(state)
	err := app.Boot()
	s.Require().NoError(err)

	// Since requestid middleware was registered by initBaseMiddleware,
	// incoming requests receive an X-Request-Id header in response.
	req := httptest.NewRequest(fiber.MethodGet, "/internal/any", nil)
	ctx := context.WithValue(req.Context(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:    1,
		Name:  "Test User",
		Roles: []base.UserRole{base.UserRoleUser},
	})
	req = req.WithContext(ctx)
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
		Config: config.Config{},
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
		Config: config.Config{},
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
		Config: config.Config{},
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
		Config: config.Config{},
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

func (s *AppSuite) Test_Run_GracefulShutdown() {
	if runtime.GOOS == "windows" {
		s.T().Skip("skip test on windows")
	}
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   s.entCli,
		Config: config.Config{},
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

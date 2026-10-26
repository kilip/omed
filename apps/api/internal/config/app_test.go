package config_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/kilip/omed/api/internal/platform/s3x"
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

type spyMailer struct {
	mailx.Mailer
	closed  atomic.Bool
	onClose func()
}

func (m *spyMailer) Close() error {
	m.closed.Store(true)
	if m.onClose != nil {
		m.onClose()
	}
	return m.Mailer.Close()
}

type spyStorage struct {
	s3x.Storage
	closed  atomic.Bool
	onClose func()
}

func (s *spyStorage) Close() error {
	s.closed.Store(true)
	if s.onClose != nil {
		s.onClose()
	}
	return s.Storage.Close()
}

func getFreePort(t testing.TB) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to acquire free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

func (s *AppSuite) Test_New() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   s.entCli,
		Config:   config.Config{},
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
		Config:   config.Config{},
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

func (s *AppSuite) Test_Boot_TestRoutesWarning_Enabled() {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	cfg := config.Config{
		Env: "development",
	}
	fApp := config.GetFiber(cfg, logger)
	state := config.State{
		FiberApp: fApp,
		Logger:   logger,
		EntCli:   s.entCli,
		Config:   cfg,
	}

	app := config.New(state)
	err := app.Boot()
	s.Require().NoError(err)

	s.Contains(buf.String(), "⚠️ WARNING: Test endpoints enabled (/test/*). DO NOT ENABLE IN PRODUCTION!")

	// Verify test routes are mounted and accessible without auth
	req := httptest.NewRequest(fiber.MethodGet, "/test/mailer/messages", nil)
	resp, err := fApp.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *AppSuite) Test_Boot_TestRoutesWarning_DisabledInProduction() {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	cfg := config.Config{
		Env: "production",
	}
	fApp := config.GetFiber(cfg, logger)
	state := config.State{
		FiberApp: fApp,
		Logger:   logger,
		EntCli:   s.entCli,
		Config:   cfg,
	}

	app := config.New(state)
	err := app.Boot()
	s.Require().NoError(err)

	s.NotContains(buf.String(), "Test endpoints enabled")

	// Verify test routes are not accessible without auth in production (firewall returns 401)
	req := httptest.NewRequest(fiber.MethodGet, "/test/mailer/messages", nil)
	resp, err := fApp.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AppSuite) Test_Boot_LifecycleOrder() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config:   config.Config{},
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
		Config:   config.Config{},
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
		Config:   config.Config{},
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
		Config:   config.Config{},
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

func (s *AppSuite) Test_Run_ReturnsBootError() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config:   config.Config{},
	}

	errBoot := errors.New("boot failure")
	m := &trackingModule{name: "m1", bootErr: errBoot}

	app := config.NewWithModules(state, []config.Module{m})

	ctx := context.Background()
	err := app.Run(ctx)
	s.Require().Error(err)
	s.True(errors.Is(err, errBoot))
}

func (s *AppSuite) Test_Run_ReturnsPreBootError() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		Config:   config.Config{},
	}

	errPreBoot := errors.New("preboot failure")
	m := &trackingModule{name: "m1", preBootErr: errPreBoot}

	app := config.NewWithModules(state, []config.Module{m})

	ctx := context.Background()
	err := app.Run(ctx)
	s.Require().Error(err)
	s.True(errors.Is(err, errPreBoot))
}

func (s *AppSuite) Test_Run_GracefulShutdown() {
	fApp := fiber.New()
	tempCli := testutil.CreateTestDB("file:config_graceful_shutdown?mode=memory&cache=shared&_fk=1")
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   tempCli,
		Config: config.Config{
			Port: getFreePort(s.T()),
		},
	}

	app := config.New(state)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(150*time.Millisecond, cancel)

	err := app.Run(ctx)
	s.NoError(err)
}

func (s *AppSuite) Test_Run_ListensOnConfiguredPort() {
	port := getFreePort(s.T())
	fApp := fiber.New()

	listenData := make(chan fiber.ListenData, 1)
	fApp.Hooks().OnListen(func(data fiber.ListenData) error {
		listenData <- data
		return nil
	})

	tempCli := testutil.CreateTestDB("file:config_port_test?mode=memory&cache=shared&_fk=1")
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   tempCli,
		Config: config.Config{
			Port: port,
		},
	}
	app := config.New(state)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run(ctx)
	}()

	select {
	case data := <-listenData:
		s.Equal(strconv.Itoa(port), data.Port)
	case <-time.After(5 * time.Second):
		s.Fail("server timed out waiting to listen")
	}

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	cancel()
	select {
	case runErr := <-runErrCh:
		s.NoError(runErr)
	case <-time.After(5 * time.Second):
		s.Fail("timeout waiting for app.Run to terminate")
	}
}

func (s *AppSuite) Test_Run_GracefulShutdown_Order() {
	port := getFreePort(s.T())
	fApp := fiber.New()

	var mu sync.Mutex
	var order []string
	record := func(step string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, step)
	}

	listenStarted := make(chan struct{})
	fApp.Hooks().OnListen(func(data fiber.ListenData) error {
		close(listenStarted)
		return nil
	})

	fApp.Hooks().OnPostShutdown(func(err error) error {
		record("fiber:postshutdown")
		return nil
	})

	inFlightStarted := make(chan struct{})
	inFlightRelease := make(chan struct{})

	fApp.Get("/in-flight", func(c fiber.Ctx) error {
		close(inFlightStarted)
		<-inFlightRelease

		mu.Lock()
		defer mu.Unlock()
		if slices.Contains(order, "fiber:postshutdown") ||
			slices.Contains(order, "mailer:close") ||
			slices.Contains(order, "storage:close") {
			return c.Status(fiber.StatusInternalServerError).SendString("resources closed prematurely")
		}
		return c.SendString("ok")
	})

	baseMailer, err := mailx.New(mailx.Config{Driver: "memory"})
	s.Require().NoError(err)
	mailer := &spyMailer{
		Mailer: baseMailer,
		onClose: func() {
			record("mailer:close")
		},
	}

	baseStorage, err := s3x.New(s3x.Config{Driver: "memory"})
	s.Require().NoError(err)
	storage := &spyStorage{
		Storage: baseStorage,
		onClose: func() {
			record("storage:close")
		},
	}

	tempEnt := testutil.CreateTestDB("file:config_shutdown_order?mode=memory&cache=shared&_fk=1")

	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   tempEnt,
		Mailer:   mailer,
		Storage:  storage,
		Config: config.Config{
			Port: port,
		},
	}

	app := config.New(state)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run(ctx)
	}()

	select {
	case <-listenStarted:
	case <-time.After(5 * time.Second):
		s.Fail("server timed out waiting to listen")
	}

	// Trigger in-flight request
	inFlightDone := make(chan struct{})
	var respCode int
	go func() {
		defer close(inFlightDone)
		resp, reqErr := http.Get(fmt.Sprintf("http://127.0.0.1:%d/in-flight", port))
		if reqErr == nil {
			respCode = resp.StatusCode
			_ = resp.Body.Close()
		}
	}()

	// Wait until handler is actively processing inside Fiber
	select {
	case <-inFlightStarted:
	case <-time.After(5 * time.Second):
		s.Fail("timeout waiting for in-flight request to start")
	}

	// Now initiate shutdown
	cancel()

	// Wait briefly to allow shutdown goroutine to begin and call FiberApp.ShutdownWithContext
	time.Sleep(50 * time.Millisecond)

	// Release in-flight handler so it can finish
	close(inFlightRelease)

	select {
	case <-inFlightDone:
	case <-time.After(5 * time.Second):
		s.Fail("timeout waiting for in-flight request to finish")
	}
	s.Equal(http.StatusOK, respCode)

	select {
	case runErr := <-runErrCh:
		s.NoError(runErr)
	case <-time.After(5 * time.Second):
		s.Fail("timeout waiting for app.Run to terminate")
	}

	// Verify order: Fiber shuts down first, then mailer and storage are closed
	mu.Lock()
	finalOrder := slices.Clone(order)
	mu.Unlock()

	s.Equal([]string{"fiber:postshutdown", "mailer:close", "storage:close"}, finalOrder)
	s.True(mailer.closed.Load())
	s.True(storage.closed.Load())

	// Verify ent client was closed as part of shutdown
	_, dbErr := tempEnt.User.Query().Count(context.Background())
	s.Error(dbErr, "ent client should be closed after graceful shutdown")
}

func (s *AppSuite) Test_Healthz_Endpoint() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   s.entCli,
		Config:   config.Config{},
	}
	app := config.New(state)
	s.Require().NoError(app.Boot())

	req := httptest.NewRequest(fiber.MethodGet, "/healthz", nil)
	resp, err := fApp.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	var body map[string]any
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&body))
	s.Equal("ok", body["status"])
}

func (s *AppSuite) Test_Readyz_Endpoint_Healthy() {
	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   s.entCli,
		Config:   config.Config{},
	}
	app := config.New(state)
	s.Require().NoError(app.Boot())

	req := httptest.NewRequest(fiber.MethodGet, "/readyz", nil)
	resp, err := fApp.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	var body map[string]any
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&body))
	s.Equal("ok", body["status"])
}

func (s *AppSuite) Test_Readyz_Endpoint_DBUnavailable() {
	// Create an isolated client, boot app, then close DB to simulate outage
	tempCli := testutil.CreateTestDB("file:config_readyz_fail?mode=memory&cache=shared&_fk=1")

	fApp := fiber.New()
	state := config.State{
		FiberApp: fApp,
		Logger:   s.logger,
		EntCli:   tempCli,
		Config:   config.Config{},
	}
	app := config.New(state)
	s.Require().NoError(app.Boot())

	// Close database after boot to simulate failure
	s.Require().NoError(tempCli.Close())

	req := httptest.NewRequest(fiber.MethodGet, "/readyz", nil)
	resp, err := fApp.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusServiceUnavailable, resp.StatusCode)
	var body map[string]any
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&body))
	s.Equal("unavailable", body["status"])
	s.NotEmpty(body["error"])
}

func (s *AppSuite) Test_PingDB_NilClient() {
	err := config.PingDB(context.Background(), nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "database client is not initialized")
}

func TestAppSuite(t *testing.T) {
	suite.Run(t, new(AppSuite))
}

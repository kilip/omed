package service_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type SettingServiceSuite struct {
	suite.Suite
	client      *ent.Client
	workspaceID int64
	userID      int64
	svc         service.SettingService
	accountRepo repository.AccountRepository
}

func TestSettingServiceSuite(t *testing.T) {
	suite.Run(t, new(SettingServiceSuite))
}

func (s *SettingServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:setting_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
	settingRepo := repository.NewSettingRepository(s.client)
	s.accountRepo = repository.NewAccountRepository(s.client)
	currencyRepo := repository.NewCurrencyRepository(s.client)
	s.svc = service.NewSettingService(settingRepo, s.accountRepo, service.WithSettingCurrencyRepository(currencyRepo))
}

func (s *SettingServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *SettingServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.FinSetting.Delete().Exec(ctx)
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Admin").
		SetEmail("admin@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Admin",
	})
	ws, err := s.client.Workspace.Create().
		SetName("Test WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = ws.ID
}

func (s *SettingServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Admin",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *SettingServiceSuite) Test_GetFXAccounts_Empty() {
	ctx := s.ctx()
	resp, err := s.svc.GetFXAccounts(ctx, s.workspaceID)
	s.Require().NoError(err)
	s.Nil(resp.FXGainAccountID)
	s.Nil(resp.FXLossAccountID)
	s.Nil(resp.FXRoundingAccountID)
}

func (s *SettingServiceSuite) Test_UpdateFXAccounts_ValidationAndSuccess() {
	ctx := s.ctx()

	// 1. Non-existent account ID
	badID := int64(999999)
	_, err := s.svc.UpdateFXAccounts(ctx, s.workspaceID, model.UpdateFXAccountsRequest{
		FXGainAccountID: &badID,
	})
	s.Require().ErrorIs(err, domain.ErrAccountNotFound)

	// 2. Inactive account
	inactiveAcc, err := s.client.Account.Create().
		SetCode("7210").
		SetName("Gain").
		SetType("REVENUE").
		SetNormalBalance("CREDIT").
		SetIsActive(false).
		SetIsPostable(true).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.UpdateFXAccounts(ctx, s.workspaceID, model.UpdateFXAccountsRequest{
		FXGainAccountID: &inactiveAcc.ID,
	})
	s.Require().ErrorIs(err, domain.ErrFXAccountInvalid)

	// 3. Not postable account
	headerAcc, err := s.client.Account.Create().
		SetCode("7200").
		SetName("Gain Group").
		SetType("REVENUE").
		SetNormalBalance("CREDIT").
		SetIsActive(true).
		SetIsPostable(false).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.UpdateFXAccounts(ctx, s.workspaceID, model.UpdateFXAccountsRequest{
		FXGainAccountID: &headerAcc.ID,
	})
	s.Require().ErrorIs(err, domain.ErrFXAccountInvalid)

	// 4. Valid active postable accounts
	gainAcc, err := s.client.Account.Create().
		SetCode("7211").
		SetName("Realized FX Gain").
		SetType("REVENUE").
		SetNormalBalance("CREDIT").
		SetIsActive(true).
		SetIsPostable(true).
		Save(ctx)
	s.Require().NoError(err)

	lossAcc, err := s.client.Account.Create().
		SetCode("7311").
		SetName("Realized FX Loss").
		SetType("EXPENSE").
		SetNormalBalance("DEBIT").
		SetIsActive(true).
		SetIsPostable(true).
		Save(ctx)
	s.Require().NoError(err)

	roundAcc, err := s.client.Account.Create().
		SetCode("7401").
		SetName("FX Rounding").
		SetType("EXPENSE").
		SetNormalBalance("DEBIT").
		SetIsActive(true).
		SetIsPostable(true).
		Save(ctx)
	s.Require().NoError(err)

	updateResp, err := s.svc.UpdateFXAccounts(ctx, s.workspaceID, model.UpdateFXAccountsRequest{
		FXGainAccountID:     &gainAcc.ID,
		FXLossAccountID:     &lossAcc.ID,
		FXRoundingAccountID: &roundAcc.ID,
	})
	s.Require().NoError(err)
	s.Equal(&gainAcc.ID, updateResp.FXGainAccountID)
	s.Equal(&lossAcc.ID, updateResp.FXLossAccountID)
	s.Equal(&roundAcc.ID, updateResp.FXRoundingAccountID)

	// Verify retrieval
	getResp, err := s.svc.GetFXAccounts(ctx, s.workspaceID)
	s.Require().NoError(err)
	s.Equal(&gainAcc.ID, getResp.FXGainAccountID)
	s.Equal(&lossAcc.ID, getResp.FXLossAccountID)
	s.Equal(&roundAcc.ID, getResp.FXRoundingAccountID)
}

func (s *SettingServiceSuite) Test_GetSettings_Default() {
	ctx := s.ctx()
	resp, err := s.svc.GetSettings(ctx, s.workspaceID)
	s.Require().NoError(err)
	s.False(resp.Onboarding)
	s.Equal("comma", resp.DecimalStyle)
	s.Equal("IDR", resp.BaseCurrency)
	s.Nil(resp.FXGainAccountID)
	s.Nil(resp.FXLossAccountID)
	s.Nil(resp.FXRoundingAccountID)
}

func (s *SettingServiceSuite) Test_UpdateSettings_PreservesFXAccounts() {
	ctx := s.ctx()

	// 1. Initial update: set DecimalStyle: "dot", Onboarding: true
	onboarding := true
	updateResp, err := s.svc.UpdateSettings(ctx, s.workspaceID, model.UpdateFinSettingRequest{
		Onboarding:   &onboarding,
		DecimalStyle: "dot",
	})
	s.Require().NoError(err)
	s.True(updateResp.Onboarding)
	s.Equal("dot", updateResp.DecimalStyle)
	s.Equal("IDR", updateResp.BaseCurrency)

	// 2. Create valid accounts and update FX accounts
	gainAcc, err := s.client.Account.Create().
		SetCode("7211").
		SetName("Realized FX Gain").
		SetType("REVENUE").
		SetNormalBalance("CREDIT").
		SetIsActive(true).
		SetIsPostable(true).
		Save(ctx)
	s.Require().NoError(err)

	lossAcc, err := s.client.Account.Create().
		SetCode("7311").
		SetName("Realized FX Loss").
		SetType("EXPENSE").
		SetNormalBalance("DEBIT").
		SetIsActive(true).
		SetIsPostable(true).
		Save(ctx)
	s.Require().NoError(err)

	fxResp, err := s.svc.UpdateFXAccounts(ctx, s.workspaceID, model.UpdateFXAccountsRequest{
		FXGainAccountID: &gainAcc.ID,
		FXLossAccountID: &lossAcc.ID,
	})
	s.Require().NoError(err)
	s.Equal(&gainAcc.ID, fxResp.FXGainAccountID)
	s.Equal(&lossAcc.ID, fxResp.FXLossAccountID)

	// Verify settings still preserve onboarding == true and decimalStyle == "dot"
	getResp, err := s.svc.GetSettings(ctx, s.workspaceID)
	s.Require().NoError(err)
	s.True(getResp.Onboarding)
	s.Equal("dot", getResp.DecimalStyle)
	s.Equal(&gainAcc.ID, getResp.FXGainAccountID)
	s.Equal(&lossAcc.ID, getResp.FXLossAccountID)

	// 3. Update DecimalStyle to "comma" via UpdateSettings (leaving Onboarding nil to preserve)
	updateResp2, err := s.svc.UpdateSettings(ctx, s.workspaceID, model.UpdateFinSettingRequest{
		DecimalStyle: "comma",
	})
	s.Require().NoError(err)
	s.True(updateResp2.Onboarding)                    // preserved!
	s.Equal("comma", updateResp2.DecimalStyle)        // updated!
	s.Equal(&gainAcc.ID, updateResp2.FXGainAccountID) // preserved!
	s.Equal(&lossAcc.ID, updateResp2.FXLossAccountID) // preserved!
}

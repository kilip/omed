package service_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/period"
	"github.com/kilip/omed/api/internal/base"
	corerepo "github.com/kilip/omed/api/internal/core/repository"
	coresvc "github.com/kilip/omed/api/internal/core/service"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/internal/platform/s3x"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type CashFlowServiceSuite struct {
	suite.Suite
	client          *ent.Client
	storage         s3x.Storage
	fileSvc         coresvc.FileService
	svc             service.CashFlowService
	workspaceID     int64
	userID          int64
	assetAccID      int64
	expenseAccID    int64
	revenueAccID    int64
	equityAccID     int64
	inactiveAccID   int64
	nonPostAccID    int64
	usdAssetAccID   int64
	openPeriodID    int64
	closedPeriodID  int64
}

func (s *CashFlowServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:cashflow_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	var err error
	s.storage, err = s3x.New(s3x.Config{Driver: "memory", BucketPrefix: "test-ws-"})
	s.Require().NoError(err)
	fileRepo := corerepo.NewFileRepository(s.client)
	s.fileSvc = coresvc.NewFileService(fileRepo, s.storage)

	cfRepo := repository.NewCashFlowRepository(s.client)
	periodRepo := repository.NewPeriodRepository(s.client)
	accountRepo := repository.NewAccountRepository(s.client)
	rateRepo := repository.NewExchangeRateRepository(s.client)
	currencyRepo := repository.NewCurrencyRepository(s.client)

	s.svc = service.NewCashFlowService(
		cfRepo,
		periodRepo,
		accountRepo,
		rateRepo,
		currencyRepo,
		s.fileSvc,
	)
}

func (s *CashFlowServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *CashFlowServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.File.Delete().Exec(ctx)
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.Period.Delete().Exec(ctx)
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Bob").
		SetEmail("bob@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Bob",
	})
	w, err := s.client.Workspace.Create().
		SetName("CashFlow WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID

	wsCtx := base.WithWorkspaceID(userCtx, w.ID)

	// Create test accounts
	a1, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("1001").
		SetName("Cash").
		SetType(account.Type(domain.AccountTypeAsset)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceDebit)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(wsCtx)
	s.Require().NoError(err)
	s.assetAccID = a1.ID

	a2, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("5001").
		SetName("Office Supplies").
		SetType(account.Type(domain.AccountTypeExpense)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceDebit)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(wsCtx)
	s.Require().NoError(err)
	s.expenseAccID = a2.ID

	a3, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("4001").
		SetName("Consulting Revenue").
		SetType(account.Type(domain.AccountTypeRevenue)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceCredit)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(wsCtx)
	s.Require().NoError(err)
	s.revenueAccID = a3.ID

	a4, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("3001").
		SetName("Equity").
		SetType(account.Type(domain.AccountTypeEquity)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceCredit)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(wsCtx)
	s.Require().NoError(err)
	s.equityAccID = a4.ID

	a5, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("1002").
		SetName("Inactive Cash").
		SetType(account.Type(domain.AccountTypeAsset)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceDebit)).
		SetIsActive(false).
		SetIsPostable(true).
		Save(wsCtx)
	s.Require().NoError(err)
	s.inactiveAccID = a5.ID

	a6, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("5000").
		SetName("NonPostable Expense").
		SetType(account.Type(domain.AccountTypeExpense)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceDebit)).
		SetIsActive(true).
		SetIsPostable(false).
		Save(wsCtx)
	s.Require().NoError(err)
	s.nonPostAccID = a6.ID

	usd := "USD"
	a7, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("1003").
		SetName("USD Cash").
		SetType(account.Type(domain.AccountTypeAsset)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceDebit)).
		SetIsActive(true).
		SetIsPostable(true).
		SetCurrency(usd).
		Save(wsCtx)
	s.Require().NoError(err)
	s.usdAssetAccID = a7.ID

	// Create periods
	p1, err := s.client.Period.Create().
		SetWorkspaceID(w.ID).
		SetName("Jan 2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)).
		SetFiscalYear(2026).
		SetStatus(period.Status(domain.PeriodStatusOpen)).
		Save(wsCtx)
	s.Require().NoError(err)
	s.openPeriodID = p1.ID

	p2, err := s.client.Period.Create().
		SetWorkspaceID(w.ID).
		SetName("Feb 2026").
		SetStartDate(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)).
		SetFiscalYear(2026).
		SetStatus(period.Status(domain.PeriodStatusClosed)).
		Save(wsCtx)
	s.Require().NoError(err)
	s.closedPeriodID = p2.ID
}

func (s *CashFlowServiceSuite) authCtx() context.Context {
	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Bob",
	})
	return base.WithWorkspaceID(userCtx, s.workspaceID)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_ValidWithAttachments_Success() {
	ctx := s.authCtx()

	// Upload a file first
	content := []byte("%PDF-1.4 test receipt content")
	up, err := s.fileSvc.Upload(ctx, coremodel.UploadFileRequest{
		WorkspaceID: s.workspaceID,
		Name:        "receipt.pdf",
		MimeType:    "application/pdf",
		Size:        int64(len(content)),
		EntityType:  "draft",
		EntityID:    0,
		Reader:      bytes.NewReader(content),
	})
	s.Require().NoError(err)

	req := model.CreateExpenseRequest{
		PaymentAccountID:  s.assetAccID,
		ExpenseAccountID:  s.expenseAccID,
		Amount:            decimal.NewFromInt(150000),
		TransactionDate:   model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Description:       "Office stationery",
		Reference:         "REF-EXP-001",
		AttachmentFileIDs: []int64{up.ID},
	}

	item, err := s.svc.CreateExpense(ctx, req)
	s.Require().NoError(err)
	s.Equal("EXPENSE", item.Type)
	s.Equal("POSTED", item.EntryStatus)
	s.True(item.Amount.Equal(decimal.NewFromInt(150000)))
	s.Equal(s.assetAccID, item.PaymentAccount.ID)
	s.Equal(s.expenseAccID, item.CategoryAccount.ID)
	s.Require().Len(item.Attachments, 1)
	s.Equal(up.ID, item.Attachments[0].ID)
	s.Equal(domain.TransactionEntityType, item.Attachments[0].EntityType)
	s.Equal(item.ID, item.Attachments[0].EntityID)

	// Verify it appears in SearchExpenses
	searchRes, _, err := s.svc.SearchExpenses(ctx, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Require().Len(searchRes, 1)
	s.Equal(item.ID, searchRes[0].ID)
	s.Require().Len(searchRes[0].Attachments, 1)
}

func (s *CashFlowServiceSuite) Test_CreateIncome_Valid_Success() {
	ctx := s.authCtx()

	req := model.CreateIncomeRequest{
		PaymentAccountID: s.assetAccID,
		RevenueAccountID: s.revenueAccID,
		Amount:           decimal.NewFromInt(500000),
		TransactionDate:  model.Date(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)),
		Description:      "Project down payment",
		Reference:        "REF-INC-001",
	}

	item, err := s.svc.CreateIncome(ctx, req)
	s.Require().NoError(err)
	s.Equal("INCOME", item.Type)
	s.Equal("POSTED", item.EntryStatus)
	s.True(item.Amount.Equal(decimal.NewFromInt(500000)))
	s.Equal(s.assetAccID, item.PaymentAccount.ID)
	s.Equal(s.revenueAccID, item.CategoryAccount.ID)

	// Verify it appears in SearchIncomes
	searchRes, _, err := s.svc.SearchIncomes(ctx, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Require().Len(searchRes, 1)
	s.Equal(item.ID, searchRes[0].ID)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_InvalidAccountTypes() {
	ctx := s.authCtx()

	// Payment account is not ASSET (use Equity)
	_, err := s.svc.CreateExpense(ctx, model.CreateExpenseRequest{
		PaymentAccountID: s.equityAccID,
		ExpenseAccountID: s.expenseAccID,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrAccountInvalidType)

	// Category account is not EXPENSE (use Revenue)
	_, err = s.svc.CreateExpense(ctx, model.CreateExpenseRequest{
		PaymentAccountID: s.assetAccID,
		ExpenseAccountID: s.revenueAccID,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrAccountInvalidType)
}

func (s *CashFlowServiceSuite) Test_CreateIncome_InvalidAccountTypes() {
	ctx := s.authCtx()

	// Category account is not REVENUE (use Expense)
	_, err := s.svc.CreateIncome(ctx, model.CreateIncomeRequest{
		PaymentAccountID: s.assetAccID,
		RevenueAccountID: s.expenseAccID,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrAccountInvalidType)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_InactiveOrNonPostableAccount() {
	ctx := s.authCtx()

	// Inactive payment account
	_, err := s.svc.CreateExpense(ctx, model.CreateExpenseRequest{
		PaymentAccountID: s.inactiveAccID,
		ExpenseAccountID: s.expenseAccID,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrPostingAccountInactive)

	// Non-postable category account
	_, err = s.svc.CreateExpense(ctx, model.CreateExpenseRequest{
		PaymentAccountID: s.assetAccID,
		ExpenseAccountID: s.nonPostAccID,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrPostingAccountNotPostable)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_PeriodNotFoundOrClosed() {
	ctx := s.authCtx()

	// Date outside any open period (e.g. March 2026)
	_, err := s.svc.CreateExpense(ctx, model.CreateExpenseRequest{
		PaymentAccountID: s.assetAccID,
		ExpenseAccountID: s.expenseAccID,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrPeriodNotFound)

	// Date in closed period (Feb 2026)
	_, err = s.svc.CreateExpense(ctx, model.CreateExpenseRequest{
		PaymentAccountID: s.assetAccID,
		ExpenseAccountID: s.expenseAccID,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrPeriodClosed)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_InvalidAmount() {
	ctx := s.authCtx()

	_, err := s.svc.CreateExpense(ctx, model.CreateExpenseRequest{
		PaymentAccountID: s.assetAccID,
		ExpenseAccountID: s.expenseAccID,
		Amount:           decimal.Zero,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrPostingInvalidAmount)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_CurrencyMismatch() {
	ctx := s.authCtx()

	// Account is USD, transaction currency is default (IDR)
	_, err := s.svc.CreateExpense(ctx, model.CreateExpenseRequest{
		PaymentAccountID: s.usdAssetAccID,
		ExpenseAccountID: s.expenseAccID,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.ErrorIs(err, domain.ErrPostingCurrencyMismatch)
}

func TestCashFlowServiceSuite(t *testing.T) {
	suite.Run(t, new(CashFlowServiceSuite))
}

package service_test

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/period"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	corerepo "github.com/kilip/omed/api/internal/core/repository"
	coresvc "github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/seed"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/internal/platform/s3x"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
	_ "modernc.org/sqlite"
)

func createTestDB(dsn string) *ent.Client {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("Can't create test db client: %v", err)
	}
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to create schema resources: %v", err)
	}
	core.InitEntCli(client)
	return client
}

type CashFlowServiceSuite struct {
	suite.Suite
	client         *ent.Client
	storage        s3x.Storage
	fileSvc        coresvc.FileService
	svc            service.CashFlowService
	workspaceID    int64
	userID         int64
	assetAccID     int64
	expenseAccID   int64
	revenueAccID   int64
	revenueAcc2ID  int64
	equityAccID    int64
	inactiveAccID  int64
	nonPostAccID   int64
	usdAssetAccID  int64
	openPeriodID   int64
	closedPeriodID int64
}

func (s *CashFlowServiceSuite) SetupSuite() {
	s.client = createTestDB("file:cashflow_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	_ = seed.SeedCurrencies(context.Background(), s.client)
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

	a3b, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("4002").
		SetName("Development Revenue").
		SetType(account.Type(domain.AccountTypeRevenue)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceCredit)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(wsCtx)
	s.Require().NoError(err)
	s.revenueAcc2ID = a3b.ID

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

	req := model.SaveCashFlowRequest{
		PaymentAccountID:  s.assetAccID,
		TransactionDate:   model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Description:       "Office stationery",
		Reference:         "REF-EXP-001",
		Items: []model.CashFlowItemRequest{
			{
				AccountID:   s.expenseAccID,
				Description: "Office stationery",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(150000),
			},
		},
		AttachmentFileIDs: []int64{up.ID},
	}

	item, err := s.svc.CreateExpense(ctx, req)
	s.Require().NoError(err)
	s.Equal("EXPENSE", item.Type)
	s.Equal("POSTED", item.EntryStatus)
	s.True(item.Amount.Equal(decimal.NewFromInt(150000)))
	s.Equal(s.assetAccID, item.PaymentAccount.ID)
	s.Require().Len(item.Items, 1)
	s.Equal(s.expenseAccID, item.Items[0].Account.ID)
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

func (s *CashFlowServiceSuite) Test_CreateIncome_TwoItems_PostsBalancedDocument() {
	ctx := s.authCtx()

	req := model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)),
		Reference:        "inv-123456",
		Description:      "Consulting and scaffolding",
		Status:           domain.EntryStatusPosted,
		Items: []model.CashFlowItemRequest{
			{
				AccountID:   s.revenueAccID,
				Description: "Konsultasi Improvement",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(1000000),
			},
			{
				AccountID:   s.revenueAcc2ID,
				Description: "Scaffold base project structure and ci/cd",
				Quantity:    decimal.NewFromInt(6),
				UnitPrice:   decimal.NewFromInt(500000),
			},
		},
	}

	item, err := s.svc.CreateIncome(ctx, req)
	s.Require().NoError(err)
	s.Equal("INCOME", item.Type)
	s.Equal("POSTED", item.EntryStatus)
	s.Equal("inv-123456", item.Reference)
	s.True(item.Amount.Equal(decimal.NewFromInt(4000000)))
	s.True(item.BaseAmount.Equal(decimal.NewFromInt(4000000)))
	s.Require().Len(item.Items, 2)
	s.True(item.Items[0].Amount.Equal(decimal.NewFromInt(1000000)))
	s.True(item.Items[1].Amount.Equal(decimal.NewFromInt(3000000)))
}

func (s *CashFlowServiceSuite) Test_CreateExpense_SameAccountTwice_Total125000() {
	ctx := s.authCtx()

	req := model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Description:      "Warung belanja",
		Items: []model.CashFlowItemRequest{
			{
				AccountID:   s.expenseAccID,
				Description: "telur",
				Quantity:    decimal.NewFromInt(2),
				UnitPrice:   decimal.NewFromInt(55000),
			},
			{
				AccountID:   s.expenseAccID,
				Description: "indomie",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromInt(3000),
			},
		},
	}

	item, err := s.svc.CreateExpense(ctx, req)
	s.Require().NoError(err)
	s.Equal("EXPENSE", item.Type)
	s.True(item.Amount.Equal(decimal.NewFromInt(125000)))
	s.Require().Len(item.Items, 2)
	s.True(item.Items[0].Amount.Equal(decimal.NewFromInt(110000)))
	s.True(item.Items[1].Amount.Equal(decimal.NewFromInt(15000)))
	s.Equal("telur", item.Items[0].Description)
	s.Equal("indomie", item.Items[1].Description)
}

func (s *CashFlowServiceSuite) Test_CreateIncome_Draft_EntryIsDraft() {
	ctx := s.authCtx()

	req := model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)),
		Description:      "Draft proposal",
		Status:           domain.EntryStatusDraft,
		Items: []model.CashFlowItemRequest{
			{
				AccountID:   s.revenueAccID,
				Description: "Down payment draft",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(500000),
			},
		},
	}

	item, err := s.svc.CreateIncome(ctx, req)
	s.Require().NoError(err)
	s.Equal("INCOME", item.Type)
	s.Equal("DRAFT", item.EntryStatus)
}

func (s *CashFlowServiceSuite) Test_Create_InvalidItems_ReturnErrors() {
	ctx := s.authCtx()

	// 1. empty items
	_, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items:            []model.CashFlowItemRequest{},
	})
	s.ErrorIs(err, domain.ErrPostingInvalidAmount)

	// 2. 101 items
	tooManyItems := make([]model.CashFlowItemRequest, 101)
	for i := range tooManyItems {
		tooManyItems[i] = model.CashFlowItemRequest{
			AccountID: s.expenseAccID,
			Quantity:  decimal.NewFromInt(1),
			UnitPrice: decimal.NewFromInt(100),
		}
	}
	_, err = s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items:            tooManyItems,
	})
	s.ErrorIs(err, domain.ErrPostingInvalidAmount)

	// 3. quantity 0 -> ErrPostingInvalidQuantity
	_, err = s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.Zero,
				UnitPrice: decimal.NewFromInt(1000),
			},
		},
	})
	s.ErrorIs(err, domain.ErrPostingInvalidQuantity)

	// 4. unit price -1 -> ErrPostingInvalidUnitPrice
	_, err = s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(-1),
			},
		},
	})
	s.ErrorIs(err, domain.ErrPostingInvalidUnitPrice)

	// 5. line amount 0 -> ErrPostingInvalidAmount
	_, err = s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.Zero,
			},
		},
	})
	s.ErrorIs(err, domain.ErrPostingInvalidAmount)
}

func (s *CashFlowServiceSuite) Test_Create_ItemAccountWrongType_ReturnsErrAccountInvalidType() {
	ctx := s.authCtx()

	// Expense account on an income document
	_, err := s.svc.CreateIncome(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(100000),
			},
		},
	})
	s.ErrorIs(err, domain.ErrAccountInvalidType)

	// Revenue account on an expense document
	_, err = s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.revenueAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(100000),
			},
		},
	})
	s.ErrorIs(err, domain.ErrAccountInvalidType)

	// Payment account is not ASSET (use Equity)
	_, err = s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.equityAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(100000),
			},
		},
	})
	s.ErrorIs(err, domain.ErrAccountInvalidType)
}

func (s *CashFlowServiceSuite) Test_Create_FractionalQuantityRoundsToCurrencyDecimals() {
	ctx := s.authCtx()

	// IDR (0 decimals), 2.5 × 3333 = 8332.5 gives line amount 8333
	item, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Currency:         "IDR",
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.RequireFromString("2.5"),
				UnitPrice: decimal.NewFromInt(3333),
			},
		},
	})
	s.Require().NoError(err)
	s.True(item.Amount.Equal(decimal.NewFromInt(8333)))
	s.Require().Len(item.Items, 1)
	s.True(item.Items[0].Amount.Equal(decimal.NewFromInt(8333)))
}

func (s *CashFlowServiceSuite) Test_Create_ForeignCurrency_BaseAmountsBalance() {
	ctx := s.authCtx()

	rate := decimal.RequireFromString("16000.5")
	req := model.SaveCashFlowRequest{
		PaymentAccountID: s.usdAssetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Currency:         "USD",
		ExchangeRate:     &rate,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.RequireFromString("1.5"),
				UnitPrice: decimal.RequireFromString("10.33"),
			},
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.RequireFromString("2.25"),
				UnitPrice: decimal.RequireFromString("5.50"),
			},
		},
	}
	item, err := s.svc.CreateExpense(ctx, req)
	s.Require().NoError(err)
	var expectedBaseSum decimal.Decimal
	for _, it := range item.Items {
		expectedBaseSum = expectedBaseSum.Add(it.BaseAmount)
	}
	s.True(item.BaseAmount.Equal(expectedBaseSum))
}

func (s *CashFlowServiceSuite) Test_Update_Draft_ReplacesItemsAndPostsWhenRequested() {
	ctx := s.authCtx()

	created, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Status:           domain.EntryStatusDraft,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(50000),
			},
		},
	})
	s.Require().NoError(err)
	s.Equal("DRAFT", created.EntryStatus)

	updated, err := s.svc.UpdateExpense(ctx, created.ID, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)),
		Status:           domain.EntryStatusPosted,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(2),
				UnitPrice: decimal.NewFromInt(60000),
			},
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(40000),
			},
		},
	})
	s.Require().NoError(err)
	s.Equal("POSTED", updated.EntryStatus)
	s.True(updated.Amount.Equal(decimal.NewFromInt(160000)))
	s.Require().Len(updated.Items, 2)
	s.True(updated.Items[0].Amount.Equal(decimal.NewFromInt(120000)))
	s.True(updated.Items[1].Amount.Equal(decimal.NewFromInt(40000)))
}

func (s *CashFlowServiceSuite) Test_Update_PostedDocument_ReturnsErrEntryNotDraft() {
	ctx := s.authCtx()

	posted, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Status:           domain.EntryStatusPosted,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(50000),
			},
		},
	})
	s.Require().NoError(err)
	s.Equal("POSTED", posted.EntryStatus)

	_, err = s.svc.UpdateExpense(ctx, posted.ID, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(70000),
			},
		},
	})
	s.ErrorIs(err, domain.ErrEntryNotDraft)
}

func (s *CashFlowServiceSuite) Test_Update_LinksAdditionalAttachments() {
	ctx := s.authCtx()

	content1 := []byte("receipt 1")
	up1, err := s.fileSvc.Upload(ctx, coremodel.UploadFileRequest{
		WorkspaceID: s.workspaceID,
		Name:        "receipt1.pdf",
		MimeType:    "application/pdf",
		Size:        int64(len(content1)),
		EntityType:  "draft",
		EntityID:    0,
		Reader:      bytes.NewReader(content1),
	})
	s.Require().NoError(err)

	created, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID:  s.assetAccID,
		TransactionDate:   model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Status:            domain.EntryStatusDraft,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(50000),
			},
		},
		AttachmentFileIDs: []int64{up1.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(created.Attachments, 1)

	content2 := []byte("receipt 2")
	up2, err := s.fileSvc.Upload(ctx, coremodel.UploadFileRequest{
		WorkspaceID: s.workspaceID,
		Name:        "receipt2.pdf",
		MimeType:    "application/pdf",
		Size:        int64(len(content2)),
		EntityType:  "draft",
		EntityID:    0,
		Reader:      bytes.NewReader(content2),
	})
	s.Require().NoError(err)

	updated, err := s.svc.UpdateExpense(ctx, created.ID, model.SaveCashFlowRequest{
		PaymentAccountID:  s.assetAccID,
		TransactionDate:   model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Status:            domain.EntryStatusDraft,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(50000),
			},
		},
		AttachmentFileIDs: []int64{up2.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(updated.Attachments, 2)
}

func (s *CashFlowServiceSuite) Test_Update_PeriodClosedAfterSave_StaysDraft() {
	ctx := s.authCtx()

	created, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Status:           domain.EntryStatusDraft,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(50000),
			},
		},
	})
	s.Require().NoError(err)
	s.Equal("DRAFT", created.EntryStatus)

	// Attempt to post into closed period (Feb 2026)
	_, err = s.svc.UpdateExpense(ctx, created.ID, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)),
		Status:           domain.EntryStatusPosted,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(50000),
			},
		},
	})
	s.ErrorIs(err, domain.ErrPeriodClosed)

	got, err := s.svc.GetExpense(ctx, created.ID)
	s.Require().NoError(err)
	s.Equal("DRAFT", got.EntryStatus)
}

func (s *CashFlowServiceSuite) Test_Update_DateMovesToOtherPeriod_RecomputesPeriod() {
	ctx := s.authCtx()

	pMarch, err := s.client.Period.Create().
		SetWorkspaceID(s.workspaceID).
		SetName("Mar 2026").
		SetStartDate(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)).
		SetFiscalYear(2026).
		SetStatus(period.Status(domain.PeriodStatusOpen)).
		Save(ctx)
	s.Require().NoError(err)
	_ = pMarch

	created, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Status:           domain.EntryStatusDraft,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(50000),
			},
		},
	})
	s.Require().NoError(err)

	updated, err := s.svc.UpdateExpense(ctx, created.ID, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)),
		Status:           domain.EntryStatusPosted,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.expenseAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(50000),
			},
		},
	})
	s.Require().NoError(err)
	s.Equal("POSTED", updated.EntryStatus)
	s.Equal(time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), updated.TransactionDate)
}

func (s *CashFlowServiceSuite) Test_GetExpense_OnIncomeDocument_ReturnsErrTransactionNotFound() {
	ctx := s.authCtx()

	inc, err := s.svc.CreateIncome(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{
				AccountID: s.revenueAccID,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(100000),
			},
		},
	})
	s.Require().NoError(err)

	_, err = s.svc.GetExpense(ctx, inc.ID)
	s.ErrorIs(err, domain.ErrTransactionNotFound)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_InactiveOrNonPostableAccount() {
	ctx := s.authCtx()

	// Inactive payment account
	_, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.inactiveAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{AccountID: s.expenseAccID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.ErrorIs(err, domain.ErrPostingAccountInactive)

	// Non-postable category account
	_, err = s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{AccountID: s.nonPostAccID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.ErrorIs(err, domain.ErrPostingAccountNotPostable)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_PeriodNotFoundOrClosed() {
	ctx := s.authCtx()

	// Date outside any open period (April 2026)
	_, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{AccountID: s.expenseAccID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.ErrorIs(err, domain.ErrPeriodNotFound)

	// Date in closed period (Feb 2026)
	_, err = s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.assetAccID,
		TransactionDate:  model.Date(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{AccountID: s.expenseAccID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.ErrorIs(err, domain.ErrPeriodClosed)
}

func (s *CashFlowServiceSuite) Test_CreateExpense_CurrencyMismatch() {
	ctx := s.authCtx()

	// Account is USD, transaction currency is default (IDR)
	_, err := s.svc.CreateExpense(ctx, model.SaveCashFlowRequest{
		PaymentAccountID: s.usdAssetAccID,
		TransactionDate:  model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{AccountID: s.expenseAccID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.ErrorIs(err, domain.ErrPostingCurrencyMismatch)
}

func TestCashFlowServiceSuite(t *testing.T) {
	suite.Run(t, new(CashFlowServiceSuite))
}

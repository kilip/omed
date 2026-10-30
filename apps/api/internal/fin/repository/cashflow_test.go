package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type CashFlowRepoTestSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.CashFlowRepository
	workspaceID int64
	userID      int64
	user        *ent.User
	period      *ent.Period
	cashAcc     *ent.Account
	bankAcc     *ent.Account
	expenseAcc  *ent.Account
	expenseAcc2 *ent.Account
	revenueAcc  *ent.Account
	ctx         context.Context
}

func (s *CashFlowRepoTestSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:cashflow_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewCashFlowRepository(s.client)
}

func (s *CashFlowRepoTestSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *CashFlowRepoTestSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.Period.Delete().Exec(ctx)
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Alice").
		SetEmail("alice@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.user = u
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Alice",
	})
	w, err := s.client.Workspace.Create().
		SetName("Finance WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID

	s.ctx = base.WithWorkspaceID(userCtx, w.ID)

	cashAcc, err := s.client.Account.Create().
		SetCode("1101").
		SetName("Cash").
		SetType(domain.AccountTypeAsset).
		SetNormalBalance(domain.PostingDirectionDebit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.cashAcc = cashAcc

	bankAcc, err := s.client.Account.Create().
		SetCode("1102").
		SetName("Bank BCA").
		SetType(domain.AccountTypeAsset).
		SetNormalBalance(domain.PostingDirectionDebit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.bankAcc = bankAcc

	expAcc, err := s.client.Account.Create().
		SetCode("5101").
		SetName("Office Supplies").
		SetType(domain.AccountTypeExpense).
		SetNormalBalance(domain.PostingDirectionDebit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.expenseAcc = expAcc

	expAcc2, err := s.client.Account.Create().
		SetCode("5102").
		SetName("Meals & Entertainment").
		SetType(domain.AccountTypeExpense).
		SetNormalBalance(domain.PostingDirectionDebit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.expenseAcc2 = expAcc2

	revAcc, err := s.client.Account.Create().
		SetCode("4101").
		SetName("Sales Revenue").
		SetType(domain.AccountTypeRevenue).
		SetNormalBalance(domain.PostingDirectionCredit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.revenueAcc = revAcc

	p, err := s.client.Period.Create().
		SetName("FY2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(s.ctx)
	s.Require().NoError(err)
	s.period = p
}

func (s *CashFlowRepoTestSuite) TestCreatePostedExpenseAndSearch() {
	item, err := s.repo.CreatePosted(s.ctx, repository.CreateCashFlowParams{
		FlowType:          domain.CashFlowTypeExpense,
		PeriodID:          s.period.ID,
		PaymentAccountID:  s.cashAcc.ID,
		CategoryAccountID: s.expenseAcc.ID,
		Amount:            decimal.NewFromInt(75000),
		BaseAmount:        decimal.NewFromInt(75000),
		TransactionDate:   time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
		Description:       "Belanja di Indomaret",
		Reference:         "STRUK-001",
		Currency:          "IDR",
		ExchangeRate:      decimal.NewFromInt(1),
		PostedBy:          s.user.ID,
	})
	s.Require().NoError(err)
	s.Equal(domain.CashFlowTypeExpense, item.Type)
	s.Equal(s.cashAcc.ID, item.PaymentAccount.ID)
	s.Equal("1101", item.PaymentAccount.Code)
	s.Equal("Cash", item.PaymentAccount.Name)
	s.Equal(domain.AccountTypeAsset, item.PaymentAccount.Type)
	s.Equal(s.expenseAcc.ID, item.CategoryAccount.ID)
	s.Equal("5101", item.CategoryAccount.Code)
	s.Equal("Office Supplies", item.CategoryAccount.Name)
	s.Equal(domain.AccountTypeExpense, item.CategoryAccount.Type)
	s.True(decimal.NewFromInt(75000).Equal(item.Amount))
	s.True(decimal.NewFromInt(75000).Equal(item.BaseAmount))
	s.Equal("Belanja di Indomaret", item.Description)
	s.Equal("STRUK-001", item.Reference)
	s.Equal("IDR", item.Currency)
	s.NotEmpty(item.Number)
	s.NotEmpty(item.EntryNumber)
	s.Equal(domain.EntryStatusPosted, item.EntryStatus)
	s.Equal(s.workspaceID, item.WorkspaceID)

	items, cursor, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Empty(cursor)
	s.Len(items, 1)
	s.Equal(item.ID, items[0].ID)
	s.Equal(domain.CashFlowTypeExpense, items[0].Type)
	s.Equal(s.cashAcc.ID, items[0].PaymentAccount.ID)
	s.Equal(s.expenseAcc.ID, items[0].CategoryAccount.ID)

	incItems, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeIncome, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Empty(incItems)
}

func (s *CashFlowRepoTestSuite) TestCreatePostedIncomeAndSearch() {
	item, err := s.repo.CreatePosted(s.ctx, repository.CreateCashFlowParams{
		FlowType:          domain.CashFlowTypeIncome,
		PeriodID:          s.period.ID,
		PaymentAccountID:  s.bankAcc.ID,
		CategoryAccountID: s.revenueAcc.ID,
		Amount:            decimal.NewFromInt(1500000),
		BaseAmount:        decimal.NewFromInt(1500000),
		TransactionDate:   time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC),
		Description:       "Pembayaran Proyek",
		Reference:         "INV-001",
		Currency:          "IDR",
		ExchangeRate:      decimal.NewFromInt(1),
		PostedBy:          s.user.ID,
	})
	s.Require().NoError(err)
	s.Equal(domain.CashFlowTypeIncome, item.Type)
	s.Equal(s.bankAcc.ID, item.PaymentAccount.ID)
	s.Equal(s.revenueAcc.ID, item.CategoryAccount.ID)
	s.True(decimal.NewFromInt(1500000).Equal(item.Amount))
	s.Equal(domain.EntryStatusPosted, item.EntryStatus)

	// Search income finds this item
	incItems, cursor, err := s.repo.Search(s.ctx, domain.CashFlowTypeIncome, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Empty(cursor)
	s.Len(incItems, 1)
	s.Equal(item.ID, incItems[0].ID)
	s.Equal(domain.CashFlowTypeIncome, incItems[0].Type)
	s.Equal(s.bankAcc.ID, incItems[0].PaymentAccount.ID)
	s.Equal(s.revenueAcc.ID, incItems[0].CategoryAccount.ID)

	// Search expense does not find this income
	expItems, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Empty(expItems)
}

func (s *CashFlowRepoTestSuite) TestSearchFilters() {
	// Expense 1: Indomaret, 2026-10-01, cashAcc, expenseAcc, ref "STRUK-001"
	exp1, err := s.repo.CreatePosted(s.ctx, repository.CreateCashFlowParams{
		FlowType:          domain.CashFlowTypeExpense,
		PeriodID:          s.period.ID,
		PaymentAccountID:  s.cashAcc.ID,
		CategoryAccountID: s.expenseAcc.ID,
		Amount:            decimal.NewFromInt(50000),
		BaseAmount:        decimal.NewFromInt(50000),
		TransactionDate:   time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Description:       "Belanja Indomaret",
		Reference:         "STRUK-001",
		Currency:          "IDR",
		ExchangeRate:      decimal.NewFromInt(1),
		PostedBy:          s.user.ID,
	})
	s.Require().NoError(err)

	// Expense 2: Alfamart, 2026-10-15, bankAcc, expenseAcc2, ref "STRUK-002"
	exp2, err := s.repo.CreatePosted(s.ctx, repository.CreateCashFlowParams{
		FlowType:          domain.CashFlowTypeExpense,
		PeriodID:          s.period.ID,
		PaymentAccountID:  s.bankAcc.ID,
		CategoryAccountID: s.expenseAcc2.ID,
		Amount:            decimal.NewFromInt(100000),
		BaseAmount:        decimal.NewFromInt(100000),
		TransactionDate:   time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		Description:       "Makan Siang Alfamart",
		Reference:         "STRUK-002",
		Currency:          "IDR",
		ExchangeRate:      decimal.NewFromInt(1),
		PostedBy:          s.user.ID,
	})
	s.Require().NoError(err)

	// 1. Filter by Date range
	dateFrom := model.Date(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))
	dateTo := model.Date(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC))
	dateFiltered, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	})
	s.Require().NoError(err)
	s.Len(dateFiltered, 1)
	s.Equal(exp2.ID, dateFiltered[0].ID)

	// 2. Filter by PaymentAccountID
	cashFiltered, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		PaymentAccountID: &s.cashAcc.ID,
	})
	s.Require().NoError(err)
	s.Len(cashFiltered, 1)
	s.Equal(exp1.ID, cashFiltered[0].ID)

	bankFiltered, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		PaymentAccountID: &s.bankAcc.ID,
	})
	s.Require().NoError(err)
	s.Len(bankFiltered, 1)
	s.Equal(exp2.ID, bankFiltered[0].ID)

	// 3. Filter by CategoryAccountID
	catFiltered, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		CategoryAccountID: &s.expenseAcc.ID,
	})
	s.Require().NoError(err)
	s.Len(catFiltered, 1)
	s.Equal(exp1.ID, catFiltered[0].ID)

	// 4. Filter by Query (description / reference)
	q1 := "Indomaret"
	queryFiltered1, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		Query: &q1,
	})
	s.Require().NoError(err)
	s.Len(queryFiltered1, 1)
	s.Equal(exp1.ID, queryFiltered1[0].ID)

	q2 := "STRUK-002"
	queryFiltered2, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		Query: &q2,
	})
	s.Require().NoError(err)
	s.Len(queryFiltered2, 1)
	s.Equal(exp2.ID, queryFiltered2[0].ID)
}

func (s *CashFlowRepoTestSuite) TestSearchPagination() {
	for i := 1; i <= 5; i++ {
		_, err := s.repo.CreatePosted(s.ctx, repository.CreateCashFlowParams{
			FlowType:          domain.CashFlowTypeExpense,
			PeriodID:          s.period.ID,
			PaymentAccountID:  s.cashAcc.ID,
			CategoryAccountID: s.expenseAcc.ID,
			Amount:            decimal.NewFromInt(int64(i * 10000)),
			BaseAmount:        decimal.NewFromInt(int64(i * 10000)),
			TransactionDate:   time.Date(2026, 10, i, 0, 0, 0, 0, time.UTC),
			Description:       "Test Pagination Expense",
			Currency:          "IDR",
			ExchangeRate:      decimal.NewFromInt(1),
			PostedBy:          s.user.ID,
		})
		s.Require().NoError(err)
	}

	limit := 2
	page1, cursor1, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		Limit: &limit,
	})
	s.Require().NoError(err)
	s.Len(page1, 2)
	s.NotEmpty(cursor1)

	page2, cursor2, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		Limit:  &limit,
		Cursor: &cursor1,
	})
	s.Require().NoError(err)
	s.Len(page2, 2)
	s.NotEmpty(cursor2)
	s.NotEqual(page1[0].ID, page2[0].ID)
	s.NotEqual(page1[1].ID, page2[1].ID)

	page3, cursor3, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		Limit:  &limit,
		Cursor: &cursor2,
	})
	s.Require().NoError(err)
	s.Len(page3, 1)
	s.Empty(cursor3)
}

func TestCashFlowRepoTestSuite(t *testing.T) {
	suite.Run(t, new(CashFlowRepoTestSuite))
}

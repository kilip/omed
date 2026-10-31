package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/entry"
	"github.com/kilip/omed/api/ent/posting"
	"github.com/kilip/omed/api/ent/transaction"
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
	bankKaltim  *ent.Account
	expenseAcc  *ent.Account
	expenseAcc2 *ent.Account
	expense6710 *ent.Account
	revenueAcc  *ent.Account
	rev4130     *ent.Account
	rev4110     *ent.Account
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

	bankKaltim, err := s.client.Account.Create().
		SetCode("1120").
		SetName("Bank Kaltim").
		SetType(domain.AccountTypeAsset).
		SetNormalBalance(domain.PostingDirectionDebit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.bankKaltim = bankKaltim

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

	exp6710, err := s.client.Account.Create().
		SetCode("6710").
		SetName("Biaya Lainnya").
		SetType(domain.AccountTypeExpense).
		SetNormalBalance(domain.PostingDirectionDebit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.expense6710 = exp6710

	revAcc, err := s.client.Account.Create().
		SetCode("4101").
		SetName("Sales Revenue").
		SetType(domain.AccountTypeRevenue).
		SetNormalBalance(domain.PostingDirectionCredit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.revenueAcc = revAcc

	rev4130, err := s.client.Account.Create().
		SetCode("4130").
		SetName("Konsultasi Improvement").
		SetType(domain.AccountTypeRevenue).
		SetNormalBalance(domain.PostingDirectionCredit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.rev4130 = rev4130

	rev4110, err := s.client.Account.Create().
		SetCode("4110").
		SetName("Scaffold base project structure and ci/cd").
		SetType(domain.AccountTypeRevenue).
		SetNormalBalance(domain.PostingDirectionCredit).
		Save(s.ctx)
	s.Require().NoError(err)
	s.rev4110 = rev4110

	p, err := s.client.Period.Create().
		SetName("FY2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(s.ctx)
	s.Require().NoError(err)
	s.period = p
}

func (s *CashFlowRepoTestSuite) Test_Create_MultiItemPosted_PostingsAndPositions() {
	item, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeIncome,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.bankKaltim.ID,
		Items: []repository.CashFlowItemParams{
			{
				AccountID:   s.rev4130.ID,
				Description: "Konsultasi Improvement",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(1000000),
				Amount:      decimal.NewFromInt(1000000),
				BaseAmount:  decimal.NewFromInt(1000000),
			},
			{
				AccountID:   s.rev4110.ID,
				Description: "Scaffold base project structure and ci/cd",
				Quantity:    decimal.NewFromInt(6),
				UnitPrice:   decimal.NewFromInt(500000),
				Amount:      decimal.NewFromInt(3000000),
				BaseAmount:  decimal.NewFromInt(3000000),
			},
		},
		TotalAmount:     decimal.NewFromInt(4000000),
		TotalBaseAmount: decimal.NewFromInt(4000000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		Description:     "Consulting and scaffolding",
		Reference:       "inv-123456",
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)
	s.Equal(domain.CashFlowTypeIncome, item.Type)
	s.Equal(s.bankKaltim.ID, item.PaymentAccount.ID)
	s.Equal("Bank Kaltim", item.PaymentAccount.Name)
	s.True(decimal.NewFromInt(4000000).Equal(item.Amount))
	s.True(decimal.NewFromInt(4000000).Equal(item.BaseAmount))
	s.Equal(domain.EntryStatusPosted, item.EntryStatus)
	s.Require().Len(item.Items, 2)
	s.Equal(s.rev4130.ID, item.Items[0].Account.ID)
	s.True(decimal.NewFromInt(1).Equal(item.Items[0].Quantity))
	s.True(decimal.NewFromInt(1000000).Equal(item.Items[0].UnitPrice))
	s.True(decimal.NewFromInt(1000000).Equal(item.Items[0].Amount))
	s.Equal(s.rev4110.ID, item.Items[1].Account.ID)
	s.True(decimal.NewFromInt(6).Equal(item.Items[1].Quantity))
	s.True(decimal.NewFromInt(500000).Equal(item.Items[1].UnitPrice))
	s.True(decimal.NewFromInt(3000000).Equal(item.Items[1].Amount))

	// Verify entry and postings in DB directly
	e, err := s.client.Entry.Query().
		Where(entry.ID(item.EntryID)).
		WithPostings(func(pq *ent.PostingQuery) {
			pq.Order(ent.Asc(posting.FieldPosition))
		}).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Equal(entry.Status(domain.EntryStatusPosted), e.Status)
	s.NotNil(e.PostedAt)
	s.Require().NotNil(e.PostedBy)
	s.Equal(s.user.ID, *e.PostedBy)

	s.Require().Len(e.Edges.Postings, 3)
	// Position 0: payment DEBIT
	p0 := e.Edges.Postings[0]
	s.Equal(0, p0.Position)
	s.Equal(s.bankKaltim.ID, p0.AccountID)
	s.Equal(posting.DirectionDEBIT, p0.Direction)
	s.True(decimal.NewFromInt(4000000).Equal(p0.Amount))
	s.True(decimal.NewFromInt(4000000).Equal(p0.BaseAmount))
	s.Nil(p0.Quantity)
	s.Nil(p0.UnitPrice)

	// Position 1: rev4130 CREDIT
	p1 := e.Edges.Postings[1]
	s.Equal(1, p1.Position)
	s.Equal(s.rev4130.ID, p1.AccountID)
	s.Equal(posting.DirectionCREDIT, p1.Direction)
	s.True(decimal.NewFromInt(1000000).Equal(p1.Amount))
	s.True(decimal.NewFromInt(1000000).Equal(p1.BaseAmount))
	s.Require().NotNil(p1.Quantity)
	s.True(decimal.NewFromInt(1).Equal(*p1.Quantity))
	s.Require().NotNil(p1.UnitPrice)
	s.True(decimal.NewFromInt(1000000).Equal(*p1.UnitPrice))

	// Position 2: rev4110 CREDIT
	p2 := e.Edges.Postings[2]
	s.Equal(2, p2.Position)
	s.Equal(s.rev4110.ID, p2.AccountID)
	s.Equal(posting.DirectionCREDIT, p2.Direction)
	s.True(decimal.NewFromInt(3000000).Equal(p2.Amount))
	s.True(decimal.NewFromInt(3000000).Equal(p2.BaseAmount))
	s.Require().NotNil(p2.Quantity)
	s.True(decimal.NewFromInt(6).Equal(*p2.Quantity))
	s.Require().NotNil(p2.UnitPrice)
	s.True(decimal.NewFromInt(500000).Equal(*p2.UnitPrice))
}

func (s *CashFlowRepoTestSuite) Test_Create_Draft_HasNoPostedAt() {
	item, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeIncome,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.bankKaltim.ID,
		Items: []repository.CashFlowItemParams{
			{
				AccountID:   s.rev4130.ID,
				Description: "Konsultasi Improvement",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(1000000),
				Amount:      decimal.NewFromInt(1000000),
				BaseAmount:  decimal.NewFromInt(1000000),
			},
		},
		TotalAmount:     decimal.NewFromInt(1000000),
		TotalBaseAmount: decimal.NewFromInt(1000000),
		Status:          domain.EntryStatusDraft,
		TransactionDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		Description:     "Draft income",
		Reference:       "inv-draft",
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusDraft, item.EntryStatus)

	e, err := s.client.Entry.Get(s.ctx, item.EntryID)
	s.Require().NoError(err)
	s.Equal(entry.Status(domain.EntryStatusDraft), e.Status)
	s.Nil(e.PostedAt)
	s.Nil(e.PostedBy)
}

func (s *CashFlowRepoTestSuite) Test_Create_FXBaseAmountsBalance() {
	rate := decimal.NewFromFloat(16000.5)
	item1Base := decimal.NewFromFloat(165285.17)
	item2Base := decimal.NewFromFloat(80002.50)
	item3Base := decimal.NewFromFloat(40001.25)
	totalBase := item1Base.Add(item2Base).Add(item3Base)

	item1Amt := decimal.NewFromFloat(10.33)
	item2Amt := decimal.NewFromFloat(5.00)
	item3Amt := decimal.NewFromFloat(2.50)
	totalAmt := item1Amt.Add(item2Amt).Add(item3Amt)

	item, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeIncome,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.bankKaltim.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.rev4130.ID, Description: "Item 1", Quantity: decimal.NewFromInt(1), UnitPrice: item1Amt, Amount: item1Amt, BaseAmount: item1Base},
			{AccountID: s.rev4110.ID, Description: "Item 2", Quantity: decimal.NewFromInt(1), UnitPrice: item2Amt, Amount: item2Amt, BaseAmount: item2Base},
			{AccountID: s.revenueAcc.ID, Description: "Item 3", Quantity: decimal.NewFromInt(1), UnitPrice: item3Amt, Amount: item3Amt, BaseAmount: item3Base},
		},
		TotalAmount:     totalAmt,
		TotalBaseAmount: totalBase,
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		Currency:        "USD",
		ExchangeRate:    rate,
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	e, err := s.client.Entry.Query().
		Where(entry.ID(item.EntryID)).
		WithPostings(func(pq *ent.PostingQuery) {
			pq.Order(ent.Asc(posting.FieldPosition))
		}).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(e.Edges.Postings, 4)

	paymentBase := e.Edges.Postings[0].BaseAmount
	sumItemBase := decimal.Zero
	for _, p := range e.Edges.Postings[1:] {
		sumItemBase = sumItemBase.Add(p.BaseAmount)
	}
	s.True(paymentBase.Equal(totalBase))
	s.True(paymentBase.Equal(sumItemBase))
}

func (s *CashFlowRepoTestSuite) Test_Search_IncludesDraftAndFiltersByStatus() {
	// Draft doc
	draftDoc, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeIncome,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.bankKaltim.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.rev4130.ID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000), Amount: decimal.NewFromInt(100000), BaseAmount: decimal.NewFromInt(100000)},
		},
		TotalAmount:     decimal.NewFromInt(100000),
		TotalBaseAmount: decimal.NewFromInt(100000),
		Status:          domain.EntryStatusDraft,
		TransactionDate: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC),
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	// Posted doc
	postedDoc, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeIncome,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.bankKaltim.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.rev4130.ID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(200000), Amount: decimal.NewFromInt(200000), BaseAmount: decimal.NewFromInt(200000)},
		},
		TotalAmount:     decimal.NewFromInt(200000),
		TotalBaseAmount: decimal.NewFromInt(200000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC),
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	// Void entry transaction
	vt, err := s.client.Transaction.Create().
		SetNumber("TRX-VOID").
		SetTransactionDate(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)).
		SetCurrency("IDR").
		SetExchangeRate(decimal.NewFromInt(1)).
		SetSource(transaction.Source(domain.TransactionSourceManual)).
		Save(s.ctx)
	s.Require().NoError(err)
	ve, err := s.client.Entry.Create().
		SetTransactionID(vt.ID).
		SetPeriodID(s.period.ID).
		SetNumber("JRN-VOID").
		SetEntryDate(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)).
		SetStatus(entry.Status(domain.EntryStatusVoid)).
		Save(s.ctx)
	s.Require().NoError(err)
	_, err = s.client.Posting.Create().
		SetEntryID(ve.ID).
		SetAccountID(s.bankKaltim.ID).
		SetDirection(posting.DirectionDEBIT).
		SetAmount(decimal.NewFromInt(50000)).
		SetBaseAmount(decimal.NewFromInt(50000)).
		SetPosition(0).
		Save(s.ctx)
	s.Require().NoError(err)
	_, err = s.client.Posting.Create().
		SetEntryID(ve.ID).
		SetAccountID(s.rev4130.ID).
		SetDirection(posting.DirectionCREDIT).
		SetAmount(decimal.NewFromInt(50000)).
		SetBaseAmount(decimal.NewFromInt(50000)).
		SetPosition(1).
		Save(s.ctx)
	s.Require().NoError(err)

	// Nil status returns draft and posted (2), never void
	allDocs, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeIncome, model.SearchCashFlowRequest{})
	s.Require().NoError(err)
	s.Len(allDocs, 2)

	// Status DRAFT returns 1
	draftStatus := domain.EntryStatusDraft
	draftDocs, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeIncome, model.SearchCashFlowRequest{
		Status: &draftStatus,
	})
	s.Require().NoError(err)
	s.Len(draftDocs, 1)
	s.Equal(draftDoc.ID, draftDocs[0].ID)

	// Status POSTED returns 1
	postedStatus := domain.EntryStatusPosted
	postedDocs, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeIncome, model.SearchCashFlowRequest{
		Status: &postedStatus,
	})
	s.Require().NoError(err)
	s.Len(postedDocs, 1)
	s.Equal(postedDoc.ID, postedDocs[0].ID)
}

func (s *CashFlowRepoTestSuite) Test_Search_SameAccountOnTwoItems_ReturnsDocumentOnce() {
	// Owner's expense with 6710 twice: 2 x 55,000 and 5 x 3,000, total 125,000
	doc, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeExpense,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.cashAcc.ID,
		Items: []repository.CashFlowItemParams{
			{
				AccountID:   s.expense6710.ID,
				Description: "telur",
				Quantity:    decimal.NewFromInt(2),
				UnitPrice:   decimal.NewFromInt(55000),
				Amount:      decimal.NewFromInt(110000),
				BaseAmount:  decimal.NewFromInt(110000),
			},
			{
				AccountID:   s.expense6710.ID,
				Description: "indomie",
				Quantity:    decimal.NewFromInt(5),
				UnitPrice:   decimal.NewFromInt(3000),
				Amount:      decimal.NewFromInt(15000),
				BaseAmount:  decimal.NewFromInt(15000),
			},
		},
		TotalAmount:     decimal.NewFromInt(125000),
		TotalBaseAmount: decimal.NewFromInt(125000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC),
		Description:     "Belanja warung",
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	results, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		CategoryAccountID: &s.expense6710.ID,
	})
	s.Require().NoError(err)
	s.Len(results, 1)
	res := results[0]
	s.Equal(doc.ID, res.ID)
	s.True(decimal.NewFromInt(125000).Equal(res.Amount))
	s.Require().Len(res.Items, 2)
	s.Equal("telur", res.Items[0].Description)
	s.True(decimal.NewFromInt(2).Equal(res.Items[0].Quantity))
	s.True(decimal.NewFromInt(55000).Equal(res.Items[0].UnitPrice))
	s.True(decimal.NewFromInt(110000).Equal(res.Items[0].Amount))
	s.Equal("indomie", res.Items[1].Description)
	s.True(decimal.NewFromInt(5).Equal(res.Items[1].Quantity))
	s.True(decimal.NewFromInt(3000).Equal(res.Items[1].UnitPrice))
	s.True(decimal.NewFromInt(15000).Equal(res.Items[1].Amount))
}

func (s *CashFlowRepoTestSuite) Test_Update_ReplacesPostings_PreservesNumbers() {
	created, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeExpense,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.cashAcc.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.expenseAcc.ID, Description: "Old item", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50000), Amount: decimal.NewFromInt(50000), BaseAmount: decimal.NewFromInt(50000)},
		},
		TotalAmount:     decimal.NewFromInt(50000),
		TotalBaseAmount: decimal.NewFromInt(50000),
		Status:          domain.EntryStatusDraft,
		TransactionDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	updated, err := s.repo.Update(s.ctx, created.ID, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeExpense,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.bankAcc.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.expenseAcc2.ID, Description: "New item 1", Quantity: decimal.NewFromInt(2), UnitPrice: decimal.NewFromInt(30000), Amount: decimal.NewFromInt(60000), BaseAmount: decimal.NewFromInt(60000)},
			{AccountID: s.expense6710.ID, Description: "New item 2", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(40000), Amount: decimal.NewFromInt(40000), BaseAmount: decimal.NewFromInt(40000)},
		},
		TotalAmount:     decimal.NewFromInt(100000),
		TotalBaseAmount: decimal.NewFromInt(100000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC),
		Description:     "Updated expense",
		Reference:       "REF-UPDATED",
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	// Preserves numbers
	s.Equal(created.ID, updated.ID)
	s.Equal(created.Number, updated.Number)
	s.Equal(created.EntryID, updated.EntryID)
	s.Equal(created.EntryNumber, updated.EntryNumber)
	s.Equal(domain.EntryStatusPosted, updated.EntryStatus)
	s.True(decimal.NewFromInt(100000).Equal(updated.Amount))
	s.Require().Len(updated.Items, 2)

	// Check postings in DB
	e, err := s.client.Entry.Query().
		Where(entry.ID(created.EntryID)).
		WithPostings(func(pq *ent.PostingQuery) {
			pq.Order(ent.Asc(posting.FieldPosition))
		}).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Require().Len(e.Edges.Postings, 3)
	s.Equal(0, e.Edges.Postings[0].Position)
	s.Equal(s.bankAcc.ID, e.Edges.Postings[0].AccountID)
	s.Equal(posting.DirectionCREDIT, e.Edges.Postings[0].Direction)
	s.Equal(1, e.Edges.Postings[1].Position)
	s.Equal(s.expenseAcc2.ID, e.Edges.Postings[1].AccountID)
	s.Equal(posting.DirectionDEBIT, e.Edges.Postings[1].Direction)
	s.Equal(2, e.Edges.Postings[2].Position)
	s.Equal(s.expense6710.ID, e.Edges.Postings[2].AccountID)
	s.Equal(posting.DirectionDEBIT, e.Edges.Postings[2].Direction)
}

func (s *CashFlowRepoTestSuite) Test_Update_PostedEntry_ReturnsErrEntryNotDraft() {
	created, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeExpense,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.cashAcc.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.expenseAcc.ID, Description: "Item", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50000), Amount: decimal.NewFromInt(50000), BaseAmount: decimal.NewFromInt(50000)},
		},
		TotalAmount:     decimal.NewFromInt(50000),
		TotalBaseAmount: decimal.NewFromInt(50000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	_, err = s.repo.Update(s.ctx, created.ID, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeExpense,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.cashAcc.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.expenseAcc.ID, Description: "Item", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50000), Amount: decimal.NewFromInt(50000), BaseAmount: decimal.NewFromInt(50000)},
		},
		TotalAmount:     decimal.NewFromInt(50000),
		TotalBaseAmount: decimal.NewFromInt(50000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
	})
	s.Require().ErrorIs(err, domain.ErrEntryNotDraft)
}

func (s *CashFlowRepoTestSuite) Test_GetByID_WrongFlowType_ReturnsErrTransactionNotFound() {
	income, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeIncome,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.bankKaltim.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.rev4130.ID, Description: "Income item", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(500000), Amount: decimal.NewFromInt(500000), BaseAmount: decimal.NewFromInt(500000)},
		},
		TotalAmount:     decimal.NewFromInt(500000),
		TotalBaseAmount: decimal.NewFromInt(500000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC),
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	// Wrong flow type returns ErrTransactionNotFound
	_, err = s.repo.GetByID(s.ctx, domain.CashFlowTypeExpense, income.ID)
	s.Require().ErrorIs(err, domain.ErrTransactionNotFound)

	// Correct flow type succeeds
	got, err := s.repo.GetByID(s.ctx, domain.CashFlowTypeIncome, income.ID)
	s.Require().NoError(err)
	s.Equal(income.ID, got.ID)
	s.Equal(domain.CashFlowTypeIncome, got.Type)
}

func (s *CashFlowRepoTestSuite) Test_GetByID_LegacyPostingWithoutQuantity() {
	tx, err := s.client.Transaction.Create().
		SetNumber("TRX-LEGACY").
		SetTransactionDate(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)).
		SetCurrency("IDR").
		SetExchangeRate(decimal.NewFromInt(1)).
		SetSource(transaction.Source(domain.TransactionSourceManual)).
		Save(s.ctx)
	s.Require().NoError(err)

	e, err := s.client.Entry.Create().
		SetTransactionID(tx.ID).
		SetPeriodID(s.period.ID).
		SetNumber("JRN-LEGACY").
		SetEntryDate(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)).
		SetStatus(entry.Status(domain.EntryStatusPosted)).
		Save(s.ctx)
	s.Require().NoError(err)

	// Position 0: payment (CREDIT for expense)
	_, err = s.client.Posting.Create().
		SetEntryID(e.ID).
		SetAccountID(s.cashAcc.ID).
		SetDirection(posting.DirectionCREDIT).
		SetAmount(decimal.NewFromInt(75000)).
		SetBaseAmount(decimal.NewFromInt(75000)).
		SetPosition(0).
		Save(s.ctx)
	s.Require().NoError(err)

	// Position 1: expense item (DEBIT) without quantity or unit price
	_, err = s.client.Posting.Create().
		SetEntryID(e.ID).
		SetAccountID(s.expenseAcc.ID).
		SetDirection(posting.DirectionDEBIT).
		SetAmount(decimal.NewFromInt(75000)).
		SetBaseAmount(decimal.NewFromInt(75000)).
		SetPosition(1).
		Save(s.ctx)
	s.Require().NoError(err)

	got, err := s.repo.GetByID(s.ctx, domain.CashFlowTypeExpense, tx.ID)
	s.Require().NoError(err)
	s.Equal(tx.ID, got.ID)
	s.Require().Len(got.Items, 1)
	s.True(decimal.NewFromInt(1).Equal(got.Items[0].Quantity))
	s.True(decimal.NewFromInt(75000).Equal(got.Items[0].UnitPrice))
	s.True(decimal.NewFromInt(75000).Equal(got.Items[0].Amount))
}

func (s *CashFlowRepoTestSuite) TestSearchFilters() {
	exp1, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeExpense,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.cashAcc.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.expenseAcc.ID, Description: "Office", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(50000), Amount: decimal.NewFromInt(50000), BaseAmount: decimal.NewFromInt(50000)},
		},
		TotalAmount:     decimal.NewFromInt(50000),
		TotalBaseAmount: decimal.NewFromInt(50000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Description:     "Belanja Indomaret",
		Reference:       "STRUK-001",
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	exp2, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
		FlowType:         domain.CashFlowTypeExpense,
		PeriodID:         s.period.ID,
		PaymentAccountID: s.bankAcc.ID,
		Items: []repository.CashFlowItemParams{
			{AccountID: s.expenseAcc2.ID, Description: "Food", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000), Amount: decimal.NewFromInt(100000), BaseAmount: decimal.NewFromInt(100000)},
		},
		TotalAmount:     decimal.NewFromInt(100000),
		TotalBaseAmount: decimal.NewFromInt(100000),
		Status:          domain.EntryStatusPosted,
		TransactionDate: time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		Description:     "Makan Siang Alfamart",
		Reference:       "STRUK-002",
		Currency:        "IDR",
		ExchangeRate:    decimal.NewFromInt(1),
		PostedBy:        s.user.ID,
	})
	s.Require().NoError(err)

	dateFrom := model.Date(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))
	dateTo := model.Date(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC))
	dateFiltered, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
	})
	s.Require().NoError(err)
	s.Len(dateFiltered, 1)
	s.Equal(exp2.ID, dateFiltered[0].ID)

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

	catFiltered, _, err := s.repo.Search(s.ctx, domain.CashFlowTypeExpense, model.SearchCashFlowRequest{
		CategoryAccountID: &s.expenseAcc.ID,
	})
	s.Require().NoError(err)
	s.Len(catFiltered, 1)
	s.Equal(exp1.ID, catFiltered[0].ID)

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
		_, err := s.repo.Create(s.ctx, repository.CreateCashFlowParams{
			FlowType:         domain.CashFlowTypeExpense,
			PeriodID:         s.period.ID,
			PaymentAccountID: s.cashAcc.ID,
			Items: []repository.CashFlowItemParams{
				{AccountID: s.expenseAcc.ID, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(int64(i * 10000)), Amount: decimal.NewFromInt(int64(i * 10000)), BaseAmount: decimal.NewFromInt(int64(i * 10000))},
			},
			TotalAmount:     decimal.NewFromInt(int64(i * 10000)),
			TotalBaseAmount: decimal.NewFromInt(int64(i * 10000)),
			Status:          domain.EntryStatusPosted,
			TransactionDate: time.Date(2026, 10, i, 0, 0, 0, 0, time.UTC),
			Description:     "Test Pagination Expense",
			Currency:        "IDR",
			ExchangeRate:    decimal.NewFromInt(1),
			PostedBy:        s.user.ID,
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

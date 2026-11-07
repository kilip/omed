package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
	agentsvc "github.com/kilip/omed/api/internal/agent/service"
	"github.com/kilip/omed/api/internal/fin"
	findomain "github.com/kilip/omed/api/internal/fin/domain"
	finmodel "github.com/kilip/omed/api/internal/fin/model"
	"github.com/shopspring/decimal"
)

type draftAccountSearcher interface {
	Search(c context.Context, req finmodel.SearchAccountRequest) ([]findomain.Account, string, error)
}

type draftCurrencyLister interface {
	ListCurrencies(c context.Context) ([]findomain.Currency, error)
}

type draftPeriodSearcher interface {
	Search(c context.Context, req finmodel.SearchPeriodRequest) ([]findomain.Period, string, error)
}

type draftCashFlowService interface {
	CreateExpense(c context.Context, req finmodel.SaveCashFlowRequest) (findomain.CashFlowItem, error)
	CreateIncome(c context.Context, req finmodel.SaveCashFlowRequest) (findomain.CashFlowItem, error)
}

type draftTransactionService interface {
	Create(c context.Context, req finmodel.CreateTransactionRequest) (findomain.Transaction, error)
	Search(c context.Context, req finmodel.SearchTransactionRequest) ([]findomain.Transaction, string, error)
}

type draftInsightService interface {
	AccountBalance(ctx context.Context, req finmodel.AccountBalanceRequest) (finmodel.AccountBalanceResponse, error)
	PeriodOverview(c context.Context, req finmodel.PeriodOverviewRequest) (finmodel.PeriodOverviewResponse, error)
	FindCashflow(c context.Context, req finmodel.FindCashflowRequest) (finmodel.FindCashflowResponse, error)
}

type draftReportService interface {
	BaseCurrency(ctx context.Context) string
	TrialBalance(ctx context.Context, req finmodel.TrialBalanceRequest) (finmodel.TrialBalanceResponse, error)
	Summary(ctx context.Context, months int) (finmodel.FinancialSummaryResponse, error)
	ProfitLossBreakdown(ctx context.Context, req finmodel.ProfitLossBreakdownRequest) (finmodel.ProfitLossBreakdownResponse, error)
}

type draftEntryService interface {
	GetAccountLedger(c context.Context, accountID int64, page int, limit int) ([]findomain.Posting, int64, error)
	GetByID(c context.Context, id int64) (findomain.Entry, error)
}

type draftExchangeRateService interface {
	Lookup(c context.Context, from, to string, date time.Time) (decimal.Decimal, error)
	Search(c context.Context, req finmodel.SearchExchangeRateRequest) ([]findomain.ExchangeRate, string, error)
}

type financeDraftContextAdapter struct {
	accounts      draftAccountSearcher
	currencies    draftCurrencyLister
	periods       draftPeriodSearcher
	cashFlows     draftCashFlowService
	transactions  draftTransactionService
	insights      draftInsightService
	reports       draftReportService
	entries       draftEntryService
	exchangeRates draftExchangeRateService
}

func newFinanceDraftContext(finMod *fin.FinModule) agentsvc.FinanceDraftContext {
	if finMod == nil {
		return nil
	}
	return newFinanceDraftContextFromServices(
		finMod.AccountService,
		finMod.CurrencyService,
		finMod.PeriodService,
		finMod.CashFlowService,
		finMod.TransactionService,
		finMod.InsightService,
		finMod.ReportService,
		finMod.EntryService,
		finMod.ExchangeRateService,
	)
}

func newFinanceDraftContextFromServices(
	accounts draftAccountSearcher,
	currencies draftCurrencyLister,
	periods draftPeriodSearcher,
	cashFlows draftCashFlowService,
	transactions draftTransactionService,
	insights draftInsightService,
	reports draftReportService,
	entries draftEntryService,
	exchangeRates draftExchangeRateService,
) agentsvc.FinanceDraftContext {
	return &financeDraftContextAdapter{
		accounts:      accounts,
		currencies:    currencies,
		periods:       periods,
		cashFlows:     cashFlows,
		transactions:  transactions,
		insights:      insights,
		reports:       reports,
		entries:       entries,
		exchangeRates: exchangeRates,
	}
}

func (a *financeDraftContextAdapter) ListAccounts(ctx context.Context) ([]agentsvc.AccountOption, error) {
	if a.accounts == nil {
		return nil, nil
	}
	accs, _, err := a.accounts.Search(ctx, finmodel.SearchAccountRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]agentsvc.AccountOption, 0, len(accs))
	for _, acc := range accs {
		if !acc.IsActive || !acc.IsPostable {
			continue
		}
		out = append(out, agentsvc.AccountOption{
			ID:   acc.ID,
			Code: acc.Code,
			Name: acc.Name,
		})
	}
	return out, nil
}

func (a *financeDraftContextAdapter) ListCurrencies(ctx context.Context) ([]agentsvc.CurrencyOption, error) {
	if a.currencies == nil {
		return nil, nil
	}
	currencies, err := a.currencies.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]agentsvc.CurrencyOption, 0, len(currencies))
	for _, c := range currencies {
		out = append(out, agentsvc.CurrencyOption{
			Code:     c.Code,
			Name:     c.Name,
			Symbol:   c.Symbol,
			Decimals: c.Decimals,
		})
	}
	return out, nil
}

func (a *financeDraftContextAdapter) ListPeriods(ctx context.Context) ([]agentsvc.PeriodOption, error) {
	if a.periods == nil {
		return nil, nil
	}
	openStatus := findomain.PeriodStatusOpen
	periods, _, err := a.periods.Search(ctx, finmodel.SearchPeriodRequest{Status: &openStatus})
	if err != nil {
		return nil, err
	}
	out := make([]agentsvc.PeriodOption, 0, len(periods))
	for _, p := range periods {
		out = append(out, agentsvc.PeriodOption{
			ID:         p.ID,
			Name:       p.Name,
			StartDate:  p.StartDate,
			EndDate:    p.EndDate,
			FiscalYear: p.FiscalYear,
			Status:     p.Status,
		})
	}
	return out, nil
}

func parseDraftDate(dateStr string) (finmodel.Date, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return finmodel.Date{}, fmt.Errorf("invalid transaction date %q: %w", dateStr, err)
	}
	return finmodel.Date(t), nil
}

func (a *financeDraftContextAdapter) SaveDraft(ctx context.Context, proposal model.DraftProposal) (int64, error) {
	switch proposal.Kind {
	case model.DraftKindExpense:
		return a.saveExpenseDraft(ctx, proposal)
	case model.DraftKindIncome:
		return a.saveIncomeDraft(ctx, proposal)
	case model.DraftKindJournal:
		return a.saveJournalDraft(ctx, proposal)
	default:
		return 0, fmt.Errorf("unsupported draft kind: %s", proposal.Kind)
	}
}

func (a *financeDraftContextAdapter) saveExpenseDraft(ctx context.Context, proposal model.DraftProposal) (int64, error) {
	if a.cashFlows == nil {
		return 0, errors.New("cash flow service is not configured")
	}
	if proposal.PaymentAccountID == nil {
		return 0, errors.New("payment account is required for expense draft")
	}
	tDate, err := parseDraftDate(proposal.TransactionDate)
	if err != nil {
		return 0, err
	}

	items := make([]finmodel.CashFlowItemRequest, len(proposal.Items))
	for i, it := range proposal.Items {
		items[i] = finmodel.CashFlowItemRequest{
			AccountID:   it.AccountID,
			Description: it.Description,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		}
	}

	req := finmodel.SaveCashFlowRequest{
		PaymentAccountID: *proposal.PaymentAccountID,
		TransactionDate:  tDate,
		Reference:        proposal.Reference,
		Description:      proposal.Description,
		Currency:         proposal.Currency,
		Status:           findomain.EntryStatusDraft,
		ExchangeRate:     proposal.ExchangeRate,
		Items:            items,
	}

	item, err := a.cashFlows.CreateExpense(ctx, req)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (a *financeDraftContextAdapter) saveIncomeDraft(ctx context.Context, proposal model.DraftProposal) (int64, error) {
	if a.cashFlows == nil {
		return 0, errors.New("cash flow service is not configured")
	}
	if proposal.PaymentAccountID == nil {
		return 0, errors.New("payment account is required for income draft")
	}
	tDate, err := parseDraftDate(proposal.TransactionDate)
	if err != nil {
		return 0, err
	}

	items := make([]finmodel.CashFlowItemRequest, len(proposal.Items))
	for i, it := range proposal.Items {
		items[i] = finmodel.CashFlowItemRequest{
			AccountID:   it.AccountID,
			Description: it.Description,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
		}
	}

	req := finmodel.SaveCashFlowRequest{
		PaymentAccountID: *proposal.PaymentAccountID,
		TransactionDate:  tDate,
		Reference:        proposal.Reference,
		Description:      proposal.Description,
		Currency:         proposal.Currency,
		Status:           findomain.EntryStatusDraft,
		ExchangeRate:     proposal.ExchangeRate,
		Items:            items,
	}

	item, err := a.cashFlows.CreateIncome(ctx, req)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (a *financeDraftContextAdapter) saveJournalDraft(ctx context.Context, proposal model.DraftProposal) (int64, error) {
	if a.transactions == nil {
		return 0, errors.New("transaction service is not configured")
	}
	tDate, err := parseDraftDate(proposal.TransactionDate)
	if err != nil {
		return 0, err
	}

	var periodID int64
	if proposal.PeriodID != nil {
		periodID = *proposal.PeriodID
	} else {
		periods, pErr := a.ListPeriods(ctx)
		if pErr == nil {
			for _, p := range periods {
				if strings.EqualFold(p.Status, "open") && !tDate.Time().Before(p.StartDate) && !tDate.Time().After(p.EndDate) {
					periodID = p.ID
					break
				}
			}
			if periodID == 0 {
				for _, p := range periods {
					if strings.EqualFold(p.Status, "open") {
						periodID = p.ID
						break
					}
				}
			}
		}
		if periodID == 0 {
			return 0, errors.New("period ID is required for journal draft")
		}
	}
	postings := make([]finmodel.CreatePostingRequest, len(proposal.Entries))
	for i, entry := range proposal.Entries {
		postings[i] = finmodel.CreatePostingRequest{
			AccountID:   entry.AccountID,
			Direction:   entry.Direction,
			Amount:      entry.Amount,
			Description: entry.Description,
		}
	}

	req := finmodel.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     proposal.Description,
		Reference:       proposal.Reference,
		Currency:        proposal.Currency,
		ExchangeRate:    proposal.ExchangeRate,
		Entry: &finmodel.CreateEntryInlineRequest{
			PeriodID:    periodID,
			EntryDate:   tDate,
			Description: proposal.Description,
			Postings:    postings,
		},
	}

	tx, err := a.transactions.Create(ctx, req)
	if err != nil {
		return 0, err
	}
	return tx.ID, nil
}

func (a *financeDraftContextAdapter) GetAccountBalances(ctx context.Context, req agentsvc.AccountBalanceQuery) (agentsvc.AccountBalanceResult, error) {
	filterType := strings.ToUpper(strings.TrimSpace(req.Type))

	// If per-currency breakdown or type filter requested, use ReportService.TrialBalance
	if (req.IncludePerCurrencyBreakdown || filterType != "") && a.reports != nil {
		var asOf finmodel.Date
		if strings.TrimSpace(req.AsOfDate) != "" {
			parsedDate, err := parseDraftDate(req.AsOfDate)
			if err != nil {
				return agentsvc.AccountBalanceResult{}, err
			}
			asOf = parsedDate
		} else {
			asOf = finmodel.Date(time.Now())
		}

		includeBreakdown := req.IncludePerCurrencyBreakdown
		tb, err := a.reports.TrialBalance(ctx, finmodel.TrialBalanceRequest{
			AsOfDate:                    asOf,
			IncludePerCurrencyBreakdown: &includeBreakdown,
		})
		if err != nil {
			return agentsvc.AccountBalanceResult{}, err
		}

		accountFilter := strings.TrimSpace(req.Account)
		lowerFilter := strings.ToLower(accountFilter)

		var matchedAccounts []agentsvc.AccountBalanceItem
		totalDebit := decimal.Zero
		totalCredit := decimal.Zero
		matchedNetTotal := decimal.Zero

		for _, acc := range tb.Accounts {
			if filterType != "" && strings.ToUpper(acc.Type) != filterType {
				continue
			}
			if accountFilter != "" {
				if !strings.HasPrefix(strings.ToLower(acc.AccountCode), lowerFilter) &&
					!strings.Contains(strings.ToLower(acc.AccountName), lowerFilter) {
					continue
				}
			}

			var currencies []agentsvc.AccountCurrencyBalance
			if len(acc.Currencies) > 0 {
				currencies = make([]agentsvc.AccountCurrencyBalance, len(acc.Currencies))
				for i, c := range acc.Currencies {
					currencies[i] = agentsvc.AccountCurrencyBalance{
						Currency:   c.Currency,
						Debit:      c.Debit,
						Credit:     c.Credit,
						NetBalance: c.NetBalance,
						BaseDebit:  c.BaseDebit,
						BaseCredit: c.BaseCredit,
						BaseNet:    c.BaseNet,
					}
				}
			}

			matchedAccounts = append(matchedAccounts, agentsvc.AccountBalanceItem{
				AccountID:  acc.AccountID,
				Code:       acc.AccountCode,
				Name:       acc.AccountName,
				Type:       acc.Type,
				Debit:      acc.Debit,
				Credit:     acc.Credit,
				NetBalance: acc.NetBalance,
				Currencies: currencies,
			})
			totalDebit = totalDebit.Add(acc.Debit)
			totalCredit = totalCredit.Add(acc.Credit)
			matchedNetTotal = matchedNetTotal.Add(acc.NetBalance)
		}

		return agentsvc.AccountBalanceResult{
			AsOfDate:        tb.AsOfDate.Time().Format("2006-01-02"),
			BaseCurrency:    tb.BaseCurrency,
			Accounts:        matchedAccounts,
			TotalDebit:      totalDebit,
			TotalCredit:     totalCredit,
			MatchedNetTotal: matchedNetTotal,
			IsBalanced:      tb.IsBalanced,
		}, nil
	}

	// Otherwise delegate to InsightService.AccountBalance
	if a.insights == nil {
		return agentsvc.AccountBalanceResult{}, errors.New("insight service is not configured")
	}

	resp, err := a.insights.AccountBalance(ctx, finmodel.AccountBalanceRequest{
		AsOfDate: req.AsOfDate,
		Account:  req.Account,
	})
	if err != nil {
		return agentsvc.AccountBalanceResult{}, err
	}

	accounts := make([]agentsvc.AccountBalanceItem, len(resp.Accounts))
	matchedNetTotal := decimal.Zero
	for i, acc := range resp.Accounts {
		accounts[i] = agentsvc.AccountBalanceItem{
			Code:       acc.Code,
			Name:       acc.Name,
			Type:       acc.Type,
			Debit:      acc.Debit,
			Credit:     acc.Credit,
			NetBalance: acc.Net,
		}
		matchedNetTotal = matchedNetTotal.Add(acc.Net)
	}

	return agentsvc.AccountBalanceResult{
		AsOfDate:        resp.AsOfDate,
		BaseCurrency:    resp.BaseCurrency,
		Accounts:        accounts,
		TotalDebit:      resp.Totals.Debit,
		TotalCredit:     resp.Totals.Credit,
		MatchedNetTotal: matchedNetTotal,
		IsBalanced:      resp.IsBalanced,
		Truncated:       resp.Truncated,
		Hint:            resp.Hint,
	}, nil
}

func (a *financeDraftContextAdapter) GetFinancialSummary(ctx context.Context, req agentsvc.FinancialSummaryQuery) (agentsvc.FinancialSummaryResult, error) {
	// 1. Get balance sheet and income totals from reports.Summary
	var totals agentsvc.FinancialSummaryTotals
	var baseCurrency string
	var monthly []agentsvc.MonthlyRevenueExpense

	if a.reports != nil {
		summary, err := a.reports.Summary(ctx, 6)
		if err != nil {
			return agentsvc.FinancialSummaryResult{}, err
		}
		baseCurrency = summary.BaseCurrency
		totals = agentsvc.FinancialSummaryTotals{
			Assets:      summary.Totals.Assets,
			Liabilities: summary.Totals.Liabilities,
			Equity:      summary.Totals.Equity,
			Revenue:     summary.Totals.Revenue,
			Expense:     summary.Totals.Expense,
			NetIncome:   summary.Totals.NetIncome,
		}
		monthly = make([]agentsvc.MonthlyRevenueExpense, len(summary.Monthly))
		for i, m := range summary.Monthly {
			monthly[i] = agentsvc.MonthlyRevenueExpense{
				Month:   m.Month,
				Revenue: m.Revenue,
				Expense: m.Expense,
				Net:     m.Revenue.Sub(m.Expense),
			}
		}
	}

	// 2. If month range or compare previous specified and insight service available, use PeriodOverview for trend
	if (req.FromMonth != "" || req.ToMonth != "" || req.ComparePrevious) && a.insights != nil {
		comparePrev := req.ComparePrevious
		overview, err := a.insights.PeriodOverview(ctx, finmodel.PeriodOverviewRequest{
			FromMonth:       req.FromMonth,
			ToMonth:         req.ToMonth,
			ComparePrevious: &comparePrev,
		})
		if err != nil {
			return agentsvc.FinancialSummaryResult{}, err
		}
		if overview.BaseCurrency != "" {
			baseCurrency = overview.BaseCurrency
		}
		monthly = make([]agentsvc.MonthlyRevenueExpense, len(overview.Monthly))
		for i, m := range overview.Monthly {
			monthly[i] = agentsvc.MonthlyRevenueExpense{
				Month:   m.Month,
				Revenue: m.Revenue,
				Expense: m.Expense,
				Net:     m.Net,
			}
		}
	}

	if baseCurrency == "" && a.reports != nil {
		baseCurrency = a.reports.BaseCurrency(ctx)
	}

	return agentsvc.FinancialSummaryResult{
		BaseCurrency: baseCurrency,
		Totals:       totals,
		Monthly:      monthly,
	}, nil
}

func (a *financeDraftContextAdapter) GetProfitLossBreakdown(ctx context.Context, req agentsvc.ProfitLossQuery) (agentsvc.ProfitLossResult, error) {
	if a.reports == nil {
		return agentsvc.ProfitLossResult{}, errors.New("report service is not configured")
	}

	var dFrom, dTo finmodel.Date
	if strings.TrimSpace(req.DateFrom) != "" {
		df, err := parseDraftDate(req.DateFrom)
		if err != nil {
			return agentsvc.ProfitLossResult{}, err
		}
		dFrom = df
	}
	if strings.TrimSpace(req.DateTo) != "" {
		dt, err := parseDraftDate(req.DateTo)
		if err != nil {
			return agentsvc.ProfitLossResult{}, err
		}
		dTo = dt
	}

	res, err := a.reports.ProfitLossBreakdown(ctx, finmodel.ProfitLossBreakdownRequest{
		DateFrom: dFrom,
		DateTo:   dTo,
	})
	if err != nil {
		return agentsvc.ProfitLossResult{}, err
	}

	revenues := make([]agentsvc.ProfitLossCategoryItem, len(res.Revenues))
	for i, r := range res.Revenues {
		revenues[i] = agentsvc.ProfitLossCategoryItem{
			AccountID:   r.AccountID,
			AccountCode: r.AccountCode,
			AccountName: r.AccountName,
			Type:        r.Type,
			Amount:      r.Amount,
		}
	}

	expenses := make([]agentsvc.ProfitLossCategoryItem, len(res.Expenses))
	for i, e := range res.Expenses {
		expenses[i] = agentsvc.ProfitLossCategoryItem{
			AccountID:   e.AccountID,
			AccountCode: e.AccountCode,
			AccountName: e.AccountName,
			Type:        e.Type,
			Amount:      e.Amount,
		}
	}

	return agentsvc.ProfitLossResult{
		DateFrom:     res.DateFrom.Time().Format("2006-01-02"),
		DateTo:       res.DateTo.Time().Format("2006-01-02"),
		BaseCurrency: res.BaseCurrency,
		Revenues:     revenues,
		Expenses:     expenses,
		TotalRevenue: res.TotalRevenue,
		TotalExpense: res.TotalExpense,
		NetProfit:    res.NetProfit,
	}, nil
}

func (a *financeDraftContextAdapter) SearchCashflows(ctx context.Context, req agentsvc.CashflowSearchQuery) (agentsvc.CashflowSearchResult, error) {
	if a.insights == nil {
		return agentsvc.CashflowSearchResult{}, errors.New("insight service is not configured")
	}

	resp, err := a.insights.FindCashflow(ctx, finmodel.FindCashflowRequest{
		Type:            req.Type,
		DateFrom:        req.DateFrom,
		DateTo:          req.DateTo,
		Status:          req.Status,
		Query:           req.Query,
		PaymentAccount:  req.PaymentAccount,
		CategoryAccount: req.CategoryAccount,
		Cursor:          req.Cursor,
		Limit:           req.Limit,
	})
	if err != nil {
		return agentsvc.CashflowSearchResult{}, err
	}

	returnedSum := decimal.Zero
	items := make([]agentsvc.CashflowDocItem, len(resp.Items))
	for i, doc := range resp.Items {
		returnedSum = returnedSum.Add(doc.BaseAmount)
		lines := make([]agentsvc.CashflowLineItem, len(doc.Lines))
		for j, l := range doc.Lines {
			lines[j] = agentsvc.CashflowLineItem{
				AccountCode: l.Account.Code,
				AccountName: l.Account.Name,
				Description: l.Description,
				Quantity:    l.Quantity,
				UnitPrice:   l.UnitPrice,
				Amount:      l.Amount,
			}
		}
		items[i] = agentsvc.CashflowDocItem{
			ID:                 doc.ID,
			Number:             doc.Number,
			Date:               doc.Date,
			Type:               doc.Type,
			Status:             doc.Status,
			PaymentAccountCode: doc.PaymentAccount.Code,
			PaymentAccountName: doc.PaymentAccount.Name,
			Description:        doc.Description,
			Reference:          doc.Reference,
			Amount:             doc.Amount,
			Currency:           doc.Currency,
			BaseAmount:         doc.BaseAmount,
			Lines:              lines,
		}
	}

	return agentsvc.CashflowSearchResult{
		Items:                 items,
		ReturnedBaseAmountSum: returnedSum,
		NextCursor:            resp.NextCursor,
		Truncated:             resp.Truncated,
		Hint:                  resp.Hint,
	}, nil
}

func (a *financeDraftContextAdapter) resolveAccount(ctx context.Context, query string) (*findomain.Account, []agentsvc.AccountOption, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, nil, errors.New("account query is required")
	}
	if a.accounts == nil {
		return nil, nil, errors.New("account service is not configured")
	}

	accs, _, err := a.accounts.Search(ctx, finmodel.SearchAccountRequest{})
	if err != nil {
		return nil, nil, err
	}

	// 1. Exact numeric ID match
	for _, acc := range accs {
		if fmt.Sprintf("%d", acc.ID) == trimmed {
			return &acc, nil, nil
		}
	}

	// 2. Exact code match
	for _, acc := range accs {
		if acc.Code == trimmed {
			return &acc, nil, nil
		}
	}

	// 3. Exact name match (case-insensitive)
	var exactNameMatches []findomain.Account
	for _, acc := range accs {
		if strings.EqualFold(acc.Name, trimmed) {
			exactNameMatches = append(exactNameMatches, acc)
		}
	}
	if len(exactNameMatches) == 1 {
		return &exactNameMatches[0], nil, nil
	}
	if len(exactNameMatches) > 1 {
		candidates := make([]agentsvc.AccountOption, len(exactNameMatches))
		for i, m := range exactNameMatches {
			candidates[i] = agentsvc.AccountOption{ID: m.ID, Code: m.Code, Name: m.Name}
		}
		return nil, candidates, nil
	}

	// 4. Substring match on code or name (case-insensitive)
	lowerQuery := strings.ToLower(trimmed)
	var substrMatches []findomain.Account
	for _, acc := range accs {
		if strings.Contains(strings.ToLower(acc.Code), lowerQuery) || strings.Contains(strings.ToLower(acc.Name), lowerQuery) {
			substrMatches = append(substrMatches, acc)
		}
	}
	if len(substrMatches) == 1 {
		return &substrMatches[0], nil, nil
	}
	if len(substrMatches) > 1 {
		candidates := make([]agentsvc.AccountOption, len(substrMatches))
		for i, m := range substrMatches {
			candidates[i] = agentsvc.AccountOption{ID: m.ID, Code: m.Code, Name: m.Name}
		}
		return nil, candidates, nil
	}

	return nil, nil, fmt.Errorf("account not found: %s", trimmed)
}

func (a *financeDraftContextAdapter) GetAccountLedger(ctx context.Context, req agentsvc.AccountLedgerQuery) (agentsvc.AccountLedgerResult, error) {
	targetAccount, candidates, err := a.resolveAccount(ctx, req.Account)
	if err != nil {
		return agentsvc.AccountLedgerResult{}, err
	}
	if len(candidates) > 0 {
		return agentsvc.AccountLedgerResult{
			AmbiguousMatches: candidates,
		}, nil
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	if a.entries == nil {
		return agentsvc.AccountLedgerResult{}, errors.New("entry service is not configured")
	}

	postings, totalCount, err := a.entries.GetAccountLedger(ctx, targetAccount.ID, page, limit)
	if err != nil {
		return agentsvc.AccountLedgerResult{}, err
	}

	baseCurrency := "IDR"
	if a.reports != nil {
		baseCurrency = a.reports.BaseCurrency(ctx)
	}
	accountCurrency := targetAccount.Currency
	if accountCurrency == "" {
		accountCurrency = baseCurrency
	}

	// Cache entry lookup for enrichments
	entryCache := make(map[int64]findomain.Entry)
	ledgerItems := make([]agentsvc.AccountLedgerItem, len(postings))
	for i, p := range postings {
		var entryNumber, entryDate string
		if p.EntryID > 0 {
			e, ok := entryCache[p.EntryID]
			if !ok {
				entry, eErr := a.entries.GetByID(ctx, p.EntryID)
				if eErr == nil {
					e = entry
					entryCache[p.EntryID] = e
				}
			}
			entryNumber = e.Number
			if !e.EntryDate.IsZero() {
				entryDate = e.EntryDate.Format("2006-01-02")
			}
		}

		ledgerItems[i] = agentsvc.AccountLedgerItem{
			PostingID:          p.ID,
			EntryID:            p.EntryID,
			EntryNumber:        entryNumber,
			EntryDate:          entryDate,
			Direction:          p.Direction,
			Amount:             p.Amount,
			BaseAmount:         p.BaseAmount,
			Currency:           accountCurrency,
			Description:        p.Description,
			RunningBalance:     p.RunningBalance,
			RunningBaseBalance: p.RunningBaseBalance,
		}
	}

	return agentsvc.AccountLedgerResult{
		AccountID:    targetAccount.ID,
		AccountCode:  targetAccount.Code,
		AccountName:  targetAccount.Name,
		Type:         targetAccount.Type,
		Currency:     accountCurrency,
		BaseCurrency: baseCurrency,
		Postings:     ledgerItems,
		TotalCount:   totalCount,
		Page:         page,
		Limit:        limit,
	}, nil
}

func (a *financeDraftContextAdapter) SearchTransactions(ctx context.Context, req agentsvc.TransactionSearchQuery) (agentsvc.TransactionSearchResult, error) {
	if a.transactions == nil {
		return agentsvc.TransactionSearchResult{}, errors.New("transaction service is not configured")
	}

	var dFrom, dTo *finmodel.Date
	if strings.TrimSpace(req.DateFrom) != "" {
		df, err := parseDraftDate(req.DateFrom)
		if err != nil {
			return agentsvc.TransactionSearchResult{}, err
		}
		dFrom = &df
	}
	if strings.TrimSpace(req.DateTo) != "" {
		dt, err := parseDraftDate(req.DateTo)
		if err != nil {
			return agentsvc.TransactionSearchResult{}, err
		}
		dTo = &dt
	}

	var ref, source, q, cur *string
	if req.Reference != "" {
		r := req.Reference
		ref = &r
	}
	if req.Source != "" {
		s := req.Source
		source = &s
	}
	if req.Query != "" {
		queryStr := req.Query
		q = &queryStr
	}
	if req.Cursor != "" {
		c := req.Cursor
		cur = &c
	}
	var lim *int
	if req.Limit > 0 {
		l := req.Limit
		lim = &l
	}

	txs, nextCursor, err := a.transactions.Search(ctx, finmodel.SearchTransactionRequest{
		DateFrom:  dFrom,
		DateTo:    dTo,
		Reference: ref,
		Source:    source,
		Query:     q,
		Cursor:    cur,
		Limit:     lim,
	})
	if err != nil {
		return agentsvc.TransactionSearchResult{}, err
	}

	items := make([]agentsvc.TransactionItem, len(txs))
	for i, t := range txs {
		items[i] = agentsvc.TransactionItem{
			ID:              t.ID,
			Number:          t.Number,
			TransactionDate: t.TransactionDate.Format("2006-01-02"),
			Description:     t.Description,
			Reference:       t.Reference,
			Currency:        t.Currency,
			ExchangeRate:    t.ExchangeRate,
			Source:          t.Source,
			EntryCount:      len(t.Entries),
		}
	}

	return agentsvc.TransactionSearchResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

func (a *financeDraftContextAdapter) LookupExchangeRate(ctx context.Context, req agentsvc.ExchangeRateQuery) (agentsvc.ExchangeRateResult, error) {
	if a.exchangeRates == nil {
		return agentsvc.ExchangeRateResult{}, errors.New("exchange rate service is not configured")
	}

	from := strings.TrimSpace(req.FromCurrency)
	to := strings.TrimSpace(req.ToCurrency)
	if to == "" && a.reports != nil {
		to = a.reports.BaseCurrency(ctx)
	}
	if to == "" {
		to = "IDR"
	}

	var targetDate time.Time
	if strings.TrimSpace(req.Date) != "" {
		d, err := parseDraftDate(req.Date)
		if err != nil {
			return agentsvc.ExchangeRateResult{}, err
		}
		targetDate = d.Time()
	} else {
		targetDate = time.Now()
	}

	rate, err := a.exchangeRates.Lookup(ctx, from, to, targetDate)
	if err != nil {
		// Search historical or available rates for this pair
		var fromP, toP *string
		if from != "" {
			fromP = &from
		}
		if to != "" {
			toP = &to
		}
		searchRates, _, sErr := a.exchangeRates.Search(ctx, finmodel.SearchExchangeRateRequest{
			FromCurrency: fromP,
			ToCurrency:   toP,
		})
		if sErr == nil && len(searchRates) > 0 {
			items := make([]agentsvc.ExchangeRateItem, len(searchRates))
			for i, r := range searchRates {
				items[i] = agentsvc.ExchangeRateItem{
					ID:            r.ID,
					FromCurrency:  r.FromCurrency,
					ToCurrency:    r.ToCurrency,
					Rate:          r.Rate,
					EffectiveDate: r.EffectiveDate.Format("2006-01-02"),
					Source:        string(r.Source),
				}
			}
			return agentsvc.ExchangeRateResult{
				FromCurrency: from,
				ToCurrency:   to,
				Date:         targetDate.Format("2006-01-02"),
				Rates:        items,
			}, nil
		}
		return agentsvc.ExchangeRateResult{}, err
	}

	return agentsvc.ExchangeRateResult{
		Rate:         rate,
		FromCurrency: from,
		ToCurrency:   to,
		Date:         targetDate.Format("2006-01-02"),
	}, nil
}


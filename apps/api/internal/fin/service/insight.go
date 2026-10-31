package service

import (
	"context"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/shopspring/decimal"
)

const (
	maxInsightAccounts  = 50
	maxInsightItems     = 50
	maxInsightTextRunes = 200
)

// InsightService orchestrates financial data from multiple services to answer intent-oriented queries.
type InsightService struct {
	report   ReportService
	accounts AccountService
	cashflow CashFlowService
	now      func() time.Time
}

// InsightOption configures an InsightService instance.
type InsightOption func(*InsightService)

// WithInsightClock overrides the clock used to determine current time.
func WithInsightClock(now func() time.Time) InsightOption {
	return func(s *InsightService) {
		s.now = now
	}
}

// NewInsightService constructs a new InsightService.
func NewInsightService(report ReportService, accounts AccountService, cashflow CashFlowService, opts ...InsightOption) InsightService {
	s := InsightService{
		report:   report,
		accounts: accounts,
		cashflow: cashflow,
		now:      time.Now,
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// PeriodOverview computes revenue, expense, and net profit over a monthly range with optional previous-period comparison.
func (s InsightService) PeriodOverview(c context.Context, req model.PeriodOverviewRequest) (model.PeriodOverviewResponse, error) {
	if base.WorkspaceIDFromContext(c) == 0 && req.WorkspaceID > 0 {
		c = base.WithWorkspaceID(c, req.WorkspaceID)
	}

	clockNow := s.now()
	fromStr := req.FromMonth
	if fromStr == "" {
		fromStr = clockNow.Format("2006-01")
	}
	toStr := req.ToMonth
	if toStr == "" {
		toStr = clockNow.Format("2006-01")
	}

	fromDate, err := time.Parse("2006-01", fromStr)
	if err != nil {
		return model.PeriodOverviewResponse{}, domain.ArgumentError{Field: "from_month", Expect: "from_month must use the format YYYY-MM"}
	}
	toDate, err := time.Parse("2006-01", toStr)
	if err != nil {
		return model.PeriodOverviewResponse{}, domain.ArgumentError{Field: "to_month", Expect: "to_month must use the format YYYY-MM"}
	}

	if toDate.Before(fromDate) {
		return model.PeriodOverviewResponse{}, domain.ArgumentError{Field: "to_month", Expect: "to_month must not be before from_month"}
	}

	fromYear, fromMonth, _ := fromDate.Date()
	toYear, toMonth, _ := toDate.Date()
	numMonths := (toYear-fromYear)*12 + int(toMonth-fromMonth) + 1
	if numMonths > 24 {
		return model.PeriodOverviewResponse{}, domain.ArgumentError{Field: "range", Expect: "range must not exceed 24 months"}
	}

	comparePrevious := true
	if req.ComparePrevious != nil {
		comparePrevious = *req.ComparePrevious
	}

	items, err := s.report.MonthlyRevenueExpense(c, fromDate, numMonths)
	if err != nil {
		return model.PeriodOverviewResponse{}, err
	}

	totalRev := decimal.Zero
	totalExp := decimal.Zero
	totalNet := decimal.Zero
	monthly := make([]model.MonthTotals, 0, len(items))
	for _, item := range items {
		net := item.Revenue.Sub(item.Expense)
		totalRev = totalRev.Add(item.Revenue)
		totalExp = totalExp.Add(item.Expense)
		totalNet = totalNet.Add(net)
		monthly = append(monthly, model.MonthTotals{
			Month:   item.Month,
			Revenue: item.Revenue,
			Expense: item.Expense,
			Net:     net,
		})
	}

	var previous *model.PeriodComparison
	if comparePrevious {
		prevEnd := fromDate.AddDate(0, -1, 0)
		prevStart := prevEnd.AddDate(0, -(numMonths - 1), 0)
		prevItems, err := s.report.MonthlyRevenueExpense(c, prevStart, numMonths)
		if err != nil {
			return model.PeriodOverviewResponse{}, err
		}

		prevRev := decimal.Zero
		prevExp := decimal.Zero
		prevNet := decimal.Zero
		prevMonthly := make([]model.MonthTotals, 0, len(prevItems))
		for _, item := range prevItems {
			net := item.Revenue.Sub(item.Expense)
			prevRev = prevRev.Add(item.Revenue)
			prevExp = prevExp.Add(item.Expense)
			prevNet = prevNet.Add(net)
			prevMonthly = append(prevMonthly, model.MonthTotals{
				Month:   item.Month,
				Revenue: item.Revenue,
				Expense: item.Expense,
				Net:     net,
			})
		}

		previous = &model.PeriodComparison{
			From:    prevStart.Format("2006-01"),
			To:      prevEnd.Format("2006-01"),
			Revenue: prevRev,
			Expense: prevExp,
			Net:     prevNet,
			Monthly: prevMonthly,
			Delta: model.TotalsDelta{
				Revenue: totalRev.Sub(prevRev),
				Expense: totalExp.Sub(prevExp),
				Net:     totalNet.Sub(prevNet),
			},
		}
	}

	baseCurrency := s.report.BaseCurrency(c)

	return model.PeriodOverviewResponse{
		BaseCurrency: baseCurrency,
		From:         fromStr,
		To:           toStr,
		Revenue:      totalRev,
		Expense:      totalExp,
		Net:          totalNet,
		Monthly:      monthly,
		Previous:     previous,
	}, nil
}

// AccountBalance returns accounts and their balances as of a specific date, with optional code prefix or name filter.
func (s InsightService) AccountBalance(ctx context.Context, req model.AccountBalanceRequest) (model.AccountBalanceResponse, error) {
	if base.WorkspaceIDFromContext(ctx) == 0 && req.WorkspaceID > 0 {
		ctx = base.WithWorkspaceID(ctx, req.WorkspaceID)
	}

	asOfStr := strings.TrimSpace(req.AsOfDate)
	if asOfStr == "" {
		asOfStr = s.now().Format("2006-01-02")
	}

	asOfDate, err := time.Parse("2006-01-02", asOfStr)
	if err != nil {
		return model.AccountBalanceResponse{}, domain.ArgumentError{
			Field:  "as_of_date",
			Expect: "as_of_date must use the format YYYY-MM-DD",
		}
	}

	tbRes, err := s.report.TrialBalance(ctx, model.TrialBalanceRequest{
		AsOfDate: model.Date(asOfDate),
	})
	if err != nil {
		return model.AccountBalanceResponse{}, err
	}

	filter := strings.TrimSpace(req.Account)
	isDigits := len(filter) > 0 && isDigitsOnly(filter)
	lowerFilter := strings.ToLower(filter)

	var matched []model.AccountBalanceItem
	totalDebit := decimal.Zero
	totalCredit := decimal.Zero

	for _, acc := range tbRes.Accounts {
		if filter != "" {
			if isDigits {
				if !strings.HasPrefix(acc.AccountCode, filter) {
					continue
				}
			} else {
				if !strings.HasPrefix(strings.ToLower(acc.AccountCode), lowerFilter) && !strings.Contains(strings.ToLower(acc.AccountName), lowerFilter) {
					continue
				}
			}
		}

		item := model.AccountBalanceItem{
			Code:   acc.AccountCode,
			Name:   acc.AccountName,
			Type:   acc.Type,
			Debit:  acc.Debit,
			Credit: acc.Credit,
			Net:    acc.NetBalance,
		}
		matched = append(matched, item)
		totalDebit = totalDebit.Add(acc.Debit)
		totalCredit = totalCredit.Add(acc.Credit)
	}

	res := model.AccountBalanceResponse{
		AsOfDate:     asOfStr,
		BaseCurrency: tbRes.BaseCurrency,
		Totals: model.BalanceTotals{
			Debit:  totalDebit,
			Credit: totalCredit,
		},
		IsBalanced: tbRes.IsBalanced,
	}

	if len(matched) > maxInsightAccounts {
		res.Accounts = matched[:maxInsightAccounts]
		res.Truncated = true
		res.Hint = "narrow your account filter"
	} else if matched != nil {
		res.Accounts = matched
	} else {
		res.Accounts = []model.AccountBalanceItem{}
	}

	return res, nil
}

func isDigitsOnly(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// FindCashflow finds cashflow documents based on filter criteria.
func (s InsightService) FindCashflow(c context.Context, req model.FindCashflowRequest) (model.FindCashflowResponse, error) {
	if base.WorkspaceIDFromContext(c) == 0 && req.WorkspaceID > 0 {
		c = base.WithWorkspaceID(c, req.WorkspaceID)
	}

	limit := req.Limit
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > maxInsightItems {
		return model.FindCashflowResponse{}, domain.ArgumentError{Field: "limit", Expect: "limit must be between 1 and 50; use cursor to read more"}
	}

	typ := strings.ToLower(strings.TrimSpace(req.Type))
	if typ != "income" && typ != "expense" {
		return model.FindCashflowResponse{}, domain.ArgumentError{Field: "type", Expect: `type must be "income" or "expense"`}
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != "" && status != "draft" && status != "posted" {
		return model.FindCashflowResponse{}, domain.ArgumentError{Field: "status", Expect: `status must be "draft" or "posted"`}
	}

	// Date parsing
	var dateFrom *model.Date
	if req.DateFrom != "" {
		t, err := time.Parse("2006-01-02", req.DateFrom)
		if err != nil {
			return model.FindCashflowResponse{}, domain.ArgumentError{Field: "date_from", Expect: "invalid date format, expected YYYY-MM-DD"}
		}
		d := model.Date(t)
		dateFrom = &d
	}
	var dateTo *model.Date
	if req.DateTo != "" {
		t, err := time.Parse("2006-01-02", req.DateTo)
		if err != nil {
			return model.FindCashflowResponse{}, domain.ArgumentError{Field: "date_to", Expect: "invalid date format, expected YYYY-MM-DD"}
		}
		d := model.Date(t)
		dateTo = &d
	}

	// Account resolution if payment_account or category_account specified
	var paymentAccountID *int64
	var categoryAccountID *int64
	if strings.TrimSpace(req.PaymentAccount) != "" || strings.TrimSpace(req.CategoryAccount) != "" {
		accs, _, err := s.accounts.Search(c, model.SearchAccountRequest{})
		if err != nil {
			return model.FindCashflowResponse{}, err
		}
		if strings.TrimSpace(req.PaymentAccount) != "" {
			acc, err := resolveAccountRef(accs, req.PaymentAccount)
			if err != nil {
				return model.FindCashflowResponse{}, err
			}
			paymentAccountID = &acc.ID
		}
		if strings.TrimSpace(req.CategoryAccount) != "" {
			acc, err := resolveAccountRef(accs, req.CategoryAccount)
			if err != nil {
				return model.FindCashflowResponse{}, err
			}
			categoryAccountID = &acc.ID
		}
	}

	searchReq := model.SearchCashFlowRequest{
		DateFrom:          dateFrom,
		DateTo:            dateTo,
		PaymentAccountID:  paymentAccountID,
		CategoryAccountID: categoryAccountID,
		Limit:             &limit,
	}
	if req.Cursor != "" {
		c := req.Cursor
		searchReq.Cursor = &c
	}
	if req.Query != "" {
		q := req.Query
		searchReq.Query = &q
	}
	if status != "" {
		st := strings.ToUpper(status)
		searchReq.Status = &st
	}

	var items []domain.CashFlowItem
	var nextCursor string

	if typ == "income" {
		p, cur, err := s.cashflow.SearchIncomes(c, searchReq)
		if err != nil {
			return model.FindCashflowResponse{}, err
		}
		items = p
		nextCursor = cur
	} else {
		p, cur, err := s.cashflow.SearchExpenses(c, searchReq)
		if err != nil {
			return model.FindCashflowResponse{}, err
		}
		items = p
		nextCursor = cur
	}

	docs := make([]model.CashflowDoc, 0, len(items))
	for _, it := range items {
		docType := strings.ToLower(it.Type)
		docStatus := strings.ToLower(it.EntryStatus)

		lines := make([]model.CashflowLine, 0, len(it.Items))
		for _, l := range it.Items {
			lines = append(lines, model.CashflowLine{
				Account: model.AccountRef{
					Code: l.Account.Code,
					Name: l.Account.Name,
				},
				Description: truncateText(l.Description, maxInsightTextRunes),
				Quantity:    l.Quantity,
				UnitPrice:   l.UnitPrice,
				Amount:      l.Amount,
			})
		}

		docs = append(docs, model.CashflowDoc{
			ID:     it.ID,
			Number: it.Number,
			Date:   time.Time(it.TransactionDate).Format("2006-01-02"),
			Type:   docType,
			Status: docStatus,
			PaymentAccount: model.AccountRef{
				Code: it.PaymentAccount.Code,
				Name: it.PaymentAccount.Name,
			},
			Description: truncateText(it.Description, maxInsightTextRunes),
			Reference:   truncateText(it.Reference, maxInsightTextRunes),
			Amount:      it.Amount,
			Currency:    it.Currency,
			BaseAmount:  it.BaseAmount,
			Lines:       lines,
		})
	}

	return model.FindCashflowResponse{
		Items:      docs,
		NextCursor: nextCursor,
	}, nil
}

func resolveAccountRef(accounts []domain.Account, ref string) (domain.Account, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return domain.Account{}, domain.ErrAccountNotFound
	}

	// 1. Exact match on Code (case-sensitive) takes precedence (RF4)
	for _, a := range accounts {
		if a.Code == trimmed {
			return a, nil
		}
	}

	// 2. Exact match on Code case-insensitive
	lowerRef := strings.ToLower(trimmed)
	var exactCodeMatches []domain.Account
	for _, a := range accounts {
		if strings.ToLower(a.Code) == lowerRef {
			exactCodeMatches = append(exactCodeMatches, a)
		}
	}
	if len(exactCodeMatches) == 1 {
		return exactCodeMatches[0], nil
	}
	if len(exactCodeMatches) > 1 {
		candidates := make([]domain.AccountRef, 0, len(exactCodeMatches))
		for _, c := range exactCodeMatches {
			candidates = append(candidates, domain.AccountRef{Code: c.Code, Name: c.Name})
		}
		return domain.Account{}, domain.NewAmbiguousAccountError(candidates)
	}

	// 3. Exact match on Name (case-insensitive)
	var exactNameMatches []domain.Account
	for _, a := range accounts {
		if strings.EqualFold(a.Name, trimmed) {
			exactNameMatches = append(exactNameMatches, a)
		}
	}
	if len(exactNameMatches) == 1 {
		return exactNameMatches[0], nil
	}
	if len(exactNameMatches) > 1 {
		candidates := make([]domain.AccountRef, 0, len(exactNameMatches))
		for _, c := range exactNameMatches {
			candidates = append(candidates, domain.AccountRef{Code: c.Code, Name: c.Name})
		}
		return domain.Account{}, domain.NewAmbiguousAccountError(candidates)
	}

	// 4. Substring / contains match on Code or Name (case-insensitive)
	var matches []domain.Account
	for _, a := range accounts {
		if strings.Contains(strings.ToLower(a.Code), lowerRef) || strings.Contains(strings.ToLower(a.Name), lowerRef) {
			matches = append(matches, a)
		}
	}

	if len(matches) == 0 {
		return domain.Account{}, domain.ErrAccountNotFound
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	candidates := make([]domain.AccountRef, 0, len(matches))
	for _, c := range matches {
		candidates = append(candidates, domain.AccountRef{Code: c.Code, Name: c.Name})
	}
	return domain.Account{}, domain.NewAmbiguousAccountError(candidates)
}

func truncateText(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}

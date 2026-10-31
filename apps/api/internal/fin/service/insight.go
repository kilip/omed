package service

import (
	"context"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/shopspring/decimal"
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

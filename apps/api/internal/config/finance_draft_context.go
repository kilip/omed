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
}

type financeDraftContextAdapter struct {
	accounts    draftAccountSearcher
	currencies  draftCurrencyLister
	periods     draftPeriodSearcher
	cashFlows   draftCashFlowService
	transactions draftTransactionService
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
	)
}

func newFinanceDraftContextFromServices(
	accounts draftAccountSearcher,
	currencies draftCurrencyLister,
	periods draftPeriodSearcher,
	cashFlows draftCashFlowService,
	transactions draftTransactionService,
) agentsvc.FinanceDraftContext {
	return &financeDraftContextAdapter{
		accounts:     accounts,
		currencies:   currencies,
		periods:      periods,
		cashFlows:    cashFlows,
		transactions: transactions,
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

// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/kilip/omed/api/internal/base"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/shopspring/decimal"
)

// CashFlowRepositoryContract defines persistence operations for quick-entry cash flows.
type CashFlowRepositoryContract interface {
	CreatePosted(c context.Context, params repository.CreateCashFlowParams) (domain.CashFlowItem, error)
	Search(c context.Context, flowType string, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error)
}

// CashFlowService manages business logic and validation for quick-entry cash flows (expenses & incomes).
type CashFlowService struct {
	repo         CashFlowRepositoryContract
	periodRepo   PeriodRepository
	accountRepo  AccountRepository
	rateRepo     ExchangeRateRepository
	currencyRepo CurrencyRepository
	fileSvc      FileServiceContract
}

// NewCashFlowService constructs a new CashFlowService backed by the given repositories and services.
func NewCashFlowService(
	repo CashFlowRepositoryContract,
	periodRepo PeriodRepository,
	accountRepo AccountRepository,
	rateRepo ExchangeRateRepository,
	currencyRepo CurrencyRepository,
	fileSvc FileServiceContract,
) CashFlowService {
	return CashFlowService{
		repo:         repo,
		periodRepo:   periodRepo,
		accountRepo:  accountRepo,
		rateRepo:     rateRepo,
		currencyRepo: currencyRepo,
		fileSvc:      fileSvc,
	}
}

func (s CashFlowService) getBaseCurrencyAndDecimals(c context.Context) (string, int) {
	baseCurrency := "IDR"
	baseDecimals := 0
	wsID := base.WorkspaceIDFromContext(c)
	if s.currencyRepo != nil && wsID > 0 {
		if bc, err := s.currencyRepo.GetBaseCurrency(c, wsID); err == nil && bc != "" {
			baseCurrency = bc
		}
	}
	if s.currencyRepo != nil {
		if cur, err := s.currencyRepo.GetByCode(c, baseCurrency); err == nil {
			baseDecimals = cur.Decimals
		} else if baseCurrency == "USD" {
			baseDecimals = 2
		}
	}
	return baseCurrency, baseDecimals
}

func (s CashFlowService) resolveRate(c context.Context, currency string, explicitRate *decimal.Decimal, date time.Time) (decimal.Decimal, error) {
	if explicitRate != nil {
		if !explicitRate.IsPositive() {
			return decimal.Zero, domain.ErrExchangeRateInvalid
		}
		return *explicitRate, nil
	}
	baseCurrency, _ := s.getBaseCurrencyAndDecimals(c)
	if currency == "" || currency == baseCurrency {
		return decimal.NewFromInt(1), nil
	}
	if s.rateRepo == nil {
		return decimal.NewFromInt(1), nil
	}
	return s.rateRepo.Lookup(c, currency, baseCurrency, date)
}

func (s CashFlowService) createCashFlow(
	c context.Context,
	flowType string,
	paymentAccountID, categoryAccountID int64,
	expectedCategoryType string,
	amount decimal.Decimal,
	tDate time.Time,
	description, reference, currency string,
	explicitRate *decimal.Decimal,
	attachmentFileIDs []int64,
) (domain.CashFlowItem, error) {
	if !amount.IsPositive() {
		return domain.CashFlowItem{}, domain.ErrPostingInvalidAmount
	}

	period, err := s.periodRepo.FindByDate(c, tDate)
	if err != nil {
		if errors.Is(err, base.ErrNotFound) {
			return domain.CashFlowItem{}, domain.ErrPeriodNotFound
		}
		return domain.CashFlowItem{}, err
	}
	if period.Status != domain.PeriodStatusOpen {
		return domain.CashFlowItem{}, domain.ErrPeriodClosed
	}

	paymentAcc, err := s.accountRepo.GetByID(c, paymentAccountID)
	if err != nil {
		return domain.CashFlowItem{}, err
	}
	categoryAcc, err := s.accountRepo.GetByID(c, categoryAccountID)
	if err != nil {
		return domain.CashFlowItem{}, err
	}

	if !paymentAcc.IsActive || !categoryAcc.IsActive {
		return domain.CashFlowItem{}, domain.ErrPostingAccountInactive
	}
	if !paymentAcc.IsPostable || !categoryAcc.IsPostable {
		return domain.CashFlowItem{}, domain.ErrPostingAccountNotPostable
	}

	if paymentAcc.Type != domain.AccountTypeAsset || categoryAcc.Type != expectedCategoryType {
		return domain.CashFlowItem{}, domain.ErrAccountInvalidType
	}

	baseCurrency, baseDecimals := s.getBaseCurrencyAndDecimals(c)
	curr := baseCurrency
	if currency != "" {
		curr = currency
	}

	if paymentAcc.Currency != "" && paymentAcc.Currency != curr {
		return domain.CashFlowItem{}, domain.ErrPostingCurrencyMismatch
	}
	if categoryAcc.Currency != "" && categoryAcc.Currency != curr {
		return domain.CashFlowItem{}, domain.ErrPostingCurrencyMismatch
	}

	rate, err := s.resolveRate(c, curr, explicitRate, tDate)
	if err != nil {
		return domain.CashFlowItem{}, err
	}
	baseAmount := amount.Mul(rate).Round(int32(baseDecimals))

	var validatedFiles []coredomain.File
	if len(attachmentFileIDs) > 0 {
		if s.fileSvc == nil {
			return domain.CashFlowItem{}, base.ErrUnknown
		}
		validatedFiles = make([]coredomain.File, 0, len(attachmentFileIDs))
		for _, fileID := range attachmentFileIDs {
			f, err := s.fileSvc.GetByID(c, fileID)
			if err != nil {
				return domain.CashFlowItem{}, err
			}
			validatedFiles = append(validatedFiles, f)
		}
	}

	postedBy := base.UserIDFromContext(c)

	params := repository.CreateCashFlowParams{
		FlowType:          flowType,
		PeriodID:          period.ID,
		PaymentAccountID:  paymentAcc.ID,
		CategoryAccountID: categoryAcc.ID,
		Amount:            amount,
		BaseAmount:        baseAmount,
		TransactionDate:   tDate,
		Description:       description,
		Reference:         reference,
		Currency:          curr,
		ExchangeRate:      rate,
		PostedBy:          postedBy,
	}

	item, err := s.repo.CreatePosted(c, params)
	if err != nil {
		return domain.CashFlowItem{}, err
	}

	if s.fileSvc != nil && len(validatedFiles) > 0 {
		item.Attachments = make([]coredomain.File, 0, len(validatedFiles))
		for _, f := range validatedFiles {
			updated, err := s.fileSvc.Update(c, f.ID, coremodel.UpdateFileRequest{
				Name:       f.Name,
				EntityType: domain.TransactionEntityType,
				EntityID:   item.ID,
			})
			if err != nil {
				return domain.CashFlowItem{}, err
			}
			item.Attachments = append(item.Attachments, updated)
		}
	}

	return item, nil
}

// CreateExpense records a posted expense cash flow with Debit Expense and Credit Asset postings.
func (s CashFlowService) CreateExpense(c context.Context, req model.CreateExpenseRequest) (domain.CashFlowItem, error) {
	return s.createCashFlow(
		c,
		domain.CashFlowTypeExpense,
		req.PaymentAccountID,
		req.ExpenseAccountID,
		domain.AccountTypeExpense,
		req.Amount,
		req.TransactionDate.Time(),
		req.Description,
		req.Reference,
		req.Currency,
		req.ExchangeRate,
		req.AttachmentFileIDs,
	)
}

// CreateIncome records a posted income cash flow with Debit Asset and Credit Revenue postings.
func (s CashFlowService) CreateIncome(c context.Context, req model.CreateIncomeRequest) (domain.CashFlowItem, error) {
	return s.createCashFlow(
		c,
		domain.CashFlowTypeIncome,
		req.PaymentAccountID,
		req.RevenueAccountID,
		domain.AccountTypeRevenue,
		req.Amount,
		req.TransactionDate.Time(),
		req.Description,
		req.Reference,
		req.Currency,
		req.ExchangeRate,
		req.AttachmentFileIDs,
	)
}

// SearchExpenses searches expense cash flows matching the given criteria.
func (s CashFlowService) SearchExpenses(c context.Context, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error) {
	items, cursor, err := s.repo.Search(c, domain.CashFlowTypeExpense, req)
	if err != nil {
		return nil, "", err
	}
	s.enrichAttachments(c, items)
	return items, cursor, nil
}

// SearchIncomes searches income cash flows matching the given criteria.
func (s CashFlowService) SearchIncomes(c context.Context, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error) {
	items, cursor, err := s.repo.Search(c, domain.CashFlowTypeIncome, req)
	if err != nil {
		return nil, "", err
	}
	s.enrichAttachments(c, items)
	return items, cursor, nil
}

func (s CashFlowService) enrichAttachments(c context.Context, items []domain.CashFlowItem) {
	if s.fileSvc == nil {
		return
	}
	for i := range items {
		files, _, err := s.fileSvc.Search(c, coremodel.SearchFileRequest{
			EntityType: domain.TransactionEntityType,
			EntityID:   items[i].ID,
			Limit:      100,
		})
		if err == nil && len(files) > 0 {
			items[i].Attachments = files
		}
	}
}

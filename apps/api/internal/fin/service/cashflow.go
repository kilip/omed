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
	Create(c context.Context, params repository.CreateCashFlowParams) (domain.CashFlowItem, error)
	Update(c context.Context, id int64, params repository.CreateCashFlowParams) (domain.CashFlowItem, error)
	GetByID(c context.Context, flowType string, id int64) (domain.CashFlowItem, error)
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

func (s CashFlowService) currencyDecimals(c context.Context, currency string) int32 {
	if s.currencyRepo != nil && currency != "" {
		if cur, err := s.currencyRepo.GetByCode(c, currency); err == nil {
			dec := int32(cur.Decimals)
			if dec > 4 {
				return 4
			}
			if dec < 0 {
				return 0
			}
			return dec
		}
	}
	if currency == "IDR" {
		return 0
	}
	if currency == "USD" {
		return 2
	}
	return 4
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

func (s CashFlowService) saveCashFlow(
	c context.Context,
	flowType string,
	id int64,
	req model.SaveCashFlowRequest,
) (domain.CashFlowItem, error) {
	// 1. Items count (1 to 100), positive quantity, non-negative unit price
	if len(req.Items) < 1 || len(req.Items) > 100 {
		return domain.CashFlowItem{}, domain.ErrPostingInvalidAmount
	}
	for _, it := range req.Items {
		if !it.Quantity.IsPositive() {
			return domain.CashFlowItem{}, domain.ErrPostingInvalidQuantity
		}
		if it.UnitPrice.IsNegative() {
			return domain.CashFlowItem{}, domain.ErrPostingInvalidUnitPrice
		}
	}

	// Resolve document currency
	baseCurrency, baseDecimals := s.getBaseCurrencyAndDecimals(c)
	curr := baseCurrency
	if req.Currency != "" {
		curr = req.Currency
	}

	// 2. Line amounts = quantity * unitPrice rounded to document currency decimals (must be positive)
	currDecimals := s.currencyDecimals(c, curr)
	for _, it := range req.Items {
		lineAmount := it.Quantity.Mul(it.UnitPrice).Round(currDecimals)
		if !lineAmount.IsPositive() {
			return domain.CashFlowItem{}, domain.ErrPostingInvalidAmount
		}
	}

	// 3. Transaction date resolves to existing, open period
	tDate := req.TransactionDate.Time()
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

	// 4. Payment account (active, postable ASSET) & item accounts (active, postable REVENUE/EXPENSE)
	paymentAcc, err := s.accountRepo.GetByID(c, req.PaymentAccountID)
	if err != nil {
		return domain.CashFlowItem{}, err
	}
	if !paymentAcc.IsActive {
		return domain.CashFlowItem{}, domain.ErrPostingAccountInactive
	}
	if !paymentAcc.IsPostable {
		return domain.CashFlowItem{}, domain.ErrPostingAccountNotPostable
	}
	if paymentAcc.Type != domain.AccountTypeAsset {
		return domain.CashFlowItem{}, domain.ErrAccountInvalidType
	}

	expectedCategoryType := domain.AccountTypeExpense
	if flowType == domain.CashFlowTypeIncome {
		expectedCategoryType = domain.AccountTypeRevenue
	}

	itemAccounts := make([]domain.Account, len(req.Items))
	for i, it := range req.Items {
		catAcc, err := s.accountRepo.GetByID(c, it.AccountID)
		if err != nil {
			return domain.CashFlowItem{}, err
		}
		if !catAcc.IsActive {
			return domain.CashFlowItem{}, domain.ErrPostingAccountInactive
		}
		if !catAcc.IsPostable {
			return domain.CashFlowItem{}, domain.ErrPostingAccountNotPostable
		}
		if catAcc.Type != expectedCategoryType {
			return domain.CashFlowItem{}, domain.ErrAccountInvalidType
		}
		itemAccounts[i] = catAcc
	}

	// 5. Account currency match
	if paymentAcc.Currency != "" && paymentAcc.Currency != curr {
		return domain.CashFlowItem{}, domain.ErrPostingCurrencyMismatch
	}
	for _, acc := range itemAccounts {
		if acc.Currency != "" && acc.Currency != curr {
			return domain.CashFlowItem{}, domain.ErrPostingCurrencyMismatch
		}
	}

	// 6. Rounding: compute line base amounts and totals
	rate, err := s.resolveRate(c, curr, req.ExchangeRate, tDate)
	if err != nil {
		return domain.CashFlowItem{}, err
	}

	itemParams := make([]repository.CashFlowItemParams, len(req.Items))
	totalAmount := decimal.Zero
	totalBaseAmount := decimal.Zero

	for i, it := range req.Items {
		lineAmount := it.Quantity.Mul(it.UnitPrice).Round(currDecimals)
		lineBaseAmount := lineAmount.Mul(rate).Round(int32(baseDecimals))
		itemParams[i] = repository.CashFlowItemParams{
			AccountID:   it.AccountID,
			Description: it.Description,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			Amount:      lineAmount,
			BaseAmount:  lineBaseAmount,
		}
		totalAmount = totalAmount.Add(lineAmount)
		totalBaseAmount = totalBaseAmount.Add(lineBaseAmount)
	}

	// 7. Status
	status := req.Status
	if status == "" {
		status = domain.EntryStatusPosted
	}
	postedBy := base.UserIDFromContext(c)

	// 8. Update check
	if id != 0 {
		existing, err := s.repo.GetByID(c, flowType, id)
		if err != nil {
			return domain.CashFlowItem{}, err
		}
		if existing.EntryStatus != domain.EntryStatusDraft {
			return domain.CashFlowItem{}, domain.ErrEntryNotDraft
		}
	}

	// 9. Attachments validation
	var validatedFiles []coredomain.File
	if len(req.AttachmentFileIDs) > 0 {
		if s.fileSvc == nil {
			return domain.CashFlowItem{}, base.ErrUnknown
		}
		validatedFiles = make([]coredomain.File, 0, len(req.AttachmentFileIDs))
		for _, fileID := range req.AttachmentFileIDs {
			f, err := s.fileSvc.GetByID(c, fileID)
			if err != nil {
				return domain.CashFlowItem{}, err
			}
			validatedFiles = append(validatedFiles, f)
		}
	}

	params := repository.CreateCashFlowParams{
		FlowType:         flowType,
		PeriodID:         period.ID,
		PaymentAccountID: paymentAcc.ID,
		Items:            itemParams,
		TotalAmount:      totalAmount,
		TotalBaseAmount:  totalBaseAmount,
		Status:           status,
		TransactionDate:  tDate,
		Description:      req.Description,
		Reference:        req.Reference,
		Currency:         curr,
		ExchangeRate:     rate,
		PostedBy:         postedBy,
	}

	var item domain.CashFlowItem
	if id == 0 {
		item, err = s.repo.Create(c, params)
	} else {
		item, err = s.repo.Update(c, id, params)
	}
	if err != nil {
		return domain.CashFlowItem{}, err
	}

	if s.fileSvc != nil && len(validatedFiles) > 0 {
		for _, f := range validatedFiles {
			_, err := s.fileSvc.Update(c, f.ID, coremodel.UpdateFileRequest{
				Name:       f.Name,
				EntityType: domain.TransactionEntityType,
				EntityID:   item.ID,
			})
			if err != nil {
				return domain.CashFlowItem{}, err
			}
		}
	}

	items := []domain.CashFlowItem{item}
	s.enrichAttachments(c, items)
	return items[0], nil
}

// CreateExpense records an expense cash flow (draft or posted).
func (s CashFlowService) CreateExpense(c context.Context, req model.SaveCashFlowRequest) (domain.CashFlowItem, error) {
	return s.saveCashFlow(c, domain.CashFlowTypeExpense, 0, req)
}

// CreateIncome records an income cash flow (draft or posted).
func (s CashFlowService) CreateIncome(c context.Context, req model.SaveCashFlowRequest) (domain.CashFlowItem, error) {
	return s.saveCashFlow(c, domain.CashFlowTypeIncome, 0, req)
}

// UpdateExpense updates an existing draft expense cash flow.
func (s CashFlowService) UpdateExpense(c context.Context, id int64, req model.SaveCashFlowRequest) (domain.CashFlowItem, error) {
	return s.saveCashFlow(c, domain.CashFlowTypeExpense, id, req)
}

// UpdateIncome updates an existing draft income cash flow.
func (s CashFlowService) UpdateIncome(c context.Context, id int64, req model.SaveCashFlowRequest) (domain.CashFlowItem, error) {
	return s.saveCashFlow(c, domain.CashFlowTypeIncome, id, req)
}

// GetExpense retrieves an expense cash flow document by ID.
func (s CashFlowService) GetExpense(c context.Context, id int64) (domain.CashFlowItem, error) {
	item, err := s.repo.GetByID(c, domain.CashFlowTypeExpense, id)
	if err != nil {
		return domain.CashFlowItem{}, err
	}
	items := []domain.CashFlowItem{item}
	s.enrichAttachments(c, items)
	return items[0], nil
}

// GetIncome retrieves an income cash flow document by ID.
func (s CashFlowService) GetIncome(c context.Context, id int64) (domain.CashFlowItem, error) {
	item, err := s.repo.GetByID(c, domain.CashFlowTypeIncome, id)
	if err != nil {
		return domain.CashFlowItem{}, err
	}
	items := []domain.CashFlowItem{item}
	s.enrichAttachments(c, items)
	return items[0], nil
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

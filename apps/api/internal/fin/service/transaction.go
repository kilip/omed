// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/base"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
)

// TransactionRepository defines persistence operations for transactions.
type TransactionRepository interface {
	GetByID(c context.Context, id int64) (domain.Transaction, error)
	Create(c context.Context, req model.CreateTransactionRequest) (domain.Transaction, error)
	Update(c context.Context, id int64, req model.UpdateTransactionRequest) (domain.Transaction, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchTransactionRequest) ([]domain.Transaction, string, error)
}

// FileServiceContract defines the file operations required by TransactionService from the core module.
type FileServiceContract interface {
	Upload(c context.Context, req coremodel.UploadFileRequest) (coredomain.File, error)
	GetByID(c context.Context, id int64) (coredomain.File, error)
	Search(c context.Context, req coremodel.SearchFileRequest) ([]coredomain.File, string, error)
	Delete(c context.Context, id int64) error
}

// TransactionAttachmentConfig configures size and file type restrictions for transaction attachments.
type TransactionAttachmentConfig struct {
	MaxSizeBytes int64
	AllowedTypes []string
}

// TransactionServiceOption configures optional dependencies on TransactionService.
type TransactionServiceOption func(*TransactionService)

// WithTransactionAttachments wires FileServiceContract and attachment constraints into TransactionService.
func WithTransactionAttachments(fileSvc FileServiceContract, cfg TransactionAttachmentConfig) TransactionServiceOption {
	return func(s *TransactionService) {
		s.fileSvc = fileSvc
		s.attachmentCfg = cfg
	}
}

// TransactionService manages business logic and validation for financial transactions.
type TransactionService struct {
	repo          TransactionRepository
	periodRepo    PeriodRepository
	accountRepo   AccountRepository
	rateRepo      ExchangeRateRepository
	currencyRepo  CurrencyRepository
	fileSvc       FileServiceContract
	attachmentCfg TransactionAttachmentConfig
}

// NewTransactionService constructs a new TransactionService backed by the given repositories.
func NewTransactionService(
	repo TransactionRepository,
	periodRepo PeriodRepository,
	accountRepo AccountRepository,
	rateRepo ExchangeRateRepository,
	currencyRepo CurrencyRepository,
	opts ...TransactionServiceOption,
) TransactionService {
	svc := TransactionService{
		repo:         repo,
		periodRepo:   periodRepo,
		accountRepo:  accountRepo,
		rateRepo:     rateRepo,
		currencyRepo: currencyRepo,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&svc)
		}
	}
	return svc
}

// GetByID retrieves a transaction by ID along with its entries and attachments.
func (s TransactionService) GetByID(c context.Context, id int64) (domain.Transaction, error) {
	tx, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Transaction{}, err
	}
	if s.fileSvc != nil {
		files, _, fErr := s.fileSvc.Search(c, coremodel.SearchFileRequest{
			EntityType: domain.TransactionEntityType,
			EntityID:   id,
			Limit:      100,
		})
		if fErr != nil {
			return domain.Transaction{}, fErr
		}
		tx.Attachments = files
	}
	return tx, nil
}

func (s TransactionService) resolveRate(c context.Context, currency string, explicitRate *float64, date time.Time) (float64, error) {
	if explicitRate != nil && *explicitRate > 0 {
		return *explicitRate, nil
	}
	baseCurrency := "IDR"
	if s.currencyRepo != nil {
		wsID := base.WorkspaceIDFromContext(c)
		if wsID > 0 {
			if bc, err := s.currencyRepo.GetBaseCurrency(c, wsID); err == nil && bc != "" {
				baseCurrency = bc
			}
		}
	}
	if currency == "" || currency == baseCurrency {
		return 1.0, nil
	}
	if s.rateRepo == nil {
		return 1.0, nil
	}
	return s.rateRepo.Lookup(c, currency, baseCurrency, date)
}

// Create validates and creates a new transaction.
// If req carries an inline entry, entry validation is executed prior to transaction creation.
func (s TransactionService) Create(c context.Context, req model.CreateTransactionRequest) (domain.Transaction, error) {
	curr := "IDR"
	if req.Currency != "" {
		curr = req.Currency
	}

	rate, err := s.resolveRate(c, curr, req.ExchangeRate, req.TransactionDate.Time())
	if err != nil {
		return domain.Transaction{}, err
	}
	req.ExchangeRate = &rate

	if req.Entry != nil {
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

		balancedPostings, err := validateEntry(
			c,
			s.periodRepo,
			s.accountRepo,
			req.Entry.PeriodID,
			req.Entry.EntryDate.Time(),
			req.Entry.Postings,
			curr,
			rate,
			baseDecimals,
		)
		if err != nil {
			return domain.Transaction{}, err
		}
		req.Entry.Postings = balancedPostings
	}

	return s.repo.Create(c, req)
}

// Update updates transaction header fields (description, reference).
func (s TransactionService) Update(c context.Context, id int64, req model.UpdateTransactionRequest) (domain.Transaction, error) {
	return s.repo.Update(c, id, req)
}

// Delete soft-deletes a transaction if it has no posted entries, and deletes its linked attachments.
func (s TransactionService) Delete(c context.Context, id int64) error {
	if err := s.repo.Delete(c, id); err != nil {
		return err
	}
	if s.fileSvc != nil {
		files, _, err := s.fileSvc.Search(c, coremodel.SearchFileRequest{
			EntityType: domain.TransactionEntityType,
			EntityID:   id,
			Limit:      100,
		})
		if err == nil {
			for _, f := range files {
				_ = s.fileSvc.Delete(c, f.ID)
			}
		}
	}
	return nil
}

// UploadAttachment validates and uploads a file attachment for a draft transaction.
func (s TransactionService) UploadAttachment(c context.Context, id int64, req model.UploadTransactionAttachmentRequest) (coredomain.File, error) {
	if s.fileSvc == nil {
		return coredomain.File{}, base.ErrUnknown
	}
	if req.Reader == nil {
		return coredomain.File{}, coredomain.ErrFileRequired
	}
	if req.Size <= 0 {
		return coredomain.File{}, coredomain.ErrFileEmpty
	}

	tx, err := s.repo.GetByID(c, id)
	if err != nil {
		return coredomain.File{}, err
	}
	if hasPostedEntries(tx) {
		return coredomain.File{}, domain.ErrTransactionHasPostedEntries
	}

	if req.Size > s.maxAttachmentSize() {
		return coredomain.File{}, domain.ErrAttachmentTooLarge
	}

	name := strings.TrimSpace(req.Name)
	if !s.isAllowedAttachmentType(name) {
		return coredomain.File{}, domain.ErrAttachmentInvalidType
	}

	return s.fileSvc.Upload(c, coremodel.UploadFileRequest{
		WorkspaceID: req.WorkspaceID,
		Name:        name,
		MimeType:    resolveAttachmentMimeType(name, req.MimeType),
		Size:        req.Size,
		EntityType:  domain.TransactionEntityType,
		EntityID:    id,
		Reader:      req.Reader,
	})
}

// DeleteAttachment removes a file attachment from a draft transaction.
func (s TransactionService) DeleteAttachment(c context.Context, id int64, fileID int64) error {
	if s.fileSvc == nil {
		return domain.ErrAttachmentNotFound
	}

	tx, err := s.repo.GetByID(c, id)
	if err != nil {
		return err
	}
	if hasPostedEntries(tx) {
		return domain.ErrTransactionHasPostedEntries
	}

	f, err := s.fileSvc.GetByID(c, fileID)
	if err != nil {
		if errors.Is(err, base.ErrNotFound) {
			return domain.ErrAttachmentNotFound
		}
		return err
	}
	if f.EntityType != domain.TransactionEntityType || f.EntityID != id {
		return domain.ErrAttachmentNotFound
	}

	return s.fileSvc.Delete(c, fileID)
}

func (s TransactionService) maxAttachmentSize() int64 {
	if s.attachmentCfg.MaxSizeBytes > 0 {
		return s.attachmentCfg.MaxSizeBytes
	}
	return domain.DefaultMaxTransactionAttachmentSize
}

func (s TransactionService) isAllowedAttachmentType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if ext == "" {
		return false
	}
	allowed := s.attachmentCfg.AllowedTypes
	if len(allowed) == 0 {
		allowed = domain.DefaultAllowedAttachmentTypes
	}
	for _, a := range allowed {
		normalized := strings.ToLower(strings.TrimSpace(a))
		if normalized == "" {
			continue
		}
		if !strings.HasPrefix(normalized, ".") {
			normalized = "." + normalized
		}
		if ext == normalized {
			return true
		}
	}
	return false
}

func hasPostedEntries(tx domain.Transaction) bool {
	for _, e := range tx.Entries {
		if e.Status == domain.EntryStatusPosted {
			return true
		}
	}
	return false
}

func resolveAttachmentMimeType(filename, rawMime string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".csv":
		return "text/csv"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	trimmed := strings.TrimSpace(rawMime)
	if trimmed != "" {
		return trimmed
	}
	return "application/octet-stream"
}

// Search searches active transactions matching filter criteria.
func (s TransactionService) Search(c context.Context, req model.SearchTransactionRequest) ([]domain.Transaction, string, error) {
	return s.repo.Search(c, req)
}

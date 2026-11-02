package service

import (
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/shopspring/decimal"
)

// Warning codes emitted during financial suggestion validation.
const (
	WarningPaymentAccountUnresolved = "PAYMENT_ACCOUNT_UNRESOLVED"
	WarningCategoryUnresolved       = "CATEGORY_UNRESOLVED"
	WarningItemInvalid              = "ITEM_INVALID"
	WarningItemsTruncated           = "ITEMS_TRUNCATED"
)

// AccountOption represents a selectable account candidate.
type AccountOption struct {
	ID   int64
	Code string
	Name string
}

// FinanceSnapshot holds reference accounts and currencies for validating model suggestions.
type FinanceSnapshot struct {
	PaymentAccounts  []AccountOption
	CategoryAccounts []AccountOption // already filtered by flow
	Currencies       []string        // ISO codes in the reference list
	BaseCurrency     string
}

type rawItem struct {
	AccountID   *int64 `json:"accountId"`
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	UnitPrice   string `json:"unitPrice"`
}

type rawSuggestion struct {
	DocumentType     string    `json:"documentType"`
	TransactionDate  *string   `json:"transactionDate"`
	PaymentAccountID *int64    `json:"paymentAccountId"`
	Currency         *string   `json:"currency"`
	ExchangeRate     *string   `json:"exchangeRate"`
	Reference        *string   `json:"reference"`
	Description      *string   `json:"description"`
	DocumentTotal    *string   `json:"documentTotal"`
	Items            []rawItem `json:"items"`
}

func truncateRunes(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) > maxRunes {
		return string(r[:maxRunes])
	}
	return s
}

// validateSuggestion validates and normalizes an untrusted LLM raw suggestion against a FinanceSnapshot.
// It never returns an error: invalid or out-of-range values are dropped and reported through warnings.
func validateSuggestion(raw rawSuggestion, snap FinanceSnapshot, now time.Time) model.FinancialSuggestion {
	docType := strings.ToLower(strings.TrimSpace(raw.DocumentType))
	switch docType {
	case "invoice", "receipt", "payment_proof", "other":
		// valid document type
	default:
		docType = "other"
	}

	res := model.FinancialSuggestion{
		DocumentType: docType,
		Items:        make([]model.SuggestedItem, 0),
		Warnings:     make([]model.FinancialWarning, 0),
	}

	if raw.PaymentAccountID != nil {
		found := false
		for _, acc := range snap.PaymentAccounts {
			if acc.ID == *raw.PaymentAccountID {
				found = true
				break
			}
		}
		if found {
			res.PaymentAccountID = raw.PaymentAccountID
		} else {
			res.PaymentAccountID = nil
			res.Warnings = append(res.Warnings, model.FinancialWarning{
				Code: WarningPaymentAccountUnresolved,
			})
		}
	}

	if raw.Currency != nil {
		cStr := strings.TrimSpace(*raw.Currency)
		for _, c := range snap.Currencies {
			if strings.EqualFold(cStr, c) {
				val := c
				res.Currency = &val
				break
			}
		}
	}

	if res.Currency != nil && !strings.EqualFold(*res.Currency, snap.BaseCurrency) {
		if raw.ExchangeRate != nil {
			rateStr := strings.TrimSpace(*raw.ExchangeRate)
			if rate, err := decimal.NewFromString(rateStr); err == nil && rate.GreaterThan(decimal.Zero) {
				rateNorm := rate.String()
				res.ExchangeRate = &rateNorm
			}
		}
	}

	if raw.TransactionDate != nil {
		dStr := strings.TrimSpace(*raw.TransactionDate)
		if parsed, err := time.Parse("2006-01-02", dStr); err == nil {
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			minDate := today.AddDate(-5, 0, 0)
			maxDate := today.AddDate(1, 0, 0)
			if !parsed.Before(minDate) && !parsed.After(maxDate) {
				formatted := parsed.Format("2006-01-02")
				res.TransactionDate = &formatted
			}
		}
	}

	if raw.Reference != nil {
		ref := truncateRunes(strings.TrimSpace(*raw.Reference), 100)
		res.Reference = &ref
	}

	if raw.Description != nil {
		desc := truncateRunes(strings.TrimSpace(*raw.Description), 500)
		res.Description = &desc
	}

	if raw.DocumentTotal != nil {
		s := strings.TrimSpace(*raw.DocumentTotal)
		if total, err := decimal.NewFromString(s); err == nil {
			norm := total.String()
			res.DocumentTotal = &norm
		}
	}

	if docType == "payment_proof" {
		return res
	}

	itemsToProcess := raw.Items
	if len(itemsToProcess) > 100 {
		itemsToProcess = itemsToProcess[:100]
		res.Warnings = append(res.Warnings, model.FinancialWarning{
			Code: WarningItemsTruncated,
		})
	}

	for _, rawItem := range itemsToProcess {
		qtyStr := strings.TrimSpace(rawItem.Quantity)
		priceStr := strings.TrimSpace(rawItem.UnitPrice)
		qty, errQ := decimal.NewFromString(qtyStr)
		price, errP := decimal.NewFromString(priceStr)

		if errQ != nil || errP != nil || !qty.GreaterThan(decimal.Zero) || price.LessThan(decimal.Zero) {
			res.Warnings = append(res.Warnings, model.FinancialWarning{
				Code: WarningItemInvalid,
			})
			continue
		}

		itemIdx := len(res.Items)
		var accID *int64
		if rawItem.AccountID != nil {
			found := false
			for _, cat := range snap.CategoryAccounts {
				if cat.ID == *rawItem.AccountID {
					found = true
					break
				}
			}
			if found {
				accID = rawItem.AccountID
			} else {
				res.Warnings = append(res.Warnings, model.FinancialWarning{
					Code:      WarningCategoryUnresolved,
					ItemIndex: &itemIdx,
				})
			}
		}

		desc := truncateRunes(strings.TrimSpace(rawItem.Description), 500)
		res.Items = append(res.Items, model.SuggestedItem{
			AccountID:   accID,
			Description: desc,
			Quantity:    qty.String(),
			UnitPrice:   price.String(),
		})
	}

	return res
}

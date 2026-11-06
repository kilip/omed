package service

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/shopspring/decimal"
)

var (
	reCanonicalDecimal      = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)
	rePureInteger           = regexp.MustCompile(`^[0-9]+$`)
	reLeadingZeroComma      = regexp.MustCompile(`^0,[0-9]+$`)
	reSingleCommaDecimal    = regexp.MustCompile(`^[0-9]+,[0-9]+$`)
	reBothUS                = regexp.MustCompile(`^[1-9][0-9]{0,2}(,[0-9]{3})+(\.[0-9]+)?$`)
	reBothEU                = regexp.MustCompile(`^[1-9][0-9]{0,2}(\.[0-9]{3})+(,[0-9]+)?$`)
	reMultiDotInteger       = regexp.MustCompile(`^[1-9][0-9]{0,2}(\.[0-9]{3}){2,}$`)
	reMultiCommaInteger     = regexp.MustCompile(`^[1-9][0-9]{0,2}(,[0-9]{3}){2,}$`)
	reSingleDotThreeDigit   = regexp.MustCompile(`^[1-9][0-9]{0,2}\.[0-9]{3}$`)
	reSingleCommaThreeDigit = regexp.MustCompile(`^[1-9][0-9]{0,2},[0-9]{3}$`)

	reMultiplierPrefix = regexp.MustCompile(`^(?:[xX×*@]\s+)+`)
	reMultiplierSuffix = regexp.MustCompile(`(?:\s+[xX×*@])+$`)
	reQtyWithSpace     = regexp.MustCompile(`^([+-]?[0-9]+(?:[.,][0-9]+)?)\s+(.+)$`)
	reQtyAttached      = regexp.MustCompile(`^([+-]?[0-9]+(?:[.,][0-9]+)?)\s*([a-zA-Z×*@].*)$`)
	reMultiplierOnly   = regexp.MustCompile(`^[xX×*@]\s*([+-]?[0-9]+(?:[.,][0-9]+)?)$`)
)

// Warning codes emitted during financial suggestion validation.
const (
	WarningPaymentAccountUnresolved = "PAYMENT_ACCOUNT_UNRESOLVED"
	WarningCategoryUnresolved       = "CATEGORY_UNRESOLVED"
	WarningItemInvalid              = "ITEM_INVALID"
	WarningItemsTruncated           = "ITEMS_TRUNCATED"
	WarningDateOutsideOpenPeriod    = "DATE_OUTSIDE_OPEN_PERIOD"
)

// AccountOption represents a selectable account candidate.
type AccountOption struct {
	ID   int64
	Code string
	Name string
}

// PeriodRange represents an accounting period date range.
type PeriodRange struct {
	StartDate time.Time
	EndDate   time.Time
}

// FinanceSnapshot holds reference accounts and currencies for validating model suggestions.
type FinanceSnapshot struct {
	PaymentAccounts  []AccountOption
	CategoryAccounts []AccountOption // already filtered by flow
	Currencies       []string        // ISO codes in the reference list
	BaseCurrency     string
	OpenPeriods      []PeriodRange
	HasPeriodContext bool
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

	isZeroDec := false
	if res.Currency != nil && isZeroDecimalCurrency(*res.Currency) {
		isZeroDec = true
	} else if isZeroDecimalCurrency(snap.BaseCurrency) {
		isZeroDec = true
	}

	if res.Currency != nil && !strings.EqualFold(*res.Currency, snap.BaseCurrency) {
		if raw.ExchangeRate != nil {
			if rate, err := parseDecimalAmount(*raw.ExchangeRate, false, false); err == nil && rate.GreaterThan(decimal.Zero) {
				rateNorm := rate.String()
				res.ExchangeRate = &rateNorm
			}
		}
	}

	if raw.TransactionDate != nil {
		dStr := strings.TrimSpace(*raw.TransactionDate)
		if parsed, err := time.Parse("2006-01-02", dStr); err == nil {
			if snap.HasPeriodContext {
				parsedDate := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
				inPeriod := false
				for _, p := range snap.OpenPeriods {
					sDate := time.Date(p.StartDate.Year(), p.StartDate.Month(), p.StartDate.Day(), 0, 0, 0, 0, time.UTC)
					eDate := time.Date(p.EndDate.Year(), p.EndDate.Month(), p.EndDate.Day(), 0, 0, 0, 0, time.UTC)
					if !parsedDate.Before(sDate) && !parsedDate.After(eDate) {
						inPeriod = true
						break
					}
				}
				if inPeriod {
					formatted := parsed.Format("2006-01-02")
					res.TransactionDate = &formatted
				} else {
					res.TransactionDate = nil
					res.Warnings = append(res.Warnings, model.FinancialWarning{
						Code: WarningDateOutsideOpenPeriod,
					})
				}
			} else {
				today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
				minDate := today.AddDate(-5, 0, 0)
				maxDate := today.AddDate(1, 0, 0)
				if !parsed.Before(minDate) && !parsed.After(maxDate) {
					formatted := parsed.Format("2006-01-02")
					res.TransactionDate = &formatted
				}
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
		if total, err := parseDecimalAmount(*raw.DocumentTotal, true, isZeroDec); err == nil && total.GreaterThanOrEqual(decimal.Zero) {
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
		qty, updatedDesc, okQ := parseQuantityAndQualifier(rawItem.Quantity, rawItem.Description)
		price, errP := parseDecimalAmount(rawItem.UnitPrice, true, isZeroDec)

		if !okQ || errP != nil || !qty.GreaterThan(decimal.Zero) || price.LessThan(decimal.Zero) {
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

		desc := truncateRunes(strings.TrimSpace(updatedDesc), 500)
		res.Items = append(res.Items, model.SuggestedItem{
			AccountID:   accID,
			Description: desc,
			Quantity:    qty.String(),
			UnitPrice:   price.String(),
		})
	}

	return res
}

func isZeroDecimalCurrency(curr string) bool {
	switch strings.ToUpper(strings.TrimSpace(curr)) {
	case "IDR", "JPY", "KRW", "VND", "CLP", "BIF", "DJF", "GNF", "KMF", "PYG", "RWF", "UGX", "VUV", "XAF", "XOF", "XPF":
		return true
	default:
		return false
	}
}

func cleanCurrencyAndSymbols(s string) (string, bool) {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	hadRupiahMarker := strings.HasPrefix(lower, "rp") ||
		strings.HasPrefix(lower, "idr") ||
		strings.HasSuffix(lower, ",-") ||
		strings.HasSuffix(lower, ",--") ||
		strings.HasSuffix(lower, ".-") ||
		strings.Contains(lower, "rp.") ||
		strings.Contains(lower, "rp ")

	// Strip leading '@'
	if strings.HasPrefix(s, "@") {
		s = strings.TrimSpace(s[1:])
	}

	// Strip currency prefixes
	prefixes := []string{
		"rp.", "rp", "idr", "us$", "s$", "sgd", "myr", "rm", "$", "€", "£", "¥",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(strings.ToLower(s), p) {
			s = strings.TrimSpace(s[len(p):])
			s = strings.TrimPrefix(s, ".")
			s = strings.TrimSpace(s)
			break
		}
	}

	// Strip trailing suffixes
	suffixes := []string{",--", ",-", ".-", "idr", "usd", "eur"}
	for _, sf := range suffixes {
		if strings.HasSuffix(strings.ToLower(s), sf) {
			s = strings.TrimSpace(s[:len(s)-len(sf)])
			break
		}
	}

	return s, hadRupiahMarker
}

func parseDecimalAmount(s string, isMonetaryAmount bool, isZeroDecimalCurrency bool) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 64 {
		return decimal.Zero, errors.New("invalid amount length")
	}
	if strings.ContainsAny(s, "eE") {
		return decimal.Zero, errors.New("scientific notation not allowed")
	}

	isNegative := false
	if strings.HasPrefix(s, "-") {
		isNegative = true
		s = strings.TrimSpace(s[1:])
	}

	s, hadRupiahMarker := cleanCurrencyAndSymbols(s)

	if strings.HasPrefix(s, "-") {
		isNegative = true
		s = strings.TrimSpace(s[1:])
	}

	if s == "" {
		return decimal.Zero, errors.New("empty numeric string")
	}

	treatThreeDigitAsThousand := hadRupiahMarker || (isMonetaryAmount && isZeroDecimalCurrency)

	var norm string
	switch {
	case reBothUS.MatchString(s):
		norm = strings.ReplaceAll(s, ",", "")
	case reBothEU.MatchString(s):
		noDot := strings.ReplaceAll(s, ".", "")
		norm = strings.ReplaceAll(noDot, ",", ".")
	case strings.Contains(s, ",") && strings.Contains(s, "."):
		return decimal.Zero, errors.New("invalid mixed separators")
	case strings.Contains(s, ","):
		switch {
		case reMultiCommaInteger.MatchString(s):
			norm = strings.ReplaceAll(s, ",", "")
		case reSingleCommaThreeDigit.MatchString(s):
			if treatThreeDigitAsThousand || isMonetaryAmount {
				norm = strings.ReplaceAll(s, ",", "")
			} else {
				norm = strings.ReplaceAll(s, ",", ".")
			}
		case reLeadingZeroComma.MatchString(s) || reSingleCommaDecimal.MatchString(s):
			norm = strings.ReplaceAll(s, ",", ".")
		default:
			return decimal.Zero, errors.New("invalid comma format")
		}
	case strings.Contains(s, "."):
		switch {
		case reMultiDotInteger.MatchString(s):
			norm = strings.ReplaceAll(s, ".", "")
		case reSingleDotThreeDigit.MatchString(s):
			if treatThreeDigitAsThousand {
				norm = strings.ReplaceAll(s, ".", "")
			} else {
				norm = s
			}
		case reCanonicalDecimal.MatchString(s):
			norm = s
		default:
			return decimal.Zero, errors.New("invalid dot format")
		}
	default:
		if rePureInteger.MatchString(s) {
			norm = s
		} else {
			return decimal.Zero, errors.New("invalid integer format")
		}
	}

	if isNegative {
		norm = "-" + norm
	}

	return decimal.NewFromString(norm)
}

func parseQuantityAndQualifier(rawQty string, desc string) (decimal.Decimal, string, bool) {
	rawQty = strings.TrimSpace(rawQty)
	if rawQty == "" || len(rawQty) > 64 {
		return decimal.Zero, desc, false
	}
	if strings.ContainsAny(rawQty, "eE") {
		return decimal.Zero, desc, false
	}

	// 1. Try direct numeric parse
	if q, err := parseDecimalAmount(rawQty, false, false); err == nil {
		if q.GreaterThan(decimal.Zero) {
			return q, desc, true
		}
		return decimal.Zero, desc, false
	}

	// 2. Try multiplier prefix, e.g. "x 1" or "x1"
	if m := reMultiplierOnly.FindStringSubmatch(rawQty); len(m) == 2 {
		if q, err := parseDecimalAmount(m[1], false, false); err == nil && q.GreaterThan(decimal.Zero) {
			return q, desc, true
		}
		return decimal.Zero, desc, false
	}

	// 3. Try number with space or attached qualifier
	var numStr, remStr string
	if m := reQtyWithSpace.FindStringSubmatch(rawQty); len(m) == 3 {
		numStr = m[1]
		remStr = m[2]
	} else if m := reQtyAttached.FindStringSubmatch(rawQty); len(m) == 3 {
		numStr = m[1]
		remStr = m[2]
	} else {
		return decimal.Zero, desc, false
	}

	q, err := parseDecimalAmount(numStr, false, false)
	if err != nil || !q.GreaterThan(decimal.Zero) {
		return decimal.Zero, desc, false
	}

	remStr = strings.TrimSpace(remStr)
	remStr = reMultiplierPrefix.ReplaceAllString(remStr, "")
	remStr = reMultiplierSuffix.ReplaceAllString(remStr, "")
	remStr = strings.TrimSpace(remStr)

	if remStr == "x" || remStr == "X" || remStr == "×" || remStr == "*" || remStr == "@" {
		remStr = ""
	}

	hasLetters := strings.IndexFunc(remStr, unicode.IsLetter) >= 0
	if remStr != "" && !hasLetters {
		return decimal.Zero, desc, false
	}

	if isTrivialCountUnit(remStr) {
		remStr = ""
	}

	updatedDesc := applyQualifierToDescription(desc, remStr)
	return q, updatedDesc, true
}

func isTrivialCountUnit(u string) bool {
	switch strings.ToLower(strings.TrimSpace(u)) {
	case "pcs", "pc", "pce", "bh", "buah", "ea", "unit", "item", "items":
		return true
	default:
		return false
	}
}

func applyQualifierToDescription(desc string, qualifier string) string {
	qualifier = strings.TrimSpace(qualifier)
	if qualifier == "" {
		return desc
	}

	if strings.Contains(strings.ToLower(desc), strings.ToLower(qualifier)) {
		return desc
	}

	desc = strings.TrimSpace(desc)
	if desc == "" {
		return qualifier
	}

	if unicode.IsDigit(rune(qualifier[0])) {
		return desc + " " + qualifier
	}
	return desc + " (" + qualifier + ")"
}

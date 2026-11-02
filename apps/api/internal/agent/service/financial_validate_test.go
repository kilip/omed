package service

import (
	"strings"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

func defaultSnapshot() FinanceSnapshot {
	return FinanceSnapshot{
		PaymentAccounts: []AccountOption{
			{ID: 10, Code: "1001", Name: "Cash"},
			{ID: 20, Code: "1002", Name: "Bank BCA"},
		},
		CategoryAccounts: []AccountOption{
			{ID: 100, Code: "5001", Name: "Office Supplies"},
			{ID: 200, Code: "5002", Name: "Software Subscription"},
		},
		Currencies:   []string{"IDR", "USD", "EUR"},
		BaseCurrency: "IDR",
	}
}

func TestValidateSuggestion(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	snap := defaultSnapshot()

	tests := []struct {
		name     string
		raw      rawSuggestion
		snap     FinanceSnapshot
		now      time.Time
		expected model.FinancialSuggestion
	}{
		{
			name: "payment account in set kept",
			raw: rawSuggestion{
				DocumentType:     "receipt",
				PaymentAccountID: ptr(int64(10)),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:     "receipt",
				PaymentAccountID: ptr(int64(10)),
				Items:            []model.SuggestedItem{},
				Warnings:         []model.FinancialWarning{},
			},
		},
		{
			name: "payment account not in set dropped with PAYMENT_ACCOUNT_UNRESOLVED",
			raw: rawSuggestion{
				DocumentType:     "receipt",
				PaymentAccountID: ptr(int64(999)),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:     "receipt",
				PaymentAccountID: nil,
				Items:            []model.SuggestedItem{},
				Warnings: []model.FinancialWarning{
					{Code: WarningPaymentAccountUnresolved},
				},
			},
		},
		{
			name: "payment account nil stays nil without warning",
			raw: rawSuggestion{
				DocumentType:     "receipt",
				PaymentAccountID: nil,
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:     "receipt",
				PaymentAccountID: nil,
				Items:            []model.SuggestedItem{},
				Warnings:         []model.FinancialWarning{},
			},
		},
		{
			name: "item accountId in set kept",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						AccountID:   ptr(int64(100)),
						Description: "Office paper",
						Quantity:    "5",
						UnitPrice:   "10000.00",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items: []model.SuggestedItem{
					{
						AccountID:   ptr(int64(100)),
						Description: "Office paper",
						Quantity:    "5",
						UnitPrice:   "10000",
					},
				},
				Warnings: []model.FinancialWarning{},
			},
		},
		{
			name: "item accountId not in set kept item with nil AccountID and CATEGORY_UNRESOLVED with item index",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						AccountID:   ptr(int64(999)),
						Description: "Office paper",
						Quantity:    "5",
						UnitPrice:   "10000",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items: []model.SuggestedItem{
					{
						AccountID:   nil,
						Description: "Office paper",
						Quantity:    "5",
						UnitPrice:   "10000",
					},
				},
				Warnings: []model.FinancialWarning{
					{
						Code:      WarningCategoryUnresolved,
						ItemIndex: ptr(0),
					},
				},
			},
		},
		{
			name: "item accountId nil kept item with nil AccountID without warning",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						AccountID:   nil,
						Description: "Office paper",
						Quantity:    "5",
						UnitPrice:   "10000",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items: []model.SuggestedItem{
					{
						AccountID:   nil,
						Description: "Office paper",
						Quantity:    "5",
						UnitPrice:   "10000",
					},
				},
				Warnings: []model.FinancialWarning{},
			},
		},
		{
			name: "currency in Currencies kept",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "currency unknown dropped",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("JPY"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     nil,
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "exchangeRate dropped when currency is missing",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     nil,
				ExchangeRate: ptr("15500.00"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     nil,
				ExchangeRate: nil,
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "exchangeRate dropped when currency was unknown",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("JPY"),
				ExchangeRate: ptr("100.00"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     nil,
				ExchangeRate: nil,
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "exchangeRate dropped when currency equals BaseCurrency",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("IDR"),
				ExchangeRate: ptr("1.0"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("IDR"),
				ExchangeRate: nil,
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "exchangeRate kept when positive and currency differs",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				ExchangeRate: ptr("15500.50"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				ExchangeRate: ptr("15500.5"),
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "exchangeRate dropped when zero",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				ExchangeRate: ptr("0"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				ExchangeRate: nil,
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "exchangeRate dropped when negative",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				ExchangeRate: ptr("-15000"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				ExchangeRate: nil,
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "exchangeRate dropped when non-numeric",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				ExchangeRate: ptr("not-a-number"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Currency:     ptr("USD"),
				ExchangeRate: nil,
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "transactionDate exactly 5 years back accepted",
			raw: rawSuggestion{
				DocumentType:    "receipt",
				TransactionDate: ptr("2021-11-02"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:    "receipt",
				TransactionDate: ptr("2021-11-02"),
				Items:           []model.SuggestedItem{},
				Warnings:        []model.FinancialWarning{},
			},
		},
		{
			name: "transactionDate one day beyond 5 years back dropped",
			raw: rawSuggestion{
				DocumentType:    "receipt",
				TransactionDate: ptr("2021-11-01"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:    "receipt",
				TransactionDate: nil,
				Items:           []model.SuggestedItem{},
				Warnings:        []model.FinancialWarning{},
			},
		},
		{
			name: "transactionDate exactly 1 year ahead accepted",
			raw: rawSuggestion{
				DocumentType:    "receipt",
				TransactionDate: ptr("2027-11-02"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:    "receipt",
				TransactionDate: ptr("2027-11-02"),
				Items:           []model.SuggestedItem{},
				Warnings:        []model.FinancialWarning{},
			},
		},
		{
			name: "transactionDate one day beyond 1 year ahead dropped",
			raw: rawSuggestion{
				DocumentType:    "receipt",
				TransactionDate: ptr("2027-11-03"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:    "receipt",
				TransactionDate: nil,
				Items:           []model.SuggestedItem{},
				Warnings:        []model.FinancialWarning{},
			},
		},
		{
			name: "transactionDate unparsable dropped",
			raw: rawSuggestion{
				DocumentType:    "receipt",
				TransactionDate: ptr("2026/11/02"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:    "receipt",
				TransactionDate: nil,
				Items:           []model.SuggestedItem{},
				Warnings:        []model.FinancialWarning{},
			},
		},
		{
			name: "quantity 0 dropped with ITEM_INVALID",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						Description: "Zero qty item",
						Quantity:    "0",
						UnitPrice:   "1000",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items:        []model.SuggestedItem{},
				Warnings: []model.FinancialWarning{
					{Code: WarningItemInvalid},
				},
			},
		},
		{
			name: "quantity negative dropped with ITEM_INVALID",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						Description: "Negative qty item",
						Quantity:    "-1",
						UnitPrice:   "1000",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items:        []model.SuggestedItem{},
				Warnings: []model.FinancialWarning{
					{Code: WarningItemInvalid},
				},
			},
		},
		{
			name: "quantity non-numeric dropped with ITEM_INVALID",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						Description: "Bad qty item",
						Quantity:    "invalid",
						UnitPrice:   "1000",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items:        []model.SuggestedItem{},
				Warnings: []model.FinancialWarning{
					{Code: WarningItemInvalid},
				},
			},
		},
		{
			name: "unitPrice negative dropped with ITEM_INVALID",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						Description: "Negative price item",
						Quantity:    "1",
						UnitPrice:   "-10",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items:        []model.SuggestedItem{},
				Warnings: []model.FinancialWarning{
					{Code: WarningItemInvalid},
				},
			},
		},
		{
			name: "unitPrice non-numeric dropped with ITEM_INVALID",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						Description: "Bad price item",
						Quantity:    "1",
						UnitPrice:   "free",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items:        []model.SuggestedItem{},
				Warnings: []model.FinancialWarning{
					{Code: WarningItemInvalid},
				},
			},
		},
		{
			name: "unitPrice 0 accepted",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						Description: "Free promo item",
						Quantity:    "1",
						UnitPrice:   "0",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items: []model.SuggestedItem{
					{
						Description: "Free promo item",
						Quantity:    "1",
						UnitPrice:   "0",
					},
				},
				Warnings: []model.FinancialWarning{},
			},
		},
		{
			name: "reference 150 chars truncated to 100 with whitespace trimmed",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Reference:    ptr("   " + strings.Repeat("A", 150) + "   "),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Reference:    ptr(strings.Repeat("A", 100)),
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "description 600 chars truncated to 500 with whitespace trimmed",
			raw: rawSuggestion{
				DocumentType: "receipt",
				Description:  ptr("   " + strings.Repeat("B", 600) + "   "),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "receipt",
				Description:  ptr(strings.Repeat("B", 500)),
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "item description 600 chars truncated to 500 with whitespace trimmed",
			raw: rawSuggestion{
				DocumentType: "invoice",
				Items: []rawItem{
					{
						Description: "   " + strings.Repeat("C", 600) + "   ",
						Quantity:    "1",
						UnitPrice:   "10",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items: []model.SuggestedItem{
					{
						Description: strings.Repeat("C", 500),
						Quantity:    "1",
						UnitPrice:   "10",
					},
				},
				Warnings: []model.FinancialWarning{},
			},
		},
		{
			name: "documentType payment_proof with items returns zero items",
			raw: rawSuggestion{
				DocumentType: "payment_proof",
				Items: []rawItem{
					{
						Description: "Transfer fee",
						Quantity:    "1",
						UnitPrice:   "2500",
					},
				},
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "payment_proof",
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "unknown documentType becomes other",
			raw: rawSuggestion{
				DocumentType: "proforma_invoice",
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "other",
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "valid documentTypes preserved case-insensitively",
			raw: rawSuggestion{
				DocumentType: "INVOICE",
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType: "invoice",
				Items:        []model.SuggestedItem{},
				Warnings:     []model.FinancialWarning{},
			},
		},
		{
			name: "documentTotal valid decimal kept as normalised string",
			raw: rawSuggestion{
				DocumentType:  "receipt",
				DocumentTotal: ptr("  150000.5000  "),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:  "receipt",
				DocumentTotal: ptr("150000.5"),
				Items:         []model.SuggestedItem{},
				Warnings:      []model.FinancialWarning{},
			},
		},
		{
			name: "documentTotal invalid decimal omitted",
			raw: rawSuggestion{
				DocumentType:  "receipt",
				DocumentTotal: ptr("invalid-total"),
			},
			snap: snap,
			now:  now,
			expected: model.FinancialSuggestion{
				DocumentType:  "receipt",
				DocumentTotal: nil,
				Items:         []model.SuggestedItem{},
				Warnings:      []model.FinancialWarning{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := validateSuggestion(tc.raw, tc.snap, tc.now)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestValidateSuggestion_ItemsTruncated(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	snap := defaultSnapshot()

	// 101 items
	rawItems := make([]rawItem, 101)
	for i := 0; i < 101; i++ {
		rawItems[i] = rawItem{
			Description: "Item",
			Quantity:    "1",
			UnitPrice:   "10",
		}
	}

	raw := rawSuggestion{
		DocumentType: "invoice",
		Items:        rawItems,
	}

	got := validateSuggestion(raw, snap, now)
	require.Len(t, got.Items, 100)
	require.Len(t, got.Warnings, 1)
	assert.Equal(t, WarningItemsTruncated, got.Warnings[0].Code)
	assert.Nil(t, got.Warnings[0].ItemIndex)
}

func TestValidateSuggestion_RuneTruncation(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	snap := defaultSnapshot()

	// Multi-byte Unicode characters (e.g. Japanese Kanji/Kana: 3 bytes per rune)
	japanese150 := strings.Repeat("日", 150)
	japanese600 := strings.Repeat("本", 600)

	raw := rawSuggestion{
		DocumentType: "invoice",
		Reference:    ptr("  " + japanese150 + "  "),
		Description:  ptr("  " + japanese600 + "  "),
		Items: []rawItem{
			{
				Description: "  " + japanese600 + "  ",
				Quantity:    "1",
				UnitPrice:   "100",
			},
		},
	}

	got := validateSuggestion(raw, snap, now)

	require.NotNil(t, got.Reference)
	assert.Equal(t, 100, len([]rune(*got.Reference)))
	assert.Equal(t, strings.Repeat("日", 100), *got.Reference)

	require.NotNil(t, got.Description)
	assert.Equal(t, 500, len([]rune(*got.Description)))
	assert.Equal(t, strings.Repeat("本", 500), *got.Description)

	require.Len(t, got.Items, 1)
	assert.Equal(t, 500, len([]rune(got.Items[0].Description)))
	assert.Equal(t, strings.Repeat("本", 500), got.Items[0].Description)
}

func TestValidateSuggestion_MultipleWarningsAndItemIndex(t *testing.T) {
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	snap := defaultSnapshot()

	raw := rawSuggestion{
		DocumentType:     "invoice",
		PaymentAccountID: ptr(int64(999)), // invalid -> PAYMENT_ACCOUNT_UNRESOLVED
		Items: []rawItem{
			{
				AccountID:   ptr(int64(100)), // valid category
				Description: "Item 0",
				Quantity:    "1",
				UnitPrice:   "10",
			},
			{
				AccountID:   ptr(int64(200)),
				Description: "Item 1 invalid qty",
				Quantity:    "-1", // invalid qty -> ITEM_INVALID (dropped)
				UnitPrice:   "10",
			},
			{
				AccountID:   ptr(int64(888)), // invalid category -> CATEGORY_UNRESOLVED
				Description: "Item 2",
				Quantity:    "2",
				UnitPrice:   "20",
			},
		},
	}

	got := validateSuggestion(raw, snap, now)

	assert.Nil(t, got.PaymentAccountID)
	// Item 1 was dropped, so got.Items has Item 0 at index 0 and Item 2 at index 1
	require.Len(t, got.Items, 2)
	assert.Equal(t, "Item 0", got.Items[0].Description)
	assert.Equal(t, ptr(int64(100)), got.Items[0].AccountID)
	assert.Equal(t, "Item 2", got.Items[1].Description)
	assert.Nil(t, got.Items[1].AccountID)

	require.Len(t, got.Warnings, 3)
	assert.Equal(t, WarningPaymentAccountUnresolved, got.Warnings[0].Code)
	assert.Nil(t, got.Warnings[0].ItemIndex)

	assert.Equal(t, WarningItemInvalid, got.Warnings[1].Code)
	assert.Nil(t, got.Warnings[1].ItemIndex)

	assert.Equal(t, WarningCategoryUnresolved, got.Warnings[2].Code)
	require.NotNil(t, got.Warnings[2].ItemIndex)
	assert.Equal(t, 1, *got.Warnings[2].ItemIndex) // Index in got.Items is 1!
}

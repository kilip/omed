// Package seed provides chart of accounts template loading and workspace seeding capabilities.
package seed

import (
	"context"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/currency"
)

// DefaultCurrency defines a predefined currency for initial seeding.
type DefaultCurrency struct {
	Code     string
	Name     string
	Symbol   string
	Decimals int
}

// DefaultCurrencies contains the initial list of ISO 4217 currencies to seed.
var DefaultCurrencies = []DefaultCurrency{
	{Code: "IDR", Name: "Indonesian Rupiah", Symbol: "Rp", Decimals: 0},
	{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2},
	{Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Decimals: 0},
	{Code: "KWD", Name: "Kuwaiti Dinar", Symbol: "KD", Decimals: 3},
	{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2},
	{Code: "GBP", Name: "British Pound", Symbol: "£", Decimals: 2},
	{Code: "SGD", Name: "Singapore Dollar", Symbol: "S$", Decimals: 2},
}

// SeedCurrencies populates the global Currency table with default currencies if they do not exist.
// This function is idempotent: existing currency codes are skipped.
func SeedCurrencies(ctx context.Context, client *ent.Client) error {
	for _, c := range DefaultCurrencies {
		exists, err := client.Currency.Query().
			Where(currency.CodeEQ(c.Code)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		if err := client.Currency.Create().
			SetCode(c.Code).
			SetName(c.Name).
			SetSymbol(c.Symbol).
			SetDecimals(c.Decimals).
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

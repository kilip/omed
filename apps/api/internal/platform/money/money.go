// Package money provides deterministic decimal-based financial arithmetic and rounding helpers.
package money

import (
	"github.com/shopspring/decimal"
)

// Zero is a convenience constant for decimal.Zero.
var Zero = decimal.Zero

// Round rounds v to the specified number of decimal places using half-away-from-zero rounding.
// Negative decimal places are clamped to zero.
func Round(v decimal.Decimal, decimals int32) decimal.Decimal {
	if decimals < 0 {
		decimals = 0
	}
	return v.Round(decimals)
}

// Equal compares two monetary amounts after rounding each to the given decimal places.
func Equal(a, b decimal.Decimal, decimals int32) bool {
	return Round(a, decimals).Equal(Round(b, decimals))
}

// Convert multiplies an amount by an exchange rate and rounds the result to the given decimals.
func Convert(amount, rate decimal.Decimal, decimals int32) decimal.Decimal {
	return Round(amount.Mul(rate), decimals)
}

// FromString parses a string representation into a decimal.Decimal.
func FromString(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

// RequireFromString parses a string into a decimal.Decimal, panicking if invalid.
func RequireFromString(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}

// FromFloat creates a decimal.Decimal from a float64.
func FromFloat(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

// FromInt creates a decimal.Decimal from an int64.
func FromInt(i int64) decimal.Decimal {
	return decimal.NewFromInt(i)
}

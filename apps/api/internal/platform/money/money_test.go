package money_test

import (
	"math"
	"testing"

	"github.com/kilip/omed/api/internal/platform/money"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRound(t *testing.T) {
	tests := []struct {
		name     string
		val      decimal.Decimal
		decimals int32
		expected string
	}{
		// Decimal 0 (e.g. IDR, JPY)
		{name: "0 dec round down", val: decimal.RequireFromString("100.4"), decimals: 0, expected: "100"},
		{name: "0 dec half-way round up", val: decimal.RequireFromString("100.5"), decimals: 0, expected: "101"},
		{name: "0 dec negative half-way", val: decimal.RequireFromString("-100.5"), decimals: 0, expected: "-101"},

		// Decimal 2 (e.g. USD, EUR, SGD)
		{name: "2 dec normal", val: decimal.RequireFromString("12.344"), decimals: 2, expected: "12.34"},
		{name: "2 dec half-way up", val: decimal.RequireFromString("12.345"), decimals: 2, expected: "12.35"},
		{name: "2 dec float inaccuracy trap 1.005", val: decimal.RequireFromString("1.005"), decimals: 2, expected: "1.01"},
		{name: "2 dec float inaccuracy trap 2.005", val: decimal.RequireFromString("2.005"), decimals: 2, expected: "2.01"},
		{name: "2 dec float inaccuracy trap 3.005", val: decimal.RequireFromString("3.005"), decimals: 2, expected: "3.01"},
		{name: "2 dec negative half-way", val: decimal.RequireFromString("-12.345"), decimals: 2, expected: "-12.35"},

		// Decimal 3 (e.g. KWD)
		{name: "3 dec normal", val: decimal.RequireFromString("1.2344"), decimals: 3, expected: "1.234"},
		{name: "3 dec half-way up", val: decimal.RequireFromString("1.2345"), decimals: 3, expected: "1.235"},
		{name: "3 dec float inaccuracy trap 1.0005", val: decimal.RequireFromString("1.0005"), decimals: 3, expected: "1.001"},
		{name: "3 dec negative half-way", val: decimal.RequireFromString("-1.2345"), decimals: 3, expected: "-1.235"},

		// Decimal 4
		{name: "4 dec normal", val: decimal.RequireFromString("1.23454"), decimals: 4, expected: "1.2345"},
		{name: "4 dec half-way up", val: decimal.RequireFromString("1.23455"), decimals: 4, expected: "1.2346"},
		{name: "4 dec negative half-way", val: decimal.RequireFromString("-1.23455"), decimals: 4, expected: "-1.2346"},

		// Negative zero cases & small negative rounding
		{name: "negative zero from float", val: decimal.NewFromFloat(math.Copysign(0, -1)), decimals: 2, expected: "0"},
		{name: "small negative rounding to zero 2 dec", val: decimal.RequireFromString("-0.001"), decimals: 2, expected: "0"},
		{name: "small negative rounding to zero 0 dec", val: decimal.RequireFromString("-0.4"), decimals: 0, expected: "0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := money.Round(tc.val, tc.decimals)
			exp := decimal.RequireFromString(tc.expected)
			require.True(t, exp.Equal(got), "expected %s, got %s", exp, got)
			if got.IsZero() {
				require.Equal(t, "0", got.String(), "expected zero string representation to be 0")
			}
		})
	}
}

func TestFloatTrap(t *testing.T) {
	// In standard IEEE-754 float64, 0.1 + 0.2 = 0.30000000000000004 != 0.3
	d1 := decimal.RequireFromString("0.1")
	d2 := decimal.RequireFromString("0.2")
	sum := d1.Add(d2)
	expected := decimal.RequireFromString("0.3")
	require.True(t, sum.Equal(expected), "expected 0.1 + 0.2 == 0.3, got %s", sum)

	// Multiplication trap: 19.99 * 100 in float can be 1998.9999999999998
	price := decimal.RequireFromString("19.99")
	qty := decimal.NewFromInt(100)
	total := price.Mul(qty)
	require.True(t, total.Equal(decimal.RequireFromString("1999")), "expected 1999, got %s", total)
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        decimal.Decimal
		b        decimal.Decimal
		decimals int32
		expected bool
	}{
		{name: "exact equal", a: decimal.RequireFromString("100.50"), b: decimal.RequireFromString("100.50"), decimals: 2, expected: true},
		{name: "equal after round", a: decimal.RequireFromString("100.504"), b: decimal.RequireFromString("100.496"), decimals: 2, expected: true},
		{name: "not equal", a: decimal.RequireFromString("100.50"), b: decimal.RequireFromString("100.51"), decimals: 2, expected: false},
		{name: "zero and negative zero", a: decimal.Zero, b: decimal.NewFromFloat(math.Copysign(0, -1)), decimals: 2, expected: true},
		{name: "3 decimals equal", a: decimal.RequireFromString("1.2341"), b: decimal.RequireFromString("1.2344"), decimals: 3, expected: true},
		{name: "3 decimals not equal", a: decimal.RequireFromString("1.2341"), b: decimal.RequireFromString("1.2346"), decimals: 3, expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := money.Equal(tc.a, tc.b, tc.decimals)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestConvert(t *testing.T) {
	tests := []struct {
		name     string
		amount   decimal.Decimal
		rate     decimal.Decimal
		decimals int32
		expected string
	}{
		{
			name:     "USD to IDR",
			amount:   decimal.RequireFromString("100.50"),
			rate:     decimal.RequireFromString("16250.0"),
			decimals: 0,
			expected: "1633125",
		},
		{
			name:     "IDR to USD with 8 decimal rate",
			amount:   decimal.RequireFromString("1633125"),
			rate:     decimal.RequireFromString("0.000061538461"), // approx 1/16250
			decimals: 2,
			expected: "100.50",
		},
		{
			name:     "KWD convert",
			amount:   decimal.RequireFromString("10.5"),
			rate:     decimal.RequireFromString("3.25"),
			decimals: 3,
			expected: "34.125",
		},
		{
			name:     "zero amount",
			amount:   decimal.Zero,
			rate:     decimal.RequireFromString("15000"),
			decimals: 2,
			expected: "0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := money.Convert(tc.amount, tc.rate, tc.decimals)
			exp := decimal.RequireFromString(tc.expected)
			require.True(t, exp.Equal(got), "expected %s, got %s", exp, got)
		})
	}
}

func TestHelpers(t *testing.T) {
	d, err := money.FromString("123.45")
	require.NoError(t, err)
	require.True(t, d.Equal(decimal.RequireFromString("123.45")))

	require.True(t, money.RequireFromString("123.45").Equal(d))
	require.True(t, money.FromFloat(123.45).Round(2).Equal(d))
	require.True(t, money.FromInt(100).Equal(decimal.NewFromInt(100)))
	require.True(t, money.Zero.IsZero())
}

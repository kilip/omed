package money_test

import (
	"math"
	"testing"

	"github.com/kilip/omed/api/internal/platform/money"
	"github.com/stretchr/testify/require"
)

func TestRound(t *testing.T) {
	tests := []struct {
		name     string
		val      float64
		decimals int
		expected float64
	}{
		// Decimal 0 (e.g. IDR, JPY)
		{name: "0 dec round down", val: 100.4, decimals: 0, expected: 100},
		{name: "0 dec half-way round up", val: 100.5, decimals: 0, expected: 101},
		{name: "0 dec negative half-way", val: -100.5, decimals: 0, expected: -101},

		// Decimal 2 (e.g. USD, EUR, SGD)
		{name: "2 dec normal", val: 12.344, decimals: 2, expected: 12.34},
		{name: "2 dec half-way up", val: 12.345, decimals: 2, expected: 12.35},
		{name: "2 dec float inaccuracy 1.005", val: 1.005, decimals: 2, expected: 1.01},
		{name: "2 dec float inaccuracy 2.005", val: 2.005, decimals: 2, expected: 2.01},
		{name: "2 dec float inaccuracy 3.005", val: 3.005, decimals: 2, expected: 3.01},
		{name: "2 dec negative half-way", val: -12.345, decimals: 2, expected: -12.35},

		// Decimal 3 (e.g. KWD)
		{name: "3 dec normal", val: 1.2344, decimals: 3, expected: 1.234},
		{name: "3 dec half-way up", val: 1.2345, decimals: 3, expected: 1.235},
		{name: "3 dec float inaccuracy 1.0005", val: 1.0005, decimals: 3, expected: 1.001},
		{name: "3 dec negative half-way", val: -1.2345, decimals: 3, expected: -1.235},

		// Decimal 4
		{name: "4 dec normal", val: 1.23454, decimals: 4, expected: 1.2345},
		{name: "4 dec half-way up", val: 1.23455, decimals: 4, expected: 1.2346},
		{name: "4 dec negative half-way", val: -1.23455, decimals: 4, expected: -1.2346},

		// Negative zero cases
		{name: "negative zero input", val: math.Copysign(0, -1), decimals: 2, expected: 0.0},
		{name: "small negative rounding to zero", val: -0.001, decimals: 2, expected: 0.0},
		{name: "small negative rounding to zero 0 dec", val: -0.4, decimals: 0, expected: 0.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := money.Round(tc.val, tc.decimals)
			require.Equal(t, tc.expected, got)
			if got == 0.0 {
				require.False(t, math.Signbit(got), "expected positive zero, got negative zero")
			}
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        float64
		b        float64
		decimals int
		expected bool
	}{
		{name: "exact equal", a: 100.50, b: 100.50, decimals: 2, expected: true},
		{name: "equal after round", a: 100.504, b: 100.496, decimals: 2, expected: true},
		{name: "not equal", a: 100.50, b: 100.51, decimals: 2, expected: false},
		{name: "zero and negative zero", a: 0.0, b: math.Copysign(0, -1), decimals: 2, expected: true},
		{name: "3 decimals equal", a: 1.2341, b: 1.2344, decimals: 3, expected: true},
		{name: "3 decimals not equal", a: 1.2341, b: 1.2346, decimals: 3, expected: false},
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
		amount   float64
		rate     float64
		decimals int
		expected float64
	}{
		{name: "USD to IDR", amount: 100.50, rate: 16250.0, decimals: 0, expected: 1633125},
		{name: "IDR to USD", amount: 1633125, rate: 1.0 / 16250.0, decimals: 2, expected: 100.50},
		{name: "KWD convert", amount: 10.5, rate: 3.25, decimals: 3, expected: 34.125},
		{name: "zero amount", amount: 0, rate: 15000, decimals: 2, expected: 0.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := money.Convert(tc.amount, tc.rate, tc.decimals)
			require.Equal(t, tc.expected, got)
			if got == 0.0 {
				require.False(t, math.Signbit(got))
			}
		})
	}
}

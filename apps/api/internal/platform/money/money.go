// Package money provides deterministic floating-point financial arithmetic and rounding helpers.
package money

import "math"

// Round rounds v to the specified number of decimal places.
// It handles IEEE-754 floating-point inaccuracies around half-way values and eliminates negative zero (-0.0 -> 0.0).
func Round(v float64, decimals int) float64 {
	if decimals < 0 {
		decimals = 0
	}
	pow := math.Pow10(decimals)
	var res float64
	if v >= 0 {
		res = math.Floor(v*pow+0.5+1e-12) / pow
	} else {
		res = math.Ceil(v*pow-0.5-1e-12) / pow
	}
	if res == 0 {
		return 0.0
	}
	return res
}

// Equal compares two monetary amounts after rounding each to the given decimal places.
func Equal(a, b float64, decimals int) bool {
	return Round(a, decimals) == Round(b, decimals)
}

// Convert multiplies an amount by an exchange rate and rounds the result to the given decimals.
func Convert(amount, rate float64, decimals int) float64 {
	return Round(amount*rate, decimals)
}

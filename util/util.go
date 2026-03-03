// Package util provides shared utility functions used across the
// skill-validator codebase: number formatting, pluralization, rounding,
// sorted-key extraction, and ANSI color helpers.
package util

import (
	"fmt"
	"math"
	"sort"
)

// --- Color constants for terminal output ---

// ANSI escape codes for terminal coloring.
const (
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

// --- Number formatting ---

// FormatNumber formats an integer with thousand-separator commas.
func FormatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// RoundTo rounds val to the given number of decimal places.
func RoundTo(val float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(val*pow) / pow
}

// --- Pluralization ---

// PluralS returns "s" when n != 1, empty string otherwise.
func PluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// YSuffix returns "y" when n == 1, "ies" otherwise.
func YSuffix(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// --- Map helpers ---

// SortedKeys returns the keys of any map[string]V sorted alphabetically.
func SortedKeys[V any](m map[string]V) []string {
	if len(m) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

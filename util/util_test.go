package util

import (
	"math"
	"testing"
)

func TestRoundTo(t *testing.T) {
	tests := []struct {
		val    float64
		places int
		want   float64
	}{
		{0.12345, 4, 0.1235},
		{0.5, 2, 0.5},
		{1.0, 4, 1.0},
		{0.0, 4, 0.0},
	}
	for _, tt := range tests {
		got := RoundTo(tt.val, tt.places)
		if math.Abs(got-tt.want) > 1e-10 {
			t.Errorf("RoundTo(%f, %d) = %f, want %f", tt.val, tt.places, got, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1000000, "1,000,000"},
	}
	for _, tt := range tests {
		got := FormatNumber(tt.n)
		if got != tt.want {
			t.Errorf("FormatNumber(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestPluralS(t *testing.T) {
	if PluralS(1) != "" {
		t.Error("PluralS(1) should be empty")
	}
	if PluralS(0) != "s" {
		t.Error("PluralS(0) should be 's'")
	}
	if PluralS(2) != "s" {
		t.Error("PluralS(2) should be 's'")
	}
}

func TestYSuffix(t *testing.T) {
	if YSuffix(1) != "y" {
		t.Error("YSuffix(1) should be 'y'")
	}
	if YSuffix(2) != "ies" {
		t.Error("YSuffix(2) should be 'ies'")
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]bool{"banana": true, "apple": true, "cherry": true}
	got := SortedKeys(m)
	want := []string{"apple", "banana", "cherry"}
	if len(got) != len(want) {
		t.Fatalf("SortedKeys: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SortedKeys[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	// Empty map
	empty := SortedKeys(map[string]int{})
	if len(empty) != 0 {
		t.Errorf("SortedKeys(empty) = %v, want []", empty)
	}
}

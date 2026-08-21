package filedialog

import (
	"slices"
	"testing"
)

// The panels themselves are modal UI and need a display plus the main thread;
// examples/filedialog is their manual vehicle. What is tested here is the pure
// option-normalization logic every backend shares.

func TestCleanExtensions(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string // nil means "no restriction"
	}{
		{"empty", nil, nil},
		{"plain", []string{"png", "jpg"}, []string{"png", "jpg"}},
		{"leading dots stripped", []string{".afoil", ".dat"}, []string{"afoil", "dat"}},
		{"wildcard star disables", []string{"png", "*"}, nil},
		{"wildcard empty disables", []string{"png", ""}, nil},
		{"lone dot disables", []string{"."}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := cleanExtensions(c.in)
			if !slices.Equal(got, c.want) {
				t.Fatalf("cleanExtensions(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

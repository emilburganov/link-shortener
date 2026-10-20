package link

import (
	"strings"
	"testing"
)

func TestGenerateCode(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"empty", 0},
		{"one char", 1},
		{"typical length", 7},
		{"long", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := GenerateCode(tt.n)

			if err != nil {
				t.Fatalf("GenerateCode(%d) error: %v", tt.n, err)
			}

			if len(code) != tt.n {
				t.Errorf("len = %d, want %d", len(code), tt.n)
			}

			for _, r := range code {
				if !strings.ContainsRune(alphabet, r) {
					t.Errorf("code %q has %q outside the alphabet", code, r)
				}
			}
		})
	}
}

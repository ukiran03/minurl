package data

import (
	"testing"
)

func TestBase62RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input flake
	}{
		{"Zero ID", 0},
		{"Small ID", 61},
		{"Boundary ID", 62},
		{"Typical Snowflake ID", 1234567890123456},
		{"Max Int64", 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode it
			encoded := tt.input.Base62()

			// Decode it back
			decoded, err := ParseBase62(encoded)
			if err != nil {
				t.Fatalf("Failed to decode valid base62 string %q: %v", encoded, err)
			}

			if decoded != tt.input {
				t.Errorf(
					"Roundtrip failed! Expected %d, got %d (encoded as %q)",
					tt.input,
					decoded,
					encoded,
				)
			}
		})
	}
}

func TestParseBase62_Validation(t *testing.T) {
	invalidTests := []struct {
		name  string
		input string
	}{
		{"Empty string", ""},
		{"Too long", "ZZZZZZZZZZZZ"}, // 12 chars
		{"Invalid characters", "abc-123"},
		{"Spaces", "a b c"},
		{"Overflow MaxInt64", "ZZZZZZZZZZZ"}, // 11 chars, but wraps around past MaxInt64
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBase62(tt.input)
			if err == nil {
				t.Errorf("Expected error for invalid input %q, but got none", tt.input)
			}
		})
	}
}

func BenchmarkBase62Encode(b *testing.B) {
	f := flake(1234567890123456)
	for b.Loop() {
		_ = f.Base62()
	}
}

func BenchmarkBase62Decode(b *testing.B) {
	s := "1abcXYZ99"
	for b.Loop() {
		_, _ = ParseBase62(s)
	}
}

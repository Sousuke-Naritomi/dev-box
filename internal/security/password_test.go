package security

import (
	"strings"
	"testing"
)

func TestGeneratePassword_Length(t *testing.T) {
	tests := []int{4, 8, 16, 32, 64}
	for _, length := range tests {
		pw, err := GeneratePassword(length)
		if err != nil {
			t.Fatalf("length %d: unexpected error: %v", length, err)
		}
		if len(pw) != length {
			t.Errorf("expected password length %d, got %d", length, len(pw))
		}
	}
}

func TestGeneratePassword_TooShort(t *testing.T) {
	_, err := GeneratePassword(3)
	if err == nil {
		t.Error("expected error for length < 4")
	}
}

func TestGeneratePassword_ContainsAllCharClasses(t *testing.T) {
	// Run multiple times to account for randomness
	for i := 0; i < 20; i++ {
		pw, err := GeneratePassword(16)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasUpper := strings.ContainsAny(pw, uppercase)
		hasLower := strings.ContainsAny(pw, lowercase)
		hasDigit := strings.ContainsAny(pw, digits)
		hasSymbol := strings.ContainsAny(pw, symbols)

		if !hasUpper || !hasLower || !hasDigit || !hasSymbol {
			t.Errorf("password %q missing character class (upper=%v lower=%v digit=%v symbol=%v)",
				pw, hasUpper, hasLower, hasDigit, hasSymbol)
		}
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	passwords := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pw, err := GeneratePassword(20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if passwords[pw] {
			t.Errorf("duplicate password generated: %q", pw)
		}
		passwords[pw] = true
	}
}

func TestGeneratePassword_NoHardcodedFallback(t *testing.T) {
	for i := 0; i < 50; i++ {
		pw, err := GeneratePassword(20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(pw, "fallback") || strings.Contains(pw, "Secure_Password") {
			t.Errorf("password appears to be a hardcoded fallback: %q", pw)
		}
	}
}

package config

import "testing"

func TestValidateRejectsEmptyStack(t *testing.T) {
	cfg := ProjectConfig{
		Frontend: FrontendNone,
		Backend:  BackendNone,
		Database: DatabaseNone,
		Cache:    CacheNone,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected empty stack to be rejected")
	}
}

func TestValidateAllowsOptionalServiceOnly(t *testing.T) {
	cfg := ProjectConfig{
		Frontend:   FrontendNone,
		Backend:    BackendNone,
		Database:   DatabaseNone,
		Cache:      CacheNone,
		LocalStack: true,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected LocalStack-only configuration to be valid: %v", err)
	}
}

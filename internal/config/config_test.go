package config

import "testing"

func TestBackendInternalPort(t *testing.T) {
	tests := []struct {
		backend string
		want    int
	}{
		{"node", 3000},
		{"python", 8000},
		{"go", 8080},
		{"php", 80},
		{"java", 8080},
		{"none", 0},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		cfg := ProjectConfig{Backend: tt.backend}
		if got := cfg.BackendInternalPort(); got != tt.want {
			t.Errorf("BackendInternalPort(%q) = %d, want %d", tt.backend, got, tt.want)
		}
	}
}

package config

import "testing"

func TestNewDefaultsBaseURLAndFormat(t *testing.T) {
	t.Setenv("TWENTY_API_KEY", "env-key")
	t.Setenv("TWENTY_BASE_URL", "")

	cfg, err := New("", "", "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cfg.APIKey != "env-key" {
		t.Fatalf("APIKey = %q, want env-key", cfg.APIKey)
	}

	if cfg.BaseURL != defaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, defaultBaseURL)
	}

	if cfg.Format != "json" {
		t.Fatalf("Format = %q, want json", cfg.Format)
	}
}

func TestValidateAuthRequiresAPIKey(t *testing.T) {
	cfg, err := New("", "https://api.twenty.com", "json")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := cfg.ValidateAuth(); err == nil {
		t.Fatal("ValidateAuth() expected error, got nil")
	}
}

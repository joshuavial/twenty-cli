package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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

func TestNewLoadsSettingsFromCurrentDirectoryBeforeHome(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	writeSettings(t, filepath.Join(homeDir, ".twenty", "settings"), `{"api_key":"home-key","base_url":"https://home.example.com"}`)
	writeSettings(t, filepath.Join(workDir, ".twenty", "settings"), `{"api_key":"cwd-key","base_url":"https://cwd.example.com"}`)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()

	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	cfg, err := New("", "", "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cfg.APIKey != "cwd-key" {
		t.Fatalf("APIKey = %q, want cwd-key", cfg.APIKey)
	}

	if cfg.BaseURL != "https://cwd.example.com" {
		t.Fatalf("BaseURL = %q, want cwd settings value", cfg.BaseURL)
	}
}

func TestNewFallsBackToHomeSettings(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	writeSettings(t, filepath.Join(homeDir, ".twenty", "settings"), `{"api_key":"home-key","base_url":"https://home.example.com"}`)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()

	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	cfg, err := New("", "", "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cfg.APIKey != "home-key" {
		t.Fatalf("APIKey = %q, want home-key", cfg.APIKey)
	}

	if cfg.BaseURL != "https://home.example.com" {
		t.Fatalf("BaseURL = %q, want home settings value", cfg.BaseURL)
	}
}

func TestNewPrefersEnvAndFlagsOverSettings(t *testing.T) {
	homeDir := t.TempDir()
	workDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("TWENTY_API_KEY", "env-key")
	t.Setenv("TWENTY_BASE_URL", "https://env.example.com")

	writeSettings(t, filepath.Join(workDir, ".twenty", "settings"), `{"api_key":"cwd-key","base_url":"https://cwd.example.com"}`)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(previousWD)
	}()

	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	cfg, err := New("flag-key", "https://flag.example.com", "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if cfg.APIKey != "flag-key" {
		t.Fatalf("APIKey = %q, want flag-key", cfg.APIKey)
	}

	if cfg.BaseURL != "https://flag.example.com" {
		t.Fatalf("BaseURL = %q, want flag value", cfg.BaseURL)
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

func TestWriteSettingsHomeScope(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	path, err := WriteSettings(SettingsScopeHome, Config{
		APIKey:  "secret",
		BaseURL: "https://api.twenty.com",
	}, false)
	if err != nil {
		t.Fatalf("WriteSettings() error = %v", err)
	}

	wantPath := filepath.Join(homeDir, ".twenty", "settings")
	if path != wantPath {
		t.Fatalf("path = %q, want %q", path, wantPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "{\n  \"api_key\": \"secret\",\n  \"base_url\": \"https://api.twenty.com\"\n}\n" {
		t.Fatalf("settings = %q", string(data))
	}
}

func TestWriteSettingsRefusesOverwriteByDefault(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	writeSettings(t, filepath.Join(homeDir, ".twenty", "settings"), `{"api_key":"old","base_url":"https://old.example.com"}`)

	_, err := WriteSettings(SettingsScopeHome, Config{
		APIKey:  "secret",
		BaseURL: "https://api.twenty.com",
	}, false)
	if err == nil {
		t.Fatal("WriteSettings() expected error, got nil")
	}

	var settingsErr *SettingsError
	if !errors.As(err, &settingsErr) {
		t.Fatalf("err = %T, want *SettingsError", err)
	}
}

func TestWriteSettingsOverwritesWhenRequested(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	path := filepath.Join(homeDir, ".twenty", "settings")
	writeSettings(t, path, `{"api_key":"old","base_url":"https://old.example.com"}`)

	if _, err := WriteSettings(SettingsScopeHome, Config{
		APIKey:  "secret",
		BaseURL: "https://api.twenty.com",
	}, true); err != nil {
		t.Fatalf("WriteSettings() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "{\n  \"api_key\": \"secret\",\n  \"base_url\": \"https://api.twenty.com\"\n}\n" {
		t.Fatalf("settings = %q", string(data))
	}
}

func TestNewReturnsSettingsErrorForInvalidSettingsFile(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	writeSettings(t, filepath.Join(homeDir, ".twenty", "settings"), `{bad json`)

	_, err := New("", "", "")
	if err == nil {
		t.Fatal("New() expected error, got nil")
	}

	var settingsErr *SettingsError
	if !errors.As(err, &settingsErr) {
		t.Fatalf("err = %T, want *SettingsError", err)
	}
	if settingsErr.Path == "" {
		t.Fatal("settingsErr.Path = empty")
	}
}

func writeSettings(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

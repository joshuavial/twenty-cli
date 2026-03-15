package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const defaultBaseURL = "https://api.twenty.com"

type Config struct {
	APIKey  string
	BaseURL string
	Format  string
}

type SettingsError struct {
	Path string
	Op   string
	Err  error
}

func (e *SettingsError) Error() string {
	return fmt.Sprintf("settings file %s at %s: %v", e.Op, e.Path, e.Err)
}

func (e *SettingsError) Unwrap() error {
	return e.Err
}

type SettingsScope string

const (
	SettingsScopeHome    SettingsScope = "home"
	SettingsScopeProject SettingsScope = "project"
)

func New(apiKey, baseURL, format string) (Config, error) {
	settings, err := loadSettings()
	if err != nil {
		return Config{}, err
	}

	cfg := buildConfig(apiKey, baseURL, format, settings)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func NewWithoutSettings(apiKey, baseURL, format string) (Config, error) {
	cfg := buildConfig(apiKey, baseURL, format, settings{})
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func buildConfig(apiKey, baseURL, format string, settings settings) Config {
	cfg := Config{
		APIKey:  strings.TrimSpace(firstNonEmpty(apiKey, os.Getenv("TWENTY_API_KEY"), settings.APIKey)),
		BaseURL: strings.TrimSpace(firstNonEmpty(baseURL, os.Getenv("TWENTY_BASE_URL"), settings.BaseURL, defaultBaseURL)),
		Format:  strings.TrimSpace(firstNonEmpty(format, "json")),
	}

	if cfg.Format == "" {
		cfg.Format = "json"
	}

	if cfg.Format != "json" && cfg.Format != "text" {
		cfg.Format = ""
	}

	return cfg
}

func (c Config) Validate() error {
	if c.Format == "" {
		return errors.New("format must be one of: json, text")
	}

	parsed, err := url.ParseRequestURI(c.BaseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("base URL must be a valid http or https URL")
	}

	return nil
}

func (c Config) ValidateAuth() error {
	if c.APIKey == "" {
		return errors.New("missing API key; set TWENTY_API_KEY or pass --api-key")
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}

type settings struct {
	APIKey  string `json:"api_key"`
	BaseURL string `json:"base_url"`
}

func loadSettings() (settings, error) {
	for _, path := range candidateSettingsPaths() {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return settings{}, &SettingsError{Path: path, Op: "read failed", Err: err}
		}

		var cfg settings
		if err := json.Unmarshal(data, &cfg); err != nil {
			return settings{}, &SettingsError{Path: path, Op: "is invalid", Err: err}
		}

		return cfg, nil
	}

	return settings{}, nil
}

func SettingsPath(scope SettingsScope) (string, error) {
	switch scope {
	case SettingsScopeProject:
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(wd, ".twenty", "settings"), nil
	case SettingsScopeHome:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".twenty", "settings"), nil
	default:
		return "", fmt.Errorf("unknown settings scope %q", scope)
	}
}

func WriteSettings(scope SettingsScope, cfg Config, overwrite bool) (string, error) {
	path, err := SettingsPath(scope)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(path); err == nil && !overwrite {
		return "", &SettingsError{Path: path, Op: "already exists", Err: errors.New("use --overwrite to replace it")}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", &SettingsError{Path: path, Op: "read failed", Err: err}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", &SettingsError{Path: path, Op: "mkdir failed", Err: err}
	}

	payload, err := json.MarshalIndent(settings{
		APIKey:  strings.TrimSpace(cfg.APIKey),
		BaseURL: strings.TrimSpace(cfg.BaseURL),
	}, "", "  ")
	if err != nil {
		return "", err
	}
	payload = append(payload, '\n')

	tempFile, err := os.CreateTemp(filepath.Dir(path), "settings-*.tmp")
	if err != nil {
		return "", &SettingsError{Path: path, Op: "tempfile create failed", Err: err}
	}
	tempPath := tempFile.Name()
	if _, err := tempFile.Write(payload); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return "", &SettingsError{Path: tempPath, Op: "write failed", Err: err}
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", &SettingsError{Path: tempPath, Op: "close failed", Err: err}
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return "", &SettingsError{Path: path, Op: "replace failed", Err: err}
	}

	return path, nil
}

func candidateSettingsPaths() []string {
	paths := make([]string, 0, 2)

	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, ".twenty", "settings"))
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".twenty", "settings"))
	}

	return paths
}

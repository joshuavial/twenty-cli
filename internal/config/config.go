package config

import (
	"encoding/json"
	"errors"
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

func New(apiKey, baseURL, format string) (Config, error) {
	settings, err := loadSettings()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		APIKey:  strings.TrimSpace(firstNonEmpty(apiKey, os.Getenv("TWENTY_API_KEY"), settings.APIKey)),
		BaseURL: strings.TrimSpace(firstNonEmpty(baseURL, os.Getenv("TWENTY_BASE_URL"), settings.BaseURL, defaultBaseURL)),
		Format:  strings.TrimSpace(firstNonEmpty(format, "json")),
	}

	if cfg.Format == "" {
		cfg.Format = "json"
	}

	if cfg.Format != "json" && cfg.Format != "text" {
		return Config{}, errors.New("format must be one of: json, text")
	}

	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return Config{}, errors.New("base URL must be a valid absolute URL")
	}

	return cfg, nil
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
			return settings{}, err
		}

		var cfg settings
		if err := json.Unmarshal(data, &cfg); err != nil {
			return settings{}, err
		}

		return cfg, nil
	}

	return settings{}, nil
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

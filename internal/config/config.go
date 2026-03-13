package config

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

const defaultBaseURL = "https://api.twenty.com"

type Config struct {
	APIKey  string
	BaseURL string
	Format  string
}

func New(apiKey, baseURL, format string) (Config, error) {
	cfg := Config{
		APIKey:  strings.TrimSpace(firstNonEmpty(apiKey, os.Getenv("TWENTY_API_KEY"))),
		BaseURL: strings.TrimSpace(firstNonEmpty(baseURL, os.Getenv("TWENTY_BASE_URL"), defaultBaseURL)),
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

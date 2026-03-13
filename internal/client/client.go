package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/jv/twenty-crm-cli/internal/config"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient HTTPDoer
}

type AuthCheckResult struct {
	StatusCode int    `json:"status_code"`
	Endpoint   string `json:"endpoint"`
}

type APIError struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api request failed with status %d", e.StatusCode)
}

func New(cfg config.Config, httpClient HTTPDoer) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
	}
}

func (c *Client) AuthCheck(ctx context.Context) (AuthCheckResult, error) {
	endpoint := c.baseURL + "/rest/metadata"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return AuthCheckResult{}, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AuthCheckResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return AuthCheckResult{}, &APIError{
			StatusCode: resp.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}

	return AuthCheckResult{
		StatusCode: resp.StatusCode,
		Endpoint:   sanitizeEndpoint(endpoint),
	}, nil
}

func sanitizeEndpoint(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if parsed.RawQuery == "" {
		return parsed.Path
	}

	return parsed.Path + "?" + parsed.RawQuery
}

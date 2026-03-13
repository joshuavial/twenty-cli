package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jv/twenty-crm-cli/internal/config"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAuthCheckUsesExpectedEndpointAndAuthHeader(t *testing.T) {
	cfg, err := config.New("secret", "https://api.twenty.com", "json")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var gotPath string
	var gotAuth string

	cli := New(cfg, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		gotPath = req.URL.String()
		gotAuth = req.Header.Get("Authorization")

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	}))

	result, err := cli.AuthCheck(context.Background())
	if err != nil {
		t.Fatalf("AuthCheck() error = %v", err)
	}

	if gotPath != "https://api.twenty.com/rest/metadata" {
		t.Fatalf("path = %q", gotPath)
	}

	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}

	if result.Endpoint != "/rest/metadata" {
		t.Fatalf("Endpoint = %q", result.Endpoint)
	}
}

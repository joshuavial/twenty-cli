package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jv/twenty-crm-cli/internal/client"
	"github.com/jv/twenty-crm-cli/internal/config"
)

type authCheckerStub struct {
	result client.AuthCheckResult
	err    error
}

func (s authCheckerStub) AuthCheck(context.Context) (client.AuthCheckResult, error) {
	return s.result, s.err
}

func TestAuthCheckJSONSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)
	app.clientFactory = func(config.Config, client.HTTPDoer) authChecker {
		return authCheckerStub{
			result: client.AuthCheckResult{
				StatusCode: 200,
				Endpoint:   "/rest/people?limit=1&depth=0",
			},
		}
	}

	code := app.Run([]string{"--api-key", "secret", "auth", "check"})
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	if !strings.Contains(stdout.String(), `"command": "auth.check"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestAuthCheckMissingAPIKey(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)

	code := app.Run([]string{"auth", "check"})
	if code != exitAuth {
		t.Fatalf("Run() code = %d, want %d", code, exitAuth)
	}

	if !strings.Contains(stdout.String(), `"code": "auth.missing_api_key"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestParseFailureReturnsJSONWhenRequested(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)

	code := app.Run([]string{"--format", "json", "--wat"})
	if code != exitUsage {
		t.Fatalf("Run() code = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(stdout.String(), `"code": "cli.parse"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestConfigFailureReturnsJSONWhenRequested(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)

	code := app.Run([]string{"--format", "yaml", "version"})
	if code != exitUsage {
		t.Fatalf("Run() code = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(stdout.String(), `"code": "cli.parse"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestAuthCheckForbiddenReturnsPermissionError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)
	app.clientFactory = func(config.Config, client.HTTPDoer) authChecker {
		return authCheckerStub{
			err: &client.APIError{
				StatusCode: 403,
				Body:       `{"error":"forbidden"}`,
			},
		}
	}

	code := app.Run([]string{"--api-key", "secret", "auth", "check"})
	if code != exitAPI {
		t.Fatalf("Run() code = %d, want %d", code, exitAPI)
	}

	if !strings.Contains(stdout.String(), `"code": "auth.insufficient_permissions"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestAuthCheckUnauthorizedReturnsInvalidCredentials(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)
	app.clientFactory = func(config.Config, client.HTTPDoer) authChecker {
		return authCheckerStub{
			err: &client.APIError{
				StatusCode: 401,
				Body:       `{"error":"unauthorized"}`,
			},
		}
	}

	code := app.Run([]string{"--api-key", "secret", "auth", "check"})
	if code != exitAuth {
		t.Fatalf("Run() code = %d, want %d", code, exitAuth)
	}

	if !strings.Contains(stdout.String(), `"code": "auth.invalid_credentials"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestAuthCheckRequestFailureReturnsInternalError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)
	app.clientFactory = func(config.Config, client.HTTPDoer) authChecker {
		return authCheckerStub{
			err: errors.New("dial failed"),
		}
	}

	code := app.Run([]string{"--api-key", "secret", "auth", "check"})
	if code != exitInternal {
		t.Fatalf("Run() code = %d, want %d", code, exitInternal)
	}

	if !strings.Contains(stdout.String(), `"code": "auth.request_failed"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jv/twenty-crm-cli/internal/client"
	"github.com/jv/twenty-crm-cli/internal/config"
	"github.com/jv/twenty-crm-cli/internal/output"
)

type authCheckerStub struct {
	result client.AuthCheckResult
	err    error
}

func (s authCheckerStub) AuthCheck(context.Context) (client.AuthCheckResult, error) {
	return s.result, s.err
}

func decodeEnvelope(t *testing.T, data string) output.Envelope {
	t.Helper()

	var envelope output.Envelope
	if err := json.Unmarshal([]byte(data), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return envelope
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

	envelope := decodeEnvelope(t, stdout.String())
	if !envelope.OK {
		t.Fatalf("OK = false, want true")
	}
	if envelope.Command != "auth.check" {
		t.Fatalf("Command = %q, want %q", envelope.Command, "auth.check")
	}
}

func TestAuthCheckMissingAPIKey(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)

	code := app.Run([]string{"auth", "check"})
	if code != int(output.ExitAuth) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitAuth)
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "auth.missing_api_key" {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if envelope.Error.Kind != output.ErrorKindAuth {
		t.Fatalf("Error.Kind = %q, want %q", envelope.Error.Kind, output.ErrorKindAuth)
	}
}

func TestParseFailureReturnsJSONWhenRequested(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)

	code := app.Run([]string{"--format", "json", "--wat"})
	if code != int(output.ExitUsage) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitUsage)
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "cli.parse" {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if envelope.Command != "cli" {
		t.Fatalf("Command = %q, want %q", envelope.Command, "cli")
	}
}

func TestConfigFailureReturnsJSONWhenRequested(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)

	code := app.Run([]string{"--format", "yaml", "version"})
	if code != int(output.ExitUsage) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitUsage)
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "cli.parse" {
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
	if code != int(output.ExitAPI) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitAPI)
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "auth.insufficient_permissions" {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if envelope.Error.Kind != output.ErrorKindAPI {
		t.Fatalf("Error.Kind = %q, want %q", envelope.Error.Kind, output.ErrorKindAPI)
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
	if code != int(output.ExitAuth) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitAuth)
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "auth.invalid_credentials" {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if envelope.Error.Kind != output.ErrorKindAuth {
		t.Fatalf("Error.Kind = %q, want %q", envelope.Error.Kind, output.ErrorKindAuth)
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
	if code != int(output.ExitInternal) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitInternal)
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "auth.request_failed" {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if !envelope.Error.Retryable {
		t.Fatalf("Retryable = false, want true")
	}
}

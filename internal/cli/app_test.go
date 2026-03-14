package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/jv/twenty-crm-cli/internal/client"
	"github.com/jv/twenty-crm-cli/internal/config"
	"github.com/jv/twenty-crm-cli/internal/output"
)

type clientStub struct {
	result client.AuthCheckResult
	err    error
	list   client.ListResult
	record client.RecordResult
}

func (s clientStub) AuthCheck(context.Context) (client.AuthCheckResult, error) {
	return s.result, s.err
}

func (s clientStub) MetadataObjects(context.Context) ([]client.MetadataObject, error) {
	return nil, nil
}

func (s clientStub) ListRecords(context.Context, string, url.Values) (client.ListResult, error) {
	return s.list, s.err
}

func (s clientStub) GetRecord(context.Context, string, string, string, url.Values) (client.RecordResult, error) {
	return s.record, s.err
}

func (s clientStub) CreateRecord(context.Context, string, string, map[string]any) (client.RecordResult, error) {
	return s.record, s.err
}

func (s clientStub) UpdateRecord(context.Context, string, string, string, map[string]any) (client.RecordResult, error) {
	return s.record, s.err
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
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{
			result: client.AuthCheckResult{
				StatusCode: 200,
				Endpoint:   "/metadata",
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
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{
			err: &client.APIError{
				StatusCode: 403,
				Body:       `{"error":"forbidden"}`,
			},
		}
	}

	code := app.Run([]string{"--api-key", "secret", "auth", "check"})
	if code != int(output.ExitAuth) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitAuth)
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "auth.insufficient_permissions" {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if envelope.Error.Kind != output.ErrorKindAuth {
		t.Fatalf("Error.Kind = %q, want %q", envelope.Error.Kind, output.ErrorKindAuth)
	}
}

func TestAuthCheckUnauthorizedReturnsInvalidCredentials(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{
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
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{
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

func TestPersonGetSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(&stdout, &stderr)
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{
			record: client.RecordResult{
				Record: map[string]any{"id": "person_123"},
			},
		}
	}

	code := app.Run([]string{"--api-key", "secret", "person", "get", "--id", "person_123"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitOK)
	}

	envelope := decodeEnvelope(t, stdout.String())
	if !envelope.OK || envelope.Command != "person.get" {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

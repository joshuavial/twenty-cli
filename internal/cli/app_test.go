package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

	app := New(strings.NewReader(""), &stdout, &stderr)
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

	app := New(strings.NewReader(""), &stdout, &stderr)

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

	app := New(strings.NewReader(""), &stdout, &stderr)

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

func TestNoArgsPrintsTextHelpByDefault(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)

	code := app.Run([]string{})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitOK)
	}
	if !strings.Contains(stdout.String(), "Twenty CRM CLI") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "\"ok\"") {
		t.Fatalf("stdout unexpectedly looks like json: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestNoArgsWithExplicitJSONReturnsUsageEnvelope(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)

	code := app.Run([]string{"--format", "json"})
	if code != int(output.ExitUsage) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitUsage)
	}
	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "cli.usage" {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRootHelpPrintsText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)

	code := app.Run([]string{"--help"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitOK)
	}
	if !strings.Contains(stdout.String(), "Twenty CRM CLI") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestEntityGroupHelpPrintsText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)

	code := app.Run([]string{"people", "--help"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitOK)
	}
	if !strings.Contains(stdout.String(), "People commands") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "\"ok\"") {
		t.Fatalf("stdout unexpectedly looks like json: %q", stdout.String())
	}
}

func TestEntitySubcommandHelpPrintsText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)

	code := app.Run([]string{"people", "search", "--help"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitOK)
	}
	if !strings.Contains(stdout.String(), "Search people") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestWorkflowGroupHelpPrintsText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)

	code := app.Run([]string{"meeting", "--help"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitOK)
	}
	if !strings.Contains(stdout.String(), "Meeting commands") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestWorkflowSubcommandHelpPrintsText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)

	code := app.Run([]string{"prospect", "import", "--help"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitOK)
	}
	if !strings.Contains(stdout.String(), "Import prospects from JSON or JSONL") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestConfigFailureReturnsJSONWhenRequested(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)

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

	app := New(strings.NewReader(""), &stdout, &stderr)
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

	app := New(strings.NewReader(""), &stdout, &stderr)
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

	app := New(strings.NewReader(""), &stdout, &stderr)
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

	app := New(strings.NewReader(""), &stdout, &stderr)
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

func TestCompanySearchEmptyResultsReturnsArray(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{
			list: client.ListResult{
				Records: nil,
			},
		}
	}

	code := app.Run([]string{"--api-key", "secret", "companies", "search", "--query", "missing"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d", code, output.ExitOK)
	}

	var envelope struct {
		OK      bool             `json:"ok"`
		Command string           `json:"command"`
		Data    []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !envelope.OK || envelope.Command != "companies.search" {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if envelope.Data == nil {
		t.Fatalf("Data = nil, want empty array")
	}
	if len(envelope.Data) != 0 {
		t.Fatalf("len(Data) = %d, want 0", len(envelope.Data))
	}
}

func TestAuthLoginWritesHomeSettings(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	app := New(strings.NewReader(""), &stdout, &stderr)
	app.isTTY = func(io.Reader) bool { return false }
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{
			result: client.AuthCheckResult{
				StatusCode: 200,
				Endpoint:   "/metadata",
			},
		}
	}

	code := app.Run([]string{"auth", "login", "--api-key", "secret", "--base-url", "https://api.twenty.com"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d; stdout=%s stderr=%s", code, output.ExitOK, stdout.String(), stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Path  string `json:"path"`
			Scope string `json:"scope"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !envelope.OK {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if envelope.Data.Scope != "home" {
		t.Fatalf("scope = %q, want home", envelope.Data.Scope)
	}
	if !strings.HasSuffix(envelope.Data.Path, "/.twenty/settings") {
		t.Fatalf("path = %q", envelope.Data.Path)
	}
}

func TestAuthLoginPromptsOnTTY(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	app := New(strings.NewReader("secret\nhttps://api.twenty.com\n"), &stdout, &stderr)
	app.isTTY = func(io.Reader) bool { return true }
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{
			result: client.AuthCheckResult{
				StatusCode: 200,
				Endpoint:   "/metadata",
			},
		}
	}

	code := app.Run([]string{"auth", "login"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d; stdout=%s stderr=%s", code, output.ExitOK, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "API key: ") {
		t.Fatalf("stderr = %q, want prompt", stderr.String())
	}
}

func TestAuthLoginPromptsForBaseURLWhenFlagMissing(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	var captured config.Config
	app := New(strings.NewReader("https://custom.example\n"), &stdout, &stderr)
	app.isTTY = func(io.Reader) bool { return true }
	app.clientFactory = func(cfg config.Config, _ client.HTTPDoer) twentyClient {
		captured = cfg
		return clientStub{
			result: client.AuthCheckResult{
				StatusCode: 200,
				Endpoint:   "/metadata",
			},
		}
	}

	code := app.Run([]string{"auth", "login", "--api-key", "secret"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d; stdout=%s stderr=%s", code, output.ExitOK, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Base URL [https://api.twenty.com]: ") {
		t.Fatalf("stderr = %q, want base URL prompt", stderr.String())
	}
	if captured.BaseURL != "https://custom.example" {
		t.Fatalf("captured.BaseURL = %q, want https://custom.example", captured.BaseURL)
	}
	data, err := os.ReadFile(filepath.Join(homeDir, ".twenty", "settings"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "\"base_url\": \"https://custom.example\"") {
		t.Fatalf("settings = %q", string(data))
	}
}

func TestAuthLoginProjectScopeWritesProjectSettings(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	homeDir := t.TempDir()
	workDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(previousWD) }()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	app := New(strings.NewReader(""), &stdout, &stderr)
	app.isTTY = func(io.Reader) bool { return false }
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return clientStub{result: client.AuthCheckResult{StatusCode: 200, Endpoint: "/metadata"}}
	}

	code := app.Run([]string{"auth", "login", "--api-key", "secret", "--base-url", "https://api.twenty.com", "--scope", "project"})
	if code != int(output.ExitOK) {
		t.Fatalf("Run() code = %d, want %d; stdout=%s stderr=%s", code, output.ExitOK, stdout.String(), stderr.String())
	}

	path := filepath.Join(workDir, ".twenty", "settings")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "{\n  \"api_key\": \"secret\",\n  \"base_url\": \"https://api.twenty.com\"\n}\n" {
		t.Fatalf("settings = %q", string(data))
	}
}

func TestAuthLoginInvalidScopeFailsBeforeAuthCheck(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := New(strings.NewReader(""), &stdout, &stderr)
	app.isTTY = func(io.Reader) bool { return false }
	called := false
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		called = true
		return clientStub{result: client.AuthCheckResult{StatusCode: 200, Endpoint: "/metadata"}}
	}

	code := app.Run([]string{"auth", "login", "--api-key", "secret", "--base-url", "https://api.twenty.com", "--scope", "bad"})
	if code != int(output.ExitUsage) {
		t.Fatalf("Run() code = %d, want %d; stdout=%s stderr=%s", code, output.ExitUsage, stdout.String(), stderr.String())
	}
	if called {
		t.Fatal("clientFactory called, want scope validation before auth check")
	}

	envelope := decodeEnvelope(t, stdout.String())
	if envelope.Error == nil || envelope.Error.Code != "auth.login.scope" {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthCheckReadsConfigFileAndSnapshotsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer config-secret" {
			t.Fatalf("Authorization = %q, want Bearer config-secret", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"__typename":"Query"}}`))
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"auth", "check"},
		WorkingDirFiles: map[string]string{
			".twenty/settings": `{"api_key":"config-secret","base_url":"` + server.URL + `"}`,
		},
	})

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	harness.AssertJSONSnapshot("auth_check_config_success", result)
}

func TestAuthCheckReadsEnvConfigAndSnapshotsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer env-secret" {
			t.Fatalf("Authorization = %q, want Bearer env-secret", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"__typename":"Query"}}`))
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"auth", "check"},
		Env: map[string]string{
			"TWENTY_API_KEY":  "env-secret",
			"TWENTY_BASE_URL": server.URL,
		},
	})

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	harness.AssertJSONSnapshot("auth_check_env_success", result)
}

func TestParseErrorMatchesSnapshot(t *testing.T) {
	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"--format", "json", "--wat"},
	})

	if result.ExitCode != 2 {
		t.Fatalf("ExitCode = %d, want 2", result.ExitCode)
	}

	harness.AssertJSONSnapshot("parse_error_unknown_flag", result)
}

func TestConfigErrorMatchesSnapshot(t *testing.T) {
	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"--base-url", "not-a-url", "version"},
	})

	if result.ExitCode != 2 {
		t.Fatalf("ExitCode = %d, want 2", result.ExitCode)
	}

	harness.AssertJSONSnapshot("config_error_invalid_base_url", result)
}

func TestAuthErrorMissingAPIKeyMatchesSnapshot(t *testing.T) {
	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"auth", "check"},
	})

	if result.ExitCode != 3 {
		t.Fatalf("ExitCode = %d, want 3", result.ExitCode)
	}

	harness.AssertJSONSnapshot("auth_error_missing_api_key", result)
}

func TestAuthErrorInvalidCredentialsMatchesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"auth", "check"},
		Env: map[string]string{
			"TWENTY_API_KEY":  "bad-secret",
			"TWENTY_BASE_URL": server.URL,
		},
	})

	if result.ExitCode != 3 {
		t.Fatalf("ExitCode = %d, want 3", result.ExitCode)
	}

	harness.AssertJSONSnapshot("auth_error_invalid_credentials", result)
}

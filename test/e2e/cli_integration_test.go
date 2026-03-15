package e2e

import (
	"encoding/json"
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

func TestPeopleSearchMatchesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/people" {
			http.NotFound(w, r)
			return
		}

		body := `{
			"data": {
				"people": [
					{"id":"person_123","name":{"firstName":"Ada","lastName":"Lovelace"},"emails":{"primaryEmail":"ada@example.com"}}
				]
			},
			"totalCount": 1,
			"pageInfo": {"startCursor":"cursor_start","endCursor":"cursor_end","hasNextPage":false,"hasPreviousPage":false}
		}`
		if r.URL.Query().Get("filter") == "" {
			body = `{
				"data": {
					"people": [
						{"id":"person_123","name":{"firstName":"Ada","lastName":"Lovelace"},"emails":{"primaryEmail":"ada@example.com"}},
						{"id":"person_456","name":{"firstName":"Grace","lastName":"Hopper"},"emails":{"primaryEmail":"grace@example.com"}}
					]
				},
				"totalCount": 2,
				"pageInfo": {"startCursor":"cursor_start","endCursor":"cursor_end","hasNextPage":false,"hasPreviousPage":false}
			}`
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"people", "search", "--query", "ada", "--limit", "5"},
		Env: map[string]string{
			"TWENTY_API_KEY":  "env-secret",
			"TWENTY_BASE_URL": server.URL,
		},
	})

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	harness.AssertJSONSnapshot("people_search_success", result)
}

func TestPersonCreateMatchesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/people" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		name := body["name"].(map[string]any)
		if name["firstName"] != "Ada" || name["lastName"] != "Lovelace" {
			t.Fatalf("body = %#v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"createPerson": {
					"id":"person_new",
					"name":{"firstName":"Ada","lastName":"Lovelace"},
					"emails":{"primaryEmail":"ada@example.com"}
				}
			}
		}`))
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"person", "create", "--first-name", "Ada", "--last-name", "Lovelace", "--email", "ada@example.com"},
		Env: map[string]string{
			"TWENTY_API_KEY":  "env-secret",
			"TWENTY_BASE_URL": server.URL,
		},
	})

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	harness.AssertJSONSnapshot("person_create_success", result)
}

func TestDealUpdateMatchesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/opportunities/deal_123" || r.Method != http.MethodPatch {
			http.NotFound(w, r)
			return
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if body["stage"] != "PROPOSAL" {
			t.Fatalf("body = %#v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"updateOpportunity": {
					"id":"deal_123",
					"name":"Enterprise Expansion",
					"stage":"PROPOSAL"
				}
			}
		}`))
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"deal", "update", "--id", "deal_123", "--stage", "proposal"},
		Env: map[string]string{
			"TWENTY_API_KEY":  "env-secret",
			"TWENTY_BASE_URL": server.URL,
		},
	})

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	harness.AssertJSONSnapshot("deal_update_success", result)
}

func TestNoteAddMatchesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/notes" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"data":{"createNote":{"id":"note_123","title":"Meeting note","bodyV2":{"markdown":"hello"}}}}`))
		case r.URL.Path == "/rest/noteTargets" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"data":{"createNoteTarget":{"id":"note_target_123"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"note", "add", "--title", "Meeting note", "--body", "hello", "--person-id", "person_123"},
		Env: map[string]string{
			"TWENTY_API_KEY":  "env-secret",
			"TWENTY_BASE_URL": server.URL,
		},
	})

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	harness.AssertJSONSnapshot("note_add_success", result)
}

func TestMeetingLogMatchesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/rest/notes" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"data":{"createNote":{"id":"note_123","title":"Meeting","bodyV2":{"markdown":"met and aligned"}}}}`))
		case r.URL.Path == "/rest/noteTargets" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"data":{"createNoteTarget":{"id":"note_target_123"}}}`))
		case r.URL.Path == "/rest/tasks" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"data":{"createTask":{"id":"task_123","title":"Send proposal","status":"TODO"}}}`))
		case r.URL.Path == "/rest/taskTargets" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"data":{"createTaskTarget":{"id":"task_target_123"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"meeting", "log", "--body", "met and aligned", "--person-id", "person_123", "--create-followups", "--next-step", "Send proposal"},
		Env: map[string]string{
			"TWENTY_API_KEY":  "env-secret",
			"TWENTY_BASE_URL": server.URL,
		},
	})

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	harness.AssertJSONSnapshot("meeting_log_success", result)
}

func TestProspectImportDryRunMatchesSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/companies", "/rest/people":
			_, _ = w.Write([]byte(`{"data":{"companies":[],"people":[]},"totalCount":0,"pageInfo":{"startCursor":"","endCursor":"","hasNextPage":false,"hasPreviousPage":false}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	harness := newHarness(t)
	result := harness.Run(RunOptions{
		Args: []string{"prospect", "import", "--file", "prospects.jsonl", "--lookup-first", "--dry-run"},
		WorkingDirFiles: map[string]string{
			"prospects.jsonl": "{\"first_name\":\"Ada\",\"last_name\":\"Lovelace\",\"email\":\"ada@example.com\",\"company\":\"Analytical Engines\",\"company_domain\":\"analytical.example\"}\n",
		},
		Env: map[string]string{
			"TWENTY_API_KEY":  "env-secret",
			"TWENTY_BASE_URL": server.URL,
		},
	})

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0; stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	harness.AssertJSONSnapshot("prospect_import_dry_run", result)
}

package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteSuccessJSONOmitsEmptyMeta(t *testing.T) {
	var buf bytes.Buffer

	err := WriteSuccessJSON(&buf, Result{
		Command: "people.list",
		Data: []map[string]any{
			{"id": "person_123"},
		},
	})
	if err != nil {
		t.Fatalf("WriteSuccessJSON() error = %v", err)
	}

	got := buf.String()
	if strings.Contains(got, `"meta"`) {
		t.Fatalf("output unexpectedly contained meta: %s", got)
	}
	if !strings.Contains(got, `"command": "people.list"`) {
		t.Fatalf("output = %s", got)
	}
}

func TestWriteSuccessJSONIncludesPageInfoAndWarnings(t *testing.T) {
	var buf bytes.Buffer

	err := WriteSuccessJSON(&buf, Result{
		Command: "people.list",
		Data:    []string{"person_123"},
		Meta: &Meta{
			PageInfo: &PageInfo{
				Limit:      25,
				Returned:   1,
				NextCursor: "cursor_2",
			},
			Warnings: []Warning{
				{Code: "partial_results", Message: "result set truncated"},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteSuccessJSON() error = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `"page_info"`) {
		t.Fatalf("output = %s", got)
	}
	if !strings.Contains(got, `"warnings"`) {
		t.Fatalf("output = %s", got)
	}
}

func TestFailureEnvelopeMapsExitCodesByKind(t *testing.T) {
	cases := []struct {
		name string
		kind ErrorKind
		want ExitCode
	}{
		{name: "usage", kind: ErrorKindUsage, want: ExitUsage},
		{name: "auth", kind: ErrorKindAuth, want: ExitAuth},
		{name: "api", kind: ErrorKindAPI, want: ExitAPI},
		{name: "internal", kind: ErrorKindInternal, want: ExitInternal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failure := Failure{Kind: tc.kind}
			if got := failure.ExitCode(); got != tc.want {
				t.Fatalf("ExitCode() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestWriteFailureJSONIncludesNormalizedFields(t *testing.T) {
	var buf bytes.Buffer

	err := WriteFailureJSON(&buf, Failure{
		Command:   "auth.check",
		Kind:      ErrorKindAPI,
		Code:      "auth.check_failed",
		Message:   "api request failed with status 500",
		Retryable: true,
		Details: APIErrorDetails{
			StatusCode: 500,
			Body:       `{"error":"boom"}`,
		},
	})
	if err != nil {
		t.Fatalf("WriteFailureJSON() error = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `"kind": "api"`) {
		t.Fatalf("output = %s", got)
	}
	if !strings.Contains(got, `"retryable": true`) {
		t.Fatalf("output = %s", got)
	}
	if !strings.Contains(got, `"status_code": 500`) {
		t.Fatalf("output = %s", got)
	}
}

func TestWriteTextUsesResultModelMessages(t *testing.T) {
	var success bytes.Buffer
	var failure bytes.Buffer

	if err := WriteSuccessText(&success, Result{Text: "auth ok"}); err != nil {
		t.Fatalf("WriteSuccessText() error = %v", err)
	}
	if err := WriteFailureText(&failure, Failure{Message: "missing API key"}); err != nil {
		t.Fatalf("WriteFailureText() error = %v", err)
	}

	if got := success.String(); got != "auth ok\n" {
		t.Fatalf("success output = %q, want %q", got, "auth ok\n")
	}
	if got := failure.String(); got != "missing API key\n" {
		t.Fatalf("failure output = %q, want %q", got, "missing API key\n")
	}
}

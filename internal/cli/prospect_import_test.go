package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/jv/twenty-crm-cli/internal/client"
	"github.com/jv/twenty-crm-cli/internal/config"
)

type importClientStub struct {
	createCalls []createCall
	nextID      int
}

type createCall struct {
	plural  string
	action  string
	payload map[string]any
}

func (s *importClientStub) AuthCheck(context.Context) (client.AuthCheckResult, error) {
	return client.AuthCheckResult{}, nil
}

func (s *importClientStub) MetadataObjects(context.Context) ([]client.MetadataObject, error) {
	return nil, nil
}

func (s *importClientStub) ListRecords(context.Context, string, url.Values) (client.ListResult, error) {
	return client.ListResult{}, nil
}

func (s *importClientStub) GetRecord(context.Context, string, string, string, url.Values) (client.RecordResult, error) {
	return client.RecordResult{}, nil
}

func (s *importClientStub) CreateRecord(_ context.Context, plural, action string, payload map[string]any) (client.RecordResult, error) {
	s.createCalls = append(s.createCalls, createCall{plural: plural, action: action, payload: payload})
	s.nextID++
	return client.RecordResult{Record: map[string]any{"id": plural + "_id_" + string(rune('0'+s.nextID))}}, nil
}

func (s *importClientStub) UpdateRecord(context.Context, string, string, string, map[string]any) (client.RecordResult, error) {
	return client.RecordResult{}, nil
}

func TestProspectImportReusesSameBatchCompany(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "prospects.json")
	data := `[
	  {"first_name":"Import","last_name":"One","email":"one@example.test","company":"Acme","company_domain":"acme.example.test"},
	  {"first_name":"Import","last_name":"Two","email":"two@example.test","company":"Acme","company_domain":"acme.example.test"}
	]`
	if err := os.WriteFile(file, []byte(data), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stub := &importClientStub{}
	app := New(&stdout, &stderr)
	app.clientFactory = func(config.Config, client.HTTPDoer) twentyClient {
		return stub
	}

	code := app.Run([]string{"--api-key", "secret", "prospect", "import", "--file", file})
	if code != 0 {
		t.Fatalf("Run() code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}

	var envelope struct {
		OK   bool `json:"ok"`
		Data struct {
			Processed        int `json:"processed"`
			CreatedPeople    int `json:"created_people"`
			CreatedCompanies int `json:"created_companies"`
			SkippedCompanies int `json:"skipped_companies"`
			Failed           int `json:"failed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !envelope.OK {
		t.Fatalf("OK = false, stdout = %s", stdout.String())
	}
	if envelope.Data.Processed != 2 {
		t.Fatalf("Processed = %d, want 2", envelope.Data.Processed)
	}
	if envelope.Data.CreatedPeople != 2 {
		t.Fatalf("CreatedPeople = %d, want 2", envelope.Data.CreatedPeople)
	}
	if envelope.Data.CreatedCompanies != 1 {
		t.Fatalf("CreatedCompanies = %d, want 1", envelope.Data.CreatedCompanies)
	}
	if envelope.Data.Failed != 0 {
		t.Fatalf("Failed = %d, want 0", envelope.Data.Failed)
	}

	var companyCreates int
	var personCreates int
	for _, call := range stub.createCalls {
		switch call.plural {
		case "companies":
			companyCreates++
		case "people":
			personCreates++
		}
	}
	if companyCreates != 1 {
		t.Fatalf("companyCreates = %d, want 1", companyCreates)
	}
	if personCreates != 2 {
		t.Fatalf("personCreates = %d, want 2", personCreates)
	}
}

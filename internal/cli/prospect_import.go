package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jv/twenty-crm-cli/internal/config"
	"github.com/jv/twenty-crm-cli/internal/output"
)

type prospectRecord struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	Company      string `json:"company"`
	CompanyDomain string `json:"company_domain"`
	JobTitle     string `json:"job_title"`
	City         string `json:"city"`
}

type prospectImportSummary struct {
	Processed        int                `json:"processed"`
	CreatedPeople    int                `json:"created_people"`
	CreatedCompanies int                `json:"created_companies"`
	SkippedPeople    int                `json:"skipped_people"`
	SkippedCompanies int                `json:"skipped_companies"`
	Failed           int                `json:"failed"`
	DryRun           bool               `json:"dry_run,omitempty"`
	Results          []map[string]any   `json:"results,omitempty"`
}

type companyImportState struct {
	ID     string
	Action string
}

func (a *App) runProspectImport(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("prospect.import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var file string
	var lookupFirst bool
	var dryRun bool
	fs.StringVar(&file, "file", "", "Path to JSON/JSONL prospect file, or - for stdin")
	fs.BoolVar(&lookupFirst, "lookup-first", false, "Search before creating")
	fs.BoolVar(&dryRun, "dry-run", false, "Preview without creating")

	if err := fs.Parse(args); err != nil {
		return a.writeFailure(output.Failure{
			Command: "prospect.import",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.parse",
			Message: err.Error(),
		}, cfg.Format)
	}
	if file == "" {
		return a.writeFailure(output.Failure{
			Command: "prospect.import",
			Kind:    output.ErrorKindUsage,
			Code:    "prospect.import.missing_file",
			Message: "missing required flag: --file",
		}, cfg.Format)
	}

	records, err := loadProspects(file)
	if err != nil {
		return a.writeFailure(output.Failure{
			Command: "prospect.import",
			Kind:    output.ErrorKindUsage,
			Code:    "prospect.import.file",
			Message: err.Error(),
		}, cfg.Format)
	}

	cli := a.clientFactory(cfg, a.httpClient)
	summary := prospectImportSummary{DryRun: dryRun}
	companyCache := map[string]companyImportState{}
	for _, record := range records {
		summary.Processed++
		item := map[string]any{
			"email":   record.Email,
			"company": record.Company,
		}

		companyID := ""
		companyAction := ""
		if record.Company != "" || record.CompanyDomain != "" {
			var companyErr error
			companyAction, companyID, companyErr = ensureCompany(cli, companyCache, record, lookupFirst, dryRun)
			if companyErr != nil {
				summary.Failed++
				item["status"] = "failed"
				item["error"] = companyErr.Error()
				summary.Results = append(summary.Results, item)
				continue
			}
			if companyID != "" {
				item["company_id"] = companyID
			}
			if companyAction != "" {
				item["company_action"] = companyAction
			}
		}

		personAction, personID, err := ensurePerson(cli, record, companyID, lookupFirst, dryRun)
		if err != nil {
			summary.Failed++
			item["status"] = "failed"
			item["error"] = err.Error()
			summary.Results = append(summary.Results, item)
			continue
		}

		switch personAction {
		case "created":
			summary.CreatedPeople++
		case "skipped":
			summary.SkippedPeople++
		case "planned":
		}
		switch companyAction {
		case "created":
			summary.CreatedCompanies++
		case "skipped":
			summary.SkippedCompanies++
		}
		item["status"] = personAction
		item["person_id"] = personID
		summary.Results = append(summary.Results, item)
	}

	return a.writeSuccess(output.Result{
		Command: "prospect.import",
		Data:    summary,
		Text:    fmt.Sprintf("%d prospects processed", summary.Processed),
	}, cfg.Format)
}

func loadProspects(path string) ([]prospectRecord, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = os.ReadFile("/dev/stdin")
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("prospect file is empty")
	}

	if strings.HasPrefix(trimmed, "[") {
		var records []prospectRecord
		if err := json.Unmarshal([]byte(trimmed), &records); err != nil {
			return nil, err
		}
		return records, nil
	}

	var records []prospectRecord
	scanner := bufio.NewScanner(strings.NewReader(trimmed))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record prospectRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, scanner.Err()
}

func ensureCompany(cli twentyClient, cache map[string]companyImportState, record prospectRecord, lookupFirst, dryRun bool) (string, string, error) {
	cacheKey := companyCacheKey(record)
	if cacheKey != "" {
		if state, ok := cache[cacheKey]; ok {
			if state.Action == "created" {
				return "reused", state.ID, nil
			}
			return state.Action, state.ID, nil
		}
	}

	query := strings.TrimSpace(record.Company)
	if query == "" {
		query = strings.TrimSpace(record.CompanyDomain)
	}
	if lookupFirst && query != "" {
		result, err := searchEntityRecords(cli, entityDefs[1], defaultSearchValues(10), query)
		if err != nil {
			return "", "", err
		}
		if len(result.Records) > 0 {
			id, _ := result.Records[0]["id"].(string)
			if cacheKey != "" {
				cache[cacheKey] = companyImportState{ID: id, Action: "skipped"}
			}
			return "skipped", id, nil
		}
	}
	if dryRun {
		if cacheKey != "" {
			cache[cacheKey] = companyImportState{Action: "planned"}
		}
		return "planned", "", nil
	}

	payload := map[string]any{}
	if record.Company != "" {
		payload["name"] = record.Company
	}
	if record.CompanyDomain != "" {
		payload["domainName"] = map[string]any{
			"primaryLinkUrl": record.CompanyDomain,
			"secondaryLinks": []string{},
		}
	}
	created, err := cli.CreateRecord(context.Background(), "companies", "createCompany", payload)
	if err != nil {
		return "", "", err
	}
	id, _ := created.Record["id"].(string)
	if cacheKey != "" {
		cache[cacheKey] = companyImportState{ID: id, Action: "created"}
	}
	return "created", id, nil
}

func ensurePerson(cli twentyClient, record prospectRecord, companyID string, lookupFirst, dryRun bool) (string, string, error) {
	query := strings.TrimSpace(record.Email)
	if query == "" {
		query = strings.TrimSpace(strings.TrimSpace(record.FirstName + " " + record.LastName))
	}
	if lookupFirst && query != "" {
		result, err := searchEntityRecords(cli, entityDefs[0], defaultSearchValues(10), query)
		if err != nil {
			return "", "", err
		}
		if len(result.Records) > 0 {
			id, _ := result.Records[0]["id"].(string)
			return "skipped", id, nil
		}
	}
	if dryRun {
		return "planned", "", nil
	}

	payload := map[string]any{
		"name": map[string]any{
			"firstName": record.FirstName,
			"lastName":  record.LastName,
		},
	}
	if record.Email != "" {
		payload["emails"] = map[string]any{
			"primaryEmail":     record.Email,
			"additionalEmails": []string{},
		}
	}
	if record.JobTitle != "" {
		payload["jobTitle"] = record.JobTitle
	}
	if record.City != "" {
		payload["city"] = record.City
	}
	if companyID != "" {
		payload["companyId"] = companyID
	}
	created, err := cli.CreateRecord(context.Background(), "people", "createPerson", payload)
	if err != nil {
		return "", "", err
	}
	id, _ := created.Record["id"].(string)
	return "created", id, nil
}

func defaultSearchValues(limit int) map[string][]string {
	return map[string][]string{
		"limit": {fmt.Sprintf("%d", limit)},
		"depth": {"0"},
	}
}

func companyCacheKey(record prospectRecord) string {
	company := strings.ToLower(strings.TrimSpace(record.Company))
	domain := strings.ToLower(strings.TrimSpace(record.CompanyDomain))
	if company == "" && domain == "" {
		return ""
	}
	return company + "|" + domain
}

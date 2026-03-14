package cli

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jv/twenty-crm-cli/internal/client"
	"github.com/jv/twenty-crm-cli/internal/config"
	"github.com/jv/twenty-crm-cli/internal/output"
)

type twentyClient interface {
	AuthCheck(ctx context.Context) (client.AuthCheckResult, error)
	MetadataObjects(ctx context.Context) ([]client.MetadataObject, error)
	ListRecords(ctx context.Context, plural string, query url.Values) (client.ListResult, error)
	GetRecord(ctx context.Context, plural, singular, id string, query url.Values) (client.RecordResult, error)
	CreateRecord(ctx context.Context, plural, action string, payload map[string]any) (client.RecordResult, error)
	UpdateRecord(ctx context.Context, plural, action, id string, payload map[string]any) (client.RecordResult, error)
}

type entityDef struct {
	domain        string
	singularCmd   string
	pluralCmd     string
	pluralRoute   string
	singularRoute string
	createAction  string
	updateAction  string
}

var entityDefs = []entityDef{
	{
		domain:        "person",
		singularCmd:   "person",
		pluralCmd:     "people",
		pluralRoute:   "people",
		singularRoute: "person",
		createAction:  "createPerson",
		updateAction:  "updatePerson",
	},
	{
		domain:        "company",
		singularCmd:   "company",
		pluralCmd:     "companies",
		pluralRoute:   "companies",
		singularRoute: "company",
		createAction:  "createCompany",
		updateAction:  "updateCompany",
	},
	{
		domain:        "deal",
		singularCmd:   "deal",
		pluralCmd:     "deals",
		pluralRoute:   "opportunities",
		singularRoute: "opportunity",
		createAction:  "createOpportunity",
		updateAction:  "updateOpportunity",
	},
}

func entityForToken(token string) (entityDef, bool) {
	switch token {
	case "person", "people", "contact", "contacts":
		return entityDefs[0], true
	case "company", "companies":
		return entityDefs[1], true
	case "deal", "deals", "opportunity", "opportunities":
		return entityDefs[2], true
	default:
		return entityDef{}, false
	}
}

func (a *App) runEntity(cfg config.Config, token string, args []string) int {
	entity, ok := entityForToken(token)
	if !ok {
		return a.writeFailure(output.Failure{
			Command: "cli",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.unknown_command",
			Message: fmt.Sprintf("unknown command: %s", token),
		}, cfg.Format)
	}

	if len(args) == 0 {
		return a.writeFailure(output.Failure{
			Command: entity.singularCmd,
			Kind:    output.ErrorKindUsage,
			Code:    entity.singularCmd + ".usage",
			Message: "expected subcommand",
		}, cfg.Format)
	}

	cli := a.clientFactory(cfg, a.httpClient)
	switch args[0] {
	case "search", "list":
		return a.runEntitySearch(cli, cfg, entity, args[1:])
	case "get":
		return a.runEntityGet(cli, cfg, entity, args[1:])
	case "create":
		return a.runEntityCreate(cli, cfg, entity, args[1:])
	case "update":
		return a.runEntityUpdate(cli, cfg, entity, args[1:])
	default:
		return a.writeFailure(output.Failure{
			Command: entity.singularCmd,
			Kind:    output.ErrorKindUsage,
			Code:    entity.singularCmd + ".unknown_subcommand",
			Message: fmt.Sprintf("unknown %s subcommand: %s", entity.singularCmd, args[0]),
		}, cfg.Format)
	}
}

func (a *App) runEntitySearch(cli twentyClient, cfg config.Config, entity entityDef, args []string) int {
	fs := flag.NewFlagSet(entity.pluralCmd, flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})

	var query string
	var limit int
	var depth int
	var startingAfter string
	var endingBefore string

	fs.StringVar(&query, "query", "", "Free-text search")
	fs.IntVar(&limit, "limit", 10, "Maximum records to return")
	fs.IntVar(&depth, "depth", 0, "Relation depth")
	fs.StringVar(&startingAfter, "starting-after", "", "Cursor for next page")
	fs.StringVar(&endingBefore, "ending-before", "", "Cursor for previous page")

	if err := fs.Parse(args); err != nil {
		return a.writeFailure(output.Failure{
			Command: entity.pluralCmd + ".search",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.parse",
			Message: err.Error(),
		}, cfg.Format)
	}

	queryValues := url.Values{}
	fetchLimit := limit
	if fetchLimit <= 0 {
		fetchLimit = 10
	}
	if strings.TrimSpace(query) != "" && fetchLimit < 100 {
		fetchLimit = 100
	}
	queryValues.Set("limit", strconv.Itoa(fetchLimit))
	queryValues.Set("depth", strconv.Itoa(depth))
	if startingAfter != "" {
		queryValues.Set("starting_after", startingAfter)
	}
	if endingBefore != "" {
		queryValues.Set("ending_before", endingBefore)
	}

	result, err := cli.ListRecords(context.Background(), entity.pluralRoute, queryValues)
	if err != nil {
		return a.writeClientError(entity.pluralCmd+".search", cfg.Format, err)
	}

	records := result.Records
	if strings.TrimSpace(query) != "" {
		records = filterRecords(records, query)
	}
	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}

	return a.writeSuccess(output.Result{
		Command: entity.pluralCmd + ".search",
		Data:    records,
		Meta: &output.Meta{
			PageInfo: &output.PageInfo{
				Limit:      limit,
				Returned:   len(records),
				Total:      result.TotalCount,
				NextCursor: result.PageInfo.EndCursor,
				PrevCursor: result.PageInfo.StartCursor,
			},
		},
		Text: fmt.Sprintf("%d %s", len(records), entity.pluralCmd),
	}, cfg.Format)
}

func (a *App) runEntityGet(cli twentyClient, cfg config.Config, entity entityDef, args []string) int {
	fs := flag.NewFlagSet(entity.singularCmd, flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})

	var id string
	var depth int
	fs.StringVar(&id, "id", "", "Record ID")
	fs.IntVar(&depth, "depth", 0, "Relation depth")

	if err := fs.Parse(args); err != nil {
		return a.writeFailure(output.Failure{
			Command: entity.singularCmd + ".get",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.parse",
			Message: err.Error(),
		}, cfg.Format)
	}
	if strings.TrimSpace(id) == "" {
		return a.writeFailure(output.Failure{
			Command: entity.singularCmd + ".get",
			Kind:    output.ErrorKindUsage,
			Code:    entity.singularCmd + ".missing_id",
			Message: "missing required flag: --id",
		}, cfg.Format)
	}

	queryValues := url.Values{}
	queryValues.Set("depth", strconv.Itoa(depth))
	result, err := cli.GetRecord(context.Background(), entity.pluralRoute, entity.singularRoute, id, queryValues)
	if err != nil {
		return a.writeClientError(entity.singularCmd+".get", cfg.Format, err)
	}

	return a.writeSuccess(output.Result{
		Command: entity.singularCmd + ".get",
		Data:    result.Record,
		Text:    fmt.Sprintf("%s %s", entity.singularCmd, id),
	}, cfg.Format)
}

func (a *App) runEntityCreate(cli twentyClient, cfg config.Config, entity entityDef, args []string) int {
	payload, failure, ok := parseEntityMutation(entity, "create", args)
	if !ok {
		return a.writeFailure(failure, cfg.Format)
	}

	result, err := cli.CreateRecord(context.Background(), entity.pluralRoute, entity.createAction, payload)
	if err != nil {
		return a.writeClientError(entity.singularCmd+".create", cfg.Format, err)
	}

	return a.writeSuccess(output.Result{
		Command: entity.singularCmd + ".create",
		Data:    result.Record,
		Text:    entity.singularCmd + " created",
	}, cfg.Format)
}

func (a *App) runEntityUpdate(cli twentyClient, cfg config.Config, entity entityDef, args []string) int {
	payload, failure, ok := parseEntityMutation(entity, "update", args)
	if !ok {
		return a.writeFailure(failure, cfg.Format)
	}

	id, _ := payload["id"].(string)
	delete(payload, "id")
	result, err := cli.UpdateRecord(context.Background(), entity.pluralRoute, entity.updateAction, id, payload)
	if err != nil {
		return a.writeClientError(entity.singularCmd+".update", cfg.Format, err)
	}

	return a.writeSuccess(output.Result{
		Command: entity.singularCmd + ".update",
		Data:    result.Record,
		Text:    entity.singularCmd + " updated",
	}, cfg.Format)
}

func (a *App) writeClientError(command, format string, err error) int {
	if apiErr, ok := err.(*client.APIError); ok {
		failure := output.Failure{
			Command:   command,
			Kind:      output.ErrorKindAPI,
			Code:      command + ".failed",
			Message:   err.Error(),
			Retryable: apiErr.StatusCode >= http.StatusInternalServerError || apiErr.StatusCode == http.StatusTooManyRequests,
			Details: output.APIErrorDetails{
				StatusCode: apiErr.StatusCode,
				Body:       apiErr.Body,
			},
		}

		if apiErr.StatusCode == http.StatusUnauthorized {
			failure.Kind = output.ErrorKindAuth
			failure.Code = "auth.invalid_credentials"
		} else if apiErr.StatusCode == http.StatusForbidden {
			failure.Kind = output.ErrorKindAuth
			failure.Code = "auth.insufficient_permissions"
		}

		return a.writeFailure(failure, format)
	}

	return a.writeFailure(output.Failure{
		Command:   command,
		Kind:      output.ErrorKindInternal,
		Code:      command + ".internal",
		Message:   err.Error(),
		Retryable: true,
	}, format)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func filterRecords(records []map[string]any, query string) []map[string]any {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return records
	}

	filtered := make([]map[string]any, 0, len(records))
	for _, record := range records {
		if matchesRecordQuery(record, query) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func matchesRecordQuery(record map[string]any, query string) bool {
	for key, value := range record {
		if shouldSkipSearchKey(key) {
			continue
		}
		if valueMatchesQuery(key, value, query) {
			return true
		}
	}
	return false
}

func valueMatchesQuery(key string, value any, query string) bool {
	switch typed := value.(type) {
	case string:
		return strings.Contains(strings.ToLower(typed), query)
	case map[string]any:
		for nestedKey, nestedValue := range typed {
			if shouldSkipSearchKey(nestedKey) {
				continue
			}
			if valueMatchesQuery(nestedKey, nestedValue, query) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if valueMatchesQuery(key, item, query) {
				return true
			}
		}
	}
	return false
}

func shouldSkipSearchKey(key string) bool {
	key = strings.ToLower(key)
	if strings.HasSuffix(key, "id") || strings.HasSuffix(key, "at") {
		return true
	}

	switch key {
	case "searchvector", "createdby", "updatedby", "deletedat", "position", "context":
		return true
	default:
		return false
	}
}

func parseEntityMutation(entity entityDef, action string, args []string) (map[string]any, output.Failure, bool) {
	fs := flag.NewFlagSet(entity.singularCmd, flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})

	var id string
	var firstName string
	var lastName string
	var email string
	var jobTitle string
	var city string
	var companyID string
	var name string
	var domain string
	var employees int
	var tagline string
	var stage string
	var personID string

	fs.StringVar(&id, "id", "", "Record ID")
	fs.StringVar(&firstName, "first-name", "", "First name")
	fs.StringVar(&lastName, "last-name", "", "Last name")
	fs.StringVar(&email, "email", "", "Primary email")
	fs.StringVar(&jobTitle, "job-title", "", "Job title")
	fs.StringVar(&city, "city", "", "City")
	fs.StringVar(&companyID, "company-id", "", "Company ID")
	fs.StringVar(&name, "name", "", "Name")
	fs.StringVar(&domain, "domain", "", "Domain")
	fs.IntVar(&employees, "employees", 0, "Employees")
	fs.StringVar(&tagline, "tagline", "", "Tagline")
	fs.StringVar(&stage, "stage", "", "Deal stage")
	fs.StringVar(&personID, "person-id", "", "Person ID")

	if err := fs.Parse(args); err != nil {
		return nil, output.Failure{
			Command: entity.singularCmd + "." + action,
			Kind:    output.ErrorKindUsage,
			Code:    "cli.parse",
			Message: err.Error(),
		}, false
	}

	payload := map[string]any{}
	switch entity.domain {
	case "person":
		if firstName != "" || lastName != "" {
			payload["name"] = map[string]any{
				"firstName": firstName,
				"lastName":  lastName,
			}
		}
		if email != "" {
			payload["emails"] = map[string]any{
				"primaryEmail":     email,
				"additionalEmails": []string{},
			}
		}
		if jobTitle != "" {
			payload["jobTitle"] = jobTitle
		}
		if city != "" {
			payload["city"] = city
		}
		if companyID != "" {
			payload["companyId"] = companyID
		}
	case "company":
		if name != "" {
			payload["name"] = name
		}
		if domain != "" {
			payload["domainName"] = map[string]any{
				"primaryLinkUrl": domain,
				"secondaryLinks": []string{},
			}
		}
		if employees != 0 {
			payload["employees"] = employees
		}
		if tagline != "" {
			payload["tagline"] = tagline
		}
	case "deal":
		if name != "" {
			payload["name"] = name
		}
		if stage != "" {
			payload["stage"] = strings.ToUpper(stage)
		}
		if companyID != "" {
			payload["companyId"] = companyID
		}
		if personID != "" {
			payload["pointOfContactId"] = personID
		}
	}

	if action == "create" {
		switch entity.domain {
		case "person":
			if _, ok := payload["name"]; !ok {
				return nil, output.Failure{
					Command: entity.singularCmd + "." + action,
					Kind:    output.ErrorKindUsage,
					Code:    entity.singularCmd + ".missing_name",
					Message: "missing required flags: --first-name and/or --last-name",
				}, false
			}
		case "company", "deal":
			if _, ok := payload["name"]; !ok {
				return nil, output.Failure{
					Command: entity.singularCmd + "." + action,
					Kind:    output.ErrorKindUsage,
					Code:    entity.singularCmd + ".missing_name",
					Message: "missing required flag: --name",
				}, false
			}
		}
	}

	if action == "update" {
		if strings.TrimSpace(id) == "" {
			return nil, output.Failure{
				Command: entity.singularCmd + "." + action,
				Kind:    output.ErrorKindUsage,
				Code:    entity.singularCmd + ".missing_id",
				Message: "missing required flag: --id",
			}, false
		}
		if len(payload) == 0 {
			return nil, output.Failure{
				Command: entity.singularCmd + "." + action,
				Kind:    output.ErrorKindUsage,
				Code:    entity.singularCmd + ".missing_changes",
				Message: "no update fields provided",
			}, false
		}
		payload["id"] = id
	}

	return payload, output.Failure{}, true
}

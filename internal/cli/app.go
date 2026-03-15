package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jv/twenty-crm-cli/internal/client"
	"github.com/jv/twenty-crm-cli/internal/config"
	"github.com/jv/twenty-crm-cli/internal/output"
)

type App struct {
	stdout        io.Writer
	stderr        io.Writer
	httpClient    client.HTTPDoer
	clientFactory func(config.Config, client.HTTPDoer) twentyClient
}

func New(stdout, stderr io.Writer) *App {
	return &App{
		stdout:     stdout,
		stderr:     stderr,
		httpClient: http.DefaultClient,
		clientFactory: func(cfg config.Config, doer client.HTTPDoer) twentyClient {
			return client.New(cfg, doer)
		},
	}
}

func (a *App) Run(args []string) int {
	cfg, remaining, err := a.parseRoot(args)
	if err != nil {
		return a.writeFailure(output.Failure{
			Command: "cli",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.parse",
			Message: err.Error(),
		}, inferRequestedFormat(args))
	}

	if len(remaining) == 0 {
		return a.writeUsage(cfg.Format)
	}

	switch remaining[0] {
	case "auth":
		return a.runAuth(cfg, remaining[1:])
	case "version":
		return a.writeSuccess(output.Result{
			Command: "version",
			Data:    map[string]string{"version": "dev"},
			Text:    "dev",
		}, cfg.Format)
	case "person", "people", "contact", "contacts", "company", "companies", "deal", "deals", "opportunity", "opportunities":
		return a.runEntity(cfg, remaining[0], remaining[1:])
	case "note", "task", "meeting", "call", "prospect":
		return a.runWorkflow(cfg, remaining[0], remaining[1:])
	default:
		return a.writeFailure(output.Failure{
			Command: "cli",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.unknown_command",
			Message: fmt.Sprintf("unknown command: %s", remaining[0]),
		}, cfg.Format)
	}
}

func (a *App) parseRoot(args []string) (config.Config, []string, error) {
	fs := flag.NewFlagSet("twenty", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var apiKey string
	var baseURL string
	var format string

	fs.StringVar(&apiKey, "api-key", "", "Twenty API key")
	fs.StringVar(&baseURL, "base-url", "", "Twenty base URL")
	fs.StringVar(&format, "format", "json", "Output format: json|text")

	if err := fs.Parse(args); err != nil {
		return config.Config{}, nil, err
	}

	cfg, err := config.New(apiKey, baseURL, format)
	if err != nil {
		return config.Config{}, nil, err
	}

	return cfg, fs.Args(), nil
}

func (a *App) runAuth(cfg config.Config, args []string) int {
	if len(args) == 0 {
		return a.writeFailure(output.Failure{
			Command: "auth",
			Kind:    output.ErrorKindUsage,
			Code:    "auth.usage",
			Message: "expected subcommand: check",
		}, cfg.Format)
	}

	switch args[0] {
	case "check":
		if err := cfg.ValidateAuth(); err != nil {
			return a.writeFailure(output.Failure{
				Command: "auth.check",
				Kind:    output.ErrorKindAuth,
				Code:    "auth.missing_api_key",
				Message: err.Error(),
			}, cfg.Format)
		}

		cli := a.clientFactory(cfg, a.httpClient)
		result, err := cli.AuthCheck(context.Background())
		if err != nil {
			if apiErr, ok := err.(*client.APIError); ok {
				failure := output.Failure{
					Command:   "auth.check",
					Kind:      output.ErrorKindAPI,
					Code:      "auth.check_failed",
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

				return a.writeFailure(failure, cfg.Format)
			}

			return a.writeFailure(output.Failure{
				Command:   "auth.check",
				Kind:      output.ErrorKindInternal,
				Code:      "auth.request_failed",
				Message:   err.Error(),
				Retryable: true,
			}, cfg.Format)
		}

		return a.writeSuccess(output.Result{
			Command: "auth.check",
			Data:    result,
			Text:    "auth ok",
		}, cfg.Format)
	default:
		return a.writeFailure(output.Failure{
			Command: "auth",
			Kind:    output.ErrorKindUsage,
			Code:    "auth.unknown_subcommand",
			Message: fmt.Sprintf("unknown auth subcommand: %s", args[0]),
		}, cfg.Format)
	}
}

func (a *App) writeSuccess(result output.Result, format string) int {
	if format == "text" {
		if err := output.WriteSuccessText(a.stdout, result); err != nil {
			fmt.Fprintf(a.stderr, "write error: %v\n", err)
			return int(output.ExitInternal)
		}

		return int(output.ExitOK)
	}

	if err := output.WriteSuccessJSON(a.stdout, result); err != nil {
		fmt.Fprintf(a.stderr, "write error: %v\n", err)
		return int(output.ExitInternal)
	}

	return int(output.ExitOK)
}

func (a *App) writeFailure(failure output.Failure, format string) int {
	if format == "text" {
		if err := output.WriteFailureText(a.stderr, failure); err != nil {
			fmt.Fprintf(a.stderr, "write error: %v\n", err)
			return int(output.ExitInternal)
		}

		return int(failure.ExitCode())
	}

	if err := output.WriteFailureJSON(a.stdout, failure); err != nil {
		fmt.Fprintf(a.stderr, "write error: %v\n", err)
		return int(output.ExitInternal)
	}

	return int(failure.ExitCode())
}

func (a *App) writeUsage(format string) int {
	lines := []string{
		"Usage:",
		"  twenty [--api-key KEY] [--base-url URL] [--format json|text] <command>",
		"",
		"Commands:",
		"  auth check      Validate connectivity and API credentials",
		"  people search   Search people",
		"  person get      Fetch one person by ID",
		"  person create   Create one person",
		"  person update   Update one person",
		"  companies search Search companies",
		"  company get     Fetch one company by ID",
		"  company create  Create one company",
		"  company update  Update one company",
		"  deals search    Search deals",
		"  deal get        Fetch one deal by ID",
		"  deal create     Create one deal",
		"  deal update     Update one deal",
		"  note add        Add a linked note",
		"  task create     Create a linked follow-up task",
		"  meeting log     Log meeting notes and optional follow-ups",
		"  call capture    Capture call notes and optional next steps",
		"  prospect import Import prospect records from JSON/JSONL",
		"  version         Print CLI version",
	}

	return a.writeFailure(output.Failure{
		Command: "cli",
		Kind:    output.ErrorKindUsage,
		Code:    "cli.usage",
		Message: strings.Join(lines, "\n"),
	}, format)
}

func inferRequestedFormat(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--format" && i+1 < len(args) {
			if args[i+1] == "text" {
				return "text"
			}

			return "json"
		}

		if strings.HasPrefix(arg, "--format=") {
			value := strings.TrimPrefix(arg, "--format=")
			if value == "text" {
				return "text"
			}

			return "json"
		}
	}

	return "json"
}

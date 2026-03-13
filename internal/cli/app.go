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

const (
	exitOK       = 0
	exitUsage    = 2
	exitAuth     = 3
	exitAPI      = 4
	exitInternal = 10
)

type App struct {
	stdout        io.Writer
	stderr        io.Writer
	httpClient    client.HTTPDoer
	clientFactory func(config.Config, client.HTTPDoer) authChecker
}

type authChecker interface {
	AuthCheck(ctx context.Context) (client.AuthCheckResult, error)
}

func New(stdout, stderr io.Writer) *App {
	return &App{
		stdout:     stdout,
		stderr:     stderr,
		httpClient: http.DefaultClient,
		clientFactory: func(cfg config.Config, doer client.HTTPDoer) authChecker {
			return client.New(cfg, doer)
		},
	}
}

func (a *App) Run(args []string) int {
	cfg, remaining, err := a.parseRoot(args)
	if err != nil {
		return a.writeError("cli.parse", err.Error(), nil, inferRequestedFormat(args), exitUsage)
	}

	if len(remaining) == 0 {
		return a.writeUsage(cfg.Format)
	}

	switch remaining[0] {
	case "auth":
		return a.runAuth(cfg, remaining[1:])
	case "version":
		return a.writeSuccess("version", map[string]string{"version": "dev"}, cfg.Format)
	default:
		return a.writeError("cli.unknown_command", fmt.Sprintf("unknown command: %s", remaining[0]), nil, cfg.Format, exitUsage)
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
		return a.writeError("auth.usage", "expected subcommand: check", nil, cfg.Format, exitUsage)
	}

	switch args[0] {
	case "check":
		if err := cfg.ValidateAuth(); err != nil {
			return a.writeError("auth.missing_api_key", err.Error(), nil, cfg.Format, exitAuth)
		}

		cli := a.clientFactory(cfg, a.httpClient)
		result, err := cli.AuthCheck(context.Background())
		if err != nil {
			if apiErr, ok := err.(*client.APIError); ok {
				code := "auth.check_failed"
				exitCode := exitAPI
				if apiErr.StatusCode == http.StatusUnauthorized {
					code = "auth.invalid_credentials"
					exitCode = exitAuth
				} else if apiErr.StatusCode == http.StatusForbidden {
					code = "auth.insufficient_permissions"
				}

				return a.writeError(code, err.Error(), apiErr, cfg.Format, exitCode)
			}

			return a.writeError("auth.request_failed", err.Error(), nil, cfg.Format, exitInternal)
		}

		return a.writeSuccess("auth.check", result, cfg.Format)
	default:
		return a.writeError("auth.unknown_subcommand", fmt.Sprintf("unknown auth subcommand: %s", args[0]), nil, cfg.Format, exitUsage)
	}
}

func (a *App) writeSuccess(command string, data any, format string) int {
	if format == "text" {
		msg := "ok"
		if command == "auth.check" {
			msg = "auth ok"
		}
		if err := output.WriteText(a.stdout, msg); err != nil {
			fmt.Fprintf(a.stderr, "write error: %v\n", err)
			return exitInternal
		}

		return exitOK
	}

	if err := output.WriteJSON(a.stdout, output.Envelope{
		OK:      true,
		Command: command,
		Data:    data,
	}); err != nil {
		fmt.Fprintf(a.stderr, "write error: %v\n", err)
		return exitInternal
	}

	return exitOK
}

func (a *App) writeError(code, message string, details any, format string, exitCode int) int {
	if format == "text" {
		if err := output.WriteText(a.stderr, message); err != nil {
			fmt.Fprintf(a.stderr, "write error: %v\n", err)
			return exitInternal
		}

		return exitCode
	}

	err := output.WriteJSON(a.stdout, output.Envelope{
		OK:      false,
		Command: "error",
		Error: &output.Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
	if err != nil {
		fmt.Fprintf(a.stderr, "write error: %v\n", err)
		return exitInternal
	}

	return exitCode
}

func (a *App) writeUsage(format string) int {
	lines := []string{
		"Usage:",
		"  twenty [--api-key KEY] [--base-url URL] [--format json|text] <command>",
		"",
		"Commands:",
		"  auth check   Validate connectivity and API credentials",
		"  version      Print CLI version",
	}

	return a.writeError("cli.usage", strings.Join(lines, "\n"), nil, format, exitUsage)
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

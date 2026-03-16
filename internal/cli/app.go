package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jv/twenty-crm-cli/internal/client"
	"github.com/jv/twenty-crm-cli/internal/config"
	"github.com/jv/twenty-crm-cli/internal/output"
	"golang.org/x/term"
)

type App struct {
	stdin         io.Reader
	stdout        io.Writer
	stderr        io.Writer
	httpClient    client.HTTPDoer
	clientFactory func(config.Config, client.HTTPDoer) twentyClient
	isTTY         func(io.Reader) bool
	promptReader  *bufio.Reader
}

func New(stdin io.Reader, stdout, stderr io.Writer) *App {
	return &App{
		stdin:      stdin,
		stdout:     stdout,
		stderr:     stderr,
		httpClient: http.DefaultClient,
		clientFactory: func(cfg config.Config, doer client.HTTPDoer) twentyClient {
			return client.New(cfg, doer)
		},
		isTTY: defaultIsTTY,
	}
}

func (a *App) Run(args []string) int {
	cfg, remaining, err := a.parseRoot(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return a.writeUsageText()
		}
		return a.writeFailure(output.Failure{
			Command: "cli",
			Kind:    output.ErrorKindUsage,
			Code:    errorCodeForConfigError(err),
			Message: errorMessageForConfigError(err),
		}, inferRequestedFormat(args))
	}

	if len(remaining) == 0 {
		if explicitFormat(args) {
			return a.writeUsage(cfg.Format)
		}
		return a.writeUsageText()
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

	remaining := fs.Args()
	cfg, err := config.New(apiKey, baseURL, format)
	if err == nil {
		return cfg, remaining, nil
	}

	var settingsErr *config.SettingsError
	if errors.As(err, &settingsErr) && len(remaining) >= 2 && remaining[0] == "auth" && remaining[1] == "login" {
		setupCfg, setupErr := config.NewWithoutSettings(apiKey, baseURL, format)
		if setupErr != nil {
			return config.Config{}, nil, setupErr
		}
		return setupCfg, remaining, nil
	}

	return config.Config{}, nil, err
}

func (a *App) runAuth(cfg config.Config, args []string) int {
	if len(args) == 0 || isHelpArg(args) {
		return a.writeHelpText(authHelpLines())
	}

	switch args[0] {
	case "check":
		if isHelpArg(args[1:]) {
			return a.writeHelpText(authCheckHelpLines())
		}
		return a.runAuthCheck(cfg)
	case "login":
		if isHelpArg(args[1:]) {
			return a.writeHelpText(authLoginHelpLines())
		}
		return a.runAuthLogin(cfg, args[1:])
	default:
		return a.writeFailure(output.Failure{
			Command: "auth",
			Kind:    output.ErrorKindUsage,
			Code:    "auth.unknown_subcommand",
			Message: fmt.Sprintf("unknown auth subcommand: %s", args[0]),
		}, cfg.Format)
	}
}

func (a *App) runAuthCheck(cfg config.Config) int {
	if err := cfg.ValidateAuth(); err != nil {
		return a.writeFailure(output.Failure{
			Command: "auth.check",
			Kind:    output.ErrorKindAuth,
			Code:    "auth.missing_api_key",
			Message: "missing API key; run `twenty auth login`, set TWENTY_API_KEY, or pass --api-key",
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
}

func (a *App) runAuthLogin(cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("auth.login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var apiKey string
	var baseURL string
	var scope string
	var overwrite bool

	fs.StringVar(&apiKey, "api-key", cfg.APIKey, "Twenty API key")
	fs.StringVar(&baseURL, "base-url", cfg.BaseURL, "Twenty base URL")
	fs.StringVar(&scope, "scope", string(config.SettingsScopeHome), "Settings scope: home|project")
	fs.BoolVar(&overwrite, "overwrite", false, "Overwrite existing settings file")

	if err := fs.Parse(args); err != nil {
		return a.writeFailure(output.Failure{
			Command: "auth.login",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.parse",
			Message: err.Error(),
		}, cfg.Format)
	}
	apiKeyProvided := flagProvided(fs, "api-key")
	baseURLProvided := flagProvided(fs, "base-url")

	apiKey = strings.TrimSpace(apiKey)
	baseURL = strings.TrimSpace(baseURL)
	settingsScope := config.SettingsScope(scope)
	if settingsScope != config.SettingsScopeHome && settingsScope != config.SettingsScopeProject {
		return a.writeFailure(output.Failure{
			Command: "auth.login",
			Kind:    output.ErrorKindUsage,
			Code:    "auth.login.scope",
			Message: "invalid --scope, expected home or project",
		}, cfg.Format)
	}

	if a.isTTY(a.stdin) {
		if !apiKeyProvided && apiKey == "" {
			value, err := a.promptSecret("API key: ")
			if err != nil {
				return a.writeFailure(output.Failure{
					Command:   "auth.login",
					Kind:      output.ErrorKindInternal,
					Code:      "auth.login.prompt_failed",
					Message:   err.Error(),
					Retryable: true,
				}, cfg.Format)
			}
			apiKey = value
		}
		if !baseURLProvided {
			value, err := a.prompt("Base URL [" + cfg.BaseURL + "]: ")
			if err != nil {
				return a.writeFailure(output.Failure{
					Command:   "auth.login",
					Kind:      output.ErrorKindInternal,
					Code:      "auth.login.prompt_failed",
					Message:   err.Error(),
					Retryable: true,
				}, cfg.Format)
			}
			if value != "" {
				baseURL = value
			}
		}
	}

	setupCfg, err := config.NewWithoutSettings(apiKey, baseURL, cfg.Format)
	if err != nil {
		return a.writeFailure(output.Failure{
			Command: "auth.login",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.parse",
			Message: err.Error(),
		}, cfg.Format)
	}
	if err := setupCfg.ValidateAuth(); err != nil {
		return a.writeFailure(output.Failure{
			Command: "auth.login",
			Kind:    output.ErrorKindAuth,
			Code:    "auth.missing_api_key",
			Message: "missing API key; pass --api-key or run `twenty auth login` interactively from a TTY",
		}, cfg.Format)
	}

	authCli := a.clientFactory(setupCfg, a.httpClient)
	check, err := authCli.AuthCheck(context.Background())
	if err != nil {
		return a.writeClientError("auth.login", cfg.Format, err)
	}

	path, err := config.WriteSettings(settingsScope, setupCfg, overwrite)
	if err != nil {
		return a.writeFailure(output.Failure{
			Command: "auth.login",
			Kind:    output.ErrorKindUsage,
			Code:    "auth.login.write_failed",
			Message: err.Error(),
		}, cfg.Format)
	}

	return a.writeSuccess(output.Result{
		Command: "auth.login",
		Data: map[string]any{
			"path":        path,
			"scope":       scope,
			"status_code": check.StatusCode,
			"endpoint":    check.Endpoint,
		},
		Text: "auth saved to " + path,
	}, cfg.Format)
}

func (a *App) prompt(label string) (string, error) {
	if _, err := fmt.Fprint(a.stderr, label); err != nil {
		return "", err
	}

	value, err := a.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func (a *App) promptSecret(label string) (string, error) {
	if _, err := fmt.Fprint(a.stderr, label); err != nil {
		return "", err
	}

	file, ok := a.stdin.(*os.File)
	if ok && term.IsTerminal(int(file.Fd())) {
		value, err := term.ReadPassword(int(file.Fd()))
		if _, writeErr := fmt.Fprintln(a.stderr); writeErr != nil && err == nil {
			err = writeErr
		}
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(value)), nil
	}

	return a.prompt("")
}

func (a *App) reader() *bufio.Reader {
	if a.promptReader == nil {
		a.promptReader = bufio.NewReader(a.stdin)
	}
	return a.promptReader
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
		"  auth login      Persist API credentials to settings",
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

func (a *App) writeUsageText() int {
	lines := []string{
		"Twenty CRM CLI",
		"",
		"Usage:",
		"  twenty [--api-key KEY] [--base-url URL] [--format json|text] <command>",
		"",
		"Quick start:",
		"  twenty auth login --api-key <key> --base-url https://api.twenty.com",
		"  twenty auth check",
		"",
		"Common commands:",
		"  auth login       Save credentials to settings",
		"  auth check       Validate credentials and connectivity",
		"  people search    Search people",
		"  companies search Search companies",
		"  deals search     Search deals",
		"  meeting log      Log a meeting and follow-ups",
		"  call capture     Capture call notes and next steps",
		"  prospect import  Import prospects from JSON/JSONL",
		"",
		"More:",
		"  twenty --format json auth check",
		"  twenty version",
	}

	if err := output.WriteText(a.stdout, strings.Join(lines, "\n")); err != nil {
		fmt.Fprintf(a.stderr, "write error: %v\n", err)
		return int(output.ExitInternal)
	}
	return int(output.ExitOK)
}

func (a *App) writeHelpText(lines []string) int {
	if err := output.WriteText(a.stdout, strings.Join(lines, "\n")); err != nil {
		fmt.Fprintf(a.stderr, "write error: %v\n", err)
		return int(output.ExitInternal)
	}
	return int(output.ExitOK)
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

func explicitFormat(args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--format" && i+1 < len(args) {
			return true
		}
		if strings.HasPrefix(arg, "--format=") {
			return true
		}
	}
	return false
}

func flagProvided(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(current *flag.Flag) {
		if current.Name == name {
			found = true
		}
	})
	return found
}

func isHelpArg(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "--help", "-h", "help":
		return true
	default:
		return false
	}
}

func authHelpLines() []string {
	return []string{
		"Auth commands",
		"",
		"Usage:",
		"  twenty auth <command>",
		"",
		"Commands:",
		"  check  Validate credentials and connectivity",
		"  login  Save credentials to settings",
		"",
		"Examples:",
		"  twenty auth check",
		"  twenty auth login --api-key <key> --base-url https://api.twenty.com",
	}
}

func authCheckHelpLines() []string {
	return []string{
		"Validate credentials and connectivity",
		"",
		"Usage:",
		"  twenty auth check",
		"",
		"Notes:",
		"  Reads credentials from flags, environment, project settings, or home settings.",
		"  Use `twenty auth login` to write settings.",
	}
}

func authLoginHelpLines() []string {
	return []string{
		"Save credentials to settings",
		"",
		"Usage:",
		"  twenty auth login [--api-key KEY] [--base-url URL] [--scope home|project] [--overwrite]",
		"",
		"Flags:",
		"  --api-key KEY     Twenty API key",
		"  --base-url URL    Twenty base URL",
		"  --scope SCOPE     Settings scope: home or project",
		"  --overwrite       Replace an existing settings file",
		"",
		"Examples:",
		"  twenty auth login --api-key <key> --base-url https://api.twenty.com",
		"  twenty auth login --scope project --overwrite",
	}
}

func defaultIsTTY(r io.Reader) bool {
	file, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func errorCodeForConfigError(err error) string {
	var settingsErr *config.SettingsError
	if errors.As(err, &settingsErr) {
		return "config.settings_invalid"
	}
	return "cli.parse"
}

func errorMessageForConfigError(err error) string {
	var settingsErr *config.SettingsError
	if errors.As(err, &settingsErr) {
		return fmt.Sprintf("%s; repair it or run `twenty auth login --overwrite`", err.Error())
	}
	return err.Error()
}

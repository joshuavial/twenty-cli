package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jv/twenty-crm-cli/internal/config"
	"github.com/jv/twenty-crm-cli/internal/output"
)

type linkedTargetIDs struct {
	personID  string
	companyID string
	dealID    string
}

func (a *App) runWorkflow(cfg config.Config, token string, args []string) int {
	switch token {
	case "note":
		return a.runNote(cfg, args)
	case "task":
		return a.runTask(cfg, args)
	case "meeting":
		return a.runMeeting(cfg, args)
	case "call":
		return a.runCall(cfg, args)
	case "prospect":
		return a.runProspect(cfg, args)
	default:
		return a.writeFailure(output.Failure{
			Command: "cli",
			Kind:    output.ErrorKindUsage,
			Code:    "cli.unknown_command",
			Message: fmt.Sprintf("unknown command: %s", token),
		}, cfg.Format)
	}
}

func (a *App) runNote(cfg config.Config, args []string) int {
	if len(args) == 0 || args[0] != "add" {
		return a.writeFailure(output.Failure{
			Command: "note",
			Kind:    output.ErrorKindUsage,
			Code:    "note.usage",
			Message: "expected subcommand: add",
		}, cfg.Format)
	}

	title, markdown, targets, failure, ok := parseNoteArgs("note.add", args[1:])
	if !ok {
		return a.writeFailure(failure, cfg.Format)
	}

	cli := a.clientFactory(cfg, a.httpClient)
	note, err := createLinkedNote(cli, title, markdown, targets)
	if err != nil {
		return a.writeClientError("note.add", cfg.Format, err)
	}

	return a.writeSuccess(output.Result{
		Command: "note.add",
		Data:    note,
		Text:    "note added",
	}, cfg.Format)
}

func (a *App) runTask(cfg config.Config, args []string) int {
	if len(args) == 0 || args[0] != "create" {
		return a.writeFailure(output.Failure{
			Command: "task",
			Kind:    output.ErrorKindUsage,
			Code:    "task.usage",
			Message: "expected subcommand: create",
		}, cfg.Format)
	}

	taskSpec, failure, ok := parseTaskArgs("task.create", args[1:])
	if !ok {
		return a.writeFailure(failure, cfg.Format)
	}

	cli := a.clientFactory(cfg, a.httpClient)
	task, err := createLinkedTask(cli, taskSpec)
	if err != nil {
		return a.writeClientError("task.create", cfg.Format, err)
	}

	return a.writeSuccess(output.Result{
		Command: "task.create",
		Data:    task,
		Text:    "task created",
	}, cfg.Format)
}

func (a *App) runMeeting(cfg config.Config, args []string) int {
	if len(args) == 0 || args[0] != "log" {
		return a.writeFailure(output.Failure{
			Command: "meeting",
			Kind:    output.ErrorKindUsage,
			Code:    "meeting.usage",
			Message: "expected subcommand: log",
		}, cfg.Format)
	}

	spec, failure, ok := parseWorkflowLogArgs("meeting.log", "Meeting", args[1:])
	if !ok {
		return a.writeFailure(failure, cfg.Format)
	}

	return a.executeActivityWorkflow(cfg, "meeting.log", spec, "")
}

func (a *App) runCall(cfg config.Config, args []string) int {
	if len(args) == 0 || args[0] != "capture" {
		return a.writeFailure(output.Failure{
			Command: "call",
			Kind:    output.ErrorKindUsage,
			Code:    "call.usage",
			Message: "expected subcommand: capture",
		}, cfg.Format)
	}

	spec, failure, ok := parseWorkflowLogArgs("call.capture", "Call", args[1:])
	if !ok {
		return a.writeFailure(failure, cfg.Format)
	}

	return a.executeActivityWorkflow(cfg, "call.capture", spec, spec.dealStage)
}

type activityWorkflowSpec struct {
	title            string
	markdown         string
	targets          linkedTargetIDs
	nextSteps        []string
	createFollowups  bool
	dealStage        string
}

func (a *App) executeActivityWorkflow(cfg config.Config, command string, spec activityWorkflowSpec, dealStage string) int {
	cli := a.clientFactory(cfg, a.httpClient)

	note, err := createLinkedNote(cli, spec.title, spec.markdown, spec.targets)
	if err != nil {
		return a.writeClientError(command, cfg.Format, err)
	}

	var tasks []map[string]any
	if spec.createFollowups {
		for _, nextStep := range spec.nextSteps {
			task, err := createLinkedTask(cli, taskSpec{
				Title:   nextStep,
				Markdown: spec.markdown,
				Status:  "TODO",
				Targets: spec.targets,
			})
			if err != nil {
				return a.writeClientError(command, cfg.Format, err)
			}
			tasks = append(tasks, task)
		}
	}

	var deal map[string]any
	if dealStage != "" && spec.targets.dealID != "" {
		result, err := cli.UpdateRecord(context.Background(), "opportunities", "updateOpportunity", spec.targets.dealID, map[string]any{
			"stage": strings.ToUpper(dealStage),
		})
		if err != nil {
			return a.writeClientError(command, cfg.Format, err)
		}
		deal = result.Record
	}

	data := map[string]any{
		"note": note,
	}
	if len(tasks) > 0 {
		data["tasks"] = tasks
	}
	if len(deal) > 0 {
		data["deal"] = deal
	}

	return a.writeSuccess(output.Result{
		Command: command,
		Data:    data,
		Text:    "workflow captured",
	}, cfg.Format)
}

func parseNoteArgs(command string, args []string) (string, string, linkedTargetIDs, output.Failure, bool) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})

	var title string
	var body string
	var bodyFile string
	var targets linkedTargetIDs

	fs.StringVar(&title, "title", "", "Note title")
	fs.StringVar(&body, "body", "", "Note markdown body")
	fs.StringVar(&bodyFile, "body-file", "", "Path to note body")
	fs.StringVar(&targets.personID, "person-id", "", "Linked person ID")
	fs.StringVar(&targets.companyID, "company-id", "", "Linked company ID")
	fs.StringVar(&targets.dealID, "deal-id", "", "Linked deal ID")

	if err := fs.Parse(args); err != nil {
		return "", "", linkedTargetIDs{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: "cli.parse", Message: err.Error()}, false
	}

	markdown, err := readInlineOrFile(body, bodyFile)
	if err != nil {
		return "", "", linkedTargetIDs{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: command + ".body", Message: err.Error()}, false
	}
	if markdown == "" {
		return "", "", linkedTargetIDs{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: command + ".missing_body", Message: "missing required body: --body or --body-file"}, false
	}
	if title == "" {
		title = "Note"
	}

	return title, markdown, targets, output.Failure{}, true
}

type taskSpec struct {
	Title    string
	Markdown string
	DueAt    string
	Status   string
	Targets  linkedTargetIDs
}

func parseTaskArgs(command string, args []string) (taskSpec, output.Failure, bool) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})

	var spec taskSpec
	var bodyFile string
	fs.StringVar(&spec.Title, "title", "", "Task title")
	fs.StringVar(&spec.Markdown, "body", "", "Task markdown body")
	fs.StringVar(&bodyFile, "body-file", "", "Path to task body")
	fs.StringVar(&spec.DueAt, "due-at", "", "RFC3339 due time")
	fs.StringVar(&spec.Status, "status", "TODO", "Task status")
	fs.StringVar(&spec.Targets.personID, "person-id", "", "Linked person ID")
	fs.StringVar(&spec.Targets.companyID, "company-id", "", "Linked company ID")
	fs.StringVar(&spec.Targets.dealID, "deal-id", "", "Linked deal ID")

	if err := fs.Parse(args); err != nil {
		return taskSpec{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: "cli.parse", Message: err.Error()}, false
	}
	if spec.Title == "" {
		return taskSpec{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: command + ".missing_title", Message: "missing required flag: --title"}, false
	}
	body, err := readInlineOrFile(spec.Markdown, bodyFile)
	if err != nil {
		return taskSpec{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: command + ".body", Message: err.Error()}, false
	}
	spec.Markdown = body
	spec.Status = strings.ToUpper(spec.Status)
	if spec.DueAt != "" {
		if _, err := time.Parse(time.RFC3339, spec.DueAt); err != nil {
			return taskSpec{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: command + ".due_at", Message: "invalid --due-at, expected RFC3339"}, false
		}
	}

	return spec, output.Failure{}, true
}

func parseWorkflowLogArgs(command, defaultTitle string, args []string) (activityWorkflowSpec, output.Failure, bool) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(ioDiscard{})

	var spec activityWorkflowSpec
	var bodyFile string
	fs.StringVar(&spec.title, "title", defaultTitle, "Activity title")
	fs.StringVar(&spec.markdown, "body", "", "Body markdown")
	fs.StringVar(&bodyFile, "body-file", "", "Body file path")
	fs.StringVar(&spec.targets.personID, "person-id", "", "Linked person ID")
	fs.StringVar(&spec.targets.companyID, "company-id", "", "Linked company ID")
	fs.StringVar(&spec.targets.dealID, "deal-id", "", "Linked deal ID")
	fs.StringVar(&spec.dealStage, "deal-stage", "", "Optional deal stage update")
	fs.BoolVar(&spec.createFollowups, "create-followups", false, "Create tasks from next steps")
	var nextSteps multiString
	fs.Var(&nextSteps, "next-step", "Next step task title")

	if err := fs.Parse(args); err != nil {
		return activityWorkflowSpec{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: "cli.parse", Message: err.Error()}, false
	}
	body, err := readInlineOrFile(spec.markdown, bodyFile)
	if err != nil {
		return activityWorkflowSpec{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: command + ".body", Message: err.Error()}, false
	}
	if body == "" {
		return activityWorkflowSpec{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: command + ".missing_body", Message: "missing required body: --body or --body-file"}, false
	}
	spec.markdown = body
	spec.nextSteps = nextSteps
	if spec.createFollowups && len(spec.nextSteps) == 0 {
		return activityWorkflowSpec{}, output.Failure{Command: command, Kind: output.ErrorKindUsage, Code: command + ".missing_next_steps", Message: "use --next-step when --create-followups is set"}, false
	}
	if spec.dealStage != "" {
		spec.dealStage = strings.ToUpper(spec.dealStage)
	}

	return spec, output.Failure{}, true
}

func createLinkedNote(cli twentyClient, title, markdown string, targets linkedTargetIDs) (map[string]any, error) {
	result, err := cli.CreateRecord(context.Background(), "notes", "createNote", map[string]any{
		"title":  title,
		"bodyV2": map[string]any{"markdown": markdown},
	})
	if err != nil {
		return nil, err
	}

	if hasAnyTarget(targets) {
		_, err = cli.CreateRecord(context.Background(), "noteTargets", "createNoteTarget", targetPayload("noteId", result.Record["id"], targets))
		if err != nil {
			return nil, err
		}
	}

	return result.Record, nil
}

func createLinkedTask(cli twentyClient, spec taskSpec) (map[string]any, error) {
	payload := map[string]any{
		"title":  spec.Title,
		"status": spec.Status,
	}
	if spec.Markdown != "" {
		payload["bodyV2"] = map[string]any{"markdown": spec.Markdown}
	}
	if spec.DueAt != "" {
		payload["dueAt"] = spec.DueAt
	}

	result, err := cli.CreateRecord(context.Background(), "tasks", "createTask", payload)
	if err != nil {
		return nil, err
	}

	if hasAnyTarget(spec.Targets) {
		_, err = cli.CreateRecord(context.Background(), "taskTargets", "createTaskTarget", targetPayload("taskId", result.Record["id"], spec.Targets))
		if err != nil {
			return nil, err
		}
	}

	return result.Record, nil
}

func targetPayload(primaryField string, primaryValue any, targets linkedTargetIDs) map[string]any {
	payload := map[string]any{primaryField: primaryValue}
	if targets.personID != "" {
		payload["targetPersonId"] = targets.personID
	}
	if targets.companyID != "" {
		payload["targetCompanyId"] = targets.companyID
	}
	if targets.dealID != "" {
		payload["targetOpportunityId"] = targets.dealID
	}
	return payload
}

func hasAnyTarget(targets linkedTargetIDs) bool {
	return targets.personID != "" || targets.companyID != "" || targets.dealID != ""
}

func readInlineOrFile(inline, path string) (string, error) {
	if inline != "" && path != "" {
		return "", fmt.Errorf("use only one of inline text or file input")
	}
	if path != "" {
		if path == "-" {
			data, err := os.ReadFile("/dev/stdin")
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(data)), nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return strings.TrimSpace(inline), nil
}

type multiString []string

func (m *multiString) String() string { return strings.Join(*m, ",") }
func (m *multiString) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func (a *App) runProspect(cfg config.Config, args []string) int {
	if len(args) == 0 || args[0] != "import" {
		return a.writeFailure(output.Failure{
			Command: "prospect",
			Kind:    output.ErrorKindUsage,
			Code:    "prospect.usage",
			Message: "expected subcommand: import",
		}, cfg.Format)
	}

	return a.runProspectImport(cfg, args[1:])
}

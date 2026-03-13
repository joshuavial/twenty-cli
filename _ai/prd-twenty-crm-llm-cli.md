# PRD: Twenty CRM LLM CLI

## Document Status

- Status: Draft
- Date: 2026-03-14
- Depth: Standard
- Related research: [_ai/research/twenty-crm-api/summary.md](/Users/jv/gt/twenty_crm_cli/crew/conduit/_ai/research/twenty-crm-api/summary.md)

## Problem Statement

We need a CLI for Twenty CRM that is optimized for use by LLM agents, not just humans at a terminal. Existing API access is flexible but raw: an LLM must understand auth, schema discovery, object naming, pagination, filters, and mutation safety on its own. That increases prompt complexity, token cost, and the risk of malformed or unsafe API calls. The product goal is a reliable command-line interface that gives LLMs a narrow, deterministic, machine-readable contract for reading and mutating Twenty CRM data safely.

## Proposed Solution

Build a stateless CLI wrapper around the Twenty CRM API that is organized around user jobs, not API resource design.

The CLI should expose user-facing commands such as:

- `twenty person create`
- `twenty people search`
- `twenty company create`
- `twenty deal create`
- `twenty deal advance`
- `twenty meeting log`
- `twenty call capture`
- `twenty note add`
- `twenty task create`
- `twenty prospect import`

The tool should internally translate these intent-level commands into whatever Twenty API calls are required, including object lookups, metadata resolution, record creation, relation linking, and follow-up task generation.

The CLI should default to LLM-friendly behavior:

- JSON output by default
- stable field names and envelope shapes
- low-noise stdout
- actionable stderr errors
- predictable exit codes
- explicit dry-run and confirmation requirements for destructive actions
- concise help focused on user tasks rather than API terminology

## Goals

- Let an LLM perform common CRM tasks without writing raw HTTP requests or reasoning directly about Twenty API structure.
- Expose commands in the way a salesperson, founder, operator, or assistant would think about the work.
- Reduce prompt burden by exposing opinionated commands and workflow-oriented argument patterns.
- Make read operations easy and deterministic.
- Make write operations safe, inspectable, and reversible where possible.
- Support custom-object workspaces by using metadata discovery internally rather than exposing schema complexity at the command boundary.

## Non-Goals

- Replacing the full Twenty API surface on day one.
- Building a TUI or interactive shell as the primary interface.
- Handling webhook receipt or long-running background daemons in the initial release.
- Exposing the raw Twenty API model as the primary mental model for the CLI.

## Users

### Primary User

LLM agents operating inside coding assistants, terminal copilots, or automation frameworks.

### Secondary Users

- Developers using the CLI directly
- Operators writing scripts for imports and sync jobs
- Humans debugging or validating LLM actions

## User Stories

- As an LLM agent, I want to create a person, company, or deal using domain language so that I do not have to infer the underlying API route structure.
- As an LLM agent, I want to capture the outcome of a call or meeting and have the tool log notes, update the relationship, and create next steps.
- As an LLM agent, I want to search for people and companies before creating them so that I avoid duplicates.
- As an LLM agent, I want list and get commands to return clean JSON with stable envelopes so that I can parse results without brittle heuristics.
- As an LLM agent, I want write commands to validate payloads before sending so that I can avoid malformed writes.
- As an LLM agent, I want destructive commands to require explicit intent markers so that accidental deletes are less likely.
- As a user, I want prospecting and bulk-add flows optimized for high-volume CRM work so that I can add many contacts and companies efficiently.
- As a developer, I want the CLI internals to map cleanly to the Twenty API so that debugging against the underlying platform stays straightforward even if the public command model is user-oriented.

## Scope

### In Scope

- CLI executable for Twenty CRM
- API key auth via environment variable and explicit flags
- Workspace schema discovery
- User-oriented command groups for high-volume CRM workflows:
  - people and person management
  - company management
  - deal management
  - meeting and call capture
  - notes and tasks
  - prospecting and bulk import
- Under-the-hood translation to REST and metadata APIs as needed
- Search and retrieval flows for common business entities
- Creation and update flows for common sales workflows
- Query controls where needed, but exposed as task-oriented flags first
- LLM-oriented output modes:
  - JSON default
  - compact text mode only when explicitly requested
- Safety controls:
  - dry-run
  - confirmation token or flag for destructive commands
  - clear mutation preview
- Error normalization:
  - stable error codes
  - surfaced API status/body details
- Documentation for LLM usage patterns

### Out of Scope

- Full coverage of every Twenty metadata mutation in v1
- Webhook consumer/server features
- OAuth login flows
- Rich interactive prompts
- Multi-workspace orchestration beyond selecting one target per invocation
- A purely API-shaped CLI where every command directly mirrors raw REST or GraphQL nouns

## Functional Requirements

### 1. Authentication

- Support `TWENTY_API_KEY` and `TWENTY_BASE_URL`.
- Default cloud base URL to `https://api.twenty.com`.
- Support self-hosted instances by override.
- Provide a `whoami` or `auth check` command that validates connectivity and permissions.

### 2. Schema Discovery And Internal Mapping

- Provide commands to list objects available in the workspace.
- Provide commands to inspect object fields, types, and relation metadata.
- Use metadata endpoints where needed to support custom objects and fields.
- Maintain an internal mapping layer so user-facing commands like `person`, `people`, `deal`, `meeting`, and `task` resolve to the correct Twenty objects and fields.
- Cache schema locally only if the cache can be refreshed explicitly and bypassed safely.

### 3. High-Frequency Entity Commands

- Support user-facing entity commands such as:
  - `twenty people search`
  - `twenty person create`
  - `twenty person update`
  - `twenty companies search`
  - `twenty company create`
  - `twenty deals list`
  - `twenty deal create`
  - `twenty deal update`
- Support fetching a single person, company, or deal by ID or other common lookup fields where practical.
- Support search, ordering, limit, and pagination in a way that is easy for an LLM to call.
- Support optional relation expansion when it improves workflow results.

### 4. Workflow Commands For Calls, Meetings, Notes, And Follow-Up

- Support a `meeting log` flow that can:
  - attach notes to a person, company, or deal
  - associate the meeting with the right records
  - optionally create follow-up tasks
- Support a `call capture` flow that can:
  - record the summary of a call
  - update a contact or deal status
  - extract and create next steps
- Support note and task creation as standalone commands.
- Support dry-run validation where the CLI can validate command shape and show intended API requests before mutation.

### 5. Prospecting And High-Volume Input

- Support bulk-oriented prospecting commands for adding many people or companies.
- Support `--file` and stdin input to avoid shell-escaping failures.
- Support duplicate-check or lookup-first behavior before creating records.
- Support import summaries with created, updated, skipped, and failed counts.

### 6. LLM-Oriented Contracts

- Default stdout must be valid JSON unless `--format text` is requested.
- The JSON envelope should be stable across commands, for example:
  - `ok`
  - `command`
  - `object`
  - `data`
  - `page_info`
  - `error`
  - `meta`
- Errors must be deterministic and concise.
- Help output must be short enough to fit comfortably in tool-usage contexts.
- Commands should favor explicit long-form flags over ambiguous positional arguments.
- Commands should use domain language first and hide API structure unless the caller asks for debug detail.

### 7. Safety And Guardrails

- Destructive operations must require one of:
  - `--confirm`
  - a typed confirmation token
  - a higher-friction `--execute` after `--dry-run`
- Batch mutations must expose count and object preview before execution.
- The CLI must reject obviously invalid combinations before making an API call.
- Exit codes should separate user error, auth failure, API rejection, and internal failure.

## UX Principles For LLM Callers

- Be boring and parseable.
- Minimize prose on stdout.
- Keep one command responsible for one operation.
- Prefer explicit flags to magical defaults.
- Organize the CLI around user intent, not API implementation.
- Translate from user concepts into API operations internally.
- Return enough metadata for follow-up calls, especially IDs and pagination cursors.

## Priority Workflows

The first release should optimize for the highest-frequency CRM jobs:

1. Adding a contact
2. Updating a contact after learning new information
3. Creating and updating a deal
4. Logging a meeting or a call
5. Capturing a large block of meeting notes and turning it into:
   - CRM notes
   - contact updates
   - deal updates
   - follow-up tasks
6. Prospecting and bulk-adding people and companies
7. Searching the CRM before creating records to avoid duplication

These workflows should shape the command model before lower-level generic coverage does.

## Example Command Shapes

These are illustrative and may evolve during scoping:

```bash
twenty people search --name "Ada Lovelace"
twenty person create --first-name Ada --last-name Lovelace --email ada@example.com
twenty person update --id 123 --title "Founder" --company "Analytical Engines"
twenty company create --name "Analytical Engines"
twenty deals list --stage new
twenty deal create --name "Enterprise Expansion" --company "Analytical Engines" --value 25000
twenty deal advance --id 456 --stage proposal
twenty meeting log --person 123 --notes-file notes.md --create-followups
twenty call capture --person 123 --summary-file call.md --update-contact --update-deal --create-tasks
twenty prospect import --file leads.json --lookup-first --dry-run
```

## Constraints

- The API is workspace-shaped, so the CLI cannot hard-code a static universal schema.
- Public docs are stronger for core REST than for Metadata REST; metadata GraphQL may be required.
- Rate limits currently appear to be 100 calls per minute and 60 records per batch.
- LLMs are error-prone with quoting and shell escaping, so payload input paths and stdin support may matter.
- Human-readable text output cannot be the primary contract if the main caller is an LLM.
- User-facing concepts like "meeting", "call", or "next step" may require multiple API objects and relations under the hood.

## Dependencies

- Twenty API key management in the target workspace
- Verified base URL strategy for cloud and self-hosted deployments
- Metadata discovery layer for custom objects
- A CLI framework and JSON serialization strategy
- Test fixtures or a sandbox workspace for integration testing

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Metadata API contract is under-documented | Schema discovery becomes brittle | Prefer live-playground validation and wrap metadata access behind a narrow internal adapter |
| LLMs produce malformed JSON or shell escaping | High error rate for writes | Support `--file`, `@stdin`, and preflight validation |
| CLI becomes a thin wrapper over too many raw API concepts | Poor usability for LLMs | Keep a small opinionated command set and stable output envelopes |
| Destructive commands are called accidentally | Data loss | Require explicit confirmations and dry-run previews |
| Workspace-specific custom objects break assumptions | Query failures | Make object discovery a first-class workflow, not an afterthought |
| API behavior differs slightly between repo tests and deployed cloud | Runtime drift | Add integration tests against a real workspace and document version assumptions |

## Success Metrics

- An LLM can complete common workflows with no raw HTTP:
  - search for a contact or company
  - add a contact
  - update a contact
  - create a deal
  - advance a deal
  - log a meeting
  - capture a call and generate follow-up tasks
- At least 95% of successful command invocations produce valid JSON output with no extra stdout noise.
- Destructive operations require explicit confirmation in 100% of cases.
- A new workspace with custom objects can be queried after schema discovery without code changes.
- Integration test coverage exists for auth, object discovery, CRUD, pagination, error normalization, and mutation safety.

## Proposal Breakdown

### Proposal 1: CLI Skeleton And Auth Contract

**Summary**: Establish the executable, config model, output envelope, and auth/base-URL behavior.

**Scope**:
- Choose CLI framework and command structure
- Implement config resolution from env and flags
- Implement JSON/text format handling
- Implement standardized error model and exit codes
- Add `auth check` or equivalent connectivity command

**Dependencies**: None

### Proposal 2: Domain Command Model And Schema Mapping

**Summary**: Define the user-facing command language and map it to workspace-specific Twenty schema details.

**Scope**:
- Define the domain-oriented command groups and nouns
- Implement schema lookup and mapping for user-facing entities
- Add metadata adapter using the safest available API surface
- Normalize schema output for LLM callers and internal resolvers

**Dependencies**: Proposal 1

### Proposal 3: Search And Entity Read Flows

**Summary**: Implement user-oriented search and read commands for people, companies, and deals.

**Scope**:
- `people search`
- `person get`
- `companies search`
- `company get`
- `deals list`
- `deal get`
- lookup-first behavior and pagination support
- pagination envelope and next-step metadata

**Dependencies**: Proposal 1, Proposal 2

### Proposal 4: Contact, Company, And Deal Mutation Flows

**Summary**: Implement create and update commands for the main CRM entities with dry-run and safety behavior.

**Scope**:
- `person create`
- `person update`
- `company create`
- `company update`
- `deal create`
- `deal update`
- `deal advance`
- payload validation
- mutation preview
- safety checks and deterministic errors

**Dependencies**: Proposal 1, Proposal 2, Proposal 3

### Proposal 5: Meeting, Call, Note, And Task Workflows

**Summary**: Implement the high-value post-call and post-meeting workflows that turn raw notes into CRM actions.

**Scope**:
- `meeting log`
- `call capture`
- `note add`
- `task create`
- next-step extraction and task creation
- relation linking to people, companies, and deals

**Dependencies**: Proposal 1, Proposal 2, Proposal 3, Proposal 4

### Proposal 6: Prospecting And Bulk Ingestion

**Summary**: Add high-volume contact and company ingestion workflows optimized for prospecting.

**Scope**:
- `prospect import`
- file and stdin input modes
- duplicate lookup before create
- created or updated or skipped result summaries
- resumable error reporting

**Dependencies**: Proposal 1, Proposal 2, Proposal 3, Proposal 4

### Proposal 7: Integration Test Harness

**Summary**: Add end-to-end verification against a Twenty workspace or controlled fixture environment.

**Scope**:
- auth tests
- schema mapping tests
- search tests
- entity mutation tests
- workflow command tests
- pagination tests
- safety behavior tests
- contract snapshots for JSON envelopes

**Dependencies**: Proposal 1, Proposal 2, Proposal 3, Proposal 4, Proposal 5, Proposal 6

## Open Questions

- Should metadata mutation commands be included in v1 or limited to read-only schema discovery?
- Should the CLI expose raw GraphQL passthrough for escape hatches, or is that anti-goal for LLM simplicity?
- Should batch update/upsert be included in MVP, or only batch create after live validation?
- Should command style prefer `twenty person create` or `twenty people create`, and where should singular vs plural be standardized?
- How much AI-specific actioning should be inside the CLI itself versus handled by the calling LLM before invocation?
- Should `meeting log` and `call capture` accept raw notes only, or also structured outcome fields for more deterministic behavior?

## Next Steps

Run `/scope` against the proposals in order:

1. `/scope cli-skeleton-and-auth-contract`
2. `/scope domain-command-model-and-schema-mapping`
3. `/scope search-and-entity-read-flows`
4. `/scope contact-company-and-deal-mutation-flows`
5. `/scope meeting-call-note-and-task-workflows`
6. `/scope prospecting-and-bulk-ingestion`
7. `/scope integration-test-harness`

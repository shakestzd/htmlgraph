# CLI Reference

All commands are invoked as `htmlgraph <command>`. Run `htmlgraph help --compact` for a quick summary, or `htmlgraph <command> --help` for detailed usage on any command.

---

## Work Items

Commands for managing the core work item types. All four types share the same lifecycle subcommands.

| Command | Description |
|---------|-------------|
| `feature [create\|show\|start\|complete\|list\|add-step\|delete]` | Feature work items |
| `bug [create\|show\|start\|complete\|list\|add-step\|delete]` | Bug tracking |
| `spike [create\|show\|start\|complete\|list\|add-step\|delete]` | Time-boxed investigation spikes |
| `track [create\|show\|start\|complete\|list\|add-step\|delete]` | Multi-feature tracks (initiatives) |

### Common subcommands

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `create` | `htmlgraph feature create "Title" --track <trk-id> --description "..."` | Create a new work item |
| `show` | `htmlgraph feature show <id>` | Display work item details |
| `start` | `htmlgraph feature start <id>` | Mark as in-progress and set as active |
| `complete` | `htmlgraph feature complete <id>` | Mark as done |
| `list` | `htmlgraph feature list [--status todo\|in-progress\|done]` | List work items with optional status filter |
| `add-step` | `htmlgraph feature add-step <id> "Step description"` | Add an implementation step |
| `delete` | `htmlgraph feature delete <id>` | Delete a work item |

!!! note "Required flags"
    `feature create` and `bug create` require `--track <trk-id>` and `--description "..."`.

---

## Search & Status

Quick commands for finding work items and checking project state.

| Command | Usage | Description |
|---------|-------|-------------|
| `find` | `htmlgraph find <query>` | Search work items by title or ID |
| `wip` | `htmlgraph wip` | Show all in-progress work items |
| `status` | `htmlgraph status` | Quick project status summary |
| `snapshot` | `htmlgraph snapshot [--summary]` | Full project overview with counts and details |

---

## Planning

Commands for creating, reviewing, and executing structured CRISPI plans.

| Command | Usage | Description |
|---------|-------|-------------|
| `plan create` | `htmlgraph plan create "Title" --track <trk-id>` | Create a new plan |
| `plan create-yaml` | `htmlgraph plan create-yaml "Title" --track <trk-id>` | Create a v2 YAML plan file |
| `plan show` | `htmlgraph plan show <id>` | Display plan details |
| `plan start` | `htmlgraph plan start <id>` | Mark plan as in-progress |
| `plan complete` | `htmlgraph plan complete <id>` | Mark plan as done |
| `plan list` | `htmlgraph plan list` | List all plans |
| `plan list-yaml` | `htmlgraph plan list-yaml` | List all YAML plans sorted by created_at |
| `plan generate` | `htmlgraph plan generate <trk-id>` | Generate a CRISPI YAML plan for a track |
| `plan rewrite-yaml` | `htmlgraph plan rewrite-yaml <id> --file <path>` | Validated atomic update of plan YAML |
| `plan validate-yaml` | `htmlgraph plan validate-yaml <id>` | Validate a YAML plan's schema |

### v2 Slice Lifecycle

Commands for per-slice review and incremental promotion (v2 plans only).

| Command | Usage | Description |
|---------|-------|-------------|
| `plan approve-slice` | `htmlgraph plan approve-slice <plan-id> <num>` | Set `approval_status=approved` for a slice |
| `plan reject-slice` | `htmlgraph plan reject-slice <plan-id> <num> [--changes-requested]` | Set `approval_status=rejected` (or `changes_requested`) |
| `plan answer-slice-question` | `htmlgraph plan answer-slice-question <plan-id> <num> <question-id> <answer-key>` | Record answer to a slice-local question |
| `plan set-slice-status` | `htmlgraph plan set-slice-status <plan-id> <num> <status>` | Set execution status (`not_started\|promoted\|in_progress\|done\|blocked\|superseded`) |
| `plan promote-slice` | `htmlgraph plan promote-slice <plan-id> <num> [--waive-deps]` | Promote an approved slice to a feature work item |

---

## Specifications & Quality

Commands for feature specs, test generation, code review, and quality enforcement.

| Command | Usage | Description |
|---------|-------|-------------|
| `spec` | `htmlgraph spec [generate\|show] <feature-id>` | Generate or view feature specifications |
| `tdd` | `htmlgraph tdd <feature-id>` | Generate test stubs from spec acceptance criteria |
| `review` | `htmlgraph review` | Structured diff summary against base branch |
| `compliance` | `htmlgraph compliance <feature-id>` | Score implementation against spec |
| `check` | `htmlgraph check` | Run automated quality gate checks |
| `health` | `htmlgraph health` | Code health metrics (module sizes, function lengths) |

---

## Sessions & Observability

Commands for session management, analytics, and work item relationships.

| Command | Usage | Description |
|---------|-------|-------------|
| `session list` | `htmlgraph session list` | List recorded sessions |
| `session show` | `htmlgraph session show <id>` | Display session details and tool calls |
| `analytics summary` | `htmlgraph analytics summary` | Work analytics overview |
| `analytics velocity` | `htmlgraph analytics velocity` | Development velocity insights |
| `link add` | `htmlgraph link add <from-id> <to-id> --type <type>` | Create a typed edge between work items |
| `link remove` | `htmlgraph link remove <from-id> <to-id>` | Remove an edge |
| `link list` | `htmlgraph link list <id>` | List edges for a work item |

---

## Data Management

Commands for data import, export, and index maintenance.

| Command | Usage | Description |
|---------|-------|-------------|
| `batch apply` | `htmlgraph batch apply <file.yaml>` | Apply bulk work item operations from YAML |
| `batch export` | `htmlgraph batch export` | Export work items to YAML |
| `ingest` | `htmlgraph ingest` | Ingest Claude Code session transcripts (JSONL) |
| `backfill` | `htmlgraph backfill [feature-files\|tool-calls-feature]` | Rebuild derived tables |
| `reindex` | `htmlgraph reindex` | Sync HTML work items to SQLite index |

---

## Development & Operations

Commands for autonomous development, building, serving, agent configuration, and maintenance.

| Command | Usage | Description |
|---------|-------|-------------|
| `claude` | `htmlgraph claude [--dev] [--continue\|--resume <session-id>]` | Launch Claude Code with HtmlGraph plugin; `--resume <id>` resumes a specific prior session |
| `yolo` | `htmlgraph yolo --feature <id> [--track <id>] [--resume <session-id>]` | Autonomous dev mode with engineering guardrails |
| `build` | `htmlgraph build` | Build Go binary (dev workflow) |
| `serve` | `htmlgraph serve` | Start local dashboard server at `localhost:4000` |
| `agent-init` | `htmlgraph agent-init` | Output shared agent context (safety, attribution, quality gates) |
| `statusline` | `htmlgraph statusline` | OMP/Starship prompt integration |
| `upgrade` / `update` | `htmlgraph upgrade [--check] [--version 0.54.9]` | Self-update CLI from GitHub releases |

---

## Work Item Types

| Type | Prefix | Purpose |
|------|--------|---------|
| Feature | `feat-` | Units of deliverable work |
| Bug | `bug-` | Defects to fix |
| Spike | `spk-` | Time-boxed investigations |
| Track | `trk-` | Initiatives grouping related work |
| Plan | `plan-` | CRISPI implementation plans |

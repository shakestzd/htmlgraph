# Session Ingestion - Parallel Execution Plan

**Track:** trk-97f85b3b
**Features:** 9 tasks across 3 waves
**Parallelization:** Max 5 concurrent worktrees

## Architecture Overview

### Unified Data Model

All ingesters produce `IngestedMessage` objects stored in a new `ingested_messages` table:

```python
@dataclass
class IngestedMessage:
    id: str                    # Unique message ID
    session_id: str            # Original session ID from tool
    tool_source: str           # "claude_code" | "gemini_cli" | "copilot_cli" | "codex_cli" | "opencode" | "cursor"
    role: str                  # "user" | "assistant" | "system" | "tool"
    content: str               # Message text content
    timestamp: datetime        # Message timestamp
    parent_id: str | None      # Parent message ID (for hierarchy)
    agent_id: str | None       # Subagent identifier
    is_sidechain: bool         # True if from a subagent
    model: str | None          # Model used (e.g. "claude-opus-4-6")
    tool_name: str | None      # Tool name if tool call
    tool_input: str | None     # Tool input JSON
    tool_output: str | None    # Tool output text
    tokens_input: int | None   # Input tokens
    tokens_output: int | None  # Output tokens
    cost_usd: float | None     # Cost in USD (if available)
    project_path: str | None   # Project directory
    metadata: dict             # Tool-specific extra data
```

### Base Ingester Interface

```python
class BaseIngester(ABC):
    @abstractmethod
    def discover_sessions(self) -> list[SessionInfo]: ...

    @abstractmethod
    def parse_session(self, path: Path) -> list[IngestedMessage]: ...

    def ingest_all(self, incremental: bool = True) -> IngestResult: ...
    def ingest_session(self, session_id: str) -> IngestResult: ...
```

### SQLite Schema Extensions

```sql
CREATE TABLE ingested_sessions (
    session_id TEXT PRIMARY KEY,
    tool_source TEXT NOT NULL,
    project_path TEXT,
    start_time DATETIME,
    end_time DATETIME,
    message_count INTEGER DEFAULT 0,
    first_prompt TEXT,
    summary TEXT,
    git_branch TEXT,
    ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    source_path TEXT NOT NULL,
    source_mtime REAL  -- For incremental ingestion
);

CREATE TABLE ingested_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES ingested_sessions(session_id),
    role TEXT NOT NULL,
    content TEXT,
    timestamp DATETIME,
    parent_id TEXT,
    agent_id TEXT,
    is_sidechain BOOLEAN DEFAULT FALSE,
    model TEXT,
    tool_name TEXT,
    tool_input JSON,
    tool_output TEXT,
    tokens_input INTEGER,
    tokens_output INTEGER,
    cost_usd REAL,
    metadata JSON,
    FOREIGN KEY (parent_id) REFERENCES ingested_messages(id)
);

CREATE INDEX idx_im_session ON ingested_messages(session_id);
CREATE INDEX idx_im_timestamp ON ingested_messages(timestamp);
CREATE INDEX idx_im_tool ON ingested_messages(tool_name);
CREATE INDEX idx_im_agent ON ingested_messages(agent_id);
```

---

## Wave 0: Blockers (2 parallel tasks)

### Task 0: Base Ingester Framework + Claude Code Parser
**Feature:** feat-717bd042
**Branch:** feature/ingester-base-claude
**Priority:** blocker
**Dependencies:** none

**Files to CREATE:**
- `src/python/htmlgraph/ingest/__init__.py`
- `src/python/htmlgraph/ingest/base.py` (BaseIngester, IngestedMessage, SessionInfo, IngestResult)
- `src/python/htmlgraph/ingest/models.py` (dataclasses)
- `src/python/htmlgraph/ingest/claude_code.py` (ClaudeCodeIngester)
- `src/python/htmlgraph/ingest/registry.py` (ingester registry)
- `tests/ingest/__init__.py`
- `tests/ingest/test_base.py`
- `tests/ingest/test_claude_code.py`

**Files to MODIFY:**
- `src/python/htmlgraph/db/schema.py` (add ingested_sessions + ingested_messages tables)
- `src/python/htmlgraph/cli/` (add `htmlgraph ingest` command skeleton)
- `pyproject.toml` (if any new deps needed)

**Claude Code Parser Details:**
- Discovery: Read `~/.claude/projects/*/sessions-index.json` for fast session listing
- Parse: Line-by-line JSONL from `<session-uuid>.jsonl`
- Record types to handle: `user`, `assistant`, `system`, `progress`, `queue-operation`
- Skip: `file-history-snapshot` (no conversation data)
- Hierarchy: Use `parentUuid` field for parent-child chains
- Subagents: `isSidechain: true` + `agentId` field
- Token usage: `assistant.message.usage` (input_tokens, output_tokens, cache_*)
- Compaction: `type=system, subtype=compact_boundary` marks compaction points
- Subagent transcripts: `<session>/<sessionId>/subagents/agent-<agentId>.jsonl`

**Acceptance Criteria:**
- [ ] BaseIngester ABC with discover/parse/ingest methods
- [ ] IngestedMessage dataclass with all fields
- [ ] SQLite schema migration for new tables
- [ ] ClaudeCodeIngester discovers sessions from sessions-index.json
- [ ] ClaudeCodeIngester parses JSONL with correct hierarchy
- [ ] Incremental ingestion (tracks source_mtime)
- [ ] `uv run htmlgraph ingest --tool claude` works
- [ ] Tests pass: `uv run pytest tests/ingest/`

---

### Task 1: Hook Hierarchy Fix
**Feature:** feat-c08cdb8e
**Branch:** feature/hook-hierarchy
**Priority:** blocker
**Dependencies:** none

**Files to CREATE:**
- `packages/claude-plugin/hooks/scripts/subagent-start.py` (may already exist, wire it)

**Files to MODIFY:**
- `packages/claude-plugin/hooks/hooks.json` (add SubagentStart entry)
- `src/python/htmlgraph/hooks/constants.py` (remove SUBAGENT_SUFFIXES hack)
- `src/python/htmlgraph/hooks/context.py` (read agent_id/agent_type from stdin JSON)
- `packages/claude-plugin/hooks/scripts/posttooluse-integrator.py` (use agent_id for parent linkage)
- `packages/claude-plugin/hooks/scripts/pretooluse-integrator.py` (use agent_id)
- `packages/claude-plugin/hooks/scripts/subagent-stop.py` (capture last_assistant_message, agent_transcript_path)
- `src/python/htmlgraph/db/schema.py` (ensure agent_id/agent_type columns exist on agent_events)

**Implementation Details:**
- Claude Code now injects `agent_id` and `agent_type` into ALL hook inputs when inside a subagent
- Read these from stdin JSON in hook scripts (they're in the hook_input)
- Store agent_id and agent_type on the agent_events record
- Use agent_id as parent_agent_id for child events
- SubagentStop now provides: agent_id, agent_type, agent_transcript_path, last_assistant_message
- The SUBAGENT_SUFFIXES list in constants.py is obsolete — delete it
- context.py should use agent_id field instead of parsing session ID strings

**Acceptance Criteria:**
- [ ] SubagentStart hook wired in hooks.json and executes
- [ ] All PostToolUse/PreToolUse hooks read agent_id from stdin
- [ ] agent_id and agent_type stored on agent_events records
- [ ] Parent-child linkage works (subagent events linked to parent Agent call)
- [ ] SUBAGENT_SUFFIXES removed from constants.py
- [ ] SubagentStop captures last_assistant_message and transcript path
- [ ] Dashboard shows events grouped under their parent agent task
- [ ] Tests pass: `uv run pytest tests/hooks/`

---

## Wave 1: Core Implementation (5 parallel tasks)

### Task 2: Gemini CLI Parser
**Feature:** feat-40836773
**Branch:** feature/ingester-gemini
**Priority:** high
**Dependencies:** [task-0]

**Files to CREATE:**
- `src/python/htmlgraph/ingest/gemini_cli.py`
- `tests/ingest/test_gemini_cli.py`

**Gemini CLI Parser Details:**
- Location: `~/.gemini/tmp/<sha256-project-hash>/chats/session-*.json`
- Format: JSON (not JSONL) with top-level fields: sessionId, projectHash, startTime, lastUpdated, messages[]
- Message types: "user", "gemini", "info"
- Gemini messages have: model, tokens{input,output,cached,thoughts,tool,total}, thoughts[], toolCalls[]
- Tool calls: id, name, args, result, status, timestamp, displayName
- Project hash: SHA256 of working directory path (need to reverse-map)
- Cross-session index: `~/.gemini/tmp/<hash>/logs.json` (array of user messages)

**Acceptance Criteria:**
- [ ] Discovers sessions by scanning ~/.gemini/tmp/*/chats/
- [ ] Parses JSON sessions into IngestedMessage objects
- [ ] Maps thinking blocks to metadata
- [ ] Captures per-message token counts
- [ ] Handles tool calls with full args/results
- [ ] Reverse-maps project hash to path where possible
- [ ] Tests with fixture data

---

### Task 3: Copilot + Codex CLI Parsers
**Feature:** feat-a7021f50
**Branch:** feature/ingester-copilot-codex
**Priority:** high
**Dependencies:** [task-0]

**Files to CREATE:**
- `src/python/htmlgraph/ingest/copilot_cli.py`
- `src/python/htmlgraph/ingest/codex_cli.py`
- `tests/ingest/test_copilot_cli.py`
- `tests/ingest/test_codex_cli.py`

**Copilot CLI Parser Details:**
- Location: `~/.copilot/session-state/*.jsonl`
- Format: JSONL event stream with envelope: {type, data, id, timestamp, parentId}
- Event types: session.start, session.model_change, user.message, assistant.message, tool.execution_start, tool.execution_complete, session.error, session.info, abort
- parentId forms a linked chain (tree structure)
- session.start has: sessionId, version, copilotVersion, startTime
- Tool linking: tool.execution_start and tool.execution_complete share toolCallId

**Codex CLI Parser Details:**
- Location: `~/.codex/sessions/*.jsonl` (research to confirm exact path)
- Format: JSONL with function calls rendered as tool blocks
- Parse function_call entries as tool use

**Acceptance Criteria:**
- [ ] Copilot discovers sessions from ~/.copilot/session-state/
- [ ] Parses JSONL event stream with correct parentId hierarchy
- [ ] Links tool start/complete events via toolCallId
- [ ] Tracks model changes per session
- [ ] Codex parser handles function call format
- [ ] Tests with fixture data

---

### Task 4: HtmlGraph MCP Server
**Feature:** feat-4c3fc1fa
**Branch:** feature/mcp-server
**Priority:** high
**Dependencies:** none (fully independent)

**Files to CREATE:**
- `src/python/htmlgraph/mcp/__init__.py`
- `src/python/htmlgraph/mcp/server.py`
- `tests/mcp/__init__.py`
- `tests/mcp/test_server.py`

**Files to MODIFY:**
- `pyproject.toml` (add `mcp` dependency)
- `packages/claude-plugin/.claude-plugin/plugin.json` (register MCP server)

**MCP Server Design:**
Tools to expose:
- `record_event(tool_name, input, output, agent_id?)` — Record a tool use event
- `query_sessions(tool_source?, project_path?, limit?)` — List sessions
- `search_messages(query, tool_source?, limit?)` — Full-text search
- `get_session_tree(session_id)` — Get message hierarchy for a session
- `get_status()` — Current HtmlGraph status (features, sessions, etc.)

Resources to expose:
- `htmlgraph://sessions` — List of recent sessions
- `htmlgraph://features` — Active features

Implementation: Use `mcp` Python package with stdio transport.

**Acceptance Criteria:**
- [ ] MCP server starts and responds to tool list
- [ ] record_event writes to SQLite
- [ ] query_sessions returns session list
- [ ] search_messages works (basic, FTS5 later)
- [ ] Registered in plugin.json for Claude Code
- [ ] Documentation for adding to Gemini/Copilot MCP config
- [ ] Tests pass

---

### Task 5: Missing Hook Events
**Feature:** feat-195fcccc
**Branch:** feature/missing-hooks
**Priority:** high
**Dependencies:** [task-1]

**Files to CREATE:**
- `packages/claude-plugin/hooks/scripts/precompact.py`
- `packages/claude-plugin/hooks/scripts/instructions-loaded.py`
- `packages/claude-plugin/hooks/scripts/permission-request.py`

**Files to MODIFY:**
- `packages/claude-plugin/hooks/hooks.json` (add entries for new events)
- `src/python/htmlgraph/hooks/event_recording.py` (handle new event types)

**Hook Events to Add:**
1. **PreCompact** — Archive conversation before compaction. Matcher: `auto|manual`. Input has `transcript_path`.
2. **InstructionsLoaded** — Track which CLAUDE.md/rules shaped session. Fires when files are loaded.
3. **PermissionRequest** — Permission audit trail. Matcher: tool name.
4. **SessionStart matcher expansion** — Handle `startup|resume|clear|compact` matchers differently.

**Acceptance Criteria:**
- [ ] PreCompact hook fires and archives transcript
- [ ] InstructionsLoaded hook records loaded files
- [ ] PermissionRequest hook creates audit events
- [ ] SessionStart handles `compact` matcher (session resumed after compaction)
- [ ] All new hooks in hooks.json
- [ ] Tests pass

---

### Task 6: OpenCode + Cursor Parsers
**Feature:** feat-26a50458
**Branch:** feature/ingester-opencode-cursor
**Priority:** medium
**Dependencies:** [task-0]

**Files to CREATE:**
- `src/python/htmlgraph/ingest/opencode.py`
- `src/python/htmlgraph/ingest/cursor.py`
- `tests/ingest/test_opencode.py`
- `tests/ingest/test_cursor.py`

**OpenCode Parser Details:**
- Location: `~/.local/share/opencode/storage/`
- Hierarchy: project/{hash}.json → session/{projectHash}/{sessionID}.json → message/{sessionID}/{messageID}.json → part/{messageID}/{partID}.json
- Part types: text, step-start, step-finish, tool
- IDs: Sortable base62 with prefixes (ses_, msg_, prt_)
- Token/cost data at both message level and per step-finish part
- Project ID: SHA1 hash of git worktree path

**Cursor Parser Details:**
- Location: `~/Library/Application Support/Cursor/User/globalStorage/state.vscdb`
- Format: SQLite with `ItemTable(key TEXT PRIMARY KEY, value TEXT)`
- Chat data in `cursorDiskKV` table
- Per-workspace: `workspaceStorage/<hash>/state.vscdb`

**Acceptance Criteria:**
- [ ] OpenCode parser traverses storage hierarchy
- [ ] Handles all part types (text, tool, step-start/finish)
- [ ] Cursor parser reads SQLite vscdb files
- [ ] Cross-tool correlation by project directory
- [ ] Tests with fixture data

---

## Wave 2: Search & Integration (2 parallel tasks)

### Task 7: FTS5 Full-Text Search
**Feature:** feat-cc787a00
**Branch:** feature/fts5-search
**Priority:** medium
**Dependencies:** [task-0, task-2] (needs data model + at least one parser)

**Files to CREATE:**
- `src/python/htmlgraph/search.py`
- `tests/test_search.py`

**Files to MODIFY:**
- `src/python/htmlgraph/db/schema.py` (add FTS5 virtual table)
- `src/python/htmlgraph/cli/` (add `htmlgraph search` command)
- `src/python/htmlgraph/ingest/base.py` (index messages after ingestion)

**FTS5 Schema:**
```sql
CREATE VIRTUAL TABLE messages_fts USING fts5(
    content,
    tool_input,
    tool_output,
    content='ingested_messages',
    content_rowid='rowid',
    tokenize='porter unicode61'
);
```

**Acceptance Criteria:**
- [ ] FTS5 virtual table created in schema migration
- [ ] Messages indexed on ingest
- [ ] `htmlgraph search "query"` returns ranked results
- [ ] Results include session context and source tool
- [ ] Tests pass

---

### Task 8: HTTP Hooks + OpenTelemetry
**Feature:** feat-a86f8555
**Branch:** feature/http-hooks-otel
**Priority:** medium
**Dependencies:** [task-1, task-5]

**Files to CREATE:**
- `src/python/htmlgraph/server/api.py` (HTTP event receiver)
- `src/python/htmlgraph/otel.py` (OTel receiver/processor)
- `tests/test_api.py`
- `tests/test_otel.py`

**Files to MODIFY:**
- `src/python/htmlgraph/cli/` (add `htmlgraph serve --api` mode)
- `packages/claude-plugin/hooks/hooks.json` (add type:http alternatives)

**Implementation:**
- `htmlgraph serve --api` starts HTTP server on localhost:8081
- Convert select hooks to `type:"http"` for reduced subprocess overhead
- Add OTel receiver endpoint that accepts OTLP JSON for cost/token data
- SSE endpoint for real-time dashboard streaming

**Acceptance Criteria:**
- [ ] HTTP server receives events at /api/events
- [ ] At least one hook converted to type:http
- [ ] OTel endpoint processes cost/token metrics
- [ ] SSE streaming works for dashboard
- [ ] Tests pass

---

## Worktree Setup Script

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "Setting up worktrees for Session Ingestion parallel development..."

MAIN_BRANCH=$(git branch --show-current)
TASKS=(
    "ingester-base-claude"
    "hook-hierarchy"
    "ingester-gemini"
    "ingester-copilot-codex"
    "mcp-server"
    "missing-hooks"
    "ingester-opencode-cursor"
    "fts5-search"
    "http-hooks-otel"
)

mkdir -p worktrees

for task in "${TASKS[@]}"; do
    branch="feature/$task"
    worktree="worktrees/$task"

    if [ -d "$worktree" ]; then
        echo "  Worktree exists: $task"
    elif git show-ref --verify --quiet "refs/heads/$branch"; then
        git worktree add "$worktree" "$branch" 2>/dev/null
        echo "  Created: $task (existing branch)"
    else
        git worktree add "$worktree" -b "$branch" 2>/dev/null
        echo "  Created: $task (new branch from $MAIN_BRANCH)"
    fi
done

echo ""
echo "Setup complete! Active worktrees:"
git worktree list | grep "worktrees/"
echo ""
echo "Wave 0 (start now):  worktrees/ingester-base-claude, worktrees/hook-hierarchy"
echo "Wave 1 (after Wave 0): worktrees/ingester-gemini, worktrees/ingester-copilot-codex, worktrees/mcp-server, worktrees/missing-hooks, worktrees/ingester-opencode-cursor"
echo "Wave 2 (after Wave 1): worktrees/fts5-search, worktrees/http-hooks-otel"
```

## Execution Commands

### Wave 0 (2 parallel agents)
```bash
# Both launched simultaneously via Agent tool
Agent(prompt="Task 0: Base Ingester + Claude Parser...", subagent_type="htmlgraph:sonnet-coder", isolation="worktree")
Agent(prompt="Task 1: Hook Hierarchy Fix...", subagent_type="htmlgraph:sonnet-coder", isolation="worktree")
```

### Wave 1 (5 parallel agents, after Wave 0 merges)
```bash
Agent(prompt="Task 2: Gemini CLI Parser...", subagent_type="htmlgraph:haiku-coder", isolation="worktree")
Agent(prompt="Task 3: Copilot + Codex Parsers...", subagent_type="htmlgraph:haiku-coder", isolation="worktree")
Agent(prompt="Task 4: MCP Server...", subagent_type="htmlgraph:sonnet-coder", isolation="worktree")
Agent(prompt="Task 5: Missing Hook Events...", subagent_type="htmlgraph:haiku-coder", isolation="worktree")
Agent(prompt="Task 6: OpenCode + Cursor Parsers...", subagent_type="htmlgraph:haiku-coder", isolation="worktree")
```

### Wave 2 (2 parallel agents, after Wave 1 merges)
```bash
Agent(prompt="Task 7: FTS5 Search...", subagent_type="htmlgraph:sonnet-coder", isolation="worktree")
Agent(prompt="Task 8: HTTP Hooks + OTel...", subagent_type="htmlgraph:sonnet-coder", isolation="worktree")
```

## Merge Strategy

After each wave:
1. Run tests in each worktree: `cd worktrees/<task> && uv run pytest`
2. Merge to main: `git checkout main && git merge --no-ff feature/<task>`
3. Push: `git push origin main`
4. Clean up: `git worktree remove worktrees/<task>`
5. Verify main: `uv run ruff check --fix && uv run mypy src/ && uv run pytest`

## Conflict Mitigation

| Shared Resource | Tasks | Strategy |
|---|---|---|
| `hooks.json` | Task 1, Task 5 | Sequential (Wave 0 then Wave 1) |
| `db/schema.py` | Task 0, Task 1, Task 7 | Task 0 creates tables, Task 1 adds columns, Task 7 adds FTS5 |
| `cli/` | Task 0, Task 7 | Task 0 creates `ingest` cmd, Task 7 creates `search` cmd |
| `pyproject.toml` | Task 0, Task 4 | Different sections (unlikely conflict) |
| `ingest/__init__.py` | Tasks 0,2,3,6 | Task 0 creates, others add imports (easy merge) |

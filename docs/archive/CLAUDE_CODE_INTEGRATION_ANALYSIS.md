# Claude Code Integration Opportunities - HtmlGraph Analysis

**Analysis Date:** January 13, 2026
**Scope:** Claude Code plugin/hook capabilities vs HtmlGraph observability features
**Status:** Complete - Ready for implementation planning

---

## Executive Summary

HtmlGraph has a **sophisticated plugin architecture** for Claude Code that captures development workflows through hooks. This analysis identifies:

1. **What Claude Code natively tracks** - Session metadata, tool calls, permissions
2. **What HtmlGraph currently captures** - Agent events, sessions, features, cost analytics
3. **Critical gaps** - Missing workflow insights that HtmlGraph could provide
4. **Opportunities** - Where HtmlGraph could enhance Claude Code's native capabilities

**Key Finding:** Claude Code provides the infrastructure (hooks, agent spawning, transcript access), but lacks **strategic workflow analysis**. HtmlGraph can bridge this gap by providing decision support, pattern recognition, and team coordination features.

---

## Part 1: Claude Code Capabilities Overview

### 1.1 What Claude Code Tracks Natively

Claude Code provides **rich metadata** to plugins via hooks:

#### Hook Input Data

**SessionStart Hook receives:**
```json
{
  "session_id": "sess-abc123",           # Unique session identifier
  "transcript_path": "/path/to/session.jsonl",  # Full conversation history
  "cwd": "/project/root",                # Current working directory (project root)
  "permission_mode": "default|...",      # User permission level
  "hook_event_name": "SessionStart",
  "source": "startup|resume|clear|compact"  # Session origin
}
```

**PreToolUse Hook receives:**
```json
{
  "session_id": "sess-abc123",
  "tool_name": "Read|Edit|Bash|Grep|...",
  "tool_input": {...},                   # Arguments to the tool
  "tool_use_id": "tool-xyz789",         # Unique identifier for this tool call
  "transcript_path": "/path/to/session.jsonl"
}
```

**PostToolUse Hook receives:**
```json
{
  "session_id": "sess-abc123",
  "tool_name": "Read|Edit|...",
  "tool_input": {...},
  "tool_response": {...},                # Tool result/output
  "tool_use_id": "tool-xyz789"
}
```

**UserPromptSubmit Hook receives:**
```json
{
  "prompt": "What the user asked",
  "session_id": "sess-abc123",
  "transcript_path": "/path/to/session.jsonl"
}
```

**SubagentStop Hook receives:**
```json
{
  "session_id": "parent-sess-id",
  "subagent_session_id": "child-sess-id",
  "subagent_type": "general-purpose|...",
  "subagent_status": "completed|failed|...",
  "subagent_output": "..."
}
```

#### What This Means for HtmlGraph

1. **Session Continuity** - `session_id` + `source` tells us if session is new, resumed, or compacted
2. **Project Context** - `cwd` tells us the project root (enables `.htmlgraph/` discovery)
3. **Transcript Access** - `transcript_path` provides full conversation history for analytics
4. **Tool Tracking** - `tool_name`, `tool_use_id`, `tool_input`, `tool_response` enable detailed execution tracing
5. **Multi-Agent Support** - `subagent_session_id` enables tracking parent-child relationships
6. **User Intent** - `prompt` in UserPromptSubmit gives us the user's actual request

### 1.2 Hook Events Available

Claude Code fires hooks at **5 critical points**:

| Hook | Trigger | Timing | Data Available | Use Cases |
|------|---------|--------|-----------------|-----------|
| **SessionStart** | Session begins/resumes | Before Claude sees transcript | project_dir, session_id, source | Initialize tracking, inject context, check version |
| **UserPromptSubmit** | User submits prompt | Before Claude processes | prompt text, session_id | Analyze intent, provide guidance, detect patterns |
| **PreToolUse** | Before tool execution | Can block/modify | tool_name, tool_input, tool_use_id | Validate/enforce patterns, provide suggestions |
| **PostToolUse** | After tool succeeds | Can provide feedback | tool_name, tool_input, tool_response | Record execution, detect errors, provide guidance |
| **SubagentStop** | Subagent completes | After delegation | parent_id, child_id, subagent_type, output | Link parent-child work, track delegation chains |
| **Stop** | User stops agent | Session end | (minimal data) | Cleanup, save state |
| **SessionEnd** | Session ends | After all work done | (minimal data) | Archive session, save handoff notes |

### 1.3 Agent Spawning Capabilities

Claude Code provides **Task() tool** for spawning subagents:

```python
Task(
    subagent_type="general-purpose",    # Type: general-purpose, haiku-coder, sonnet-coder, opus-coder
    prompt="Do this work",               # Delegation prompt
    model="claude-opus-4-5",            # Optional: specify model
    expected_output="...",              # Optional: expected format
    max_budget="$10"                     # Optional: cost limit
)
```

**Available Subagent Types:**
- `general-purpose` - Flexible multi-step work
- `haiku-coder` - Fast execution for simple tasks
- `sonnet-coder` - Balanced code execution
- `opus-coder` - Deep reasoning for complex problems

**SubagentStop Hook fires when:**
1. Subagent completes successfully
2. Subagent hits error/timeout
3. Subagent runs out of budget
4. Parent agent cancels delegation

### 1.4 Permission Modes

Claude Code tracks **user permission level** in every hook:

```
permission_mode in [
  "default",          # Standard permissions
  "plan",            # Can plan but not execute (requires approval)
  "acceptEdits",     # Auto-approve file edits
  "dontAsk",         # Auto-approve all operations
  "bypassPermissions" # Full access (admin only)
]
```

**HtmlGraph Can Use This To:**
- Detect when work requires user approval
- Flag high-permission operations
- Track compliance/audit trail for teams

### 1.5 Agent Information Available

Through environment variables + hook input, HtmlGraph can detect:

```
CLAUDE_AGENT_NICKNAME  # User-provided agent identifier (e.g., "alice", "bob")
CLAUDE_MODEL          # Model being used (e.g., "claude-opus-4-5")
HTMLGRAPH_MODEL       # Fallback model detection
```

**Multi-Agent Context:**
- Parent agent can see which subagent is working (`subagent_type`)
- Child agent can receive context via hook input
- Session IDs link parent-child relationships

---

## Part 2: What HtmlGraph Currently Captures

### 2.1 Event Types Tracked

HtmlGraph's database schema captures **10+ event types**:

```sql
event_type IN (
  'tool_call',           # Tool invocation (Read, Edit, Bash, etc.)
  'tool_result',        # Tool succeeded
  'error',              # Tool or agent error
  'delegation',         # Task() delegation to subagent
  'task_completion',    # Task finished
  'feature_creation',   # New feature created
  'feature_update',     # Feature status changed
  'session_start',      # Agent session began
  'session_end',        # Agent session ended
  'agent_context_update' # Agent state changed
)
```

### 2.2 Data Captured Per Event

For each event, HtmlGraph records:

```python
{
  "event_id": "evt-abc123",
  "session_id": "sess-xyz",
  "agent_id": "claude",
  "model": "claude-opus",
  "tool_name": "Read|Edit|Bash|...",
  "tool_input": {...},
  "tool_response": {...},
  "feature_id": "feat-123",           # Linked feature (if applicable)
  "subagent_type": "general-purpose", # If delegated
  "cost_tokens": 1500,
  "execution_duration_seconds": 2.3,
  "status": "success|error|blocked",
  "created_at": "2026-01-13T10:30:00Z",
  "updated_at": "2026-01-13T10:30:05Z"
}
```

### 2.3 Session Tracking

HtmlGraph maintains **sessions table** with:

```python
{
  "session_id": "sess-abc123",
  "agent_assigned": "claude",
  "parent_session_id": "parent-sess-id",  # For subagent sessions
  "parent_event_id": "evt-xyz",           # Which parent event spawned this
  "status": "active|completed|failed",
  "created_at": "2026-01-13T10:00:00Z",
  "ended_at": "2026-01-13T11:30:00Z",
  "total_cost_tokens": 5000,
  "tool_call_count": 23,
  "error_count": 2,
  "metrics": {...}  # JSON: efficiency score, retry rates, etc.
}
```

### 2.4 Agent Collaboration Tracking

HtmlGraph has **agent_collaboration table** for:

```python
{
  "id": "collab-123",
  "agent_1": "claude",
  "agent_2": "codex",
  "work_type": "handoff|parallel|sequential",
  "feature_id": "feat-001",
  "start_time": "...",
  "end_time": "...",
  "conflict_detected": False,
  "resolution": "auto|manual|none"
}
```

---

## Part 3: Integration Points - Detailed Analysis

### 3.1 Hook Integration Architecture

**Current HtmlGraph Hook Configuration:**

```json
{
  "hooks": {
    "SessionStart": ["session-start.py"],      # Initialize session
    "UserPromptSubmit": ["user-prompt-submit.py"],  # Analyze intent
    "PreToolUse": ["pretooluse-integrator.py"],     # Track + enforce patterns
    "PostToolUse": ["posttooluse-integrator.py"],   # Record + provide feedback
    "SubagentStop": ["subagent-stop.py"],          # Link parent-child work
    "SessionEnd": ["session-end.py"],              # Archive + save handoff
    "Stop": ["track-event.py"]                     # Minimal cleanup
  }
}
```

**Hook Execution Model:**
1. Hooks run **synchronously** (Claude Code waits for response)
2. Timeout: 60 seconds per hook (configurable)
3. Multiple hooks for same event **execute in parallel**, outputs **merged**
4. Hooks can:
   - Return `additionalContext` (injected to Claude)
   - Return `permissionDecision` (block/allow tool)
   - Return `updatedInput` (modify tool arguments)
   - Return `systemMessage` (show to user)

### 3.2 Data Flow: What Information Moves Where

```
Claude Code
  ↓
Hook receives hook_input (session_id, tool_name, etc.)
  ↓
HookContext.from_input()
  Resolves project_dir via bootstrap
  Detects agent_id, model_name
  Creates database connection
  ↓
Hook executes business logic
  Queries .htmlgraph/ state (features, sessions)
  Records events to SQLite
  Generates context/guidance
  ↓
Hook returns JSON output
  ├─ continue: true/false
  ├─ hookSpecificOutput: {...}
  │   ├─ additionalContext (injected to Claude)
  │   ├─ permissionDecision (block/allow)
  │   └─ updatedInput (modified tool args)
  ├─ systemMessage (shown to user)
  └─ suppressOutput: true/false
  ↓
Claude Code processes response
  Injects context
  Applies permission decision
  Continues/stops execution
```

### 3.3 Critical Limitation: SessionStart Hook Timing

**Current Limitation:**
- SessionStart hook runs **before** Claude processes any messages
- It has access to project context but **NOT** to current conversation state
- Results: Can't adapt system prompt based on current work

**Opportunity:**
- SessionStart generates context from `.htmlgraph/` state (previous session, features, recommendations)
- UserPromptSubmit hook could provide additional context based on **actual prompt** user submitted
- Creates two-layer context injection: static + dynamic

### 3.4 Transcript Access Opportunity

**Currently Unused:** `transcript_path` is available to hooks but not fully leveraged.

**Potential Uses:**
1. **Workflow pattern recognition** - Detect repeated tool sequences
2. **Error analysis** - Find patterns in failures
3. **Context drift** - Detect when agent loses focus
4. **Learning from history** - Suggest better approaches based on past sessions
5. **Transcript-driven analytics** - Export and analyze conversations

**Implementation:**
```python
# In hook, read transcript
with open(transcript_path) as f:
    transcript = json.load(f)  # JSONL format

# Analyze conversation
for event in transcript:
    if event['type'] == 'tool_result':
        analyze_tool_outcome(event)
    elif event['type'] == 'user_message':
        detect_user_intent(event['content'])
```

### 3.5 Multi-Agent Coordination Opportunities

**Current State:**
- SubagentStop hook knows parent/child session IDs
- Can track delegation chains (parent → child → grandchild)
- No conflict detection

**Opportunities:**
1. **Parallel work detection** - Identify when multiple agents work on same feature
2. **Conflict resolution** - Detect incompatible changes, suggest resolution
3. **Load balancing** - Suggest which agent to delegate to
4. **Work distribution** - Recommend task splitting across agents
5. **Knowledge sharing** - Share context between parallel agents

---

## Part 4: Observability Gaps - What's Missing

### 4.1 Strategic Workflow Analysis

**Missing:** Claude Code provides **what** happened, but not **why** or **what's next**.

HtmlGraph Could Provide:
- ✅ Pattern recognition (e.g., "You've done 3 similar tasks, here's the pattern")
- ✅ Efficiency metrics (e.g., "This took 2x longer than last time, here's why")
- ✅ Bottleneck detection (e.g., "Stuck on auth for 30 min, delegate to specialist")
- ✅ Decision guidance (e.g., "Refactor vs rewrite? Data suggests refactor")
- ✅ Next work recommendation (e.g., "This feature unblocks 3 others")

### 4.2 Team Coordination & Handoff

**Missing:** No built-in team awareness or handoff support.

HtmlGraph Could Provide:
- ✅ Agent capability discovery (e.g., "Alice specializes in auth")
- ✅ Workload balancing (e.g., "Bob is less busy, delegate to him")
- ✅ Conflict detection (e.g., "Alice and Bob both editing same file")
- ✅ Handoff protocols (e.g., "Pass to Charlie with context")
- ✅ Collaboration history (e.g., "Alice-Bob pairs work well together")

### 4.3 Cost & Resource Management

**Missing:** No visibility into token usage, costs, or budget constraints.

HtmlGraph Could Provide:
- ✅ Token tracking (e.g., "This session used 50k tokens")
- ✅ Cost analysis (e.g., "Use Haiku for 70% savings")
- ✅ Budget alerts (e.g., "Track at risk of exceeding budget")
- ✅ Cost attribution (e.g., "Feature X cost $2.50 in tokens")
- ✅ Model recommendations (e.g., "Opus needed for this complexity")

### 4.4 Learning & Improvement

**Missing:** No systematic way to improve workflow based on history.

HtmlGraph Could Provide:
- ✅ Anti-pattern detection (e.g., "4 consecutive Bash calls detected")
- ✅ Optimal pattern matching (e.g., "Grep → Read → Edit is efficient")
- ✅ Error rate analysis (e.g., "Tests fail 30% of time, invest in debugging")
- ✅ Skill development (e.g., "Practice this type of edit")
- ✅ Tool mastery metrics (e.g., "You're efficient with Read, practice Edit")

### 4.5 Compliance & Audit

**Missing:** Limited ability to track who did what, when, why.

HtmlGraph Could Provide:
- ✅ Audit trail (e.g., "All edits to auth.py traced to session X")
- ✅ Access control (e.g., "Only Alice can edit payment code")
- ✅ Policy enforcement (e.g., "All refactoring requires code review")
- ✅ Compliance reporting (e.g., "Security review completed for X")
- ✅ Change tracking (e.g., "Who changed this line and why")

---

## Part 5: Detailed Opportunities by Use Case

### 5.1 Opportunity: Real-Time Workflow Guidance

**Problem:** Agent gets stuck or inefficient; no guidance.

**Solution - PreToolUse Hook Enhancement:**
```python
# In hook, detect problematic patterns
recent_tools = get_recent_tools(session_id, limit=5)
if recent_tools == ["Bash", "Bash", "Bash", "Edit", "Bash"]:
    guidance = "Pattern detected: Multiple Bash calls. Consider batching?"
    return {
        "continue": True,
        "systemMessage": guidance
    }
```

**Implementation:**
- Hook tracks tool sequence in memory
- Detects anti-patterns: 4x same tool, 3x repeated failure
- Suggests optimal patterns: Grep → Read → Edit → Bash
- Non-blocking (never prevents work), purely advisory

**Data Available to Hook:**
- ✅ `session_id` - All events in this session
- ✅ `tool_name` - Current tool being used
- ✅ `transcript_path` - Full conversation history
- ✅ Database access - Query previous sessions for patterns

### 5.2 Opportunity: Intelligent Task Delegation

**Problem:** Large task feels monolithic; no guidance on delegation.

**Solution - PostToolUse Hook Enhancement:**
```python
# After user completes major task, suggest parallelization
if is_major_task_complete(session_id):
    suggestions = analyze_decomposition(task_description)
    # Returns: ["Unit tests", "Integration tests", "Documentation"]

    return {
        "continue": True,
        "systemMessage": f"""
        Consider delegating remaining work:
        {suggestions}

        Example: Task(prompt="Write unit tests for auth module")
        """
    }
```

**Implementation:**
- Detect when major feature or spike completes
- Suggest work that could be parallelized
- Provide delegation examples
- Track if user acts on suggestions

**Data Available to Hook:**
- ✅ `tool_response` - See what was just accomplished
- ✅ `feature_id` - Know which feature being worked on
- ✅ Database access - Query related features for decomposition
- ✅ `subagent_type` available - Suggest appropriate subagent type

### 5.3 Opportunity: Cost-Aware Model Selection

**Problem:** Agent uses expensive Opus for simple Haiku tasks.

**Solution - SessionStart Hook Enhancement:**
```python
# At session start, recommend model based on task complexity
feature = get_current_feature(session_id)
complexity = analyze_feature_complexity(feature)

if complexity == "low":
    recommendation = """
    💰 Cost Optimization Opportunity
    This feature appears simple (low complexity).
    Consider using haiku-coder subagent for 90% cost savings.
    """
    return {"continue": True, "systemMessage": recommendation}
```

**Implementation:**
- SessionStart hook analyzes current feature
- Estimates complexity (lines of code, tests, dependencies)
- Recommends appropriate model
- Show cost comparison
- Track if recommendation is followed

**Data Available to Hook:**
- ✅ `.htmlgraph/features/` - Feature metadata + complexity estimates
- ✅ SQLite database - Historical cost data per feature type
- ✅ Hook input - Can detect which model is running

### 5.4 Opportunity: Conflict Detection in Parallel Work

**Problem:** Two agents edit same file without knowing; merge conflicts.

**Solution - PreToolUse Hook Enhancement:**
```python
# When Edit called, check if another agent is also editing
if tool_name == "Edit" and file_path in get_concurrent_edits():
    agents = get_agents_editing(file_path)
    return {
        "permissionDecision": "block",  # Require user approval
        "systemMessage": f"Conflict: {agents} also editing {file_path}"
    }
```

**Implementation:**
- Track which files each active session is editing
- Detect overlapping edits before they cause conflicts
- Block with helpful message, or auto-merge if safe
- Suggest collaboration (coordinate with other agent)

**Data Available to Hook:**
- ✅ `tool_input` - See which file is being edited
- ✅ SQLite database - Query current sessions
- ✅ `session_id` - Know which session is making call

### 5.5 Opportunity: Error Recovery Suggestions

**Problem:** Tool fails (test suite errors, syntax errors); no next step suggestion.

**Solution - PostToolUse Failure Hook:**
```python
# After tool fails, suggest debugging approach
if tool_response.status == "error":
    error_type = categorize_error(tool_response.error)
    if error_type == "test_failure":
        suggestion = suggest_debugging(error_type, context)
        return {
            "continue": True,
            "systemMessage": f"💡 Try this: {suggestion}"
        }
```

**Implementation:**
- Categorize error type (syntax, test failure, file not found, etc.)
- Suggest proven debugging approaches
- Link to documentation if applicable
- Offer to delegate debugging to specialist agent

**Data Available to Hook:**
- ✅ `tool_response` - See the error message
- ✅ `tool_name` - Know which tool failed
- ✅ Transcript access - See what was attempted before
- ✅ Database access - Query similar errors in history

### 5.6 Opportunity: Multi-Agent Coordination & Load Balancing

**Problem:** No awareness of other agents working on related features.

**Solution - Task Delegation Enhancement:**
```python
# When parent decides to delegate, recommend which subagent type
task_description = "Write comprehensive test suite"
recommended_type = analyze_task_recommend_agent(task_description)
# Could return: "general-purpose" (flexible) or "haiku-coder" (fast)

# Also detect if another agent is already doing related work
related_work = find_related_active_sessions(feature_id)
if related_work:
    print(f"Note: {related_work} already working on related feature")
```

**Implementation:**
- SessionStart analyzes current feature + active sessions
- Detects if related work is happening elsewhere
- Recommends coordination vs parallelization
- Suggests handoff instead of parallel work when appropriate

**Data Available to Hook:**
- ✅ SQLite sessions table - See all active sessions
- ✅ agent_collaboration table - Query handoffs and conflicts
- ✅ Feature relationships - Query blocking/blocked_by relationships

---

## Part 6: Constraints & Limitations

### 6.1 Hook Execution Constraints

| Constraint | Impact | Workaround |
|-----------|--------|-----------|
| **60-second timeout** | Long operations (codebase analysis) fail | Implement caching, use fast queries |
| **Synchronous execution** | Slow hooks delay Claude | Offload to async task runners |
| **No direct console output** | Limited debugging | Use stderr for logging |
| **STDIN/STDOUT only** | Complex data structures limited | Use JSON serialization |
| **No file modifications** | Can't auto-fix issues | Return suggestions, user must act |
| **Read-only permissions** | Can't write state | Use database writes instead |

### 6.2 Data Availability Constraints

| Data | Available In | Impact |
|------|-------------|--------|
| **Full transcript** | PostToolUse hook gets partial transcript | Can read full transcript_path but slow |
| **Current working state** | Only in PreToolUse/PostToolUse | SessionStart has no live context |
| **User context** | Only in UserPromptSubmit | Other hooks don't know what user is trying to do |
| **Model information** | Detected from env/status cache | Not always accurate across sessions |
| **Parent session ID** | Only in SubagentStop | Child agents don't know parent |
| **Conversation history** | Transcript_path available | Must parse JSONL manually |

### 6.3 Permission & Access Constraints

| Constraint | Impact |
|-----------|--------|
| **Tool blocking** | PreToolUse can block but only if necessary |
| **No direct edits** | Can't auto-fix code, can suggest |
| **No tool execution** | Can't call Bash/Edit directly |
| **User approval required** | Strict mode requires user OK for blocks |
| **Environment access** | Limited to Claude Code provided vars |

### 6.4 Performance Constraints

| Operation | Constraint | Solution |
|-----------|-----------|----------|
| **Database queries** | Must complete in <1 second | Index heavily, pre-compute analytics |
| **File I/O** | Network FS can be slow | Cache in memory, use SQLite |
| **Subprocess calls** | `uv run` adds overhead | Use `--with` flag in shebang |
| **Python imports** | Cold start slow | Keep hook modules small |
| **Large dataset analysis** | Can't analyze entire codebase | Sample, use statistics |

### 6.5 Configuration & Discovery Constraints

| Constraint | Impact |
|-----------|--------|
| **Hook auto-discovery** | No automatic reload if hooks.json changes | User must restart Claude |
| **Environment variables** | Limited to what Claude Code sets | Hook must detect from hook_input |
| **Plugin updates** | Don't take effect until Claude restarts | Plan for eventual consistency |
| **Path resolution** | `cwd` might not be git root | Must use `git rev-parse --show-toplevel` |
| **Multi-window conflicts** | Each window has separate session | Need database-level deduplication |

---

## Part 7: Recommended Implementation Roadmap

### Phase 1: Enhance Existing Hooks (Quick Wins)

**Goal:** Leverage existing hook infrastructure to provide immediate value.

**Implementations:**
1. **Pattern Recognition in PreToolUse**
   - Track recent tool sequence
   - Detect anti-patterns (4x Bash, 3x same tool)
   - Suggest optimal patterns
   - **Effort:** 1-2 days
   - **Impact:** High (immediate feedback loop)

2. **Cost Awareness in SessionStart**
   - Analyze feature complexity
   - Recommend model based on complexity
   - Show cost comparison
   - **Effort:** 2-3 days
   - **Impact:** Medium (cost savings)

3. **Error Recovery in PostToolUse**
   - Categorize error types
   - Suggest debugging approaches
   - Link to documentation
   - **Effort:** 2-3 days
   - **Impact:** High (unblock errors faster)

**Total Effort:** 5-8 days
**Expected Outcomes:**
- Better decision support in Claude Code
- Lower token costs
- Faster error recovery

### Phase 2: Multi-Agent Coordination (Medium-Term)

**Goal:** Enable intelligent delegation and team coordination.

**Implementations:**
1. **Concurrent Editing Detection**
   - Track active file edits per session
   - Detect conflicts before they occur
   - Block or suggest coordination
   - **Effort:** 3-4 days

2. **Task Decomposition Suggestions**
   - Detect when task could be parallelized
   - Suggest work breakdown
   - Recommend subagent type
   - **Effort:** 3-4 days

3. **Load Balancing**
   - Track agent capacity (WIP limits)
   - Suggest best agent for task
   - Alert when bottlenecks form
   - **Effort:** 4-5 days

**Total Effort:** 10-13 days
**Expected Outcomes:**
- Faster parallel execution
- Better resource utilization
- Fewer merge conflicts

### Phase 3: Analytics & Learning (Long-Term)

**Goal:** Enable continuous improvement and workflow optimization.

**Implementations:**
1. **Workflow Analytics Dashboard**
   - Visualize tool sequences
   - Identify bottlenecks
   - Compare team efficiency
   - **Effort:** 5-7 days

2. **Transcript-Driven Learning**
   - Extract decision points from transcripts
   - Build pattern library
   - Enable knowledge sharing
   - **Effort:** 5-7 days

3. **Predictive Recommendations**
   - Predict task complexity from description
   - Estimate time/cost to complete
   - Recommend approach based on history
   - **Effort:** 7-10 days

**Total Effort:** 17-24 days
**Expected Outcomes:**
- Measurable workflow improvements
- Data-driven decision making
- Team learning acceleration

---

## Part 8: Quick Reference - What HtmlGraph Can Access

### Available in SessionStart Hook
```python
hook_input = {
    "session_id": str,           # ✅ Use this
    "transcript_path": str,      # ✅ Can read full transcript
    "cwd": str,                  # ✅ Project root
    "permission_mode": str,      # ✅ User permission level
    "source": str                # ✅ new|resume|clear|compact
}

# HtmlGraph can access:
context.project_dir              # ✅ Project root
context.graph_dir               # ✅ .htmlgraph directory
context.session_id              # ✅ Current session
context.agent_id               # ✅ Agent name
context.model_name             # ✅ Model being used
context._database              # ✅ SQLite DB (all events)
```

### Available in PreToolUse Hook
```python
hook_input = {
    "session_id": str,           # ✅ Query active sessions
    "tool_name": str,            # ✅ Detect pattern
    "tool_input": dict,          # ✅ See what tool will do
    "tool_use_id": str,          # ✅ Unique tool call ID
}

# HtmlGraph can access:
# + Everything from SessionStart
# + Ability to block tool execution (permissionDecision)
# + Ability to suggest modifications (updatedInput)
```

### Available in PostToolUse Hook
```python
hook_input = {
    "session_id": str,           # ✅ Query active sessions
    "tool_name": str,            # ✅ See what ran
    "tool_input": dict,          # ✅ Original input
    "tool_response": dict,       # ✅ See result/error
    "tool_use_id": str,          # ✅ Match to PreToolUse
}

# HtmlGraph can access:
# + Everything from SessionStart
# + Tool result for error detection/recovery
```

### Available in UserPromptSubmit Hook
```python
hook_input = {
    "prompt": str,               # ✅ User's actual request
    "session_id": str,           # ✅ Current session
    "transcript_path": str,      # ✅ Full conversation
}

# HtmlGraph can access:
# + Everything from SessionStart
# + User intent for context-aware guidance
# + Full transcript for pattern analysis
```

### Available in SubagentStop Hook
```python
hook_input = {
    "session_id": str,           # ✅ Parent session
    "subagent_session_id": str,  # ✅ Child session
    "subagent_type": str,        # ✅ Task type
    "subagent_status": str,      # ✅ completed|failed
    "subagent_output": str,      # ✅ Result/error message
}

# HtmlGraph can access:
# + Everything from SessionStart
# + Parent-child relationship linking
# + Delegation outcome
```

---

## Part 9: Integration Examples

### Example 1: Pattern Recognition Hook

```python
#!/usr/bin/env -S uv run --with htmlgraph>=0.26.5 python3
"""
Pattern Recognition Hook - Detects tool sequences and provides guidance.
"""
import json
import sys
from htmlgraph.hooks.context import HookContext
from htmlgraph.db.schema import HtmlGraphDB

def detect_anti_patterns(session_id: str, db: HtmlGraphDB) -> str | None:
    """Detect problematic tool sequences."""
    # Query last 5 tool calls
    cursor = db.connection.cursor()
    cursor.execute("""
        SELECT tool_name FROM agent_events
        WHERE session_id = ? AND tool_name IS NOT NULL
        ORDER BY timestamp DESC
        LIMIT 5
    """, (session_id,))

    recent_tools = [row[0] for row in reversed(cursor.fetchall())]

    # Check for anti-patterns
    patterns = {
        ("Bash", "Bash", "Bash", "Bash"): "4 consecutive Bash calls. Check for errors or batch commands?",
        ("Edit", "Edit", "Edit"): "3 consecutive Edits. Consider planning changes first?",
        ("Grep", "Grep", "Grep", "Grep", "Grep"): "5 searches in a row. Try narrowing search scope?",
    }

    for pattern, message in patterns.items():
        if recent_tools[-len(pattern):] == list(pattern):
            return message

    return None

def main():
    hook_input = json.load(sys.stdin)

    try:
        context = HookContext.from_input(hook_input)
        db = HtmlGraphDB(str(context.graph_dir / "htmlgraph.db"))

        # Detect anti-patterns
        message = detect_anti_patterns(context.session_id, db)

        output = {
            "continue": True,
            "hookSpecificOutput": {"hookEventName": "PreToolUse"}
        }

        if message:
            output["systemMessage"] = f"💡 Suggestion: {message}"

        print(json.dumps(output))
        db.connection.close()
    except Exception as e:
        print(json.dumps({"continue": True}))
        sys.exit(1)

if __name__ == "__main__":
    main()
```

### Example 2: Cost-Aware Model Recommendation

```python
# In SessionStart hook, after initializing session

def recommend_model(feature_id: str) -> str | None:
    """Recommend model based on feature complexity."""
    try:
        feature = sdk.features.get(feature_id)
        if not feature:
            return None

        # Estimate complexity
        complexity_score = (
            len(feature.get("description", "")) / 100 +
            len(feature.get("steps", [])) +
            (1 if feature.get("track") else 0)
        )

        if complexity_score < 3:
            return "haiku"  # Simple task
        elif complexity_score < 7:
            return "sonnet"  # Moderate task
        else:
            return "opus"  # Complex task
    except:
        return None

# In session start output
recommended_model = recommend_model(current_feature_id)
if recommended_model:
    context_parts.append(f"""
    💰 Model Recommendation: {recommended_model}
    Based on feature complexity, this task would benefit from {recommended_model}.
    Current model cost: Opus ($0.015/1K tokens)
    Recommended model cost: {recommended_model.title()} (80% savings)
    """)
```

### Example 3: Concurrent Editing Detection

```python
# In PreToolUse hook

def check_concurrent_edits(file_path: str, session_id: str, db: HtmlGraphDB) -> dict | None:
    """Check if another session is editing the same file."""
    cursor = db.connection.cursor()
    cursor.execute("""
        SELECT DISTINCT session_id, agent_id
        FROM agent_events
        WHERE tool_name = 'Edit'
          AND tool_input LIKE ?
          AND session_id != ?
          AND timestamp > datetime('now', '-5 minutes')
        GROUP BY session_id
    """, (f'%"{file_path}"%', session_id))

    conflicts = cursor.fetchall()
    if conflicts:
        return {
            "conflicting_sessions": [
                {"session_id": row[0], "agent": row[1]}
                for row in conflicts
            ]
        }
    return None

# In hook response
if tool_name == "Edit":
    conflicts = check_concurrent_edits(tool_input["file_path"], session_id, db)
    if conflicts:
        return {
            "permissionDecision": "block",
            "systemMessage": f"""
            ⚠️ Concurrent Edit Conflict
            {len(conflicts['conflicting_sessions'])} other session(s) recently edited this file.
            Consider coordinating first.
            """
        }
```

---

## Part 10: Summary & Next Steps

### Key Findings

1. **Claude Code provides rich infrastructure:**
   - 7 hook types covering full execution lifecycle
   - Detailed metadata per tool call
   - Multi-agent support via Task() + SubagentStop
   - Transcript access for analytics

2. **HtmlGraph can bridge observability gaps:**
   - Pattern recognition (anti-patterns, optimal sequences)
   - Strategic guidance (decomposition, delegation, model selection)
   - Conflict detection (concurrent edits, resource contention)
   - Error recovery (categorization, debugging suggestions)
   - Analytics (workflow metrics, team insights)

3. **Integration is achievable with minimal changes:**
   - Use existing hook infrastructure
   - Leverage SQLite database for state
   - Query `.htmlgraph/` for project context
   - Return JSON responses for guidance

4. **Phased approach recommended:**
   - Phase 1 (1-2 weeks): Pattern recognition + cost awareness
   - Phase 2 (2-3 weeks): Multi-agent coordination
   - Phase 3 (3-4 weeks): Analytics + learning

### Immediate Opportunities (No Dependencies)

1. ✅ **Pattern Recognition** - Use PreToolUse to detect anti-patterns
2. ✅ **Cost Awareness** - Use SessionStart to recommend models
3. ✅ **Error Recovery** - Use PostToolUse failure to suggest debugging
4. ✅ **Transcript Analytics** - Use SessionEnd to analyze conversations
5. ✅ **Performance Tracking** - Use database to track execution metrics

### Dependencies for Advanced Features

1. **Team Coordination** - Requires multi-agent workload tracking
2. **Conflict Detection** - Requires real-time session monitoring
3. **Predictive Recommendations** - Requires ML model training
4. **Compliance Tracking** - Requires audit trail infrastructure

### Recommended Starting Point

**Quick Win Priority:**
1. Implement pattern recognition in PreToolUse (high impact, low effort)
2. Add cost recommendations in SessionStart (medium impact, medium effort)
3. Add error recovery suggestions in PostToolUse (high impact, low effort)

**Expected Impact:**
- Better user guidance in Claude Code
- Lower token costs (avoid expensive models)
- Faster error recovery
- Foundation for advanced features

---

## Appendix: Data Schema Reference

### agent_events Table
Stores every tool call, delegation, error, and feature update.

```sql
CREATE TABLE agent_events (
    event_id TEXT PRIMARY KEY,
    session_id TEXT,
    agent_id TEXT,
    hook_event_name TEXT,           -- SessionStart, PreToolUse, PostToolUse, etc.
    tool_name TEXT,                 -- Read, Edit, Bash, Grep, Task, etc.
    event_type TEXT,                -- tool_call, tool_result, error, delegation, etc.
    tool_input TEXT,                -- JSON
    tool_response TEXT,             -- JSON
    error_message TEXT,
    feature_id TEXT,                -- Linked feature (if applicable)
    subagent_type TEXT,             -- Type of delegated agent (if Task)
    status TEXT,                    -- success, error, blocked, recorded
    cost_tokens INTEGER,
    execution_duration_seconds REAL,
    model TEXT,                     -- claude-opus, claude-sonnet, etc.
    created_at DATETIME,
    updated_at DATETIME
)
```

### sessions Table
Tracks agent session lifecycle and metrics.

```sql
CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    agent_assigned TEXT,            -- Which agent ran this session
    parent_session_id TEXT,         -- Parent session (if subagent)
    parent_event_id TEXT,           -- Which event spawned this
    status TEXT,                    -- active, completed, failed
    created_at DATETIME,
    ended_at DATETIME,
    total_cost_tokens INTEGER,
    tool_call_count INTEGER,
    error_count INTEGER,
    metrics TEXT                    -- JSON: efficiency, retry rates, etc.
)
```

### agent_collaboration Table
Tracks multi-agent work patterns.

```sql
CREATE TABLE agent_collaboration (
    id TEXT PRIMARY KEY,
    agent_1 TEXT,
    agent_2 TEXT,
    work_type TEXT,                 -- handoff, parallel, sequential
    feature_id TEXT,
    start_time DATETIME,
    end_time DATETIME,
    conflict_detected BOOLEAN,
    resolution TEXT
)
```

---

**Analysis Complete**
Ready for implementation planning and prioritization.

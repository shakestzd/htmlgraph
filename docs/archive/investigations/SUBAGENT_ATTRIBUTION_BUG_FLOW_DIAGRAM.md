# Subagent Event Attribution Bug - Flow Diagrams

## Current (Broken) Flow

```
╔══════════════════════════════════════════════════════════════════════════╗
║                    ORCHESTRATOR (Sonnet 4.5)                            ║
╚══════════════════════════════════════════════════════════════════════════╝

┌─ SessionStart Hook ────────────────────────────────────────────────────┐
│ 1. Orchestrator starts                                                │
│ 2. SessionStart hook runs                                            │
│ 3. Creates session-abc123                                            │
│ 4. Stores in .wipnote/session.json (GLOBAL CACHE)                 │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

┌─ UserPromptSubmit Hook ─────────────────────────────────────────────────┐
│ 1. User submits query                                                 │
│ 2. UserQuery event created                                           │
│ 3. Records to session-abc123                                         │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

┌─ PostToolUse Hook (Task) ───────────────────────────────────────────────┐
│ 1. User calls: Task(subagent_type="gemini", prompt="analyze...")      │
│ 2. Track event for Task                                              │
│ 3. Records to session-abc123                                         │
│                                                                       │
│ ❌ ISSUE: Environment not prepared for subagent                       │
│    - HTMLGRAPH_SUBAGENT_TYPE not set                                 │
│    - HTMLGRAPH_PARENT_SESSION not set                                │
│    - Only HTMLGRAPH_PARENT_EVENT set                                 │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

┌─ PreToolUse Hook (Spawner Router) ───────────────────────────────────┐
│ 1. Intercepts Task() call with subagent_type="gemini"               │
│ 2. Checks CLI available                                             │
│ 3. Builds spawner command with environment:                         │
│    ✓ HTMLGRAPH_PARENT_EVENT = event-123                             │
│    ✓ HTMLGRAPH_MODEL = claude-sonnet                                │
│    ✓ HTMLGRAPH_PROJECT_ROOT = /path/to/project                      │
│                                                                      │
│    ❌ MISSING:                                                        │
│    ❌ HTMLGRAPH_SUBAGENT_TYPE = gemini                               │
│    ❌ HTMLGRAPH_PARENT_SESSION = session-abc123                      │
│    ❌ HTMLGRAPH_PARENT_AGENT = orchestrator                          │
│                                                                      │
│ 4. Spawns subprocess with incomplete environment                    │
│    env = {                                                           │
│      "HTMLGRAPH_PARENT_EVENT": "event-123",                         │
│      "HTMLGRAPH_MODEL": "claude-sonnet",                            │
│      "HTMLGRAPH_PROJECT_ROOT": "...",                               │
│      ...                                                             │
│      # MISSING SUBAGENT CONTEXT!                                     │
│    }                                                                 │
└──────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

╔══════════════════════════════════════════════════════════════════════════╗
║                  SUBAGENT PROCESS (Gemini 2.0-Flash)                     ║
║                     (Separate Python Process)                            ║
╚══════════════════════════════════════════════════════════════════════════╝

┌─ PostToolUse Hook (Read) ──────────────────────────────────────────────┐
│ 1. Subagent calls: Read(file_path="src/file.py")                     │
│ 2. PostToolUse hook runs                                             │
│ 3. Detects environment:                                              │
│    ✓ detected_agent = "gemini-2.0-flash" (correct!)                 │
│    ✓ HTMLGRAPH_MODEL = "claude-sonnet" (parent's model)             │
│    ✓ HTMLGRAPH_PARENT_EVENT = "event-123" (correct!)                │
│                                                                      │
│    ❌ ISSUE: No subagent context variables set                        │
│       os.environ.get("HTMLGRAPH_SUBAGENT_TYPE") → None              │
│       os.environ.get("HTMLGRAPH_PARENT_SESSION") → None             │
│                                                                      │
│ 4. Flow to session ID resolution:                                   │
│    manager = SessionManager(".wipnote")                           │
│    active_session = manager.get_active_session()                    │
│                                                                      │
│    ❌ BUG: get_active_session() reads .wipnote/session.json       │
│          (the GLOBAL cache written by orchestrator)                 │
│          Returns: session-abc123 (PARENT'S SESSION!)                │
│                                                                      │
│ 5. Records Read event to session-abc123 (WRONG!)                    │
│    - session_id: session-abc123 ❌ Should be new subagent session   │
│    - tool_name: Read ✓                                              │
│    - model: gemini-2.0-flash ✓ (correct model detected)            │
│    - parent_event_id: event-123 ✓                                   │
│                                                                      │
│ 6. Result: Event appears in parent session with subagent model      │
│    This is confusing! Looks like orchestrator has a "Read" event    │
│    but with Gemini model?                                           │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼ (Same pattern for each tool)

┌─ PostToolUse Hook (Grep, Edit, Write, etc.) ──────────────────────────┐
│ Same issue: All events recorded to session-abc123 (parent session)  │
│ No subagent session ever created                                    │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

╔══════════════════════════════════════════════════════════════════════════╗
║                        DATABASE STATE (BROKEN)                           ║
╚══════════════════════════════════════════════════════════════════════════╝

┌─ Sessions Table ────────────────────────────────────────────────────────┐
│ session_id      | agent_assigned  | is_subagent | parent_session_id    │
│ session-abc123  | orchestrator     | 0           | NULL                 │
│ (No subagent session created!)                                         │
└────────────────────────────────────────────────────────────────────────┘

┌─ Agent Events Table ────────────────────────────────────────────────────┐
│ event_id        | session_id      | tool_name | model              │
│ event-task-001  | session-abc123  | Task      | claude-sonnet      │
│ event-read-001  | session-abc123  | Read      | gemini-2.0-flash   │ ❌
│ event-grep-001  | session-abc123  | Grep      | gemini-2.0-flash   │ ❌
│ event-edit-001  | session-abc123  | Edit      | gemini-2.0-flash   │ ❌
│                                                                      │
│ All in same session! Cannot distinguish orchestrator from subagent! │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

╔══════════════════════════════════════════════════════════════════════════╗
║                      HTMLGRAPH DASHBOARD (BROKEN)                        ║
╚══════════════════════════════════════════════════════════════════════════╝

Session: session-abc123
├── Agent: orchestrator
├── Events:
│   ├── Task (claude-sonnet)       ← Orchestrator's work ✓
│   ├── Read (gemini-2.0-flash)    ← Subagent's work ❌ WRONG SESSION
│   ├── Grep (gemini-2.0-flash)    ← Subagent's work ❌ WRONG SESSION
│   └── Edit (gemini-2.0-flash)    ← Subagent's work ❌ WRONG SESSION
│
│ PROBLEM: Cannot tell which events are orchestrator vs subagent!
│ Cost analysis: All costs attributed to orchestrator session!
│ Attribution: Impossible to determine who did what!


═══════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════
═══════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════════


```

---

## Fixed Flow (After Applying Fixes)

```
╔══════════════════════════════════════════════════════════════════════════╗
║                    ORCHESTRATOR (Sonnet 4.5)                            ║
╚══════════════════════════════════════════════════════════════════════════╝

[SessionStart, UserPromptSubmit, and Task flows same as before...]

                                    │
                                    ▼

┌─ PreToolUse Hook (Spawner Router) ──────────────────────────────────┐
│ 1. Intercepts Task() call with subagent_type="gemini"              │
│ 2. Checks CLI available                                            │
│                                                                     │
│ ✅ FIX: Prepare subagent context before spawning                    │
│                                                                     │
│    # Get current session (now we do this!)                         │
│    current_session = manager.get_active_session()                  │
│    current_session_id = current_session.id  # session-abc123       │
│                                                                     │
│    # Enhanced environment setup:                                   │
│    env = {                                                          │
│      "HTMLGRAPH_PARENT_EVENT": "event-123",         ✓ (already)    │
│      "HTMLGRAPH_MODEL": "claude-sonnet",            ✓ (already)    │
│      "HTMLGRAPH_PROJECT_ROOT": "/path/to/project",  ✓ (already)    │
│                                                                     │
│      "HTMLGRAPH_SUBAGENT_TYPE": "gemini",           ✅ NEW!       │
│      "HTMLGRAPH_PARENT_SESSION": "session-abc123",  ✅ NEW!       │
│      "HTMLGRAPH_PARENT_AGENT": "orchestrator",      ✅ NEW!       │
│    }                                                                │
│                                                                     │
│ 3. Spawns subprocess with complete environment                    │
└──────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

╔══════════════════════════════════════════════════════════════════════════╗
║                  SUBAGENT PROCESS (Gemini 2.0-Flash)                     ║
║                     (Separate Python Process)                            ║
╚══════════════════════════════════════════════════════════════════════════╝

┌─ PostToolUse Hook (Read) ──────────────────────────────────────────────┐
│ 1. Subagent calls: Read(file_path="src/file.py")                    │
│ 2. PostToolUse hook runs                                            │
│ 3. Detects environment:                                             │
│    ✓ detected_agent = "gemini-2.0-flash"                            │
│    ✓ HTMLGRAPH_PARENT_EVENT = "event-123"                           │
│    ✓ HTMLGRAPH_SUBAGENT_TYPE = "gemini"          ✅ NOW PRESENT!    │
│    ✓ HTMLGRAPH_PARENT_SESSION = "session-abc123" ✅ NOW PRESENT!    │
│                                                                      │
│ 4. ✅ FIX: Check for subagent context BEFORE get_active_session()  │
│                                                                      │
│    is_subagent = os.environ.get("HTMLGRAPH_SUBAGENT_TYPE") != None  │
│    parent_session_id = os.environ.get("HTMLGRAPH_PARENT_SESSION")   │
│                                                                      │
│    if is_subagent and parent_session_id:  # ✅ TRUE!               │
│        # Create NEW subagent session                               │
│        active_session = manager.start_session(                     │
│            session_id=None,                                         │
│            agent="gemini-2.0-flash",                                │
│            is_subagent=True,                                        │
│            parent_session_id="session-abc123",  # Links to parent! │
│            title="gemini-2.0-flash (subagent)"                     │
│        )                                                            │
│        # Result: session-xyz789 created with parent linkage        │
│    else:                                                            │
│        # Normal flow (not taken in this case)                      │
│        active_session = manager.get_active_session()               │
│                                                                      │
│ 5. Records Read event to session-xyz789 (CORRECT!)                 │
│    - session_id: session-xyz789 ✅ New subagent session            │
│    - tool_name: Read ✓                                             │
│    - model: gemini-2.0-flash ✓                                     │
│    - parent_event_id: event-123 ✓                                  │
│    - created_at: now                                               │
│                                                                      │
│ 6. Result: Event correctly attributed to subagent session!         │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼ (Same pattern for each tool)

┌─ PostToolUse Hook (Grep, Edit, Write, etc.) ──────────────────────────┐
│ Same logic applies: All events recorded to session-xyz789          │
│ (the NEW subagent session created on first tool call)             │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

╔══════════════════════════════════════════════════════════════════════════╗
║                        DATABASE STATE (FIXED)                            ║
╚══════════════════════════════════════════════════════════════════════════╝

┌─ Sessions Table ────────────────────────────────────────────────────────┐
│ session_id      | agent_assigned  | is_subagent | parent_session_id    │
│ session-abc123  | orchestrator     | 0           | NULL                 │
│ session-xyz789  | gemini-2.0-flash| 1           | session-abc123  ✅ │
│                                                                      │
│ TWO SESSIONS! Clear separation!                                    │
└────────────────────────────────────────────────────────────────────────┘

┌─ Agent Events Table ────────────────────────────────────────────────────┐
│ event_id        | session_id      | tool_name | model          │
│ event-task-001  | session-abc123  | Task      | claude-sonnet   │ ✅
│ event-read-001  | session-xyz789  | Read      | gemini-2.0-... │ ✅
│ event-grep-001  | session-xyz789  | Grep      | gemini-2.0-... │ ✅
│ event-edit-001  | session-xyz789  | Edit      | gemini-2.0-... │ ✅
│                                                                      │
│ Events in correct sessions! Clear attribution!                    │
└────────────────────────────────────────────────────────────────────────┘

                                    │
                                    ▼

╔══════════════════════════════════════════════════════════════════════════╗
║                      HTMLGRAPH DASHBOARD (FIXED)                         ║
╚══════════════════════════════════════════════════════════════════════════╝

Session: session-abc123 (Orchestrator)
├── Agent: orchestrator
├── Model: claude-sonnet
├── Parent: (none)
├── Events:
│   └── Task (claude-sonnet) - Spawned subagent  ✓
│
├── Child Sessions:
│   └── session-xyz789 (Click to view details)

Session: session-xyz789 (Subagent) ✅ SEPARATE SESSION
├── Agent: gemini-2.0-flash
├── Model: gemini-2.0-flash
├── Parent: session-abc123
├── Events:
│   ├── Read (gemini-2.0-flash)    ← Subagent's work ✓
│   ├── Grep (gemini-2.0-flash)    ← Subagent's work ✓
│   └── Edit (gemini-2.0-flash)    ← Subagent's work ✓
│
├── Cost Analysis:
│   ├── Orchestrator: $X (Sonnet 4.5)
│   └── Subagent: $Y (Gemini FREE)

RESULT: Clear separation! Correct attribution! Accurate cost analysis!
```

---

## Session Lifecycle Timeline

### Current (Broken)

```
┌─────────────────────────────────────────────────────────────┐
│                       GLOBAL CACHE                         │
│             .wipnote/session.json                        │
│                                                             │
│  session-abc123  ← Written once, never updated!           │
└─────────────────────────────────────────────────────────────┘

Timeline:
─────────────────────────────────────────────────────────────

T0: Orchestrator SessionStart Hook
    ┌─ Write to session.json
    │  session.json = {id: "session-abc123", agent: "orchestrator"}
    └─ All future reads get session-abc123

T1: Orchestrator UserPromptSubmit
    ├─ manager.get_active_session() → session-abc123 ✓
    └─ Record UserQuery to session-abc123 ✓

T2: Orchestrator Task()
    ├─ manager.get_active_session() → session-abc123 ✓
    └─ Record Task to session-abc123 ✓
    └─ Spawn Gemini subagent (environment incomplete) ✗

T3: Subagent Read() [NEW PROCESS]
    ├─ manager.get_active_session() → session-abc123 ✗ WRONG!
    │  (Reads session.json written by orchestrator)
    └─ Record Read to session-abc123 ✗ SHOULD BE NEW SESSION!

T4: Subagent Grep()
    ├─ manager.get_active_session() → session-abc123 ✗ WRONG!
    └─ Record Grep to session-abc123 ✗ SHOULD BE NEW SESSION!

T5: Subagent Edit()
    ├─ manager.get_active_session() → session-abc123 ✗ WRONG!
    └─ Record Edit to session-abc123 ✗ SHOULD BE NEW SESSION!

RESULT: All events in one session! ✗
```

### Fixed

```
┌─────────────────────────────────────────────────────────────┐
│                   ENVIRONMENT VARIABLES                     │
│                  (Per-Process Context)                      │
│                                                             │
│  ORCHESTRATOR:                 SUBAGENT:                    │
│  [not set initially]           HTMLGRAPH_SUBAGENT_TYPE=gemini
│                                HTMLGRAPH_PARENT_SESSION=    │
│                                session-abc123               │
│                                HTMLGRAPH_PARENT_AGENT=      │
│                                orchestrator                 │
└─────────────────────────────────────────────────────────────┘

Timeline:
─────────────────────────────────────────────────────────────

T0: Orchestrator SessionStart Hook
    ├─ manager.start_session() → session-abc123
    └─ Cache in memory (NOT persistent global cache)

T1: Orchestrator UserPromptSubmit
    ├─ get_active_session() → session-abc123 ✓
    └─ Record UserQuery to session-abc123 ✓

T2: Orchestrator Task()
    ├─ get_active_session() → session-abc123 ✓
    └─ Record Task to session-abc123 ✓
    └─ PreToolUse hook:
       ├─ Reads current_session = session-abc123 ✓
       ├─ Sets env["HTMLGRAPH_SUBAGENT_TYPE"] = "gemini" ✓
       ├─ Sets env["HTMLGRAPH_PARENT_SESSION"] = "session-abc123" ✓
       └─ Spawns Gemini with complete environment ✓

T3: Subagent Read() [NEW PROCESS WITH ENV VARS]
    ├─ Checks: HTMLGRAPH_SUBAGENT_TYPE = "gemini" ✓
    ├─ Checks: HTMLGRAPH_PARENT_SESSION = "session-abc123" ✓
    ├─ Creates NEW session:
    │  └─ manager.start_session(
    │       is_subagent=True,
    │       parent_session_id="session-abc123"
    │     ) → session-xyz789 ✓
    └─ Record Read to session-xyz789 ✓

T4: Subagent Grep()
    ├─ get_active_session() → session-xyz789 (now cached) ✓
    └─ Record Grep to session-xyz789 ✓

T5: Subagent Edit()
    ├─ get_active_session() → session-xyz789 (still cached) ✓
    └─ Record Edit to session-xyz789 ✓

RESULT: Events in separate sessions with parent-child link! ✓
```

---

## Environment Variable Flow

### Current (Incomplete)

```
PreToolUse Hook                    Subprocess
─────────────────                  ──────────

Setup env:
  HTMLGRAPH_PARENT_EVENT ────────→ ✓ Used by track_event
  HTMLGRAPH_MODEL ───────────────→ ✓ Used for model detection
  HTMLGRAPH_PROJECT_ROOT ───────→ ✓ Used for DB path

  ✗ HTMLGRAPH_SUBAGENT_TYPE       (NOT SET)
  ✗ HTMLGRAPH_PARENT_SESSION      (NOT SET)
  ✗ HTMLGRAPH_PARENT_AGENT        (NOT SET)

Subprocess doesn't know:
  - It's a subagent
  - Parent session ID
  - Parent agent name

Result: Uses global cache, gets parent session!
```

### Fixed

```
PreToolUse Hook                    Subprocess
─────────────────                  ──────────

Setup env:
  ✓ HTMLGRAPH_PARENT_EVENT ─────→ Used by track_event
  ✓ HTMLGRAPH_MODEL ────────────→ Used for model detection
  ✓ HTMLGRAPH_PROJECT_ROOT ────→ Used for DB path

  ✅ HTMLGRAPH_SUBAGENT_TYPE ───→ NEW: Signals "create new session"
  ✅ HTMLGRAPH_PARENT_SESSION ──→ NEW: Provides parent session ID
  ✅ HTMLGRAPH_PARENT_AGENT ────→ NEW: Provides parent agent name

Subprocess knows:
  ✓ It's a subagent (HTMLGRAPH_SUBAGENT_TYPE)
  ✓ Parent session ID (HTMLGRAPH_PARENT_SESSION)
  ✓ Parent agent (HTMLGRAPH_PARENT_AGENT)

track_event() detects subagent context:
  if os.environ.get("HTMLGRAPH_SUBAGENT_TYPE"):
      # Create NEW session with parent link

Result: Separate subagent session created! ✓
```

# Phase 2 Feature 1: Smart Delegation Suggestions - Implementation Specification

**Goal:** Proactive delegation suggestions in orchestrator mode that educate users and drive adoption.

**Effort:** 1-2 weeks

**Status:** Ready for Implementation

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Pattern Detection](#pattern-detection)
4. [Suggestion Engine](#suggestion-engine)
5. [User Preferences](#user-preferences)
6. [Database Schema](#database-schema)
7. [Implementation Plan](#implementation-plan)
8. [Testing Strategy](#testing-strategy)
9. [Success Criteria](#success-criteria)

---

## Overview

### Current State

Orchestrator mode currently **blocks** operations with reflection messages:

```
🚫 ORCHESTRATOR MODE VIOLATION (1/3): Multiple Read calls detected.
⚠️  WARNING: Direct operations waste context and break delegation pattern!

Suggested delegation:
Task(
    prompt='''Find pattern in codebase...''',
    subagent_type='Explore'
)
```

**Problem:** Users hit violations **after** they've already performed the work.

### Target State

Smart suggestion system that **proactively suggests** delegation **before** patterns emerge:

```
⚠️ ORCHESTRATOR: You've read 5 files exploring the auth system.

SUGGESTED DELEGATION:
Task(
    prompt="""
    Analyze the authentication system and document:
    1. All authentication endpoints
    2. Token flow and refresh logic
    3. Integration points with other modules

    Return a structured summary.
    """,
    subagent_type="general-purpose"
)

[Y]es, run  [N]o, continue  [A]lways delegate exploration  [?]Learn more
```

**Benefits:**
- ✅ **Proactive** - Suggests delegation before violations occur
- ✅ **Contextual** - Task descriptions generated from user's activities
- ✅ **Educational** - Shows proper delegation patterns
- ✅ **Adaptive** - Learns user preferences over time

---

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     PreToolUse Hook                         │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  1. Pattern Detector                                  │  │
│  │     - Reads tool history from database                │  │
│  │     - Detects exploration/implementation patterns     │  │
│  │     - Returns pattern type + confidence               │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  2. Suggestion Engine                                 │  │
│  │     - Generates Task() call with contextual prompt    │  │
│  │     - Selects appropriate subagent type               │  │
│  │     - Formats as copy-paste ready code                │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  3. Preference Manager                                │  │
│  │     - Checks user's preference history                │  │
│  │     - Auto-delegates if "always" preference set       │  │
│  │     - Stores acceptance/rejection for learning        │  │
│  └──────────────────────────────────────────────────────┘  │
│                           ↓                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  4. Response Formatter                                │  │
│  │     - Shows suggestion with interactive prompt        │  │
│  │     - Supports [Y]es/[N]o/[A]lways/[?]Learn more      │  │
│  │     - Updates database on user response               │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### File Structure

```
src/python/htmlgraph/orchestration/
├── suggestion_engine.py        # NEW - Core suggestion logic
├── pattern_detector.py         # NEW - Pattern detection
├── preference_manager.py       # NEW - User preference storage
└── formatters.py               # NEW - Response formatting

src/python/htmlgraph/hooks/
├── orchestrator.py             # MODIFY - Integrate suggestion engine
└── pretooluse.py               # MODIFY - Add suggestion check

tests/python/
├── test_suggestion_engine.py   # NEW - Engine tests
├── test_pattern_detector.py    # NEW - Pattern detection tests
├── test_preference_manager.py  # NEW - Preference tests
└── test_suggestion_integration.py # NEW - End-to-end tests
```

---

## Pattern Detection

### Detectable Patterns

#### 1. Exploration Pattern

**Trigger:**
- Read >3 files in sequence (last 5 tool calls)
- Grep >2 times (last 4 tool calls)
- Mixed Read/Grep/Glob >5 times (last 7 tool calls)

**Confidence:**
- High (0.9): 5+ exploration calls in last 7 calls
- Medium (0.7): 3-4 exploration calls in last 5 calls
- Low (0.5): 2 exploration calls in last 3 calls

**Example Detection:**

```python
history = [
    {"tool": "Read", "timestamp": "..."},
    {"tool": "Grep", "timestamp": "..."},
    {"tool": "Read", "timestamp": "..."},
    {"tool": "Read", "timestamp": "..."},
    {"tool": "Glob", "timestamp": "..."},
]
# Result: Exploration pattern, confidence=0.9
```

#### 2. Implementation Pattern

**Trigger:**
- Multiple Edit/Write calls (>2 different files)
- Mixed Edit + Read pattern (editing after reading)
- NotebookEdit sequences

**Confidence:**
- High (0.9): 3+ edits to different files
- Medium (0.7): 2+ edits + reads
- Low (0.5): 1 edit after multiple reads

#### 3. Debugging Pattern

**Trigger:**
- Test runs with failures (pytest, npm test)
- Bash error codes (exit code != 0)
- Read + Edit + Bash cycle

**Confidence:**
- High (0.9): Failed test + subsequent edits
- Medium (0.7): Error in Bash + Read pattern
- Low (0.5): Single failed test

#### 4. Refactoring Pattern

**Trigger:**
- Multiple Edit calls to same file
- Mixed Read/Edit across multiple related files
- File renames/moves

**Confidence:**
- High (0.9): 4+ edits across related files
- Medium (0.7): 2-3 edits to same file
- Low (0.5): Single large edit operation

### Pattern Detector Implementation

```python
# src/python/htmlgraph/orchestration/pattern_detector.py

from dataclasses import dataclass
from typing import Literal

@dataclass
class Pattern:
    """Detected pattern from tool history."""

    pattern_type: Literal["exploration", "implementation", "debugging", "refactoring"]
    confidence: float  # 0.0-1.0
    tool_sequence: list[str]  # Tools that triggered pattern
    file_paths: list[str]  # Files involved
    description: str  # Human-readable pattern description


class PatternDetector:
    """Detects delegation-worthy patterns from tool call history."""

    def __init__(self, db_path: str):
        self.db = HtmlGraphDB(db_path)

    def detect_pattern(
        self,
        session_id: str,
        current_tool: str,
        lookback: int = 10
    ) -> Pattern | None:
        """
        Detect if current tool call is part of a delegation-worthy pattern.

        Args:
            session_id: Current session ID
            current_tool: Tool about to be called
            lookback: How many previous tool calls to analyze

        Returns:
            Pattern if detected, None otherwise
        """
        # Load recent tool history
        history = self._load_tool_history(session_id, lookback)
        history.append({"tool": current_tool, "timestamp": datetime.now()})

        # Check each pattern type (order matters - most specific first)
        for detector in [
            self._detect_exploration,
            self._detect_implementation,
            self._detect_debugging,
            self._detect_refactoring,
        ]:
            pattern = detector(history)
            if pattern and pattern.confidence >= 0.5:  # Confidence threshold
                return pattern

        return None

    def _detect_exploration(self, history: list[dict]) -> Pattern | None:
        """Detect exploration pattern (Read/Grep/Glob sequences)."""
        exploration_tools = ["Read", "Grep", "Glob"]
        recent = history[-7:]  # Last 7 tool calls

        exploration_count = sum(
            1 for h in recent if h["tool"] in exploration_tools
        )

        if exploration_count >= 5:
            confidence = 0.9
        elif exploration_count >= 3:
            confidence = 0.7
        elif exploration_count >= 2:
            confidence = 0.5
        else:
            return None

        file_paths = self._extract_file_paths(recent)

        return Pattern(
            pattern_type="exploration",
            confidence=confidence,
            tool_sequence=[h["tool"] for h in recent if h["tool"] in exploration_tools],
            file_paths=file_paths,
            description=f"Exploring codebase ({exploration_count} lookups in {len(recent)} calls)"
        )

    def _detect_implementation(self, history: list[dict]) -> Pattern | None:
        """Detect implementation pattern (Edit/Write sequences)."""
        impl_tools = ["Edit", "Write", "NotebookEdit"]
        recent = history[-5:]  # Last 5 tool calls

        impl_count = sum(1 for h in recent if h["tool"] in impl_tools)
        unique_files = len(set(self._extract_file_paths(recent)))

        if impl_count >= 3 and unique_files >= 2:
            confidence = 0.9
        elif impl_count >= 2:
            confidence = 0.7
        elif impl_count >= 1 and len(recent) >= 3:
            # Single edit after multiple reads
            read_count = sum(1 for h in recent[:-1] if h["tool"] == "Read")
            if read_count >= 2:
                confidence = 0.5
            else:
                return None
        else:
            return None

        file_paths = self._extract_file_paths(recent)

        return Pattern(
            pattern_type="implementation",
            confidence=confidence,
            tool_sequence=[h["tool"] for h in recent if h["tool"] in impl_tools],
            file_paths=file_paths,
            description=f"Implementing changes ({impl_count} edits across {unique_files} files)"
        )

    def _detect_debugging(self, history: list[dict]) -> Pattern | None:
        """Detect debugging pattern (test failures + edits)."""
        recent = history[-5:]

        # Check for failed Bash commands (tests)
        failed_tests = [
            h for h in recent
            if h["tool"] == "Bash" and self._is_test_command(h) and self._has_error(h)
        ]

        if not failed_tests:
            return None

        # Check for subsequent Read/Edit operations
        subsequent_ops = [
            h for h in recent[recent.index(failed_tests[0]) + 1:]
            if h["tool"] in ["Read", "Edit", "Write"]
        ]

        if len(subsequent_ops) >= 2:
            confidence = 0.9
        elif len(subsequent_ops) == 1:
            confidence = 0.7
        else:
            confidence = 0.5

        return Pattern(
            pattern_type="debugging",
            confidence=confidence,
            tool_sequence=[h["tool"] for h in recent],
            file_paths=self._extract_file_paths(subsequent_ops),
            description=f"Debugging failed tests ({len(failed_tests)} failures)"
        )

    def _detect_refactoring(self, history: list[dict]) -> Pattern | None:
        """Detect refactoring pattern (multiple edits to related files)."""
        recent = history[-6:]

        edit_tools = ["Edit", "Write"]
        edits = [h for h in recent if h["tool"] in edit_tools]

        if not edits:
            return None

        file_paths = self._extract_file_paths(edits)
        unique_files = len(set(file_paths))

        # Check for same file edits
        if len(edits) >= 3 and unique_files <= 2:
            confidence = 0.9
            description = f"Refactoring (multiple edits to {unique_files} files)"
        # Check for related file edits
        elif len(edits) >= 2 and self._are_related_files(file_paths):
            confidence = 0.7
            description = f"Refactoring related files ({unique_files} files)"
        else:
            return None

        return Pattern(
            pattern_type="refactoring",
            confidence=confidence,
            tool_sequence=[h["tool"] for h in edits],
            file_paths=file_paths,
            description=description
        )

    def _load_tool_history(self, session_id: str, limit: int) -> list[dict]:
        """Load recent tool calls from database."""
        cursor = self.db.connection.cursor()
        cursor.execute(
            """
            SELECT tool_name, timestamp, context
            FROM agent_events
            WHERE session_id = ? AND tool_name IS NOT NULL
            ORDER BY timestamp DESC
            LIMIT ?
            """,
            (session_id, limit),
        )

        rows = cursor.fetchall()
        return [
            {
                "tool": row[0],
                "timestamp": row[1],
                "context": json.loads(row[2]) if row[2] else {},
            }
            for row in reversed(rows)  # Oldest first
        ]

    def _extract_file_paths(self, history: list[dict]) -> list[str]:
        """Extract file paths from tool history."""
        paths = []
        for h in history:
            context = h.get("context", {})
            if "file_paths" in context:
                paths.extend(context["file_paths"])
        return paths

    def _is_test_command(self, tool_call: dict) -> bool:
        """Check if Bash command is a test command."""
        context = tool_call.get("context", {})
        command = context.get("command", "")
        test_patterns = ["pytest", "npm test", "cargo test", "mvn test"]
        return any(p in command for p in test_patterns)

    def _has_error(self, tool_call: dict) -> bool:
        """Check if tool call resulted in error."""
        context = tool_call.get("context", {})
        return context.get("is_error", False)

    def _are_related_files(self, file_paths: list[str]) -> bool:
        """Check if files are related (same directory or module)."""
        if len(file_paths) < 2:
            return False

        # Simple heuristic: same parent directory
        from pathlib import Path
        parents = [Path(p).parent for p in file_paths]
        return len(set(parents)) == 1
```

---

## Suggestion Engine

### Task Generation

The suggestion engine generates contextual `Task()` calls based on detected patterns:

```python
# src/python/htmlgraph/orchestration/suggestion_engine.py

from dataclasses import dataclass

@dataclass
class Suggestion:
    """A delegation suggestion for the user."""

    task_code: str  # Copy-paste ready Task() call
    explanation: str  # Why this delegation makes sense
    subagent_type: str  # Recommended subagent
    estimated_savings: str  # Context savings estimate


class SuggestionEngine:
    """Generates Task() delegation suggestions from detected patterns."""

    def generate_suggestion(
        self,
        pattern: Pattern,
        tool_history: list[dict]
    ) -> Suggestion:
        """
        Generate a delegation suggestion from detected pattern.

        Args:
            pattern: Detected pattern from PatternDetector
            tool_history: Recent tool call history for context

        Returns:
            Suggestion with Task() code and explanation
        """
        if pattern.pattern_type == "exploration":
            return self._suggest_exploration_delegation(pattern, tool_history)
        elif pattern.pattern_type == "implementation":
            return self._suggest_implementation_delegation(pattern, tool_history)
        elif pattern.pattern_type == "debugging":
            return self._suggest_debugging_delegation(pattern, tool_history)
        elif pattern.pattern_type == "refactoring":
            return self._suggest_refactoring_delegation(pattern, tool_history)
        else:
            return self._suggest_generic_delegation(pattern)

    def _suggest_exploration_delegation(
        self,
        pattern: Pattern,
        tool_history: list[dict]
    ) -> Suggestion:
        """Generate exploration delegation suggestion."""

        # Extract context from tool history
        files_explored = pattern.file_paths[:5]  # Limit to 5 for brevity
        exploration_goal = self._infer_exploration_goal(tool_history)

        task_code = f'''Task(
    prompt="""
    Explore and document {exploration_goal}.

    Files involved:
    {self._format_file_list(files_explored)}

    Please provide:
    1. Overview of functionality
    2. Key components and their relationships
    3. Any issues or concerns discovered

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='explorer')
    sdk.spikes.create('Exploration Results') \\
        .set_findings('Summary of findings...') \\
        .save()
    """,
    subagent_type="general-purpose"
)'''

        explanation = (
            f"You've explored {len(pattern.tool_sequence)} files. "
            f"Delegation would save ~{self._estimate_token_savings(pattern)} tokens of context."
        )

        return Suggestion(
            task_code=task_code,
            explanation=explanation,
            subagent_type="general-purpose",
            estimated_savings=self._estimate_token_savings(pattern)
        )

    def _suggest_implementation_delegation(
        self,
        pattern: Pattern,
        tool_history: list[dict]
    ) -> Suggestion:
        """Generate implementation delegation suggestion."""

        files_to_edit = pattern.file_paths
        impl_goal = self._infer_implementation_goal(tool_history)

        task_code = f'''Task(
    prompt="""
    Implement {impl_goal}.

    Files to modify:
    {self._format_file_list(files_to_edit)}

    Requirements:
    1. Make necessary code changes
    2. Run tests to verify
    3. Report any issues encountered

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='coder')
    sdk.spikes.create('Implementation Complete') \\
        .set_findings('Changes made and test results...') \\
        .save()
    """,
    subagent_type="general-purpose"
)'''

        explanation = (
            f"You're implementing changes across {len(set(files_to_edit))} files. "
            f"Delegation saves context and isolates implementation work."
        )

        return Suggestion(
            task_code=task_code,
            explanation=explanation,
            subagent_type="general-purpose",
            estimated_savings=self._estimate_token_savings(pattern)
        )

    def _suggest_debugging_delegation(
        self,
        pattern: Pattern,
        tool_history: list[dict]
    ) -> Suggestion:
        """Generate debugging delegation suggestion."""

        # Find failed test command
        failed_test = next(
            (h for h in tool_history if h.get("tool") == "Bash" and "test" in str(h.get("context", {}).get("command", ""))),
            None
        )

        test_command = failed_test.get("context", {}).get("command", "pytest") if failed_test else "pytest"

        task_code = f'''Task(
    prompt="""
    Debug and fix test failures.

    Failed test command: {test_command}

    Tasks:
    1. Run tests and capture full output
    2. Identify root cause of failures
    3. Implement fixes
    4. Verify all tests pass

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='debugger')
    sdk.spikes.create('Debug Results') \\
        .set_findings('Root cause and fixes applied...') \\
        .save()
    """,
    subagent_type="general-purpose"
)'''

        explanation = (
            "Debugging cycles waste context with repeated test runs. "
            "Delegation isolates the debugging work."
        )

        return Suggestion(
            task_code=task_code,
            explanation=explanation,
            subagent_type="general-purpose",
            estimated_savings="high"
        )

    def _suggest_refactoring_delegation(
        self,
        pattern: Pattern,
        tool_history: list[dict]
    ) -> Suggestion:
        """Generate refactoring delegation suggestion."""

        files_to_refactor = list(set(pattern.file_paths))

        task_code = f'''Task(
    prompt="""
    Refactor code in the following files:
    {self._format_file_list(files_to_refactor)}

    Goals:
    1. Improve code structure and readability
    2. Maintain existing functionality
    3. Run tests to verify no regressions

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='refactorer')
    sdk.spikes.create('Refactoring Complete') \\
        .set_findings('Changes made and test results...') \\
        .save()
    """,
    subagent_type="general-purpose"
)'''

        explanation = (
            f"Refactoring {len(files_to_refactor)} files is implementation work. "
            "Delegation keeps your context clean for high-level decisions."
        )

        return Suggestion(
            task_code=task_code,
            explanation=explanation,
            subagent_type="general-purpose",
            estimated_savings="medium"
        )

    def _suggest_generic_delegation(self, pattern: Pattern) -> Suggestion:
        """Fallback generic suggestion."""
        task_code = '''Task(
    prompt="""
    <Describe the task based on your recent activities>

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='subagent')
    sdk.spikes.create('Task Results') \\
        .set_findings('Summary of work...') \\
        .save()
    """,
    subagent_type="general-purpose"
)'''

        return Suggestion(
            task_code=task_code,
            explanation="Consider delegating this work to a subagent.",
            subagent_type="general-purpose",
            estimated_savings="unknown"
        )

    def _infer_exploration_goal(self, tool_history: list[dict]) -> str:
        """Infer what the user is trying to explore."""
        # Simple heuristic: look at file paths and patterns
        file_paths = []
        for h in tool_history:
            context = h.get("context", {})
            if "file_paths" in context:
                file_paths.extend(context["file_paths"])

        if not file_paths:
            return "the codebase"

        # Extract common themes from file paths
        from pathlib import Path
        dirs = [Path(p).parent.name for p in file_paths if p]
        common_dir = max(set(dirs), key=dirs.count) if dirs else "the codebase"

        return f"the {common_dir} module"

    def _infer_implementation_goal(self, tool_history: list[dict]) -> str:
        """Infer what the user is trying to implement."""
        # Look for patterns in file names or recent reads
        file_paths = []
        for h in tool_history:
            context = h.get("context", {})
            if "file_paths" in context:
                file_paths.extend(context["file_paths"])

        if not file_paths:
            return "the required changes"

        # Extract common themes
        from pathlib import Path
        files = [Path(p).stem for p in file_paths if p]

        # Simple heuristic: if files share prefix, use that
        if files:
            common_prefix = os.path.commonprefix(files)
            if common_prefix:
                return f"{common_prefix} functionality"

        return "the required changes"

    def _format_file_list(self, files: list[str]) -> str:
        """Format file list for task prompt."""
        if not files:
            return "(no files specified)"

        return "\n    ".join(f"- {f}" for f in files[:5])

    def _estimate_token_savings(self, pattern: Pattern) -> str:
        """Estimate context token savings from delegation."""
        # Rough heuristic based on tool count and file count
        tool_count = len(pattern.tool_sequence)
        file_count = len(set(pattern.file_paths))

        estimated_tokens = tool_count * 500 + file_count * 1000

        if estimated_tokens > 5000:
            return "high (>5k tokens)"
        elif estimated_tokens > 2000:
            return "medium (2-5k tokens)"
        else:
            return "low (<2k tokens)"
```

---

## User Preferences

### Preference Storage

Store user preferences in database for persistence and learning:

```python
# src/python/htmlgraph/orchestration/preference_manager.py

from enum import Enum

class PreferenceAction(Enum):
    """User response to a suggestion."""
    ACCEPTED = "accepted"  # User ran the suggested Task()
    REJECTED = "rejected"  # User declined the suggestion
    ALWAYS = "always"      # User set "always delegate this pattern"
    NEVER = "never"        # User set "never suggest this pattern"


class PreferenceManager:
    """Manages user preferences for delegation suggestions."""

    def __init__(self, db_path: str):
        self.db = HtmlGraphDB(db_path)
        self._ensure_preferences_table()

    def _ensure_preferences_table(self) -> None:
        """Create preferences table if it doesn't exist."""
        cursor = self.db.connection.cursor()
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS delegation_preferences (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                session_id TEXT NOT NULL,
                pattern_type TEXT NOT NULL,
                action TEXT NOT NULL CHECK(
                    action IN ('accepted', 'rejected', 'always', 'never')
                ),
                confidence REAL NOT NULL,
                timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (session_id) REFERENCES sessions(session_id)
            )
        """)
        self.db.connection.commit()

    def should_suggest(
        self,
        session_id: str,
        pattern_type: str
    ) -> bool:
        """
        Check if we should show a suggestion for this pattern type.

        Returns:
            True if suggestion should be shown, False otherwise
        """
        # Check for "never" preference
        cursor = self.db.connection.cursor()
        cursor.execute(
            """
            SELECT action FROM delegation_preferences
            WHERE session_id = ? AND pattern_type = ? AND action = 'never'
            ORDER BY timestamp DESC
            LIMIT 1
            """,
            (session_id, pattern_type),
        )

        if cursor.fetchone():
            return False  # User said "never suggest this"

        return True

    def should_auto_delegate(
        self,
        session_id: str,
        pattern_type: str
    ) -> bool:
        """
        Check if we should automatically delegate this pattern.

        Returns:
            True if "always" preference is set, False otherwise
        """
        cursor = self.db.connection.cursor()
        cursor.execute(
            """
            SELECT action FROM delegation_preferences
            WHERE session_id = ? AND pattern_type = ? AND action = 'always'
            ORDER BY timestamp DESC
            LIMIT 1
            """,
            (session_id, pattern_type),
        )

        return cursor.fetchone() is not None

    def record_action(
        self,
        session_id: str,
        pattern_type: str,
        action: PreferenceAction,
        confidence: float
    ) -> None:
        """Record user's response to a suggestion."""
        cursor = self.db.connection.cursor()
        cursor.execute(
            """
            INSERT INTO delegation_preferences
            (session_id, pattern_type, action, confidence)
            VALUES (?, ?, ?, ?)
            """,
            (session_id, pattern_type, action.value, confidence),
        )
        self.db.connection.commit()

    def get_acceptance_rate(
        self,
        session_id: str,
        pattern_type: str | None = None
    ) -> float:
        """
        Calculate suggestion acceptance rate.

        Args:
            session_id: Session to analyze
            pattern_type: Optional pattern type filter

        Returns:
            Acceptance rate (0.0-1.0)
        """
        cursor = self.db.connection.cursor()

        if pattern_type:
            cursor.execute(
                """
                SELECT
                    SUM(CASE WHEN action = 'accepted' THEN 1 ELSE 0 END) as accepted,
                    COUNT(*) as total
                FROM delegation_preferences
                WHERE session_id = ? AND pattern_type = ?
                  AND action IN ('accepted', 'rejected')
                """,
                (session_id, pattern_type),
            )
        else:
            cursor.execute(
                """
                SELECT
                    SUM(CASE WHEN action = 'accepted' THEN 1 ELSE 0 END) as accepted,
                    COUNT(*) as total
                FROM delegation_preferences
                WHERE session_id = ?
                  AND action IN ('accepted', 'rejected')
                """,
                (session_id,),
            )

        row = cursor.fetchone()
        if row and row[1] > 0:
            return row[0] / row[1]
        return 0.0
```

### Database Schema Addition

Add to existing schema:

```sql
-- delegation_preferences table
CREATE TABLE IF NOT EXISTS delegation_preferences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    pattern_type TEXT NOT NULL CHECK(
        pattern_type IN ('exploration', 'implementation', 'debugging', 'refactoring')
    ),
    action TEXT NOT NULL CHECK(
        action IN ('accepted', 'rejected', 'always', 'never')
    ),
    confidence REAL NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE
);

-- delegation_suggestions table (track all suggestions shown)
CREATE TABLE IF NOT EXISTS delegation_suggestions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    pattern_type TEXT NOT NULL,
    confidence REAL NOT NULL,
    suggestion_text TEXT NOT NULL,
    user_action TEXT CHECK(
        user_action IN ('accepted', 'rejected', 'always', 'never', 'pending')
    ),
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(session_id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_delegation_preferences_session
    ON delegation_preferences(session_id, pattern_type);

CREATE INDEX IF NOT EXISTS idx_delegation_suggestions_session
    ON delegation_suggestions(session_id, timestamp);
```

---

## Implementation Plan

### Phase 1: Core Pattern Detection (Days 1-3)

**Files:**
- `src/python/htmlgraph/orchestration/pattern_detector.py` (NEW)
- `tests/python/test_pattern_detector.py` (NEW)

**Tasks:**
1. Implement `PatternDetector` class
2. Add exploration pattern detection
3. Add implementation pattern detection
4. Add debugging pattern detection
5. Add refactoring pattern detection
6. Write comprehensive unit tests (aim for 90%+ coverage)

**Success Criteria:**
- ✅ All 4 pattern types detected correctly
- ✅ Confidence scores accurate (validated against manual analysis)
- ✅ Tests pass with >90% coverage
- ✅ Performance: <100ms for pattern detection

### Phase 2: Suggestion Engine (Days 4-6)

**Files:**
- `src/python/htmlgraph/orchestration/suggestion_engine.py` (NEW)
- `tests/python/test_suggestion_engine.py` (NEW)

**Tasks:**
1. Implement `SuggestionEngine` class
2. Add Task() code generation for each pattern type
3. Implement context extraction from tool history
4. Add token savings estimation
5. Write unit tests for all suggestion types

**Success Criteria:**
- ✅ Task() code is syntactically correct
- ✅ Prompts are contextual and actionable
- ✅ Token savings estimates are reasonable
- ✅ Tests pass with >85% coverage

### Phase 3: Preference Management (Days 7-8)

**Files:**
- `src/python/htmlgraph/orchestration/preference_manager.py` (NEW)
- `src/python/htmlgraph/db/schema.py` (MODIFY - add tables)
- `tests/python/test_preference_manager.py` (NEW)

**Tasks:**
1. Implement `PreferenceManager` class
2. Add database schema for preferences
3. Implement preference storage/retrieval
4. Add acceptance rate tracking
5. Write tests for all preference operations

**Success Criteria:**
- ✅ Preferences persist across sessions
- ✅ "Always" and "Never" preferences respected
- ✅ Acceptance rate calculated correctly
- ✅ Database migrations work cleanly

### Phase 4: Hook Integration (Days 9-10)

**Files:**
- `src/python/htmlgraph/hooks/orchestrator.py` (MODIFY)
- `src/python/htmlgraph/hooks/pretooluse.py` (MODIFY)
- `tests/python/test_suggestion_integration.py` (NEW)

**Tasks:**
1. Integrate PatternDetector into PreToolUse hook
2. Add SuggestionEngine to generate suggestions
3. Integrate PreferenceManager for auto-delegation
4. Add interactive prompt for user responses
5. Write end-to-end integration tests

**Success Criteria:**
- ✅ Suggestions show before violations
- ✅ User can accept/reject/set preferences
- ✅ Auto-delegation works for "always" preferences
- ✅ Integration tests pass

### Phase 5: Response Formatting (Days 11-12)

**Files:**
- `src/python/htmlgraph/orchestration/formatters.py` (NEW)
- `tests/python/test_formatters.py` (NEW)

**Tasks:**
1. Implement rich formatting for suggestions
2. Add interactive prompt handling
3. Add "Learn more" documentation
4. Style suggestions for readability
5. Write tests for formatting

**Success Criteria:**
- ✅ Suggestions are visually clear
- ✅ Interactive prompts work correctly
- ✅ "Learn more" provides helpful context
- ✅ Formatting tests pass

### Phase 6: Testing & Refinement (Days 13-14)

**Tasks:**
1. Run full test suite
2. Fix any failing tests
3. Refine confidence thresholds based on testing
4. Add edge case handling
5. Update documentation

**Success Criteria:**
- ✅ All tests pass (>90% coverage overall)
- ✅ No regressions in existing functionality
- ✅ Edge cases handled gracefully
- ✅ Documentation complete

---

## Testing Strategy

### Unit Tests

#### Pattern Detector Tests

```python
# tests/python/test_pattern_detector.py

def test_detect_exploration_pattern_high_confidence():
    """Test exploration pattern detection with high confidence."""
    detector = PatternDetector(":memory:")

    # Simulate 5 Read calls in last 7
    history = [
        {"tool": "Read", "timestamp": "..."},
        {"tool": "Read", "timestamp": "..."},
        {"tool": "Bash", "timestamp": "..."},
        {"tool": "Read", "timestamp": "..."},
        {"tool": "Grep", "timestamp": "..."},
        {"tool": "Read", "timestamp": "..."},
        {"tool": "Read", "timestamp": "..."},
    ]

    pattern = detector._detect_exploration(history)

    assert pattern is not None
    assert pattern.pattern_type == "exploration"
    assert pattern.confidence >= 0.9
    assert len(pattern.tool_sequence) >= 5


def test_detect_implementation_pattern_multiple_files():
    """Test implementation pattern across multiple files."""
    detector = PatternDetector(":memory:")

    history = [
        {"tool": "Read", "timestamp": "...", "context": {"file_paths": ["a.py"]}},
        {"tool": "Edit", "timestamp": "...", "context": {"file_paths": ["a.py"]}},
        {"tool": "Read", "timestamp": "...", "context": {"file_paths": ["b.py"]}},
        {"tool": "Edit", "timestamp": "...", "context": {"file_paths": ["b.py"]}},
        {"tool": "Edit", "timestamp": "...", "context": {"file_paths": ["c.py"]}},
    ]

    pattern = detector._detect_implementation(history)

    assert pattern is not None
    assert pattern.pattern_type == "implementation"
    assert pattern.confidence >= 0.9
    assert len(set(pattern.file_paths)) >= 2


def test_no_pattern_below_threshold():
    """Test that no pattern is detected below confidence threshold."""
    detector = PatternDetector(":memory:")

    history = [
        {"tool": "Read", "timestamp": "..."},
        {"tool": "Bash", "timestamp": "..."},
    ]

    pattern = detector.detect_pattern("session-1", "Read", lookback=10)

    assert pattern is None  # Not enough for any pattern
```

#### Suggestion Engine Tests

```python
# tests/python/test_suggestion_engine.py

def test_generate_exploration_suggestion():
    """Test exploration suggestion generation."""
    engine = SuggestionEngine()

    pattern = Pattern(
        pattern_type="exploration",
        confidence=0.9,
        tool_sequence=["Read", "Grep", "Read", "Read"],
        file_paths=["auth.py", "user.py", "session.py"],
        description="Exploring auth module"
    )

    suggestion = engine.generate_suggestion(pattern, [])

    assert "Task(" in suggestion.task_code
    assert "general-purpose" in suggestion.task_code
    assert "auth" in suggestion.task_code.lower()
    assert len(suggestion.explanation) > 0


def test_task_code_is_syntactically_valid():
    """Test that generated Task() code is valid Python."""
    engine = SuggestionEngine()

    pattern = Pattern(
        pattern_type="implementation",
        confidence=0.8,
        tool_sequence=["Edit", "Edit"],
        file_paths=["test.py"],
        description="Implementing changes"
    )

    suggestion = engine.generate_suggestion(pattern, [])

    # Should be valid Python (no syntax errors)
    try:
        compile(suggestion.task_code, "<string>", "exec")
    except SyntaxError:
        pytest.fail("Generated Task() code has syntax errors")
```

#### Preference Manager Tests

```python
# tests/python/test_preference_manager.py

def test_record_and_retrieve_preference():
    """Test storing and retrieving user preferences."""
    manager = PreferenceManager(":memory:")

    manager.record_action(
        session_id="sess-1",
        pattern_type="exploration",
        action=PreferenceAction.ALWAYS,
        confidence=0.9
    )

    assert manager.should_auto_delegate("sess-1", "exploration") is True
    assert manager.should_suggest("sess-1", "exploration") is True


def test_never_preference_blocks_suggestions():
    """Test that 'never' preference blocks suggestions."""
    manager = PreferenceManager(":memory:")

    manager.record_action(
        session_id="sess-1",
        pattern_type="debugging",
        action=PreferenceAction.NEVER,
        confidence=0.7
    )

    assert manager.should_suggest("sess-1", "debugging") is False


def test_acceptance_rate_calculation():
    """Test acceptance rate calculation."""
    manager = PreferenceManager(":memory:")

    # 3 accepted, 2 rejected = 60% acceptance
    for _ in range(3):
        manager.record_action("sess-1", "exploration", PreferenceAction.ACCEPTED, 0.9)
    for _ in range(2):
        manager.record_action("sess-1", "exploration", PreferenceAction.REJECTED, 0.8)

    rate = manager.get_acceptance_rate("sess-1", "exploration")

    assert rate == pytest.approx(0.6, abs=0.01)
```

### Integration Tests

```python
# tests/python/test_suggestion_integration.py

def test_end_to_end_suggestion_flow():
    """Test complete suggestion flow from pattern detection to user response."""
    # Setup
    db = HtmlGraphDB(":memory:")
    detector = PatternDetector(str(db.db_path))
    engine = SuggestionEngine()
    preferences = PreferenceManager(str(db.db_path))

    # Simulate tool history leading to pattern
    session_id = "sess-test"
    for i in range(5):
        db.insert_event(
            event_id=f"evt-{i}",
            agent_id="claude",
            event_type="tool_call",
            session_id=session_id,
            tool_name="Read",
            input_summary=f"Read file-{i}.py",
            output_summary="success",
            context={"file_paths": [f"file-{i}.py"]},
        )

    # Detect pattern
    pattern = detector.detect_pattern(session_id, "Read", lookback=10)

    assert pattern is not None
    assert pattern.pattern_type == "exploration"

    # Generate suggestion
    suggestion = engine.generate_suggestion(pattern, [])

    assert "Task(" in suggestion.task_code

    # Record user acceptance
    preferences.record_action(
        session_id=session_id,
        pattern_type=pattern.pattern_type,
        action=PreferenceAction.ACCEPTED,
        confidence=pattern.confidence
    )

    # Verify recorded
    rate = preferences.get_acceptance_rate(session_id)
    assert rate == 1.0


def test_auto_delegation_with_always_preference():
    """Test automatic delegation when 'always' preference is set."""
    preferences = PreferenceManager(":memory:")

    # Set "always" preference
    preferences.record_action(
        session_id="sess-1",
        pattern_type="implementation",
        action=PreferenceAction.ALWAYS,
        confidence=0.9
    )

    # Should auto-delegate
    assert preferences.should_auto_delegate("sess-1", "implementation") is True
```

---

## Success Criteria

### Functional Requirements

- ✅ **Pattern Detection**: All 4 pattern types detected with >85% accuracy
- ✅ **Suggestion Quality**: Task() code is syntactically correct and contextual
- ✅ **User Preferences**: "Always" and "Never" preferences persist and work correctly
- ✅ **Auto-Delegation**: "Always" preferences trigger automatic delegation
- ✅ **Performance**: Pattern detection completes in <100ms
- ✅ **Database**: Preferences and suggestions stored persistently

### Quality Requirements

- ✅ **Test Coverage**: >90% coverage across all new modules
- ✅ **No Regressions**: All existing tests continue to pass
- ✅ **Documentation**: All public APIs documented with examples
- ✅ **Code Quality**: Passes ruff, mypy, and pylint checks

### User Experience Requirements

- ✅ **Proactive**: Suggestions appear before violations occur
- ✅ **Contextual**: Task descriptions reflect user's actual work
- ✅ **Non-Intrusive**: Easy to accept or dismiss suggestions
- ✅ **Educational**: Helps users learn proper delegation patterns
- ✅ **Respectful**: Honors user preferences (always/never)

### Analytics Requirements

- ✅ **Track Suggestions**: All suggestions logged to database
- ✅ **Track Responses**: User actions (accept/reject/always/never) recorded
- ✅ **Acceptance Rate**: Calculate per-pattern and overall acceptance rates
- ✅ **Pattern Frequency**: Track which patterns occur most often

---

## Future Enhancements (Phase 3+)

### ML-Based Pattern Detection

Replace rule-based detection with ML model:
- Train on historical tool sequences
- Predict delegation opportunities with higher accuracy
- Adapt to individual user patterns

### Contextual Prompt Generation

Use LLM to generate Task() prompts:
- Analyze file contents (not just names)
- Infer user intent from recent prompts
- Generate more specific and actionable prompts

### Cross-Session Learning

Learn patterns across all users:
- Aggregate preference data
- Identify universally beneficial delegation patterns
- Suggest emerging best practices

### Dashboard Analytics

Add delegation analytics to dashboard:
- Acceptance rate over time
- Context savings per session
- Most/least effective pattern types
- User delegation adoption trends

---

## Appendix: Example Suggestions

### Exploration Pattern

**Detected After:** 5 Read calls exploring auth module

**Suggestion:**

```
⚠️ ORCHESTRATOR: You've read 5 files exploring the auth system.

SUGGESTED DELEGATION:
Task(
    prompt="""
    Analyze the authentication system and document:
    1. All authentication endpoints
    2. Token flow and refresh logic
    3. Integration points with other modules

    Files explored:
    - src/auth/routes.py
    - src/auth/middleware.py
    - src/auth/models.py
    - src/auth/utils.py
    - src/auth/tokens.py

    Return a structured summary.

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='explorer')
    sdk.spikes.create('Auth System Analysis') \
        .set_findings('Summary of findings...') \
        .save()
    """,
    subagent_type="general-purpose"
)

Delegation saves ~3-5k tokens of context.

[Y]es, run  [N]o, continue  [A]lways delegate exploration  [?]Learn more
```

### Implementation Pattern

**Detected After:** 3 Edit calls across different files

**Suggestion:**

```
⚠️ ORCHESTRATOR: You're implementing changes across 3 files.

SUGGESTED DELEGATION:
Task(
    prompt="""
    Implement the required changes to:
    - src/api/routes.py (add endpoint)
    - src/models/user.py (add field)
    - tests/test_api.py (add test)

    Requirements:
    1. Add new user profile endpoint
    2. Update User model with profile_data field
    3. Write integration test
    4. Run tests to verify

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='coder')
    sdk.spikes.create('Implementation Complete') \
        .set_findings('Changes made and test results...') \
        .save()
    """,
    subagent_type="general-purpose"
)

Delegation isolates implementation work from strategic planning.

[Y]es, run  [N]o, continue  [A]lways delegate implementation  [?]Learn more
```

### Debugging Pattern

**Detected After:** Failed test run + subsequent edits

**Suggestion:**

```
⚠️ ORCHESTRATOR: Test failures detected. Debugging cycles waste context.

SUGGESTED DELEGATION:
Task(
    prompt="""
    Debug and fix test failures from: pytest tests/test_auth.py

    Tasks:
    1. Run tests and capture full error output
    2. Identify root cause
    3. Implement fix
    4. Verify all tests pass

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='debugger')
    sdk.spikes.create('Debug Results') \
        .set_findings('Root cause and fixes applied...') \
        .save()
    """,
    subagent_type="general-purpose"
)

Delegation prevents context pollution from test run outputs.

[Y]es, run  [N]o, continue  [A]lways delegate debugging  [?]Learn more
```

---

## Questions & Decisions

### Q: Should suggestions be shown in strict mode only?

**A:** No, show in both strict and guidance modes. In guidance mode, they're purely educational. In strict mode, they help avoid violations.

### Q: What if user ignores suggestion and hits violation anyway?

**A:** Still block with violation message, but note: "A suggestion was provided earlier. Consider using it next time."

### Q: How to handle rapid-fire suggestions?

**A:** Add cooldown (e.g., max 1 suggestion per 5 minutes) to avoid spamming user.

### Q: Should we suggest delegation for single tool calls?

**A:** No, only suggest after patterns emerge (confidence >= 0.5). Single calls don't warrant delegation overhead.

### Q: How to measure success of this feature?

**A:** Track metrics:
1. Suggestion acceptance rate (target: >40%)
2. Violation reduction (target: 30% fewer violations)
3. User feedback (qualitative)
4. Context savings (estimated tokens saved)

---

## References

- [Orchestrator Mode Architecture](src/python/htmlgraph/hooks/orchestrator.py)
- [Event Tracking System](src/python/htmlgraph/hooks/event_tracker.py)
- [Database Schema](src/python/htmlgraph/db/schema.py)
- [Pattern Detection Research](/Users/shakes/DevProjects/htmlgraph/.htmlgraph/spikes/)

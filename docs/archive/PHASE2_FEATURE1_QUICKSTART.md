# Phase 2 Feature 1: Smart Delegation Suggestions - Quick Start Guide

**For Developers Implementing This Feature**

---

## TL;DR

You're building a system that proactively suggests Task() delegation when users exhibit delegation-worthy patterns (exploration, implementation, debugging, refactoring).

**Key Components:**
1. **PatternDetector** - Analyzes tool history to identify patterns
2. **SuggestionEngine** - Generates contextual Task() calls
3. **PreferenceManager** - Stores/retrieves user preferences
4. **ResponseFormatter** - Displays suggestions with interactive prompts

**Timeline:** 2 weeks (6 phases)

---

## Getting Started

### 1. Read Documentation (30 minutes)

**Must Read:**
- [PHASE2_FEATURE1_SUMMARY.md](PHASE2_FEATURE1_SUMMARY.md) - High-level overview
- [PHASE2_FEATURE1_IMPLEMENTATION_SPEC.md](PHASE2_FEATURE1_IMPLEMENTATION_SPEC.md) - Detailed spec
- [PHASE2_FEATURE1_ARCHITECTURE.md](PHASE2_FEATURE1_ARCHITECTURE.md) - Visual diagrams

**Optional:**
- [src/python/htmlgraph/hooks/orchestrator.py](src/python/htmlgraph/hooks/orchestrator.py) - Current orchestrator
- [src/python/htmlgraph/hooks/event_tracker.py](src/python/htmlgraph/hooks/event_tracker.py) - Event tracking

### 2. Set Up Environment (15 minutes)

```bash
# Clone and set up project
cd /path/to/htmlgraph
git checkout -b feature/smart-delegation-suggestions

# Install dependencies
uv sync

# Run existing tests to ensure baseline
uv run pytest tests/python/test_orchestrator_enforce_hook.py
uv run pytest tests/python/test_orchestrator_mode.py

# All should pass before you start
```

### 3. Create Initial File Structure (10 minutes)

```bash
# Create new orchestration module
mkdir -p src/python/htmlgraph/orchestration
touch src/python/htmlgraph/orchestration/__init__.py
touch src/python/htmlgraph/orchestration/pattern_detector.py
touch src/python/htmlgraph/orchestration/suggestion_engine.py
touch src/python/htmlgraph/orchestration/preference_manager.py
touch src/python/htmlgraph/orchestration/formatters.py

# Create test files
mkdir -p tests/python
touch tests/python/test_pattern_detector.py
touch tests/python/test_suggestion_engine.py
touch tests/python/test_preference_manager.py
touch tests/python/test_formatters.py
touch tests/python/test_suggestion_integration.py
```

---

## Phase 1: Pattern Detection (Days 1-3)

### Quick Implementation Path

**1. Start with Pattern Dataclass (30 minutes)**

```python
# src/python/htmlgraph/orchestration/pattern_detector.py

from dataclasses import dataclass
from typing import Literal

@dataclass
class Pattern:
    """Detected pattern from tool history."""
    pattern_type: Literal["exploration", "implementation", "debugging", "refactoring"]
    confidence: float  # 0.0-1.0
    tool_sequence: list[str]
    file_paths: list[str]
    description: str
```

**2. Add PatternDetector Class Skeleton (1 hour)**

```python
from htmlgraph.db.schema import HtmlGraphDB

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
        """Main entry point - detect pattern from tool history."""
        history = self._load_tool_history(session_id, lookback)
        history.append({"tool": current_tool, "timestamp": datetime.now()})

        # Check patterns (most specific first)
        for detector in [
            self._detect_exploration,
            self._detect_implementation,
            self._detect_debugging,
            self._detect_refactoring,
        ]:
            pattern = detector(history)
            if pattern and pattern.confidence >= 0.5:
                return pattern

        return None

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
            for row in reversed(rows)
        ]
```

**3. Implement Exploration Pattern First (2 hours)**

```python
def _detect_exploration(self, history: list[dict]) -> Pattern | None:
    """Detect exploration pattern (Read/Grep/Glob sequences)."""
    exploration_tools = ["Read", "Grep", "Glob"]
    recent = history[-7:]  # Last 7 tool calls

    exploration_count = sum(
        1 for h in recent if h["tool"] in exploration_tools
    )

    # Determine confidence based on count
    if exploration_count >= 5:
        confidence = 0.9
    elif exploration_count >= 3:
        confidence = 0.7
    elif exploration_count >= 2:
        confidence = 0.5
    else:
        return None  # Not enough for pattern

    file_paths = self._extract_file_paths(recent)

    return Pattern(
        pattern_type="exploration",
        confidence=confidence,
        tool_sequence=[h["tool"] for h in recent if h["tool"] in exploration_tools],
        file_paths=file_paths,
        description=f"Exploring codebase ({exploration_count} lookups in {len(recent)} calls)"
    )

def _extract_file_paths(self, history: list[dict]) -> list[str]:
    """Extract file paths from tool history."""
    paths = []
    for h in history:
        context = h.get("context", {})
        if "file_paths" in context:
            paths.extend(context["file_paths"])
    return paths
```

**4. Write Tests Immediately (1 hour)**

```python
# tests/python/test_pattern_detector.py

import pytest
from htmlgraph.orchestration.pattern_detector import PatternDetector, Pattern

def test_detect_exploration_high_confidence():
    """Test exploration pattern with high confidence."""
    detector = PatternDetector(":memory:")

    history = [
        {"tool": "Read", "timestamp": "...", "context": {"file_paths": ["a.py"]}},
        {"tool": "Read", "timestamp": "...", "context": {"file_paths": ["b.py"]}},
        {"tool": "Bash", "timestamp": "...", "context": {}},
        {"tool": "Read", "timestamp": "...", "context": {"file_paths": ["c.py"]}},
        {"tool": "Grep", "timestamp": "...", "context": {}},
        {"tool": "Read", "timestamp": "...", "context": {"file_paths": ["d.py"]}},
        {"tool": "Read", "timestamp": "...", "context": {"file_paths": ["e.py"]}},
    ]

    pattern = detector._detect_exploration(history)

    assert pattern is not None
    assert pattern.pattern_type == "exploration"
    assert pattern.confidence >= 0.9
    assert len(pattern.tool_sequence) >= 5

# Run tests
# uv run pytest tests/python/test_pattern_detector.py::test_detect_exploration_high_confidence -v
```

**5. Implement Other Patterns (Days 2-3)**

Follow same pattern:
1. Implement detector method
2. Write tests immediately
3. Verify with `pytest`

### Testing Strategy

```bash
# Test single pattern
uv run pytest tests/python/test_pattern_detector.py::test_detect_exploration_high_confidence -v

# Test all patterns
uv run pytest tests/python/test_pattern_detector.py -v

# Check coverage
uv run pytest tests/python/test_pattern_detector.py --cov=src/python/htmlgraph/orchestration/pattern_detector --cov-report=term-missing
```

### Common Issues & Solutions

**Issue: Database not found**
```python
# Solution: Use :memory: for tests
detector = PatternDetector(":memory:")
```

**Issue: Pattern not detected**
```python
# Debug: Print history
print("History:", history)
print("Exploration count:", exploration_count)
print("Confidence:", confidence)
```

**Issue: File paths empty**
```python
# Ensure context has file_paths
context = {"file_paths": ["test.py"]}
```

---

## Phase 2: Suggestion Engine (Days 4-6)

### Quick Implementation Path

**1. Create Suggestion Dataclass (30 minutes)**

```python
# src/python/htmlgraph/orchestration/suggestion_engine.py

from dataclasses import dataclass

@dataclass
class Suggestion:
    """A delegation suggestion for the user."""
    task_code: str
    explanation: str
    subagent_type: str
    estimated_savings: str
```

**2. Implement SuggestionEngine (2 hours)**

```python
class SuggestionEngine:
    """Generates Task() delegation suggestions from patterns."""

    def generate_suggestion(
        self,
        pattern: Pattern,
        tool_history: list[dict]
    ) -> Suggestion:
        """Generate suggestion from pattern."""
        if pattern.pattern_type == "exploration":
            return self._suggest_exploration_delegation(pattern, tool_history)
        elif pattern.pattern_type == "implementation":
            return self._suggest_implementation_delegation(pattern, tool_history)
        # ... etc
```

**3. Start with Exploration Suggestion (2 hours)**

```python
def _suggest_exploration_delegation(
    self,
    pattern: Pattern,
    tool_history: list[dict]
) -> Suggestion:
    """Generate exploration delegation suggestion."""

    files_explored = pattern.file_paths[:5]
    exploration_goal = self._infer_exploration_goal(tool_history)

    task_code = f'''Task(
    prompt="""
    Explore and document {exploration_goal}.

    Files involved:
    {self._format_file_list(files_explored)}

    Provide:
    1. Overview of functionality
    2. Key components and relationships
    3. Issues or concerns

    🔴 CRITICAL - Report Results:
    from htmlgraph import SDK
    sdk = SDK(agent='explorer')
    sdk.spikes.create('Exploration Results') \\
        .set_findings('Summary...') \\
        .save()
    """,
    subagent_type="general-purpose"
)'''

    explanation = (
        f"You've explored {len(pattern.tool_sequence)} files. "
        f"Delegation saves ~{self._estimate_token_savings(pattern)} tokens."
    )

    return Suggestion(
        task_code=task_code,
        explanation=explanation,
        subagent_type="general-purpose",
        estimated_savings=self._estimate_token_savings(pattern)
    )
```

**4. Test Task() Code is Valid (1 hour)**

```python
# tests/python/test_suggestion_engine.py

def test_task_code_is_syntactically_valid():
    """Ensure generated Task() code compiles."""
    engine = SuggestionEngine()

    pattern = Pattern(
        pattern_type="exploration",
        confidence=0.9,
        tool_sequence=["Read", "Read"],
        file_paths=["test.py"],
        description="Exploring"
    )

    suggestion = engine.generate_suggestion(pattern, [])

    # Should compile without syntax errors
    try:
        compile(suggestion.task_code, "<string>", "exec")
    except SyntaxError as e:
        pytest.fail(f"Generated code has syntax errors: {e}")
```

### Key Tips

**1. Keep Prompts Contextual**
- Include file paths
- Infer goals from history
- Add clear deliverables

**2. Always Add Reporting Pattern**
```python
🔴 CRITICAL - Report Results:
from htmlgraph import SDK
sdk = SDK(agent='<agent>')
sdk.spikes.create('<title>') \
    .set_findings('...') \
    .save()
```

**3. Test Generated Code**
- Compile check (syntax)
- Manual inspection (readability)
- Copy-paste test (does it work?)

---

## Phase 3: Preference Management (Days 7-8)

### Quick Implementation Path

**1. Add Database Tables (1 hour)**

```python
# src/python/htmlgraph/db/schema.py

# Add to create_tables() method

cursor.execute("""
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
    )
""")

cursor.execute("""
    CREATE INDEX IF NOT EXISTS idx_delegation_preferences_session
    ON delegation_preferences(session_id, pattern_type)
""")
```

**2. Implement PreferenceManager (2 hours)**

```python
# src/python/htmlgraph/orchestration/preference_manager.py

from enum import Enum

class PreferenceAction(Enum):
    ACCEPTED = "accepted"
    REJECTED = "rejected"
    ALWAYS = "always"
    NEVER = "never"

class PreferenceManager:
    """Manages user delegation preferences."""

    def __init__(self, db_path: str):
        self.db = HtmlGraphDB(db_path)
        self._ensure_preferences_table()

    def should_suggest(self, session_id: str, pattern_type: str) -> bool:
        """Check if we should show suggestion."""
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

        return cursor.fetchone() is None  # True if no "never" found

    def should_auto_delegate(self, session_id: str, pattern_type: str) -> bool:
        """Check if we should auto-delegate."""
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
        """Record user's response."""
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
```

**3. Test Preferences (1 hour)**

```python
# tests/python/test_preference_manager.py

def test_never_preference_blocks_suggestions():
    """Test 'never' preference blocks suggestions."""
    manager = PreferenceManager(":memory:")

    manager.record_action(
        session_id="sess-1",
        pattern_type="debugging",
        action=PreferenceAction.NEVER,
        confidence=0.7
    )

    assert manager.should_suggest("sess-1", "debugging") is False
```

---

## Phase 4: Hook Integration (Days 9-10)

### Quick Implementation Path

**1. Modify Orchestrator Hook (3 hours)**

```python
# src/python/htmlgraph/hooks/orchestrator.py

from htmlgraph.orchestration.pattern_detector import PatternDetector
from htmlgraph.orchestration.suggestion_engine import SuggestionEngine
from htmlgraph.orchestration.preference_manager import PreferenceManager

def enforce_orchestrator_mode(
    tool: str, params: dict[str, Any], session_id: str = "unknown"
) -> dict[str, Any]:
    """Enforce orchestrator mode with smart suggestions."""

    # ... existing code ...

    # NEW: Check for delegation patterns BEFORE violations
    try:
        detector = PatternDetector(str(get_database_path()))
        pattern = detector.detect_pattern(session_id, tool)

        if pattern:
            # Check preferences
            prefs = PreferenceManager(str(get_database_path()))

            # Never suggest this pattern?
            if not prefs.should_suggest(session_id, pattern.pattern_type):
                # User said "never", continue without suggestion
                pass
            # Always delegate this pattern?
            elif prefs.should_auto_delegate(session_id, pattern.pattern_type):
                # Auto-delegate!
                engine = SuggestionEngine()
                suggestion = engine.generate_suggestion(pattern, [])

                # Execute Task() automatically
                # ... (Task execution logic)

                return {
                    "hookSpecificOutput": {
                        "hookEventName": "PreToolUse",
                        "permissionDecision": "allow",
                        "additionalContext": f"Auto-delegated {pattern.pattern_type} pattern"
                    }
                }
            else:
                # Show suggestion
                engine = SuggestionEngine()
                suggestion = engine.generate_suggestion(pattern, [])

                # Format for display
                message = f"""⚠️ ORCHESTRATOR: {pattern.description}

SUGGESTED DELEGATION:
{suggestion.task_code}

{suggestion.explanation}

[Y]es, run  [N]o, continue  [A]lways delegate {pattern.pattern_type}  [?]Learn more
"""

                return {
                    "hookSpecificOutput": {
                        "hookEventName": "PreToolUse",
                        "permissionDecision": "allow",  # Allow but show suggestion
                        "additionalContext": message,
                    }
                }
    except Exception as e:
        # Graceful degradation - don't break orchestrator mode
        print(f"Warning: Suggestion engine error: {e}", file=sys.stderr)

    # Continue with existing orchestrator logic
    is_allowed, reason, category = is_allowed_orchestrator_operation(
        tool, params, session_id
    )

    # ... rest of existing code ...
```

**2. Test Integration (2 hours)**

```python
# tests/python/test_suggestion_integration.py

def test_end_to_end_suggestion_flow():
    """Test complete flow from pattern to suggestion."""
    db = HtmlGraphDB(":memory:")
    session_id = "sess-test"

    # Simulate tool history
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

    # Trigger pattern detection
    detector = PatternDetector(str(db.db_path))
    pattern = detector.detect_pattern(session_id, "Read")

    assert pattern is not None
    assert pattern.pattern_type == "exploration"

    # Generate suggestion
    engine = SuggestionEngine()
    suggestion = engine.generate_suggestion(pattern, [])

    assert "Task(" in suggestion.task_code
    assert "general-purpose" in suggestion.task_code

    # Record acceptance
    prefs = PreferenceManager(str(db.db_path))
    prefs.record_action(
        session_id=session_id,
        pattern_type=pattern.pattern_type,
        action=PreferenceAction.ACCEPTED,
        confidence=pattern.confidence
    )

    # Verify recorded
    rate = prefs.get_acceptance_rate(session_id)
    assert rate == 1.0
```

---

## Testing Checklist

### Quick Test Commands

```bash
# Test individual component
uv run pytest tests/python/test_pattern_detector.py -v

# Test with coverage
uv run pytest tests/python/test_pattern_detector.py --cov=src/python/htmlgraph/orchestration/pattern_detector --cov-report=term-missing

# Test all new modules
uv run pytest tests/python/test_pattern_detector.py tests/python/test_suggestion_engine.py tests/python/test_preference_manager.py -v

# Test integration
uv run pytest tests/python/test_suggestion_integration.py -v

# Full test suite (verify no regressions)
uv run pytest

# Code quality
uv run ruff check --fix
uv run ruff format
uv run mypy src/python/htmlgraph/orchestration/
```

---

## Debugging Tips

### Pattern Not Detected?

```python
# Add debug prints
def detect_pattern(self, session_id: str, current_tool: str):
    history = self._load_tool_history(session_id, 10)
    print(f"DEBUG: History: {history}")  # See what's loaded

    pattern = self._detect_exploration(history)
    print(f"DEBUG: Pattern: {pattern}")  # See what's detected

    return pattern
```

### Preferences Not Working?

```python
# Check database
sqlite3 .htmlgraph/htmlgraph.db

SELECT * FROM delegation_preferences;

# Should see your preference records
```

### Task() Code Invalid?

```python
# Test compile
suggestion = engine.generate_suggestion(pattern, [])
try:
    compile(suggestion.task_code, "<string>", "exec")
    print("✓ Valid Python")
except SyntaxError as e:
    print(f"✗ Syntax error: {e}")
```

---

## Common Pitfalls

### 1. Forgetting Database Connection

```python
# BAD
detector = PatternDetector(None)  # Will crash

# GOOD
from htmlgraph.config import get_database_path
detector = PatternDetector(str(get_database_path()))
```

### 2. Not Checking Pattern is None

```python
# BAD
pattern = detector.detect_pattern(session_id, tool)
suggestion = engine.generate_suggestion(pattern, [])  # Crash if None!

# GOOD
pattern = detector.detect_pattern(session_id, tool)
if pattern:
    suggestion = engine.generate_suggestion(pattern, [])
```

### 3. Forgetting to Commit Database Changes

```python
# BAD
cursor.execute("INSERT INTO ...")
# Not committed!

# GOOD
cursor.execute("INSERT INTO ...")
self.db.connection.commit()  # Actually save
```

### 4. Using Relative Paths

```python
# BAD
db = HtmlGraphDB(".htmlgraph/htmlgraph.db")  # Relative!

# GOOD
from htmlgraph.config import get_database_path
db = HtmlGraphDB(str(get_database_path()))  # Absolute
```

---

## Quick Reference: Key Functions

### PatternDetector
```python
detector = PatternDetector(db_path)
pattern = detector.detect_pattern(session_id, current_tool, lookback=10)
# Returns: Pattern | None
```

### SuggestionEngine
```python
engine = SuggestionEngine()
suggestion = engine.generate_suggestion(pattern, tool_history)
# Returns: Suggestion
```

### PreferenceManager
```python
prefs = PreferenceManager(db_path)

# Check if should show suggestion
should_show = prefs.should_suggest(session_id, pattern_type)

# Check if should auto-delegate
auto_delegate = prefs.should_auto_delegate(session_id, pattern_type)

# Record user action
prefs.record_action(session_id, pattern_type, PreferenceAction.ACCEPTED, confidence)
```

---

## Final Checklist Before Deployment

- [ ] All tests pass: `uv run pytest`
- [ ] Coverage >90%: `uv run pytest --cov=...`
- [ ] No ruff errors: `uv run ruff check`
- [ ] No mypy errors: `uv run mypy src/`
- [ ] Manual testing in dev mode: `uv run htmlgraph claude --dev`
- [ ] Database migrations work
- [ ] Documentation updated
- [ ] Commit and push

---

## Getting Help

**Documentation:**
- [PHASE2_FEATURE1_IMPLEMENTATION_SPEC.md](PHASE2_FEATURE1_IMPLEMENTATION_SPEC.md) - Full spec
- [PHASE2_FEATURE1_ARCHITECTURE.md](PHASE2_FEATURE1_ARCHITECTURE.md) - Diagrams

**Code References:**
- `src/python/htmlgraph/hooks/orchestrator.py` - Existing orchestrator
- `src/python/htmlgraph/hooks/event_tracker.py` - Event tracking patterns
- `src/python/htmlgraph/db/schema.py` - Database schema

**Testing:**
- `tests/python/test_orchestrator_enforce_hook.py` - Orchestrator tests
- Run tests with `-v` for verbose output
- Use `--pdb` to drop into debugger on failure

---

**Last Updated:** 2026-01-13
**Estimated Time:** 2 weeks (phased implementation)

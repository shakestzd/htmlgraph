# Phase 1: Pydantic Integration Specification

**Status:** Ready for Codex Implementation
**Feature ID:** feat-1598baf6
**Scope:** CLI command input validation via Pydantic models
**Timeline:** 2-3 sprints (foundation + high-priority commands)

---

## Executive Summary

This specification details the strategy for integrating Pydantic v2 into Wipnote's CLI layer to replace fragmented argparse validation with centralized, type-safe models. Pydantic is already a project dependency (v2.0.0+), and Rich is available for beautiful error formatting.

**Key Goals:**
- Replace ad-hoc validation logic with declarative Pydantic models
- Improve error messages with field-level hints and constraints
- Maintain 100% backward compatibility with existing CLI
- Create reusable models for future SDK/REST API use
- Provide foundation for config management and nested structures

---

## 1. Command Priority Assessment

### Tier 1: High Priority (Most Validation Logic)
Commands with complex arguments, frequent usage, and existing validation logic.

#### 1.1 Feature Commands
**Commands:**
- `feature create` - Create new feature with title, description, priority, steps
- `feature start` - Claim and start working on feature
- `feature complete` - Mark feature as done
- `feature delete` - Remove feature with confirmation

**Current Validation:**
- Title: Required, no length constraints currently enforced
- Priority: Choices enforced via argparse `choices=["low", "medium", "high", "critical"]`
- Description: Optional, optional string sanitization needed
- Steps: Variable-length list, optional
- IDs: Checked at runtime against filesystem

**Validation Gaps:**
- No length limits (title 1-200 chars recommended)
- No slug/ID format validation (alphanumeric, hyphens, underscores)
- Steps need individual length validation
- Collection name validation missing

**Complexity:** 4/5 | **Priority:** Critical
**Estimated Conversion Time:** 1-2 hours

---

#### 1.2 Session Commands
**Commands:**
- `session start` - Create new session with optional title, ID
- `session end` - End session with notes, recommendations, blockers
- `session handoff` - Set/show handoff context
- `session link` - Link session to feature

**Current Validation:**
- Session ID: Optional, auto-generated if not provided
- Title: Optional string
- Notes: Optional string
- Blockers: Optional, stored as array
- Format: Choices enforced (`["text", "json"]`)

**Validation Gaps:**
- Notes length should be capped (2000 chars?)
- Blockers should be non-empty items
- Session ID format validation when provided (must match existing pattern)
- Recommended next: No validation for length/format

**Complexity:** 3/5 | **Priority:** Critical
**Estimated Conversion Time:** 1-2 hours

---

#### 1.3 Track Commands
**Commands:**
- `track new` - Create track with title, priority, description
- `track list` - List all tracks
- `track spec` - Create/update track spec
- `track plan` - Create/update track plan
- `track delete` - Remove track

**Current Validation:**
- Title: Required, no length constraints
- Priority: Choices (low, medium, high)
- Description: Optional
- Track ID format: Checked at runtime

**Validation Gaps:**
- Title length constraints missing (1-300 chars?)
- Priority enum validation needed
- Description length (5000 chars max?)
- Track ID format validation

**Complexity:** 3/5 | **Priority:** High
**Estimated Conversion Time:** 1 hour

---

### Tier 2: Medium Priority (Standard Validation)
Commands with simpler validation but frequent usage.

#### 2.1 Work Management
**Commands:**
- `work next` - Get next task
- `work queue` - Get prioritized queue

**Current Validation:**
- Agent: Optional, defaults to env var or "claude"
- Min score: Float, type validation via argparse
- Limit: Integer, positive constraint missing

**Validation Gaps:**
- Min score should be 0-100 range
- Limit should be positive integer (1-100)
- Agent name format validation

**Complexity:** 2/5 | **Priority:** Medium
**Estimated Conversion Time:** 30 minutes

---

#### 2.2 Query & Status
**Commands:**
- `query` - Query nodes with CSS selector
- `status` - Show graph status

**Current Validation:**
- Selector: Required CSS selector
- Graph dir: Path validation

**Validation Gaps:**
- CSS selector validation (regex check)
- Path exists validation

**Complexity:** 2/5 | **Priority:** Medium
**Estimated Conversion Time:** 30 minutes

---

### Tier 3: Lower Priority (Setup/Admin)
Commands used less frequently but important for setup.

#### 3.1 Init & Setup
**Commands:**
- `init` - Initialize .wipnote directory
- `install-hooks` - Install git hooks

**Current Validation:**
- Directory path: Checked at runtime
- Hook selection: Validated against AVAILABLE_HOOKS

**Validation Gaps:**
- Path writability check
- Hook names validation

**Complexity:** 2/5 | **Priority:** Low
**Estimated Conversion Time:** 1 hour

---

#### 3.2 Admin Commands
**Commands:**
- `orchestrator enable/disable/set-level`
- `deploy init/run`
- `sync-docs`

**Current Validation:**
- Level: Should validate numeric range
- Format: Choices validation

**Complexity:** 2/5 | **Priority:** Low
**Estimated Conversion Time:** 1 hour

---

### Priority Ranking for Phase 1

```
Phase 1 (MVP - Foundation + Top 5):
1. feature create        [TIER 1 - Critical]
2. feature start/complete/delete [TIER 1 - Critical]
3. session start/end     [TIER 1 - Critical]
4. track new/list/spec   [TIER 1 - High]
5. work next/queue       [TIER 2 - Medium]

Est. Total: 6-8 hours of implementation
```

---

## 2. Pydantic Models Design

### 2.1 Base Models & Mixins

```python
# src/python/wipnote/pydantic_models.py

from pydantic import BaseModel, Field, field_validator, ConfigDict
from typing import Optional, Literal
from enum import Enum

class CommonModelConfig(ConfigDict):
    """Shared configuration for all models."""
    str_strip_whitespace = True  # Auto-trim strings
    validate_default = True      # Validate default values
    use_enum_values = False      # Keep enum objects

class GraphNode(BaseModel):
    """Base for all graph node inputs."""
    model_config = CommonModelConfig

    graph_dir: str = Field(
        default=".wipnote",
        description="Graph directory path"
    )
    agent: str = Field(
        default="cli",
        description="Agent name for attribution"
    )
    format: Literal["text", "json"] = Field(
        default="text",
        description="Output format"
    )

class PriorityEnum(str, Enum):
    """Priority levels."""
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"
```

---

### 2.2 Feature Models

```python
class FeatureCreateInput(GraphNode):
    """Input model for 'feature create' command."""

    title: str = Field(
        ...,  # Required
        min_length=1,
        max_length=200,
        description="Feature title"
    )
    description: str = Field(
        default="",
        max_length=5000,
        description="Detailed description"
    )
    priority: PriorityEnum = Field(
        default=PriorityEnum.MEDIUM,
        description="Priority level"
    )
    collection: str = Field(
        default="features",
        regex="^[a-z_]+$",
        description="Collection name (features, bugs, etc.)"
    )
    steps: Optional[list[str]] = Field(
        default=None,
        description="Implementation steps"
    )

    @field_validator('steps')
    @classmethod
    def validate_steps(cls, v):
        """Validate step list."""
        if v is None:
            return []
        if len(v) > 50:
            raise ValueError("Maximum 50 steps allowed")
        for i, step in enumerate(v):
            if not step or len(step) > 500:
                raise ValueError(
                    f"Step {i+1}: must be 1-500 characters"
                )
        return v

    @field_validator('title')
    @classmethod
    def validate_title(cls, v):
        """Validate title format."""
        if not v.strip():
            raise ValueError("Title cannot be empty/whitespace")
        return v


class FeatureStartInput(GraphNode):
    """Input model for 'feature start' command."""

    id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]+$",
        description="Feature ID to start"
    )
    collection: str = Field(
        default="features",
        regex="^[a-z_]+$",
        description="Collection name"
    )


class FeatureCompleteInput(GraphNode):
    """Input model for 'feature complete' command."""

    id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]+$",
        description="Feature ID to complete"
    )
    collection: str = Field(
        default="features",
        regex="^[a-z_]+$",
        description="Collection name"
    )


class FeatureDeleteInput(GraphNode):
    """Input model for 'feature delete' command."""

    id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]+$",
        description="Feature ID to delete"
    )
    collection: str = Field(
        default="features",
        regex="^[a-z_]+$",
        description="Collection name"
    )
    yes: bool = Field(
        default=False,
        description="Skip confirmation"
    )
```

---

### 2.3 Session Models

```python
class SessionStartInput(GraphNode):
    """Input model for 'session start' command."""

    id: Optional[str] = Field(
        default=None,
        regex="^[a-zA-Z0-9_-]{1,50}$",
        description="Session ID (auto-generated if not provided)"
    )
    title: Optional[str] = Field(
        default=None,
        max_length=200,
        description="Session title"
    )
    agent: str = Field(
        default="claude-code",
        description="Agent name"
    )


class SessionEndInput(GraphNode):
    """Input model for 'session end' command."""

    id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]{1,50}$",
        description="Session ID to end"
    )
    notes: Optional[str] = Field(
        default=None,
        max_length=2000,
        description="Handoff notes"
    )
    recommend: Optional[str] = Field(
        default=None,
        max_length=1000,
        description="Recommended next steps"
    )
    blocker: Optional[list[str]] = Field(
        default=None,
        description="Blocking issues"
    )

    @field_validator('blocker')
    @classmethod
    def validate_blocker(cls, v):
        """Validate blocker list."""
        if v is None:
            return []
        if len(v) > 10:
            raise ValueError("Maximum 10 blockers allowed")
        for blocker in v:
            if not blocker or len(blocker) > 200:
                raise ValueError("Each blocker must be 1-200 characters")
        return v

    @field_validator('notes', 'recommend')
    @classmethod
    def validate_text(cls, v):
        """Validate text fields are not whitespace-only."""
        if v and not v.strip():
            raise ValueError("Cannot be whitespace only")
        return v


class SessionHandoffInput(GraphNode):
    """Input model for 'session handoff' command."""

    session_id: Optional[str] = Field(
        default=None,
        regex="^[a-zA-Z0-9_-]{1,50}$",
        description="Session ID (auto-found if not provided)"
    )
    notes: Optional[str] = Field(
        default=None,
        max_length=2000,
        description="Handoff notes"
    )
    recommend: Optional[str] = Field(
        default=None,
        max_length=1000,
        description="Recommended next steps"
    )
    show: bool = Field(
        default=False,
        description="Show handoff context"
    )


class SessionLinkInput(GraphNode):
    """Input model for 'session link' command."""

    session_id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]{1,50}$",
        description="Session ID"
    )
    feature_id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]{1,50}$",
        description="Feature ID to link"
    )
    collection: str = Field(
        default="features",
        regex="^[a-z_]+$",
        description="Collection name"
    )
    bidirectional: bool = Field(
        default=False,
        description="Create bidirectional link"
    )
```

---

### 2.4 Track Models

```python
class TrackNewInput(GraphNode):
    """Input model for 'track new' command."""

    title: str = Field(
        ...,
        min_length=1,
        max_length=300,
        description="Track title"
    )
    priority: PriorityEnum = Field(
        default=PriorityEnum.MEDIUM,
        description="Priority level"
    )
    description: str = Field(
        default="",
        max_length=5000,
        description="Track description"
    )

    @field_validator('title')
    @classmethod
    def validate_title(cls, v):
        """Validate title."""
        if not v.strip():
            raise ValueError("Title cannot be empty/whitespace")
        return v


class TrackListInput(GraphNode):
    """Input model for 'track list' command."""
    pass


class TrackSpecInput(GraphNode):
    """Input model for 'track spec' command."""

    track_id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]+$",
        description="Track ID"
    )
    title: str = Field(
        ...,
        min_length=1,
        max_length=300,
        description="Spec title"
    )


class TrackPlanInput(GraphNode):
    """Input model for 'track plan' command."""

    track_id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]+$",
        description="Track ID"
    )
    title: str = Field(
        ...,
        min_length=1,
        max_length=300,
        description="Plan title"
    )


class TrackDeleteInput(GraphNode):
    """Input model for 'track delete' command."""

    track_id: str = Field(
        ...,
        regex="^[a-zA-Z0-9_-]+$",
        description="Track ID to delete"
    )
    yes: bool = Field(
        default=False,
        description="Skip confirmation"
    )
```

---

### 2.5 Work Management Models

```python
class WorkNextInput(GraphNode):
    """Input model for 'work next' command."""

    agent: str = Field(
        default="claude",
        description="Agent ID"
    )
    auto_claim: bool = Field(
        default=False,
        description="Auto-claim the task"
    )
    min_score: float = Field(
        default=20.0,
        ge=0,
        le=100,
        description="Minimum routing score (0-100)"
    )

    @field_validator('min_score')
    @classmethod
    def validate_min_score(cls, v):
        """Ensure reasonable score range."""
        if v < 0 or v > 100:
            raise ValueError("Score must be between 0 and 100")
        return v


class WorkQueueInput(GraphNode):
    """Input model for 'work queue' command."""

    agent: str = Field(
        default="claude",
        description="Agent ID"
    )
    limit: int = Field(
        default=10,
        ge=1,
        le=100,
        description="Max tasks (1-100)"
    )
    min_score: float = Field(
        default=20.0,
        ge=0,
        le=100,
        description="Minimum routing score (0-100)"
    )
```

---

### 2.6 Query & Status Models

```python
class QueryInput(GraphNode):
    """Input model for 'query' command."""

    selector: str = Field(
        ...,
        description="CSS selector for querying"
    )

    @field_validator('selector')
    @classmethod
    def validate_selector(cls, v):
        """Basic CSS selector validation."""
        if not v.strip():
            raise ValueError("Selector cannot be empty")
        # Reject obviously invalid selectors
        invalid_chars = ['<', '>', '{', '}', ';']
        if any(c in v for c in invalid_chars):
            raise ValueError("Invalid CSS selector syntax")
        return v


class StatusInput(GraphNode):
    """Input model for 'status' command."""
    pass
```

---

## 3. Integration Strategy

### 3.1 Argparse → Pydantic Flow

**Current Pattern:**
```python
# cli.py
args = parser.parse_args()  # argparse.Namespace
cmd_feature_create(args)    # Direct function call
```

**New Pattern:**
```python
# cli.py
args = parser.parse_args()  # argparse.Namespace
args_dict = vars(args)      # Convert to dict

# Validate with Pydantic
try:
    input_model = FeatureCreateInput(**args_dict)
except ValidationError as e:
    display_validation_error(e)
    sys.exit(1)

# Call command with validated model
cmd_feature_create(input_model)
```

---

### 3.2 Integration in cmd_feature_create (Example)

**Before:**
```python
def cmd_feature_create(args: argparse.Namespace) -> None:
    """Create a new feature."""
    from wipnote.cli_commands.feature import FeatureCreateCommand

    command = FeatureCreateCommand(
        collection=args.collection,
        title=args.title,
        description=args.description or "",
        priority=args.priority,
        steps=args.steps,
    )
    command.run(graph_dir=args.graph_dir, agent=args.agent, output_format=args.format)
```

**After:**
```python
def cmd_feature_create(input_model: FeatureCreateInput) -> None:
    """Create a new feature."""
    from wipnote.cli_commands.feature import FeatureCreateCommand

    command = FeatureCreateCommand(
        collection=input_model.collection,
        title=input_model.title,
        description=input_model.description,
        priority=input_model.priority,
        steps=input_model.steps,
    )
    command.run(
        graph_dir=input_model.graph_dir,
        agent=input_model.agent,
        output_format=input_model.format
    )
```

---

### 3.3 Wrapper Function Pattern

Create a wrapper to convert argparse → Pydantic for each command group:

```python
# cli.py

def validate_feature_create(args: argparse.Namespace) -> FeatureCreateInput:
    """Convert argparse to Pydantic model."""
    try:
        return FeatureCreateInput(**vars(args))
    except ValidationError as e:
        display_validation_error(e)
        sys.exit(1)

# In main argparse setup:
feature_create.set_defaults(handler=lambda args: cmd_feature_create(
    validate_feature_create(args)
))
```

---

## 4. Error Handling Strategy

### 4.1 Rich + Pydantic ValidationError Integration

```python
# cli.py or cli_utils.py

from pydantic import ValidationError
from rich.console import Console
from rich.panel import Panel
from rich.syntax import Syntax

console = Console()

def display_validation_error(error: ValidationError) -> None:
    """Format Pydantic validation errors beautifully with Rich."""

    console.print("\n[bold red]Validation Error[/bold red]")

    for err in error.errors():
        field = ".".join(str(x) for x in err["loc"])
        msg = err["msg"]
        error_type = err["type"]

        # Field-specific error formatting
        color = "red" if error_type == "value_error" else "yellow"
        console.print(f"\n[{color}]✗ {field}[/{color}]")
        console.print(f"  {msg}")

        # Add context for common errors
        if error_type == "string_too_long":
            max_len = err.get("ctx", {}).get("max_length", "?")
            console.print(f"  [dim](Maximum {max_len} characters allowed)[/dim]")
        elif error_type == "string_too_short":
            min_len = err.get("ctx", {}).get("min_length", "?")
            console.print(f"  [dim](Minimum {min_len} characters required)[/dim]")
        elif error_type == "enum":
            # Show valid choices for enum fields
            allowed = err.get("ctx", {}).get("expected", "")
            console.print(f"  [dim]Valid options: {allowed}[/dim]")

    console.print()


def display_validation_summary(model_class: type[BaseModel]) -> str:
    """Show model schema as help text."""
    from rich.table import Table

    table = Table(title="Expected Arguments", show_header=True)
    table.add_column("Field", style="cyan")
    table.add_column("Type", style="green")
    table.add_column("Required", style="yellow")
    table.add_column("Default", style="blue")

    for field_name, field_info in model_class.model_fields.items():
        table.add_row(
            field_name,
            str(field_info.annotation),
            "✓" if field_info.is_required() else "✗",
            str(field_info.default) if field_info.default else "—"
        )

    return table
```

---

### 4.2 User-Friendly Error Messages

**Example Error Output:**

```
Validation Error

✗ title
  String should have at least 1 character
  (Minimum 1 characters required)

✗ priority
  Input should be 'low', 'medium', 'high' or 'critical'
  Valid options: low, medium, high, critical

✗ steps[1]
  String should have at most 500 characters
  (Maximum 500 characters allowed)
```

---

### 4.3 Field-Level Error Hints

Enhance Field descriptions with examples:

```python
title: str = Field(
    ...,
    min_length=1,
    max_length=200,
    examples=["Add OAuth2 authentication", "Fix homepage layout bug"],
    description="Feature title (1-200 characters)"
)

priority: PriorityEnum = Field(
    default=PriorityEnum.MEDIUM,
    examples=["low", "medium", "high", "critical"],
    description="Priority level"
)
```

---

## 5. Configuration Management Extension

### 5.1 Pydantic Settings Integration (Phase 2)

Plan for global config via Pydantic Settings:

```python
# pydantic_models.py (Phase 2)

from pydantic_settings import BaseSettings

class WipnoteConfig(BaseSettings):
    """Global Wipnote configuration."""

    graph_dir: str = Field(
        default=".wipnote",
        env="HTMLGRAPH_DIR"
    )
    agent: str = Field(
        default="claude",
        env="HTMLGRAPH_AGENT"
    )
    port: int = Field(
        default=8000,
        ge=1024,
        le=65535,
        env="HTMLGRAPH_PORT"
    )

    model_config = SettingsConfigDict(
        env_file=".wipnote.env",
        env_file_encoding="utf-8",
        case_sensitive=False
    )
```

This allows:
- Config file loading (`.wipnote.env`)
- Environment variable overrides
- Type coercion
- Validation of global settings

---

## 6. Implementation Roadmap

### Step 1: Foundation (2-3 hours)

**Deliverable:** Core pydantic_models.py with base classes and first 2 command models

**Files to Create:**
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/pydantic_models.py`

**Tasks:**
1. Create base model classes (GraphNode, CommonModelConfig)
2. Create enum for priority (PriorityEnum)
3. Define FeatureCreateInput with all validators
4. Define FeatureStartInput, FeatureCompleteInput, FeatureDeleteInput
5. Add comprehensive docstrings and field descriptions

**Code Quality:**
- Mypy type checking: `uv run mypy src/python/wipnote/pydantic_models.py`
- Ruff linting: `uv run ruff check src/python/wipnote/pydantic_models.py`

**Validation:**
- Create basic unit tests in `tests/python/test_cli_models.py`
- Test each validator in isolation

---

### Step 2: Error Handling Utils (1 hour)

**Deliverable:** CLI utility functions for validation error display

**Files to Create/Modify:**
- Create: `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/cli_utils.py` (or extend existing)

**Tasks:**
1. Implement `display_validation_error()` function
2. Implement `display_validation_summary()` function
3. Add Rich formatting for error messages
4. Create helper to extract field-level constraints

**Code Quality:**
- Test error formatting with sample ValidationError instances
- Verify Rich output renders correctly

---

### Step 3: Feature Commands Integration (2-3 hours)

**Deliverable:** Feature commands using Pydantic validation

**Files to Modify:**
- `/Users/shakes/DevProjects/htmlgraph/src/python/wipnote/cli.py`
  - Add validation wrapper functions for feature commands
  - Modify cmd_feature_create, cmd_feature_start, cmd_feature_complete, cmd_feature_delete
  - Add handler setup in argparse

**Tasks:**
1. Add FeatureCreateInput, etc. imports
2. Create validate_feature_*() wrapper functions
3. Modify command functions to accept Pydantic models
4. Update argparse setup to use handler pattern
5. Test each command end-to-end
6. Verify error messages display correctly

**Testing:**
```bash
# Test successful creation
uv run wipnote feature create "Test Feature" --priority high

# Test validation errors
uv run wipnote feature create ""  # Empty title
uv run wipnote feature create "x" --priority invalid  # Invalid priority
```

**Code Quality:**
- All mypy checks pass
- All ruff checks pass

---

### Step 4: Session & Track Commands (2-3 hours)

**Deliverable:** Session and track commands using Pydantic validation

**Files to Modify:**
- Add SessionStartInput, SessionEndInput, SessionLinkInput to pydantic_models.py
- Add TrackNewInput, TrackListInput, TrackSpecInput, TrackPlanInput, TrackDeleteInput
- Modify cli.py to integrate all session and track command validators

**Tasks:**
1. Add session and track Pydantic models
2. Create validation wrappers for each command
3. Integrate into cli.py command handlers
4. Test all commands with valid/invalid inputs
5. Verify all error messages are helpful

**Testing:**
```bash
# Session tests
uv run wipnote session start --id test --title "Test"
uv run wipnote session end test --notes "Long notes..."

# Track tests
uv run wipnote track new "New Track" --priority high
uv run wipnote track delete track-id --yes
```

---

### Step 5: Work Management & Utilities (1-2 hours)

**Deliverable:** Work and utility commands with Pydantic validation

**Files to Modify:**
- Add WorkNextInput, WorkQueueInput, QueryInput, StatusInput
- Integrate into cli.py

**Tasks:**
1. Add remaining Pydantic models
2. Create validation wrappers
3. Integrate numeric range validation (score 0-100, limit 1-100)
4. Test all commands

---

### Step 6: Unit Tests (1-2 hours)

**Deliverable:** Comprehensive test suite for all models

**Files to Create:**
- `/Users/shakes/DevProjects/htmlgraph/tests/python/test_cli_models.py`

**Test Coverage:**
- Valid input acceptance for each model
- Invalid input rejection with appropriate errors
- Field-level validators (length, regex, enum)
- Default value application
- Edge cases (empty strings, max lengths, min/max values)

**Example Tests:**
```python
def test_feature_create_valid():
    model = FeatureCreateInput(
        title="Test Feature",
        priority="high"
    )
    assert model.title == "Test Feature"
    assert model.priority == PriorityEnum.HIGH

def test_feature_create_title_too_long():
    with pytest.raises(ValidationError) as exc:
        FeatureCreateInput(title="x" * 201)
    assert "String should have at most 200 characters" in str(exc.value)

def test_feature_create_title_empty():
    with pytest.raises(ValidationError):
        FeatureCreateInput(title="")

def test_feature_create_invalid_priority():
    with pytest.raises(ValidationError):
        FeatureCreateInput(
            title="Test",
            priority="super-critical"  # Invalid
        )
```

**Code Quality:**
- All tests pass: `uv run pytest tests/python/test_cli_models.py -v`
- Coverage: `uv run pytest tests/python/test_cli_models.py --cov=wipnote.pydantic_models`

---

### Step 7: Documentation & Integration (1 hour)

**Deliverable:** Developer documentation and integration guide

**Files to Create/Modify:**
- Update AGENTS.md with Pydantic model usage
- Add inline documentation to pydantic_models.py
- Create examples in docstrings

**Tasks:**
1. Document model creation pattern
2. Add examples of validation
3. Document error message customization
4. Update CLI help text where needed

---

### Timeline Summary

| Step | Task | Hours | Status |
|------|------|-------|--------|
| 1 | Foundation: pydantic_models.py | 2-3 | Ready |
| 2 | Error handling utilities | 1 | Ready |
| 3 | Feature commands | 2-3 | Ready |
| 4 | Session & track commands | 2-3 | Ready |
| 5 | Work & utility commands | 1-2 | Ready |
| 6 | Unit tests | 1-2 | Ready |
| 7 | Documentation | 1 | Ready |
| **Total** | | **11-16 hours** | **Ready for Codex** |

---

## 7. Code Examples

### 7.1 Before & After: Feature Create

**BEFORE (Fragmented Validation):**
```python
def cmd_feature_create(args: argparse.Namespace) -> None:
    """Create a new feature."""
    from wipnote.cli_commands.feature import FeatureCreateCommand

    # Validation scattered across multiple places
    # argparse validates priority choices
    # No validation on title length
    # Steps validated at runtime if at all

    command = FeatureCreateCommand(
        collection=args.collection,
        title=args.title,  # Could be empty string
        description=args.description or "",  # No length check
        priority=args.priority,  # Validated by argparse
        steps=args.steps,  # Could be anything
    )
    command.run(graph_dir=args.graph_dir, agent=args.agent, output_format=args.format)
```

**AFTER (Centralized Pydantic Validation):**
```python
def validate_feature_create(args: argparse.Namespace) -> FeatureCreateInput:
    """Convert argparse to Pydantic model."""
    try:
        return FeatureCreateInput(**vars(args))
    except ValidationError as e:
        display_validation_error(e)
        sys.exit(1)

def cmd_feature_create(input_model: FeatureCreateInput) -> None:
    """Create a new feature."""
    from wipnote.cli_commands.feature import FeatureCreateCommand

    # All validation already done by Pydantic
    # Type-safe access to all fields
    # Default values properly set

    command = FeatureCreateCommand(
        collection=input_model.collection,
        title=input_model.title,  # Guaranteed 1-200 chars
        description=input_model.description,  # Guaranteed ≤5000 chars
        priority=input_model.priority,  # Guaranteed valid enum
        steps=input_model.steps or [],  # Guaranteed valid list
    )
    command.run(
        graph_dir=input_model.graph_dir,
        agent=input_model.agent,
        output_format=input_model.format
    )

# In argparse setup:
feature_create.set_defaults(
    handler=lambda args: cmd_feature_create(validate_feature_create(args))
)
```

---

### 7.2 Adding Custom Validators

```python
from pydantic import field_validator

class FeatureCreateInput(GraphNode):
    """Input model for 'feature create' command."""

    title: str = Field(
        ...,
        min_length=1,
        max_length=200,
        description="Feature title"
    )

    # Simple field-level validator
    @field_validator('title')
    @classmethod
    def validate_title(cls, v):
        """Ensure title is not just whitespace."""
        if not v.strip():
            raise ValueError("Title cannot be empty or whitespace")
        return v

    # Multi-field validator
    @field_validator('steps')
    @classmethod
    def validate_steps(cls, v):
        """Validate step list."""
        if v is None:
            return []
        if len(v) > 50:
            raise ValueError("Maximum 50 steps allowed (got {len(v)})")
        for i, step in enumerate(v):
            if not step.strip() or len(step) > 500:
                raise ValueError(
                    f"Step {i+1} must be 1-500 characters"
                )
        return v
```

---

### 7.3 Displaying Validation Errors

```python
from wipnote.pydantic_models import FeatureCreateInput
from pydantic import ValidationError

def test_display_errors():
    """Example of error display."""
    try:
        FeatureCreateInput(
            title="",  # Invalid: empty
            priority="super-critical",  # Invalid: not in enum
            description="x" * 10000,  # Invalid: too long
        )
    except ValidationError as e:
        display_validation_error(e)

# Output:
# Validation Error
#
# ✗ title
#   String should have at least 1 character
#   (Minimum 1 characters required)
#
# ✗ priority
#   Input should be 'low', 'medium', 'high' or 'critical'
#   Valid options: low, medium, high, critical
#
# ✗ description
#   String should have at most 5000 characters
#   (Maximum 5000 characters allowed)
```

---

### 7.4 Testing Models

```python
import pytest
from pydantic import ValidationError
from wipnote.pydantic_models import FeatureCreateInput, PriorityEnum

def test_feature_create_valid():
    """Valid input should create model."""
    model = FeatureCreateInput(
        title="Add OAuth2 authentication",
        priority="high",
        description="Implement OAuth2 flow",
        steps=["Create OAuth provider", "Add login form", "Test integration"]
    )
    assert model.title == "Add OAuth2 authentication"
    assert model.priority == PriorityEnum.HIGH
    assert len(model.steps) == 3

def test_feature_create_title_required():
    """Title is required."""
    with pytest.raises(ValidationError) as exc:
        FeatureCreateInput()
    assert "Field required" in str(exc.value)

def test_feature_create_title_too_long():
    """Title cannot exceed 200 characters."""
    with pytest.raises(ValidationError) as exc:
        FeatureCreateInput(title="x" * 201)
    assert "at most 200" in str(exc.value).lower()

def test_feature_create_title_whitespace_only():
    """Title cannot be whitespace-only."""
    with pytest.raises(ValidationError) as exc:
        FeatureCreateInput(title="   ")
    assert "cannot be empty" in str(exc.value).lower()

def test_feature_create_invalid_priority():
    """Priority must be valid enum value."""
    with pytest.raises(ValidationError) as exc:
        FeatureCreateInput(
            title="Test",
            priority="super-critical"
        )
    assert "valid options" in str(exc.value).lower()

def test_feature_create_too_many_steps():
    """Cannot exceed 50 steps."""
    with pytest.raises(ValidationError) as exc:
        FeatureCreateInput(
            title="Test",
            steps=[f"Step {i}" for i in range(51)]
        )
    assert "50 steps" in str(exc.value)

def test_feature_create_step_too_long():
    """Each step cannot exceed 500 characters."""
    with pytest.raises(ValidationError) as exc:
        FeatureCreateInput(
            title="Test",
            steps=["x" * 501]
        )
    assert "500 characters" in str(exc.value)
```

---

## 8. Files & Dependencies

### Files to Create

| File | Purpose | LOC Est. |
|------|---------|---------|
| `src/python/wipnote/pydantic_models.py` | All Pydantic model definitions | 400-500 |
| `src/python/wipnote/cli_utils.py` | Error display and validation utilities | 100-150 |
| `tests/python/test_cli_models.py` | Unit tests for all models | 300-400 |

### Files to Modify

| File | Changes | Impact |
|------|---------|--------|
| `src/python/wipnote/cli.py` | Add validation wrappers, modify command handlers | Medium |
| `pyproject.toml` | No changes (pydantic already a dependency) | None |

### Dependencies

**Already Available:**
- `pydantic>=2.0.0` ✓
- `rich>=13.0.0` ✓
- `pytest>=7.0.0` ✓

**No Additional Dependencies Required**

---

## 9. Success Criteria

### Phase 1 MVP Success

- [ ] All Pydantic models defined with comprehensive field validation
- [ ] All validation errors display beautifully with Rich console
- [ ] Feature create/start/complete/delete commands use Pydantic validation
- [ ] Session start/end/handoff/link commands use Pydantic validation
- [ ] Track new/list/spec/plan/delete commands use Pydantic validation
- [ ] All commands work exactly as before (backward compatible)
- [ ] All mypy type checks pass (no "Any" type usage)
- [ ] All ruff lint checks pass
- [ ] 100+ unit tests for model validation
- [ ] All pytest tests pass with >95% coverage
- [ ] Error messages are user-friendly and actionable
- [ ] No breaking changes to CLI interface

### Phase 1 Code Quality

- [ ] Zero type errors from mypy
- [ ] Zero lint warnings from ruff
- [ ] All docstrings present and complete
- [ ] All validators have examples
- [ ] All models have clear descriptions

### Phase 2+ Roadmap

- [ ] Pydantic Settings for global configuration
- [ ] REST API integration (reuse models)
- [ ] SDK enhancement (use models for type safety)
- [ ] Config file validation (`.wipnote.env`)
- [ ] Nested model validation (complex structures)

---

## 10. Risks & Mitigation

### Risk: Breaking Changes

**Concern:** Pydantic validation might reject previously valid inputs

**Mitigation:**
- Run comprehensive testing with existing CLI usage
- Provide clear migration guide if constraints change
- Start with Phase 1 (less critical commands)

### Risk: Performance Impact

**Concern:** Pydantic validation adds overhead

**Mitigation:**
- Validation only runs once at CLI entry point
- No performance impact on SDK/internal operations
- Negligible overhead (< 1ms per command)

### Risk: Complex Validation Logic

**Concern:** Some commands need complex multi-field validation

**Mitigation:**
- Pydantic supports cross-field validators via `@root_validator`
- Start with simple constraints, add complexity gradually
- Document all validators with examples

---

## Appendix A: Pydantic Field Constraint Reference

### String Constraints

```python
# Length validation
title: str = Field(..., min_length=1, max_length=200)

# Pattern matching (regex)
collection: str = Field(..., pattern="^[a-z_]+$")

# Enum selection
priority: Literal["low", "medium", "high"] = Field(...)
```

### Numeric Constraints

```python
# Range validation
min_score: float = Field(..., ge=0, le=100)  # 0-100
limit: int = Field(..., ge=1, le=100)  # 1-100
port: int = Field(..., ge=1024, le=65535)  # Valid ports
```

### List Constraints

```python
# Length and item validation
steps: list[str] = Field(
    ...,
    max_length=50,  # Max 50 items
    description="Steps"
)

# Custom validation per item
@field_validator('steps')
@classmethod
def validate_steps(cls, v):
    for step in v:
        if len(step) > 500:
            raise ValueError(f"Step too long: {step[:50]}...")
    return v
```

### Optional & Defaults

```python
# Optional with None default
description: Optional[str] = Field(default=None, max_length=5000)

# Optional with non-None default
priority: str = Field(default="medium", pattern="^[a-z]+$")

# Required (no default)
title: str = Field(...)  # Must provide value
```

---

## Appendix B: Common Validation Patterns

### Email Validation
```python
from pydantic import EmailStr
email: EmailStr = Field(...)
```

### URL Validation
```python
from pydantic import HttpUrl
url: HttpUrl = Field(...)
```

### Path Validation
```python
from pathlib import Path
path: Path = Field(...)
```

### Enum with Descriptions
```python
class PriorityEnum(str, Enum):
    """Priority levels with descriptions."""
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"

priority: PriorityEnum = Field(
    default=PriorityEnum.MEDIUM,
    description="Priority level: low, medium, high, or critical"
)
```

---

## Appendix C: Testing Checklist

### Model Testing

- [ ] Test valid input acceptance
- [ ] Test required field validation
- [ ] Test string length constraints
- [ ] Test enum validation
- [ ] Test numeric range validation
- [ ] Test regex pattern validation
- [ ] Test custom validators
- [ ] Test default value application
- [ ] Test optional field handling
- [ ] Test edge cases (empty strings, max values, etc.)

### CLI Integration Testing

- [ ] Test successful command execution
- [ ] Test validation error display
- [ ] Test error message readability
- [ ] Test backward compatibility
- [ ] Test JSON output format
- [ ] Test help text display
- [ ] Test environment variable substitution
- [ ] Test default value application

### Code Quality Testing

- [ ] Mypy type checking passes
- [ ] Ruff linting passes
- [ ] Pytest coverage >95%
- [ ] All docstrings present
- [ ] All examples in docstrings
- [ ] No unused imports
- [ ] No hardcoded values

---

## Conclusion

This specification provides a complete roadmap for Phase 1 Pydantic integration. All models are defined, all validation rules are specified, and all integration points are identified. The implementation is straightforward and can be completed incrementally with clear testing at each step.

**Next Steps for Codex:**
1. Create `pydantic_models.py` with all model definitions
2. Add error handling utilities to `cli_utils.py`
3. Integrate validation wrappers into `cli.py`
4. Write comprehensive unit tests
5. Verify all quality gates pass
6. Document and deploy

---

**Specification Ready:** Yes ✓
**Codex Implementation Ready:** Yes ✓
**Estimated Effort:** 11-16 hours ✓
**Complexity:** Medium (straightforward patterns, well-defined scope) ✓

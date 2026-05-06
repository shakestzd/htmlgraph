# Pydantic Integration Guide

## Overview

Phase 1 of Wipnote implementation uses Pydantic v2 models for type-safe CLI argument validation. This ensures all CLI inputs are validated before being passed to SDK operations.

## Why Pydantic?

- **Type Safety**: Static typing for CLI arguments at the Pydantic model level
- **Validation**: Automatic validation with clear error messages
- **Whitespace Handling**: Automatic stripping of leading/trailing whitespace
- **Defaults**: Sensible defaults for optional fields
- **Settings Integration**: Support for environment variables and .env files via pydantic-settings

## Using Pydantic Models

### Feature Commands

```python
from wipnote.pydantic_models import FeatureCreateInput

# Valid input
input = FeatureCreateInput(
    title="New Feature",
    priority="high",
    description="Feature description"
)

# Invalid input - raises ValidationError
input = FeatureCreateInput(title="")  # Title required
input = FeatureCreateInput(title="x" * 201)  # Max 200 characters
input = FeatureCreateInput(priority="critical")  # Invalid priority
```

### Session Commands

```python
from wipnote.pydantic_models import SessionStartInput, SessionEndInput

# Start session with minimal input
session_input = SessionStartInput()

# Start with all fields
session_input = SessionStartInput(
    session_id="sess-123",
    title="My Session",
    agent="claude"
)

# End session
end_input = SessionEndInput(
    session_id="sess-123",
    notes="Work completed",
    recommend="Review PR",
    blocker=["missing dependency"]
)
```

## Model Reference

### FeatureCreateInput
- `title`: str (1-200 chars, required, whitespace stripped)
- `priority`: Literal["low", "medium", "high"] (default: "medium")
- `description`: str | None (max 1000 chars, whitespace stripped)
- `steps`: int | None (1-50, optional)
- `collection`: str (default: "features")

### FeatureStartInput, FeatureCompleteInput, FeaturePrimaryInput, FeatureClaimInput, FeatureReleaseInput
- `feature_id`: str (required, min 1 char, whitespace stripped)
- `collection`: str (default: "features")

### SessionStartInput
- `session_id`: str | None (optional, whitespace stripped)
- `title`: str | None (max 500 chars, optional)
- `agent`: str | None (optional)

### SessionEndInput
- `session_id`: str (required, min 1 char, whitespace stripped)
- `notes`: str | None (max 2000 chars, optional)
- `recommend`: str | None (max 500 chars, optional)
- `blocker`: list[str] | None (optional)

### SessionListInput
- `status`: Literal["active", "ended"] | None (optional filter)
- `limit`: int (default: 20, min: 1, max: 100)
- `offset`: int (default: 0, min: 0)

### ActivityTrackInput
- `tool`: str (required, 1-100 chars, whitespace stripped)
- `summary`: str (required, 1-500 chars, whitespace stripped)
- `files`: list[str] | None (optional)
- `session`: str | None (optional, auto-detected if not provided)
- `failed`: bool (default: False)

### SpikeCreateInput
- `title`: str (required, 1-200 chars, whitespace stripped)
- `findings`: str | None (max 5000 chars, optional)
- `priority`: Literal["low", "medium", "high"] (default: "medium")

### TrackCreateInput
- `title`: str (required, 1-200 chars, whitespace stripped)
- `priority`: Literal["low", "medium", "high"] (default: "medium")
- `description`: str | None (max 1000 chars, optional)

### TrackSpecInput, TrackPlanInput
- `track_id`: str (required, min 1 char, whitespace stripped)
- `title`: str (required, 1-200 chars, whitespace stripped)
- `content`: str | None (max 5000 chars, optional)

### ArchiveCreateInput
- `title`: str (required, 1-200 chars, whitespace stripped)
- `items`: list[str] | None (optional)
- `description`: str | None (max 1000 chars, optional)

## Error Handling

Validation errors are displayed with Rich formatting and helpful context:

```python
from wipnote.pydantic_models import FeatureCreateInput
from pydantic import ValidationError

try:
    input = FeatureCreateInput(title="")
except ValidationError as e:
    # Error message indicates why validation failed
    print(e)
    # Output: "1 validation error for FeatureCreateInput\ntitle\n  String should have at least 1 character..."
```

## Configuration Management

Wipnote uses `pydantic-settings` (BaseSettings) for configuration management:

```python
from wipnote.config import config

# Access configuration
print(config.graph_dir)        # ~/.wipnote
print(config.features_dir)     # ~/.wipnote/features
print(config.debug)            # False by default

# Configuration from environment variables
# Set HTMLGRAPH_DEBUG=true to enable debug mode
# Set HTMLGRAPH_GRAPH_DIR=/path/to/graph for custom graph directory

# Create directories
config.ensure_directories()

# Get config as dictionary
config_dict = config.get_config_dict()
```

## CLI Integration

CLI commands use Pydantic models for argument validation:

```bash
# Valid command
wipnote feature create "My Feature" --priority high

# Invalid command (caught by Pydantic validation)
wipnote feature create ""           # Error: title required
wipnote feature create "x" * 201    # Error: title max 200 chars
wipnote feature start --id ""       # Error: feature_id required
```

## Testing Pydantic Models

Comprehensive tests for Pydantic models are in:
- `tests/python/test_cli_pydantic_models.py` - Validation tests
- `tests/python/test_error_handling.py` - Integration tests

Run tests with:

```bash
uv run pytest tests/python/test_cli_pydantic_models.py -v
uv run pytest tests/python/test_error_handling.py -v
```

## Configuration via Environment Variables

All configuration options support environment variables with `HTMLGRAPH_` prefix:

```bash
export HTMLGRAPH_GRAPH_DIR="/path/to/graph"
export HTMLGRAPH_DEBUG=true
export HTMLGRAPH_VERBOSE=true
export HTMLGRAPH_LOG_LEVEL=DEBUG
export HTMLGRAPH_MAX_SESSIONS=200
```

Or use a `.env` file:

```
HTMLGRAPH_GRAPH_DIR=/path/to/graph
HTMLGRAPH_DEBUG=true
HTMLGRAPH_VERBOSE=true
```

## Next Steps

See Phase 2 documentation for:
- Command Pattern refactoring
- Decorator-based CLI command registration
- Dependency injection for SessionManager
- Rich output formatting

## Related Files

- Source: `src/python/wipnote/pydantic_models.py`
- Config: `src/python/wipnote/config.py`
- Tests: `tests/python/test_cli_pydantic_models.py`
- Integration: `tests/python/test_error_handling.py`

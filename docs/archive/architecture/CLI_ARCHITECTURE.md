# CLI Architecture - Technical Design Document

**Version**: 0.9.4
**Status**: Implemented
**Date**: January 12, 2026

---

## Overview

The Wipnote CLI has been refactored from a monolithic 500+ line module into a modular, hierarchical command structure. This document explains the architecture, design patterns, and how to extend it.

---

## Design Principles

### 1. **Separation of Concerns**
Each module handles a single responsibility:
- `base.py` - Command base class
- `core.py` - Core functionality (status, serve, sync)
- `work/orchestration.py` - Orchestrator control
- `work/features.py` - Feature management
- `templates/` - Output formatting

### 2. **Hierarchical Commands**
Commands are organized by domain with a tree structure:
```
wipnote
├── status
├── serve
├── sync-docs
├── feature
│   ├── create
│   ├── list
│   ├── show
│   └── update
├── track
│   ├── create
│   ├── list
│   └── plan
├── session
│   ├── list
│   ├── show
│   └── export
└── orchestrator
    ├── enable
    ├── disable
    ├── set-level
    └── reset-violations
```

### 3. **Auto-Discovery**
Commands are automatically discovered through registration:
```python
class FeatureCommand(BaseCommand):
    subcommands = {
        "create": CreateFeatureCommand(),
        "list": ListFeaturesCommand(),
        "show": ShowFeatureCommand(),
    }
```

### 4. **Consistent Interface**
All commands follow the same pattern:
```python
class MyCommand(BaseCommand):
    name = "my-command"
    description = "What it does"

    async def execute(self, args, ctx):
        # Async execution
        return result
```

---

## Core Components

### BaseCommand Class

```python
class BaseCommand:
    """Base class for all CLI commands"""

    # Command metadata
    name: str                              # "status"
    description: str                       # Help text
    aliases: List[str] = []                # ["st", "check"]

    # Subcommands (for hierarchical structure)
    subcommands: Dict[str, BaseCommand] = {}

    async def execute(self, args, ctx) -> Any:
        """Execute the command"""
        raise NotImplementedError()

    def get_help(self) -> str:
        """Generate help text"""
        # Auto-generated from description
```

**Key Features**:
- Async execution for I/O operations
- Context object for shared state
- Subcommand support for hierarchies
- Built-in help generation

### CLI Dispatcher

```python
# cli/main.py
async def dispatch_command(name: str, args: List[str]) -> None:
    """Route command to appropriate handler"""

    # Resolve command name to handler
    handler = command_registry.get(name)

    if not handler:
        print(f"Unknown command: {name}")
        return

    # Execute command
    ctx = CLIContext()
    result = await handler.execute(args, ctx)

    # Format output
    output = format_output(result)
    print(output)
```

**Routing**:
1. Parse command name from args[0]
2. Look up in `command_registry`
3. If subcommand, recursively resolve
4. Execute handler
5. Format and display output

### Command Registry

```python
command_registry = {
    "status": StatusCommand(),
    "serve": ServeCommand(),
    "feature": FeatureCommand(),
    "track": TrackCommand(),
    "session": SessionCommand(),
    "orchestrator": OrchestratorCommand(),
    "install-hooks": InstallHooksCommand(),
}
```

**Registration Points**:
- Top-level commands in `main.py`
- Subcommands in parent command's `subcommands` dict
- Aliases resolved transparently

---

## Module Organization

### `cli/base.py` - Foundation
Defines `BaseCommand` abstract class and `CLIContext`.

**Classes**:
- `BaseCommand` - Abstract base for all commands
- `CLIContext` - Execution context (session, config, SDK)
- `CommandError` - Exception for command failures

**Usage**:
```python
from wipnote.cli.base import BaseCommand, CLIContext

class MyCommand(BaseCommand):
    async def execute(self, args, ctx: CLIContext):
        sdk = ctx.sdk  # Get SDK instance
        config = ctx.config  # Get config
```

### `cli/core.py` - Core Commands
Essential commands for Wipnote operation.

**Commands**:
- `StatusCommand` - Show project status
- `ServeCommand` - Start dashboard server
- `SyncDocsCommand` - Sync documentation files

**Example**:
```python
class StatusCommand(BaseCommand):
    name = "status"
    description = "Show project status"

    async def execute(self, args, ctx: CLIContext):
        sdk = ctx.sdk

        # Get status data
        features = sdk.features.list()
        sessions = sdk.sessions.list()

        # Format and return
        return {
            "features": len(features),
            "sessions": len(sessions),
            "status": "healthy"
        }
```

### `cli/work/orchestration.py` - Orchestrator Commands
Control enforcement and violation tracking.

**Commands**:
- `OrchestratorCommand` - Parent command
- `EnableCommand` - Enable enforcement
- `DisableCommand` - Disable enforcement
- `SetLevelCommand` - Set enforcement level
- `ResetViolationsCommand` - Clear violations

**Example**:
```python
class EnableCommand(BaseCommand):
    name = "enable"
    description = "Enable orchestrator enforcement"

    async def execute(self, args, ctx: CLIContext):
        config = ctx.config
        config.orchestrator.enabled = True
        config.save()
        return {"status": "enabled"}
```

### `cli/work/features.py` - Feature Commands
Manage features in Wipnote.

**Commands**:
- `FeatureCommand` - Parent command
- `CreateFeatureCommand` - Create feature
- `ListFeaturesCommand` - List features
- `ShowFeatureCommand` - Show details
- `UpdateFeatureCommand` - Update feature

### `cli/work/sessions.py` - Session Commands
Query and manage sessions.

**Commands**:
- `SessionCommand` - Parent command
- `ListSessionsCommand` - List sessions
- `ShowSessionCommand` - Show session details
- `ExportSessionCommand` - Export session data

### `cli/work/tracks.py` - Track Commands
Manage tracks (multi-feature initiatives).

**Commands**:
- `TrackCommand` - Parent command
- `CreateTrackCommand` - Create track
- `ListTracksCommand` - List tracks
- `PlanTrackCommand` - Planning

### `cli/models.py` - Data Types
Type definitions for CLI models.

**Types**:
- `CLIModel` - Base model
- `CommandResult` - Command output
- `CLIConfig` - Configuration data
- Field validators and serializers

### `cli/constants.py` - Shared Constants
Shared values across CLI modules.

**Constants**:
```python
DEFAULT_PAGE_SIZE = 20
MAX_OUTPUT_WIDTH = 120
TIMESTAMP_FORMAT = "%Y-%m-%d %H:%M:%S"
```

### `cli/templates/` - Output Formatting
Templates for consistent output.

**Templates**:
- `cost_dashboard.py` - Cost/performance dashboard
- `table_formatter.py` - Table output formatting
- `json_formatter.py` - JSON output

---

## Execution Flow

### Command Execution Sequence

```
1. User Input
   $ wipnote feature create "New Feature"

2. Parse Arguments
   → command: "feature"
   → subcommand: "create"
   → args: ["New Feature"]

3. Resolve Handler
   handler = registry["feature"].subcommands["create"]

4. Create Context
   ctx = CLIContext()
   ctx.sdk = SDK()
   ctx.config = load_config()

5. Execute Handler
   result = await handler.execute(["New Feature"], ctx)

6. Format Output
   formatted = format_output(result)

7. Display Result
   print(formatted)
```

### Context Management

```python
class CLIContext:
    """Execution context shared across commands"""

    def __init__(self):
        self.sdk = SDK(agent="cli")           # SDK instance
        self.config = load_orchestrator_config()  # Config
        self.session_id = os.getenv("HTMLGRAPH_SESSION_ID")
        self.output_format = "table"            # Output format
        self.verbose = False                    # Verbosity level
```

---

## Adding New Commands

### Step 1: Create Command Class

```python
# cli/work/my_commands.py

from wipnote.cli.base import BaseCommand, CLIContext

class MyCommand(BaseCommand):
    name = "my-command"
    description = "What my command does"

    async def execute(self, args, ctx: CLIContext):
        # Your implementation here
        return {"result": "success"}
```

### Step 2: Register Command

```python
# cli/main.py

from wipnote.cli.work.my_commands import MyCommand

command_registry = {
    # ... other commands
    "my-command": MyCommand(),
}
```

### Step 3: Test Command

```python
# tests/python/test_my_command.py

import pytest
from wipnote.cli.work.my_commands import MyCommand
from wipnote.cli.base import CLIContext

@pytest.mark.asyncio
async def test_my_command():
    ctx = CLIContext()
    cmd = MyCommand()

    result = await cmd.execute([], ctx)

    assert result["result"] == "success"
```

### Step 4: Command Available

```bash
$ wipnote my-command
What my command does
result: success
```

---

## Configuration Management

### Orchestrator Configuration

```yaml
# orchestrator-config.yaml
orchestrator:
  enabled: false                    # Master switch
  mode: "warn"                      # warn|enforce|block
  circuit_breaker:
    violations: 3                   # Threshold
    decay_time: 120                 # Seconds
    window: 10                       # Rapid collapse window
  delegation:
    min_context_ratio: 0.9          # Context preservation
    parallelization_threshold: 0.7  # Parallel work trigger
```

### Configuration Loading

```python
def load_orchestrator_config() -> OrchestratorConfig:
    """Load configuration with defaults"""

    config_path = get_config_path()

    if config_path.exists():
        return OrchestratorConfig.from_yaml(config_path)

    # Return defaults if missing
    return OrchestratorConfig.defaults()
```

### CLI Modification

```bash
# Enable enforcement
wipnote orchestrator enable
# → Updates orchestrator-config.yaml
# → Effective immediately

# Set enforcement level
wipnote orchestrator set-level enforce
# → Changes mode to "enforce"

# Reset violations
wipnote orchestrator reset-violations
# → Clears violation counter
```

---

## Error Handling

### Command Errors

```python
class CommandError(Exception):
    """Raised when command execution fails"""

    def __init__(self, message: str, exit_code: int = 1):
        self.message = message
        self.exit_code = exit_code
```

### Usage

```python
async def execute(self, args, ctx: CLIContext):
    if not args:
        raise CommandError("Missing required argument: name")

    # Command logic
```

### Error Display

```bash
$ wipnote my-command
Error: Missing required argument: name
exit code: 1
```

---

## Testing Strategy

### Unit Tests
Test individual commands in isolation.

```python
@pytest.mark.asyncio
async def test_command_success():
    ctx = CLIContext()
    cmd = MyCommand()
    result = await cmd.execute(["arg1"], ctx)
    assert result["status"] == "success"

@pytest.mark.asyncio
async def test_command_error():
    ctx = CLIContext()
    cmd = MyCommand()
    with pytest.raises(CommandError):
        await cmd.execute([], ctx)
```

### Integration Tests
Test commands with actual database.

```python
@pytest.mark.asyncio
async def test_command_integration():
    # Setup database
    sdk = SDK()

    # Create test data
    feature = sdk.features.create("Test")

    # Execute command
    ctx = CLIContext()
    ctx.sdk = sdk
    cmd = ListFeaturesCommand()
    result = await cmd.execute([], ctx)

    # Verify result
    assert len(result["features"]) >= 1
```

### End-to-End Tests
Test command-line interface.

```python
def test_cli_command(capsys):
    # Execute CLI
    result = subprocess.run(
        ["wipnote", "status"],
        capture_output=True,
        text=True
    )

    # Verify output
    assert result.returncode == 0
    assert "status" in result.stdout
```

---

## Performance Considerations

### Lazy Loading
Only load commands when needed:
```python
async def dispatch(name: str):
    # Only import the specific command handler
    handler = command_registry.get(name)
    # Execute
```

### Caching
Cache SDK instances to avoid recreating:
```python
class CLIContext:
    _sdk = None

    @property
    def sdk(self):
        if self._sdk is None:
            self._sdk = SDK()
        return self._sdk
```

### Async Execution
Commands use async/await for I/O:
```python
async def execute(self, args, ctx):
    # Non-blocking I/O
    result = await ctx.sdk.features.list()
```

---

## Extending the CLI

### Adding Subcommands

```python
class ParentCommand(BaseCommand):
    name = "parent"

    # Register subcommands
    subcommands = {
        "child1": ChildCommand1(),
        "child2": ChildCommand2(),
    }

# Usage:
# $ wipnote parent child1
# $ wipnote parent child2
```

### Adding Aliases

```python
class StatusCommand(BaseCommand):
    name = "status"
    aliases = ["st", "check", "health"]

# Usage:
# $ wipnote status
# $ wipnote st
# $ wipnote check
```

### Custom Output Formatting

```python
from wipnote.cli.templates import TableFormatter

class MyCommand(BaseCommand):
    async def execute(self, args, ctx):
        data = [
            {"id": "1", "name": "Feature 1"},
            {"id": "2", "name": "Feature 2"},
        ]

        formatter = TableFormatter()
        return formatter.format_table(data, columns=["id", "name"])
```

---

## Troubleshooting

### Command Not Found
```bash
$ wipnote unknown-command
Unknown command: unknown-command

# Check available commands
$ wipnote --help
```

### Command Execution Error
```bash
$ wipnote feature create
Error: Missing required argument: name

# Check command help
$ wipnote feature create --help
```

### Configuration Issues
```bash
# Check current configuration
$ wipnote orchestrator status

# Reset to defaults
$ rm orchestrator-config.yaml
$ wipnote orchestrator status
```

---

## Future Enhancements

### Planned Features
- [ ] Command auto-completion (bash/zsh)
- [ ] Interactive command mode
- [ ] Command aliases and shortcuts
- [ ] Config file validation
- [ ] Multi-command batching
- [ ] Command chaining with pipes

### Extensibility Points
- Custom command implementations
- Custom output formatters
- Custom data models
- Hook integration points

---

## References

- [CLI Module Refactoring Summary](./CLI_MODULE_REFACTORING_SUMMARY.md)
- [Release Notes 0.9.4](./RELEASE_NOTES_0.9.4.md)
- [AGENTS.md](./AGENTS.md) - SDK reference
- [CLAUDE.md](./CLAUDE.md) - Development guide

---

**End of Technical Design Document**

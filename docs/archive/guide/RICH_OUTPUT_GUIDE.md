# Rich Output Formatting Guide

**For:** Wipnote CLI Implementation
**Feature:** Phase 1A/1B - Maximize Rich Console
**Target Audience:** Codex, Copilot, and future CLI developers

---

## Table of Contents

1. [Color Scheme](#color-scheme)
2. [Symbols & Icons](#symbols--icons)
3. [Components](#components)
4. [Common Patterns](#common-patterns)
5. [Examples](#examples)
6. [Testing & Validation](#testing--validation)
7. [Backward Compatibility](#backward-compatibility)

---

## Color Scheme

Wipnote uses a consistent color scheme for all CLI output:

### Primary Colors

| Color | Markup | Usage | Example |
|-------|--------|-------|---------|
| Red | `[red]...[/red]` | Errors, failures, blocking issues | `[red]✗ Operation failed[/red]` |
| Green | `[green]...[/green]` | Success, completion, positive results | `[green]✓ Feature created[/green]` |
| Yellow | `[yellow]...[/yellow]` | Warnings, caution, potential issues | `[yellow]⚠ Uncommitted changes[/yellow]` |
| Cyan | `[cyan]...[/cyan]` | Information, status, metadata | `[cyan]ℹ Project: wipnote[/cyan]` |
| Magenta | `[magenta]...[/magenta]` | Important items, highlights | `[magenta]★ High priority[/magenta]` |

### Style Modifiers

| Style | Markup | Usage |
|-------|--------|-------|
| Bold | `[bold]...[/bold]` | Headers, emphasized text |
| Dim | `[dim]...[/dim]` | Secondary info, hints |
| Italic | `[italic]...[/italic]` | Emphasis, alternatives |
| Underline | `[underline]...[/underline]` | Links, important terms |

### Combination Example

```python
# Bold red error with symbol
console.print("[bold red]✗ Critical Error:[/bold red] Operation failed")

# Dim cyan hint
console.print("[dim cyan]Hint: Use --help for more options[/dim cyan]")

# Bold green success
console.print("[bold green]✓ Successfully created feature[/bold green]")
```

---

## Symbols & Icons

Standard symbols for consistent visual feedback:

### Status Symbols

| Symbol | Code | Usage |
|--------|------|-------|
| ✓ | U+2713 | Success, completed, checked |
| ✗ | U+2717 | Error, failed, blocking |
| ⚠ | U+26A0 | Warning, caution, attention needed |
| ℹ | U+2139 | Information, help, details |
| ✌ | U+270C | Good, okay, approved |
| ✋ | U+270B | Stop, pause, hold |
| ★ | U+2605 | Important, priority, highlight |
| ⊙ | U+2299 | In progress, processing |
| → | U+2192 | Next, following, direction |
| ← | U+2190 | Previous, back, revert |

### Usage Examples

```python
# Success
console.print("[green]✓ Feature created successfully[/green]")

# Error
console.print("[red]✗ Failed to create feature[/red]")

# Warning
console.print("[yellow]⚠ This action cannot be undone[/yellow]")

# Info
console.print("[cyan]ℹ Run 'wipnote status' to check progress[/cyan]")

# Priority
console.print("[magenta]★ High priority feature detected[/magenta]")

# In Progress
console.print("[cyan]⊙ Processing analytics...[/cyan]")
```

---

## Components

Rich provides several components for structured output:

### 1. Console.print() - Basic Text Output

**Purpose:** Display colored, styled text

**Syntax:**
```python
console.print("Text", style="color")
console.print("[color]Text[/color]")
console.print("[color bold]Text[/color bold]")
```

**Examples:**
```python
# Error message
console.print("[red]Error: Invalid feature ID[/red]")

# Success message
console.print("[green]✓ Feature saved[/green]")

# Info with style
console.print("[cyan bold]Project Status[/cyan bold]")

# Combined
console.print("[bold green]✓[/bold green] Feature [bold]my-feature[/bold] created")
```

### 2. Table - Structured Data

**Purpose:** Display data in rows and columns

**Syntax:**
```python
from rich.table import Table

table = Table(title="Features", box=box.ROUNDED, show_header=True, header_style="bold cyan")
table.add_column("ID", style="magenta")
table.add_column("Title", style="green")
table.add_column("Status", style="yellow")
table.add_column("Priority", style="red")

table.add_row("feat-1", "Feature One", "todo", "high")
table.add_row("feat-2", "Feature Two", "in-progress", "medium")

console.print(table)
```

**Output:**
```
╭─────────────────────────────────────────╮
│ Features                                │
├──────┬───────────────┬──────────┬─────┤
│ ID   │ Title         │ Status   │ Pri │
├──────┼───────────────┼──────────┼─────┤
│ feat │ Feature One   │ todo     │ hi  │
│ feat │ Feature Two   │ in-prog… │ med │
╰──────┴───────────────┴──────────┴─────╯
```

**Common Styles:**
- `box.ROUNDED` - Rounded corners (default)
- `box.SIMPLE` - Minimal borders
- `box.DOUBLE` - Double line borders
- `box.ASCII` - ASCII-only (for compatibility)

**Column Options:**
```python
table.add_column(
    "Name",
    style="cyan",              # Column color
    justify="left",            # left, center, right
    width=20,                  # Fixed width
    no_wrap=False              # Wrap long text
)
```

### 3. Panel - Grouped Content

**Purpose:** Display content in a highlighted box

**Syntax:**
```python
from rich.panel import Panel

panel = Panel(
    "Content here",
    title="Header",
    style="blue",
    expand=False
)
console.print(panel)
```

**Output:**
```
╭─ Header ───────────────────╮
│                             │
│ Content here                │
│                             │
╰─────────────────────────────╯
```

**Options:**
```python
Panel(
    content,
    title="Title",             # Header text
    subtitle="Subtitle",       # Footer text
    style="cyan",              # Border color
    expand=True,               # Full width
    border_style="blue"        # Different border style
)
```

**Examples:**
```python
# Help text
from rich.panel import Panel
panel = Panel(
    "[cyan]Commands:\n"
    "  feature list    - Show all features\n"
    "  feature create  - Create new feature[/cyan]",
    title="[bold]Wipnote CLI[/bold]",
    style="bold green"
)
console.print(panel)

# Error panel
Panel(
    "[red]Feature not found: invalid-id[/red]",
    title="[bold red]Error[/bold red]",
    style="red"
)
```

### 4. Progress - Long Operations

**Purpose:** Show progress for time-consuming operations

**Syntax:**
```python
from rich.progress import Progress

with Progress() as progress:
    task = progress.add_task("[cyan]Processing...", total=100)
    while not progress.finished:
        # Do work
        progress.update(task, advance=10)
```

**Output:**
```
Processing... ████████░░░░░░░░░░ 50% 0:00:30
```

**Advanced Usage:**
```python
from rich.progress import Progress, SpinnerColumn, TextColumn, BarColumn, TimeElapsedColumn

with Progress(
    SpinnerColumn(),
    TextColumn("[progress.description]{task.description}"),
    BarColumn(),
    TimeElapsedColumn(),
) as progress:
    task = progress.add_task("Building features...", total=50)
    for i in range(50):
        # Do work
        progress.update(task, advance=1)
```

### 5. Prompt - Interactive Input

**Purpose:** Get user input with validation

**Syntax:**
```python
from rich.prompt import Prompt, Confirm

# Text input
name = Prompt.ask("Enter your name", default="Claude")

# Yes/No confirmation
confirmed = Confirm.ask("Create feature?", default=True)

# With choices
choice = Prompt.ask(
    "Select priority",
    choices=["high", "medium", "low"],
    default="medium"
)
```

**Output:**
```
Enter your name [Claude]:
Create feature? [Y/n]: y
Select priority [high/medium/low] [medium]: high
```

### 6. Status - Indeterminate Progress

**Purpose:** Show activity status for unknown duration

**Syntax:**
```python
with console.status("[bold cyan]Processing..."):
    # Do work
    time.sleep(2)
```

**Output:**
```
⠙ Processing...
```

---

## Common Patterns

### Pattern 1: Success Message

```python
# Simple
console.print("[green]✓ Operation successful[/green]")

# With details
console.print(f"[green]✓[/green] Feature [bold]{feature_id}[/bold] created successfully")

# With secondary text
console.print("[green]✓ Created[/green]", "feature-001", style="dim")
```

### Pattern 2: Error Message

```python
# Simple
console.print("[red]✗ Operation failed[/red]")

# With details
console.print(f"[red]✗ Error:[/red] Invalid feature ID: {feature_id}")

# Multi-line error
console.print("[red]✗ Failed to create feature[/red]")
console.print(f"  [dim]Reason: {error_message}[/dim]")
```

### Pattern 3: Warning Message

```python
# Simple
console.print("[yellow]⚠ Warning: This action is permanent[/yellow]")

# With suggestion
console.print(f"[yellow]⚠[/yellow] No features found. Create one with: wipnote feature create")
```

### Pattern 4: Info Message

```python
# Simple
console.print("[cyan]ℹ Total features: 15[/cyan]")

# With details
console.print(f"[cyan]ℹ Project:[/cyan] {project_name}")
console.print(f"[cyan]ℹ Location:[/cyan] {location}")
```

### Pattern 5: List/Table of Items

```python
# Simple list
table = Table(show_header=True, header_style="bold cyan")
table.add_column("ID", style="magenta")
table.add_column("Name", style="green")
table.add_column("Status", style="yellow")

for item in items:
    table.add_row(item.id, item.name, item.status)

console.print(table)
```

### Pattern 6: Grouped Help Text

```python
from rich.panel import Panel

help_text = """
[cyan]Commands:[/cyan]
  [green]feature list[/green]      Show all features
  [green]feature create[/green]    Create new feature
  [green]feature show ID[/green]   Show feature details

[cyan]Options:[/cyan]
  [green]--format json[/green]     Output as JSON
  [green]--help[/green]            Show this message
"""

panel = Panel(help_text, title="[bold]Wipnote CLI Help[/bold]", style="bold cyan")
console.print(panel)
```

### Pattern 7: Progress for Long Operations

```python
from rich.progress import Progress

features = get_features()  # Expensive operation
with Progress() as progress:
    task = progress.add_task("[cyan]Loading features...", total=len(features))

    for feature in features:
        process_feature(feature)
        progress.update(task, advance=1)

console.print("[green]✓ All features processed[/green]")
```

---

## Examples

### Example 1: Feature List Command

**Before (Plain Print):**
```python
print("Features:")
for feature in features:
    print(f"  {feature.id}: {feature.title} ({feature.status})")
```

**After (Rich Output):**
```python
from rich.table import Table
from rich import box

table = Table(
    title="[bold cyan]Features[/bold cyan]",
    box=box.ROUNDED,
    show_header=True,
    header_style="bold cyan"
)
table.add_column("ID", style="magenta", width=12)
table.add_column("Title", style="green", width=30)
table.add_column("Status", style="yellow")
table.add_column("Priority", style="red")

for feature in features:
    status_color = "green" if feature.status == "done" else "yellow"
    table.add_row(
        feature.id,
        feature.title,
        f"[{status_color}]{feature.status}[/{status_color}]",
        feature.priority
    )

console.print(table)
```

### Example 2: Feature Creation with Prompts

**Before:**
```python
import sys
name = input("Feature name: ")
if not name:
    print("Error: Name required")
    sys.exit(1)

priority = input("Priority (high/medium/low) [medium]: ") or "medium"
print(f"Creating feature: {name}")
# ... create feature
print(f"✓ Feature created: {name}")
```

**After:**
```python
from rich.prompt import Prompt, Confirm

name = Prompt.ask("Feature name")
priority = Prompt.ask(
    "Priority",
    choices=["high", "medium", "low"],
    default="medium"
)

with console.status("[cyan]Creating feature..."):
    # ... create feature

console.print(f"[green]✓ Feature[/green] [bold]{name}[/bold] [green]created successfully[/green]")
```

### Example 3: Error Handling

**Before:**
```python
try:
    feature = get_feature(feature_id)
except FeatureNotFound:
    print(f"Error: Feature not found: {feature_id}")
    sys.exit(1)
```

**After:**
```python
try:
    feature = get_feature(feature_id)
except FeatureNotFound:
    console.print(f"[red]✗ Error:[/red] Feature not found: [bold]{feature_id}[/bold]")
    console.print("[dim cyan]Use 'wipnote feature list' to see available features[/dim cyan]")
    sys.exit(1)
```

---

## Testing & Validation

### Testing Rich Output

```python
# test_cli_rich_output.py
import io
from rich.console import Console

def test_error_message_has_color():
    """Test error messages use red markup."""
    # Create string console to capture output
    string_io = io.StringIO()
    test_console = Console(file=string_io, force_terminal=True)

    # Your command that produces error
    test_console.print("[red]✗ Error message[/red]")

    output = string_io.getvalue()
    # Verify ANSI color codes are present (for terminal output)
    assert "[" in output or "\x1b[" in output  # ANSI codes

def test_json_output_clean():
    """Test JSON output has no Rich markup."""
    result = run_command(["wipnote", "feature", "list", "--format", "json"])

    # Parse JSON
    data = json.loads(result.stdout)

    # Verify no markup in output
    output_str = json.dumps(data)
    assert "[red]" not in output_str
    assert "[green]" not in output_str
    assert "\x1b[" not in output_str  # No ANSI codes
```

### Validating Implementation

```bash
# Count print() statements converted
grep -c "console.print" src/python/wipnote/cli.py

# Check for remaining plain print()
grep -n "print(" src/python/wipnote/cli.py | \
  grep -v "console.print" | grep -v "# "

# Verify Rich imports
grep "from rich" src/python/wipnote/cli.py

# Check for color usage
grep -c "\[red\]\|\[green\]\|\[yellow\]\|\[cyan\]" \
  src/python/wipnote/cli.py
```

---

## Backward Compatibility

### JSON Output

**CRITICAL: JSON output must NOT contain Rich markup**

```python
# Wrong - Rich markup in JSON
console.print("[green]✓ Feature created[/green]", file=json_output)

# Right - Plain JSON, Rich markup only in terminal
if args.format == "json":
    print(json.dumps(data), file=output)
else:
    console.print(f"[green]✓ Feature created[/green]")
```

### Environment Variables

**Disable colors if needed:**
```python
import os

# Check for NO_COLOR environment variable
if os.environ.get("NO_COLOR"):
    console = Console(no_color=True)

# Force colors even without terminal
if os.environ.get("FORCE_COLOR"):
    console = Console(force_terminal=True)
```

### Plain Text Output

**Support --plain flag if needed:**
```python
if args.plain:
    # Output without colors
    print("Feature created: my-feature")
else:
    # Output with colors
    console.print("[green]✓ Feature created: [bold]my-feature[/bold][/green]")
```

---

## Style Reference

### Available Styles

```python
# Text styles
"bold", "dim", "italic", "underline", "blink", "reverse", "conceal", "strike"

# Colors
"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"

# 256 colors (if supported)
"color(123)", "rgb(255,0,0)"

# Combinations
"bold red", "dim cyan", "italic magenta"
```

### Box Styles

```python
from rich import box

box.ROUNDED       # ╭─ ─╮
box.SQUARE        # ┌─ ─┐
box.MINIMAL       # ┌─ ─┐
box.SIMPLE        # ─────
box.DOUBLE        # ╔═ ═╗
box.THICK         # ┏━ ━┓
box.ASCII         # +--- (for compatibility)
```

---

## Quick Reference

| Need | Solution |
|------|----------|
| Colored text | `console.print("[red]text[/red]")` |
| Table | `table = Table(); console.print(table)` |
| Panel | `panel = Panel(content); console.print(panel)` |
| Progress | `with Progress() as progress: ...` |
| User input | `Prompt.ask("name")` |
| Yes/No | `Confirm.ask("continue?")` |
| Bold/Italic | `[bold]text[/bold]`, `[italic]text[/italic]` |
| Success | `[green]✓ text[/green]` |
| Error | `[red]✗ text[/red]` |
| Warning | `[yellow]⚠ text[/yellow]` |
| Info | `[cyan]ℹ text[/cyan]` |

---

## Additional Resources

- **Rich Documentation:** https://rich.readthedocs.io/
- **ANSI Color Codes:** https://en.wikipedia.org/wiki/ANSI_escape_code
- **Unicode Symbols:** https://unicode-table.com/
- **Wipnote CLI:** `src/python/wipnote/cli.py`
- **Quality Gates:** `QUALITY_GATES_PHASE_1AB.md`

---

**Version:** 1.0
**Last Updated:** 2026-01-05
**Status:** Active

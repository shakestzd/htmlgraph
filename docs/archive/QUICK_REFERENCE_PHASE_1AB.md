# Quick Reference - Phase 1A/1B: Maximize Rich Console

**For:** Codex, Copilot, and developers
**Time:** Keep visible during implementation
**Last Updated:** 2026-01-05

---

## The Task in 30 Seconds

Convert 550 remaining `print()` statements to Rich formatted `console.print()` with colors, symbols, and components.

**Baseline:** 698 print() → **Current:** 550 remaining → **Target:** 0

**Impact:** Beautiful, colored, interactive CLI output with ✓/✗ symbols, tables, panels, and progress bars.

---

## Before You Start

1. Read: `/Users/shakes/DevProjects/htmlgraph/docs/RICH_OUTPUT_GUIDE.md`
2. Review existing patterns: Lines 54-130 in `cli.py`
3. Keep this guide open while coding

---

## Three Steps for Each Conversion

### Step 1: Replace print() with console.print()

```python
# BEFORE
print(f"Error: {message}")

# AFTER
console.print(f"[red]✗ Error:[/red] {message}")
```

### Step 2: Add Color & Symbol

```python
# Errors
console.print(f"[red]✗ Error: {msg}[/red]")

# Success
console.print(f"[green]✓ Success: {msg}[/green]")

# Warnings
console.print(f"[yellow]⚠ Warning: {msg}[/yellow]")

# Info
console.print(f"[cyan]ℹ Note: {msg}[/cyan]")
```

### Step 3: Use Rich Components for Complex Output

```python
# TABLES (for lists of items)
from rich.table import Table
table = Table(show_header=True, header_style="bold cyan")
table.add_column("ID", style="magenta")
table.add_column("Status", style="yellow")
table.add_row("feat-001", "in-progress")
console.print(table)

# PANELS (for grouped content)
from rich.panel import Panel
panel = Panel("[cyan]Help text here[/cyan]", title="Help")
console.print(panel)

# PROGRESS (for long operations)
from rich.progress import Progress
with Progress() as progress:
    task = progress.add_task("Processing...", total=100)
    # ... do work
    progress.update(task, advance=10)

# PROMPTS (for user input)
from rich.prompt import Prompt, Confirm
name = Prompt.ask("Enter name")
if Confirm.ask("Continue?"):
    # ...
```

---

## Color & Symbol Cheat Sheet

| Need | Code |
|------|------|
| Red error | `[red]✗ Error[/red]` |
| Green success | `[green]✓ Done[/green]` |
| Yellow warning | `[yellow]⚠ Warning[/yellow]` |
| Cyan info | `[cyan]ℹ Note[/cyan]` |
| Bold text | `[bold]Important[/bold]` |
| Dim/secondary | `[dim]Secondary text[/dim]` |
| Combine | `[bold red]✗ Critical![/bold red]` |

---

## Quality Gates (Required Before Commit)

```bash
# Run ALL of these - they must ALL pass
uv run ruff check --fix src/python/wipnote/cli.py
uv run ruff format src/python/wipnote/cli.py
uv run mypy src/python/wipnote/cli.py --strict
uv run pytest tests/python/test_cli_rich_output.py -v
uv run pytest tests/python/test_cli_commands.py -v
```

**If any fail:** Don't commit. Fix issues first.

---

## Regression Check

```bash
# Count remaining plain print() statements
# Should DECREASE or stay same, NEVER INCREASE

grep -n "print(" src/python/wipnote/cli.py | \
  grep -v "console.print" | grep -v "# " | wc -l

# Current baseline: ~550
# Test fails if > 600
```

---

## Critical Rules

### ✅ DO

- Use `console.print()` for all terminal output
- Add colors: `[red]`, `[green]`, `[yellow]`, `[cyan]`
- Add symbols: ✓, ✗, ⚠, ℹ
- Use Rich components (Table, Panel, Progress)
- Run quality gates before committing
- Test JSON output (should have NO markup)

### ❌ DON'T

- Use plain `print()` in new code
- Mix colors inconsistently
- Forget to run tests
- Commit with failing quality gates
- Add Rich markup to JSON output
- Skip the regression check

---

## Common Patterns

### Pattern: List of Items

```python
# Use Rich.Table
table = Table(show_header=True, header_style="bold cyan", title="Features")
table.add_column("ID", style="magenta")
table.add_column("Title", style="green")
table.add_column("Status", style="yellow")

for item in items:
    table.add_row(item.id, item.title, item.status)

console.print(table)
```

### Pattern: Success Message

```python
# Simple
console.print(f"[green]✓ Created feature[/green]")

# With details
console.print(f"[green]✓[/green] Feature [bold]{name}[/bold] created")
```

### Pattern: Error with Suggestion

```python
console.print(f"[red]✗ Error:[/red] Feature not found")
console.print("[dim cyan]Hint: Use 'wipnote feature list' to see available[/dim cyan]")
```

### Pattern: Long Operation

```python
from rich.progress import Progress

with Progress() as progress:
    task = progress.add_task("[cyan]Processing...", total=len(items))
    for item in items:
        # Do work
        progress.update(task, advance=1)

console.print("[green]✓ Complete[/green]")
```

---

## JSON Output - CRITICAL

If outputting JSON (e.g., `--format json`), NO Rich markup:

```python
# WRONG
console.print(f"[green]✓ Success[/green]")  # Don't do this for JSON

# RIGHT - Conditional output
if args.format == "json":
    print(json.dumps(data))  # Plain JSON, no Rich
else:
    console.print(f"[green]✓ Success[/green]")  # Rich for terminal
```

---

## Testing Your Changes

### Manual Testing

```bash
# Run a command that uses your changes
uv run wipnote feature list

# Verify:
# ✓ Colors display correctly
# ✓ Symbols render (✓, ✗, ⚠, ℹ)
# ✓ Tables format properly
# ✓ No errors in output
```

### Automated Testing

```bash
# Rich output tests (28 tests)
uv run pytest tests/python/test_cli_rich_output.py -v

# CLI tests (17 tests)
uv run pytest tests/python/test_cli_commands.py -v

# All tests
uv run pytest tests/ -v --tb=short
```

---

## Commit Message Format

```
feat: convert [COMMAND] output to Rich formatting

- Replaced N print() statements
- Added color markup for errors/success/warnings/info
- Added symbols (✓, ✗, ⚠, ℹ)
- Used Rich.Table for [COLLECTION] listing
- Verified JSON output remains clean
- All quality gates passing

Tracked by: feat-4d5b889e (Phase 1A/1B: Maximize Rich Console)
```

---

## Troubleshooting

### "Tests are failing"
1. Run: `uv run pytest tests/python/test_cli_rich_output.py -vv`
2. Read error message
3. Check: Are you using `console.print()` instead of `print()`?

### "Mypy errors"
```bash
uv run mypy src/python/wipnote/cli.py --show-error-context --pretty
# Fix type hints or add comments like: # type: ignore
```

### "JSON output has [red] tags"
Check that JSON output branches don't call `console.print()`:
```python
if args.format == "json":
    print(json.dumps(data))  # Not console.print()
```

### "print() count not decreasing"
1. Check you're only modifying `cli.py`
2. Verify grep correctly counts: `grep -v "console.print"`
3. Look for print() in comments or docstrings

---

## Quick Stats

- **Total print() to convert:** ~550 remaining
- **Tests to keep passing:** 45 (28 Rich + 17 CLI)
- **Color scheme:** 4 colors (red, green, yellow, cyan)
- **Symbols:** 4 main symbols (✓, ✗, ⚠, ℹ)
- **Rich components:** Table, Panel, Progress, Prompt, Confirm
- **Time per conversion:** 2-5 min per command function
- **Estimated total:** 40-80 hours for complete conversion

---

## Links & References

| Resource | Link |
|----------|------|
| Rich Output Guide | `docs/RICH_OUTPUT_GUIDE.md` |
| Quality Gates | `QUALITY_GATES_PHASE_1AB.md` |
| Validation Report | `QUALITY_GATES_VALIDATION_REPORT.md` |
| Feature Tracking | `.wipnote/features/feat-4d5b889e.html` |
| Rich Documentation | https://rich.readthedocs.io/ |
| CLI Implementation | `src/python/wipnote/cli.py` |

---

## Current Progress

```
PRINT() STATEMENTS:
├─ Baseline (2026-01-04):     698
├─ Current (2026-01-05):      ~550
└─ Target (End Phase 1A/1B):  0

PROGRESS: 21.3% converted (148 statements)
STATUS: On track, ready for implementation
```

---

## Status Check Command

```bash
# Quick status (run before and after coding)
echo "=== Print() Count ===" && \
grep -c "console.print(" src/python/wipnote/cli.py && \
echo "=== Remaining print() ===" && \
grep -n "print(" src/python/wipnote/cli.py | \
  grep -v "console.print" | grep -v "# " | wc -l && \
echo "=== Tests ===" && \
uv run pytest tests/python/test_cli_rich_output.py -q
```

---

**Remember:** Quality gates are NON-NEGOTIABLE. Run them before every commit!

**Questions?** See `docs/RICH_OUTPUT_GUIDE.md` and `QUALITY_GATES_PHASE_1AB.md`

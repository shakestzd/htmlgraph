# Pre-Commit Hook: Systematic Changes Detection

## Overview

The pre-commit hook enhancement detects **incomplete systematic refactorings** before commits are made. It scans staged changes for refactoring patterns (renamed variables, functions, classes) and warns developers about potentially incomplete changes by checking if old patterns still exist elsewhere in the codebase.

**Key Feature**: This is a **warning-only check** — it never blocks commits, just alerts you to potential issues.

## What It Detects

The detection script identifies these refactoring patterns:

1. **Function Renames** — `def old_function(` → `def new_function(`
2. **Class Renames** — `class OldClass(` → `class NewClass(`
3. **Variable Renames** — `OLD_VAR = ` → `NEW_VAR = ` (constants and private vars only)
4. **Import Changes** — `from foo import old_name` → `from foo import new_name`

For each pattern detected, it searches the codebase to find remaining uses of the old name and warns you about them.

## Example Output

```
⚠️  Potential incomplete systematic change detected:
  You renamed 'old_method' → 'new_method'
  But 'old_method' still appears in:
    - src/bar.py:42
    - src/baz.py:18
  Consider updating these references.
```

## How It Works

### Hook Execution Flow

```
Pre-commit triggered
  ↓
[NEW] Check for systematic changes (warning only)
  ↓
Ruff linting (blocking)
  ↓
Ruff formatting (blocking)
  ↓
Mypy type checking (blocking)
  ↓
Commit allowed
```

### Detection Algorithm

1. **Extract staged diff** — `git diff --cached`
2. **Parse patterns** — Scan for function/class/variable definitions in added vs removed lines
3. **Find old names** — For each old pattern, search codebase with `grep -rn`
4. **Filter results** — Exclude:
   - `.wipnote/` directory
   - `tests/` directory
   - `.git/` directory
   - Generated files (`.pyc`, `.egg-info`, etc.)
   - Files already being changed
5. **Report warnings** — Show first 5 occurrences, indicate if more exist

## Files

### Created
- `scripts/hooks/check-systematic-changes.py` — Detection script (180 lines)

### Modified
- `scripts/hooks/pre-commit` — Updated to call detection script
- `.git/hooks/pre-commit` — Updated to call detection script

## Usage

The hook runs automatically on every `git commit`. No manual invocation needed.

### Manual Testing

Test the script directly (even with no staged changes):

```bash
cd /Users/shakes/DevProjects/htmlgraph
uv run scripts/hooks/check-systematic-changes.py
```

## Implementation Details

### Code Structure

```python
get_staged_diff()                    # Retrieve staged changes
get_changed_files()                  # List modified .py files
should_skip_file(filepath)           # Filter excluded directories
extract_patterns_from_diff(diff)     # Parse refactoring patterns
find_remaining_uses(...)             # Search for old pattern in codebase
main()                               # Orchestrate flow
```

### Pattern Detection

The script uses regex to identify:

```python
# Function: def\s+(\w+)\s*\(
# Class: class\s+(\w+)\s*[\(:]
# Variable: ^(\w+)\s*=
```

It tracks removed lines (prefixed `-`) and added lines (prefixed `+`), then matches patterns between them.

### Performance

- **Time**: ~200ms per commit (one grep search per detected pattern)
- **Impact**: Minimal — runs before linting checks
- **Graceful degradation**: If `grep` not available, silently skips search

## Limitations

1. **Simple heuristics** — Detects obvious renames only, not complex refactorings
2. **Python-only** — Currently checks `.py` files only
3. **False positives possible** — May warn about unrelated name changes (e.g., `old_var` → `new_var` in unrelated contexts)
4. **Grep-dependent** — Requires `grep` command (available on macOS/Linux, not Windows by default)

## Quality Gates

All code passes:

```bash
uv run ruff check --fix scripts/hooks/check-systematic-changes.py
uv run ruff format scripts/hooks/check-systematic-changes.py
uv run mypy scripts/hooks/check-systematic-changes.py
```

## Integration with CI/CD

The hook is LOCAL-ONLY. It:
- ✅ Runs on every commit (developer machine)
- ❌ Does NOT run in CI/CD (hooks don't run on remote)
- ❌ Does NOT block commits (warnings only)

For CI/CD checking, create a separate lint job that calls the script.

## Troubleshooting

### Script not running

Verify hook is executable:

```bash
ls -la /Users/shakes/DevProjects/htmlgraph/.git/hooks/pre-commit
# Should show -rwxr-xr-x
```

### Not detecting my rename

The script detects patterns in **staged changes only**. Verify:

1. Changes are staged: `git diff --cached | grep old_name`
2. New name appears in diff: `git diff --cached | grep new_name`
3. Old name still exists: `grep -r old_name src/ packages/`

### Too many false positives

Reduce noise by:
- Only renaming at function/class level (not inline variables)
- Avoiding common words (`old`, `new`, `temp`, etc.)
- Checking staged diff before committing: `git diff --cached`

## Future Enhancements

Potential improvements:

1. **Multi-language support** — Extend to JS, TypeScript, Go, Rust
2. **Smart filtering** — Reduce false positives with AST parsing
3. **Configuration** — Allow project-specific patterns via `.wipnote-config.yml`
4. **Integration** — Hook into CI/CD with optional blocking mode
5. **Database tracking** — Log systematic changes for project analytics

## See Also

- `CLAUDE.md` — General development guidelines
- `.claude/rules/code-hygiene.md` — Quality standards
- `scripts/hooks/pre-commit` — Hook implementation

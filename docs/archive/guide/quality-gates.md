# Quality Gates - Pre-Commit Hooks and Type Safety

Wipnote enforces automated quality standards through pre-commit hooks and type checking. This ensures all code merged to `main` meets strict quality criteria.

## Quick Start

### Install Pre-Commit Hooks

```bash
# Install pre-commit framework and hooks
pre-commit install

# Verify hooks are installed
pre-commit run --all-files
```

### Run Checks Before Commit

Pre-commit hooks automatically run when you attempt a commit:

```bash
# Try to commit - hooks will run automatically
git add .
git commit -m "Your message"

# If checks fail, fix errors and try again
# Hooks will re-run on the next commit attempt
```

## Quality Gates Overview

All commits to Wipnote are validated against these gates:

| Gate | Tool | Purpose | Auto-Fix |
|------|------|---------|----------|
| **Formatting** | Ruff | Python code style (PEP 8) | ✅ Yes |
| **Type Safety** | Mypy | Static type checking | ❌ No (requires manual fix) |
| **Linting** | Ruff | Code quality issues | ✅ Yes (most) |
| **Tests** | Pytest | All tests must pass | ❌ No (requires fix) |
| **Whitespace** | Pre-commit | Trailing whitespace, EOF | ✅ Yes |
| **File Size** | Pre-commit | Large files detection | ❌ No (requires review) |

## Gate Details

### 1. Ruff Format (Auto-fixes)

Ensures consistent code formatting across the project.

```bash
# Run manually
uv run ruff format src/python/wipnote/

# Pre-commit runs with --fix flag (auto-corrects)
```

**Common auto-fixes:**
- Line length (88 characters)
- Quote style (double quotes preferred)
- Whitespace around operators
- Import sorting

**Example:**
```python
# Before (fails format check)
x=1+2
result = some_function(  arg1 , arg2  )

# After (auto-fixed)
x = 1 + 2
result = some_function(arg1, arg2)
```

### 2. Ruff Lint (Auto-fixes most issues)

Detects code quality problems and style violations.

```bash
# Run manually
uv run ruff check src/python/wipnote/

# Pre-commit runs with --fix flag
```

**Checked rules:**
- Unused imports (E401, F401)
- Undefined names (F821)
- Duplicate code (E501)
- Import ordering (I001)
- Variable naming (N806, N802)

**Example:**
```python
# Before (fails lint check)
import os
import sys
import unused_module
result = undefined_var

# After (auto-fixed)
import os
import sys
result = some_defined_var
```

### 3. Mypy Type Checking (Manual fix required)

Enforces static type safety - **NO AUTO-FIX**.

```bash
# Run manually
uv run mypy src/python/wipnote/
```

**Enforced rules:**
- All function parameters must have type hints
- All function return types must be specified
- No untyped function definitions
- No `Any` types without justification

**Example - FAILS:**
```python
# ❌ Missing parameter type
def calculate(x):  # Error: Function is missing a type annotation
    return x + 1

# ❌ Missing return type
def process_data(items: list):  # Error: Function is missing a return type
    return [item * 2 for item in items]

# ❌ Untyped variable
data = fetch_remote_api()  # Error: Need type annotation for "data"
```

**Example - PASSES:**
```python
# ✅ Correct typing
def calculate(x: int) -> int:
    return x + 1

def process_data(items: list[int]) -> list[int]:
    return [item * 2 for item in items]

from typing import Any
data: dict[str, Any] = fetch_remote_api()
```

### 4. Pytest (Manual fix required)

All tests must pass - **NO AUTO-FIX**.

```bash
# Run manually
uv run pytest

# Pre-commit runs all tests
```

**Test execution:**
- Located in: `tests/python/`
- Coverage required: ≥80%
- Format: pytest standard

**Example failure:**
```
FAILED tests/python/test_sdk.py::test_create_feature - AssertionError: ...
```

**Fix:**
- Examine test failure
- Fix code logic or test expectations
- Re-run: `uv run pytest`

### 5. Standard Hooks

Basic file validation.

```bash
# Trailing whitespace
# End-of-file newlines
# YAML/JSON validation
# Merge conflict markers
# Large file detection (>1MB)
```

## Pre-Commit Hook Installation

### First Time Setup

```bash
# Install pre-commit framework
pip install pre-commit

# Install hooks from .pre-commit-config.yaml
pre-commit install

# Test on all files
pre-commit run --all-files
```

### Verify Installation

```bash
# List installed hooks
pre-commit run --all-files --dry-run

# Show hook configuration
cat .git/hooks/pre-commit

# Test with a commit
git add .
git commit -m "test"  # Hooks will run
```

## Common Scenarios

### Scenario 1: Format Issues Auto-Fixed

```bash
# Make a change with formatting issues
echo "x=1+2" >> src/python/wipnote/example.py

# Try to commit
git add .
git commit -m "Add example"

# Output:
# ruff-format: Fixed 1 formatting issue
# ruff: Fixed 2 lint issues
# ✅ Commit succeeded (changes were auto-fixed)
```

### Scenario 2: Type Error Blocks Commit

```bash
# Add untyped function
cat >> src/python/wipnote/example.py << 'EOF'
def calculate(x):  # Missing type hint!
    return x + 1
EOF

# Try to commit
git commit -m "Add calculator"

# Output:
# mypy: src/python/wipnote/example.py:42: error: Function is missing a return type annotation
# ❌ Commit blocked (requires manual fix)

# Fix it
cat >> src/python/wipnote/example.py << 'EOF'
def calculate(x: int) -> int:
    return x + 1
EOF

# Retry
git commit -m "Add calculator"
# ✅ Commit succeeds
```

### Scenario 3: Test Failure Blocks Commit

```bash
# Make code change that breaks test
# ... edit a file ...

# Try to commit
git commit -m "Update feature"

# Output:
# pytest: FAILED tests/python/test_features.py::test_create
# ❌ Commit blocked (test failed)

# Fix the test or code
# ... fix test or implementation ...

# Run tests locally to verify
uv run pytest tests/python/test_features.py::test_create

# Retry commit
git commit -m "Update feature"
# ✅ Commit succeeds
```

## Bypassing Hooks (Use with Caution)

### Skip All Hooks

```bash
# NOT RECOMMENDED - Bypasses all quality gates
git commit --no-verify -m "Emergency fix"

# This should only be used for genuine emergencies
# All code will still need to pass gates during review/CI
```

### Skip Specific Hook

Pre-commit doesn't support skipping individual hooks, but you can temporarily modify `.pre-commit-config.yaml`:

```bash
# Temporarily disable mypy (emergency only)
# Edit .pre-commit-config.yaml and comment out mypy hook

# After emergency is resolved, re-enable it
git add .pre-commit-config.yaml
git commit -m "Re-enable mypy hook"
```

## Type Safety Standards

Wipnote enforces strict type safety:

### Required Type Annotations

**All function signatures:**
```python
# ❌ BAD
def create_feature(title, description):
    pass

# ✅ GOOD
def create_feature(title: str, description: str) -> Feature:
    pass
```

**All variable declarations (when type isn't obvious):**
```python
# ❌ BAD
features = fetch_features()

# ✅ GOOD
from wipnote.collections import Feature
features: list[Feature] = fetch_features()
```

**Return types (always required):**
```python
# ❌ BAD
def get_stats(self):
    return {"count": 42}

# ✅ GOOD
from typing import TypedDict
class Stats(TypedDict):
    count: int

def get_stats(self) -> Stats:
    return {"count": 42}
```

### Using `Any` Type

Avoid `Any` unless justified. When you must use it, document why:

```python
# ❌ BAD - No justification
def process(data: Any) -> Any:
    return data

# ✅ GOOD - Documented justification
from typing import Any

def process(data: Any) -> dict:
    # Any type used because data comes from untyped external API
    # that we cannot control
    return {"processed": data}
```

## Pre-Commit Workflow

### Step-by-Step Flow

```
Developer attempts commit
    ↓
Pre-commit hooks trigger
    ↓
Ruff Format (auto-fix) ─→ Success or failure
    ↓
Ruff Lint (auto-fix) ─→ Success or failure
    ↓
Mypy Type Check ─→ Success or failure (no auto-fix)
    ↓
Pytest ─→ Success or failure (no auto-fix)
    ↓
Standard Hooks (trailing space, etc) ─→ Success or failure
    ↓
All Pass?
  ├─ YES → Commit proceeds ✅
  └─ NO → Commit blocked, show errors ❌
```

### Hook Output Example

```
======================================
pre-commit hook: ruff-format
======================================
Fixed 3 issues in src/python/wipnote/example.py

======================================
pre-commit hook: ruff
======================================
Fixed 2 issues in src/python/wipnote/example.py

======================================
pre-commit hook: mypy
======================================
✅ All type checks passed

======================================
pre-commit hook: pytest
======================================
========================= test session starts ==========================
collected 123 tests
passed 123 in 2.45s
========================= 123 passed in 2.45s ==========================

======================================
All hooks passed! ✅
======================================
Commit successful
```

## Troubleshooting

### Issue: Hooks Not Running

**Solution:**
```bash
# Reinstall hooks
pre-commit install

# Verify
pre-commit run --all-files
```

### Issue: "Module not found" in Mypy

**Solution:**
```bash
# Update mypy dependencies in .pre-commit-config.yaml
pre-commit autoupdate

# Or manually run
uv run mypy src/python/wipnote/
```

### Issue: Ruff "Unknown rule" Error

**Solution:**
```bash
# Update ruff version
uv pip install --upgrade ruff

# Check configuration in pyproject.toml
cat pyproject.toml | grep -A 20 "\[tool.ruff\]"
```

### Issue: Pre-commit Takes Too Long

**Optimization:**
```bash
# Pre-commit has built-in caching
# Clear cache if needed:
pre-commit clean

# Run only changed files
pre-commit run --files src/python/wipnote/changed_file.py
```

## Configuration Reference

### .pre-commit-config.yaml

Location: `/Users/shakes/DevProjects/htmlgraph/.pre-commit-config.yaml`

Key sections:
- **ruff-format** - Code formatting with auto-fix
- **ruff** - Linting with auto-fix
- **mypy** - Type checking (no auto-fix)
- **Standard hooks** - File validation
- **pytest** - Test execution

### pyproject.toml

Location: `/Users/shakes/DevProjects/htmlgraph/pyproject.toml`

Configuration sections:
- **[tool.ruff]** - Ruff settings (line length, ignored rules)
- **[tool.ruff.lint]** - Specific lint rules
- **[tool.mypy]** - Type checking settings
- **[tool.pytest.ini_options]** - Test configuration

## Best Practices

### Before Committing

1. **Run quality checks locally first:**
   ```bash
   uv run ruff check --fix
   uv run ruff format
   uv run mypy src/
   uv run pytest
   ```

2. **Review what changed:**
   ```bash
   git diff
   git status
   ```

3. **Add only what you want:**
   ```bash
   git add file1.py file2.py
   ```

### During Development

1. **Use IDE support:**
   - VS Code: Install Pylance or Pyright
   - PyCharm: Built-in type checking
   - Both highlight mypy errors as you type

2. **Run hooks frequently:**
   ```bash
   # Check current changes
   pre-commit run

   # Check all files
   pre-commit run --all-files
   ```

3. **Fix incrementally:**
   - Don't wait until the end to fix all errors
   - Address types/linting as you code

### In Code Reviews

- **Respect the quality gates** - If hooks fail, the issue is real
- **Don't merge failing checks** - CI will block it anyway
- **Help teammates fix issues** - Share common patterns

## Performance

### Hook Execution Time

Typical execution times:
- **Ruff format**: 1-2 seconds
- **Ruff lint**: 2-3 seconds
- **Mypy**: 5-10 seconds
- **Pytest**: 10-30 seconds (depends on test count)
- **Total**: 20-45 seconds per commit

### Optimization

```bash
# Skip slow hooks for emergency commits (not recommended)
git commit --no-verify -m "Emergency"

# Use incremental mypy checking
uv run mypy src/ --incremental

# Run specific test file instead of all
uv run pytest tests/python/test_specific.py
```

## Integration with CI/CD

The same hooks run in CI/CD pipelines:

```bash
# GitHub Actions will run equivalent checks
pre-commit run --all-files

# If any fail, the build fails
# This prevents bad code from merging to main
```

## Resources

- **Pre-commit documentation:** https://pre-commit.com/
- **Ruff documentation:** https://docs.astral.sh/ruff/
- **Mypy documentation:** https://mypy.readthedocs.io/
- **Pytest documentation:** https://docs.pytest.org/

## Getting Help

**Check hook status:**
```bash
# List all hooks
pre-commit run --dry-run

# Verbose output
pre-commit run --verbose
```

**Debug specific issues:**
```bash
# Run specific hook
pre-commit run mypy --verbose

# Show what files trigger hook
pre-commit run ruff --files "src/python/wipnote/*.py"
```

**Reset and reinstall:**
```bash
# Remove all hooks
pre-commit uninstall

# Clean pre-commit cache
pre-commit clean

# Reinstall everything
pre-commit install
pre-commit run --all-files
```

---
name: code-quality
description: Code hygiene, quality gates, and pre-commit workflows. Use for linting, type checking, testing, and fixing errors.
---

# Code Quality Skill

Use this skill for code hygiene, quality gates, and pre-commit workflows.

**Trigger keywords:** code quality, lint, mypy, ruff, pytest, pre-commit, type checking, clean code, fix errors

---

## Quick Workflow

```bash
# Before EVERY commit:
uv run ruff check --fix     # Lint + autofix
uv run ruff format          # Format code
uv run mypy src/            # Type checking
uv run pytest               # Run tests

# Only commit when ALL checks pass
git commit -m "..."
```

## Research First

**Before implementing anything new:**

- Search PyPI/stdlib for existing libraries before writing custom implementations
- Check `pyproject.toml` for what is already available as a dependency
- Check `src/python/htmlgraph/utils/` for shared utilities before duplicating logic
- Prefer well-maintained packages over one-off custom code

## Philosophy

**CRITICAL: Fix ALL errors with every commit, regardless of when introduced.**

- Errors compound over time
- Pre-existing errors are YOUR responsibility when touching related code
- Clean as you go - leave code better than you found it
- Every commit should reduce technical debt, not accumulate it

## Quality Gates

The deployment script (`deploy-all.sh`) blocks on:
- Mypy type errors
- Ruff lint errors
- Test failures

This is intentional - maintain quality gates.

## Tools Reference

### Ruff (Linting + Formatting)

```bash
# Check for issues
uv run ruff check

# Check and auto-fix
uv run ruff check --fix

# Format code
uv run ruff format

# Check specific files
uv run ruff check src/htmlgraph/models.py
```

### Mypy (Type Checking)

```bash
# Check all source
uv run mypy src/

# Check specific module
uv run mypy src/htmlgraph/sdk.py

# Ignore missing imports
uv run mypy src/ --ignore-missing-imports
```

### Pytest (Testing)

```bash
# Run all tests
uv run pytest

# Verbose output
uv run pytest -v

# Run specific test file
uv run pytest tests/test_sdk.py

# Run specific test
uv run pytest tests/test_sdk.py::test_feature_create
```

## Common Fix Patterns

### Type Errors

```python
# Before (type error)
def get_user(id):
    return db.query(id)

# After (typed)
def get_user(id: str) -> User | None:
    return db.query(id)
```

### Lint Errors

```python
# Before (unused import)
import os
import sys
x = 1

# After (clean)
x = 1
```

### Format Issues

```bash
# Auto-fix all formatting
uv run ruff format
```

## Integration with HtmlGraph

Track quality improvements:

```python
from htmlgraph import SDK
sdk = SDK(agent='code-quality')

spike = sdk.spikes.create('Fix mypy errors in models.py') \
    .set_findings("""
    Fixed 3 type errors:
    - Added return type to get_user()
    - Fixed Optional annotation on config
    - Added type hints to utility functions
    """) \
    .save()
```

---

**Remember:** Fixing errors immediately is faster than letting them accumulate.

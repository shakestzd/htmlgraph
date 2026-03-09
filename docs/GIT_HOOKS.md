# Git Hooks

Documentation for HtmlGraph git hooks that automate development workflows.

## Overview

HtmlGraph uses git hooks to:
- Enforce code quality standards before commits
- Automatically track work in `.htmlgraph/`
- Prevent incomplete deployments
- Maintain clean git history

## Available Hooks

### Pre-commit Hook

**Location:** `.git/hooks/pre-commit`

**Triggers:** Before committing any changes

**Actions:**
1. Run ruff linter: `uv run ruff check --fix`
2. Run ruff formatter: `uv run ruff format`
3. Run mypy type checker: `uv run mypy src/`
4. Run pytest: `uv run pytest`

**Behavior:**
- If any check fails, commit is blocked
- Auto-fixes minor issues (ruff formatting)
- Requires manual fixes for type/test failures
- Can be bypassed with `--no-verify` (not recommended)

### Pre-push Hook

**Location:** `.git/hooks/pre-push`

**Triggers:** Before pushing to remote

**Actions:**
1. Verify version files are synchronized
2. Check that all quality gates pass
3. Confirm deployment readiness

**Behavior:**
- Prevents pushing with broken versions
- Ensures deployed code matches published version
- Can be bypassed with `--no-verify` (use only for emergency fixes)

### Post-merge Hook

**Location:** `.git/hooks/post-merge`

**Triggers:** After merging branches

**Actions:**
1. Update dependencies: `uv sync`
2. Verify compatibility with merged changes
3. Alert if breaking changes detected

## Setup and Installation

### Automatic Installation

Hooks are installed automatically when you first run:

```bash
uv run htmlgraph init
```

### Manual Installation

If hooks aren't installed, set them up manually:

```bash
# Create hooks directory if needed
mkdir -p .git/hooks

# Copy hook scripts
cp packages/claude-plugin/.claude-plugin/hooks/scripts/pre-commit.sh .git/hooks/pre-commit
cp packages/claude-plugin/.claude-plugin/hooks/scripts/pre-push.sh .git/hooks/pre-push

# Make executable
chmod +x .git/hooks/*
```

### Verify Installation

```bash
ls -la .git/hooks/
# Should show:
# -rwxr-xr-x  pre-commit
# -rwxr-xr-x  pre-push
```

## Workflow Examples

### Normal Commit Flow

```bash
# Make changes
vim src/python/htmlgraph/api/services.py

# Stage changes
git add src/python/htmlgraph/api/services.py

# Commit (hooks run automatically)
git commit -m "fix: improve service error handling"

# Output:
# Running pre-commit hook...
# ✓ Linting with ruff
# ✓ Formatting with ruff
# ✓ Type checking with mypy
# ✓ Running tests
# [main abc1234] fix: improve service error handling
```

### Emergency Bypass

When you need to skip hooks (rarely):

```bash
# Bypass ALL hooks
git commit --no-verify -m "emergency: hotfix for production"

# IMPORTANT: Still run checks manually afterward
uv run ruff check --fix
uv run ruff format
uv run mypy src/
uv run pytest
```

### Failed Commit

```bash
# Make changes with type error
vim src/python/htmlgraph/api/services.py
git add .
git commit -m "add feature"

# Output:
# Running pre-commit hook...
# ✗ Type checking with mypy
# Error: Incompatible types in assignment
# [1] mypy error in src/python/htmlgraph/api/services.py:42
#
# Fix errors and try again

# Fix the type error
vim src/python/htmlgraph/api/services.py
git add src/python/htmlgraph/api/services.py
git commit -m "add feature"
# ✓ All hooks passed
# [main xyz7890] add feature
```

## Hook Scripts

### pre-commit.sh

```bash
#!/bin/bash
set -e

echo "Running pre-commit hook..."

# Lint with ruff
echo "Linting with ruff..."
uv run ruff check --fix

# Format with ruff
echo "Formatting with ruff..."
uv run ruff format

# Type check with mypy
echo "Type checking with mypy..."
uv run mypy src/

# Run tests
echo "Running tests..."
uv run pytest

echo "✓ All checks passed"
```

### pre-push.sh

```bash
#!/bin/bash
set -e

echo "Running pre-push hook..."

# Verify versions match
echo "Verifying version synchronization..."
./scripts/verify-versions.sh

# Ensure tests pass
echo "Running final test suite..."
uv run pytest

echo "✓ Ready to push"
```

## Configuration

### Skip Specific Hooks

To skip a specific hook for a single commit:

```bash
# Skip pre-commit hook
SKIP=pre-commit git commit -m "message"

# Skip pre-push hook
SKIP=pre-push git push
```

### Custom Hook Configuration

Edit hook behavior in `.git/hooks/pre-commit`:

```bash
# Example: Skip type checking for documentation-only commits
if [[ $GIT_COMMIT_MSG == docs:* ]]; then
    echo "Documentation commit - skipping type checks"
else
    uv run mypy src/
fi
```

### Disable Hooks Temporarily

Not recommended, but if necessary:

```bash
# Temporarily disable all hooks
chmod -x .git/hooks/*

# Re-enable hooks
chmod +x .git/hooks/*
```

## Troubleshooting

### Issue: Hook Not Running

**Problem:** You commit changes but the hook doesn't execute

**Solution:**
```bash
# 1. Verify hook is executable
ls -la .git/hooks/pre-commit
# Should show: -rwxr-xr-x

# 2. Fix permissions if needed
chmod +x .git/hooks/pre-commit

# 3. Verify hook content
head -5 .git/hooks/pre-commit

# 4. Reinstall hooks
uv run htmlgraph init --force-hooks
```

### Issue: Hook Fails but Changes Are Important

**Problem:** Pre-commit hook fails but you need to commit anyway

**Solution:**
```bash
# Option 1: Fix the issue (recommended)
# Address the failing check first
uv run ruff check --fix
uv run pytest

# Option 2: Bypass for emergency (last resort)
git commit --no-verify -m "emergency: hotfix"
# Then immediately run checks manually
uv run ruff check --fix && uv run ruff format && uv run mypy src/ && uv run pytest
```

### Issue: Hook Is Slow

**Problem:** Hooks take too long, slowing down development

**Solution:**
```bash
# 1. Run tests only for changed files (faster)
# Add to .git/hooks/pre-commit:
uv run pytest --co -q | grep $(git diff --name-only)

# 2. Cache type check results
# mypy has built-in caching - clear if needed:
rm -rf .mypy_cache/

# 3. Profile hook execution
time ./git/hooks/pre-commit
```

## Hook Output Examples

### Successful Commit

```
Running pre-commit hook...
Linting with ruff...
✓ 0 violations found

Formatting with ruff...
✓ Already formatted

Type checking with mypy...
✓ Success: no issues found

Running tests...
test_services.py::test_error_handling PASSED
test_services.py::test_api_response PASSED
=============== 2 passed in 0.45s ===============

✓ All checks passed
[main 3a7b9c1] fix: improve service error handling
 1 file changed, 15 insertions(+), 5 deletions(-)
```

### Failed Commit

```
Running pre-commit hook...
Linting with ruff...
✓ 0 violations found

Formatting with ruff...
✓ Already formatted

Type checking with mypy...
✗ Error: Incompatible types in assignment
  src/python/htmlgraph/api/services.py:42: error: Incompatible types in assignment
    (expression has type "str", variable has type "int")  [assignment]
  Found 1 error in 1 file (checked 15 source files)

✗ Type checking failed - fix errors and try again
```

## Best Practices

1. **Don't skip hooks**
   - They catch real issues before they reach production
   - Take time to understand why they failed
   - Fix the root cause, not the symptom

2. **Run hooks locally before pushing**
   - Faster feedback than CI failures
   - Understand what you're committing
   - Avoid blocking the team

3. **Keep hooks fast**
   - Consider running full tests only on pre-push
   - Use cache mechanisms to speed up checks
   - Profile if hooks slow down development

4. **Customize for your workflow**
   - Add team-specific hooks in `.git/hooks/`
   - Document why each hook exists
   - Review hooks quarterly to ensure they're useful

## See Also

- [Code Quality Standards](./.claude/rules/code-hygiene.md)
- [Deployment Guide](./.claude/rules/deployment.md)
- [Plugin Synchronization](./PLUGIN_SYNC.md)

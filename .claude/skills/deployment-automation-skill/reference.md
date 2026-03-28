# Deployment Automation - Complete Reference

This document provides comprehensive deployment and release workflows for HtmlGraph.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Deployment Script Usage](#deployment-script-usage)
3. [Version Numbering](#version-numbering)
4. [Publishing Checklist](#publishing-checklist)
5. [PyPI Credentials Setup](#pypi-credentials-setup)
6. [Post-Release Procedures](#post-release-procedures)
7. [Dashboard File Synchronization](#dashboard-file-synchronization)
8. [Memory File Synchronization](#memory-file-synchronization)
9. [Common Release Commands](#common-release-commands)
10. [Rollback / Unpublish](#rollback--unpublish)
11. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Streamlined Workflow (v0.9.4+)

```bash
# 1. Run tests
uv run pytest

# 2. Deploy (one command, fully automated!)
./scripts/deploy-all.sh 0.9.4 --no-confirm

# That's it! The script handles:
# ✅ Dashboard file sync (index.html ← dashboard.html)
# ✅ Version updates in all files (Step 0)
# ✅ Auto-commit of version changes
# ✅ Git push with tags
# ✅ Build, publish, install
# ✅ Plugin updates
# ✅ No interactive prompts with --no-confirm
```

### Pre-Deployment Checklist

**IMPORTANT:**
1. ✅ **MUST be in project root directory** - Script will fail if run from subdirectories like `dist/`
2. ~~✅ **Commit all changes first**~~ - **AUTOMATED!** Script auto-commits version changes in Step 0
3. ~~✅ **Verify version numbers**~~ - **AUTOMATED!** Script auto-updates all version numbers in Step 0
4. ✅ **Run tests** - `uv run pytest` must pass before deployment

### Session Tracking Files

**Excluded from git (regenerable):**
```
.gitignore now excludes regenerable session tracking:
- .htmlgraph/sessions/*.jsonl
- .htmlgraph/events/*.jsonl
- .htmlgraph/parent-activity.json

This eliminates the multi-commit cycle problem.
```

---

## Deployment Script Usage

### Using deploy-all.sh (FLEXIBLE OPTIONS)

**CRITICAL: Use `./scripts/deploy-all.sh` for all deployment operations.**

### Quick Usage Examples

```bash
# Full release (non-interactive, recommended)
./scripts/deploy-all.sh 0.9.4 --no-confirm

# Full release (with confirmations)
./scripts/deploy-all.sh 0.9.4

# Documentation changes only (commit + push)
./scripts/deploy-all.sh --docs-only

# Build package only (test builds)
./scripts/deploy-all.sh --build-only

# Skip PyPI publishing (build + install only)
./scripts/deploy-all.sh 0.9.4 --skip-pypi

# Preview what would happen (dry-run)
./scripts/deploy-all.sh --dry-run

# Show all options
./scripts/deploy-all.sh --help
```

### Available Flags

- `--no-confirm` - Skip all confirmation prompts (non-interactive mode) **[NEW]**
- `--docs-only` - Only commit and push to git (skip build/publish)
- `--build-only` - Only build package (skip git/publish/install)
- `--skip-pypi` - Skip PyPI publishing step
- `--skip-plugins` - Skip plugin update steps
- `--dry-run` - Show what would happen without executing

### What the Script Does (9 Steps)

- **Pre-flight: Dashboard Sync** - Sync `src/python/htmlgraph/dashboard.html` → `index.html` **[NEW]**
- **Pre-flight: Code Quality** - Run linters (ruff, mypy) and tests
- **Pre-flight: Plugin Sync** - Verify packages/claude-plugin and .claude are synced
0. **Update & Commit Versions** - Auto-update version numbers in all files and commit
1. **Git Push** - Push commits and tags to origin/main
2. **Build Package** - Create wheel and source distributions
3. **Publish to PyPI** - Upload package to PyPI
4. **Local Install** - Install latest version locally
5. **Update Claude Plugin** - Run `claude plugin update htmlgraph`
6. **Update Gemini Extension** - Update version in gemini-extension.json
7. **Update Codex Skill** - Check for Codex and update if present
8. **Create GitHub Release** - Create release with distribution files

---

## Version Numbering

HtmlGraph follows [Semantic Versioning](https://semver.org/):
- **MAJOR.MINOR.PATCH** (e.g., 0.3.0)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Version Files

**Automatically updated by deploy-all.sh:**
1. `pyproject.toml` - Package version
2. `src/python/htmlgraph/__init__.py` - `__version__` variable
3. `packages/claude-plugin/.claude-plugin/plugin.json` - Claude plugin version
4. `packages/gemini-extension/gemini-extension.json` - Gemini extension version

### Manual Version Updates (Rare)

If you need to update versions manually (script handles this normally):

```bash
VERSION="0.3.0"

# Update pyproject.toml
sed -i '' "s/version = \".*\"/version = \"$VERSION\"/" pyproject.toml

# Update Python __init__.py
sed -i '' "s/__version__ = \".*\"/__version__ = \"$VERSION\"/" src/python/htmlgraph/__init__.py

# Update Claude plugin
sed -i '' "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" packages/claude-plugin/.claude-plugin/plugin.json

# Update Gemini extension
sed -i '' "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" packages/gemini-extension/gemini-extension.json
```

---

## Publishing Checklist

### Pre-Release

- [ ] All tests pass: `uv run pytest`
- [ ] Linters pass: `uv run ruff check --fix && uv run mypy src/`
- [ ] Documentation updated
- [ ] CHANGELOG.md updated (if exists)
- [ ] Changes committed to git (or use --no-confirm for auto-commit)

### Build & Publish

**Automated via deploy-all.sh:**
```bash
./scripts/deploy-all.sh 0.3.0 --no-confirm
```

**Manual workflow (if needed):**
```bash
# 1. Update versions
./scripts/deploy-all.sh 0.3.0 --build-only

# 2. Commit version bump
git add pyproject.toml src/python/htmlgraph/__init__.py \
  packages/claude-plugin/.claude-plugin/plugin.json \
  packages/gemini-extension/gemini-extension.json
git commit -m "chore: bump version to 0.3.0"

# 3. Create git tag
git tag v0.3.0
git push origin main --tags

# 4. Build distributions
uv build
# Creates: dist/htmlgraph-0.3.0-py3-none-any.whl
#          dist/htmlgraph-0.3.0.tar.gz

# 5. Publish to PyPI
source .env  # Load PyPI_API_TOKEN
uv publish dist/htmlgraph-0.3.0* --token "$PyPI_API_TOKEN"

# 6. Verify publication
open https://pypi.org/project/htmlgraph/
```

---

## PyPI Credentials Setup

### Option 1: API Token (Recommended)

1. Create token at: https://pypi.org/manage/account/token/
2. Add to `.env` file:
   ```bash
   PyPI_API_TOKEN=pypi-YOUR_TOKEN_HERE
   ```
3. Use with: `source .env && uv publish dist/* --token "$PyPI_API_TOKEN"`

**Note:** deploy-all.sh automatically sources `.env` if it exists.

### Option 2: Environment Variable

```bash
export UV_PUBLISH_TOKEN="pypi-YOUR_TOKEN_HERE"
uv publish dist/*
```

### Option 3: Command-line Arguments

```bash
uv publish dist/* --username YOUR_USERNAME --password YOUR_PASSWORD
```

---

## Post-Release Procedures

### Update Claude Plugin

```bash
# Users update with:
claude plugin update htmlgraph

# Or fresh install:
claude plugin install htmlgraph@0.3.0
```

### Update Gemini Extension

```bash
# Distribution mechanism TBD
# Users may need to manually update or use extension marketplace
```

### Verify Installation

```bash
# Test PyPI package
pip install htmlgraph==0.3.0
python -c "import htmlgraph; print(htmlgraph.__version__)"

# Check PyPI page
curl -s https://pypi.org/pypi/htmlgraph/json | \
  python -c "import sys, json; print(json.load(sys.stdin)['info']['version'])"
```

### Create GitHub Release

**Automated by deploy-all.sh Step 8:**
- Creates release tag on GitHub
- Uploads distribution files (wheel + source)
- Generates release notes from commit messages

**Manual creation (if needed):**
```bash
gh release create v0.3.0 \
  --title "Release v0.3.0" \
  --notes "See CHANGELOG.md for details" \
  dist/htmlgraph-0.3.0*
```

---

## Dashboard File Synchronization

**AUTOMATIC: Dashboard sync happens automatically during deployment.**

HtmlGraph maintains two versions of the dashboard HTML file:
- **Source of Truth**: `src/python/htmlgraph/dashboard.html` (packaged with Python library)
- **Project Root**: `index.html` (for easy viewing in development)

### Automatic Sync Behavior

- ✅ **During Deployment**: `deploy-all.sh` automatically syncs dashboard files in pre-flight
- ✅ **Auto-Commit**: If changes detected, automatically commits with message "chore: sync index.html with dashboard.html"
- ✅ **Idempotent**: Safe to run multiple times, only commits when out of sync
- ✅ **Dry-Run Support**: `--dry-run` flag shows what would be synced without executing

### Manual Sync (if needed)

```bash
# Sync manually (rare - deployment handles this)
cp src/python/htmlgraph/dashboard.html index.html

# Check if files are in sync
git diff --quiet index.html && echo "In sync" || echo "Out of sync"
```

### Why This Matters

- ✅ Ensures packaged dashboard matches development version
- ✅ Eliminates manual copy-paste errors
- ✅ Prevents deployment with stale dashboard
- ✅ Maintains consistency automatically

---

## Memory File Synchronization

**CRITICAL: Use `# sync-docs not yet in Go CLI` to maintain documentation consistency.**

HtmlGraph uses a centralized documentation pattern:
- **AGENTS.md** - Single source of truth (SDK, API, CLI, workflows)
- **CLAUDE.md** - Platform-specific notes + references AGENTS.md
- **GEMINI.md** - Platform-specific notes + references AGENTS.md

### Quick Usage

```bash
# Check if files are synchronized
# sync-docs not yet in Go CLI

# Generate platform-specific file
# sync-docs not yet in Go CLI
# sync-docs not yet in Go CLI

# Synchronize all files (default)
# sync-docs not yet in Go CLI
```

### Why This Matters

- ✅ Single source of truth in AGENTS.md
- ✅ Platform-specific notes in separate files
- ✅ Easy maintenance (update once, not 3+ times)
- ✅ Consistency across all platforms

---

## Common Release Commands

### Full Release Workflow

```bash
#!/bin/bash
# release.sh - Complete release workflow

VERSION="0.3.0"

# Run deploy-all.sh (handles version updates, commit, tag, build, publish)
./scripts/deploy-all.sh "$VERSION" --no-confirm

echo "✅ Published htmlgraph $VERSION to PyPI"
echo "📦 https://pypi.org/project/htmlgraph/$VERSION/"
```

### Testing a Build Locally

```bash
# Build without publishing
./scripts/deploy-all.sh 0.3.0 --build-only

# Install locally
uv pip install dist/htmlgraph-0.3.0-py3-none-any.whl --force-reinstall

# Test import
python -c "import htmlgraph; print(htmlgraph.__version__)"

# Test CLI
htmlgraph --help
htmlgraph status
```

### Documentation-Only Updates

```bash
# Commit and push documentation changes (no build/publish)
./scripts/deploy-all.sh --docs-only
```

### Build and Test Without Publishing

```bash
# Build package but skip PyPI
./scripts/deploy-all.sh 0.3.0 --skip-pypi

# Test locally installed package
htmlgraph status
python -c "import htmlgraph; print(htmlgraph.__version__)"
```

---

## Rollback / Unpublish

### WARNING: PyPI Does NOT Allow Unpublishing

**⚠️ CRITICAL: PyPI does NOT allow unpublishing or replacing versions.**

Once published, a version is permanent. If you need to fix an issue:

### Option 1: Patch Release (Recommended)

Bump to next patch version (e.g., 0.3.0 → 0.3.1)

```bash
# Fix the issue in code
# Then deploy new version
./scripts/deploy-all.sh 0.3.1 --no-confirm
```

### Option 2: Yank Release

Mark as unavailable (doesn't delete, just warns users):

```bash
# Use twine to yank (uv doesn't support this yet)
pip install twine
twine yank htmlgraph 0.3.0 -r pypi
```

**What yank does:**
- ✅ Prevents new installations of yanked version
- ✅ Shows warning to users
- ❌ Does NOT delete the version
- ❌ Does NOT affect existing installations

### Option 3: Publish Fix

Release corrected version with clear release notes:

```bash
# Deploy fixed version
./scripts/deploy-all.sh 0.3.1 --no-confirm

# Update GitHub release with notes
gh release edit v0.3.1 --notes "Fixes critical bug in v0.3.0 (yanked)"
```

---

## Troubleshooting

### Common Issues

#### 1. Script Fails: "Not in project root"

**Symptom:**
```
Error: Must be run from project root (htmlgraph/)
```

**Solution:**
```bash
# Navigate to project root
cd /Users/shakes/DevProjects/htmlgraph

# Verify you're in correct directory
ls -la scripts/deploy-all.sh  # Should exist

# Run script
./scripts/deploy-all.sh 0.3.0
```

#### 2. Pre-commit Hooks Fail

**Symptom:**
```
Error: Ruff check failed
Error: Mypy type check failed
```

**Solution:**
```bash
# Fix linting errors
uv run ruff check --fix
uv run ruff format

# Fix type errors
uv run mypy src/

# Re-run deployment
./scripts/deploy-all.sh 0.3.0
```

#### 3. Tests Fail

**Symptom:**
```
Error: Tests failed
```

**Solution:**
```bash
# Run tests to see failures
uv run pytest -v

# Fix failing tests
# ...

# Re-run deployment
./scripts/deploy-all.sh 0.3.0
```

#### 4. PyPI Authentication Fails

**Symptom:**
```
Error: Authentication failed
```

**Solution:**
```bash
# Check .env file exists
cat .env  # Should show PyPI_API_TOKEN=...

# Or set environment variable
export UV_PUBLISH_TOKEN="pypi-YOUR_TOKEN_HERE"

# Re-run deployment
./scripts/deploy-all.sh 0.3.0
```

#### 5. Git Push Fails (Conflicts)

**Symptom:**
```
Error: Git push failed (conflict)
```

**Solution:**
```bash
# Pull latest changes
git pull origin main

# Resolve any conflicts
# ...

# Re-run deployment
./scripts/deploy-all.sh 0.3.0
```

#### 6. Dashboard Out of Sync

**Symptom:**
```
Warning: Dashboard files out of sync
```

**Solution:**
```bash
# Let deploy-all.sh handle it (automatic)
./scripts/deploy-all.sh 0.3.0

# Or sync manually
cp src/python/htmlgraph/dashboard.html index.html
git add index.html
git commit -m "chore: sync index.html with dashboard.html"
```

#### 7. Plugin Update Fails

**Symptom:**
```
Error: Claude plugin update failed
```

**Solution:**
```bash
# Update plugin manually
claude plugin update htmlgraph

# Or reinstall
claude plugin uninstall htmlgraph
claude plugin install htmlgraph@0.3.0
```

---

## Version History

Track major releases and their features:

- **0.16.1** (2025-12-31) - Validator blocking for orchestrator mode violations
- **0.16.0** (2025-12-31) - Enhanced orchestrator directives and strict delegation
- **0.9.4** (2025-12-22) - Dashboard sync automation in deploy-all.sh
- **0.3.0** (2025-12-22) - TrackBuilder fluent API, multi-pattern glob support
- **0.2.2** (2025-12-21) - Enhanced session tracking, drift detection
- **0.2.0** (2025-12-21) - Initial public release with SDK
- **0.1.x** - Development versions

---

## See Also

- `scripts/README.md` - Deployment script documentation
- `scripts/deploy-all.sh` - Deployment script source
- `.claude/skills/git-commit-skill/` - Git workflows
- `CLAUDE.md` - Project documentation and workflows

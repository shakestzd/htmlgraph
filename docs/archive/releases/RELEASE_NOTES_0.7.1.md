# Wipnote 0.7.1 Release Notes

**Released**: December 23, 2025
**Type**: Minor Release (Feature Addition + Bug Fixes)
**Package**: https://pypi.org/project/wipnote/0.7.1/

---

## 🎯 What's New

### Memory File Synchronization Tool

A new CLI command to maintain documentation consistency across AI platforms:

```bash
# Check if files are synchronized
wipnote sync-docs --check

# Generate platform-specific file
wipnote sync-docs --generate gemini
wipnote sync-docs --generate claude

# Synchronize all files (default)
wipnote sync-docs
```

**Features:**
- ✅ Validates that platform files (CLAUDE.md, GEMINI.md, etc.) reference central AGENTS.md
- ✅ Supports custom platform templates
- ✅ Generates platform-specific files with consistent structure
- ✅ Single source of truth for AI agent documentation

**Why This Matters:**
- Eliminates documentation drift across platforms
- Makes maintenance easier (update once in AGENTS.md)
- Ensures all AI agents have access to complete documentation

---

## 🚀 Deployment Script Enhancements

The `deploy-all.sh` script now supports flexible deployment modes:

```bash
# Documentation changes only (commit + push)
./scripts/deploy-all.sh --docs-only

# Full release (all 7 steps)
./scripts/deploy-all.sh 0.7.1

# Build package only (test builds)
./scripts/deploy-all.sh --build-only

# Skip PyPI publishing (build + install only)
./scripts/deploy-all.sh 0.7.1 --skip-pypi

# Preview what would happen (dry-run)
./scripts/deploy-all.sh --dry-run

# Skip plugin updates
./scripts/deploy-all.sh 0.7.1 --skip-plugins
```

**New Flags:**
- `--docs-only` - Only commit and push to git (skip build/publish)
- `--build-only` - Only build package (skip git/publish/install)
- `--skip-pypi` - Skip PyPI publishing step
- `--skip-plugins` - Skip plugin update steps
- `--dry-run` - Show what would happen without executing

**7 Deployment Steps:**
1. Git Push - Push commits and tags to origin/main
2. Build Package - Create wheel and source distributions
3. Publish to PyPI - Upload package to PyPI
4. Local Install - Install latest version locally
5. Update Claude Plugin - Run `claude plugin update wipnote`
6. Update Gemini Extension - Update version in gemini-extension.json
7. Update Codex Skill - Check for Codex and update if present

---

## 📚 Documentation Improvements

### Centralized AI Agent Documentation

All AI agent documentation is now centralized in `AGENTS.md`:
- Python SDK quick start and API reference
- Deployment instructions with flexible script usage
- Memory file synchronization workflow
- Best practices for AI agents
- Complete workflow examples

### Platform-Specific Enhancements

**CLAUDE.md:**
- Added deployment script documentation with all flags
- Added memory file sync tool usage
- Added dogfooding context explaining dual purpose
- Clarified general workflows vs project-specific

**GEMINI.md:**
- Added deployment script documentation
- Added memory file sync tool usage
- Updated with imperative language (DO THIS, NEVER, MUST)
- Proper cross-references to AGENTS.md

### Imperative Documentation Style

All platform documentation now uses imperative language:
- ✅ "DO THIS" instead of "you can do this"
- ✅ "NEVER" instead of "you should avoid"
- ✅ "REQUIRED" instead of "it's recommended"
- ✅ Clear commands and actionable instructions

---

## 🐛 Bug Fixes & Improvements

### Removed Non-Functional Git Hooks

**Problem:** Git hooks were causing issues without providing value:
- File locking conflicts (errno 35) preventing event logging
- Pre-commit hook not executable, showing warnings on every commit
- Zero git events actually logged despite 21+ failures
- Duplicating functionality already provided by `git log`

**Solution:** Removed all git hooks:
- Deleted all hook symlinks from `.git/hooks/`
- Deleted all hook scripts from `.wipnote/hooks/`
- Deleted error log file with 21+ failures
- **Result:** No more annoying warnings on every commit!

**Why This Improves Things:**
- ✅ Cleaner commit experience (no warnings)
- ✅ Faster commits (no background hook failures)
- ✅ Simpler codebase (225 lines deleted)
- ✅ Git already provides commit tracking via `git log`

### SDK CRUD Operations

Completed SDK with delete operations:
- `sdk.features.delete(feature_id)` - Delete a feature
- `sdk.bugs.delete(bug_id)` - Delete a bug
- `sdk.chores.delete(chore_id)` - Delete a chore
- Full CRUD support for all collections

---

## 📦 What's Included

### New Files

- `src/python/wipnote/sync_docs.py` - Memory file sync tool
- `scripts/README.md` - Comprehensive script documentation
- `RELEASE_NOTES_0.7.1.md` - This file

### Enhanced Files

- `src/python/wipnote/cli.py` - Added `sync-docs` command
- `scripts/deploy-all.sh` - Added flexible deployment flags
- `AGENTS.md` - Deployment and memory sync documentation
- `CLAUDE.md` - Deployment sections and dogfooding context
- `GEMINI.md` - Deployment sections and imperative language

### Removed Files

- `.wipnote/hooks/` - All git hook scripts (non-functional)
- `.git/hooks/pre-commit` - Broken hook causing warnings
- `.wipnote/git-hook-errors.log` - Error log

---

## 📊 Statistics

**Changes Since 0.7.0:**
- **12 commits** included in this release
- **+729 insertions, -293 deletions** (net +436 lines)
- **3 new features** (sync-docs, deployment flags, CRUD deletes)
- **225 lines removed** (git hooks cleanup)
- **4 major documentation files** updated

**Package Size:**
- Wheel: 222 KB (`wipnote-0.7.1-py3-none-any.whl`)
- Source: 240 KB (`wipnote-0.7.1.tar.gz`)

---

## 🔄 Migration Guide

### Upgrading from 0.7.0

```bash
# Update via pip
pip install --upgrade wipnote

# Or install specific version
pip install wipnote==0.7.1

# Update Claude plugin
claude plugin update wipnote
```

**Breaking Changes:** None

**New Features You Can Use:**
1. Run `wipnote sync-docs --check` to validate your documentation
2. Use `./scripts/deploy-all.sh --docs-only` for doc-only deploys
3. Use SDK delete methods: `sdk.features.delete(id)`

**What You'll Notice:**
- No more git hook warnings on commits
- Cleaner, faster commit experience
- New `wipnote sync-docs` command available

---

## 🎓 Examples

### Memory File Sync Workflow

```bash
# 1. Check current state
wipnote sync-docs --check

# Output:
# ✅ AGENTS.md exists
# ✅ root:GEMINI.md references AGENTS.md
# ✅ root:CLAUDE.md references AGENTS.md
# ✅ All files are properly synchronized!

# 2. Generate a new platform file
wipnote sync-docs --generate codex

# 3. Force overwrite existing file
wipnote sync-docs --generate gemini --force
```

### Flexible Deployment

```bash
# Quick doc update workflow
./scripts/deploy-all.sh --docs-only

# Test build without publishing
./scripts/deploy-all.sh --build-only

# Preview release without execution
./scripts/deploy-all.sh 0.7.2 --dry-run

# Full release
./scripts/deploy-all.sh 0.7.2
```

### SDK Delete Operations

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Delete a completed feature
sdk.features.delete("feature-old-123")

# Delete multiple bugs
for bug_id in ["bug-001", "bug-002", "bug-003"]:
    sdk.bugs.delete(bug_id)

# Delete a chore
sdk.chores.delete("chore-cleanup")
```

---

## 🔗 Links

- **PyPI Package**: https://pypi.org/project/wipnote/0.7.1/
- **GitHub Repository**: https://github.com/shakestzd/wipnote
- **Documentation**: See `AGENTS.md` in the repository
- **Previous Release**: [RELEASE_NOTES_0.7.0.md](./RELEASE_NOTES_0.7.0.md)

---

## 🙏 Acknowledgments

This release focused on improving developer experience:
- Removing friction (git hook warnings)
- Adding flexibility (deployment script flags)
- Improving consistency (memory file sync)
- Completing features (SDK CRUD operations)

Thanks to all users who reported issues with git hooks!

---

## 🛣️ What's Next

**Upcoming in 0.7.2+:**
- Package deployment script pattern for all users
- Improve `wipnote init` with better defaults
- Enhanced drift detection algorithm
- Additional platform support (Codex skill packaging)

---

**Released**: December 23, 2025
**Version**: 0.7.1
**License**: MIT

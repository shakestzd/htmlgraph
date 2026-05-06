# Deployment Automation Skill

Use this skill for deployment, versioning, and release workflows in the Wipnote project.

**Trigger keywords:** deployment, release, publish, version, pypi, package, build, distribute

---

## Quick Reference

### Fast Deployment (Recommended)

```bash
# Full release (non-interactive)
./scripts/deploy-all.sh 0.9.4 --no-confirm

# Full release (with confirmations)
./scripts/deploy-all.sh 0.9.4

# Documentation changes only
./scripts/deploy-all.sh --docs-only

# Build package only (test builds)
./scripts/deploy-all.sh --build-only

# Preview changes (dry-run)
./scripts/deploy-all.sh --dry-run
```

### Pre-Deployment Checklist

1. ✅ **MUST be in project root** - Script will fail from subdirectories
2. ✅ **Run tests** - `uv run pytest` must pass
3. ~~Version updates~~ - **AUTOMATED** by script in Step 0
4. ~~Git commits~~ - **AUTOMATED** by script in Step 0

### Version Numbering

Wipnote follows [Semantic Versioning](https://semver.org/):
- **MAJOR.MINOR.PATCH** (e.g., 0.9.4)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### What deploy-all.sh Does

**9 Steps (fully automated):**
- **Pre-flight: Dashboard Sync** - Sync dashboard.html → index.html
- **Pre-flight: Code Quality** - Run linters (ruff, mypy) and tests
- **Pre-flight: Plugin Sync** - Verify packages/claude-plugin and .claude are synced
0. **Update & Commit Versions** - Auto-update version numbers and commit
1. **Git Push** - Push commits and tags to origin/main
2. **Build Package** - Create wheel and source distributions
3. **Publish to PyPI** - Upload package to PyPI
4. **Local Install** - Install latest version locally
5. **Update Claude Plugin** - Run `claude plugin update wipnote`
6. **Update Gemini Extension** - Update version in gemini-extension.json
7. **Update Codex Skill** - Check for Codex and update if present
8. **Create GitHub Release** - Create release with distribution files

### Available Flags

- `--no-confirm` - Skip all confirmation prompts (non-interactive)
- `--docs-only` - Only commit and push to git (skip build/publish)
- `--build-only` - Only build package (skip git/publish/install)
- `--skip-pypi` - Skip PyPI publishing step
- `--skip-plugins` - Skip plugin update steps
- `--dry-run` - Show what would happen without executing

### Common Workflows

**Testing a build locally:**
```bash
# Build without publishing
./scripts/deploy-all.sh 0.9.4 --build-only

# Install and test locally
uv pip install dist/wipnote-0.9.4-py3-none-any.whl --force-reinstall
python -c "import wipnote; print(wipnote.__version__)"
```

**Documentation-only updates:**
```bash
# Commit and push docs changes (no build/publish)
./scripts/deploy-all.sh --docs-only
```

**Full release workflow:**
```bash
# 1. Run tests
uv run pytest

# 2. Deploy (one command!)
./scripts/deploy-all.sh 0.9.4 --no-confirm

# 3. Verify publication
open https://pypi.org/project/wipnote/
```

### PyPI Credentials

**Option 1: API Token (Recommended)**
```bash
# Add to .env file:
PyPI_API_TOKEN=pypi-YOUR_TOKEN_HERE

# Script will source .env automatically
```

**Option 2: Environment Variable**
```bash
export UV_PUBLISH_TOKEN="pypi-YOUR_TOKEN_HERE"
```

### Version History

- **0.16.1** (2025-12-31) - Validator blocking for orchestrator mode
- **0.16.0** (2025-12-31) - Enhanced orchestrator directives
- **0.9.4** (2025-12-22) - Dashboard sync automation
- **0.3.0** (2025-12-22) - TrackBuilder fluent API
- **0.2.2** (2025-12-21) - Enhanced session tracking
- **0.2.0** (2025-12-21) - Initial public release

---

## When to Use This Skill

Activate this skill when:
- Preparing to release a new version
- Publishing to PyPI
- Updating version numbers
- Building distributions
- Testing deployment locally
- Updating Claude plugin or Gemini extension
- Need deployment troubleshooting

---

## Detailed Documentation

For complete deployment workflows, troubleshooting, and advanced usage:
→ See [reference.md](./reference.md)

For deployment script internals and options:
→ See `scripts/README.md` in project root

---

## Integration with Wipnote SDK

Track deployment activities:

```python
from wipnote import SDK
sdk = SDK(agent='deployment-automation')

# Track deployment
spike = sdk.spikes.create('Deploy v0.9.4 to PyPI') \
    .add_finding('Ran deploy-all.sh with --no-confirm') \
    .add_finding('All pre-flight checks passed') \
    .add_finding('Published to PyPI successfully') \
    .add_finding('Updated Claude plugin to v0.9.4') \
    .save()
```

---

**See also:**
- `.claude/skills/git-commit-skill/` - Git workflows
- `scripts/deploy-all.sh` - Deployment script
- `scripts/README.md` - Script documentation

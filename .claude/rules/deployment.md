# Deployment & Release Rules

**CRITICAL: Use `./scripts/deploy-all.sh` for all deployment operations.**

## Using the Deployment Script

**IMPORTANT PRE-DEPLOYMENT CHECKLIST:**
1. **MUST be in project root directory** - Script will fail if run from subdirectories
2. ~~**Commit all changes first**~~ - **AUTOMATED!** Script auto-commits version changes in Step 0
3. ~~**Verify version numbers**~~ - **AUTOMATED!** Script auto-updates all version numbers in Step 0
4. **Run tests** - `(cd packages/go && go test ./...)` must pass before deployment

**Streamlined Workflow:**
```bash
# 1. Run tests
(cd packages/go && go build ./... && go vet ./... && go test ./...)

# 2. Deploy (one command, fully automated!)
./scripts/deploy-all.sh 1.0.0 --no-confirm

# The script handles:
# Version updates in all files (Step 0)
# Auto-commit of version changes
# Git push with tags
# Go binary build for all platforms
# GitHub Release with binaries
# Plugin updates
# No interactive prompts with --no-confirm
```

**Quick Usage:**
```bash
# Full release (non-interactive, recommended)
./scripts/deploy-all.sh 1.0.0 --no-confirm

# Full release (with confirmations)
./scripts/deploy-all.sh 1.0.0

# Documentation changes only (commit + push)
./scripts/deploy-all.sh --docs-only

# Build binary only (test builds)
./scripts/deploy-all.sh --build-only

# Preview what would happen (dry-run)
./scripts/deploy-all.sh --dry-run

# Show all options
./scripts/deploy-all.sh --help
```

**Available Flags:**
- `--no-confirm` - Skip all confirmation prompts (non-interactive mode)
- `--docs-only` - Only commit and push to git (skip build/publish)
- `--build-only` - Only build binary (skip git/publish)
- `--skip-plugins` - Skip plugin update steps
- `--dry-run` - Show what would happen without executing

**What the Script Does:**
- **Pre-flight: Code Quality** - Run `go build`, `go vet`, `go test`
- **Pre-flight: Plugin Sync** - Verify packages/go-plugin and .claude are synced
0. **Update & Commit Versions** - Auto-update version numbers in all files and commit
1. **Git Push** - Push commits and tags to origin/main
2. **Build Binary** - Cross-compile Go binary for darwin/linux (amd64/arm64)
3. **Create GitHub Release** - Upload binaries to GitHub Releases
4. **Update Claude Plugin** - Run `claude plugin update htmlgraph`
5. **Update Gemini Extension** - Update version in gemini-extension.json

**See:** `scripts/README.md` for complete documentation

## Version Numbering

HtmlGraph follows [Semantic Versioning](https://semver.org/):
- **MAJOR.MINOR.PATCH** (e.g., 1.0.0)
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

**Version Files to Update:**
1. `packages/go-plugin/.claude-plugin/plugin.json` - Plugin version
2. `packages/gemini-extension/gemini-extension.json` - Gemini extension version

## Publishing Checklist

**Pre-Release:**
- [ ] All tests pass: `(cd packages/go && go test ./...)`
- [ ] Documentation updated
- [ ] Version bumped in all files
- [ ] Changes committed to git
- [ ] Create git tag: `git tag v1.0.0`

## Post-Release

**Update Claude Plugin:**
```bash
# Users update with:
claude plugin update htmlgraph

# Or fresh install:
claude plugin install htmlgraph
```

**Verify Installation:**
```bash
# Check binary version
htmlgraph --version

# Check plugin version
grep '"version"' packages/go-plugin/.claude-plugin/plugin.json
```

## Rollback

If a release has issues:

1. **Patch Release:** Bump to next patch version (e.g., 1.0.0 -> 1.0.1)
2. **Delete GitHub Release:** `gh release delete v1.0.0` (if caught quickly)
3. **Publish Fix:** Release corrected version

## Memory File Synchronization

**CRITICAL: Use `htmlgraph sync-docs` to maintain documentation consistency.**

HtmlGraph uses a centralized documentation pattern:
- **AGENTS.md** - Single source of truth (SDK, API, CLI, workflows)
- **CLAUDE.md** - Platform-specific notes + references AGENTS.md
- **GEMINI.md** - Platform-specific notes + references AGENTS.md

**Quick Usage:**
```bash
htmlgraph sync-docs --check   # Check sync status
htmlgraph sync-docs           # Synchronize all files
```

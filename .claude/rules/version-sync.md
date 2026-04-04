---
paths:
  - "scripts/**"
  - "plugin/.claude-plugin/**"
---

# Version Synchronization Rules

**CRITICAL: Plugin version and GitHub Release version MUST always match.**

## Why This Matters

The Go binary is distributed via GitHub Releases. The plugin bootstrap script downloads the binary version matching `plugin.json`. If versions diverge, users get the wrong binary.

## Version Files (Must All Match)

```
plugin/.claude-plugin/plugin.json:                "version": "X.Y.Z"
```

## For Claude: Always Check Version

```bash
# Check local plugin version
grep '"version"' plugin/.claude-plugin/plugin.json

# Check latest GitHub Release
gh release view --json tagName -q .tagName 2>/dev/null || echo "No releases"
```

## Publishing Requirements

**Before running `./scripts/deploy-all.sh VERSION`:**

1. Verify all version files match target version
2. Run: `./scripts/verify-versions.sh X.Y.Z`
3. Confirm all files were updated

**The publishing script will:**
- FAIL if versions are inconsistent
- Automatically sync all version files

## Version Update Workflow

When deploying new version:

```bash
# 1. Update version files
# 2. Run verification
./scripts/verify-versions.sh 1.0.0

# 3. Deploy
./scripts/deploy-all.sh 1.0.0 --no-confirm
```

## For AI Agents

**DIRECTIVE**: When discussing version numbers:
1. Check plugin.json for the current version
2. Verify version files are consistent
3. Recommend running `./scripts/verify-versions.sh` before deployment

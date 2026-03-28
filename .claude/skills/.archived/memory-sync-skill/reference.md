# Memory Sync - Complete Reference

Complete documentation for HtmlGraph's centralized documentation synchronization system.

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Command Reference](#command-reference)
4. [Synchronization Patterns](#synchronization-patterns)
5. [Implementation Details](#implementation-details)
6. [Workflows](#workflows)
7. [Troubleshooting](#troubleshooting)

---

## Overview

### The Problem

Multi-platform AI tools (Claude Code, Gemini, etc.) need documentation in multiple files:
- AGENTS.md - Complete SDK/API docs
- CLAUDE.md - Claude-specific integration
- GEMINI.md - Gemini-specific integration

**Without sync:**
- ❌ Update SDK example in AGENTS.md
- ❌ Forget to update CLAUDE.md
- ❌ GEMINI.md has stale example
- ❌ Users see inconsistent docs

**With sync:**
- ✅ Update SDK example in AGENTS.md
- ✅ Run `sync-docs`
- ✅ All files reference latest version
- ✅ Consistency guaranteed

### The Solution

**Centralized Documentation Pattern:**
1. **AGENTS.md** = Single source of truth (SDK, API, CLI)
2. **CLAUDE.md** = Platform-specific + references AGENTS.md
3. **GEMINI.md** = Platform-specific + references AGENTS.md
4. **sync-docs** = Verification and generation tool

### Key Principles

1. **Single Source of Truth** - SDK/API docs live in AGENTS.md only
2. **Reference, Don't Duplicate** - Platform files link to AGENTS.md
3. **Platform Flexibility** - Platform files can add specific notes
4. **Automated Verification** - sync-docs checks consistency
5. **Version Control Friendly** - Git tracks master document changes

---

## Architecture

### File Hierarchy

```
htmlgraph/
├── AGENTS.md              # Master documentation (SDK, API, CLI)
│   ├── Complete Python SDK reference
│   ├── CLI command examples
│   ├── Deployment instructions
│   ├── Best practices
│   └── Code examples
│
├── CLAUDE.md              # Claude Code-specific
│   ├── Project vision & architecture
│   ├── Orchestrator directives
│   ├── Code hygiene rules
│   ├── Debugging workflows
│   └── → References AGENTS.md for SDK/API
│
└── GEMINI.md              # Gemini-specific
    ├── Google AI Studio integration
    ├── Gemini-specific workflows
    ├── Extension setup
    └── → References AGENTS.md for SDK/API
```

### Content Distribution

**AGENTS.md (Master):**
- ✅ Python SDK (complete API reference)
- ✅ CLI commands (htmlgraph status, features, tracks, etc.)
- ✅ Deployment guide (PyPI publishing, versioning)
- ✅ Best practices (dogfooding, quality gates)
- ✅ Code examples (SDK usage, tracking patterns)

**CLAUDE.md (Platform-Specific):**
- ✅ Project vision ("HTML is All You Need")
- ✅ Orchestrator directives (delegation rules)
- ✅ Code hygiene (fix all errors, research-first debugging)
- ✅ Git workflows (commit patterns)
- ❌ SDK reference (links to AGENTS.md instead)

**GEMINI.md (Platform-Specific):**
- ✅ Gemini extension setup
- ✅ Google AI Studio integration
- ✅ Gemini-specific features
- ❌ SDK reference (links to AGENTS.md instead)

### Reference Pattern

**In platform-specific files:**

```markdown
## SDK Usage

For complete SDK documentation, see **[AGENTS.md](./AGENTS.md)**.

Quick example:
```python
from htmlgraph import SDK
sdk = SDK(agent='claude')
feature = sdk.features.create('Add authentication').save()
```

For more: deployment, CLI, best practices → see AGENTS.md
```

---

## Command Reference

### `# sync-docs not yet in Go CLI`

Main command for documentation synchronization.

#### Usage

```bash
# Check synchronization status
# sync-docs not yet in Go CLI

# Synchronize all files (default)
# sync-docs not yet in Go CLI

# Generate specific platform file
# sync-docs not yet in Go CLI
# sync-docs not yet in Go CLI

# Verbose output
# sync-docs not yet in Go CLI --verbose
```

#### Options

| Option | Description |
|--------|-------------|
| `--check` | Check if files are synchronized (no changes) |
| `--generate <platform>` | Generate specific platform file |
| `--verbose` | Show detailed sync operations |
| `--help` | Show help message |

#### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (all files synchronized) |
| 1 | Out of sync (changes needed) |
| 2 | Error (file not found, permission denied) |

#### Examples

**Check before commit:**
```bash
# sync-docs not yet in Go CLI
if [ $? -eq 0 ]; then
    echo "✅ Documentation synchronized"
    git commit -m "docs: update SDK examples"
else
    echo "⚠️  Documentation out of sync"
    exit 1
fi
```

**Auto-sync in pre-commit hook:**
```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Checking documentation sync..."
# sync-docs not yet in Go CLI || {
    echo "Auto-syncing documentation..."
    # sync-docs not yet in Go CLI
    git add AGENTS.md CLAUDE.md GEMINI.md
}
```

**Generate platform file:**
```bash
# Regenerate GEMINI.md from AGENTS.md
# sync-docs not yet in Go CLI

# Review changes
git diff GEMINI.md
```

---

## Synchronization Patterns

### Pattern 1: Reference with Brief Example

**Best for:** SDK methods, CLI commands, common workflows

**Template:**
```markdown
## [Feature Name]

For complete documentation, see **[AGENTS.md](./AGENTS.md#section)**.

Quick example:
[brief code snippet]

For more details: [specific topics] → see AGENTS.md
```

**Example:**
```markdown
## SDK Features API

For complete feature management documentation, see **[AGENTS.md](./AGENTS.md#features-api)**.

Quick example:
```python
sdk = SDK(agent='claude')
feature = sdk.features.create('Add authentication') \
    .set_priority('high') \
    .add_steps(['Setup OAuth', 'Add middleware']) \
    .save()
```

For more: tracks, analytics, delegation → see AGENTS.md
```

### Pattern 2: Platform-Specific Extension

**Best for:** Platform-unique features, integration notes

**Template:**
```markdown
## [Platform Feature]

[Platform-specific content here]

### Integration with SDK

See **[AGENTS.md](./AGENTS.md)** for SDK usage patterns.

[Platform-specific example using SDK]
```

**Example (CLAUDE.md):**
```markdown
## Orchestrator Delegation

Claude Code orchestrator mode requires strict delegation patterns.

**Rule:** ALWAYS delegate git operations to subagents.

```python
# Use Task tool for delegation
Task(
    prompt="Commit and push changes...",
    subagent_type="general-purpose"
)
```

### Track with SDK

See **[AGENTS.md](./AGENTS.md#sdk-orchestration)** for orchestration patterns.

```python
sdk = SDK(agent='orchestrator')
explorer = sdk.spawn_explorer(task="Find auth code")
Task(prompt=explorer["prompt"])
```
```

### Pattern 3: Complete Separation

**Best for:** Architecture, vision, project-specific docs

**Template:**
```markdown
## [Project-Specific Topic]

[Complete content - no sync needed]

[No reference to AGENTS.md]
```

**Example (CLAUDE.md):**
```markdown
## Project Vision

HtmlGraph: "HTML is All You Need"

Core Philosophy: The web is already a giant graph database.

[Full vision content with no SDK references]
```

---

## Implementation Details

### How sync-docs Works

**1. Content Extraction:**
```python
def extract_sdk_sections(agents_md_path):
    """Extract SDK-related sections from AGENTS.md"""
    sections = [
        'SDK Usage',
        'CLI Commands',
        'Deployment',
        'Best Practices'
    ]
    return parse_markdown_sections(agents_md_path, sections)
```

**2. Reference Injection:**
```python
def inject_references(platform_file, sections):
    """Replace duplicated content with references"""
    for section in sections:
        replace_with_reference(
            platform_file,
            section,
            template=REFERENCE_TEMPLATE
        )
```

**3. Verification:**
```python
def check_sync(agents_md, platform_files):
    """Verify no duplication, only references"""
    for platform_file in platform_files:
        if has_duplicated_content(platform_file, agents_md):
            return False
    return True
```

### Configuration

**Future:** Add `.sync-docs.yaml` config:

```yaml
# .sync-docs.yaml
source: AGENTS.md
targets:
  - file: CLAUDE.md
    sections:
      - SDK Usage
      - CLI Commands
    preserve:
      - Project Vision
      - Orchestrator Directives

  - file: GEMINI.md
    sections:
      - SDK Usage
      - CLI Commands
    preserve:
      - Gemini Integration
```

### File Detection

**Auto-detect memory files:**
```python
def find_memory_files(project_root):
    """Find all documentation files"""
    return {
        'master': project_root / 'AGENTS.md',
        'claude': project_root / 'CLAUDE.md',
        'gemini': project_root / 'GEMINI.md',
        'codex': project_root / 'CODEX.md'  # if exists
    }
```

---

## Workflows

### Workflow 1: Add New SDK Feature

**Scenario:** Implemented new SDK method, need to document.

```bash
# 1. Add documentation to AGENTS.md (master)
vim AGENTS.md

# Add section:
# ### SDK.features.bulk_update()
# Update multiple features in one call...

# 2. Sync to platform files
# sync-docs not yet in Go CLI

# 3. Verify synchronization
# sync-docs not yet in Go CLI

# 4. Commit
git add AGENTS.md CLAUDE.md GEMINI.md
git commit -m "docs: add bulk_update SDK method"
git push
```

### Workflow 2: Add Claude-Specific Pattern

**Scenario:** New orchestrator pattern for Claude only.

```bash
# 1. Edit CLAUDE.md directly
vim CLAUDE.md

# Add section:
# ## Parallel Task Coordination
# [Claude-specific delegation pattern]

# 2. No sync needed (platform-specific)

# 3. Commit
git add CLAUDE.md
git commit -m "docs: add parallel task coordination pattern"
git push
```

### Workflow 3: Fix Stale Example

**Scenario:** Example in AGENTS.md is outdated.

```bash
# 1. Update AGENTS.md (source of truth)
vim AGENTS.md
# Fix example code

# 2. Sync propagates to all platform files
# sync-docs not yet in Go CLI

# 3. All files now reference updated example
git diff CLAUDE.md GEMINI.md  # No changes (they reference AGENTS.md)

# 4. Commit
git add AGENTS.md
git commit -m "docs: fix SDK initialization example"
```

### Workflow 4: Pre-Commit Check

**Scenario:** Automated sync check before every commit.

```bash
# .git/hooks/pre-commit
#!/bin/bash

echo "Checking documentation sync..."
# sync-docs not yet in Go CLI

if [ $? -ne 0 ]; then
    echo ""
    echo "⚠️  Documentation out of sync!"
    echo "Run: # sync-docs not yet in Go CLI"
    echo ""
    exit 1
fi
```

### Workflow 5: Deploy-Time Verification

**Scenario:** Verify docs before publishing package.

```bash
# In deploy-all.sh
step "Verify Documentation Sync" || {
    # sync-docs not yet in Go CLI || {
        echo "Documentation out of sync. Auto-syncing..."
        # sync-docs not yet in Go CLI
        git add AGENTS.md CLAUDE.md GEMINI.md
        git commit -m "chore: sync documentation"
    }
}
```

---

## Troubleshooting

### Issue 1: "Out of Sync" Warning

**Symptom:**
```bash
$ # sync-docs not yet in Go CLI
⚠️  CLAUDE.md out of sync with AGENTS.md
```

**Cause:** CLAUDE.md has duplicated SDK content instead of reference.

**Fix:**
```bash
# Auto-fix with sync
# sync-docs not yet in Go CLI

# Or manually replace with reference:
vim CLAUDE.md
# Replace duplicated section with:
# See **[AGENTS.md](./AGENTS.md#section)** for details.
```

### Issue 2: Lost Platform-Specific Content

**Symptom:** Sync deleted Claude-specific notes.

**Cause:** Sync incorrectly identified platform content as duplication.

**Fix:**
```bash
# Restore from git
git checkout CLAUDE.md

# Mark section as platform-specific (add to .sync-docs.yaml)
# OR move content to different section
```

### Issue 3: Broken References

**Symptom:** Links to AGENTS.md don't work.

**Cause:** Incorrect anchor in reference link.

**Fix:**
```bash
# Check AGENTS.md section headers
grep -n "^##" AGENTS.md

# Update reference with correct anchor
vim CLAUDE.md
# Change: [AGENTS.md](./AGENTS.md#wrong-anchor)
# To:     [AGENTS.md](./AGENTS.md#correct-anchor)
```

### Issue 4: Sync Command Not Found

**Symptom:**
```bash
$ # sync-docs not yet in Go CLI
Error: Command 'sync-docs' not found
```

**Cause:** Old version of htmlgraph package.

**Fix:**
```bash
# Update to latest version
uv pip install --upgrade htmlgraph

# Verify version
htmlgraph --version
```

---

## Advanced Usage

### Custom Sync Rules

**Define sections to sync:**

```python
# scripts/custom-sync.py
from htmlgraph.sync import DocumentSync

sync = DocumentSync(
    master='AGENTS.md',
    targets={
        'CLAUDE.md': {
            'sync_sections': ['SDK Usage', 'CLI Commands'],
            'preserve_sections': ['Orchestrator', 'Vision']
        },
        'GEMINI.md': {
            'sync_sections': ['SDK Usage', 'CLI Commands'],
            'preserve_sections': ['Gemini Integration']
        }
    }
)

sync.run()
```

### Partial Sync

**Sync only specific sections:**

```bash
# Future feature
# sync-docs not yet in Go CLI --section "SDK Usage"
# sync-docs not yet in Go CLI --section "CLI Commands"
```

### Dry-Run Mode

**Preview changes without applying:**

```bash
# Future feature
# sync-docs not yet in Go CLI --dry-run

# Output:
# Would update CLAUDE.md:
#   - Replace lines 45-67 with reference to AGENTS.md#sdk-usage
# Would update GEMINI.md:
#   - Replace lines 23-34 with reference to AGENTS.md#cli-commands
```

---

## See Also

- **[SKILL.md](./SKILL.md)** - Quick reference and common patterns
- **[AGENTS.md](../../AGENTS.md)** - Master SDK/API documentation
- **[CLAUDE.md](../../CLAUDE.md)** - Claude-specific project docs
- **[scripts/README.md](../../scripts/README.md)** - Deployment scripts

---

*Centralized documentation for consistent multi-platform AI tools*

# Memory Sync Skill

Use this skill for documentation synchronization and memory file management.

**Trigger keywords:** sync, documentation, memory, AGENTS.md, CLAUDE.md, GEMINI.md, sync-docs

---

## Quick Reference

### Sync Documentation Files

```bash
# Check if files are synchronized
# sync-docs not yet in Go CLI

# Synchronize all files (default)
# sync-docs not yet in Go CLI

# Generate platform-specific file
# sync-docs not yet in Go CLI
# sync-docs not yet in Go CLI
```

### Memory File Hierarchy

**Single Source of Truth:**
- **AGENTS.md** - Primary documentation (SDK, API, CLI, workflows, examples)
  - Contains all complete documentation
  - Platform-agnostic
  - Updated first when making changes

**Platform-Specific Files:**
- **CLAUDE.md** - Claude Code-specific notes + references AGENTS.md
  - Project vision, architecture, roadmap
  - Orchestrator directives
  - Code hygiene rules
  - Links to AGENTS.md for SDK/API details

- **GEMINI.md** - Gemini-specific notes + references AGENTS.md
  - Google AI Studio integration
  - Gemini-specific workflows
  - Links to AGENTS.md for SDK/API details

### When to Sync vs Edit Directly

**Use `sync-docs` when:**
- ✅ You updated AGENTS.md and need to propagate changes
- ✅ You want to verify all files are synchronized
- ✅ Before committing documentation changes
- ✅ After pulling updates from git

**Edit directly when:**
- ✅ Adding platform-specific notes to CLAUDE.md or GEMINI.md
- ✅ Updating project-specific sections (vision, roadmap, directives)
- ✅ Making changes that don't affect SDK/API documentation

### Documentation Patterns

#### 1. Reference Pattern (Recommended)

**In platform-specific files:**
```markdown
## SDK Usage

For complete SDK documentation, see:
- **[AGENTS.md](./AGENTS.md)** - Python SDK, API, CLI, deployment, best practices

Quick example:
[brief example here]
```

**Why?** Keeps platform files lean, single source of truth in AGENTS.md

#### 2. Duplication Pattern (Legacy)

**Avoid unless necessary:**
```markdown
## SDK Usage

[Complete SDK documentation duplicated here]
```

**Why avoid?** Becomes out of sync, maintenance overhead

### Common Workflows

#### Workflow 1: Update SDK Documentation

```bash
# 1. Edit AGENTS.md (single source of truth)
vim AGENTS.md

# 2. Sync to platform files
# sync-docs not yet in Go CLI

# 3. Verify sync
# sync-docs not yet in Go CLI

# 4. Commit
git add AGENTS.md CLAUDE.md GEMINI.md
git commit -m "docs: update SDK examples"
```

#### Workflow 2: Add Platform-Specific Notes

```bash
# 1. Edit platform file directly
vim CLAUDE.md

# 2. No sync needed (platform-specific content)

# 3. Commit
git add CLAUDE.md
git commit -m "docs: add Claude orchestrator patterns"
```

#### Workflow 3: Check Synchronization

```bash
# Check if files are in sync
# sync-docs not yet in Go CLI

# Output examples:
# ✅ All files synchronized
# ⚠️  CLAUDE.md out of sync with AGENTS.md
```

#### Workflow 4: Generate Platform File

```bash
# Generate Gemini-specific file from AGENTS.md
# sync-docs not yet in Go CLI

# Generate Claude-specific file from AGENTS.md
# sync-docs not yet in Go CLI
```

### File Structure

```
htmlgraph/
├── AGENTS.md              # Single source of truth
├── CLAUDE.md              # Platform-specific + refs to AGENTS.md
├── GEMINI.md              # Platform-specific + refs to AGENTS.md
└── .claude/
    └── skills/
        └── memory-sync-skill/
            ├── SKILL.md         # This file
            └── reference.md     # Detailed patterns
```

### Integration with Development

**Pre-commit Check:**
```bash
# Add to pre-commit hook
# sync-docs not yet in Go CLI || {
    echo "⚠️  Documentation out of sync. Run: # sync-docs not yet in Go CLI"
    exit 1
}
```

**Pre-deployment Check:**
```bash
# In deploy-all.sh
echo "Checking documentation sync..."
# sync-docs not yet in Go CLI
if [ $? -ne 0 ]; then
    echo "Auto-syncing documentation..."
    # sync-docs not yet in Go CLI
fi
```

### Why This Matters

**Benefits:**
- ✅ **Single Source of Truth** - Update once in AGENTS.md, not 3+ times
- ✅ **Consistency** - All platforms reference same SDK/API docs
- ✅ **Easy Maintenance** - Change in one place, propagate everywhere
- ✅ **Platform Flexibility** - Add platform-specific notes without duplication
- ✅ **Version Control** - Git tracks changes to master document

**Problems Solved:**
- ❌ Inconsistent documentation across platforms
- ❌ Stale examples in one file but not others
- ❌ Manual copy-paste errors
- ❌ Forgot to update GEMINI.md after updating CLAUDE.md

### Quick Decision Tree

```
Need to update docs?
├─ SDK/API/CLI changes?
│  ├─ YES → Edit AGENTS.md → Run sync-docs
│  └─ NO → Continue
│
├─ Platform-specific changes?
│  ├─ Claude orchestrator → Edit CLAUDE.md directly
│  ├─ Gemini integration → Edit GEMINI.md directly
│  └─ Architecture/vision → Edit CLAUDE.md directly
│
└─ Unsure?
   └─ Check AGENTS.md first → If there, edit + sync
      └─ If not there, edit platform file directly
```

### Examples

#### Example 1: Add New SDK Method

```bash
# 1. Add to AGENTS.md
cat >> AGENTS.md << 'EOF'

### SDK.spikes.add_finding()

Add finding to existing spike:

```python
sdk.spikes.get('spk-12345').add_finding('Discovered issue').save()
```
EOF

# 2. Sync
# sync-docs not yet in Go CLI

# 3. Verify CLAUDE.md and GEMINI.md reference it
grep -A 2 "SDK Usage" CLAUDE.md GEMINI.md
```

#### Example 2: Add Claude-Specific Orchestrator Pattern

```bash
# 1. Edit CLAUDE.md directly (platform-specific)
vim CLAUDE.md
# Add section: "## Orchestrator Delegation Patterns"

# 2. No sync needed
git add CLAUDE.md
git commit -m "docs: add orchestrator delegation patterns"
```

#### Example 3: Fix Stale Example

```bash
# 1. Fix in AGENTS.md (source of truth)
vim AGENTS.md
# Update example

# 2. Sync to propagate
# sync-docs not yet in Go CLI

# 3. All platform files now have updated example
```

---

## See Also

- **[reference.md](./reference.md)** - Complete sync-docs implementation details
- **[AGENTS.md](../../AGENTS.md)** - Master SDK/API documentation
- **[CLAUDE.md](../../CLAUDE.md)** - Claude-specific project docs
- **[GEMINI.md](../../GEMINI.md)** - Gemini-specific integration docs

---

*Maintain documentation consistency with centralized sync patterns*

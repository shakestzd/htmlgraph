# /htmlgraph:git-commit

Commit and push all staged/unstaged changes in a single script call.

**Cost:** ~$0.002 (one Bash call) vs ~$0.037-0.086 (8 separate git tool calls)
**Savings:** 93-97% token reduction vs multi-tool approach

## Usage

```
/htmlgraph:git-commit <message>
```

## Parameters

- `message` (required): Commit message in conventional commit format
- `--no-confirm` (optional, default: included): Skip confirmation prompt

## Examples

```bash
/htmlgraph:git-commit "feat: add parallel execution engine"
/htmlgraph:git-commit "fix: resolve PostToolUse import error"
/htmlgraph:git-commit "chore: bump version to 0.9.5"
```

## Instructions for Claude

Execute the commit-and-push script in a **single Bash call**. Do NOT use multiple git tool calls.

### Step 1: Check what changed (one call)

```bash
git status --short
```

Understand the files changed — use this to write an accurate commit message.

### Step 2: Stage, commit, and push (one call)

```bash
git add -A && git commit -m "your message here" && git push origin $(git branch --show-current)
```

That's it — stage everything, commit, and push in a single Bash call.

> **Note on `${CLAUDE_PLUGIN_ROOT}`:** This variable is only substituted in JSON config files
> (`hooks.json`, `.mcp.json`). It is NOT available as a shell env var in command markdown or
> direct Bash calls. Use the one-liner above instead of referencing the bundled script.

### Commit Message Format

```
<type>: <short description>

[optional body]

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

When working on a tracked feature, reference it:
```
feat(feat-abc123): implement session ingester

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

### NEVER do this (wastes context):

```bash
# ❌ 8 separate calls = 8K-25K tokens
git status
git add .
git status --short
git diff --cached
git commit -m "message"
git log -1
git push origin main
git status
```

### ALWAYS do this (saves 93-97%):

```bash
# ✅ 2 calls = ~545 tokens
git status --short   # understand what changed
git add -A && git commit -m "feat: description" && git push origin $(git branch --show-current)
```

## Output Format

✅ **Changes committed and pushed**

Commit: `<hash>`
Files changed: `<count>`
Branch: `main → origin/main`

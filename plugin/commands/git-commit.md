# /htmlgraph:git-commit

Commit changes using the copilot-operator delegation pattern.

## Usage

```
/htmlgraph:git-commit [message] [--push] [--files <list>]
```

## Parameters

- `message` (optional): Commit message. If omitted, Claude analyzes the diff and drafts one.
- `--push`: Also push after committing.
- `--files <list>`: Specific files to stage. Default: analyze git diff and select source files, excluding `.htmlgraph/`.

## Examples

```
/htmlgraph:git-commit "feat: add user authentication"
/htmlgraph:git-commit --push
/htmlgraph:git-commit --files src/foo.py tests/test_foo.py "fix: resolve null pointer"
```

## Instructions for Claude

Follow this priority order. Never skip straight to direct git — always try earlier paths first.

### Step 1: Analyze what to commit

```bash
git diff --stat HEAD
git status --short
```

Select source files to stage. Exclude `.htmlgraph/` directory unless explicitly requested.

If no message was provided, draft one from the diff using conventional commit format.

### Priority 1: Delegate to copilot-operator agent (preferred)

```python
Agent(
    subagent_type="htmlgraph:copilot-operator",
    description="Commit: <message>",
    prompt="Stage these files: <list>. Commit with message: '<message>'. Do NOT push."
)
```

The copilot-operator agent tries Copilot CLI first, falls back to direct git, and hook
enforcement verifies compliance. Use this path whenever the agent is available.

If `--push` was passed, add: "Then push to origin."

### Priority 2: Bash-copilot pattern (fallback if agent unavailable)

If the copilot-operator agent is not loaded (e.g., mid-session before restart), try copilot
directly in a single Bash call:

```bash
copilot -p "Stage files: <list>. Commit with message: '<message>'. Do NOT push." \
  --allow-all-tools --no-color --add-dir . 2>&1
```

If copilot is not installed, times out, or errors — fall back to direct git:

```bash
git add <files> && git commit -m "<message>"
```

### Priority 3: Direct git (last resort only)

```bash
git add <files> && git commit -m "<message>"
```

Only reach this if both copilot-operator agent and Bash-copilot have failed or are unavailable.

## Commit Message Format

```
<type>: <short description>

[optional body]

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

Always append the Co-Authored-By line.

## Pre-commit Checks

This project uses `.githooks/` with pre-commit checks (ruff, mypy, systematic change
detection). If pre-commit fails, the commit fails — fix the reported issues and retry.

## Output Format

Report which path was used and the result:

```
Committed via: copilot-operator agent | bash-copilot | direct git
Commit: <hash>
Files changed: <count>
Message: <message>
```

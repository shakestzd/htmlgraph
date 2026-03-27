---
name: copilot
description: GitHub CLI (gh) operations and CopilotSpawner with full event tracking
when_to_use:
  - Creating pull requests and issues
  - Managing GitHub repositories
  - Git operations via gh CLI (direct execution via Bash)
  - GitHub Copilot for code generation (via CopilotSpawner)
  - Git workflows with full HtmlGraph tracking
  - GitHub authentication and configuration
skill_type: executable
---

> **DEPRECATED:** This skill is replaced by the `copilot-operator` agent.
> Use `Agent(subagent_type="htmlgraph:copilot-operator", prompt="...")` instead.
> The copilot-operator agent tries Copilot CLI first with hook-based compliance verification.

# GitHub Copilot & GitHub CLI (gh) Operations

⚠️ **IMPORTANT: This skill provides TWO EXECUTION PATTERNS**

1. **GitHub CLI (gh)** - Documentation for command syntax. Execute via **Bash tool**.
2. **CopilotSpawner** - AI-powered code generation. Invoke via **Python SDK with parent event context**.

This skill teaches HOW to use both. See "EXECUTION PATTERNS" below for when to use each.

---

## 🚀 CopilotSpawner Pattern: Full Event Tracking

### What is CopilotSpawner?

CopilotSpawner is the HtmlGraph-integrated way to invoke external CLIs (Copilot, Gemini, Codex) with **full parent event context and subprocess tracking**.

**Key distinction**: CopilotSpawner is invoked directly via Python SDK - NOT wrapped in Task(). Task() is only for Claude subagents (Haiku, Sonnet, Opus).

Instead of running CLI commands directly (which creates "black boxes"), CopilotSpawner:
- ✅ Invokes external Copilot CLI directly
- ✅ Creates parent event context in database
- ✅ Links to parent Task delegation event
- ✅ Records subprocess invocations as child events
- ✅ Tracks all activities in HtmlGraph event hierarchy
- ✅ Provides full observability of external tool execution

### When to Use CopilotSpawner

**Use CopilotSpawner when:**
- You need to invoke external CLIs (copilot, gemini, codex)
- You want full event tracking in HtmlGraph
- Parent event context is available (from hooks)
- You need subprocess event recording
- You're in a Claude Code session with hook system

**Use Bash directly when:**
- Running simple one-off commands
- No tracking needed
- Testing/debugging
- Not in Claude Code environment

### How to Use CopilotSpawner

Use the `htmlgraph:copilot-operator` agent — it tries Copilot CLI first, then falls back to direct git/gh commands:

```python
# PRIMARY: Delegate to copilot-operator agent
Task(
    subagent_type="htmlgraph:copilot-operator",
    prompt="Recommend next semantic version and git workflow commands",
)
```

### Key Parameters for CopilotSpawner.spawn()

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `prompt` | str | ✅ | Task description for Copilot |
| `track_in_htmlgraph` | bool | ❌ | Enable SDK activity tracking (default: True) |
| `tracker` | SpawnerEventTracker | ❌ | Tracker instance for subprocess events |
| `parent_event_id` | str | ❌ | Parent event ID for event hierarchy |
| `allow_tools` | list[str] | ❌ | Tools to auto-approve (e.g., ["shell(git)"]) |
| `allow_all_tools` | bool | ❌ | Auto-approve all tools (default: False) |
| `deny_tools` | list[str] | ❌ | Tools to deny |
| `timeout` | int | ❌ | Max seconds to wait (default: 120) |

### Event Tracking Hierarchy

With CopilotSpawner, you get this event hierarchy in HtmlGraph:

```
UserQuery Event (from UserPromptSubmit hook)
├── Task Delegation Event (from PreToolUse hook)
    ├── CopilotSpawner Start (activity tracking)
    ├── Subprocess Invocation (subprocess event)
    │   └── subprocess.copilot tool call
    ├── CopilotSpawner Result (activity tracking)
    └── All activities linked with parent_event_id
```

This provides complete observability - no "black boxes" for external tool execution.

### Real Example: Version Update Workflow

```python
Task(
    subagent_type="htmlgraph:copilot-operator",
    prompt="""HtmlGraph project status:
- Completed CLI module refactoring (all tests passing)
- Completed skill documentation clarification
- Completed spawner architecture modularization
- Current version: ~0.26.x

Please recommend:
1. Next semantic version (MAJOR.MINOR.PATCH)
2. Version update git workflow including all necessary files
3. Tag and push commands"""
)
```

### Fallback & Error Handling Pattern

The `htmlgraph:copilot-operator` agent handles the fallback automatically:
1. Tries Copilot CLI first (cheap, GitHub-native)
2. Falls back to direct git/gh commands if Copilot CLI unavailable

No manual fallback code needed — just delegate to the agent.

**Why fallback to Task()?**
- ✅ External CLI may not be installed on user's system
- ✅ Network/permissions issues may affect external tools
- ✅ Claude sub-agent provides guaranteed execution fallback
- ✅ Never attempt direct execution as fallback (violates orchestration principles)
- ✅ Task() handles all retries, error recovery, and parent context automatically

**Pattern Summary:**
1. Try external spawner first (Copilot CLI)
2. If spawner succeeds → return result
3. If spawner fails → delegate to Claude sub-agent via Task()
4. Never try direct execution as fallback

---

## What is GitHub CLI (gh)?

GitHub CLI (`gh`) is a command-line tool for GitHub operations:
- **Pull Requests** - Create, view, merge, and manage PRs
- **Issues** - Create, list, and manage GitHub issues
- **Repositories** - Clone, fork, and manage repos
- **Authentication** - Manage GitHub credentials and tokens
- **Workflows** - Trigger and view GitHub Actions

Works in your terminal with the `gh` command.

## Skill vs Execution Model

**CRITICAL DISTINCTION:**

| What | Description |
|------|-------------|
| **This Skill** | Documentation teaching HOW to use `gh` CLI |
| **Bash Tool** | ACTUAL execution of `gh` commands |

**Workflow:**
1. Read this skill to learn `gh` CLI syntax and options
2. Use **Bash tool** to execute the actual commands
3. Use SDK to track results in HtmlGraph

## Installation

```bash
# Install GitHub CLI
# macOS
brew install gh

# Ubuntu/Debian
sudo apt update && sudo apt install gh

# Windows
choco install gh

# Authenticate with GitHub
gh auth login

# Verify installation
gh --version
```

## EXECUTION - Real Commands to Use in Bash Tool

**⚠️ To actually execute GitHub operations, use these commands via the Bash tool:**

### Create Pull Request
```bash
# Basic PR creation
gh pr create --title "Feature: Add authentication" --body "Implements JWT auth"

# PR with multiple options
gh pr create --title "Fix bug" --body "Description" --base main --head feature-branch

# Interactive PR creation
gh pr create --web
```

### Manage Issues
```bash
# Create issue
gh issue create --title "Bug: Login fails" --body "Steps to reproduce..."

# List issues
gh issue list --state open

# View issue
gh issue view 123
```

### Repository Operations
```bash
# Clone repository
gh repo clone owner/repo

# Fork repository
gh repo fork owner/repo --clone

# Create repository
gh repo create my-new-repo --public
```

### Git Operations via gh
```bash
# Check status and create commit
git add . && git commit -m "feat: add new feature"

# Push and create PR in one command
git push && gh pr create --fill

# View PR status
gh pr status
```

## How to Use This Skill

**STEP 1: Read this skill to learn gh CLI syntax**
```python
# This loads the documentation (this file)
Skill(skill=".claude-plugin:copilot")
```

**STEP 2: Execute commands via Bash tool**
```python
# This ACTUALLY creates a PR
Bash("gh pr create --title 'Feature' --body 'Description'")

# This ACTUALLY creates an issue
Bash("gh issue create --title 'Bug' --body 'Details'")

# This ACTUALLY clones a repo
Bash("gh repo clone user/repo")
```

**What this skill does:**
- ✅ Provides documentation and examples
- ✅ Teaches gh CLI syntax and options
- ✅ Shows common workflows and patterns
- ❌ Does NOT execute commands
- ❌ Does NOT create PRs or issues
- ❌ Does NOT run git operations

**To execute: Use Bash tool with the commands shown in "EXECUTION" section.**

## Example Use Cases (Execute via Bash)

### 1. Pull Request Workflows

```bash
# Create PR after committing changes
Bash("gh pr create --title 'Add feature X' --body 'Implements X with tests'")

# Create draft PR
Bash("gh pr create --draft --title 'WIP: Feature Y'")

# List your PRs
Bash("gh pr list --author @me")

# Merge PR
Bash("gh pr merge 123 --squash")
```

### 2. Issue Management

```bash
# Create bug report
Bash("gh issue create --title 'Bug: Auth fails' --body 'Steps: 1. Login 2. Error'")

# List open issues
Bash("gh issue list --state open")

# Close issue
Bash("gh issue close 456")
```

### 3. Repository Operations

```bash
# Clone repo
Bash("gh repo clone anthropics/claude-code")

# Fork and clone
Bash("gh repo fork user/repo --clone")

# View repo details
Bash("gh repo view")
```

### 4. Commit and Push Workflows

```bash
# Commit all changes and create PR
Bash("git add . && git commit -m 'feat: new feature' && git push && gh pr create --fill")

# Amend last commit and force push
Bash("git commit --amend --no-edit && git push --force-with-lease")

# Check PR status
Bash("gh pr status")
```

### 5. GitHub Actions

```bash
# List workflow runs
Bash("gh run list")

# View specific run
Bash("gh run view 123")

# Re-run failed jobs
Bash("gh run rerun 123 --failed")
```

## Use Cases

- **Pull Requests** - Create, manage, and merge PRs via command line
- **Issues** - Create and track GitHub issues efficiently
- **Repository Management** - Clone, fork, and configure repos
- **Authentication** - Manage GitHub credentials and tokens
- **CI/CD** - Trigger and monitor GitHub Actions workflows
- **Code Review** - View and comment on PRs from terminal

## Requirements

- GitHub account
- `gh` CLI installed
- Authenticated via `gh auth login`
- Git configured (for git operations)

## Integration with HtmlGraph

Track GitHub operations:

```bash
# Create or update a feature to document PR creation
htmlgraph feature create "Authentication Feature PR"
# Note: PR created via gh CLI — branch: feature/jwt-auth, status: ready for review
```

## When to Use

✅ **Use GitHub CLI (gh) for:**
- Creating and managing pull requests
- Creating and tracking issues
- Cloning and forking repositories
- GitHub authentication
- Viewing and managing workflows
- Command-line GitHub operations

❌ **Don't use gh CLI for:**
- Code generation (use Task() delegation or direct implementation)
- Exploring codebases (use `/gemini` skill instead)
- Complex git operations (use git commands directly)
- Local file operations (use standard bash/python)

## Tips for Best Results

1. **Use --fill flag** - Auto-populate PR details from commits: `gh pr create --fill`
2. **Check status first** - Run `gh pr status` or `gh issue list` before creating new items
3. **Use templates** - Leverage repository issue/PR templates when available
4. **Authenticate early** - Run `gh auth login` at start of session
5. **Combine with git** - Chain git and gh commands: `git push && gh pr create`
6. **Use --web flag** - Open operations in browser when needed: `gh pr create --web`

## Common Patterns (Execute via Bash)

### Pattern 1: Feature Development Workflow

```bash
# 1. Create feature branch
Bash("git checkout -b feature/new-feature")

# 2. Make changes, commit
Bash("git add . && git commit -m 'feat: implement feature'")

# 3. Push and create PR
Bash("git push -u origin feature/new-feature && gh pr create --fill")
```

### Pattern 2: Bug Fix Workflow

```bash
# 1. Create issue for bug
Bash("gh issue create --title 'Bug: Description' --body 'Steps to reproduce'")

# 2. Create branch from issue
Bash("git checkout -b fix/issue-123")

# 3. Fix, commit, and create PR
Bash("git add . && git commit -m 'fix: resolve issue #123' && gh pr create --fill")
```

### Pattern 3: Quick PR Creation

```bash
# 1. Commit all changes
Bash("git add . && git commit -m 'feat: quick feature'")

# 2. Push and create PR in one command
Bash("git push && gh pr create --title 'Quick Feature' --body 'Description'")
```

## Limitations

- Requires GitHub account and authentication
- Internet connection required for GitHub API
- Rate limits apply to API calls
- Some operations require repository permissions
- Cannot execute local git operations (use git directly)

## Related Skills

- `/gemini` - For exploring repository structure and large codebases
- `/codex` - For code generation and implementation
- `/code-quality` - For validating code quality before creating PR
- Git commands - For local repository operations (use Bash directly)

## When NOT to Use

Avoid GitHub CLI for:
- Code generation (use Task() delegation)
- Exploratory research (use `/gemini` skill)
- Local file operations (use standard bash commands)
- Operations not related to GitHub (use appropriate tool)

## Error Handling (Use Bash for Diagnostics)

### GitHub CLI Not Installed

```bash
Error: "gh: command not found"

Solution (via Bash):
# macOS
Bash("brew install gh")

# Verify
Bash("gh --version")
```

### Authentication Required

```bash
Error: "authentication required"

Solution (via Bash):
Bash("gh auth login")
# Follow prompts in terminal
```

### Permission Denied

```bash
Error: "permission denied to repository"

Solution (via Bash):
# Check authentication status
Bash("gh auth status")

# Re-authenticate if needed
Bash("gh auth login --force")
```

## Advanced Usage (Execute via Bash)

### PR with Reviewers and Labels

```bash
Bash("gh pr create --title 'Feature' --body 'Desc' --reviewer user1,user2 --label bug,enhancement")
```

### Create Issue from Template

```bash
Bash("gh issue create --template bug_report.md")
```

### Batch Operations

```bash
# Close multiple issues
Bash("gh issue list --state open --json number --jq '.[].number' | xargs -I {} gh issue close {}")
```

### Integration with Git Workflows

```bash
# Rebase and force push
Bash("git rebase main && git push --force-with-lease && gh pr ready")

# Squash commits and update PR
Bash("git rebase -i HEAD~3 && git push --force-with-lease")
```

## Key Features

| Feature | GitHub CLI (gh) | Description |
|---------|-----------------|-------------|
| Pull Requests | ✅ | Create, merge, review, comment |
| Issues | ✅ | Create, list, close, assign |
| Repositories | ✅ | Clone, fork, create, view |
| Authentication | ✅ | Login, status, token management |
| GitHub Actions | ✅ | List, view, re-run workflows |
| API Access | ✅ | Direct GitHub API calls via `gh api` |

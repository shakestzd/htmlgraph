# Orchestrator Mode - Complete Guide

## Table of Contents

- [Quick Start (30 Seconds)](#quick-start-30-seconds)
- [How It Works](#how-it-works)
- [Operation Reference](#operation-reference)
- [Examples & Patterns](#examples--patterns)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

---

## Quick Start (30 Seconds)

Orchestrator Mode helps you **preserve context** and **work faster** by delegating low-value operations to subagents.

```bash
# Enable orchestrator mode
uv run wipnote orchestrator enable

# Check if it's working
uv run wipnote orchestrator status

# Start working - you'll get guidance when you should delegate
# Example: After 3 Bash calls, you'll see:
# ⚠️ ORCHESTRATOR MODE: Consider delegating to Task tool
```

**That's it!** Orchestrator mode will guide you to better workflow patterns.

---

## How It Works

### Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    AI Agent (You)                        │
│  Attempts to use tool: Bash, Edit, Grep, etc.          │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│              PreToolUse Hook (Interceptor)               │
│  - Check orchestrator.json config                       │
│  - Count tool usage in current session                  │
│  - Classify operation (allowed/warned/blocked)          │
└────────────────┬────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────┐
│                    Decision Logic                        │
│                                                          │
│  ✅ ALLOWED → Execute tool                              │
│  ⚠️  WARNED → Execute + Show delegation suggestion      │
│  🚫 BLOCKED → Prevent execution + Show error            │
└─────────────────────────────────────────────────────────┘
```

### Enforcement Modes

**1. Strict Mode (Default)**
- Blocks operations that exceed thresholds
- Forces delegation to subagents
- Best for: Production workflows, complex projects

**2. Guidance Mode**
- Warns but allows all operations
- Shows delegation suggestions
- Best for: Learning, experimentation

### Operation Lifecycle

1. **Agent initiates tool call**
   ```python
   bash("uv run pytest tests/")
   ```

2. **Hook intercepts before execution**
   ```python
   # PreToolUse hook checks:
   # - Current bash_call_count = 3
   # - Threshold = 3
   # - Mode = strict
   ```

3. **Classification decision**
   ```python
   # Decision: BLOCKED (exceeded threshold)
   ```

4. **Guidance provided**
   ```
   ⚠️ ORCHESTRATOR MODE: Exceeded threshold for Bash calls (3/3)

   Suggestion: Delegate to subagent using Task tool
   Example: Task(subagent_type="general-purpose",
                 prompt="Run pytest and report failures")

   Rationale: Running tests fills orchestrator context with test output.
   Subagents can run tests in parallel and return summaries.
   ```

5. **Agent adjusts approach**
   ```python
   # Instead of direct call, delegate:
   Task(
       subagent_type="general-purpose",
       prompt="Run pytest and report only failures"
   )
   ```

---

## Operation Reference

### Complete Classification Matrix

| Operation | Threshold | Strict Mode | Guidance Mode | Rationale |
|-----------|-----------|-------------|---------------|-----------|
| **CLI tracking ops** | Unlimited | ✅ Allowed | ✅ Allowed | High-level, minimal context |
| **Task tool** | Unlimited | ✅ Allowed | ✅ Allowed | Designed for delegation |
| **TodoWrite** | Unlimited | ✅ Allowed | ✅ Allowed | Task management |
| **Read** (≤5) | 5 | ✅ Allowed | ✅ Allowed | Reasonable exploration |
| **Read** (>5) | 5 | 🚫 Blocked | ⚠️ Warned | Fills context, delegate search |
| **Bash** (≤3) | 3 | ✅ Allowed | ✅ Allowed | Quick commands OK |
| **Bash** (>3) | 3 | 🚫 Blocked | ⚠️ Warned | Test output fills context |
| **Edit** (≤5) | 5 | ✅ Allowed | ✅ Allowed | Focused changes OK |
| **Edit** (>5) | 5 | 🚫 Blocked | ⚠️ Warned | Large refactors, delegate |
| **Grep** (≤5) | 5 | ✅ Allowed | ✅ Allowed | Targeted searches OK |
| **Grep** (>5) | 5 | 🚫 Blocked | ⚠️ Warned | Exploration, delegate |
| **Glob** (≤5) | 5 | ✅ Allowed | ✅ Allowed | File discovery OK |
| **Glob** (>5) | 5 | 🚫 Blocked | ⚠️ Warned | Large-scale, delegate |

### Why These Thresholds?

**Bash (3 calls):**
- 1-3 calls: Quick commands, status checks → Allowed
- 4+ calls: Running tests, builds, multiple commands → Delegate

**Read (5 files):**
- 1-5 files: Understanding specific modules → Allowed
- 6+ files: Codebase exploration → Delegate

**Edit (5 files):**
- 1-5 edits: Focused bug fix or feature → Allowed
- 6+ edits: Large refactoring → Delegate

**Grep/Glob (5 calls):**
- 1-5 searches: Targeted queries → Allowed
- 6+ searches: Broad exploration → Delegate

---

## Examples & Patterns

### Pattern 1: Test Execution

#### ❌ Anti-Pattern (Direct Execution)

```python
# Orchestrator runs tests directly
result1 = bash("uv run pytest tests/unit/")
# ✅ Allowed (1/3 bash calls)

result2 = bash("uv run pytest tests/integration/")
# ✅ Allowed (2/3 bash calls)

result3 = bash("uv run pytest tests/e2e/")
# ✅ Allowed (3/3 bash calls)

result4 = bash("uv run pytest tests/performance/")
# 🚫 BLOCKED (4/3 bash calls)
# Error: Orchestrator mode blocks this operation
```

**Problems:**
- Sequential execution (slow)
- Test output fills context
- Can't run more than 3 test suites
- Context consumed by test results

#### ✅ Success Pattern (Delegation)

```python
# Orchestrator delegates to parallel subagents
unit_results = Task(
    subagent_type="general-purpose",
    prompt="""
    Run unit tests and report:
    - Total tests run
    - Failures (name + error)
    - Summary statistics
    """
)

integration_results = Task(
    subagent_type="general-purpose",
    prompt="""
    Run integration tests and report:
    - Total tests run
    - Failures (name + error)
    - Summary statistics
    """
)

e2e_results = Task(
    subagent_type="general-purpose",
    prompt="""
    Run e2e tests and report:
    - Total tests run
    - Failures (name + error)
    - Summary statistics
    """
)
# All 3 run in parallel, orchestrator gets summaries only
```

**Benefits:**
- ✅ Parallel execution (3x faster)
- ✅ Context preserved (summaries only)
- ✅ Unlimited test suites
- ✅ Orchestrator focuses on decisions

### Pattern 2: Multi-File Refactoring

#### ❌ Anti-Pattern (Direct Editing)

```python
# Orchestrator edits 10 files
files = [
    "src/api/users.py",
    "src/api/posts.py",
    "src/api/comments.py",
    "src/api/auth.py",
    "src/api/profiles.py",
    "src/api/settings.py",  # 6th file → BLOCKED
    # ... 4 more files won't be processed
]

for file in files:
    Edit(file, old="old_api", new="new_api")
    # First 5: ✅ Allowed
    # 6th+: 🚫 BLOCKED
```

**Problems:**
- Can't complete refactor (blocked after 5)
- Context filled with diffs
- Sequential edits
- No summary of changes

#### ✅ Success Pattern (Delegation)

```python
# Orchestrator delegates entire refactor
Task(
    subagent_type="general-purpose",
    prompt="""
    Update all API files to use new_api instead of old_api:

    Files to update:
    - src/api/users.py
    - src/api/posts.py
    - src/api/comments.py
    - src/api/auth.py
    - src/api/profiles.py
    - src/api/settings.py
    - src/api/notifications.py
    - src/api/search.py
    - src/api/admin.py
    - src/api/webhooks.py

    Report:
    - Files updated successfully
    - Any files skipped (with reason)
    - Summary of changes
    - Any issues encountered
    """
)
```

**Benefits:**
- ✅ All files processed
- ✅ Context preserved (summary only)
- ✅ Subagent handles details
- ✅ Clear completion report

### Pattern 3: Codebase Exploration

#### ❌ Anti-Pattern (Direct Searching)

```python
# Orchestrator searches directly
grep("class.*API", output_mode="content")     # 1/5
grep("def.*endpoint", output_mode="content")  # 2/5
grep("@router", output_mode="content")        # 3/5
grep("async def", output_mode="content")      # 4/5
grep("database.*query", output_mode="content") # 5/5
grep("cache.*redis", output_mode="content")   # 🚫 BLOCKED
```

**Problems:**
- Limited searches (only 5)
- Results fill context
- Can't complete exploration
- Manual synthesis required

#### ✅ Success Pattern (Delegation)

```python
# Orchestrator delegates exploration
Task(
    subagent_type="general-purpose",
    prompt="""
    Explore the codebase and document the API architecture:

    Find and report:
    1. All API endpoint classes (class.*API)
    2. Route definitions (@router)
    3. Async endpoints (async def)
    4. Database queries (database.*query)
    5. Caching patterns (cache.*redis)
    6. Authentication middleware

    Provide:
    - Summary of architecture
    - List of endpoints by category
    - Database access patterns
    - Caching strategy
    - Security mechanisms

    Format as structured report with code examples.
    """
)
```

**Benefits:**
- ✅ Unlimited searches
- ✅ Comprehensive exploration
- ✅ Structured report
- ✅ Context preserved

### Pattern 4: Debugging Session

#### ❌ Anti-Pattern (Direct Investigation)

```python
# Orchestrator investigates directly
read("src/api/users.py")     # 1/5
read("src/models/user.py")   # 2/5
read("src/db/queries.py")    # 3/5
read("tests/test_users.py")  # 4/5
read("src/utils/auth.py")    # 5/5

bash("uv run pytest tests/test_users.py -v")  # 1/3
bash("uv run pytest tests/test_users.py --pdb")  # 2/3
bash("grep -r 'UserModel' src/")  # 3/3

# Can't read more files or run more commands
# 🚫 BLOCKED on next operation
```

#### ✅ Success Pattern (Delegation)

```python
# Orchestrator delegates investigation
Task(
    subagent_type="general-purpose",
    prompt="""
    Debug the user authentication failure in tests/test_users.py:

    Investigation steps:
    1. Read relevant source files (users.py, user.py, auth.py)
    2. Read test file and understand failure
    3. Run tests with verbose output
    4. Identify root cause

    Report:
    - Root cause of failure
    - Files involved
    - Proposed fix
    - Test results before/after fix (if implemented)
    """
)
```

**Benefits:**
- ✅ Unrestricted investigation
- ✅ Focused report
- ✅ Context preserved
- ✅ Actionable results

---

## Configuration

### Default Configuration

Located at `.wipnote/orchestrator.json`:

```json
{
  "enabled": false,
  "mode": "strict",
  "thresholds": {
    "max_bash_calls": 3,
    "max_file_reads": 5,
    "max_file_edits": 5,
    "max_grep_calls": 5,
    "max_glob_calls": 5
  },
  "allowed_tools": [
    "SDK",
    "Task",
    "TodoWrite"
  ]
}
```

### Customizing Thresholds

**Option 1: Edit configuration file**

```bash
# Edit directly
vim .wipnote/orchestrator.json

# Example: Allow 10 file reads instead of 5
{
  "thresholds": {
    "max_file_reads": 10
  }
}
```

**Option 2: CLI (Future)**

```bash
# Not yet implemented, but planned:
uv run wipnote orchestrator set-threshold max_bash_calls 5
uv run wipnote orchestrator set-threshold max_file_reads 10
```

### Mode Switching

```bash
# Strict mode (blocks operations)
uv run wipnote orchestrator enable --mode strict

# Guidance mode (warns only)
uv run wipnote orchestrator enable --mode guidance

# Disable entirely
uv run wipnote orchestrator disable
```

### Per-Project Configuration

Each project can have different settings:

```bash
# Project A: Strict with low thresholds
cd /path/to/project-a
uv run wipnote orchestrator enable --mode strict
vim .wipnote/orchestrator.json  # Set max_bash_calls=2

# Project B: Guidance with high thresholds
cd /path/to/project-b
uv run wipnote orchestrator enable --mode guidance
vim .wipnote/orchestrator.json  # Set max_bash_calls=10
```

### Temporarily Disable for Single Session

```bash
# Disable for this task
uv run wipnote orchestrator disable

# Do your work
# ...

# Re-enable when done
uv run wipnote orchestrator enable
```

---

## Troubleshooting

### Problem: "Operation blocked but I need to do it"

**Scenario:** You're blocked from running a 4th Bash command, but you really need to run it.

**Solutions:**

1. **Use Guidance Mode (Recommended)**
   ```bash
   uv run wipnote orchestrator enable --mode guidance
   # Now you'll get warnings but not blocks
   ```

2. **Increase Threshold (If justified)**
   ```bash
   vim .wipnote/orchestrator.json
   # Change max_bash_calls from 3 to 5
   ```

3. **Delegate Instead (Best)**
   ```python
   # Delegate the operation to subagent
   Task(prompt="Run the command and report results")
   ```

4. **Temporarily Disable**
   ```bash
   uv run wipnote orchestrator disable
   # Run your commands
   uv run wipnote orchestrator enable
   ```

### Problem: "Too many operations blocked"

**Scenario:** Every operation is getting blocked, can't make progress.

**Solutions:**

1. **Check if you're doing the right task**
   - Are you exploring? → Delegate to subagent
   - Are you refactoring? → Delegate to subagent
   - Are you testing? → Delegate to subagent

2. **Switch to Guidance Mode**
   ```bash
   uv run wipnote orchestrator enable --mode guidance
   # Learn the patterns without blocks
   ```

3. **Review your workflow**
   ```python
   # If you're doing this:
   for file in files:
       Edit(file, ...)

   # Consider this instead:
   Task(prompt=f"Update all files in {files}")
   ```

### Problem: "Don't understand why operation was blocked"

**Scenario:** Got blocked but the reason isn't clear.

**Solution:** Read the guidance message carefully:

```
⚠️ ORCHESTRATOR MODE: Exceeded threshold for Edit calls (5/5)

Suggestion: Delegate to subagent using Task tool
Example: Task(subagent_type="general-purpose",
             prompt="Update all files to use new API")

Rationale: Editing many files fills orchestrator context with diffs.
Subagents can handle bulk edits and return summaries.
```

**Key parts:**
1. **What:** "Exceeded threshold for Edit calls"
2. **Why:** "Editing many files fills orchestrator context"
3. **How:** Delegate using Task tool
4. **Example:** Actual delegation code

### Problem: "Orchestrator mode not activating"

**Scenario:** Enabled orchestrator mode but not seeing any enforcement.

**Solutions:**

1. **Check status**
   ```bash
   uv run wipnote orchestrator status
   # Should show: enabled=true
   ```

2. **Verify configuration exists**
   ```bash
   cat .wipnote/orchestrator.json
   # Should have: "enabled": true
   ```

3. **Check PreToolUse hook is installed**
   ```bash
   ls .claude/hooks/pre-tool-use/
   # Should contain orchestrator hook
   ```

4. **Restart Claude Code**
   ```bash
   # Sometimes hooks need reload
   # Close and reopen Claude Code
   ```

### Problem: "Want different thresholds per operation"

**Scenario:** 3 Bash calls is too low, but 5 Edit calls is fine.

**Solution:** Edit configuration:

```json
{
  "thresholds": {
    "max_bash_calls": 10,    // Increased
    "max_file_reads": 5,     // Keep default
    "max_file_edits": 3,     // Decreased
    "max_grep_calls": 5,     // Keep default
    "max_glob_calls": 5      // Keep default
  }
}
```

---

## FAQ

### General Questions

**Q: What is orchestrator mode?**

A: An enforcement system that guides you to delegate context-filling operations to subagents, preserving your context for high-level decisions.

**Q: Why should I use it?**

A: Benefits:
- ✅ Preserve context for strategic thinking
- ✅ Work faster (parallel subagents)
- ✅ Learn better workflow patterns
- ✅ Scale to larger projects

**Q: Will this slow me down?**

A: No - delegation is actually faster:
- Direct: 3 sequential Bash calls (slow)
- Delegated: 3 parallel subagents (fast)
- Plus, you preserve context for decisions

**Q: Can I disable it?**

A: Yes:
```bash
uv run wipnote orchestrator disable
```

### Mode & Configuration

**Q: What's the difference between strict and guidance mode?**

A:
- **Strict:** Blocks operations that exceed thresholds (enforces delegation)
- **Guidance:** Warns but allows all operations (teaches patterns)

**Q: Which mode should I start with?**

A: Start with **guidance mode** to learn patterns:
```bash
uv run wipnote orchestrator enable --mode guidance
```

After you understand the patterns, switch to strict:
```bash
uv run wipnote orchestrator enable --mode strict
```

**Q: Can I customize thresholds?**

A: Yes, edit `.wipnote/orchestrator.json`:
```json
{
  "thresholds": {
    "max_bash_calls": 5  // Changed from 3
  }
}
```

**Q: Do different projects have different configs?**

A: Yes - each project has its own `.wipnote/orchestrator.json`.

### Operations & Thresholds

**Q: Why is Read limited to 5 files?**

A: After 5 file reads, you're likely exploring the codebase. That's better delegated to a subagent who can:
- Read unlimited files
- Provide a structured report
- Not fill your context with file contents

**Q: Why only 3 Bash calls?**

A: After 3 Bash calls, you're likely:
- Running test suites (output fills context)
- Running builds (output fills context)
- Debugging with multiple commands

Subagents can handle this and return summaries.

**Q: What if I just need to check a status?**

A: That's fine - 1-3 Bash calls are allowed:
```bash
bash("git status")           # ✅ Allowed
bash("uv run pytest --co")   # ✅ Allowed
bash("ls .wipnote/")       # ✅ Allowed
```

**Q: Why are CLI/tracking operations unlimited?**

A: CLI tracking operations are high-level and context-efficient:
```bash
wipnote feature create "Title"  # Minimal context
wipnote feature start <id>      # Minimal context
```

### Delegation

**Q: How do I delegate operations?**

A: Use the Task tool:
```python
Task(
    subagent_type="general-purpose",
    prompt="Your detailed instructions here"
)
```

**Q: What makes a good delegation prompt?**

A: Include:
1. **What to do:** "Run pytest tests"
2. **How to report:** "Report only failures"
3. **Format:** "Structured as: total, failures, errors"

Example:
```python
Task(prompt="""
Run all unit tests in tests/unit/ and report:
- Total tests run
- Failures (name + error message)
- Summary statistics

Format as markdown list.
""")
```

**Q: Can subagents delegate further?**

A: Yes - subagents can also use Task tool for recursive delegation.

**Q: How many subagents can I spawn?**

A: Unlimited - orchestrator mode doesn't restrict Task tool usage.

### Workflow

**Q: When should I use orchestrator mode?**

A: Use when:
- ✅ Managing complex multi-step workflows
- ✅ Coordinating multiple features
- ✅ Running comprehensive test suites
- ✅ Large-scale refactoring
- ✅ Codebase exploration

**Q: When should I skip orchestrator mode?**

A: Skip when:
- ❌ Single, focused task (bug fix)
- ❌ Quick prototype
- ❌ Documentation writing
- ❌ Learning/experimenting

**Q: How do I know if orchestrator mode is working?**

A: Check status:
```bash
uv run wipnote orchestrator status
```

You should see guidance messages when approaching thresholds.

**Q: What if I disagree with a block?**

A: Options:
1. Use guidance mode instead of strict
2. Increase the threshold
3. Open an issue - we want to improve classification logic

### Advanced

**Q: Can I add custom operation classifications?**

A: Not yet, but planned. Future:
```json
{
  "custom_rules": {
    "Write": {
      "max_calls": 3,
      "rationale": "Writing many files should be delegated"
    }
  }
}
```

**Q: Can I see statistics on my delegations?**

A: Not yet, but planned:
```bash
uv run wipnote orchestrator stats
# Shows: delegations, blocks, time saved
```

**Q: Does orchestrator mode work with other AI agents?**

A: Yes - any agent using Wipnote respects orchestrator mode (Claude, Gemini, etc.).

**Q: How do I contribute improvements?**

A: Open issues/PRs at: https://github.com/shakestzd/wipnote

---

## Summary

**Orchestrator Mode in 3 Steps:**

1. **Enable it**
   ```bash
   uv run wipnote orchestrator enable --mode guidance
   ```

2. **Learn patterns**
   - Watch for warnings
   - Adjust your workflow
   - Delegate context-filling work

3. **Enforce it**
   ```bash
   uv run wipnote orchestrator enable --mode strict
   ```

**Key Takeaway:** Orchestrator mode helps you **preserve context** and **work faster** by teaching you to delegate effectively.

**Get Help:**
- Docs: `/Users/shakes/DevProjects/htmlgraph/AGENTS.md#orchestrator-mode`
- Issues: https://github.com/shakestzd/wipnote/issues
- Examples: See "Examples & Patterns" section above

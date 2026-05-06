# Orchestrator System Prompt - Implementation Guide

**Date:** 2025-01-03
**Project:** Wipnote
**Version:** 1.0

---

## Overview

This guide shows how to deploy and use the Comprehensive Orchestrator System Prompt in your Claude Code environment.

**Three deployment options:**
1. **Full Replacement** (2500 tokens) - Maximum orchestrator behavior
2. **Append Mode** (condensed, ~600 tokens) - Hybrid behavior
3. **Environment Variable** - Persistent across sessions

---

## Option 1: Full System Prompt Replacement

### Setup

```bash
# Set permanent environment variable
export CLAUDE_SYSTEM_PROMPT="$(cat orchestrator-system-prompt.txt)"

# Verify it's loaded
echo $CLAUDE_SYSTEM_PROMPT | head -c 100

# Now all Claude Code invocations use orchestrator mode
claude -p "Your task..."
```

### Single-Use Alternative

```bash
# Use just for this invocation
claude --system-prompt "$(cat orchestrator-system-prompt.txt)" -p "Your task..."

# Or with append mode
claude --append-system-prompt "$(cat orchestrator-system-prompt-condensed.txt)" -p "Your task..."
```

### Example Workflow

```bash
# Deploy orchestrator prompt
export CLAUDE_SYSTEM_PROMPT="$(cat orchestrator-system-prompt.txt)"

# Now use orchestrator-style commands
claude -p "
Design a deployment workflow for Wipnote considering:
- Multiple AI providers (Claude, Gemini, Copilot, Codex)
- Cost optimization
- Parallel execution capability
- Integration with Wipnote SDK

Use the HeadlessSpawner framework. For each major component,
decide: execute directly, delegate with Task(), or spawn specialized agent.
"
```

---

## Option 2: Append Mode (Quick Setup)

### For Occasional Orchestrator Use

```bash
# Append condensed prompt to session
claude --append-system-prompt "$(cat orchestrator-system-prompt-condensed.txt)" -p "
Coordinate analysis of 5 Python files for code quality issues.
Use HeadlessSpawner to spawn parallel Gemini agents.
Save results to Wipnote.
"
```

### Minimal Prompt Version (100 tokens)

```bash
claude --append-system-prompt "
You are an ORCHESTRATOR. For work:
- Decompose into tasks
- Use spawn_codex() for code, spawn_gemini() for analysis
- Use Task() for sequential dependent work
- Track results in Wipnote using SDK
- Delegate everything except strategic decisions

When to delegate:
- Git operations
- Code implementation
- Research/testing
- Build/deployment

Decision: Is this independent/parallel? → spawn_*
Is this sequential dependent? → Task()
Is this strategic? → Execute directly
" -p "Your task..."
```

---

## Option 3: Plugin Integration

### For Wipnote Plugin Users

```python
# In your Claude Code plugin configuration (.claude/settings.json)
{
  "plugins": [
    {
      "name": "orchestrator",
      "enabled": true,
      "config": {
        "system_prompt": "orchestrator-system-prompt.txt",
        "mode": "full",
        "spawner_enabled": true,
        "auto_track_wipnote": true
      }
    }
  ]
}
```

### Using with Wipnote Plugin

```bash
# Plugin automatically loads orchestrator prompt
claude -p "Your orchestration task..."

# Access Wipnote SDK in orchestrator context
from wipnote import SDK
from wipnote.orchestration import HeadlessSpawner

sdk = SDK(agent='orchestrator')
spawner = HeadlessSpawner()

# Orchestrator automatically tracks work
feature = sdk.features.create("Implement OAuth")
# ... rest of orchestration workflow
```

---

## Common Deployment Scenarios

### Scenario 1: Single Project Setup

```bash
# In project root directory
cd ~/DevProjects/my-project

# Create local orchestrator prompt
cp orchestrator-system-prompt.txt .claude/orchestrator-prompt.txt

# Set for this project only (add to .bashrc or .zshrc)
alias claude-orchestrator="claude --system-prompt \"\$(cat .claude/orchestrator-prompt.txt)\""

# Use it
claude-orchestrator -p "Design the feature flow"
```

### Scenario 2: Global Setup for All Projects

```bash
# Copy prompt to user config
mkdir -p ~/.claude
cp orchestrator-system-prompt.txt ~/.claude/system-prompt.txt

# Set permanent environment variable (add to .bashrc, .zshrc, etc)
echo 'export CLAUDE_SYSTEM_PROMPT="$(cat ~/.claude/system-prompt.txt)"' >> ~/.zshrc
source ~/.zshrc

# Now all claude invocations use orchestrator mode globally
claude -p "Any task uses orchestrator mode automatically"
```

### Scenario 3: Hybrid Mode (Project-Specific Override)

```bash
# Global orchestrator prompt (default)
export CLAUDE_SYSTEM_PROMPT="$(cat ~/.claude/system-prompt.txt)"

# Project-specific override
cd ~/DevProjects/special-project
alias claude="claude --system-prompt \"\$(cat .claude/special-prompt.txt)\""

# In this project only, uses special-prompt.txt
# In other projects, falls back to global CLAUDE_SYSTEM_PROMPT
```

### Scenario 4: Conditional Use

```bash
#!/bin/bash
# Detect task type and choose prompt automatically

if [[ "$1" == "orchestrate" || "$1" == "design" ]]; then
    # Use full orchestrator prompt
    claude --system-prompt "$(cat orchestrator-system-prompt.txt)" -p "$2"
elif [[ "$1" == "code" ]]; then
    # Use regular Claude (default)
    claude -p "$2"
else
    # Use condensed orchestrator
    claude --append-system-prompt "$(cat orchestrator-system-prompt-condensed.txt)" -p "$1"
fi

# Usage
./claude-smart.sh orchestrate "Design the feature"
./claude-smart.sh code "Implement the function"
./claude-smart.sh "Quick analysis"
```

---

## Integration with Wipnote Workflows

### Orchestrator + SDK Pattern

```python
#!/usr/bin/env python3
"""
Orchestrator workflow using Wipnote SDK.
Run with: CLAUDE_SYSTEM_PROMPT="..." python orchestrator.py
"""

from wipnote import SDK
from wipnote.orchestration import (
    HeadlessSpawner,
    delegate_with_id,
    save_task_results,
    get_results_by_task_id
)

def main():
    # Initialize SDK
    sdk = SDK(agent='orchestrator')
    spawner = HeadlessSpawner()

    # Step 1: Create feature (strategic decision)
    feature = sdk.features.create("Add OAuth authentication") \
        .set_priority("high") \
        .add_steps([
            "Analyze current auth system",
            "Design OAuth flow",
            "Implement OAuth",
            "Write tests",
            "Update documentation"
        ]) \
        .save()

    print(f"Created feature: {feature.id}")

    # Step 2: Delegate analysis (parallel work)
    print("Delegating analysis tasks...")

    analyze_id, analyze_prompt = delegate_with_id(
        "Analyze current auth",
        "Review existing authentication implementation...",
        "general-purpose"
    )

    design_id, design_prompt = delegate_with_id(
        "Design OAuth flow",
        "Create OAuth 2.0 architecture...",
        "general-purpose"
    )

    # Spawn in parallel
    analysis = spawner.spawn_claude(
        "Analyze the current auth implementation for compatibility with OAuth",
        permission_mode="plan"
    )

    design = spawner.spawn_claude(
        "Design an OAuth 2.0 implementation strategy",
        permission_mode="plan"
    )

    # Step 3: Save analysis
    save_task_results(sdk, analyze_id, "Analyze", analysis.response,
                     feature_id=feature.id)
    save_task_results(sdk, design_id, "Design", design.response,
                     feature_id=feature.id)

    print(f"Analysis saved. Ready for implementation.")

    # Step 4: Delegate implementation (sequential with shared context)
    impl_id, impl_prompt = delegate_with_id(
        "Implement OAuth",
        f"""
        Based on this analysis: {analysis.response}
        And this design: {design.response}

        Implement the OAuth flow in the application.
        """,
        "general-purpose"
    )

    # Would normally delegate to Task() here
    print(f"Implementation task created: {impl_id}")

    # Step 5: Mark feature progress
    feature.set_status("in-progress").save()

    print(f"Feature {feature.id} orchestration complete")

if __name__ == "__main__":
    main()
```

### Running the Orchestrator Script

```bash
# Set orchestrator prompt first
export CLAUDE_SYSTEM_PROMPT="$(cat orchestrator-system-prompt.txt)"

# Run orchestrator workflow
python orchestrator.py

# Output:
# Created feature: feat-abc123
# Delegating analysis tasks...
# Analysis saved. Ready for implementation.
# Implementation task created: task-xyz789
# Feature feat-abc123 orchestration complete
```

---

## Decision-Making Walkthrough

### Example 1: "Implement User Authentication"

```
ORCHESTRATOR DECISION PROCESS:

Task: "Implement user authentication"

Q1: Is this strategic?
→ YES (what features to build)
→ Execute directly: Create feature in Wipnote

Q2: Decompose into work items:
   a) Research existing patterns (independent)
   b) Design OAuth flow (strategic)
   c) Implement OAuth (tactical, multi-file)
   d) Write tests (tactical)
   e) Update docs (tactical)

Q3: For each item:
   a) Research → spawn_claude (plan mode) - independent
   b) Design → spawn_claude (plan mode) - strategic
   c) Implement → Task() - needs shared context with tests
   d) Tests → Task() - dependent on implementation
   e) Docs → Task() - dependent on implementation

ORCHESTRATOR PLAN:
1. Execute: Create feature, prioritize, decompose (strategic)
2. Spawn: spawn_claude for research and design (parallel, independent)
3. Save results to Wipnote
4. Delegate: Task() for implementation + tests + docs (sequential, shared context)
5. Track: Update feature status throughout
```

### Example 2: "Analyze 10 Python Files for Security Issues"

```
ORCHESTRATOR DECISION PROCESS:

Task: "Analyze 10 Python files for security issues"

Q1: Is this strategic?
→ NO (tactical analysis)

Q2: Can one tool call?
→ NO (10 files)

Q3: Error handling needed?
→ YES (analysis might fail on some files)
→ Delegate

Q4: Independent or shared context?
→ INDEPENDENT (each file analyzed separately)
→ Use spawn_*

Q5: Which spawner?
→ Code analysis + security focus
→ spawn_codex with sandbox="read-only"
→ Or spawn_gemini (cheaper, fast)

ORCHESTRATOR PLAN:
1. Execute: Parse file list, set up coordination (strategic)
2. Spawn: 10 parallel spawn_gemini tasks (one per file)
3. Collect: Aggregate results as they complete
4. Save: Store findings in Wipnote spike
5. Analyze: Orchestrator reviews findings and decides next steps
6. Delegate: If issues found, Task() to fix them sequentially
```

### Example 3: "Fix a Bug in the Authentication Module"

```
ORCHESTRATOR DECISION PROCESS:

Task: "Fix bug in auth.py where tokens expire too quickly"

Q1: Is this strategic?
→ NO (tactical fix)

Q2: Can one tool call?
→ NO (requires: understand, fix, test, verify)

Q3: Error handling needed?
→ YES (bugs might be complex)
→ Delegate

Q4: Independent or shared context?
→ DEPENDENT (fix needs validation with tests)
→ Use Task()

Q5: Why Task() instead of spawn_codex?
→ Code changes are dependent on tests
→ Tests depend on fixes
→ Shared context saves tokens (cache hits)
→ Sequential workflow

ORCHESTRATOR PLAN:
1. Execute: Understand the bug (read code, search logs) - strategic
2. Create: Feature/bug tracking in Wipnote
3. Delegate: Task() "Analyze token expiry logic and suggest fix"
4. Delegate: Task() "Implement the fix based on analysis"
5. Delegate: Task() "Write tests validating token expiry"
6. Delegate: Task() "Verify fix in staging environment"
5. Track: Update Wipnote with completion
```

---

## Measuring Orchestrator Effectiveness

### Metrics to Track

```python
# Track these metrics to measure orchestrator effectiveness

metrics = {
    "total_tool_calls": 0,
    "direct_executions": 0,
    "delegations": 0,
    "spawns": 0,
    "tokens_used": 0,
    "cascade_failures": 0,  # Should be 0
    "untracked_work": 0,    # Should be 0
    "context_preserved": "Yes/No",
    "strategic_decisions": 0,
    "tactical_work": 0
}

# Good orchestration signs:
# - tool_calls / features_completed < 5 (high efficiency)
# - cascade_failures == 0 (good delegation)
# - untracked_work == 0 (good tracking)
# - context_preserved == "Yes"
# - cascading delegations reduce tokens by 80%+
```

### Example Metrics Report

```
ORCHESTRATOR EFFECTIVENESS REPORT
=================================

Feature: OAuth Implementation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Without Orchestrator (direct execution):
  - Tool calls: 23
  - Tokens: 45,000
  - Failures: 3 (retry cascade)
  - Time: 45 minutes

With Orchestrator:
  - Tool calls: 5
  - Tokens: 8,000
  - Failures: 0
  - Time: 25 minutes

Efficiency Gain:
  - Tool calls reduced: 78% (23 → 5)
  - Tokens saved: 82% (45K → 8K)
  - Failures prevented: 100% (3 → 0)
  - Time saved: 44% (45m → 25m)

Key Success Factors:
  ✅ Delegation to Task() for sequential work (cache hits)
  ✅ spawn_gemini for parallel analysis (cost savings)
  ✅ spawn_codex for code generation (specialized)
  ✅ All work tracked in Wipnote
  ✅ Strategic context maintained
```

---

## Troubleshooting Common Issues

### Issue 1: "Too Many Tool Calls"

**Symptom:** Orchestrator makes 8+ tool calls before completing task

**Root Cause:** Not delegating enough, trying to do everything directly

**Fix:**
```python
# ❌ Wrong: Direct execution leading to cascading calls
bash("git add .")
bash("git commit -m message")
bash("git push")
# Fails if: conflicts, hooks fail, tests fail in hooks
# Cascades into 3+ retries

# ✅ Right: Delegate
Task(prompt="Commit and push changes with error handling...")
# Subagent handles retries, error recovery
# Orchestrator gets clean success/failure
```

### Issue 2: "Lost Context Between Operations"

**Symptom:** Can't remember previous findings or decisions

**Root Cause:** Using spawn_* for dependent work instead of Task()

**Fix:**
```python
# ❌ Wrong: Each spawn is fresh context
spawner.spawn_claude("Implement feature step 1")
spawner.spawn_claude("Implement feature step 2")  # Doesn't know about step 1

# ✅ Right: Use Task() for dependent work
Task(prompt="Implement feature step 1")
Task(prompt="Implement feature step 2 (building on step 1)")
# Both share context, step 2 knows about step 1
# Caching saves 5x cost
```

### Issue 3: "Untracked Work Everywhere"

**Symptom:** Can't find previous analysis or results

**Root Cause:** Not using Wipnote SDK to track delegated work

**Fix:**
```python
# ❌ Wrong: Delegate but don't track
result = Task(prompt="Analyze this")
# Result is lost, can't find it later

# ✅ Right: Track with Wipnote
from wipnote.orchestration import delegate_with_id, save_task_results

task_id, prompt = delegate_with_id("Analyze", "...", "general-purpose")
result = Task(prompt=prompt, description=f"{task_id}: Analyze")
save_task_results(sdk, task_id, "Analyze", result)
# Result saved to Wipnote spike, always findable
```

### Issue 4: "Choosing Wrong Spawner"

**Symptom:** spawn_claude() for quick checks (expensive), or spawn_gemini() for code generation (lower quality)

**Root Cause:** Not following decision tree

**Fix:**
```python
# Use the priority order:
# 1. Code gen? → spawn_codex
# 2. Images? → spawn_gemini
# 3. GitHub? → spawn_copilot
# 4. Quick check? → spawn_gemini
# 5. Complex reasoning? → spawn_claude

# Examples:
spawner.spawn_codex("Fix this bug")      # Code generation
spawner.spawn_gemini("Analyze image")    # Images
spawner.spawn_copilot("Review PR")       # GitHub
spawner.spawn_gemini("Check syntax")     # Quick check
spawner.spawn_claude("Design system")    # Complex reasoning
```

### Issue 5: "Expensive Token Usage"

**Symptom:** 30K+ tokens for what should be 5K

**Root Cause:** Not leveraging caching with Task() or parallelization with spawn_*

**Fix:**
```python
# ❌ Wrong: 3 sequential Task calls, each is fresh context
Task(prompt="Implement auth")         # 10K tokens (cache miss)
Task(prompt="Implement tests")        # 10K tokens (cache miss)
Task(prompt="Implement docs")         # 10K tokens (cache miss)
# Total: 30K tokens

# ✅ Right: Related sequential work
Task(prompt="Implement auth")         # 5K tokens (cache miss, but related)
Task(prompt="Implement tests (auth)")  # 1K tokens (cache hit on auth context)
Task(prompt="Implement docs (auth)")   # 1K tokens (cache hit on auth context)
# Total: 7K tokens (77% savings!)

# OR ✅ Right: Independent parallel work
spawner.spawn_gemini("Analyze file 1")  # 500 tokens
spawner.spawn_gemini("Analyze file 2")  # 500 tokens
spawner.spawn_gemini("Analyze file 3")  # 500 tokens
# Total: 1.5K tokens (parallel, cheap)
```

---

## Quick Command Reference

```bash
# Deploy full orchestrator prompt
export CLAUDE_SYSTEM_PROMPT="$(cat orchestrator-system-prompt.txt)"

# Use condensed version
claude --append-system-prompt "$(cat orchestrator-system-prompt-condensed.txt)" -p "task"

# Create task with tracking
claude -p "Use Wipnote SDK to delegate with delegate_with_id()"

# Check if prompt is loaded
echo "$CLAUDE_SYSTEM_PROMPT" | head -c 50

# Unload orchestrator (back to default)
unset CLAUDE_SYSTEM_PROMPT

# Verify orchestrator behavior
claude -p "How would you approach: [task]?" | head -20
```

---

## Best Practices Summary

1. **Lead with Strategy** - Orchestrator makes architectural decisions
2. **Decompose Work** - Break complex tasks into independent pieces
3. **Choose Spawner Wisely** - Use decision tree for selection
4. **Track Everything** - Use Wipnote SDK for all work items
5. **Optimize Cost** - Task() for dependent, spawn_* for parallel
6. **Preserve Context** - Keep strategic context throughout
7. **Validate Results** - Check quality gates before committing
8. **Document Decisions** - Save reasoning in Wipnote spikes

---

**Next Steps:**

1. Choose deployment option (full, append, or environment variable)
2. Copy appropriate prompt file(s) to your system
3. Test with a simple orchestration task
4. Monitor metrics and adjust strategies
5. Build organizational patterns around orchestrator mode
6. Share learnings with team

For detailed design rationale, see: `ORCHESTRATOR_SYSTEM_PROMPT_DESIGN.md`
For full system prompt, see: `orchestrator-system-prompt.txt`
For condensed version, see: `orchestrator-system-prompt-condensed.txt`

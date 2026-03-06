# Debugging Workflow Skill

Use this skill for systematic debugging with research-first approach.

**Trigger keywords:** debug, error, bug, troubleshoot, investigate, research, fix

---

## When to Use This Skill

Activate this skill when you encounter:
- Errors or unexpected behavior
- Failing tests or quality gates
- Unfamiliar systems or frameworks
- Multiple failed fix attempts
- Need to understand root cause before implementing solutions

## Core Principle

**NEVER implement solutions based on assumptions. ALWAYS research documentation first.**

### Research Checkpoint Questions

Before implementing any fix, ask yourself:

1. **Have I researched the documentation?**
   - Official docs, GitHub issues, hook documentation
   - Use claude-code-guide agent for Claude-specific questions

2. **Do I understand the root cause?**
   - Evidence-based diagnosis, not guesses
   - Reproduce the error consistently

3. **Have I considered using debugging agents?**
   - Researcher agent - Documentation research
   - Debugger agent - Systematic analysis
   - Test runner agent - Validation

4. **Am I about to try multiple solutions hoping one works?**
   - STOP - Research first instead

## Quick Debugging Workflow

```
1. 🔍 RESEARCH
   - Read documentation
   - Use claude-code-guide agent
   - Check GitHub issues

2. 🎯 UNDERSTAND
   - Identify root cause
   - Gather evidence (logs, errors)
   - Reproduce consistently

3. 🔧 IMPLEMENT
   - Apply fix based on understanding
   - Make minimal changes
   - One change at a time

4. ✅ VALIDATE
   - Run tests
   - Verify fix works
   - Check quality gates

5. 📝 DOCUMENT
   - Capture learning in HtmlGraph spike
   - Record research findings
   - Share patterns discovered
```

## Built-in Debug Tools

```bash
# Claude Code debugging
claude --debug <command>        # Verbose output
/hooks                          # List all active hooks
/hooks PreToolUse              # Show specific hook type
/doctor                         # System diagnostics
claude --verbose               # Detailed logging

# HtmlGraph debugging
uv run htmlgraph orchestrator status
uv run htmlgraph status
uv run htmlgraph feature show <id>
uv run htmlgraph session list --active
```

## Quality Gates

Always run before committing:

```bash
# Full quality gate
uv run ruff check --fix && \
uv run ruff format && \
uv run mypy src/ && \
uv run pytest
```

## Debugging Agents

### 1. Researcher Agent
**Purpose:** Research documentation BEFORE implementing solutions

**Use when:**
- Encountering unfamiliar errors
- Working with Claude Code hooks/plugins
- Before implementing assumptions
- Multiple fix attempts failed

### 2. Debugger Agent
**Purpose:** Systematically analyze and resolve errors

**Methodology:**
1. Gather evidence (logs, errors, traces)
2. Reproduce consistently
3. Isolate variables
4. Analyze context
5. Form hypothesis
6. Test hypothesis
7. Implement minimal fix

### 3. Test Runner Agent
**Purpose:** Validate changes, enforce quality gates

**Use when:**
- After code changes
- Before marking tasks complete
- After fixing bugs
- Before committing

## Anti-Patterns to Avoid

❌ **Trial-and-Error Debugging**
- Making multiple fix attempts without research
- Hoping one solution works
- Not understanding root cause

❌ **Assumption-Based Fixes**
- "This should work" without evidence
- Implementing based on guesses
- Skipping documentation research

❌ **Skipping Validation**
- Not running tests after fixes
- Ignoring quality gate failures
- Committing broken code

## Example: Correct Workflow

**Scenario:** Duplicate hooks causing errors

✅ **Correct approach:**
1. Research Claude Code hook loading behavior
2. Use claude-code-guide to understand hook merging
3. Identify hooks from multiple sources MERGE
4. Check all hook sources (.claude/settings.json, plugins)
5. Remove duplicates based on understanding
6. Verify fix works
7. Document learning in spike

❌ **Wrong approach:**
1. Remove .claude/hooks/hooks.json - Still broken
2. Clear plugin cache - Still broken
3. Remove old plugins - Still broken
4. Remove symlinks - Still broken
5. (Finally research after wasting time)

## Integration with HtmlGraph

```python
from htmlgraph import SDK

# Document debugging findings
sdk = SDK(agent='debugger')
spike = sdk.spikes.create('Investigation: Duplicate hooks error') \
    .set_findings("""
    Research discovered:
    - Claude Code merges hooks from multiple sources
    - Sources: .claude/settings.json, plugin hooks
    - Solution: Remove duplicates from settings.json
    """) \
    .save()
```

## WIP Limit Issues

**Symptom:** `ValueError: WIP limit (3) reached. Complete existing work first.`

**Critical:** Do NOT iterate with multiple Bash calls to debug this. Delegate once.

### What counts toward WIP limit
- Features (`feat-*`) with `in-progress` status
- Spikes (`spk-*`) with `in-progress` status
- Both live in `.htmlgraph/features/` directory

### Quick diagnosis
```bash
uv run htmlgraph wip   # Shows all active items with ages
```

Or via SDK:
```python
active = sdk.session_manager.get_active_features()
print([(n.id, n.title, n.status) for n in active])
```

### Reset stale items (delegate this)
Instead of debugging iteratively, delegate to haiku-coder:

> "Reset stale WIP items [list IDs] and start feature [feat-xxx]"

The coder can: edit HTML `data-status` directly → start new feature → all in one delegation.

## Documentation References

**Detailed methodology:** See `reference.md` in this skill directory

**Debugging agents:** See `packages/claude-plugin/agents/`
- `researcher.md` - Research-first methodology
- `debugger.md` - Systematic error analysis
- `test-runner.md` - Quality gates and testing

**Past debugging sessions:** See `.htmlgraph/spikes/`
- Learn from previous debugging workflows
- Avoid repeating mistakes

---

## Skill Metadata

**Version:** 1.0.0
**Category:** Development Workflow
**Complexity:** Intermediate
**Estimated time:** Varies by issue complexity

**Related skills:**
- Test automation
- Code quality enforcement
- Documentation research

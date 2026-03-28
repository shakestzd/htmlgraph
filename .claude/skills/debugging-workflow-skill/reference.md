# Debugging Workflow - Complete Reference

This document provides comprehensive debugging methodology for HtmlGraph development.

## Table of Contents

1. [Core Principle](#core-principle)
2. [Debugging Agents](#debugging-agents)
3. [Built-in Debug Tools](#built-in-debug-tools)
4. [Debugging Workflow Patterns](#debugging-workflow-patterns)
5. [HtmlGraph Debug Commands](#htmlgraph-debug-commands)
6. [Integration with Orchestrator Mode](#integration-with-orchestrator-mode)
7. [Quality Gates](#quality-gates)
8. [Common Debugging Scenarios](#common-debugging-scenarios)
9. [Documentation References](#documentation-references)

---

## Core Principle

**NEVER implement solutions based on assumptions. ALWAYS research documentation first.**

This principle emerged from dogfooding HtmlGraph development. We repeatedly violated it by:
- ❌ Making multiple trial-and-error attempts before researching
- ❌ Implementing "fixes" based on guesses instead of documentation
- ❌ Not using available debugging tools and agents

### The Correct Approach

1. ✅ **Research** - Use claude-code-guide agent, read documentation
2. ✅ **Understand** - Identify root cause through evidence
3. ✅ **Implement** - Apply fix based on understanding
4. ✅ **Validate** - Test to confirm fix works
5. ✅ **Document** - Capture learning in HtmlGraph spike

### Why Research First?

**Time Efficiency:**
- Trial-and-error: 5-10 attempts = 30-60 minutes wasted
- Research-first: 10 minutes reading = correct fix on first try

**Context Preservation:**
- Multiple failed attempts consume LLM context
- Research once, implement once = minimal context usage

**Learning:**
- Understanding root cause prevents future issues
- Trial-and-error teaches nothing reusable

**Code Quality:**
- Evidence-based fixes are robust
- Guess-based fixes often have edge cases

---

## Debugging Agents

HtmlGraph plugin includes three specialized agents for systematic debugging.

### 1. Researcher Agent

**Purpose:** Research documentation BEFORE implementing solutions

**Use when:**
- Encountering unfamiliar errors or behaviors
- Working with Claude Code hooks, plugins, or configuration
- Before implementing solutions based on assumptions
- When multiple attempted fixes have failed
- Integrating with unfamiliar libraries or frameworks

**Workflow:**
```bash
# Activate researcher agent (use Task tool to delegate)
# 1. Clearly state what needs research
# 2. Provide context (error messages, behavior observed)
# 3. Specify documentation sources to check
# 4. Request summary of findings

# Example delegation:
Task(
    prompt="""
    Research Claude Code hook loading and merging behavior.

    Context:
    - Duplicate PreToolUse hooks appearing
    - Hooks defined in both .claude/settings.json and plugin
    - Need to understand how Claude Code merges hooks

    Research sources:
    1. Claude Code docs: https://code.claude.com/docs
    2. Hook documentation: https://code.claude.com/docs/en/hooks.md
    3. GitHub issues related to hook merging

    Document findings in HtmlGraph spike.
    """,
    subagent_type="researcher"
)
```

**Key Resources:**
- Claude Code docs: https://code.claude.com/docs
- GitHub issues: https://github.com/anthropics/claude-code/issues
- Hook documentation: https://code.claude.com/docs/en/hooks.md
- Plugin development: https://code.claude.com/docs/en/plugins.md

**Output Format:**
Document findings in HtmlGraph spike:
```python
from htmlgraph import SDK
sdk = SDK(agent='researcher')
spike = sdk.spikes.create('Research: Claude Code hook merging') \
    .set_findings("""
    Key findings:
    1. Hooks from multiple sources MERGE (not replace)
    2. Sources: .claude/settings.json, plugin hooks, global settings
    3. Duplicate hooks = same type defined in multiple sources
    4. Solution: Define hooks in ONE location only

    References:
    - https://code.claude.com/docs/en/hooks.md#hook-merging
    - GitHub issue #123: Hook deduplication
    """) \
    .save()
```

### 2. Debugger Agent

**Purpose:** Systematically analyze and resolve errors

**Use when:**
- Error messages appear but root cause is unclear
- Behavior doesn't match expectations
- Tests are failing
- Hooks or plugins aren't working as expected
- Performance issues or unexpected slowdowns
- Integration points behaving unexpectedly

**Methodology:**

1. **Gather Evidence**
   - Collect error messages, stack traces, logs
   - Note exact steps to reproduce
   - Record system state (versions, config)
   - Capture screenshots if UI-related

2. **Reproduce Consistently**
   - Document exact reproduction steps
   - Create minimal reproduction case
   - Test in clean environment if possible
   - Verify it's not a one-time fluke

3. **Isolate Variables**
   - Change one thing at a time
   - Use git bisect to find breaking commit
   - Test with minimal configuration
   - Disable features one by one

4. **Analyze Context**
   - What changed recently?
   - When did this start happening?
   - Does it happen in all environments?
   - Are there related issues in issue tracker?

5. **Form Hypothesis**
   - Propose root cause based on evidence
   - Make it testable/falsifiable
   - Consider multiple hypotheses
   - Prioritize by likelihood

6. **Test Hypothesis**
   - Design experiment to validate/refute
   - Look for confirming and disconfirming evidence
   - Adjust hypothesis based on results
   - Repeat until root cause identified

7. **Implement Fix**
   - Make minimal change to fix root cause
   - Avoid fixing symptoms instead of cause
   - Add tests to prevent regression
   - Document why fix works

**Built-in Debug Tools:**

```bash
# Claude Code debugging
claude --debug <command>        # Verbose output with internals
/hooks                          # List all active hooks
/hooks PreToolUse              # Show specific hook type
/doctor                         # System diagnostics
claude --verbose               # More detailed logging

# Python debugging
uv run python -m pdb script.py  # Interactive debugger
uv run pytest -vv               # Verbose test output
uv run pytest -x                # Stop on first failure
uv run pytest --pdb             # Drop into debugger on failure

# HtmlGraph debugging
htmlgraph orchestrator status  # Check orchestrator state
htmlgraph status              # List active features
htmlgraph feature show <id>   # Feature details
htmlgraph session list  # Active sessions
```

**Example Delegation:**

```python
Task(
    prompt="""
    Debug failing test: test_feature_creation

    Error message:
    AssertionError: Expected feature ID 'feat-123' but got 'feat-456'

    Steps to reproduce:
    1. uv run pytest tests/test_features.py::test_feature_creation
    2. Error occurs consistently

    Investigate:
    1. Gather full stack trace
    2. Check feature ID generation logic
    3. Verify test expectations are correct
    4. Identify root cause
    5. Propose minimal fix

    Document findings in spike.
    """,
    subagent_type="debugger"
)
```

### 3. Test Runner Agent

**Purpose:** Automatically test changes, enforce quality gates

**Use when:**
- After implementing code changes
- Before marking features/tasks complete
- After fixing bugs
- Before committing code
- During CI/CD pipeline
- Before deployment

**Test Commands:**

```bash
# Run all tests
uv run pytest

# Run specific test file
uv run pytest tests/test_features.py

# Run specific test
uv run pytest tests/test_features.py::test_feature_creation

# Run with coverage
uv run pytest --cov=src/python/htmlgraph --cov-report=html

# Type checking
uv run mypy src/

# Linting
uv run ruff check --fix
uv run ruff format

# Full quality gate (pre-commit)
uv run ruff check --fix && \
uv run ruff format && \
uv run mypy src/ && \
uv run pytest

# Watch mode (re-run on changes)
uv run pytest-watch
```

**Quality Gate Enforcement:**

The deployment script (`deploy-all.sh`) blocks on:
- Mypy type errors
- Ruff lint errors
- Test failures

This is intentional - maintain quality gates.

**Example Delegation:**

```python
Task(
    prompt="""
    Run full quality gate before committing.

    Steps:
    1. uv run ruff check --fix
    2. uv run ruff format
    3. uv run mypy src/
    4. uv run pytest

    If any step fails:
    - Report the specific error
    - Do NOT commit
    - Wait for fix

    If all pass:
    - Report success
    - OK to proceed with commit
    """,
    subagent_type="test-runner"
)
```

---

## Built-in Debug Tools

### Claude Code Tools

```bash
# Verbose debugging output
claude --debug <command>
# Shows: hook execution, tool calls, internal state

# List all active hooks
/hooks
# Shows: all hooks from all sources (merged)

# Show specific hook type
/hooks PreToolUse
# Shows: all PreToolUse hooks and their sources

# System diagnostics
/doctor
# Shows: version, config, plugins, environment

# Detailed logging
claude --verbose
# Shows: more context in logs

# Check plugin status
claude plugin list
# Shows: installed plugins and versions

# Update plugin
claude plugin update htmlgraph
# Updates to latest version
```

### HtmlGraph CLI Tools

```bash
# Check orchestrator status
htmlgraph orchestrator status
# Shows: mode (strict/permissive), active features

# List all features
htmlgraph status
# Shows: features, status, priority

# View specific feature
htmlgraph feature show <id>
# Shows: full feature details, edges, steps

# List active sessions
htmlgraph session list
# Shows: current sessions with activity

# View session details
htmlgraph session show <session-id>
# Shows: session events, timeline

# Sync documentation
# sync-docs not yet in Go CLI
# Shows: which docs are out of sync

# Analytics
htmlgraph analytics summary
# Shows: recommended work based on analytics

htmlgraph analytics summary
# Shows: blocking features
```

### Python Debugging

```bash
# Interactive debugger
uv run python -m pdb script.py

# Add breakpoint in code
import pdb; pdb.set_trace()

# Better debugger (ipdb)
uv add --dev ipdb
import ipdb; ipdb.set_trace()

# Print debugging (simple but effective)
print(f"Debug: variable={variable}")

# Logging
import logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)
logger.debug("Debug message")
```

---

## Debugging Workflow Patterns

### Pattern 1: Duplicate Hooks Issue (Real Example)

**Scenario:** Duplicate PreToolUse hooks causing errors

**❌ What we did initially (WRONG):**
1. Removed `.claude/hooks/hooks.json` - Still broken
2. Cleared plugin cache - Still broken
3. Removed old plugin versions - Still broken
4. Removed marketplaces symlink - Still broken
5. Finally researched documentation
6. Found root cause: Hook merging behavior

**Time wasted:** ~45 minutes of trial-and-error

**✅ What we should have done (CORRECT):**
1. Research Claude Code hook loading behavior first (10 min)
2. Use claude-code-guide agent to understand hook merging
3. Identify that hooks from multiple sources MERGE, not replace
4. Check all hook sources (`.claude/settings.json`, plugin hooks)
5. Remove duplicates based on understanding
6. Verify fix works (1 try, successful)
7. Document learning in spike

**Time saved:** ~30 minutes

### Pattern 2: Test Failures

**❌ WRONG approach:**
```python
# Just try different implementations hoping one works
def fix_attempt_1():
    # Maybe this works?
    pass

def fix_attempt_2():
    # How about this?
    pass
```

**✅ CORRECT approach:**
```python
# 1. Research what the test expects
# 2. Understand why current implementation fails
# 3. Implement fix based on understanding

# Before fixing:
# - Read test code to understand expectations
# - Check documentation for correct behavior
# - Verify test assumptions are correct
```

### Pattern 3: Integration Issues

**❌ WRONG approach:**
- Try different API endpoints randomly
- Guess at parameter formats
- Copy-paste from StackOverflow without understanding

**✅ CORRECT approach:**
1. Read official API documentation
2. Check example usage in docs/GitHub
3. Verify API version compatibility
4. Test with minimal example
5. Build up complexity gradually

### Pattern 4: Performance Problems

**❌ WRONG approach:**
- Randomly add caching
- Optimize based on guesses
- Pre-mature optimization

**✅ CORRECT approach:**
1. Measure first (profiling)
2. Identify actual bottlenecks
3. Research optimization strategies for that specific bottleneck
4. Implement targeted fix
5. Measure improvement

---

## HtmlGraph Debug Commands

### Orchestrator Status

```bash
htmlgraph orchestrator status
```

**Shows:**
- Current mode (strict/permissive)
- Active features
- Delegation statistics
- Context usage

**Use when:**
- Checking if orchestrator mode is enabled
- Debugging delegation issues
- Understanding current work state

### Feature Management

```bash
# List all features
htmlgraph status

# Show specific feature
htmlgraph feature show feat-abc123

# List by status
htmlgraph feature list --status in-progress

# List by priority
htmlgraph feature list --priority high
```

**Shows:**
- Feature metadata (title, status, priority)
- Dependencies (blocks, blocked-by)
- Implementation steps
- Activity log

### Session Management

```bash
# List all sessions
htmlgraph session list

# Show active sessions only
htmlgraph session list

# Show specific session
htmlgraph session show sess-abc123

# Session timeline
htmlgraph session timeline sess-abc123
```

**Shows:**
- Session metadata
- Events and activities
- Associated features
- Timeline of work

### Analytics

```bash
# Recommend next work
htmlgraph analytics summary

# Find bottlenecks
htmlgraph analytics summary

# Show dependency graph
htmlgraph analytics dependency-graph
```

**Shows:**
- Strategic recommendations
- Blocking features
- Dependency chains
- Work prioritization

---

## Integration with Orchestrator Mode

When orchestrator mode is enabled (strict), you'll receive reflections after direct tool execution:

```
ORCHESTRATOR REFLECTION: You executed code directly.

Ask yourself:
- Could this have been delegated to a subagent?
- Would parallel Task() calls have been faster?
- Is a work item tracking this effort?
- What if this operation fails - how many retries will consume context?
```

### Delegation Pattern for Debugging

**Instead of:**
```python
# Orchestrator directly debugging
Read("src/feature.py")
# Try fix 1
Edit("src/feature.py", ...)
Bash("uv run pytest")  # Fails
# Try fix 2
Edit("src/feature.py", ...)
Bash("uv run pytest")  # Fails
# (Consuming orchestrator context)
```

**Do this:**
```python
# Orchestrator delegates to debugger agent
Task(
    prompt="""
    Debug failing test in src/feature.py

    Error: AssertionError in test_feature_creation

    Steps:
    1. Read test to understand expectations
    2. Research correct implementation pattern
    3. Implement fix based on understanding
    4. Validate with tests
    5. Report findings

    Track in spike.
    """,
    subagent_type="debugger"
)

# Orchestrator stays available for strategic decisions
# Debugger agent handles tactical debugging in isolated context
```

---

## Quality Gates

### Pre-Commit Quality Gate

Always run before committing:

```bash
# Full quality gate
uv run ruff check --fix && \
uv run ruff format && \
uv run mypy src/ && \
uv run pytest
```

**Why all four?**
1. **Ruff check** - Catch code quality issues
2. **Ruff format** - Consistent code style
3. **Mypy** - Type safety
4. **Pytest** - Functional correctness

### Deployment Quality Gate

The `deploy-all.sh` script enforces:

```bash
# Pre-flight checks
1. Ruff linting
2. Mypy type checking
3. Pytest test suite
4. Plugin sync verification

# Only proceeds if ALL pass
```

**Why strict?**
- Prevents broken code from reaching production
- Maintains code quality standards
- Catches regressions early
- Forces fixing errors immediately

### Code Hygiene Rule

**CRITICAL: Always fix ALL errors with every commit, regardless of when they were introduced.**

**Philosophy:**
- Maintaining clean, error-free code is non-negotiable
- Every commit should reduce technical debt, not accumulate it

**Rules:**
1. Fix all errors before committing
2. No "I'll fix it later" mentality
3. Pre-existing errors are YOUR responsibility when you touch related code
4. Clean as you go - leave code better than you found it

**Why?**
- Prevents error accumulation
- Better code hygiene
- Faster development (no debugging old errors)
- Professional standards

---

## Common Debugging Scenarios

### Scenario 1: Import Errors

**Symptoms:**
```python
ImportError: cannot import name 'Feature' from 'htmlgraph'
```

**Research First:**
1. Check package structure in `src/python/htmlgraph/`
2. Verify `__init__.py` exports
3. Confirm installation: `uv pip list | grep htmlgraph`

**Common Causes:**
- Package not installed: `uv pip install -e .`
- Circular imports: Refactor module structure
- Missing `__init__.py` exports: Add to `__all__`

### Scenario 2: Hook Not Executing

**Symptoms:**
- Hook defined but not running
- Expected side effects not happening

**Research First:**
1. Check hook loading: `/hooks`
2. Verify hook syntax in docs
3. Confirm hook type is correct

**Common Causes:**
- Hook in wrong location: Move to `.claude/settings.json`
- Invalid hook format: Check schema
- Duplicate hooks: Remove duplicates

### Scenario 3: Tests Passing Locally, Failing in CI

**Research First:**
1. Check CI environment differences
2. Verify dependency versions match
3. Review CI logs for specific errors

**Common Causes:**
- Environment variables missing
- Different Python version
- Timezone/locale differences
- File path differences (absolute vs relative)

### Scenario 4: Type Errors

**Symptoms:**
```
error: Incompatible types in assignment
```

**Research First:**
1. Check Mypy documentation for error code
2. Verify type annotations are correct
3. Understand Python typing rules

**Common Causes:**
- Incorrect type annotation
- Missing Optional for None values
- Wrong generic type parameters

---

## Documentation References

### Internal Documentation

**Debugging agents:**
- `packages/claude-plugin/agents/researcher.md` - Research methodology
- `packages/claude-plugin/agents/debugger.md` - Systematic analysis
- `packages/claude-plugin/agents/test-runner.md` - Quality gates

**Debugging workflows:**
- `.htmlgraph/spikes/` - Past debugging sessions
- Learn from previous sessions
- Avoid repeating mistakes

### External Resources

**Claude Code:**
- Main docs: https://code.claude.com/docs
- Hooks: https://code.claude.com/docs/en/hooks.md
- Plugins: https://code.claude.com/docs/en/plugins.md
- GitHub issues: https://github.com/anthropics/claude-code/issues

**HtmlGraph:**
- README.md - Project overview
- AGENTS.md - SDK and API documentation
- CLAUDE.md - Development guidelines

**Python:**
- Pytest docs: https://docs.pytest.org
- Mypy docs: https://mypy.readthedocs.io
- Ruff docs: https://docs.astral.sh/ruff

---

## Summary: Research-First Checklist

Before implementing any fix:

- [ ] Have I researched the official documentation?
- [ ] Do I understand the root cause (not just symptoms)?
- [ ] Have I considered using specialized debugging agents?
- [ ] Am I about to try multiple solutions hoping one works? (STOP)
- [ ] Can I explain WHY this fix will work?
- [ ] Have I planned how to validate the fix?
- [ ] Will I document the learning in a spike?

**Remember:** 10 minutes of research saves 30 minutes of trial-and-error.

# Skills Overview Guide

## What Are Skills?

Skills are specialized commands and guides that extend Wipnote with focused, reusable capabilities. Think of them as expert consultants you can summon for specific tasks:

- **Skills provide progressive disclosure** - Start simple, dive deep only when needed
- **Skills integrate with Wipnote** - Seamlessly coordinate with tracking, delegation, and analytics
- **Skills are discoverable** - List, search, and learn at your own pace

When you need guidance on a specific area, invoke a skill to access expert-level documentation and decision frameworks—without overwhelming yourself with unnecessary detail.

---

## Available Skills

| Skill | Purpose | When to Use |
|-------|---------|------------|
| **`/orchestrator-directives`** | Comprehensive orchestration guidance | Planning complex workflows, understanding delegation decisions |
| **`/multi-ai-orchestration`** | Multi-model spawner selection & cost optimization | Choosing the right AI model for subagent tasks |
| **`/code-quality`** | Linting, type checking, testing workflows | Pre-commit validation, ensuring quality gates pass |
| **`/deployment-automation`** | Release management and version workflows | Publishing packages, managing releases |
| **`/debugging-workflow`** | Research-first debugging methodology | Systematic problem-solving, documentation-driven diagnosis |
| **`/memory-sync`** | Documentation synchronization patterns | Keeping docs consistent across platforms |

---

## Decision Tree: Which Skill Should I Use?

Use this flowchart to find the right skill for your task:

### "I'm planning a complex workflow"
→ Use **`/orchestrator-directives`** for:
- Deciding whether to delegate work
- Understanding parent-child session relationships
- Cost optimization strategies
- Complex multi-agent coordination patterns

**Example:** "I need to run 5 parallel test suites and then deploy. Should I delegate?"
```bash
# Get orchestrator guidance
/orchestrator-directives
```

### "I need to choose a model for a subagent"
→ Use **`/multi-ai-orchestration`** for:
- Selecting spawner types (Gemini, Copilot, Codex, Claude)
- Cost vs. capability tradeoffs
- When to use cheap exploratory models vs. expensive reasoning models
- Spawner compatibility with your task

**Example:** "Should I use Gemini (free) or Claude (expensive) to explore the codebase?"
```bash
# Get spawner selection guidance
/multi-ai-orchestration
```

### "I need to run tests/linters before committing"
→ Use **`/code-quality`** for:
- Running ruff, mypy, pytest in the correct order
- Fixing linting errors systematically
- Type checking and test failures
- Full quality gate workflow

**Example:** "My type checker is failing. What's the right fix order?"
```bash
# Get code quality workflow
/code-quality
```

### "I need to release or publish code"
→ Use **`/deployment-automation`** for:
- Version number management
- PyPI publishing
- Plugin updates (Claude, Gemini, etc.)
- Release checklists

**Example:** "I need to bump the version and publish to PyPI."
```bash
# Get deployment workflow
/deployment-automation
```

### "I'm stuck debugging something"
→ Use **`/debugging-workflow`** for:
- Research-first methodology (read docs before guessing)
- Systematic root cause analysis
- When to use debugger vs. researcher agents
- Using Wipnote spike documentation

**Example:** "My hooks aren't loading. I've tried 3 fixes already."
```bash
# Get debugging methodology
/debugging-workflow
```

### "I need to keep multiple docs synchronized"
→ Use **`/memory-sync`** for:
- Central documentation (single source of truth)
- Platform-specific file generation
- Automated consistency checking
- Documentation maintenance patterns

**Example:** "I updated AGENTS.md but forgot to update CLAUDE.md and GEMINI.md"
```bash
# Get memory sync workflow
/memory-sync
```

---

## Skill Descriptions

### `/orchestrator-directives`

**Purpose:** Complete orchestration guidance for complex workflows

**Contains:**
- Full orchestrator decision framework
- When to delegate vs. direct execution
- Cost optimization patterns
- Session hierarchy management
- Multi-agent coordination examples
- Advanced patterns (parallel execution, sequential handoff, divide-and-conquer)

**Use this when:**
- Planning a feature that spans multiple agents
- Deciding whether to delegate work
- Optimizing for cost and speed simultaneously
- Understanding how orchestrator tracking works
- Creating complex delegation hierarchies

**Related reading:** [Delegation Guide](delegation.md), AGENTS.md Orchestrator Mode section

---

### `/multi-ai-orchestration`

**Purpose:** Spawner selection and multi-model cost optimization

**Contains:**
- Spawner types and their capabilities
- Cost vs. capability matrix
- When to use Gemini (free) vs. Copilot vs. Claude
- Dynamic spawner composition patterns
- Cost calculation examples
- Model selection algorithm

**Use this when:**
- You need to choose which AI model to delegate work to
- Optimizing budget for complex workflows
- Mixing multiple spawner types in one workflow
- Balancing speed vs. cost
- Understanding spawner compatibility with tools

**Related reading:** README.md Orchestrator Architecture, AGENTS.md Multi-Agent section

---

### `/code-quality`

**Purpose:** Linting, type checking, and testing validation workflows

**Contains:**
- Complete quality gate sequence
- Ruff fixing and formatting
- Mypy type checking troubleshooting
- Pytest execution patterns
- Batch testing strategies
- Pre-commit workflow

**Use this when:**
- Before committing code
- Fixing linting or type errors
- Running test suites
- Ensuring all quality checks pass
- Understanding test failure patterns

**Related reading:** `.claude/rules/code-hygiene.md`, Deployment Guide

---

### `/deployment-automation`

**Purpose:** Complete release and deployment workflows

**Contains:**
- Version number management across all files
- `deploy-all.sh` script options and flags
- PyPI publishing workflow
- Plugin update patterns (Claude, Gemini, Codex)
- Release checklist
- Rollback strategies

**Use this when:**
- Bumping version numbers
- Publishing a release to PyPI
- Updating Claude plugin
- Managing release tags
- Creating GitHub releases

**Related reading:** `.claude/rules/deployment.md`, AGENTS.md Deployment & Release section

---

### `/debugging-workflow`

**Purpose:** Research-first debugging methodology and systematic problem-solving

**Contains:**
- Research-first vs. trial-and-error comparison
- Built-in debug tools and agents
- Systematic error analysis process
- When to use researcher vs. debugger vs. test-runner agents
- Integration with Wipnote spikes for documentation
- Anti-patterns to avoid

**Use this when:**
- You encounter an unfamiliar error
- You've tried 2+ fixes without success
- Working with Claude Code hooks or plugins
- Need systematic root cause analysis
- Want to document your debugging process

**Related reading:** `.claude/rules/debugging.md`, AGENTS.md Debugging & Quality section

---

### `/memory-sync`

**Purpose:** Documentation synchronization patterns for platform-specific files

**Contains:**
- Central documentation (AGENTS.md) concept
- Platform-specific file generation
- Sync checking and validation
- Automated consistency workflows
- Single source of truth patterns
- Multi-platform maintenance

**Use this when:**
- You've updated AGENTS.md and need to sync to other files
- Checking if documentation is in sync
- Setting up synchronization for a new project
- Understanding documentation architecture
- Maintaining consistency across Claude, Gemini, Codex docs

**Related reading:** AGENTS.md Documentation Synchronization, README.md Links section

---

## How to Use Skills

### Listing Available Skills

```bash
# See all skills in Claude Code
/help
```

### Invoking a Skill

```bash
# Simple invocation
/orchestrator-directives

# With context (optional)
/code-quality --for-python-projects
```

### Progressive Disclosure

Skills follow a **progressive disclosure model**:

1. **Quick summary** - Start with the basics
2. **Decision guide** - Help you make choices
3. **Examples** - Real-world usage patterns
4. **Details** - Deep technical information

You don't need to read everything at once. Start with what you need, dive deeper as needed.

---

## Skill Integration with Wipnote

### Tracking Your Skill Usage

When you use a skill to solve a problem, document it in Wipnote:

```bash
# Create a spike documenting your learnings
wipnote spike create "Learned debugging workflow - resolved hook loading issue: Research first, check all hook sources, hooks MERGE not replace, verify with /hooks. Fixed by removing duplicate hooks."
```

### Orchestrator Directives Integration

When delegating complex tasks, reference the orchestrator skill:

```python
# Orchestrator coordinates based on skill guidance
Task(
    subagent_type="general-purpose",
    prompt="""Using /orchestrator-directives skill guidance:

    Task: Run comprehensive test suite in parallel
    Scope: tests/unit/, tests/integration/, tests/e2e/

    Success criteria: Report pass/fail counts only
    Time limit: 10 minutes total
    """
)
```

### Code Quality Gates

Integrate code quality skill with deployment:

```bash
# Before deploying, ensure all quality gates pass
# (Follows /code-quality skill workflow)
uv run ruff check --fix && \
uv run ruff format && \
uv run mypy src/ && \
uv run pytest

# Only then deploy
./scripts/deploy-all.sh 0.9.4
```

---

## Examples

### Example 1: Planning a Feature with Orchestrator Directives

**Scenario:** You need to implement a complex authentication system

```bash
# Step 1: Use orchestrator directives to plan
/orchestrator-directives

# Understand:
# - Should I delegate work?
# - What sessions will be created?
# - How will costs be optimized?

# Step 2: Create feature tracking
# (Use Wipnote SDK as shown in delegation.md)

# Step 3: Execute with proper delegation pattern
# Task(subagent_type="...", prompt="...")
```

### Example 2: Debugging a Mysterious Error

**Scenario:** Your tests are failing with an unclear error

```bash
# Step 1: Use debugging workflow
/debugging-workflow

# Learn:
# - Research first methodology
# - Systematic error analysis
# - When to use debugging agents

# Step 2: Document findings in spike
wipnote spike create "Debug: test failure in X - [findings here]"

# Step 3: Don't guess - research and understand root cause
```

### Example 3: Release Workflow

**Scenario:** You need to publish version 0.10.0

```bash
# Step 1: Ensure quality with /code-quality
/code-quality

# Step 2: Follow deployment automation
/deployment-automation

# Step 3: Execute steps in order
./scripts/deploy-all.sh 0.10.0

# Step 4: Verify publication
# Check PyPI, update plugins as documented
```

---

## Quick Reference Table

| Need | Skill | Reference |
|------|-------|-----------|
| Delegate work? | `/orchestrator-directives` | [Delegation Guide](delegation.md) |
| Choose model | `/multi-ai-orchestration` | AGENTS.md Orchestrator section |
| Fix tests/lints | `/code-quality` | code-hygiene.md rules |
| Release version | `/deployment-automation` | deployment.md rules |
| Diagnose error | `/debugging-workflow` | debugging.md rules |
| Sync docs | `/memory-sync` | AGENTS.md Sync section |

---

## FAQs

**Q: How do I know which skill to use?**

A: Use the decision tree at the top of this guide. It walks through common scenarios and points you to the right skill.

**Q: Can I use multiple skills for one task?**

A: Yes! For example, planning a release might use `/deployment-automation` + `/code-quality` + `/orchestrator-directives` for delegating tests.

**Q: Where are skills defined?**

A: Skills are defined in `packages/claude-plugin/skills/` in the Wipnote project.

**Q: Do I need to memorize all skills?**

A: No! Skills are discoverable. Use `/help` to list them, and this guide to understand what each does.

**Q: Can I create custom skills?**

A: Yes! See `packages/claude-plugin/` for the skill development framework.

---

## Related Reading

- [Delegation Guide](delegation.md) - Deep dive on Task() and orchestration
- [AGENTS.md - Orchestrator Mode](../AGENTS.md#orchestrator-mode) - Overview and quick start
- [CLAUDE.md - Skills Reference](../CLAUDE.md#skills-reference) - Project-specific skill guidance
- `.claude/rules/orchestration.md` - Complete orchestrator directives
- `packages/claude-plugin/skills/` - Skill definitions and implementations

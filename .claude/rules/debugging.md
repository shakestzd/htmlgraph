# Debugging Workflow - Research First

**CRITICAL: HtmlGraph enforces a research-first debugging philosophy.**

## Core Principle

**NEVER implement solutions based on assumptions. ALWAYS research documentation first.**

This principle emerged from dogfooding HtmlGraph development. We repeatedly violated it by:
- ❌ Making multiple trial-and-error attempts before researching
- ❌ Implementing "fixes" based on guesses instead of documentation
- ❌ Not using available debugging tools and agents

**The correct approach:**
1. ✅ **Research** - Use claude-code-guide agent, read documentation
2. ✅ **Understand** - Identify root cause through evidence
3. ✅ **Implement** - Apply fix based on understanding
4. ✅ **Validate** - Test to confirm fix works
5. ✅ **Document** - Capture learning in HtmlGraph spike

## Debugging Agents (Plugin-Provided)

HtmlGraph plugin includes three specialized agents for systematic debugging:

### 1. Researcher Agent
**Purpose**: Research documentation BEFORE implementing solutions

**Use when**:
- Encountering unfamiliar errors or behaviors
- Working with Claude Code hooks, plugins, or configuration
- Before implementing solutions based on assumptions
- When multiple attempted fixes have failed

**Workflow**:
```bash
# Activate researcher agent
# Use claude-code-guide for Claude-specific questions
# Document findings in HtmlGraph spike
```

**Key resources**:
- Claude Code docs: https://code.claude.com/docs
- GitHub issues: https://github.com/anthropics/claude-code/issues
- Hook documentation: https://code.claude.com/docs/en/hooks.md

### 2. Debugger Agent
**Purpose**: Systematically analyze and resolve errors

**Use when**:
- Error messages appear but root cause is unclear
- Behavior doesn't match expectations
- Tests are failing
- Hooks or plugins aren't working as expected

**Built-in debug tools**:
```bash
claude --debug <command>        # Verbose output
/hooks                          # List all active hooks
/hooks PreToolUse              # Show specific hook type
/doctor                         # System diagnostics
claude --verbose               # More detailed logging
```

**Methodology**:
1. Gather evidence (logs, error messages, stack traces)
2. Reproduce consistently (exact steps, minimal case)
3. Isolate variables (test one change at a time)
4. Analyze context (what changed recently?)
5. Form hypothesis (root cause theory)
6. Test hypothesis (validate or refute)
7. Implement fix (minimal change to fix root cause)

### 3. Test Runner Agent
**Purpose**: Automatically test changes, enforce quality gates

**Use when**:
- After implementing code changes
- Before marking features/tasks complete
- After fixing bugs
- Before committing code

**Test commands**:
```bash
# Full quality gate (pre-commit)
(cd packages/go && go build ./... && go vet ./... && go test ./...)
```

## Debugging Workflow Pattern

**Example: Duplicate Hooks Issue**

**❌ What we did initially (wrong)**:
1. Removed .claude/hooks/hooks.json - Still broken
2. Cleared plugin cache - Still broken
3. Removed old plugin versions - Still broken
4. Removed marketplaces symlink - Still broken
5. Finally researched documentation
6. Found root cause: Hook merging behavior

**✅ What we should have done (correct)**:
1. Research Claude Code hook loading behavior first
2. Use claude-code-guide agent to understand hook merging
3. Identify that hooks from multiple sources MERGE, not replace
4. Check all hook sources (.claude/settings.json, plugin hooks)
5. Remove duplicates based on understanding
6. Verify fix works
7. Document learning in spike

## HtmlGraph Debug Commands

```bash
# Check orchestrator status
htmlgraph orchestrator status

# List active features
htmlgraph status

# View specific feature
htmlgraph feature show <id>

# Check session state
htmlgraph session list --active
```

## Integration with Orchestrator Mode

When orchestrator mode is enabled (strict), you'll receive reflections after direct tool execution:

```
ORCHESTRATOR REFLECTION: You executed code directly.

Ask yourself:
- Could this have been delegated to a subagent?
- Would parallel Task() calls have been faster?
- Is a work item tracking this effort?
```

This encourages delegation to specialized agents (researcher, debugger, test-runner) for systematic problem-solving.

## Documentation References

**For debugging agents**: See `packages/go-plugin/agents/`
- `researcher.md` - Research-first methodology
- `debugger.md` - Systematic error analysis
- `test-runner.md` - Quality gates and testing

**For debugging workflows**: See `.htmlgraph/spikes/`
- Spikes document research findings and debugging processes
- Learn from past debugging sessions
- Avoid repeating the same mistakes

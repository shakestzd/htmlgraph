# Claude Code Hooks in Subagent Contexts - Research Findings

**Research Date**: January 8, 2026
**Status**: Complete research documented in Wipnote spike
**Critical Finding**: Hook propagation to subagents is NOT supported by design.

---

## Executive Summary

This research answers the critical question: **Do Claude Code hooks fire in subagent/delegated Task() contexts?**

**Answer: NO.** PreToolUse and PostToolUse hooks do NOT fire in Task() subagents. This is an intentional architectural decision to prevent settings pollution and recursive loops.

### Impact for Wipnote
- ✅ Main orchestrator session activity captured via PreToolUse hooks
- ❌ Task() delegated subagent activity NOT captured (no hook events)
- ⚠️ Subagent completion tracked via SubagentStop (but limited info)
- **Result**: Dashboard captures orchestrator activity, misses delegated work

---

## Key Findings

### 1. Hook Scope is Session-Specific

**Hooks that WORK in main session:**
- ✅ PreToolUse - Fires before tool execution
- ✅ PostToolUse - Fires after tool execution
- ✅ Stop - Fires when agent stops
- ✅ SubagentStop - Fires when subagent completes
- ✅ UserPromptSubmit, SessionStart, SessionEnd - Lifecycle events

**Hooks that DON'T fire in Task() subagents:**
- ❌ PreToolUse - Does NOT fire in subagent
- ❌ PostToolUse - Does NOT fire in subagent
- ❌ No equivalent for subagent tool activity

### 2. Why Hooks Don't Propagate to Subagents

**Design Rationale**: Prevent three serious problems:

1. **Recursive Loops**
   - PostToolUse hook calls Claude
   - Claude spawns Task() subagent
   - Hook fires again in subagent
   - Infinite recursion

2. **Settings Pollution**
   - Parent's security hooks affect child unexpectedly
   - Parent's tool matchers don't make sense in child context
   - Accidental behavior override

3. **Context Contamination**
   - Parent's configuration leaks to child
   - Child inherits irrelevant settings
   - Isolation needed for safety

**Solution Used**: Subagents run in isolated contexts with separate configuration via `--settings` flag.

### 3. Current SubagentStop Hook (Limited)

Only hook that fires for subagent events:
```json
{
  "hooks": {
    "SubagentStop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "log-subagent-completion.sh"
          }
        ]
      }
    ]
  }
}
```

**Problem**: Hook input doesn't distinguish between subagents:
```json
{
  "session_id": "abc123",           // SAME for all subagents
  "hook_event_name": "SubagentStop"
  // NO agent_id, parent_id, or subagent_type fields
}
```

**Missing fields** (proposed in issue #14859):
- `agent_id` - Unique ID for each subagent
- `parent_agent_id` - Parent session ID
- `subagent_type` - Agent type (e.g., "Explore", "Review")

### 4. Environment Variables Available

**All hooks can access:**
- `$CLAUDE_PROJECT_DIR` - Project root directory
- `$CLAUDE_CODE_REMOTE` - "true" if web, unset if CLI
- Standard shell environment variables

**SessionStart hooks only:**
- `$CLAUDE_ENV_FILE` - Path to persist environment variables

**NOT Available:**
- ❌ `CLAUDE_HOOKS` - Configuration doesn't exist
- ❌ `CLAUDE_PROPAGATE_HOOKS` - No propagation mechanism
- ❌ No hook inheritance controls

---

## GitHub Issues Documentation

### Critical Issues (Blocking Full Solution)

| Issue | Title | Status | Impact |
|-------|-------|--------|--------|
| [#6305](https://github.com/anthropics/claude-code/issues/6305) | PreToolUse/PostToolUse Hooks Not Executing | Under investigation | Tool hooks not firing in some contexts |
| [#14859](https://github.com/anthropics/claude-code/issues/14859) | Agent Hierarchy in Hook Events + SubagentStart Hook | Feature proposal | Needed to distinguish subagents |
| [#10354](https://github.com/anthropics/claude-code/issues/10354) | Sub-Agent Support in Hooks | Closed (duplicate) | No explicit subagent invocation |
| [#7881](https://github.com/anthropics/claude-code/issues/7881) | SubagentStop Cannot Identify Subagent | Acknowledged | All subagents share session_id |

### Secondary Issues

- **#3148** - PreToolUse/PostToolUse not triggered with `*` matcher (regex edge case)
- **#15617** - PostToolUse hooks not firing on Termux/Android (platform-specific)

---

## Recommended Solutions for Wipnote

### Option A: Current Best Practice (Immediate)

**Use SubagentStop + Manual Logging**

```python
# In .claude/settings.json
{
  "hooks": {
    "SubagentStop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/record-subagent-completion.sh"
          }
        ]
      }
    ]
  }
}
```

**Limitations**:
- Only logs completion, not individual tool calls
- Can't distinguish between multiple subagents
- Incomplete activity tracking

### Option B: Wait for GitHub Issue #14859 (Future)

When Anthropic implements hook enhancements:
- `agent_id` field to identify each subagent
- `SubagentStart` hook for spawn events
- Full parent-child relationship tracing
- Complete multi-agent observability

**Timeline**: Unknown (under review, not in active development)

### Option C: Hybrid Approach (Practical Now)

1. **Keep PreToolUse hooks** in main session (captures orchestrator)
2. **Add SubagentStop hooks** to track completion
3. **Document limitation** in dashboard UI
4. **Reference GitHub issue** #14859 in code
5. **Plan migration** when hooks support agent hierarchy

### Option D: Alternative - SDK Event Logging

Instead of relying on hooks for subagent activity:
- Use Wipnote SDK to log explicit operations
- Track features, tracks, spikes created
- Log Task() invocations programmatically
- Build observability outside hook system

**Best for**: Non-tool operations, explicit event capture

---

## Implementation Recommendation for Wipnote

### Short Term (Now)
```python
# Document in code with GitHub issue reference
# LIMITATION: PreToolUse hooks don't fire in Task() subagents
# See: https://github.com/anthropics/claude-code/issues/14859
# Awaiting: SubagentStart hook + agent_id fields in hook input
```

### Dashboard UI Update
- ✅ Show orchestrator tool activity
- ⚠️ Show "Subagent activity not tracked (Claude Code limitation)"
- 🔗 Link to GitHub issue #14859
- 📅 Plan migration when supported

### Long Term (When #14859 Implemented)
- Migrate to SubagentStart hook for spawn events
- Use agent_id to track individual subagents
- Capture parent-child relationships
- Full multi-agent activity graph

---

## Why This Matters for Observability

### Current Architecture Gap

```
Main Session (Orchestrator)
├── PreToolUse: Bash → CAPTURED ✅
├── PostToolUse: Read → CAPTURED ✅
├── Task() Spawn → Subagent
│   └── PreToolUse: Bash → NOT CAPTURED ❌
│   └── PostToolUse: Write → NOT CAPTURED ❌
└── SubagentStop → CAPTURED (but no agent_id) ⚠️
```

### Result
Wipnote dashboard shows:
- All orchestrator tool calls
- Subagent start/stop
- BUT: Individual subagent tool calls invisible

### Future (If #14859 Implemented)

```
Main Session (Orchestrator)
├── PreToolUse: Bash → CAPTURED ✅
├── Task() Spawn → Agent-456
│   ├── SubagentStart (agent_id=456) → CAPTURED ✅
│   ├── PreToolUse: Bash → CAPTURED ✅
│   ├── PreToolUse: Write → CAPTURED ✅
│   └── SubagentStop (agent_id=456) → CAPTURED ✅
```

**Impact**: Complete visibility into all work (orchestrator + delegated)

---

## Design Pattern Insights

### How Other Tools Handle This

**GitHub Actions**: Jobs run in isolated contexts
- No automatic workflow propagation between jobs
- Explicit configuration needed for each job

**Kubernetes**: Pods are isolated units
- Hooks only apply to their namespace
- No cross-pod hook inheritance

**AWS Lambda**: Functions are independent
- Parent layers don't propagate to async invocations
- Must explicitly configure each function

**Pattern**: Isolation with explicit propagation prevents accidental interference.

---

## Key Takeaways

1. **Hooks are session-scoped** - They don't propagate to Task() subagents by design
2. **Why not?** - Prevents recursive loops, settings pollution, and context contamination
3. **Current limitation** - Wipnote can't capture delegated task activity
4. **Workaround** - SubagentStop hook captures completion (limited info)
5. **Future solution** - GitHub issue #14859 proposes agent hierarchy fields
6. **Timeline** - Unknown (under review by Anthropic)
7. **Best practice** - Document limitation, reference issue, plan migration
8. **Alternative** - Use SDK event logging for explicit operations

---

## References

### Official Documentation
- [Claude Code Hooks Reference](https://code.claude.com/docs/en/hooks)
- [Claude Code Hooks Guide](https://code.claude.com/docs/en/hooks-guide)
- [Claude Code SubAgents](https://code.claude.com/docs/en/sub-agents)

### Critical GitHub Issues
- [#6305 - PreToolUse/PostToolUse Hooks Not Executing](https://github.com/anthropics/claude-code/issues/6305)
- [#14859 - Agent Hierarchy in Hook Events](https://github.com/anthropics/claude-code/issues/14859)
- [#10354 - Sub-Agent Support in Hooks](https://github.com/anthropics/claude-code/issues/10354)
- [#7881 - SubagentStop Cannot Identify Subagent](https://github.com/anthropics/claude-code/issues/7881)

### Community Resources
- [egghead.io - Avoid Settings Pollution in Subagents](https://egghead.io/avoid-the-dangers-of-settings-pollution-in-subagents-hooks-and-scripts~xrecv)
- [GitHub - claude-code-hooks-mastery](https://github.com/disler/claude-code-hooks-mastery)
- [ClaudeLog - Hooks Documentation](https://claudelog.com/mechanics/hooks/)

### Articles & Guides
- [Best Practices for Claude Code Subagents - PubNub Blog](https://www.pubnub.com/blog/best-practices-for-claude-code-sub-agents/)
- [Guide to Claude Code Subagents & Hooks - ArsTurn](https://www.arsturn.com/blog/a-beginners-guide-to-using-subagents-and-hooks-in-claude-code)
- [DEV Community - Claude Code Subagents and Task Delegation](https://dev.to/letanure/claude-code-part-6-subagents-and-task-delegation-k6f)
- [Medium - Mastering Main Agent and Sub-Agents](https://jewelhuq.medium.com/practical-guide-to-mastering-claude-codes-main-agent-and-sub-agents-fd52952dcf00)

---

## Conclusion

**Hook propagation to subagents is not supported by design.** This is an intentional architectural decision by Anthropic to prevent settings pollution and recursive loops. Wipnote's event capture is currently limited to the main orchestrator session.

Full multi-agent observability requires the enhancement proposed in GitHub issue #14859 (SubagentStart hook + agent_id/parent_id fields), which is currently under review but not yet implemented.

**For immediate use**: Document the limitation in the dashboard and plan migration when hooks support agent hierarchy fields. Consider using SubagentStop + manual event logging for partial observability of delegated work.

---

**Research Spike**: `.wipnote/spikes/` contains full detailed findings with code examples and implementation guidance.

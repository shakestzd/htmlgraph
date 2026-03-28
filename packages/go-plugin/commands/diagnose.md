---
name: diagnose
description: Diagnose orchestrator delegation enforcement gaps in the current session
allowed-tools: ["Bash", "Read"]
user_invocable: true
---

<!-- Efficiency: SDK calls: 1, Bash calls: 1, Context: ~8% -->

# /htmlgraph:diagnose

Audit delegation compliance for the current session and identify enforcement gaps.

## Usage

```
/htmlgraph:diagnose
```

## Examples

```
/htmlgraph:diagnose
```
Analyze current session and show delegation score, gaps, and recommendations.

## Instructions for Claude

Run the delegation diagnostic and present a structured report.

### Implementation

```bash
# 1. Get orchestrator and session state
htmlgraph status
htmlgraph session list --limit 1

# 2. Query current session events (delegation audit)
htmlgraph analytics summary
htmlgraph analytics summary
```

Parse the output to identify:
- Current session ID (from `htmlgraph session list`)
- Active features and their status
- Any bottlenecks or risks

Compute a delegation score based on observed tool usage in the current conversation:
- Count direct `Edit`/`Write`/`Bash` calls vs `Task`/`Agent` delegations
- `score = delegations / (delegations + direct_impl + git_writes) * 100`

### Output Format

Present the results as:

```markdown
## Delegation Diagnostic Report

### Orchestrator State
- Mode: {enabled/disabled}
- Enforcement: {strict/guidance}
- Violations: {N}/3
- Circuit breaker: {triggered/normal}

### Delegation Score: {score}% ({delegations}/{total} actions delegated)

### Gaps Found

#### Git Write Operations (should use /htmlgraph:copilot)
| Time | Command | Recommended |
|------|---------|-------------|
| HH:MM | git commit ... | /htmlgraph:copilot |

#### Direct Implementation (should delegate to agent)
| Time | Tool | Summary | Recommended |
|------|------|---------|-------------|
| HH:MM | Edit | file.py | Task("htmlgraph:sonnet-coder", ...) |

### Recommendations
{Numbered list of specific actions based on gaps found}
```

If no gaps: report "Delegation score: 100%. No enforcement gaps found in this session."

If no session data: report "No events found. Verify hooks are active with `/hooks`."

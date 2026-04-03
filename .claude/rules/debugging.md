---
paths:
  - "cmd/**"
  - "internal/**"
  - "plugin/**"
---

# Debugging Workflow - Research First

**Core principle: research first, implement second. NEVER guess; always verify.**

## Debug Commands

```bash
claude --debug <command>   # Verbose output
/hooks                     # List all active hooks
/doctor                    # System diagnostics
```

## Agents

- **researcher** — Research documentation BEFORE implementing solutions
- **debugger** — Systematic error analysis and root cause investigation
- **test-runner** — Quality gates; run after every code change

## Workflow

1. Research (docs, agents) — form a hypothesis from evidence
2. Implement the minimal fix
3. Validate with `go build ./... && go vet ./... && go test ./...`

For full methodology, use `/htmlgraph:diagnose`.

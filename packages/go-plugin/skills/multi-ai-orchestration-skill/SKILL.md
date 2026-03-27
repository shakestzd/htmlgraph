---
name: multi-ai-orchestration
description: Spawner selection, cost optimization, and HeadlessSpawner patterns for coordinating multiple AI models (Codex, Gemini, Copilot, Claude).
---

# Multi-AI Orchestration Skill

Use this skill for spawner selection, cost optimization, and HeadlessSpawner patterns. MUST coordinate multiple AI models efficiently.

**Trigger keywords:** spawner, multi-ai, headless, codex, gemini, copilot, model selection, cost optimization, parallel agents

---

## CRITICAL: Cost-First Routing (IMPERATIVE)

**Claude Code is EXPENSIVE and has usage limits. You MUST use FREE/CHEAP AIs first.**

```
COST HIERARCHY (lowest to highest):
1. Gemini 2.0-Flash: FREE (2M tokens/min) ← USE FIRST
2. Codex (GPT-4):    $ (cheap, code-specialized)
3. Copilot:          $ (cheap, GitHub integration)
4. Claude Haiku:     $$ (fallback ONLY)
5. Claude Sonnet:    $$$ (coordination only)
6. Claude Opus:      $$$$ (strategic decisions only)
```

### PRE-DELEGATION CHECKLIST (MUST EXECUTE)

**BEFORE delegating ANY task, you MUST ask these questions IN ORDER:**

```
┌─────────────────────────────────────────────────────────┐
│ 1. Can Gemini do this? (exploration, research, batch)  │
│    → YES = MUST use spawn_gemini (FREE)                │
│    → NO  = Continue to step 2                          │
├─────────────────────────────────────────────────────────┤
│ 2. Is this code work? (implementation, fixes, tests)   │
│    → YES = MUST use spawn_codex (cheap, specialized)   │
│    → NO  = Continue to step 3                          │
├─────────────────────────────────────────────────────────┤
│ 3. Is this git/GitHub? (commits, PRs, issues)          │
│    → YES = MUST use spawn_copilot (GitHub integration) │
│    → NO  = Continue to step 4                          │
├─────────────────────────────────────────────────────────┤
│ 4. Does this require deep reasoning?                   │
│    → YES = Use Claude Opus (expensive, but needed)     │
│    → NO  = Continue to step 5                          │
├─────────────────────────────────────────────────────────┤
│ 5. Is this multi-agent coordination?                   │
│    → YES = Use Claude Sonnet (mid-tier)                │
│    → NO  = Use Gemini (FREE) or Haiku (fallback)       │
└─────────────────────────────────────────────────────────┘
```

### WRONG vs CORRECT Delegation

```
❌ WRONG (NEVER use Haiku for everything):
- Implementation → Haiku          # WRONG: MUST use Codex
- Git commits → Haiku             # WRONG: MUST use Copilot
- Code generation → Haiku         # WRONG: MUST use Codex
- Research → Haiku                # WRONG: MUST use Gemini (FREE!)
- File analysis → Haiku           # WRONG: MUST use Gemini (FREE!)

✅ CORRECT (ALWAYS use cost-first routing):
- Implementation → spawn_codex    # MUST use: Cheap, code-specialized
- Git commits → spawn_copilot     # MUST use: Cheap, GitHub integration
- Research → spawn_gemini         # MUST use: FREE, high context
- File analysis → spawn_gemini    # MUST use: FREE, multimodal
- Strategic planning → Opus       # Use when needed: Expensive, but needed
- Haiku → FALLBACK ONLY           # ONLY when others fail
```

---

## Task-to-AI Routing Table (IMPERATIVE)

| Task Type | MUST Use | Fallback | Why |
|-----------|----------|----------|-----|
| Exploration, research, codebase analysis | **spawn_gemini** | Haiku | FREE, 2M tokens/min, high context |
| Code generation, implementation | **spawn_codex** | Sonnet | Code-specialized, sandbox isolation |
| Bug fixes, refactoring | **spawn_codex** | Haiku | Edit tracking, workspace-write |
| Git operations, commits, PRs | **spawn_copilot** | Haiku | GitHub integration, tool permissions |
| File operations, batch processing | **spawn_gemini** | Haiku | FREE, fast, multimodal |
| Image/screenshot analysis | **spawn_gemini** | - | Vision API, multimodal |
| Testing, validation | **spawn_codex** | Haiku | Can execute tests in sandbox |
| Strategic planning, architecture | **Opus** | Sonnet | Deep reasoning required |
| Multi-agent coordination | **Sonnet** | - | Complex coordination |
| Last resort fallback | **Haiku** | - | When Gemini/Codex/Copilot fail |

---

## Cost Awareness (CRITICAL)

```
MONTHLY USAGE IMPACT:

Claude Code (Sonnet/Opus): $$$$
- Limited usage quota
- Exhausts quickly with heavy use
- RESERVE for strategic work only

Gemini 2.0-Flash: FREE
- 2M tokens per minute (rate limited)
- 1M token context window
- Multimodal (images, PDFs, audio)
- Use FIRST for exploration

Codex (GPT-4): $
- Cheap for code work
- Sandbox isolation
- Worth premium for specialization

Copilot: $
- Cheap for GitHub work
- Tool permission controls
- Native GitHub integration
```

### Cost Optimization Impact

```
BEFORE (using Haiku everywhere):
- 10 implementations × Haiku = $$$$
- 5 git commits × Haiku = $$$
- 20 file searches × Haiku = $$$$$

AFTER (cost-first routing):
- 10 implementations × Codex = $$
- 5 git commits × Copilot = $
- 20 file searches × Gemini = FREE

SAVINGS: 80-90% reduction in Claude Code usage
```

---

## Spawner Selection Matrix

**Priority order (first match wins, cost-first):**

| Priority | Use Case | Spawner | Cost |
|----------|----------|---------|------|
| 1 | Exploration, research, batch ops | `spawn_gemini` | FREE |
| 2 | Code generation, bug fixes | `spawn_codex` | $ |
| 3 | Git/GitHub workflows, PRs | `spawn_copilot` | $ |
| 4 | Image/multimodal analysis | `spawn_gemini` | FREE |
| 5 | Complex reasoning, architecture | `spawn_claude` | $$$$ |
| 6 | Fallback when others fail | `Task(haiku)` | $$ |

## Decision Aid

- **"Is this exploratory?"** → MUST use `spawn_gemini` (FREE)
- **"Is this about code?"** → MUST use `spawn_codex` (cheap)
- **"Does this involve git?"** → MUST use `spawn_copilot` (cheap)
- **"Do I need vision?"** → MUST use `spawn_gemini` (FREE)
- **"Is deep reasoning critical?"** → Use `spawn_claude` (expensive)
- **"Everything else"** → ALWAYS use `spawn_gemini` FIRST, then Haiku fallback

## Task() vs spawn_*() Decision

**Use spawn_*() (PRIMARY):**
- Work can run in isolation (most cases)
- MUST optimize cost (Gemini FREE)
- Specialized tool needed (Codex sandbox, Copilot GitHub)

**Use Task(haiku) (FALLBACK ONLY):**
- Work depends on conversation context
- Cache hits matter (same conversation)
- **AND** spawn_*() has failed or is unavailable

---

## Integration Patterns

### Pattern 1: Cost-First Exploration
```python
# ALWAYS start with Gemini for exploration
result = spawn_gemini("Search codebase for all auth patterns")
if not result.success:
    # Fallback to Haiku ONLY if Gemini fails
    Task(prompt="Search codebase for auth patterns", subagent_type="haiku")
```

### Pattern 2: Code Implementation
```python
# Use Codex for code work (not Haiku!)
result = spawn_codex(
    prompt="Implement OAuth authentication",
    sandbox="workspace-write"
)
if not result.success:
    Task(prompt="Implement OAuth", subagent_type="sonnet")  # Fallback
```

### Pattern 3: Git Workflow
```python
# Use Copilot for git (not Haiku!)
result = spawn_copilot(
    prompt="Commit changes and create PR",
    allow_tools=["shell(git)", "github(*)"]
)
```

### Pattern 4: Multi-Provider (Cost-Optimized)
```python
# Research with FREE Gemini
research = spawn_gemini("Analyze current auth implementation")

# Code with cheap Codex
code = spawn_codex("Implement OAuth based on research")

# Git with cheap Copilot
pr = spawn_copilot("Create PR for OAuth implementation")

# Reserve Claude for strategic decisions ONLY
# architecture = spawn_claude("Design long-term auth strategy")
```

## Cost Optimization Rules (IMPERATIVE)

1. **ANY exploratory work** → MUST use `spawn_gemini` (FREE)
2. **ANY code work** → MUST use `spawn_codex` (cheap, specialized)
3. **ANY git/GitHub work** → MUST use `spawn_copilot` (cheap, integrated)
4. **Complex reasoning** → MAY use `spawn_claude` (expensive)
5. **Haiku** → ONLY as fallback when above fail

**Violating these rules wastes Claude Code quota unnecessarily.**

---

## Verification After Spawning

After Gemini/Codex generates code, ALWAYS verify quality:

```bash
# MUST run quality verification script
./scripts/test-quality.sh src/path/to/file.py

# Returns: exit code 0 (pass) or 1 (fail)
# Runs: ruff check, ruff format, mypy, pytest
```

If verification fails, MUST iterate with the same spawner (NEVER Claude).

---

**For detailed API documentation:** → See [REFERENCE.md](./REFERENCE.md)
**For real-world examples:** → See [EXAMPLES.md](./EXAMPLES.md)

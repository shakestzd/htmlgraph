# Multi-AI Orchestration - Complete Reference

## COST-FIRST ROUTING (IMPERATIVE)

**Before using any spawner, MUST follow this decision tree:**

```
┌──────────────────────────────────────────────────────────────┐
│ COST-FIRST ROUTING CHECKLIST                                 │
│                                                              │
│ 1. Is this exploration/research/batch work?                 │
│    → MUST use spawn_gemini (FREE)                           │
│                                                              │
│ 2. Is this code generation/fixes/tests?                     │
│    → MUST use spawn_codex (cheap, specialized)              │
│                                                              │
│ 3. Is this git/GitHub work?                                 │
│    → MUST use spawn_copilot (cheap, integrated)             │
│                                                              │
│ 4. Does this REQUIRE deep reasoning?                        │
│    → MAY use spawn_claude (expensive)                       │
│                                                              │
│ 5. Everything else?                                         │
│    → spawn_gemini FIRST (FREE), Haiku fallback              │
└──────────────────────────────────────────────────────────────┘
```

## Cost Hierarchy

| Tier | Spawner | Cost | Use Case |
|------|---------|------|----------|
| FREE | spawn_gemini | $0 | Exploration, research, batch ops, multimodal |
| $ | spawn_codex | Low | Code generation, fixes, tests, refactoring |
| $ | spawn_copilot | Low | Git operations, GitHub workflows |
| $$ | Task(haiku) | Medium | Fallback ONLY when above fail |
| $$$ | Task(sonnet) | High | Multi-agent coordination |
| $$$$ | spawn_claude | Very High | Strategic architecture, complex reasoning |

---

## HeadlessSpawner API

### spawn_gemini (USE FIRST - FREE!)

**Purpose:** Exploration, research, batch operations, multimodal analysis

**Cost:** FREE (2M tokens/minute rate limit)

**Configuration:**
```python
# Delegate to gemini-operator agent (tries Gemini CLI first)
Task(
    subagent_type="htmlgraph:gemini-operator",
    prompt="Search codebase for all auth patterns",
)
```

**Features:**
- FREE tier with 2M tokens/minute
- 1M token context window
- Vision API for image analysis
- Multimodal (images, PDFs, audio)
- Fastest response times

**MUST use for:**
- Codebase exploration and research
- File searching and analysis
- Batch operations over many files
- Document/image analysis
- Any exploratory work before implementation

### spawn_codex (USE FOR CODE - CHEAP)

**Purpose:** Code generation, bug fixes, workspace edits

**Cost:** $ (cheap, code-specialized)

**Configuration:**
```python
# Delegate to codex-operator agent (tries Codex CLI first)
Task(
    subagent_type="htmlgraph:codex-operator",
    prompt="Implement OAuth authentication endpoint",
)
```

**Sandbox modes:**
- `workspace-write` - Auto-approve code edits
- `workspace-read` - Read-only access
- `network` - Allow network operations

**MUST use for:**
- Implementing features
- Fixing bugs
- Refactoring code
- Writing tests
- Any code generation work

### spawn_copilot (USE FOR GIT - CHEAP)

**Purpose:** Git operations, GitHub workflows

**Cost:** $ (cheap, GitHub-integrated)

**Configuration:**
```python
# Delegate to copilot-operator agent (tries Copilot CLI first)
Task(
    subagent_type="htmlgraph:copilot-operator",
    prompt="Commit changes and create PR",
)
```

**Tool permissions:**
- `shell(git)` - Git command access
- `read(*.py)` - File read access
- `github(*)` - GitHub API access

**MUST use for:**
- Git commits and pushes
- PR creation and review
- Branch management
- GitHub issue management
- Any git/GitHub workflow

### spawn_claude (EXPENSIVE - STRATEGIC ONLY)

**Purpose:** Complex reasoning, architecture, design

**Cost:** $$$$ (very high - use sparingly)

**Configuration:**
```python
# Delegate to opus-coder agent for deep reasoning
Task(
    subagent_type="htmlgraph:opus-coder",
    prompt="Design scalable notification system",
)
```

**Permission modes:**
| Mode | Description |
|------|-------------|
| `bypassPermissions` | Auto-approve all |
| `acceptEdits` | Auto-approve code edits only |
| `dontAsk` | Fail on any permission |
| `plan` | Generate plan without executing |
| `delegate` | Balanced safety + autonomy |

**ONLY use for:**
- System architecture decisions
- Complex multi-domain analysis
- Strategic planning
- Deep reasoning that other AIs cannot handle

## Spawner Comparison Table (Updated with Costs)

| Spawner | Cost Tier | Price | Speed | Primary Use |
|---------|-----------|-------|-------|-------------|
| `spawn_gemini` | FREE | $0 | Fast | Exploration, research, batch |
| `spawn_codex` | $ | Low | Medium | Code generation, fixes |
| `spawn_copilot` | $ | Low | Medium | Git/GitHub operations |
| `spawn_claude` | $$$$ | High | Slow | Strategic reasoning only |

---

## Enforcement Mechanism

### Pre-Delegation Validation

Before any delegation, validate agent selection:

| Task type | Required agent | CLI |
|-----------|---------------|-----|
| exploration, research, batch | `htmlgraph:gemini-operator` | FREE |
| code generation, bug fix, testing | `htmlgraph:codex-operator` | $ |
| git commit, push, PR | `htmlgraph:copilot-operator` | $ |
| architecture, strategic planning | `htmlgraph:opus-coder` | $$$$ |

### Cost Tracking

Track delegation usage for cost analysis:

```bash
# After completing delegations, record in a spike
htmlgraph spike create "Delegation Summary: [task description] — used [agent], cost tier [tier]"
```

### Verification After Spawning

After Gemini/Codex generates code, MUST verify:

```bash
# Quick verification (fast)
./scripts/verify-code.sh src/path/to/file.py

# Full quality check (thorough)
./scripts/test-quality.sh src/path/to/file.py

# If verification fails:
# 1. Iterate with SAME spawner (not Claude)
# 2. Only escalate if 3+ failures
```

## HtmlGraph Integration

Track all spawned work:

```bash
# Create tracked feature
htmlgraph feature create "Implement OAuth"
htmlgraph feature start <feat-id>
```

```python
# Delegate with tracking (dispatch all in parallel in single message)
Task(
    subagent_type="htmlgraph:codex-operator",
    prompt="Implement OAuth: Add JWT tokens to API endpoints"
)
```

```bash
# Save findings in a spike
htmlgraph spike create "Orchestration: Implement OAuth — completed"
```

## Parallel Coordination Pattern

```python
# Spawn all independent tasks in a single message (true parallel execution)
Task(subagent_type="htmlgraph:codex-operator", prompt="Implement auth: Add JWT tokens...")
Task(subagent_type="htmlgraph:sonnet-coder", prompt="Write tests: unit + integration for auth...")
Task(subagent_type="htmlgraph:gemini-operator", prompt="Update docs: auth patterns and API reference...")
```

## Anti-Patterns to Avoid

**1. Using spawn_claude for simple queries**
```python
# BAD - expensive for simple work
spawn_claude("Search for all TODO comments")

# GOOD - cheap and fast
spawn_gemini("Search for all TODO comments")
```

**2. Sequential when parallel is possible**
```python
# BAD - total time = T1 + T2 + T3
spawn_codex("Fix auth bugs")  # wait
spawn_codex("Fix db bugs")    # wait
spawn_codex("Fix api bugs")   # wait

# GOOD - total time = max(T1, T2, T3)
spawn_codex("Fix auth bugs")
spawn_codex("Fix db bugs")
spawn_codex("Fix api bugs")
# all run in parallel
```

**3. Mixing business logic with spawning**
```python
# BAD - orchestrator doing tactical work
if file_exists("config.py"):
    spawn_codex("Update config")

# GOOD - delegate everything
Task(prompt="Check if config.py exists and update if needed")
```

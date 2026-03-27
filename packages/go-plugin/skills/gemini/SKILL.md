> **DEPRECATED:** This skill is replaced by the `gemini-operator` agent.
> Use `Agent(subagent_type="htmlgraph:gemini-operator", prompt="...")` instead.
> The gemini-operator agent tries Gemini CLI first with 2M context and JSON output. Free tier.

---
name: gemini
description: GeminiSpawner with full event tracking for exploration and large-context research
when_to_use:
  - Large codebase exploration with full tracking
  - Research tasks requiring extensive context
  - Large-context workflows with event hierarchy
  - Multimodal tasks (images, PDFs, documents)
  - AI-powered code analysis with observability
skill_type: executable
---

# GeminiSpawner - Exploration & Research with Full Event Tracking

⚠️ **IMPORTANT: This skill teaches TWO EXECUTION PATTERNS**

1. **Task(subagent_type="Explore")** - Built-in Claude Explore agent (simplest, recommended)
2. **GeminiSpawner** - Direct Google Gemini CLI with full HtmlGraph parent event tracking

Choose based on your needs. See "EXECUTION PATTERNS" below.

## Quick Summary

| Pattern | Use Case | When to Use |
|---------|----------|-----------|
| **Task(Explore)** | General exploration via Claude Explore agent | When you want simplicity and Claude handles everything |
| **GeminiSpawner** | Direct Gemini CLI invocation with full subprocess tracking | When you need precise Gemini control + full parent event context |

**CRITICAL: GeminiSpawner is invoked DIRECTLY via Python SDK, NOT via Task().**

---

## 🚀 GeminiSpawner Pattern: Full Event Tracking

### What is GeminiSpawner?

GeminiSpawner is the HtmlGraph-integrated way to invoke Google Gemini CLI directly with **full parent event context and subprocess tracking**.

**Key distinction**: GeminiSpawner is invoked directly via Python SDK - NOT wrapped in Task(). Task() is only for Claude subagents (Haiku, Sonnet, Opus).

GeminiSpawner:
- ✅ Invokes external Gemini CLI directly
- ✅ Creates parent event context in database
- ✅ Links to parent Task delegation event
- ✅ Records subprocess invocations as child events
- ✅ Tracks all activities in HtmlGraph event hierarchy
- ✅ Provides full observability of Gemini execution

### When to Use GeminiSpawner vs Task(Explore)

**Use Task(Explore):**
```python
# Simple Claude exploration via subagent
Task(subagent_type="Explore",
     prompt="Analyze this codebase for patterns")
# Task() delegates to Claude Explore agent - no external CLI needed
```

**Use GeminiSpawner (direct Python invocation):**
```python
# Direct Gemini CLI invocation with full tracking
spawner = GeminiSpawner()
result = spawner.spawn(
    prompt="Analyze codebase",
    track_in_htmlgraph=True,
    tracker=tracker,
    parent_event_id=parent_event_id
)
# NOT Task(GeminiSpawner) - invoke directly!
```

### How to Use GeminiSpawner

Use the `htmlgraph:gemini-operator` agent — it tries Gemini CLI first, then falls back to direct exploration:

```python
# PRIMARY: Delegate to gemini-operator agent (handles CLI check + fallback)
Task(
    subagent_type="htmlgraph:gemini-operator",
    prompt="Analyze the refactored spawner architecture for quality",
)
```

### Key Parameters for GeminiSpawner.spawn()

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `prompt` | str | ✅ | Research/analysis task for Gemini |
| `model` | str \| None | ❌ | Model selection (default: None = RECOMMENDED, uses latest models including Gemini 3 preview) |
| `output_format` | str | ❌ | "json" or "stream-json" (default: "stream-json") |
| `track_in_htmlgraph` | bool | ❌ | Enable SDK activity tracking (default: True) |
| `tracker` | SpawnerEventTracker | ❌ | Tracker instance for subprocess events |
| `parent_event_id` | str | ❌ | Parent event ID for event hierarchy |
| `timeout` | int | ❌ | Max seconds to wait (default: 120) |

**Model Selection Note:**
- `model=None` (default): **RECOMMENDED** - CLI chooses best available model (gemini-2.5-flash-lite, gemini-3-flash-preview)
- Explicitly setting a model is discouraged as older models may fail with newer CLI versions
- DEPRECATED: `gemini-2.0-flash`, `gemini-1.5-flash` (may cause "thinking mode" errors)

### Real Example: Code Quality Analysis

```python
Task(
    subagent_type="htmlgraph:gemini-operator",
    prompt="""Analyze the quality of this refactored spawner architecture:

src/python/htmlgraph/orchestration/spawners/
├── base.py (BaseSpawner - 195 lines)
├── gemini.py (GeminiSpawner - 430 lines)
├── codex.py (CodexSpawner - 443 lines)
├── copilot.py (CopilotSpawner - 300 lines)
└── claude.py (ClaudeSpawner - 171 lines)

Evaluate: separation of concerns, code reusability, error handling patterns, event tracking integration."""
)
```

### Fallback & Error Handling Pattern

The `htmlgraph:gemini-operator` agent handles the fallback automatically:
1. Tries Gemini CLI first (free, 2M context)
2. Falls back to direct Read/Grep/Glob exploration if Gemini CLI unavailable

No manual fallback code needed — just delegate to the agent.

**Why fallback to Task()?**
- ✅ Gemini CLI may not be installed on user's system
- ✅ Google API credentials/quota issues may affect external tool
- ✅ Claude Explore agent provides guaranteed exploration fallback
- ✅ Never attempt direct execution as fallback (violates orchestration principles)
- ✅ Task() handles all retries, error recovery, and parent context automatically

**Pattern Summary:**
1. Try external spawner first (Gemini CLI)
2. If spawner succeeds → return result
3. If spawner fails → delegate to Claude sub-agent via Task(subagent_type="Explore")
4. Never try direct execution as fallback

---

<!-- Embedded Python block removed: use htmlgraph:gemini-operator agent instead -->

Use Google Gemini (latest models including Gemini 3 preview) for exploration and research tasks via the GeminiSpawner SDK.

## Skill vs Execution Model

**CRITICAL DISTINCTION:**

| What | Description |
|------|-------------|
| **This Skill** | Documentation + embedded coordination logic |
| **Embedded Python** | Internal check for gemini CLI → spawns if available |
| **Task() Tool** | PRIMARY execution path for exploration work |
| **Bash Tool** | ALTERNATIVE for direct CLI invocation (if you have gemini CLI) |

**Workflow:**
1. Read this skill to understand Gemini capabilities
2. Use **Task(subagent_type="Explore")** for actual exploration (PRIMARY)
3. OR use **Bash** if you have gemini CLI installed (ALTERNATIVE)

## EXECUTION - Real Commands for Exploration

**⚠️ To actually perform exploration, use these approaches:**

### PRIMARY: Task() Delegation (Recommended)
```python
# Use Claude's Explore agent (automatically uses appropriate model)
Task(
    subagent_type="Explore",
    prompt="Analyze all authentication patterns in the codebase and document findings"
)

# For large-context research
Task(
    subagent_type="Explore",
    prompt="Review entire API documentation and extract deprecated endpoints"
)
```

### ALTERNATIVE: Direct CLI (if gemini CLI installed)
```bash
# If you have gemini CLI installed on your system
gemini analyze "Find all authentication patterns"
```

## When to Use

- **Large Context** - 2M token context window for large codebases
- **Multimodal** - Process images, PDFs, and documents
- **Batch Operations** - Analyze many files efficiently
- **Fast Inference** - Quick turnaround for exploratory work

## How to Invoke

**PRIMARY: Use Skill() to invoke (tries external CLI first):**

```python
# Recommended approach - uses external gemini CLI via agent spawner
Skill(skill=".claude-plugin:gemini", args="Analyze authentication patterns in the codebase")
```

**What happens internally:**
1. Check if `gemini` CLI is installed on your system
2. If **YES** → Use agent spawner SDK to execute: `gemini analyze "auth patterns"`
3. If **NO** → Automatically fallback to: `Task(subagent_type="Explore", prompt="Analyze auth patterns")`

**FALLBACK: Direct Task() invocation (when Skill unavailable):**

```python
# Manual fallback - uses Claude's built-in Explore agent
Task(
    subagent_type="Explore",
    prompt="Analyze authentication patterns in the codebase",
    model="haiku"  # Optional: specify model
)
```

The Explore agent automatically uses Gemini for large-context work.

## Capabilities

- **Context Window**: 2M tokens (large-context support)
- **Multimodal**: Process images, PDFs, audio, documents
- **Fast Inference**: Sub-second latency
- **Best For**: Exploration, research, understanding systems

## Example Use Cases

### 1. Codebase Exploration

```python
Task(
    subagent_type="Explore",
    prompt="""
    Search codebase for all authentication patterns:
    1. Where auth is implemented
    2. What auth methods are used
    3. Where auth is validated
    4. Recommendations for adding OAuth 2.0
    """
)
```

### 2. Batch File Analysis

```python
# Analyze multiple files for security issues
Task(
    subagent_type="Explore",
    prompt="Review all API endpoints in src/ for security vulnerabilities"
)
```

### 3. Multimodal Processing

```python
# Extract information from diagrams or images
Task(
    subagent_type="Explore",
    prompt="Extract all text and tables from architecture diagrams in docs/"
)
```

### 4. Large Documentation Review

```python
# Process extensive documentation
Task(
    subagent_type="Explore",
    prompt="Summarize all API documentation and find deprecated endpoints"
)
```

## When to Use Gemini vs Claude

**Use Gemini for:**
- Large-context exploration
- Multimodal document analysis
- Tasks not requiring complex reasoning
- Exploration phase before implementation

**Use Claude for:**
- Precise code generation
- Complex reasoning tasks
- Production code writing
- Critical decision-making

## Fallback Strategy

The skill implements a multi-level fallback strategy:

### Level 1: External CLI (Preferred)
```python
Skill(skill=".claude-plugin:gemini", args="Your exploration task")
# Attempts to use external gemini CLI via agent spawner SDK
```

### Level 2: Claude Explore Agent (Automatic Fallback)
```python
# If gemini CLI not found, automatically falls back to:
Task(subagent_type="Explore", prompt="Your exploration task")
# Uses Claude's built-in Explore agent
```

### Level 3: Claude Models (Final Fallback)
```python
# If alternative unavailable, uses Claude models for exploration
# Maintains full functionality with different inference model
```

**Error Handling:**
- Transparent fallback (no silent failures)
- Clear error messages if all methods fail
- Automatic retry with different methods

## Integration with HtmlGraph

Track exploration work in spikes:

```bash
# Create spike for research findings
htmlgraph spike create "Auth Pattern Analysis via Gemini"
# Then record findings via: htmlgraph spike edit <id>
```

## When NOT to Use

Avoid Gemini for:
- Precise code generation (use Claude Sonnet)
- Critical production code (use Claude with tests)
- Tasks requiring Claude's reasoning (use Sonnet/Opus)
- Small context tasks (overhead not needed)

## Tips for Best Results

1. **Be specific** - Clear prompts get better results
2. **Use for exploration first** - Research before implementing
3. **Leverage large context** - Include entire codebases
4. **Batch operations** - Process many files at once
5. **Document findings** - Save results in HtmlGraph spikes

## Related Skills

- `/codex` - For code implementation after exploration
- `/copilot` - For GitHub integration and git operations
- `/debugging-workflow` - Research-first debugging methodology

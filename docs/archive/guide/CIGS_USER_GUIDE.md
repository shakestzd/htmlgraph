# Computational Imperative Guidance System (CIGS) - User Guide

**Version:** 1.0
**Status:** Design Complete
**Last Updated:** 2026-01-04

## Table of Contents

- [What is CIGS?](#what-is-cigs)
- [How It Works](#how-it-works)
  - [3-Layer Architecture](#3-layer-architecture)
  - [Data Flow](#data-flow)
- [Imperative Messaging System](#imperative-messaging-system)
  - [4 Escalation Levels](#4-escalation-levels)
  - [Message Examples](#message-examples)
- [Pattern Detection & Learning](#pattern-detection--learning)
  - [Recognized Anti-Patterns](#recognized-anti-patterns)
  - [Learning Mechanism](#learning-mechanism)
- [Autonomy Levels](#autonomy-levels)
  - [Decision Matrix](#decision-matrix)
  - [Level Selection](#level-selection)
- [How to Use CIGS](#how-to-use-cigs)
  - [Quick Start](#quick-start)
  - [Daily Workflow](#daily-workflow)
  - [Responding to Guidance](#responding-to-guidance)
- [Configuration Options](#configuration-options)
  - [System Settings](#system-settings)
  - [Per-Session Overrides](#per-session-overrides)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

---

## What is CIGS?

**Computational Imperative Guidance System** (CIGS) is an AI behavioral guidance system that helps you follow delegation best practices without blocking your work.

### Core Philosophy

CIGS uses **imperative guidance** (commanding, well-reasoned suggestions) rather than **restrictions** (blocking operations) to encourage better workflows. Research shows AI agents respond 6-16x more effectively to well-designed nudges than humans do to the same nudges.

### Key Design Principles

1. **Imperative > Advisory** - Clear commands ("YOU MUST delegate") not suggestions ("Consider delegating")
2. **Computational > Heuristic** - Data-driven decisions based on tracked metrics, not assumptions
3. **Reinforcing > Restricting** - Positive feedback for correct behavior, escalating guidance for violations
4. **Intelligent > Mechanical** - Context-aware decisions, not rigid rule enforcement

### What CIGS Does

- **Tracks delegation patterns** - Records when you bypass recommended workflows
- **Provides context-aware guidance** - Shows WHY delegation is better with cost analysis
- **Detects anti-patterns** - Identifies behavioral patterns that waste tokens/time
- **Adapts guidance intensity** - Adjusts based on your demonstrated competence
- **Generates session summaries** - Reports compliance, costs, and improvement recommendations

### What CIGS Does NOT Do

- ❌ Block your work (guidance mode) or force acknowledgments only when truly needed (strict mode)
- ❌ Make decisions for you - you remain in control
- ❌ Store sensitive information - only tracks operation types, not content
- ❌ Require configuration - works with sensible defaults

---

## How It Works

### 3-Layer Architecture

CIGS operates across three complementary layers:

#### Layer 1: System Prompt (Constitutional Framework)

**Purpose:** Establish core delegation principles and autonomy levels

- Constitutional-style rules for self-critique
- Decision frameworks for when to delegate
- Cost model education (why delegation is more efficient)
- Autonomy level definitions

**How It Works:**
- Injected at session start via `session-start.py` hook
- Provides personalized context based on your violation history
- Updates dynamically based on your performance

**Example:**
```
You are operating at "GUIDED" autonomy level (70% compliance last 5 sessions).

Core Principle: Exploration work MUST be delegated to subagents.
You have demonstrated repeated exploration sequences (Read→Grep→Glob→Read).
Pattern: This wastes ~15% of your context tokens on tactical details.

Delegation Options:
- spawn_gemini() for exploration (FREE)
- spawn_codex() for implementation (70% cheaper than Task)
- spawn_copilot() for git operations (60% cheaper than Task)
```

#### Layer 2: Plugin Hooks (Real-Time Intervention)

**Purpose:** Intercept operations and provide imperative guidance before/after execution

**Hooks Used:**

| Hook | Purpose | Actions |
|------|---------|---------|
| **SessionStart** | Initialize tracking | Load violation history, set autonomy level, inject context |
| **UserPromptSubmit** | Pre-response guidance | Classify intent, remind of decision framework |
| **PreToolUse** | Pre-execution enforcement | Classify tool, generate imperative, record violation |
| **PostToolUse** | Post-execution feedback | Calculate actual cost, provide reinforcement |
| **Stop** | Session summary | Report compliance, patterns, autonomy recommendation |

**How It Works:**
1. When you use a tool, PreToolUse hook intercepts
2. Hook classifies the operation (exploration, implementation, testing, git)
3. Checks violation count and autonomy level
4. Generates appropriate imperative message (if needed)
5. Records violation for learning (if applicable)
6. When session ends, Stop hook analyzes patterns and recommends adjustments

**Example Flow:**
```
You: Bash("grep -r 'authentication' src/")
     ↓
PreToolUse Hook:
  - Classify: exploration_sequence (2+ exploration tools in last 5 calls)
  - Violation count: 1
  - Autonomy: guided
  ↓
Generate Imperative:
  🔴 IMPERATIVE: YOU MUST delegate exploration.
  (Full message with cost analysis)
  ↓
Record Violation
  ↓
Execute Bash
  ↓
PostToolUse Hook:
  - Calculate actual cost: 3,000 tokens
  - Compare to optimal: 500 tokens (via spawn_gemini)
  - Record cost impact
```

#### Layer 3: Python Package (Analytics & State Management)

**Purpose:** Persistent tracking, pattern analysis, and autonomy recommendations

**Components:**

| Component | Purpose |
|-----------|---------|
| **ViolationTracker** | Records violations with full context (tool, params, waste estimate) |
| **PatternAnalyzer** | Detects anti-patterns and improvements from violation history |
| **CostCalculator** | Predicts operation costs and calculates actual efficiency |
| **AutonomyManager** | Recommends autonomy levels based on compliance history |
| **ImperativeMessageGenerator** | Creates escalating messages with context-aware messaging |

**Storage Format:**

CIGS stores all data in Wipnote format (JSONL + JSON):

```
.wipnote/cigs/
├── violations.jsonl          # Violation records (1 per line)
├── patterns.json             # Detected patterns (anti + good)
├── session-summaries/        # Per-session analytics
│   └── {session_id}.json
└── autonomy.json             # Current autonomy settings
```

### Data Flow

```
User Request
     │
     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      UserPromptSubmit Hook                               │
│  Load session context → Classify intent → Inject decision framework      │
└─────────────────────────────────────────────────────────────────────────┘
     │
     ▼ (Claude generates response with tool calls)
     │
┌─────────────────────────────────────────────────────────────────────────┐
│                       PreToolUse Hook                                    │
│  Classify tool → Check autonomy level → Generate imperative message      │
│  Predict cost → Record violation (if applicable)                         │
└─────────────────────────────────────────────────────────────────────────┘
     │
     ▼ (Tool executes)
     │
┌─────────────────────────────────────────────────────────────────────────┐
│                      PostToolUse Hook                                    │
│  Calculate actual cost → Compare to prediction → Provide feedback        │
│  Update learning model → Record pattern data                             │
└─────────────────────────────────────────────────────────────────────────┘
     │
     ▼ (Session ends)
     │
┌─────────────────────────────────────────────────────────────────────────┐
│                        Stop Hook                                         │
│  Generate session summary → Detect patterns → Recommend autonomy level   │
│  Persist all analytics → Prepare for next session                        │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Imperative Messaging System

CIGS communicates through **imperative messages** - clear, commanding guidance with escalating intensity based on your behavior.

### 4 Escalation Levels

#### Level 0: Guidance (Informative)

**When:** First encounter with a pattern or initial guidance
**Tone:** Informative, educational
**Example:**
```
💡 GUIDANCE: Consider delegating exploration to spawn_gemini()
for comprehensive search.

spawn_gemini() is FREE and can search your entire codebase at once.
Direct Read operations add ~5000 tokens to your context per file.
```

#### Level 1: Imperative (Commanding)

**When:** Second violation of a pattern in the same session
**Tone:** Direct, commanding
**Includes:** WHY + Cost impact + Suggestion
**Example:**
```
🔴 IMPERATIVE: YOU MUST delegate file reading to Explorer subagent.

**WHY:** Exploration operations have unpredictable scope.
What looks like 'one Read' often becomes 3-5 reads.
Each direct read pollutes your strategic context with tactical details.

**COST IMPACT:** Direct execution costs ~5000 tokens in your context.
Delegation would cost ~500 tokens (90% savings).

**INSTEAD:**
spawn_gemini(prompt="Search and analyze codebase for authentication patterns")
```

#### Level 2: Final Warning (Urgent)

**When:** Third violation or detected anti-pattern
**Tone:** Urgent, consequence-aware
**Includes:** WHY + Cost + Suggestion + Consequences
**Example:**
```
⚠️ FINAL WARNING: YOU MUST delegate NOW. Pattern detected: exploration sequence.

**WHY:** You have already executed 3 exploration operations.
This is research work that should be delegated.
Subagent can explore comprehensively and return a summary.

**COST IMPACT:** Session waste so far: 15,000 tokens. Optimal path: 2,000 tokens.

**INSTEAD:**
spawn_gemini(prompt="Comprehensive search for all authentication-related code")

**CONSEQUENCE:** Next violation will trigger circuit breaker,
requiring manual acknowledgment.
```

#### Level 3: Circuit Breaker (Blocking)

**When:** 3+ violations in a session
**Tone:** Blocking, requires acknowledgment
**Includes:** WHY + Session impact + Acknowledgment requirement
**Example:**
```
🚨 CIRCUIT BREAKER: Delegation violations exceeded threshold (3/3).

**WHY:** Repeated direct execution despite warnings indicates
need for mandatory intervention.

**SESSION IMPACT:**
- Violations: 3
- Total waste: 25,000 tokens
- Efficiency score: 45/100

**REQUIRED:** Acknowledge this violation before proceeding:
uv run wipnote orchestrator acknowledge-violation

OR disable enforcement:
uv run wipnote orchestrator set-level guidance
```

### Message Components

Every imperative message includes:

1. **Prefix with indicator** - Shows escalation level (💡 → 🔴 → ⚠️ → 🚨)
2. **Core message** - What you MUST do
3. **WHY section** - Reasoning behind the imperative
4. **COST IMPACT section** - Token savings/waste (when applicable)
5. **INSTEAD section** - What to do instead
6. **CONSEQUENCE section** - What happens next (level 2+)
7. **Required action** (level 3 only) - How to acknowledge

### Positive Reinforcement

When you follow best practices, CIGS provides positive feedback:

```
✅ Excellent delegation pattern!

**Impact:**
- Saved ~4,500 tokens of context
- Subagent handled tactical details
- Your strategic view remains clean

**Session Stats:**
- Delegation compliance: 87%
- Efficiency score: 82/100
- Keep it up! Consistent delegation improves response quality.
```

---

## Pattern Detection & Learning

CIGS learns from your behavior and detects recurring patterns to provide targeted guidance.

### Recognized Anti-Patterns

#### 1. Exploration Sequence

**What it is:** Multiple exploration tools (Read, Grep, Glob) used in sequence

**Detection:** 3+ exploration tools in last 5 tool calls

**Why it's problematic:**
- What looks like "one search" becomes a multi-step investigation
- Each step adds tactical details to your context
- Exploration is inherently uncertain and iterative - perfect for subagents

**Remedy:**
```python
# Instead of:
Read("src/auth/login.py")  # 5000 tokens
Grep("pattern", "src/")    # 3000 tokens
Read("src/auth/handler.py") # 5000 tokens
# Total: 13,000 tokens in your context

# Use:
spawn_gemini(prompt="Find all authentication patterns in codebase")
# Returns: comprehensive summary, 500 tokens in your context
# Savings: 12,500 tokens (96%)
```

#### 2. Edit Without Testing

**What it is:** Code changes without immediately delegating verification

**Detection:** Edit in last 3 calls without Task/spawn_codex for testing

**Why it's problematic:**
- Implementation is always iterative (edit → test → fix → test)
- You end up testing anyway, but in separate operations
- Should be delegated as a unit to subagent for efficiency

**Remedy:**
```python
# Instead of:
Edit("src/auth.py", ...)      # Edit
# Later...
Task("Run tests for auth module")  # Test

# Use:
spawn_codex(prompt="Implement authentication and add tests...")
# Subagent does: edit → test → iterate → return verified result
```

#### 3. Direct Git Operations

**What it is:** Git commands executed directly instead of via Copilot

**Detection:** Bash command containing "git commit", "git push", etc.

**Why it's problematic:**
- Git operations are non-deterministic (conflicts, hooks, permissions)
- Copilot specializes in git and costs 60% less than Task()
- Should always be delegated to git expert

**Remedy:**
```python
# Instead of:
Bash("git add src/auth.py && git commit -m 'Add auth'")

# Use:
spawn_copilot(prompt="Commit authentication changes with descriptive message")
# Copilot handles: proper message formatting, commit conventions, potential issues
```

#### 4. Repeated Read of Same File

**What it is:** Reading the same file multiple times instead of analyzing once

**Detection:** 70%+ of Read operations in last 10 calls are to same file

**Why it's problematic:**
- Indicates incomplete analysis on first read
- Second read adds redundant context
- Better to read once, analyze thoroughly, or delegate

**Remedy:**
```python
# Instead of:
Read("config.json")    # First read - partial analysis
# Later...
Read("config.json")    # Second read - get the rest

# Use:
Read("config.json")    # Read once
Grep("pattern", "config.json")  # Analyze thoroughly
# OR delegate entire analysis:
spawn_gemini(prompt="Analyze config.json structure and return summary")
```

### Learning Mechanism

CIGS learns from every interaction:

1. **Detection** - When you trigger an anti-pattern, it's recorded
2. **Analysis** - Patterns are classified and trend-analyzed
3. **Customization** - Future guidance is tailored to YOUR patterns
4. **Feedback** - Session summaries show what you've improved on

**Example Learning Cycle:**

```
Session 1: Exploration sequence detected (3+ reads)
  → Level 1 imperative provided

Session 2: Exploration sequence detected again
  → Level 2 final warning provided
  → Pattern recorded: "exploration_sequence" detected 2x

Session 3: Exploration sequence detected a third time
  → Level 3 circuit breaker triggered
  → Pattern analysis updated: high-risk pattern

Session 4: System recommends "Operator" autonomy level
  → More aggressive guidance pre-loaded
  → Pattern customization active

Session 5: User delegates exploration properly
  → Pattern "good_pattern: immediate_delegation" recorded
  → Positive reinforcement increases
```

---

## Autonomy Levels

CIGS adapts its guidance intensity based on demonstrated competence through **autonomy levels**.

### Decision Matrix

| Level | Compliance Rate | Anti-Patterns | Messaging | Use Case |
|-------|-----------------|---------------|-----------|----------|
| **Observer** | >90% | 0 | Minimal | Demonstrated expertise, light guidance |
| **Consultant** | 70-90% | 1-2 | Moderate | Good performance, room for improvement |
| **Collaborator** | 50-70% | 3+ | High | Needs support, active learning |
| **Operator** | <50% | 4+ | Maximal | Strict guidance, frequent violations |

### Level Selection

#### Observer Level (Minimal Guidance)

**Recommended when:**
- Delegation compliance >90%
- No detected anti-patterns
- Consistent positive reinforcement

**What you experience:**
```
✅ Excellent delegation pattern!
Compliance: 91% | Efficiency: 85/100
```

**Best for:** Experienced users, high-stakes workflows

#### Consultant Level (Moderate Guidance)

**Recommended when:**
- Delegation compliance 70-90%
- 1-2 anti-patterns detected
- Room for improvement but solid foundation

**What you experience:**
```
💡 GUIDANCE: Consider delegating exploration to spawn_gemini()
```

**Best for:** Most users, balanced workflow

#### Collaborator Level (High Guidance)

**Recommended when:**
- Delegation compliance 50-70%
- 3+ anti-patterns detected
- Active pattern of suboptimal choices

**What you experience:**
```
🔴 IMPERATIVE: YOU MUST delegate this operation.
(Full explanation with cost analysis)
```

**Best for:** Learning mode, complex projects requiring strict discipline

#### Operator Level (Maximal Guidance)

**Recommended when:**
- Delegation compliance <50%
- 4+ anti-patterns detected
- Circuit breaker triggered

**What you experience:**
```
🚨 CIRCUIT BREAKER: Violations exceeded threshold.
(Requires acknowledgment before proceeding)
```

**Best for:** Reset situations, enforced workflow adherence

### How Autonomy is Determined

At session start, CIGS:

1. **Loads violation history** (last 5 sessions)
2. **Calculates compliance rate** - delegations / (delegations + violations)
3. **Detects patterns** - active anti-patterns in history
4. **Computes recommendation** using decision matrix
5. **Applies immediately** - guidance intensity adjusted

**Example:**

```
Last 5 sessions:
- Session 1: 87% compliance, "exploration_sequence" anti-pattern
- Session 2: 81% compliance, "exploration_sequence" anti-pattern
- Session 3: 76% compliance, "exploration_sequence" + "direct_git" anti-patterns
- Session 4: 72% compliance, 2 anti-patterns detected
- Session 5: 78% compliance, 2 anti-patterns continuing

Analysis:
- Average: 78.8% compliance
- Anti-patterns: 2 active
- Trend: Slight decline

Recommendation: COLLABORATOR level
Messaging: High intensity
Reason: "Moderate compliance with persistent anti-patterns"
```

---

## How to Use CIGS

### Quick Start

1. **Enable CIGS** (if not already enabled)
   ```bash
   uv run wipnote cigs enable
   ```

2. **Check your autonomy level**
   ```bash
   uv run wipnote cigs status
   ```

3. **Work normally** - guidance appears automatically
   ```python
   # As you work, you'll see guidance messages
   # Follow them to improve delegation compliance
   ```

4. **Check session summary at end**
   ```bash
   uv run wipnote cigs summary
   ```

### Daily Workflow

#### Morning (Session Start)

1. Begin your session
2. CIGS loads your violation history
3. System prompt injected with personalized context
4. You see your autonomy level and recent patterns

**Example Output:**
```
CIGS Status - Session Start
Autonomy Level: GUIDED
Previous Session: 82% compliance, 1 violation
Active Patterns: exploration_sequence (detected in last 3 sessions)
Pattern Status: No circuit breaker active
```

#### During Work (Real-Time Guidance)

As you attempt tools:

1. **If tool is allowed** - executes immediately
   ```
   ✅ Direct Bash execution allowed
   Allowed: Single file operation
   ```

2. **If guidance needed** - receive imperative message
   ```
   💡 GUIDANCE: This looks like exploration work.
   Consider delegating to spawn_gemini() (FREE)
   ```

3. **If violation detected** - recorded and escalated
   ```
   🔴 IMPERATIVE: YOU MUST delegate exploration.
   Cost impact: 12,000 tokens waste predicted.
   ```

4. **If circuit breaker triggered** - requires acknowledgment
   ```
   🚨 CIRCUIT BREAKER: You have 3 violations this session.
   Run: uv run wipnote orchestrator acknowledge-violation
   ```

#### End of Session (Summary)

1. Session ends automatically or you explicitly stop
2. CIGS generates comprehensive summary
3. Patterns analyzed and persisted
4. Autonomy level recommendation for next session

**Example Summary:**
```
📊 CIGS Session Summary

Delegation Metrics
- Compliance Rate: 78%
- Violations: 2 (threshold: 3)
- Circuit Breaker: Not triggered

Cost Analysis
- Total Context Used: 450,000 tokens
- Estimated Waste: 45,000 tokens (10%)
- Optimal Path Cost: 405,000 tokens
- Efficiency Score: 78/100

Detected Patterns
- exploration_sequence: 4 occurrences this session
- edit_without_test: 1 occurrence

Anti-Patterns Identified
- Exploration Sequence: Pattern detected
  Remediation: Use spawn_gemini() for comprehensive search

Autonomy Recommendation
Next Session: GUIDED
Reason: Moderate compliance with persistent anti-patterns
Messaging Intensity: Moderate

Learning Applied
- Violation patterns added to detection model
- Cost predictions updated with actual data
- Messaging intensity adjusted for next session
```

### Responding to Guidance

#### When You See Level 0-1 Guidance

**Best practice:** Follow the suggestion

```
🔴 IMPERATIVE: YOU MUST delegate file reading.
**INSTEAD:** spawn_gemini(prompt="Search...")

Your response:
result = spawn_gemini(prompt="Search authentication patterns in codebase")
```

#### When You See Level 2 Final Warning

**Best practice:** Strongly consider taking the suggestion

This indicates the pattern is getting serious. The next violation will trigger circuit breaker. This is your opportunity to change course before stricter enforcement activates.

#### When You See Level 3 Circuit Breaker

**Three options:**

**Option 1: Acknowledge and adjust (recommended)**
```bash
# Acknowledge the violation
uv run wipnote orchestrator acknowledge-violation

# Then switch to delegation strategy
# This resets the counter for the session
```

**Option 2: Temporarily disable CIGS**
```bash
# Lower enforcement level for this session
uv run wipnote cigs set-level guidance

# Later, review why you needed to disable:
uv run wipnote cigs summary
# Then re-enable: uv run wipnote cigs set-level strict
```

**Option 3: Understand and improve**
```bash
# Check what patterns triggered it
uv run wipnote cigs violations --session-id <current>

# Review examples of correct behavior
uv run wipnote cigs examples --pattern exploration_sequence --successful

# Return to work with delegation focus
```

---

## Configuration Options

### System Settings

#### Autonomy Level

Control guidance intensity globally:

```bash
# Set autonomy level
uv run wipnote cigs set-level [observer|consultant|collaborator|operator]

# Example: Strict mode during critical work
uv run wipnote cigs set-level operator

# Example: Light mode for experimentation
uv run wipnote cigs set-level observer
```

#### Messaging Intensity

Control how verbose CIGS messages are:

```bash
# Set messaging style
uv run wipnote cigs set-messaging [minimal|moderate|high|maximal]

# Examples:
uv run wipnote cigs set-messaging minimal  # Just core message
uv run wipnote cigs set-messaging maximal  # Full explanation with examples
```

#### Pattern Detection Sensitivity

Control how aggressively patterns are detected:

```bash
# Set pattern sensitivity
uv run wipnote cigs set-sensitivity [low|medium|high]

# low: Only obvious anti-patterns detected
# medium: Standard patterns (default)
# high: Sensitive to subtle patterns
```

#### Circuit Breaker Threshold

Control when circuit breaker triggers:

```bash
# Set violation threshold
uv run wipnote cigs set-threshold [violations-count]

# Example: Trigger at 5 violations instead of 3
uv run wipnote cigs set-threshold 5

# Example: More aggressive at 2
uv run wipnote cigs set-threshold 2
```

### Per-Session Overrides

#### Disable CIGS for a Session

```bash
# Start session with guidance disabled
uv run wipnote cigs session --no-enforcement

# Work normally (no messages)
# At end of session, you can review data:
uv run wipnote cigs summary
```

#### Enable Strict Mode for a Session

```bash
# Start session with maximal enforcement
uv run wipnote cigs session --strict

# Circuit breaker triggers at 1 violation instead of 3
```

#### Focus on Specific Pattern

```bash
# Enable only specific pattern detection
uv run wipnote cigs session --focus exploration_sequence

# Only get guidance about exploration patterns
# Other operations allowed without guidance
```

---

## Troubleshooting

### Issue: Getting too many guidance messages

**Problem:** CIGS is too noisy/intrusive

**Solutions:**
1. Lower messaging intensity
   ```bash
   uv run wipnote cigs set-messaging minimal
   ```

2. Switch to lower autonomy level
   ```bash
   uv run wipnote cigs set-level observer
   ```

3. Increase circuit breaker threshold
   ```bash
   uv run wipnote cigs set-threshold 5
   ```

4. Check if you've improved compliance - autonomy might auto-adjust
   ```bash
   uv run wipnote cigs status
   ```

### Issue: Not getting guidance when I expect it

**Problem:** CIGS isn't detecting a violation

**Causes and solutions:**

1. **You're already compliant** - Check your status
   ```bash
   uv run wipnote cigs status
   # If "observer" level, minimal guidance is expected
   ```

2. **Tool is in allowed list** - Some operations don't need delegation
   ```bash
   uv run wipnote cigs allowed-tools
   # Shows which tools are always allowed
   ```

3. **Pattern sensitivity too low** - Increase it
   ```bash
   uv run wipnote cigs set-sensitivity high
   ```

4. **First occurrence of pattern** - Level 0 guidance is subtle
   ```bash
   # Next occurrence of same pattern will be more prominent
   ```

### Issue: Circuit breaker triggered but I need to work

**Problem:** Stuck at 3+ violations, can't proceed

**Solutions:**

1. **Acknowledge and reset**
   ```bash
   uv run wipnote orchestrator acknowledge-violation
   # Counter resets to 0 for this session
   # Proceed with delegation focus
   ```

2. **Lower autonomy level temporarily**
   ```bash
   uv run wipnote cigs set-level guidance
   # Guidance mode allows all operations
   # Circuit breaker deactivated
   # Review patterns afterward to understand why needed
   ```

3. **Understand the pattern**
   ```bash
   uv run wipnote cigs explain [pattern-name]
   # Get detailed explanation of why it's problematic
   # Review correct examples
   # Return with better understanding
   ```

### Issue: Autonomy level doesn't match my actual compliance

**Problem:** System recommends "Operator" but you think you're compliant

**Investigation:**

```bash
# Check recent violation history
uv run wipnote cigs violations --limit 10

# Check compliance calculation
uv run wipnote cigs compliance --sessions 5

# Check detected patterns
uv run wipnote cigs patterns --active

# Manual review
# If you disagree with assessment, review specific violations
# Identify any that were "exceptions" vs actual problems
```

### Issue: Want to reset CIGS history

**Problem:** Starting fresh or cleaning up old data

```bash
# Warning: This deletes all tracking data
uv run wipnote cigs reset

# Or just reset violations (keep patterns)
uv run wipnote cigs reset --violations-only

# Or reset specific session
uv run wipnote cigs reset --session-id [session-id]
```

---

## FAQ

### General Questions

**Q: How is CIGS different from just using the Orchestrator?**

A: Orchestrator is a simpler "tool counter" system - it blocks at thresholds. CIGS is a learning system that:
- Understands WHY certain patterns are problematic
- Provides cost analysis (token savings)
- Detects behavioral anti-patterns
- Adapts guidance based on your competence
- Explains reasoning in imperative messages

Think of it: Orchestrator = traffic light (red/green), CIGS = intelligent coach (understands game, provides feedback).

---

**Q: Does CIGS track the content of my work?**

A: No. CIGS only records:
- Tool name (not parameters or content)
- Operation category (exploration, implementation, testing, git)
- Timestamp and violation type
- Cost metrics (tokens, not content)

It does NOT record:
- File contents
- Code being edited
- Search terms
- Sensitive business logic

Storage location: `.wipnote/cigs/` (local, not shared)

---

**Q: What if I genuinely need to bypass CIGS guidance?**

A: You have full control:

1. **For a single operation:** Acknowledge the violation and continue
   ```bash
   uv run wipnote orchestrator acknowledge-violation
   ```

2. **For a session:** Lower enforcement level
   ```bash
   uv run wipnote cigs set-level guidance
   ```

3. **Permanently:** Disable CIGS
   ```bash
   uv run wipnote cigs disable
   ```

The system is designed to guide, not restrict. You can always override.

---

**Q: How do I know if CIGS is working?**

A: Check these indicators:

```bash
# 1. Verify it's active
uv run wipnote cigs status
# Should show: "CIGS Status: Enabled"

# 2. Check autonomy level is set
# Should show: "Autonomy Level: [observer|consultant|collaborator|operator]"

# 3. Review recent messages
uv run wipnote cigs messages --recent 10
# Should show guidance messages from recent sessions

# 4. Verify hook integration
uv run wipnote hooks list
# Should show CIGS hooks in hook list
```

---

### Violation & Pattern Questions

**Q: What counts as a violation?**

A: An operation counts as a violation when:
1. It's in a delegable category (exploration, implementation, testing, git)
2. Your current autonomy level marks it as needing delegation
3. You haven't already delegated similar work this session

Not violations:
- Core orchestrator operations (Task, AskUserQuestion, TodoWrite)
- Allowed tools (single config file reads, quick lookups)
- Operations under your autonomy level's threshold

---

**Q: Can I tell CIGS to ignore a specific operation type?**

A: Yes, for specific patterns:

```bash
# Whitelist specific operations
uv run wipnote cigs whitelist-operation [operation-type]

# Example: Allow direct git commits
uv run wipnote cigs whitelist-operation direct_git

# Remove whitelist
uv run wipnote cigs remove-whitelist [operation-type]
```

Use sparingly - whitelisting defeats the learning system.

---

**Q: How often does CIGS detect anti-patterns?**

A: When conditions are met (typically immediate):

- **Exploration Sequence:** Detected when 3+ exploration tools in last 5 calls
- **Edit Without Test:** Detected when edit happens without test delegation in next 3 calls
- **Direct Git:** Detected on next git command after initial session pattern
- **Repeated Read:** Detected when 70%+ of recent reads are to same file

Patterns persist until broken for 5+ consecutive sessions.

---

### Autonomy & Configuration Questions

**Q: How does the system choose between autonomy levels?**

A: Automatically based on:

1. **Compliance rate** (past 5 sessions)
   - >90% = Observer
   - 70-90% = Consultant
   - 50-70% = Collaborator
   - <50% = Operator

2. **Anti-pattern count**
   - 0 = Lower guidance
   - 1-2 = Moderate guidance
   - 3+ = Higher guidance
   - 4+ = Maximum guidance

3. **Trend** (improving or declining)
   - Improving → autonomy level can decrease
   - Declining → autonomy level increases

The system recalculates at session start.

---

**Q: Can I manually set autonomy level?**

A: Yes, but it will auto-correct on next session:

```bash
# Manually override
uv run wipnote cigs set-level observer

# Next session start: System analyzes compliance
# If you're actually at 65%, system updates back to "collaborator"
```

Best practice: Let the system recommend, then adjust if you disagree with its assessment.

---

**Q: What's the difference between "Guidance Mode" and "Consultant" level?**

A: Different configurations:

- **Guidance Mode** (enforcement mode)
  - Allows all operations
  - Provides suggestions only
  - No circuit breaker
  - Best for: Learning, experimentation

- **Consultant Level** (autonomy level)
  - Moderate guidance intensity
  - Provides imperatives for violations
  - Circuit breaker at 3 violations
  - Best for: Most users, balanced workflow

You can combine them: `set-level collaborator` + `set-enforcement guidance` = high-intensity guidance without blocking.

---

### Cost & Efficiency Questions

**Q: How does CIGS calculate token costs?**

A: Three sources:

1. **Predicted costs** (PreToolUse hook)
   - Estimates based on operation type
   - Tool parameters and history
   - Known cost ratios from empirical data

2. **Actual costs** (PostToolUse hook)
   - Records real token usage
   - Updates prediction model
   - Measures efficiency score

3. **Optimal costs** (CostCalculator)
   - What delegation would have cost
   - Based on subagent baseline costs
   - Adjusted for context savings

**Formula:**
```
Efficiency Score = (Optimal Cost / Actual Cost) * 100 - (Violation Count * 5)
```

---

**Q: Is delegation always more efficient?**

A: Usually, but not always:

**When delegation is more efficient:**
- Exploration work (unpredictable scope)
- Implementation (requires iteration)
- Git operations (non-deterministic)
- Research (complex analysis)

**When direct execution might be fine:**
- Single file config read
- Quick grep verification
- Status checks
- Simple one-off operations

CIGS learns from your patterns to distinguish these cases.

---

**Q: How is "waste" calculated?**

A:
```
Waste = Actual Cost - Optimal Cost

Example:
- Direct Read: 5,000 tokens
- Optimal (delegated): 500 tokens
- Waste: 4,500 tokens

Waste Percentage = (Waste / Actual Cost) * 100 = 90%
```

CIGS tracks cumulative waste to show true cost of delegation violations.

---

### Session & Learning Questions

**Q: How does CIGS learn from my behavior?**

A: Three-step learning cycle:

1. **Record** - Every violation stored with context
   ```json
   {
     "tool": "Grep",
     "timestamp": "2026-01-04T10:15:00Z",
     "violation_type": "exploration_sequence",
     "actual_cost": 3000,
     "optimal_cost": 500
   }
   ```

2. **Analyze** - Violations aggregated to find patterns
   - Same pattern repeated?
   - Cost impact accumulating?
   - Trend improving or declining?

3. **Adapt** - System adjusts:
   - Autonomy level recommendation
   - Guidance intensity
   - Pattern detection sensitivity
   - Message customization

Learning is cumulative and persistent across sessions.

---

**Q: Can I see my learning history?**

A: Yes, multiple ways:

```bash
# Recent violations
uv run wipnote cigs violations --limit 20

# Pattern history
uv run wipnote cigs patterns --history

# Compliance trend
uv run wipnote cigs compliance --graph

# Full session analysis
uv run wipnote cigs analyze-sessions --limit 10
```

---

**Q: How long does it take to move to lower autonomy level?**

A: Depends on sustained improvement:

**From Operator → Collaborator:**
- Need >50% compliance for 3 consecutive sessions
- AND reduce active anti-patterns to <3
- Minimum: 3 sessions

**From Collaborator → Consultant:**
- Need >70% compliance for 3 consecutive sessions
- AND reduce active anti-patterns to <2
- Minimum: 3 sessions

**From Consultant → Observer:**
- Need >90% compliance for 3 consecutive sessions
- AND zero active anti-patterns
- Minimum: 3 sessions

System checks at session start and auto-adjusts if criteria met.

---

**Q: What if I have mixed compliance (good in some areas, bad in others)?**

A: CIGS considers overall compliance rate, but provides pattern-specific guidance:

```
Overall: 72% compliance (recommends Collaborator)
Active Patterns:
- exploration_sequence: HIGH (7 violations)
- edit_without_test: LOW (1 violation)
- direct_git: NONE (0 violations)

Guidance: Heavy focus on exploration delegation,
          light guidance on edit-test pattern,
          no git-specific messages
```

---

### Integration Questions

**Q: Does CIGS work with the Orchestrator?**

A: Yes, they're complementary:

- **Orchestrator** - Simple tool-counting system
- **CIGS** - Intelligent learning system

They work together:
1. Orchestrator sets basic thresholds
2. CIGS adds learned pattern detection
3. Together provide comprehensive guidance

You can use one or both:
```bash
# Orchestrator only
uv run wipnote orchestrator enable
uv run wipnote cigs disable

# CIGS only
uv run wipnote cigs enable
uv run wipnote orchestrator disable

# Both (recommended)
uv run wipnote cigs enable
uv run wipnote orchestrator enable
```

---

**Q: Does CIGS affect performance?**

A: Minimal impact:

- Hook execution: <50ms per tool call
- Storage: ~1KB per violation record
- Memory: <10MB for typical usage
- Database queries: Cached, <10ms

No noticeable slowdown in typical workflows.

---

**Q: Can CIGS be integrated into my own project?**

A: Yes! CIGS is part of the Wipnote package:

```python
from wipnote.cigs import (
    ViolationTracker,
    PatternAnalyzer,
    CostCalculator,
    AutonomyManager,
    ImperativeMessageGenerator
)

# Use in your own hook system
tracker = ViolationTracker(graph_dir)
patterns = PatternAnalyzer(graph_dir)
cost = CostCalculator()
```

See `docs/CIGS_INTEGRATION.md` for detailed integration guide.

---

### Troubleshooting Questions

**Q: CIGS messages are different from documentation. Why?**

A: Messages are personalized based on:
- Your autonomy level
- Current violation count
- Detected patterns
- Session context

Documentation shows examples, actual messages adapt to your situation. If you think a message is incorrect, report it:

```bash
uv run wipnote cigs report-issue [issue-type]
```

---

**Q: I disabled CIGS but violations are still being recorded. Why?**

A: CIGS has two independent modes:

- **Enforcement** (generates messages) - Can be disabled
- **Tracking** (records operations) - Always runs unless explicitly stopped

To completely stop tracking:
```bash
uv run wipnote cigs disable --include-tracking
```

To resume:
```bash
uv run wipnote cigs enable
```

---

**Q: How do I report a CIGS bug?**

A: Provide diagnostic information:

```bash
# Collect diagnostics
uv run wipnote cigs diagnostics > cigs-diagnostics.txt

# Include in bug report:
# - What happened
# - What you expected
# - cigs-diagnostics.txt output
# - Recent violations (uv run wipnote cigs violations --limit 5)
```

---

## Support & Resources

### Documentation

- **Quick Start:** See "How to Use CIGS" above
- **Integration Guide:** `docs/CIGS_INTEGRATION.md`
- **API Reference:** `docs/api/cigs.md`
- **Design Document:** `.wipnote/spikes/computational-imperative-guidance-system-design.md`

### Commands

```bash
# Core status and info
uv run wipnote cigs enable                  # Enable CIGS
uv run wipnote cigs disable                 # Disable CIGS
uv run wipnote cigs status                  # Check current status
uv run wipnote cigs summary                 # Session summary

# Configuration
uv run wipnote cigs set-level [level]       # Set autonomy level
uv run wipnote cigs set-messaging [style]   # Set messaging intensity
uv run wipnote cigs set-threshold [count]   # Set circuit breaker threshold

# Analysis & Learning
uv run wipnote cigs violations --limit N    # View recent violations
uv run wipnote cigs patterns --active       # View active patterns
uv run wipnote cigs compliance --sessions N # View compliance trend
uv run wipnote cigs explain [pattern]       # Explain a pattern

# Session Management
uv run wipnote cigs session --no-enforcement # Run without guidance
uv run wipnote cigs session --strict         # Run with strict guidance
uv run wipnote cigs reset                   # Clear all data (warning!)
```

### Getting Help

1. **Check FAQ** - Most common questions answered above
2. **Review examples** - See message examples in Imperative Messaging System section
3. **Check diagnostics** - `uv run wipnote cigs diagnostics`
4. **Report issue** - `uv run wipnote cigs report-issue`

---

**CIGS User Guide - Complete**

This guide covers all aspects of using the Computational Imperative Guidance System. For deeper technical details, see the design document in `.wipnote/spikes/`.

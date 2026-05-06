# Multi-AI Orchestration Guide

**Wipnote enables seamless orchestration across multiple AI platforms** using specialized spawner agents. This guide explains the architecture, how to use each agent, and comprehensive troubleshooting.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Three Spawner Agents](#three-spawner-agents)
3. [Agent Selection Guide](#agent-selection-guide)
4. [Installation & Setup](#installation--setup)
5. [Usage Examples](#usage-examples)
6. [Error Handling](#error-handling)
7. [Fallback Strategies](#fallback-strategies)
8. [Event Tracking](#event-tracking)
9. [Performance Optimization](#performance-optimization)
10. [FAQ](#faq)

---

## Architecture Overview

### How It Works

Wipnote uses a **Task delegation pattern** to route work to specialized agents:

```
┌─────────────────────────────────────────────┐
│ Orchestrator (Claude/Human)                 │
│ "This needs exploration"                    │
└──────────────────┬──────────────────────────┘
                   │
                   ├──→ Reads agent descriptions
                   │    (plugin.json)
                   │
                   ├──→ Selects best match
                   │    (Gemini = FREE + exploration)
                   │
                   └──→ Task(subagent_type="gemini")
                        │
                        └──→ gemini-spawner.py
                             ├─ Initialize spawner
                             ├─ Execute Gemini CLI
                             ├─ Track events in Wipnote
                             └─ Return JSON result
                                 {
                                   "success": true,
                                   "response": "...",
                                   "cost": "FREE"
                                 }
```

### Key Principles

1. **Capability-Based Routing** - Task description determines agent selection
2. **Cost Optimization** - Prefer FREE > INCLUDED > PAID
3. **Transparent Error Handling** - Clear errors when CLI unavailable
4. **Event Tracking** - All delegations recorded in Wipnote
5. **Session Continuity** - Parent-child session linking via environment variables

---

## Three Spawner Agents

### 1. Gemini Spawner (FREE)

**Best for**: Exploratory research, analysis, large-context understanding

```
┌──────────────────────────────────┐
│ Gemini Spawner Agent             │
├──────────────────────────────────┤
│ Model: Google Gemini 2.0-Flash   │
│ Cost: FREE                       │
│ Context: 2M tokens               │
│ Capabilities:                    │
│  - exploration                   │
│  - analysis                      │
│  - batch_processing              │
└──────────────────────────────────┘
```

**When to Use:**
- Codebase exploration and analysis
- Documentation review
- Large batch processing
- Research and investigation
- Context-heavy tasks (benefits from 2M context)

**CLI Tool**: `gemini` (npm: `@google/generative-ai-cli`)

**Installation:**
```bash
npm install -g @google/generative-ai-cli
gemini config set api_key "YOUR_KEY"
```

**Example:**
```python
Task(
    subagent_type="gemini",
    prompt="Analyze the codebase and find all database queries"
)
```

---

### 2. Codex Spawner (PAID)

**Best for**: Code generation, implementation, workspace-safe operations

```
┌──────────────────────────────────┐
│ Codex Spawner Agent              │
├──────────────────────────────────┤
│ Model: OpenAI GPT-4 (Codex)      │
│ Cost: Paid (OpenAI)              │
│ Context: 128K tokens             │
│ Capabilities:                    │
│  - code_generation               │
│  - implementation                │
│  - file_operations               │
└──────────────────────────────────┘
```

**When to Use:**
- Code generation and implementation
- Fixing bugs with workspace operations
- Creating boilerplate and templates
- File modifications with sandbox constraints
- When specialized code knowledge needed

**CLI Tool**: `codex` (npm: `@openai/codex-cli`)

**Installation:**
```bash
npm install -g @openai/codex-cli
codex config set api_key "sk-..."
```

**Example:**
```python
Task(
    subagent_type="codex",
    prompt="Implement a REST API for user management",
    sandbox="workspace-write"
)
```

---

### 3. Copilot Spawner (SUBSCRIPTION)

**Best for**: GitHub-integrated workflows, git operations, PR handling

```
┌──────────────────────────────────┐
│ Copilot Spawner Agent            │
├──────────────────────────────────┤
│ Model: GitHub Copilot            │
│ Cost: Subscription (GitHub)      │
│ Context: 100K tokens             │
│ Capabilities:                    │
│  - github_integration            │
│  - git_operations                │
│  - pr_handling                   │
└──────────────────────────────────┘
```

**When to Use:**
- GitHub-specific operations
- Pull request management
- Git workflow operations
- Branch management
- GitHub API integration

**CLI Tool**: `gh` (GitHub CLI)

**Installation:**
```bash
# macOS
brew install gh

# Ubuntu/Debian
sudo apt update && sudo apt install gh

# Windows
choco install gh

# Authenticate
gh auth login
```

**Example:**
```python
Task(
    subagent_type="copilot",
    prompt="Create a pull request for feature X and request review"
)
```

---

## Agent Selection Guide

### Decision Matrix

| Task Type | Recommendation | Reason | Cost |
|-----------|---|---|---|
| **Exploration** | Gemini | FREE, 2M context, great analysis | FREE |
| **Code Generation** | Codex | Specialized for code, sandbox safe | PAID |
| **Bug Fixes** | Codex | Implementation with workspace access | PAID |
| **Git Operations** | Copilot | Native GitHub integration | INCLUDED |
| **Analysis** | Gemini | Can process large amounts of text | FREE |
| **Documentation** | Gemini | Large context for docs, FREE | FREE |
| **PR Management** | Copilot | GitHub-aware PR handling | INCLUDED |
| **Infrastructure** | Codex | Code generation for config/IaC | PAID |

### Capability Matching

**Use this when task doesn't fit standard categories:**

```python
from wipnote import SDK

sdk = SDK(agent="orchestrator")

# Load agent capabilities
import json
with open("packages/claude-plugin/.claude-plugin/plugin.json") as f:
    config = json.load(f)

agents = config["agents"]

# Scenario: Need to generate CloudFormation templates
required_capabilities = ["code_generation", "infrastructure"]

for agent_name, agent_config in agents.items():
    agent_caps = agent_config.get("capabilities", [])
    match = len(set(required_capabilities) & set(agent_caps))

    if match > 0:
        print(f"{agent_name}: {match}/{len(required_capabilities)} match")
        # Use highest match
```

---

## Installation & Setup

### Step 1: Install Gemini CLI

```bash
# Install
npm install -g @google/generative-ai-cli

# Verify
gemini --version
# → @google/generative-ai-cli/0.1.0

# Get API key from: https://aistudio.google.com/app/apikeys
# Set up authentication
export GOOGLE_API_KEY="your-key-here"

# Or configure permanently
gemini config set api_key "your-key-here"

# Test
gemini chat "Hello, how are you?"
```

### Step 2: Install Codex CLI

```bash
# Install (npm recommended)
npm install -g @openai/codex-cli

# Or pip (if available)
pip install openai-codex-cli

# Verify
codex --version

# Get API key from: https://platform.openai.com/account/api-keys
# Set up authentication
export OPENAI_API_KEY="sk-..."

# Or configure permanently
codex config set api_key "sk-..."

# Test
codex chat "Hello, how are you?"
```

### Step 3: Install GitHub CLI

```bash
# macOS (Homebrew)
brew install gh

# Ubuntu/Debian
sudo apt update
sudo apt install gh

# Windows (Chocolatey)
choco install gh

# Or other methods at: https://cli.github.com/

# Verify
gh --version

# Authenticate
gh auth login
# Follow interactive prompts

# Verify
gh auth status
```

### Step 4: Test All Spawners

```bash
#!/bin/bash
# test-spawners.sh

echo "=== Testing Spawner Availability ==="

# Gemini
echo ""
echo "Testing Gemini..."
if command -v gemini &> /dev/null; then
    echo "✅ Gemini installed"
    gemini --version
else
    echo "❌ Gemini not found. Install: npm install -g @google/generative-ai-cli"
fi

# Codex
echo ""
echo "Testing Codex..."
if command -v codex &> /dev/null; then
    echo "✅ Codex installed"
    codex --version
else
    echo "❌ Codex not found. Install: npm install -g @openai/codex-cli"
fi

# GitHub CLI
echo ""
echo "Testing GitHub CLI..."
if command -v gh &> /dev/null; then
    echo "✅ GitHub CLI installed"
    gh --version
else
    echo "❌ GitHub CLI not found. Install: brew install gh (or apt/choco)"
fi
```

---

## Usage Examples

### Example 1: Exploratory Analysis

```python
from wipnote import SDK, Task

sdk = SDK(agent="orchestrator")

# Delegate codebase analysis to Gemini (FREE)
result = Task(
    subagent_type="gemini",
    prompt="""
    Analyze the codebase and provide:
    1. Architecture overview
    2. Main components and their responsibilities
    3. Key design patterns used
    4. Potential improvements
    """,
    include_directories=["src/", "docs/"]
)

print(f"Analysis complete: {result['agent']}")
print(f"Cost: {result['cost']}")
print(f"Duration: {result['duration']}s")
print(f"\nFindings:\n{result['response']}")
```

### Example 2: Code Implementation

```python
# Delegate code generation to Codex (PAID)
result = Task(
    subagent_type="codex",
    prompt="""
    Implement a user authentication module with:
    1. Password hashing with bcrypt
    2. JWT token generation
    3. Token validation middleware
    4. Unit tests
    """,
    sandbox="workspace-write"
)

if result['success']:
    print(f"✅ Implementation complete")
    print(f"Cost: ${result.get('cost_estimate', 'calculated')}")
else:
    print(f"❌ Failed: {result['error']}")
```

### Example 3: GitHub Operations

```python
# Delegate PR creation to Copilot (INCLUDED)
result = Task(
    subagent_type="copilot",
    prompt="""
    Create a pull request with:
    - Title: "feat: add user authentication"
    - Description: Include changes made
    - Request review from: team-security
    - Link to issue #123
    """,
    allow_tools=["shell", "git"]
)

if result['success']:
    print(f"✅ PR created: {result['response']}")
else:
    print(f"❌ Failed: {result['error']}")
```

### Example 4: Cost-Optimized Multi-Task

```python
from wipnote import Task

# Phase 1: Exploration (FREE with Gemini)
exploration = Task(
    subagent_type="gemini",
    prompt="Find all database query files and analyze them"
)

# Extract findings
if exploration['success']:
    findings = exploration['response']

    # Phase 2: Implementation (PAID with Codex, only if needed)
    implementation = Task(
        subagent_type="codex",
        prompt=f"""
        Based on these findings:
        {findings}

        Create optimized query patterns and apply to codebase.
        """,
        sandbox="workspace-write"
    )

    print(f"Total cost: exploration FREE + implementation PAID")
```

---

## Error Handling

### Common Errors and Solutions

#### Error: CLI Not Found

**Message:**
```json
{
  "success": false,
  "error": "Gemini CLI not found. Install with: npm install -g @google/generative-ai-cli",
  "agent": "gemini-2.0-flash"
}
```

**Solution:**
```bash
# 1. Check if installed
which gemini
gemini --version

# 2. Install if missing
npm install -g @google/generative-ai-cli

# 3. Verify npm path is in PATH
echo $PATH
npm bin -g  # Should be in PATH

# 4. Retry
python your_script.py
```

#### Error: Authentication Failed

**Message:**
```json
{
  "success": false,
  "error": "Authentication failed: invalid API key",
  "agent": "gemini-2.0-flash"
}
```

**Solution:**
```bash
# 1. Check if API key is set
echo $GOOGLE_API_KEY

# 2. Get new API key
# Gemini: https://aistudio.google.com/app/apikeys
# Codex: https://platform.openai.com/account/api-keys
# GitHub: gh auth login

# 3. Set API key
export GOOGLE_API_KEY="your-new-key"
gemini config set api_key "your-new-key"

# 4. Test authentication
gemini chat "test"
```

#### Error: Network/Timeout

**Message:**
```json
{
  "success": false,
  "error": "Request timeout after 120 seconds",
  "agent": "gemini-2.0-flash"
}
```

**Solution:**
```python
# Increase timeout for large tasks
result = Task(
    subagent_type="gemini",
    prompt="Your task",
    timeout=300  # 5 minutes instead of default 120s
)

# Or break task into smaller chunks
results = []
for chunk in large_codebase:
    results.append(Task(
        subagent_type="gemini",
        prompt=f"Analyze: {chunk}"
    ))
```

---

## Fallback Strategies

### Strategy 1: Cost-Optimized Fallback

```python
from wipnote import Task

def delegate_with_cost_optimization(prompt, task_type="general"):
    """Try agents in cost order: FREE → INCLUDED → PAID"""

    # Define cost preference
    cost_order = {
        "exploration": ["gemini", "codex"],  # Gemini first (FREE)
        "git": ["copilot", "codex"],         # Copilot first (INCLUDED)
        "implementation": ["codex"],          # Only Codex
    }

    agents = cost_order.get(task_type, ["gemini", "copilot", "codex"])

    for agent_type in agents:
        try:
            result = Task(subagent_type=agent_type, prompt=prompt)

            if result.get("success"):
                print(f"✅ Completed with {agent_type}")
                return result

        except Exception as e:
            print(f"⚠️  {agent_type} failed: {e}")
            continue

    raise RuntimeError(f"No agents available for: {task_type}")

# Usage
result = delegate_with_cost_optimization(
    prompt="Analyze authentication module",
    task_type="exploration"
)
```

### Strategy 2: Capability-Based Fallback

```python
from wipnote import Task
import json

def delegate_with_capability_matching(prompt, required_capabilities):
    """Find agent with best capability match"""

    # Load agent config
    with open("packages/claude-plugin/.claude-plugin/plugin.json") as f:
        config = json.load(f)

    agents = config["agents"]

    # Score agents
    scored = []
    for agent_name, agent_config in agents.items():
        agent_caps = set(agent_config.get("capabilities", []))
        required = set(required_capabilities)
        match_count = len(agent_caps & required)
        score = match_count / len(required) if required else 0

        scored.append((agent_name, score, agent_config))

    # Sort by score (highest first)
    scored.sort(key=lambda x: x[1], reverse=True)

    # Try each in order
    for agent_name, score, config in scored:
        if score > 0:  # Has at least some match
            try:
                result = Task(subagent_type=agent_name, prompt=prompt)
                if result.get("success"):
                    return result
            except:
                continue

    raise RuntimeError("No agents with matching capabilities")

# Usage
result = delegate_with_capability_matching(
    prompt="Generate REST API code",
    required_capabilities=["code_generation"]
)
```

### Strategy 3: Intelligent Error Recovery

```python
def delegate_with_recovery(agent_type, prompt, max_retries=2):
    """Delegate with intelligent error handling and recovery"""

    from wipnote import Task
    import time

    for attempt in range(max_retries):
        try:
            print(f"[{attempt + 1}/{max_retries}] Trying {agent_type}...")

            result = Task(subagent_type=agent_type, prompt=prompt)

            if result.get("success"):
                return result

            error = result.get("error", "Unknown error")

            # Analyze error type
            if "CLI not found" in error:
                print(f"⚠️  {agent_type} CLI missing")

                # Provide install instructions
                if agent_type == "gemini":
                    print("Install: npm install -g @google/generative-ai-cli")
                elif agent_type == "codex":
                    print("Install: npm install -g @openai/codex-cli")
                elif agent_type == "copilot":
                    print("Install: brew install gh (or apt install gh)")

                # Try fallback
                fallback_map = {
                    "gemini": "codex",
                    "codex": "copilot",
                    "copilot": "gemini"
                }

                next_agent = fallback_map.get(agent_type)
                if next_agent:
                    print(f"Trying fallback: {next_agent}...")
                    return delegate_with_recovery(next_agent, prompt, max_retries - 1)

            elif "timeout" in error.lower():
                print(f"⚠️  Timeout. Retrying in 5s...")
                time.sleep(5)
                continue

            elif "authentication" in error.lower():
                print(f"⚠️  Authentication failed. Check API keys.")
                return result

        except Exception as e:
            print(f"❌ Exception: {e}")
            if attempt < max_retries - 1:
                print("Retrying...")
                time.sleep(2)

    raise RuntimeError(f"Failed after {max_retries} attempts")

# Usage
result = delegate_with_recovery("gemini", "Analyze the codebase")
```

---

## Event Tracking

### Understanding Event Records

Every spawner delegation creates an event record in Wipnote:

```python
{
  "event_id": "event-abc123",
  "event_type": "delegation",
  "agent_id": "orchestrator",
  "tool_name": "Task",
  "session_id": "session-xyz789",

  # Parent context for session linking
  "parent_event_id": "task-def456",
  "parent_query_event": "query-ghi789",

  # What was executed
  "context": {
    "spawned_agent": "gemini-2.0-flash",
    "spawner_type": "gemini",
    "model": "gemini-2.0-flash",
    "cost": "FREE",
    "timeout": 120
  },

  # Results
  "input_summary": "Analyze codebase...",
  "output_summary": "Found 12 API endpoints...",
  "status": "completed",
  "execution_duration_seconds": 15.3,
  "cost_tokens": 0,

  # Timestamps
  "created_at": "2025-01-10T12:34:56Z",
  "updated_at": "2025-01-10T12:35:11Z"
}
```

### Query Events

```python
from wipnote import SDK

sdk = SDK(agent="orchestrator")

# Get all delegation events
sessions = sdk.sessions.all()

for session in sessions:
    events = session.get_events()

    delegations = [
        e for e in events
        if e.get("event_type") == "delegation"
    ]

    print(f"Session {session.id}: {len(delegations)} delegations")

    for event in delegations:
        spawner = event.get("context", {}).get("spawner_type")
        cost = event.get("context", {}).get("cost")
        duration = event.get("execution_duration_seconds")

        print(f"  - {spawner} ({cost}): {duration}s")
```

### Cost Analysis

```python
# Calculate total costs
total_free = 0
total_paid = 0

for event in all_events:
    if event.get("event_type") == "delegation":
        cost_model = event.get("context", {}).get("cost")

        if cost_model == "FREE":
            total_free += 1
        elif cost_model == "PAID":
            total_paid += 1

print(f"FREE delegations: {total_free} (Gemini)")
print(f"PAID delegations: {total_paid} (Codex)")
print(f"INCLUDED delegations: {len(all_events) - total_free - total_paid} (Copilot)")
```

---

## Performance Optimization

### 1. Batch Similar Tasks

```python
# ❌ BAD: Multiple sequential calls
Task(subagent_type="gemini", prompt="Analyze module A")
Task(subagent_type="gemini", prompt="Analyze module B")
Task(subagent_type="gemini", prompt="Analyze module C")

# ✅ GOOD: Single batch call
Task(
    subagent_type="gemini",
    prompt="""
    Analyze all three modules and provide:
    1. Module A analysis
    2. Module B analysis
    3. Module C analysis
    """
)
```

### 2. Use Appropriate Agent

```python
# ❌ BAD: Using expensive Codex for free work
Task(subagent_type="codex", prompt="Find all TODO comments")

# ✅ GOOD: Use free Gemini for analysis
Task(subagent_type="gemini", prompt="Find all TODO comments")
```

### 3. Add Request Context Hints

```python
# Give agent enough context to be efficient
Task(
    subagent_type="gemini",
    prompt="""
    You have access to these directories: src/, tests/, docs/

    Please analyze authentication patterns and suggest improvements.
    Focus on: security, performance, maintainability.
    """,
    include_directories=["src/", "tests/"]
)
```

### 4. Cache Results

```python
from functools import lru_cache
from wipnote import Task

@lru_cache(maxsize=10)
def analyze_codebase_pattern(pattern: str):
    """Cache analysis results"""
    return Task(
        subagent_type="gemini",
        prompt=f"Find all instances of: {pattern}"
    )

# First call - executes
result1 = analyze_codebase_pattern("async function")

# Second call - returns cached result
result2 = analyze_codebase_pattern("async function")
```

### 5. Set Appropriate Timeouts

```python
# Simple tasks: use default (120s)
Task(subagent_type="gemini", prompt="Summarize this file")

# Complex analysis: increase timeout
Task(
    subagent_type="gemini",
    prompt="Analyze entire codebase architecture",
    timeout=300  # 5 minutes
)

# Quick checks: reduce timeout
Task(
    subagent_type="copilot",
    prompt="Check if branch exists",
    timeout=30  # 30 seconds
)
```

---

## FAQ

### Q: How much does each agent cost?

**A:**
- **Gemini**: FREE (Google's public API)
- **Codex**: PAID (OpenAI credits, ~$0.02/1K tokens)
- **Copilot**: SUBSCRIPTION (included with GitHub Copilot subscription)

See [Agent Selection Guide](#agent-selection-guide) for cost-based routing.

### Q: Can I use all three agents in one workflow?

**A:** Yes! Use cost-optimized delegation:

```python
# Phase 1: FREE analysis with Gemini
exploration = Task(subagent_type="gemini", prompt="Analyze...")

# Phase 2: PAID implementation with Codex
implementation = Task(subagent_type="codex", prompt="Implement...")

# Phase 3: INCLUDED GitHub ops with Copilot
pr_creation = Task(subagent_type="copilot", prompt="Create PR...")
```

### Q: What if a CLI tool crashes?

**A:** The spawner agent will catch the error and return:

```json
{
  "success": false,
  "error": "CLI process exited with code 1: [error details]",
  "agent": "gemini-2.0-flash"
}
```

Your code can catch this and implement fallback logic (see [Fallback Strategies](#fallback-strategies)).

### Q: Can I limit which agents are available?

**A:** Yes, modify `plugin.json`:

```json
{
  "agents": {
    "gemini": { ... },      // Keep this
    "codex": null,          // Disable this
    "copilot": { ... }      // Keep this
  }
}
```

Then update Task() calls to use only available agents.

### Q: How are spawner agents different from Claude Code plugins?

**A:**
- **Spawner agents**: Autonomous subagents that execute work and return results
- **Claude Code plugins**: Extend Claude Code itself (hooks, skills, agents)

Spawner agents run ON TOP OF Claude Code (via the plugin infrastructure).

### Q: Can I add a fourth spawner agent?

**A:** Yes! Follow the pattern in `packages/claude-plugin/.claude-plugin/agents/`:

1. Create `your-spawner.py` with standard argument parsing
2. Add entry to `plugin.json` agents section
3. Document in this guide
4. Test before deploying

### Q: How do I monitor spawner costs?

**A:** Query Wipnote events:

```python
from wipnote import SDK

sdk = SDK(agent="orchestrator")

# Get cost summary
for session in sdk.sessions.all():
    costs = {}

    for event in session.get_events():
        if event.get("event_type") == "delegation":
            cost = event.get("context", {}).get("cost")
            costs[cost] = costs.get(cost, 0) + 1

    print(f"{session.id}: {costs}")
```

---

## Related Documentation

- [CLAUDE.md - Multi-AI Orchestration Section](../CLAUDE.md#multi-ai-orchestration-via-spawner-agents)
- [AGENTS.md - Complete SDK Documentation](../AGENTS.md)
- [plugin.json - Agent Configuration](../packages/claude-plugin/.claude-plugin/plugin.json)
- [Wipnote Architecture](./ARCHITECTURE.md)

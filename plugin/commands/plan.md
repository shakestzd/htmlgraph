# /htmlgraph:plan

Start planning a new track with spike or create directly. Uses strategic analytics to provide project context and creates structured tracks with specs and implementation plans.

**⚠️ IMPORTANT: Research First for Complex Features**

For complex features (auth, security, real-time, integrations), you should **complete research BEFORE planning**:

1. Use `/htmlgraph:research "{topic}"` to gather best practices
2. Document findings (libraries, patterns, anti-patterns)
3. Then use `/htmlgraph:plan` with research-informed context

This research-first approach:
- ✅ Avoids reinventing wheels
- ✅ Learns from others' mistakes
- ✅ Chooses right tools upfront
- ✅ Reduces context usage (targeted vs exploratory)

**DELEGATION PATTERN**:
- Research phase → `Task(subagent_type="htmlgraph:researcher")`
- Simple fixes (1-2 files) → `Task(subagent_type="htmlgraph:haiku-coder")`
- Features (3-8 files) → `Task(subagent_type="htmlgraph:sonnet-coder")` (default)
- Architecture (10+ files) → `Task(subagent_type="htmlgraph:opus-coder")`
- Validation → `Task(subagent_type="htmlgraph:test-runner")`

## Usage

```
/htmlgraph:plan <description> [--spike] [--timebox HOURS]
```

## Parameters

- `description` (required): What you want to plan (e.g., "User authentication system")
- `--spike` (optional) (default: True): Create a planning spike first (recommended for complex work)
- `--timebox` (optional) (default: 4.0): Time limit for spike in hours


## Examples

```bash
# RECOMMENDED: Research first for complex features
/htmlgraph:research "OAuth 2.0 implementation patterns"
/htmlgraph:plan "User authentication system"
```
Research best practices, then create planning spike

```bash
/htmlgraph:plan "Real-time notifications" --timebox 3
```
Create planning spike with 3-hour timebox

```bash
/htmlgraph:plan "Simple bug fix dashboard" --no-spike
```
Create track directly without spike (use for simple, well-defined work)


## Instructions for Claude

**⚠️ CRITICAL: Check for Research Before Planning**

Before creating the plan, check if research was completed:
1. Check if `/htmlgraph:research` was used previously in the conversation
2. If complex feature WITHOUT research → Warn and suggest research first
3. If research completed → incorporate findings into the spike/track description

### Implementation:

**STEP 1: Check if research was completed**

Look for research findings in conversation context. If not done for complex features, warn:

```
⚠️  Warning: Complex feature detected without research.
RECOMMENDED: Run /htmlgraph:research first to gather best practices.
Example: /htmlgraph:research "{description}"
```

**STEP 2: Get project context**

```bash
htmlgraph analytics summary
htmlgraph analytics summary
```

**STEP 3: Create spike or track**

For spike (default, recommended for complex work):
```bash
htmlgraph spike create "Plan: {description}"
```

For track (well-defined work):
```bash
htmlgraph track new "{title}"
```

**STEP 4: Display result**

Show spike/track ID and next steps based on CLI output.

### Creating Tracks (Advanced)

If the spike reveals a well-defined plan, create a track directly:

```bash
htmlgraph track new "User Authentication System"
```

Then create features under it:
```bash
htmlgraph feature create "Phase 1: OAuth Setup"
htmlgraph feature create "Phase 2: JWT Implementation"
htmlgraph feature create "Phase 3: Testing"
```

### Workflow Guidance

**1. Complex/Undefined Work → Use Spike:**
```bash
/htmlgraph:plan "Real-time collaboration features" --spike --timebox 6
```
- Research technical approaches
- Explore libraries/tools
- Identify risks and unknowns
- Draft requirements and plan
- Then create track from spike findings

**2. Well-Defined Work → Create Track Directly:**
```bash
/htmlgraph:plan "Add dark mode toggle" --no-spike
```
- Requirements are clear
- Implementation is straightforward
- No research needed
- Can proceed immediately

**3. During Spike → Reduce Exploratory Reads:**
When working in a planning spike, you should:
- Focus on specific research questions
- Document findings in spike notes
- Draft requirements as you discover them
- Create structured plan with phases
- Avoid reading entire codebases - use targeted searches

**Example spike workflow:**
```bash
# 1. View spike details
htmlgraph spike show {spike_id}

# 2. Research focused questions
# Instead of: Read entire auth module
# Do: Search for specific patterns

# 3. Create track from findings
htmlgraph track new "User Authentication"
htmlgraph feature create "Configure OAuth providers"
htmlgraph feature create "Implement JWT signing"
htmlgraph feature create "Write integration tests"
```

### Output Format:

```
## Planning Started

**Type:** {type}
**Title:** {title}
**ID:** {spike_id or track_id}
**Status:** {status}

### Project Context
- Bottlenecks: {project_context.bottlenecks_count}
- High-risk items: {project_context.high_risk_count}
- Parallel capacity: {project_context.parallel_capacity}

### What This Means
{context_interpretation}

### Next Steps
{next_steps}
```

**Context Interpretation Examples:**
- "3 bottlenecks detected - consider if this work helps unblock them"
- "5 high-risk items - ensure this doesn't add more complexity"
- "4 agents can work in parallel - look for parallelizable tasks"

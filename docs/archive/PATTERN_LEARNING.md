# Pattern Learning from Agent Behavior

**Phase 2 Feature 2: Learn from tool call sequences to identify patterns, anti-patterns, and optimization opportunities.**

## Overview

Pattern Learning analyzes historical tool call sequences to discover workflow patterns, identify anti-patterns, and recommend optimizations. This creates a learning loop that helps AI agents improve their efficiency over time.

## What It Does

The Pattern Learning system:

1. **Detects Patterns**: Identifies common sequences of tool calls (e.g., Read → Grep → Edit → Bash)
2. **Calculates Metrics**: Tracks frequency, success rate, and duration for each pattern
3. **Generates Insights**: Creates actionable recommendations, anti-patterns, and optimization opportunities
4. **Learns Over Time**: Stores patterns and user feedback to improve recommendations
5. **Exports Reports**: Generates markdown reports for team sharing

## Key Concepts

### Pattern
A sequence of tool calls that occurs frequently (e.g., `["Read", "Grep", "Edit"]`).

**Pattern Attributes:**
- `sequence`: List of tool names in order
- `frequency`: Number of times the pattern occurs
- `success_rate`: Percentage of times pattern led to successful outcomes
- `avg_duration_seconds`: Average execution time
- `sessions`: List of session IDs where pattern occurred
- `user_feedback`: User rating (1=helpful, 0=neutral, -1=unhelpful)

### Insight
An actionable recommendation or warning derived from pattern analysis.

**Insight Types:**
- **Recommendation**: High-success patterns worth replicating
- **Anti-Pattern**: Low-success patterns to avoid
- **Optimization**: Patterns that could be improved (e.g., too many Read operations)

**Insight Attributes:**
- `title`: Human-readable title
- `description`: Detailed explanation
- `impact_score`: Estimated impact (0-100)
- `patterns`: Related pattern IDs

## API Reference

### Pattern Detection

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Detect patterns with sliding window
patterns = sdk.pattern_learning.detect_patterns(
    window_size=3,      # Size of tool sequence window (default: 3)
    min_frequency=5     # Minimum occurrences (default: 5)
)

# Patterns are automatically stored in database for learning loop
```

### Get Recommendations

```python
# Get top 3 recommendations based on high-success patterns
recommendations = sdk.pattern_learning.get_recommendations(limit=3)

for rec in recommendations:
    print(f"{rec.title}")
    print(f"  {rec.description}")
    print(f"  Impact: {rec.impact_score:.1f}")
```

**Example Output:**
```
High Success Pattern: Read → Grep → Edit
  This pattern has a 87.5% success rate across 10 occurrences. Consider using this workflow for similar tasks.
  Impact: 87.5
```

### Identify Anti-Patterns

```python
# Get detected anti-patterns
anti_patterns = sdk.pattern_learning.get_anti_patterns()

for anti in anti_patterns[:5]:
    print(f"⚠️ {anti.title}")
    print(f"  {anti.description}")
```

**Example Output:**
```
⚠️ Low Success Pattern: Edit → Edit → Edit → Bash
  This pattern has only a 40.0% success rate across 5 occurrences. Consider alternative approaches.
```

### Generate All Insights

```python
# Get all insights (recommendations + anti-patterns + optimizations)
insights = sdk.pattern_learning.generate_insights()

# Filter by type
recommendations = [i for i in insights if i.insight_type == "recommendation"]
anti_patterns = [i for i in insights if i.insight_type == "anti-pattern"]
optimizations = [i for i in insights if i.insight_type == "optimization"]
```

### Export Learnings

```python
# Export to markdown for team sharing
sdk.pattern_learning.export_learnings("pattern_report.md")
```

**Generated Report Includes:**
- Top recommendations
- Detected anti-patterns
- Optimization opportunities
- All detected patterns with metrics

### User Feedback

```python
# Provide feedback to improve recommendations
pattern_id = patterns[0].pattern_id

# Mark as helpful
sdk.pattern_learning.learning_loop.update_feedback(pattern_id, 1)

# Mark as neutral
sdk.pattern_learning.learning_loop.update_feedback(pattern_id, 0)

# Mark as unhelpful
sdk.pattern_learning.learning_loop.update_feedback(pattern_id, -1)
```

## How It Works

### 1. Pattern Detection

**PatternMatcher** uses a sliding window to extract tool sequences:

```
Tool Calls: [Read, Grep, Edit, Bash, Read, Grep]
Window Size: 3

Sequences Extracted:
  - [Read, Grep, Edit]
  - [Grep, Edit, Bash]
  - [Edit, Bash, Read]
  - [Bash, Read, Grep]
```

Sequences are counted and filtered by minimum frequency.

### 2. Metric Calculation

**InsightGenerator** enriches patterns with metrics:

- **Success Rate**: Percentage of sessions with completions > errors
- **Average Duration**: Mean execution time across all occurrences
- **Frequency Distribution**: Which sessions used this pattern

### 3. Insight Generation

Insights are generated based on pattern analysis:

**Recommendations** (success_rate ≥ 80% AND frequency ≥ 5):
```python
if pattern.success_rate >= 80 and pattern.frequency >= 5:
    # High-success pattern worth replicating
```

**Anti-Patterns** (success_rate < 50% AND frequency ≥ 5):
```python
if pattern.success_rate < 50 and pattern.frequency >= 5:
    # Low-success pattern to avoid
```

**Optimizations** (multiple Read operations):
```python
if pattern.sequence.count("Read") >= 2:
    # Consider delegating exploration to subagent
```

### 4. Learning Loop

**LearningLoop** stores patterns in database:

```sql
CREATE TABLE tool_patterns (
    pattern_id TEXT PRIMARY KEY,
    tool_sequence TEXT NOT NULL,
    frequency INTEGER DEFAULT 0,
    success_rate REAL DEFAULT 0.0,
    avg_duration_seconds REAL DEFAULT 0.0,
    last_seen TIMESTAMP,
    sessions TEXT,
    user_feedback INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

User feedback refines future recommendations.

## Example Workflow

### Detect Patterns and Get Insights

```python
from wipnote import SDK

# Initialize SDK
sdk = SDK(agent="claude")

# Detect patterns from tool call history
patterns = sdk.pattern_learning.detect_patterns(
    window_size=3,
    min_frequency=5
)

print(f"Found {len(patterns)} patterns")

# Generate insights
insights = sdk.pattern_learning.generate_insights()

# Show top recommendations
recommendations = [i for i in insights if i.insight_type == "recommendation"]
for rec in recommendations[:3]:
    print(f"\n✅ {rec.title}")
    print(f"   {rec.description}")
    print(f"   Impact: {rec.impact_score:.1f}")

# Show anti-patterns
anti_patterns = [i for i in insights if i.insight_type == "anti-pattern"]
for anti in anti_patterns[:3]:
    print(f"\n⚠️ {anti.title}")
    print(f"   {anti.description}")
    print(f"   Impact: {anti.impact_score:.1f}")

# Export report for team
sdk.pattern_learning.export_learnings(".wipnote/pattern_report.md")
```

### Provide Feedback

```python
# Get stored pattern
pattern = sdk.pattern_learning.learning_loop.get_pattern(pattern_id)

# Update feedback based on usefulness
sdk.pattern_learning.learning_loop.update_feedback(pattern_id, 1)

# Re-generate insights (feedback influences future recommendations)
insights = sdk.pattern_learning.generate_insights()
```

## Performance

The pattern learning system is designed for fast analysis:

- **<1 second** analysis of 1000 tool call events
- **Incremental updates**: Only new patterns are analyzed
- **Efficient storage**: Patterns stored in SQLite with indexes
- **Lazy evaluation**: Insights generated on-demand

## Use Cases

### 1. Workflow Optimization

Identify most efficient tool call patterns:

```python
patterns = sdk.pattern_learning.detect_patterns(min_frequency=10)

# Find high-success patterns
efficient = [p for p in patterns if p.success_rate > 80]

print("Most efficient workflows:")
for pattern in efficient[:5]:
    print(f"  {' → '.join(pattern.sequence)}")
    print(f"  Success: {pattern.success_rate:.1f}%")
```

### 2. Onboarding New Agents

Share best practices with new team members:

```python
# Export learnings for new agent
sdk.pattern_learning.export_learnings("docs/workflow_best_practices.md")
```

### 3. Debug Inefficiencies

Identify patterns that lead to failures:

```python
anti_patterns = sdk.pattern_learning.get_anti_patterns()

for anti in anti_patterns:
    print(f"⚠️ Avoid: {' → '.join(anti.patterns)}")
    print(f"   {anti.description}")
```

### 4. Cost Optimization

Find patterns with excessive tool use:

```python
insights = sdk.pattern_learning.generate_insights()

optimizations = [i for i in insights if i.insight_type == "optimization"]
for opt in optimizations:
    print(f"💡 {opt.title}")
    print(f"   {opt.description}")
```

## Configuration

### Pattern Detection Parameters

```python
patterns = sdk.pattern_learning.detect_patterns(
    window_size=3,      # Sequence length (2-5 recommended)
    min_frequency=5     # Minimum occurrences (adjust based on data volume)
)
```

**Recommendations:**
- **Small teams**: `min_frequency=3`
- **Medium teams**: `min_frequency=5`
- **Large teams**: `min_frequency=10`

### Insight Generation Thresholds

**Success Rate Thresholds** (hardcoded in `InsightGenerator`):
- **Recommendation**: `success_rate >= 80%`
- **Anti-Pattern**: `success_rate < 50%`

**Optimization Detection**:
- **Multiple Reads**: `sequence.count("Read") >= 2`

## Database Schema

### tool_patterns Table

```sql
CREATE TABLE tool_patterns (
    pattern_id TEXT PRIMARY KEY,           -- Unique pattern ID
    tool_sequence TEXT NOT NULL,           -- "Read->Grep->Edit"
    frequency INTEGER DEFAULT 0,           -- Occurrence count
    success_rate REAL DEFAULT 0.0,         -- Success percentage
    avg_duration_seconds REAL DEFAULT 0.0, -- Average execution time
    last_seen TIMESTAMP,                   -- Last occurrence timestamp
    sessions TEXT,                         -- JSON list of session IDs
    user_feedback INTEGER DEFAULT 0,       -- User rating
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

## Troubleshooting

### No Patterns Detected

**Problem**: `detect_patterns()` returns empty list

**Solutions**:
- Lower `min_frequency` threshold
- Check database has tool call events: `SELECT COUNT(*) FROM agent_events WHERE event_type='tool_call'`
- Verify window size is appropriate for your workflow

### Success Rate Always 0%

**Problem**: All patterns show 0% success rate

**Cause**: Database lacks completion/error events

**Solution**: Ensure hooks track completion/error events:
```python
cursor.execute("""
    INSERT INTO agent_events (event_type, session_id, timestamp)
    VALUES ('completion', ?, ?)
""", (session_id, datetime.now()))
```

### Performance Issues

**Problem**: `detect_patterns()` is slow

**Solutions**:
- Increase `min_frequency` to reduce pattern count
- Reduce `window_size` for faster processing
- Add database indexes on `session_id`, `timestamp`, `tool_name`

## Example Output

### Pattern Report

```markdown
# Pattern Learning Report

Generated: 2026-01-13T10:00:00

## Recommendations

### High Success Pattern: Read → Grep → Edit

This pattern has a 87.5% success rate across 10 occurrences. Consider using this workflow for similar tasks.

**Impact Score**: 87.5

## Anti-Patterns

### Low Success Pattern: Edit → Edit → Edit → Bash

This pattern has only a 40.0% success rate across 5 occurrences. Consider alternative approaches.

**Impact Score**: 60.0

## Optimization Opportunities

### Multiple Read Operations Detected

Pattern 'Read → Read → Read → Edit' contains 3 Read operations. Consider delegating exploration to a subagent to reduce context usage.

**Impact Score**: 120.0

## All Detected Patterns

- **Read → Grep → Edit** (frequency: 10, success: 87.5%)
- **Edit → Bash → Read** (frequency: 8, success: 75.0%)
- **Grep → Read → Edit** (frequency: 6, success: 66.7%)
```

## Next Steps

1. **Integrate with Cost Analyzer**: Correlate patterns with token costs
2. **Add Time Series Analysis**: Track pattern evolution over time
3. **Build Pattern Library**: Create shared repository of best practices
4. **Implement Auto-Suggestions**: Surface patterns in real-time during work

## References

- **Source**: `src/python/wipnote/analytics/pattern_learning.py`
- **Tests**: `tests/python/test_pattern_learning.py`
- **Demo**: `examples/pattern_learning_demo.py`
- **Phase 2 Plan**: See project roadmap for Pattern Learning feature

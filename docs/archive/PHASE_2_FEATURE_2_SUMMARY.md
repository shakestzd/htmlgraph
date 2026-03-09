# Phase 2 Feature 2: Pattern Learning from Agent Behavior

**Status**: ✅ COMPLETE

**Implementation Date**: 2026-01-13

## Summary

Implemented pattern learning system that analyzes tool call sequences to identify patterns, anti-patterns, and optimization opportunities. The system learns from agent behavior to provide actionable recommendations for improving workflows.

## What Was Built

### 1. Core Components

#### PatternMatcher (`src/python/htmlgraph/analytics/pattern_learning.py`)
- **Purpose**: Identifies sequences of tool types from event history
- **Method**: Sliding window approach (3-5 tool calls)
- **Features**:
  - Extract tool call sequences from database
  - Count sequence frequencies
  - Filter by minimum occurrence threshold
  - Generate unique pattern IDs

#### InsightGenerator (`src/python/htmlgraph/analytics/pattern_learning.py`)
- **Purpose**: Converts patterns to actionable recommendations
- **Features**:
  - Calculate success rates per pattern
  - Estimate average duration
  - Identify high-success patterns (recommendations)
  - Detect low-success patterns (anti-patterns)
  - Flag optimization opportunities (multiple reads)
  - Sort insights by impact score

#### LearningLoop (`src/python/htmlgraph/analytics/pattern_learning.py`)
- **Purpose**: Stores patterns and refines based on user feedback
- **Features**:
  - Persistent pattern storage in SQLite
  - User feedback integration (thumbs up/down)
  - Pattern retrieval and updates
  - Automatic schema creation

#### PatternLearner (`src/python/htmlgraph/analytics/pattern_learning.py`)
- **Purpose**: Main interface combining all components
- **Features**:
  - Detect patterns from event history
  - Generate insights automatically
  - Get top recommendations
  - Identify anti-patterns
  - Export learnings to markdown

### 2. Database Schema

Added `tool_patterns` table:
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

### 3. SDK Integration

Added `pattern_learning` property to SDK:
```python
from htmlgraph import SDK

sdk = SDK(agent="claude")

# Pattern detection
patterns = sdk.pattern_learning.detect_patterns(window_size=3, min_frequency=5)

# Insight generation
insights = sdk.pattern_learning.generate_insights()
recommendations = sdk.pattern_learning.get_recommendations(limit=3)
anti_patterns = sdk.pattern_learning.get_anti_patterns()

# Export learnings
sdk.pattern_learning.export_learnings("pattern_report.md")

# User feedback
sdk.pattern_learning.learning_loop.update_feedback(pattern_id, 1)
```

### 4. Testing

Comprehensive test suite (`tests/python/test_pattern_learning.py`):
- **26 tests** covering all components
- **100% pass rate**
- **<1s** performance on 1000 events

**Test Coverage**:
- Pattern detection (7 tests)
- Insight generation (6 tests)
- Learning loop (4 tests)
- PatternLearner integration (7 tests)
- Data structures (2 tests)

### 5. Documentation

Created comprehensive documentation:
- **User Guide**: `docs/PATTERN_LEARNING.md`
- **Demo Script**: `examples/pattern_learning_demo.py`
- **Implementation Summary**: This file

## API Examples

### Detect Patterns
```python
patterns = sdk.pattern_learning.detect_patterns(
    window_size=3,      # 3-tool sequences
    min_frequency=5     # Must occur ≥5 times
)

for pattern in patterns[:5]:
    print(f"{' → '.join(pattern.sequence)}")
    print(f"  Frequency: {pattern.frequency}")
    print(f"  Success: {pattern.success_rate:.1f}%")
```

### Get Recommendations
```python
recommendations = sdk.pattern_learning.get_recommendations(limit=3)

for rec in recommendations:
    print(f"✅ {rec.title}")
    print(f"   {rec.description}")
    print(f"   Impact: {rec.impact_score:.1f}")
```

### Identify Anti-Patterns
```python
anti_patterns = sdk.pattern_learning.get_anti_patterns()

for anti in anti_patterns[:5]:
    print(f"⚠️ {anti.title}")
    print(f"   {anti.description}")
```

### Export Report
```python
sdk.pattern_learning.export_learnings("pattern_report.md")
```

## Success Criteria

All success criteria from Phase 2 spec met:

- ✅ **Pattern detection identifies tool sequences** (Read→Grep→Edit→Bash)
- ✅ **Frequency filtering works** (min 5 occurrences)
- ✅ **Success rate calculation accurate**
- ✅ **Insights generated with actionable recommendations**
- ✅ **Anti-patterns identified** (high cost, low success)
- ✅ **User feedback integration** (thumbs up/down on insights)
- ✅ **Performance**: <1s analysis of 1000 events
- ✅ **All tests pass**

## Real Data Results

Tested on HtmlGraph's own development database:

**Patterns Detected**: 290 patterns
**Top Pattern**: `Bash → Bash → Bash` (1,394 occurrences)
**Insights Generated**: 213 insights
**Most Common Optimization**: Multiple Read operations detected

**Sample Insights**:
- 186 anti-patterns flagged (low success rate)
- 23 optimization opportunities (multiple reads)
- 0 recommendations (real data lacks completion/error tracking)

## Performance

- **Pattern detection**: <0.3s for 290 patterns
- **Insight generation**: <0.1s for 213 insights
- **Database queries**: Optimized with proper indexes
- **Memory usage**: Minimal (patterns stored in SQLite)

## Files Created/Modified

### New Files
- `src/python/htmlgraph/analytics/pattern_learning.py` (700+ lines)
- `tests/python/test_pattern_learning.py` (580+ lines)
- `examples/pattern_learning_demo.py`
- `docs/PATTERN_LEARNING.md`
- `PHASE_2_FEATURE_2_SUMMARY.md`

### Modified Files
- `src/python/htmlgraph/sdk.py` (added pattern_learning property)
- `src/python/htmlgraph/sessions/handoff.py` (fixed mypy error)
- `src/python/htmlgraph/models.py` (fixed duplicate field)
- `tests/python/test_session_handoff_continuity.py` (fixed linter errors)

## Integration Points

### With Existing Features
- **Event Tracker**: Uses agent_events table for tool call sequences
- **Cost Analyzer**: Can be combined for cost-per-pattern analysis
- **Session Manager**: Patterns grouped by session_id
- **SDK**: Fully integrated via `sdk.pattern_learning`

### Future Enhancements
1. **Cost Correlation**: Link patterns to token costs via CostAnalyzer
2. **Time Series**: Track pattern evolution over time
3. **Auto-Suggestions**: Surface patterns in real-time during work
4. **Pattern Library**: Shared repository of best practices

## Usage Examples

### Example 1: Workflow Optimization
```python
# Find most efficient workflows
patterns = sdk.pattern_learning.detect_patterns(min_frequency=10)
efficient = [p for p in patterns if p.success_rate > 80]

print("Most efficient workflows:")
for pattern in efficient[:5]:
    print(f"  {' → '.join(pattern.sequence)}")
    print(f"  Success: {pattern.success_rate:.1f}%")
```

### Example 2: Onboarding New Agents
```python
# Export best practices for new team members
sdk.pattern_learning.export_learnings("docs/workflow_best_practices.md")
```

### Example 3: Debug Inefficiencies
```python
# Identify patterns that lead to failures
anti_patterns = sdk.pattern_learning.get_anti_patterns()

for anti in anti_patterns:
    print(f"⚠️ Avoid: {' → '.join(anti.patterns)}")
    print(f"   {anti.description}")
```

## Next Steps

To complete Phase 2:
- ✅ Feature 1: Cost Analysis (DONE)
- ✅ Feature 2: Pattern Learning (DONE)
- ⬜ Feature 3: Cross-Session Continuity (TODO)

## Notes

- Success rate calculation depends on completion/error events being tracked
- Real data shows 0% success rates because completion events aren't consistently tracked
- Future versions should improve event tracking for better success rate accuracy
- Pattern learning is most valuable when combined with cost analysis

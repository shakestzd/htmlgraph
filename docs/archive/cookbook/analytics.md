# Strategic Planning & Analytics Recipes

## Find Bottlenecks

**Problem**: Identify tasks blocking the most downstream work.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get top bottlenecks
bottlenecks = sdk.find_bottlenecks(top_n=5)

print("TOP BOTTLENECKS:")
print("=" * 60)
for bn in bottlenecks:
    print(f"{bn['title']}")
    print(f"  Blocks: {bn['blocks_count']} tasks")
    print(f"  Impact Score: {bn['impact_score']:.1f}")
    print(f"  Priority: {bn['priority']}")
    print(f"  Blocked tasks: {', '.join(bn['blocked_tasks'][:3])}")
    print()
```

**Output**:
```
TOP BOTTLENECKS:
============================================================
Database Schema
  Blocks: 5 tasks
  Impact Score: 8.5
  Priority: high
  Blocked tasks: feature-auth, feature-api, feature-profile

OAuth Integration
  Blocks: 3 tasks
  Impact Score: 5.0
  Priority: high
  Blocked tasks: feature-sessions, feature-profile, feature-admin
```

**Explanation**:
- Impact score considers both direct and transitive dependencies
- Higher score = unblocking this task unlocks more work
- Use to prioritize critical path items
- Recomputed dynamically as tasks complete

**Note**: Both calling styles are valid. The top-level convenience method is recommended:
```python
sdk.find_bottlenecks(top_n=5)          # recommended
sdk.dep_analytics.find_bottlenecks(top_n=5)  # explicit, also valid
```
The same applies to `recommend_next_work`, `get_parallel_work`, and `assess_risks`.

---

## Get Work Recommendations

**Problem**: Decide what to work on next.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get recommendations for single agent
recs = sdk.recommend_next_work(agent_count=1)

if recs:
    best = recs[0]
    print("RECOMMENDED NEXT TASK:")
    print(f"  {best['title']}")
    print(f"  Score: {best['score']:.1f}")
    print(f"  Priority: {best['priority']}")
    print(f"  Why: {', '.join(best['reasons'])}")

    if best.get('unlocks_count', 0) > 0:
        print(f"  Unlocks: {best['unlocks_count']} tasks")

    if 'estimated_hours' in best:
        print(f"  Estimated: {best['estimated_hours']}h")
```

**Output**:
```
RECOMMENDED NEXT TASK:
  Database Schema
  Score: 9.5
  Priority: high
  Why: Directly unblocks 5 features, High priority, Critical path item
  Unlocks: 5 tasks
  Estimated: 8h
```

**Explanation**:
- Score combines priority, impact, and readiness
- Higher score = better choice
- Considers dependencies (won't recommend blocked tasks)
- Factors in estimated effort if available

---

## Identify Parallel Work

**Problem**: Find tasks that can be worked on simultaneously.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Get parallel work opportunities
parallel = sdk.get_parallel_work(max_agents=5)

print("PARALLEL WORK CAPACITY:")
print(f"  Max parallelism: {parallel['max_parallelism']}")
print(f"  Ready now: {parallel['ready_now']}")
print(f"  Total ready: {parallel['total_ready']}")
print()

if parallel['ready_now'] > 1:
    print(f"CAN WORK ON {parallel['ready_now']} TASKS IN PARALLEL:")
    for level_idx, task_ids in enumerate(parallel.get('next_level', [])[:5]):
        feature = sdk.features.get(task_ids[0]) if task_ids else None
        if feature:
            print(f"  {level_idx + 1}. {feature.title} (priority: {feature.priority})")
```

**Output**:
```
PARALLEL WORK CAPACITY:
  Max parallelism: 7
  Ready now: 4
  Total ready: 7

CAN WORK ON 4 TASKS IN PARALLEL:
  1. User Profile Endpoint (priority: medium)
  2. Admin Dashboard (priority: medium)
  3. Email Notifications (priority: low)
  4. Documentation Updates (priority: low)
```

**Explanation**:
- Max parallelism = tasks with no dependencies
- Ready now = subset of those with high priority
- Useful for coordinating multiple agents
- Recomputed as dependencies resolve

---

## Assess Project Risks

**Problem**: Identify dependency-related risks.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Run risk assessment
risks = sdk.assess_risks()

print("PROJECT RISK ASSESSMENT:")
print("=" * 60)

# High-risk tasks
if risks['high_risk_count'] > 0:
    print(f"\n⚠️  {risks['high_risk_count']} HIGH-RISK TASKS:")
    for task in risks['high_risk_tasks'][:5]:
        print(f"  {task['title']}")
        print(f"    Risks: {', '.join(task['risk_factors'])}")
        print()

# Circular dependencies
if risks['circular_dependencies']:
    print("🔄 CIRCULAR DEPENDENCIES DETECTED:")
    for cycle in risks['circular_dependencies']:
        print(f"  {' → '.join(cycle)}")
    print()

# Orphaned tasks
if risks['orphaned_count'] > 0:
    print(f"🔗 {risks['orphaned_count']} ORPHANED TASKS (no dependencies)")

# Recommendations
if risks['recommendations']:
    print("\nRECOMMENDATIONS:")
    for rec in risks['recommendations']:
        print(f"  - {rec}")
```

**Output**:
```
PROJECT RISK ASSESSMENT:
============================================================

⚠️  2 HIGH-RISK TASKS:
  Database Migration
    Risks: Single point of failure, Blocks 8 tasks, No backup plan

  OAuth Provider Setup
    Risks: Single point of failure, Blocks 5 tasks

🔄 CIRCULAR DEPENDENCIES DETECTED:
  feature-auth → feature-db → feature-migrations → feature-auth

🔗 3 ORPHANED TASKS (no dependencies)

RECOMMENDATIONS:
  - Break circular dependency between feature-auth and feature-db
  - Consider splitting Database Migration into smaller tasks
  - Add backup plan for OAuth Provider Setup
```

**Explanation**:
- Identifies single points of failure
- Detects circular dependencies (deadlocks)
- Finds orphaned tasks (might be forgotten)
- Provides actionable recommendations

---

## Analyze Impact of Completing a Task

**Problem**: Understand what completing a specific task will unlock.

**Solution**:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# Analyze impact of specific task
impact = sdk.analyze_impact("feature-database")

print(f"IMPACT ANALYSIS: {impact['node_id']}")
print("=" * 60)
print(f"Direct dependents: {len(impact['direct_dependents'])}")
print(f"Total impact: {impact['total_impact']} tasks")
print(f"Completion impact: {impact['completion_impact']:.1f}% of remaining work")
print()

if impact['unlocks_count'] > 0:
    print(f"UNLOCKS {impact['unlocks_count']} TASKS:")
    for task_id in impact['affected_tasks'][:10]:
        feature = sdk.features.get(task_id)
        print(f"  - {feature.title} (priority: {feature.priority})")
```

**Output**:
```
IMPACT ANALYSIS: feature-database
============================================================
Direct dependents: 5
Total impact: 12 tasks
Completion impact: 48.0% of remaining work

UNLOCKS 5 TASKS:
  - User Authentication (priority: high)
  - API Endpoints (priority: high)
  - Admin Dashboard (priority: medium)
  - User Profile (priority: medium)
  - Email Templates (priority: low)
```

**Explanation**:
- Direct dependents = tasks immediately unblocked
- Total impact = direct + transitive dependents
- Completion impact = % of project this represents
- Use to prioritize high-leverage work

---

## Generate Progress Report

**Problem**: Create a summary of current project state.

**Solution**:

```python
from wipnote import SDK
from datetime import datetime

sdk = SDK(agent="claude")

# Gather metrics
all_features = sdk.features.all()
total = len(all_features)
done = len([f for f in all_features if f.status == "done"])
in_progress = len([f for f in all_features if f.status == "in-progress"])
todo = len([f for f in all_features if f.status == "todo"])
blocked = len([f for f in all_features if f.status == "blocked"])

# Get bottlenecks
bottlenecks = sdk.find_bottlenecks(top_n=3)

# Get recommendations
recs = sdk.recommend_next_work(agent_count=3)

# Generate report
print(f"PROJECT PROGRESS REPORT - {datetime.now().strftime('%Y-%m-%d')}")
print("=" * 70)
print()
print(f"OVERALL STATUS:")
print(f"  Total features: {total}")
print(f"  Completed: {done} ({done/total*100:.1f}%)")
print(f"  In Progress: {in_progress}")
print(f"  Todo: {todo}")
print(f"  Blocked: {blocked}")
print()

if bottlenecks:
    print(f"TOP BOTTLENECKS ({len(bottlenecks)}):")
    for bn in bottlenecks:
        print(f"  - {bn['title']} (blocks {bn['blocks_count']} tasks)")
    print()

if recs:
    print(f"RECOMMENDED NEXT TASKS:")
    for i, rec in enumerate(recs[:3], 1):
        print(f"  {i}. {rec['title']} (score: {rec['score']:.1f})")
    print()

print(f"Generated: {datetime.now().isoformat()}")
```

**Use Case**: Daily standup reports, sprint reviews, stakeholder updates

---

## Track Velocity

**Problem**: Measure team/agent productivity over time.

**Solution**:

```python
from wipnote import SDK
from datetime import datetime, timedelta

sdk = SDK(agent="claude")

# Get features completed in last week
week_ago = datetime.now() - timedelta(days=7)
recent_completions = [
    f for f in sdk.features.where(status="done")
    if f.updated and f.updated > week_ago
]

print(f"VELOCITY (Last 7 Days):")
print(f"  Features completed: {len(recent_completions)}")

# Calculate story points (if using estimated hours)
total_hours = 0
for f in recent_completions:
    # Sum step estimates if available
    for step in f.steps:
        # Parse "(2h)" format from descriptions
        import re
        match = re.search(r'\((\d+(?:\.\d+)?)\s*h\)', step.description)
        if match:
            total_hours += float(match.group(1))

if total_hours > 0:
    print(f"  Hours completed: {total_hours}h")
    print(f"  Avg per day: {total_hours/7:.1f}h")
    print(f"  Projected capacity: {total_hours/7 * 30:.1f}h/month")
```

**Explanation**:
- Tracks completion rate over time
- Uses updated timestamp to filter recent work
- Can estimate future capacity
- Useful for sprint planning

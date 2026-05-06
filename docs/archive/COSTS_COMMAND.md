# Cost Dashboard Command - Phase 1 Feature 3

Make cost visibility trivial with `wipnote costs` - one command shows token/cost breakdown by session, feature, tool, or agent.

## Quick Start

```bash
# View last week's costs by session (default)
wipnote costs

# View today's costs by feature
wipnote costs --period today --by feature

# View all costs grouped by tool
wipnote costs --by tool --period all

# Export as CSV for spreadsheet analysis
wipnote costs --format csv --by tool > costs.csv

# Use Sonnet pricing instead of Opus
wipnote costs --model sonnet
```

## Features

- **Multiple Time Periods**: today, day, week (default), month, all
- **Multiple Groupings**: session (default), feature, tool, agent
- **Multiple Output Formats**: terminal (default), csv
- **Model Selection**: opus, sonnet, haiku, auto-detect
- **Smart Insights**: Cost optimization recommendations
- **Configurable Limits**: Show top N rows (default 10)

## Command Syntax

```
wipnote costs [OPTIONS]

Options:
  --period {today|day|week|month|all}
                                  Time period to analyze (default: week)
  --by {session|feature|tool|agent}
                                  Group costs by (default: session)
  --format {terminal|csv}         Output format (default: terminal)
  --model {opus|sonnet|haiku|auto}
                                  Claude model for pricing (default: auto)
  --limit N                       Maximum rows to display (default: 10)
  -g, --graph-dir PATH            Graph directory (default: .wipnote)
```

## Examples

### View Weekly Session Costs (Terminal)

```bash
$ wipnote costs --period week

LAST 7 DAYS - COST SUMMARY
═══════════════════════════════════════════════════════════════════

Name                 Events       Tokens        Estimated Cost
────────────────────────────────────────────────────────────────────
sess-38b6aa28          47        2,100,000      $31.50
sess-9d982022          34          890,000      $13.35
sess-a30b6ebc          28          620,000       $9.30
────────────────────────────────────────────────────────────────────

Total Tokens: 3,610,000 (4.7h)
Estimated Cost: $54.15 (Opus)

Most expensive: sess-38b6aa28 (58% of total)

Insights & Recommendations
───────────────────────────────────────────────────────────────────

→ Highest cost: sess-38b6aa28 (58% of total)
→ Cost concentration: Top 3 account for 100%
→ Tip: Many sessions with costs. Consider consolidating work to fewer, focused sessions.
```

### View Costs by Feature

```bash
$ wipnote costs --by feature --period week

LAST 7 DAYS - COST SUMMARY
═══════════════════════════════════════════════════════════════════

Name                 Events       Tokens        Estimated Cost
────────────────────────────────────────────────────────────────────
auth-system             12       1,800,000      $27.00
pagination              5          850,000      $12.75
error-handling          3          950,000      $14.25
────────────────────────────────────────────────────────────────────

Total Tokens: 3,600,000 (4.5h)
Estimated Cost: $54.00 (Opus)

Most expensive: auth-system (50% of total)

Insights & Recommendations
───────────────────────────────────────────────────────────────────

→ Highest cost: auth-system (50% of total)
→ Cost concentration: Top 3 account for 100%
```

### View Costs by Tool

```bash
$ wipnote costs --by tool --period today

TODAY - COST SUMMARY
═══════════════════════════════════════════════════════════════════

Name                 Events       Tokens        Estimated Cost
────────────────────────────────────────────────────────────────────
Bash                     15        600,000       $9.00
Read                      8        400,000       $6.00
Grep                      5        250,000       $3.75
Edit                      3        150,000       $2.25
────────────────────────────────────────────────────────────────────

Total Tokens: 1,400,000 (2.7h)
Estimated Cost: $21.00 (Opus)

Most expensive: Bash (43% of total)

Insights & Recommendations
───────────────────────────────────────────────────────────────────

→ Highest cost: Bash (43% of total)
→ Cost concentration: Top 3 account for 91%
→ Tip: Bash is expensive. Consider batching operations or using more efficient approaches.
```

### Export as CSV

```bash
$ wipnote costs --by tool --format csv

Tool,Events,Tokens,Estimated Cost (USD)
Bash,15,600000,9.00
Read,8,400000,6.00
Grep,5,250000,3.75
Edit,3,150000,2.25
TOTAL,,1400000,21.00
```

### Use Different Pricing Models

```bash
# Default (Opus pricing - highest cost)
wipnote costs --model opus

# Sonnet pricing (mid-tier)
wipnote costs --model sonnet

# Haiku pricing (cheapest)
wipnote costs --model haiku

# Auto-detect (uses Opus for conservative estimates)
wipnote costs --model auto
```

## Cost Calculation

The command calculates costs using Claude's official pricing:

| Model | Input | Output |
|-------|-------|--------|
| Opus | $15/1M | $45/1M |
| Sonnet | $3/1M | $15/1M |
| Haiku | $0.80/1M | $4/1M |

**Assumptions:**
- 90% input tokens, 10% output tokens (typical ratio)
- Costs are estimates based on `cost_tokens` field in database
- Actual costs may vary based on your Claude pricing tier

## Database Schema

Costs are queried from `agent_events` table:

```sql
SELECT
    session_id,
    COUNT(*) as event_count,
    SUM(cost_tokens) as total_tokens,
    MIN(timestamp) as start_time,
    MAX(timestamp) as end_time
FROM agent_events
WHERE event_type IN ('tool_call', 'tool_result')
  AND cost_tokens > 0
  AND timestamp >= ?
GROUP BY session_id
ORDER BY total_tokens DESC
LIMIT ?
```

## Performance Notes

- Large databases may take a few seconds to analyze
- Limits default to 10 for readability (use `--limit` to change)
- CSV output is efficient for large datasets
- Terminal output uses Rich for nice formatting

## Use Cases

### Cost Optimization
```bash
# Find most expensive features
wipnote costs --by feature --period month

# Identify expensive tools
wipnote costs --by tool --period week
```

### Budget Tracking
```bash
# Weekly cost review
wipnote costs --period week

# Monthly cost report
wipnote costs --period month --format csv > monthly-costs.csv
```

### Feature Planning
```bash
# Compare feature costs
wipnote costs --by feature --period all

# See which agents are most expensive
wipnote costs --by agent --period month
```

## Troubleshooting

### No cost data found
**Problem:** Command shows "No cost data found for the specified period"

**Solution:**
- Ensure you have `cost_tokens` populated in your database
- Try a longer time period with `--period all`
- Check that you're analyzing the right graph directory with `-g`

### Missing database
**Problem:** "No Wipnote database found"

**Solution:**
```bash
# Initialize Wipnote first
wipnote init

# Or specify the correct graph directory
wipnote costs -g /path/to/.wipnote
```

### Unexpected costs
**Problem:** Costs seem too high or too low

**Solution:**
- Check the model with `--model` (Opus is most expensive)
- Compare different models: `--model sonnet` or `--model haiku`
- View the raw tokens with both models: `--by session`

## Integration with Other Tools

### Pipe to other commands
```bash
# Find sessions over $20
wipnote costs --format csv | awk -F',' '$4 > 20'

# Sort by cost descending
wipnote costs --by feature --format csv | sort -t',' -k4 -rn
```

### Create custom reports
```bash
# Weekly email report
wipnote costs --period week --format csv | \
  mail -s "Weekly Wipnote Costs" team@example.com
```

### Track over time
```bash
# Log costs daily
wipnote costs --period day >> costs-log.csv
```

## See Also

- `wipnote analytics` - Broader analytics dashboard
- `wipnote cigs cost-dashboard` - Interactive HTML dashboard
- `wipnote cigs roi-analysis` - ROI analysis of delegations

# htmlgraph costs - Command Examples

Quick reference for using the cost dashboard CLI command.

## Essential Commands

### Default (Last Week, Grouped by Session)
```bash
htmlgraph costs
```

Shows the last 7 days of costs grouped by session ID.

### By Feature
```bash
htmlgraph costs --by feature
```

Shows which features are most expensive.

### By Tool
```bash
htmlgraph costs --by tool --period week
```

Identify the most expensive tools (Read, Bash, Grep, etc).

### By Agent
```bash
htmlgraph costs --by agent --period month
```

Compare costs across different agents (Claude, Codex, Gemini).

## Time Periods

### Today
```bash
htmlgraph costs --period today
```

Last 24 hours of activity.

### This Week
```bash
htmlgraph costs --period week
```

Last 7 days (default).

### This Month
```bash
htmlgraph costs --period month
```

Last 30 days of activity.

### All Time
```bash
htmlgraph costs --period all
```

Complete cost history.

## Output Formats

### Terminal (Default - Rich Formatted)
```bash
htmlgraph costs
# or explicitly:
htmlgraph costs --format terminal
```

Color-coded table with summary stats and insights.

### CSV (Spreadsheet Ready)
```bash
htmlgraph costs --format csv
```

Export to CSV for Excel, Google Sheets, etc.

## Pricing Models

### Opus (Default - Conservative)
```bash
htmlgraph costs --model opus
```

Most expensive model. Best for worst-case estimates.

### Sonnet (Mid-Tier)
```bash
htmlgraph costs --model sonnet
```

Mid-range pricing. More realistic for most workloads.

### Haiku (Budget)
```bash
htmlgraph costs --model haiku
```

Cheapest model. Good for lower-bound estimates.

### Auto-Detect
```bash
htmlgraph costs --model auto
```

Automatically estimate (currently defaults to Opus).

## Result Limits

### Show Top 10 (Default)
```bash
htmlgraph costs
```

Shows top 10 cost drivers.

### Show Top 5
```bash
htmlgraph costs --limit 5
```

Focus on top 5.

### Show All
```bash
htmlgraph costs --limit 1000
```

No practical limit (shows all results).

## Combinations

### This Week by Feature, Top 5
```bash
htmlgraph costs --period week --by feature --limit 5
```

Most expensive features this week.

### Last Month by Tool, CSV Format
```bash
htmlgraph costs --period month --by tool --format csv
```

Export tool costs to CSV.

### All Time by Agent, Sonnet Pricing
```bash
htmlgraph costs --period all --by agent --model sonnet
```

Compare agent costs with mid-tier pricing.

### Today by Session, Budget Estimate
```bash
htmlgraph costs --period today --by session --model haiku
```

Today's costs with budget-friendly pricing.

## Real-World Workflows

### Budget Review (Weekly)
```bash
# Check weekly spend
htmlgraph costs --period week

# Export for reporting
htmlgraph costs --period week --format csv > weekly-report.csv

# Email to team
htmlgraph costs --period week --format csv | mail -s "Weekly Costs" team@example.com
```

### Feature Cost Analysis
```bash
# Find most expensive features
htmlgraph costs --by feature --period month

# Export for comparison
htmlgraph costs --by feature --format csv > features.csv

# Identify optimization targets
# Sort by cost and review top 3
```

### Tool Optimization
```bash
# Find most expensive tools
htmlgraph costs --by tool --period week

# Compare different pricing models
htmlgraph costs --by tool --model opus
htmlgraph costs --by tool --model sonnet

# Identify opportunities for batch operations
# (especially for Read, Bash, Grep)
```

### Agent Comparison
```bash
# Compare agents by cost
htmlgraph costs --by agent --period month

# Are specialized agents (Codex, Gemini) more expensive?
# Should we delegate more/less?

# Export for analysis
htmlgraph costs --by agent --format csv
```

### Trend Analysis
```bash
# Daily check
htmlgraph costs --period today > today.csv

# Weekly check
htmlgraph costs --period week > week.csv

# Monthly report
htmlgraph costs --period month --format csv > month.csv

# Compare over time
# Use spreadsheet to analyze trends
```

## Integration Examples

### Pipe to Other Commands
```bash
# Find sessions that cost more than $20
htmlgraph costs --format csv | awk -F',' '$NF > 20 {print}'

# Sort by cost (descending)
htmlgraph costs --format csv | sort -t',' -k4 -rn

# Get average cost
htmlgraph costs --format csv | awk -F',' '{sum+=$4} END {print "Avg:", sum/NR}'
```

### Create Alerts
```bash
# Alert if week's costs exceed threshold
COSTS=$(htmlgraph costs --period week --format csv | tail -1 | cut -d',' -f4)
if (( $(echo "$COSTS > 100" | bc -l) )); then
  echo "Weekly costs exceed $100: $COSTS" | mail -s "Cost Alert" admin@example.com
fi
```

### Automated Reporting
```bash
# Daily cost tracking
#!/bin/bash
DATE=$(date +%Y-%m-%d)
htmlgraph costs --period today --format csv > "costs-$DATE.csv"

# Weekly summary
if [ "$(date +%A)" = "Monday" ]; then
  htmlgraph costs --period week --format csv | mail -s "Weekly Costs" team@example.com
fi
```

### Dashboard Integration
```bash
# Get costs and post to monitoring system
htmlgraph costs --format csv > /tmp/costs.csv
curl -X POST -d @/tmp/costs.csv https://monitoring.example.com/costs
```

## Troubleshooting Commands

### Check if Database Exists
```bash
htmlgraph costs -g /path/to/.htmlgraph
```

### Verify Cost Data
```bash
# Try different periods to find data
htmlgraph costs --period today
htmlgraph costs --period week
htmlgraph costs --period all
```

### Compare Pricing Models
```bash
# See cost differences between models
echo "Opus:"; htmlgraph costs --model opus
echo "Sonnet:"; htmlgraph costs --model sonnet
echo "Haiku:"; htmlgraph costs --model haiku
```

### Export Everything for Analysis
```bash
# Export all groupings for comprehensive analysis
htmlgraph costs --by session --format csv > sessions.csv
htmlgraph costs --by feature --format csv > features.csv
htmlgraph costs --by tool --format csv > tools.csv
htmlgraph costs --by agent --format csv > agents.csv
```

## Performance Tips

### Use CSV for Large Datasets
```bash
# Terminal format is slower for 100+ rows
# Use CSV instead
htmlgraph costs --format csv > large-export.csv
```

### Limit Results for Quick Checks
```bash
# Use --limit for quick overview
htmlgraph costs --limit 5  # Quick top 5

# Use no limit for comprehensive analysis
htmlgraph costs --limit 1000  # Everything
```

### Filter Time Periods Effectively
```bash
# Today: Fastest (smallest dataset)
htmlgraph costs --period today

# Week: Default (good balance)
htmlgraph costs --period week

# All: Slowest (large datasets)
htmlgraph costs --period all
```

## Success Examples

### Example 1: Weekly Cost Review
```bash
$ htmlgraph costs --period week

LAST 7 DAYS - COST SUMMARY
═══════════════════════════════════════════════════════════════════

Name                 Events       Tokens        Estimated Cost
────────────────────────────────────────────────────────────────────
sess-38b6aa28          47        2,100,000      $31.50
sess-9d982022          34          890,000      $13.35
sess-a30b6ebc          28          620,000       $9.30

Total Tokens: 3,610,000 (4.7h)
Estimated Cost: $54.15 (Opus)

Most expensive: sess-38b6aa28 (58% of total)
```

**Action:** Session 38b6aa28 is 58% of costs. Review what was done there.

### Example 2: Tool Analysis
```bash
$ htmlgraph costs --by tool --period month

LAST 30 DAYS - COST SUMMARY
═══════════════════════════════════════════════════════════════════

Name                 Events       Tokens        Estimated Cost
────────────────────────────────────────────────────────────────────
Bash                     156       4,800,000      $72.00
Read                      89       2,200,000      $33.00
Grep                      67       1,400,000      $21.00

Total Tokens: 8,400,000 (14.2h)
Estimated Cost: $126.00 (Opus)

Most expensive: Bash (57% of total)
```

**Action:** Bash is 57% of costs. Can we batch operations? Use less shell?

### Example 3: Feature Planning
```bash
$ htmlgraph costs --by feature

LAST 7 DAYS - COST SUMMARY
═══════════════════════════════════════════════════════════════════

Name                 Events       Tokens        Estimated Cost
────────────────────────────────────────────────────────────────────
auth-system             45        1,800,000      $27.00
pagination              28          850,000      $12.75
error-handling          19          950,000      $14.25

Total Tokens: 3,600,000 (4.5h)
Estimated Cost: $54.00 (Opus)

Cost concentration: Top 3 account for 100%
```

**Action:** Auth system is most expensive feature. Worth optimizing?

## Quick Reference Cheat Sheet

```bash
# Default (week, by session)
htmlgraph costs

# By feature
htmlgraph costs --by feature

# By tool
htmlgraph costs --by tool

# By agent
htmlgraph costs --by agent

# Today only
htmlgraph costs --period today

# All time
htmlgraph costs --period all

# Export to CSV
htmlgraph costs --format csv

# Sonnet pricing
htmlgraph costs --model sonnet

# Top 5 only
htmlgraph costs --limit 5

# Combinations
htmlgraph costs --period today --by feature --format csv
htmlgraph costs --period month --by tool --model sonnet
htmlgraph costs --period all --by agent --limit 10
```

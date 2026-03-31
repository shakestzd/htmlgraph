# Multi-AI Orchestration - Real-World Examples

## Example 1: Feature Implementation Workflow

**Scenario:** Implement user authentication with OAuth

```bash
# 1. Create feature
htmlgraph feature create "Add user authentication"
htmlgraph feature start <feat-id>

# 2. Research phase - delegate to gemini-operator (fast, free)
Task(
    subagent_type="htmlgraph:gemini-operator",
    prompt="Research existing authentication patterns: What library is used? Where is validation? What OAuth providers exist?"
)

# 3. Implementation phase - delegate to codex-operator (code specialist)
Task(
    subagent_type="htmlgraph:codex-operator",
    prompt="Implement OAuth based on research: Add JWT auth to API endpoints, create token validation middleware, support Google and GitHub OAuth"
)

# 4. Testing phase - delegate to coder agent
Task(
    subagent_type="htmlgraph:sonnet-coder",
    prompt="Write comprehensive tests: unit tests for middleware, integration tests for OAuth flow, E2E tests for user login"
)

# 5. Git phase - delegate to copilot-operator
Task(
    subagent_type="htmlgraph:copilot-operator",
    prompt="Commit and push with message 'feat: add OAuth authentication'"
)

# 6. Mark feature complete
htmlgraph feature complete <feat-id>
```

## Example 2: Parallel Analysis Workflow

**Scenario:** Analyze 5 services for performance issues

```python
# Spawn parallel analysis with gemini-operator (free, fast)
# Dispatch all in a single message for true parallelism
Task(subagent_type="htmlgraph:gemini-operator", prompt="Analyze auth-service for performance: response times, N+1 queries, memory leaks")
Task(subagent_type="htmlgraph:gemini-operator", prompt="Analyze user-service for performance: response times, N+1 queries, memory leaks")
Task(subagent_type="htmlgraph:gemini-operator", prompt="Analyze order-service for performance: response times, N+1 queries, memory leaks")
Task(subagent_type="htmlgraph:gemini-operator", prompt="Analyze payment-service for performance: response times, N+1 queries, memory leaks")
Task(subagent_type="htmlgraph:gemini-operator", prompt="Analyze notification-service for performance: response times, N+1 queries, memory leaks")

# Save consolidated findings
htmlgraph spike create "Performance Analysis: All Services"
```

## Example 3: Architecture Design Workflow

**Scenario:** Design new notification system

```python
# 1. Architecture design - delegate to opus-level agent (deep reasoning)
Task(
    subagent_type="htmlgraph:opus-coder",
    prompt="""Design a scalable notification system:
Requirements: email/SMS/push support, 10M notifications/day, retry failed deliveries, track delivery status.
Provide: system architecture diagram (text), component breakdown, data flow, technology recommendations."""
)

# 2. Document design
# htmlgraph spike create "Notification System Architecture"

# 3. Implementation - delegate to codex-operator
Task(
    subagent_type="htmlgraph:codex-operator",
    prompt="""Implement the notification service based on the architecture above:
1. Create NotificationService class
2. Add email/SMS/push providers
3. Implement retry logic
4. Add status tracking"""
)
```

## Example 4: PR Review Workflow

**Scenario:** Review and merge a pull request

```python
# 1. Review with spawn_copilot (GitHub specialist)
review = spawn_copilot("""
Review PR #123:
- Check for security issues
- Verify test coverage
- Look for code style violations
- Identify potential bugs

Leave review comments on the PR.
""", allow_tools=["github", "read(*.py)"])

# 2. If approved, merge
spawn_copilot("""
If PR #123 passed review:
- Approve the PR
- Merge to main branch
- Delete the feature branch
""", allow_tools=["github", "shell(git)"])
```

## Example 5: Bug Investigation Workflow

**Scenario:** Debug session timeout issue

```bash
# 1. Create bug tracking
htmlgraph bug create "Session timeout too short"
```

```python
# 2. Investigation - delegate to gemini-operator (fast document search)
Task(
    subagent_type="htmlgraph:gemini-operator",
    prompt="Investigate session timeout issue: find session config files, search for timeout settings, check middleware, review logs"
)

# 3. Root cause analysis + fix - delegate to coder agent
Task(
    subagent_type="htmlgraph:sonnet-coder",
    prompt="""Fix session timeout issue:
Users report 5-min timeout, expected 30-min.
1. Update configuration
2. Add test to prevent regression
3. Verify fix works"""
)

# 4. Commit - delegate to copilot-operator
Task(
    subagent_type="htmlgraph:copilot-operator",
    prompt="Commit fix with message 'fix: correct session timeout to 30 minutes'"
)
```

```bash
# 5. Mark bug resolved
htmlgraph bug complete <bug-id>
```

## Example 6: Multi-Model Code Review

**Scenario:** Comprehensive code review using multiple AI models

```python
# Dispatch all reviews in parallel (single message = parallel execution)

# 1. Security review - delegate to opus-coder (deep analysis)
Task(
    subagent_type="htmlgraph:opus-coder",
    prompt="Security review of src/auth/: identify vulnerabilities, check input validation, review auth flow, assess data protection"
)

# 2. Performance review - delegate to gemini-operator (fast, free)
Task(
    subagent_type="htmlgraph:gemini-operator",
    prompt="Performance review of src/auth/: identify slow operations, check for N+1 queries, find unnecessary computations, suggest optimizations"
)

# 3. Code style review - delegate to codex-operator (code specialist)
Task(
    subagent_type="htmlgraph:codex-operator",
    prompt="Style review of src/auth/: check naming conventions, verify type hints, review documentation, assess test coverage"
)
```

```bash
# 4. Consolidate reviews in a spike
htmlgraph spike create "Comprehensive Auth Review"
```

## Cost Optimization Summary

| Workflow Type | Recommended Spawner | Why |
|---------------|---------------------|-----|
| Research/Analysis | spawn_gemini | Fast, cheap |
| Code Changes | spawn_codex | Specialized |
| Git Operations | spawn_copilot | GitHub integration |
| Architecture | spawn_claude | Deep reasoning |
| Parallel Work | spawn_gemini | Cost-effective at scale |

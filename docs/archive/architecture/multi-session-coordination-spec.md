# Multi-Session Coordination Spec

Status: Draft

## Purpose

Define how Wipnote coordinates work across:

- Multiple terminal-native AI sessions
- Multiple human-operated terminals
- Multiple subagents running in parallel inside a session
- Multiple worktrees and branches active at the same time

This specification formalizes the policy:

**One track-level orchestrator, many task-level executors, and one integration authority per branch/worktree.**

The goal is to scale parallel execution without turning coordination into a fragile chat-context problem.

## Scope

This spec covers:

- Coordination roles
- Ownership and write rules
- Claiming and leasing work
- Handoff and reassignment
- Branch/worktree integration authority
- Failure recovery for abandoned or stalled work

This spec does not define:

- UI details
- Exact database schema
- Exact CLI syntax
- Model-specific routing rules

## Design Principles

### 1. Coordination, not control

The orchestrator is a scheduling and integration role, not a global parent process.

### 2. Shared state over chat state

Durable work state must live in Wipnote artifacts and events, not only inside a session transcript.

### 3. Provenance is not authority

Parent-child session links are useful for lineage, attribution, and debugging. They do not by themselves grant write authority.

### 4. Ownership is about write scope

Concurrency is safe when write scopes are disjoint. Coordination failures usually come from ambiguous write ownership.

### 5. Integration must be serialized

Exploration, analysis, test execution, and implementation may run in parallel. Integration into a branch/worktree must be controlled by a single authority at a time.

## Normative Language

The key words "MUST", "MUST NOT", "SHOULD", and "MAY" are to be interpreted as described in RFC 2119.

## Core Entities

### Track

A track is the highest-level coordination unit for a multi-step initiative.

A track MAY contain:

- Features
- Bugs
- Spikes
- Sessions
- Claims
- Worktrees

A track MUST have at most one active track orchestrator at a time.

### Session

A session is any active execution context, including:

- A Claude Code session
- A Codex session
- A Gemini CLI session
- A human-operated terminal
- A spawned subagent

A session MUST have a stable session identifier.

A session MAY act in one or more roles:

- Track orchestrator
- Task executor
- Integration authority

### Claim

A claim is a time-bounded assignment of a work item or subtask to a session.

A claim MUST include:

- Claim ID
- Work item ID
- Owning session ID
- Owning agent or actor
- Status
- Lease expiry
- Intended output
- Declared write scope

### Write Scope

Write scope is the declared set of artifacts a claim is allowed to modify.

Examples:

- A list of files or directories
- A worktree path
- A branch name
- A work item record

Write scope MUST be explicit for implementation claims.

### Worktree

A worktree is an isolated filesystem and branch context for parallel execution.

A worktree MAY have multiple readers.

A worktree MUST have at most one active integration authority.

### Integration Authority

Integration authority is the session currently responsible for landing changes into a branch or worktree.

Integration authority MUST own:

- Merge decisions
- Conflict resolution
- Validation before landing
- Final acceptance or rejection of proposed changes

### Handoff

A handoff is a structured transfer of responsibility for a claim, worktree, or work item from one session to another.

A handoff MUST be durable and queryable after the original session exits.

## Roles

### Track Orchestrator

The track orchestrator owns track-level coordination.

Responsibilities:

- Prioritize work
- Decompose work into claims
- Assign or reassign claims
- Track dependencies and blockers
- Monitor progress across sessions
- Escalate stalled or conflicting work

The track orchestrator SHOULD avoid direct implementation except for trivial coordination tasks.

The track orchestrator MUST NOT be the default writer for every task in the track.

### Task Executor

A task executor owns a bounded claim.

Responsibilities:

- Execute the assigned task
- Stay within the declared write scope
- Emit progress and completion events
- Produce outputs matching the claim contract
- Request handoff or escalation when blocked

Task executors MAY be spawned subagents or independently opened terminal sessions.

### Integration Authority

The integration authority owns landing changes into a branch or worktree.

Responsibilities:

- Review proposed outputs from executors
- Merge or apply changes
- Resolve conflicts
- Run or delegate validation
- Mark work landed, deferred, or rejected

The integration authority MAY also be a task executor, but when acting as integration authority it MUST serialize landing operations.

## Invariants

The system MUST preserve these invariants:

1. A track has at most one active track orchestrator.
2. A branch or worktree has at most one active integration authority.
3. An implementation claim has exactly one current owner.
4. Two active implementation claims MUST NOT have overlapping write scopes unless explicitly marked as cooperative.
5. Parent-child session hierarchy MUST be preserved for lineage, but authority decisions MUST be based on claims and roles.
6. Stale claims MUST become reassignable after lease expiry or explicit abandonment.

## Coordination Model

### 1. Track-level planning

The track orchestrator breaks work into claims.

Each claim SHOULD be:

- Small enough to finish without further decomposition
- Large enough to avoid micromanagement overhead
- Specific about expected output
- Explicit about dependencies
- Explicit about write scope

### 2. Claim assignment

Claims are assigned to task executors based on:

- Capability match
- Cost
- Current availability
- Existing ownership of nearby write scopes
- Dependency readiness

### 3. Parallel execution

Parallel execution is encouraged for:

- Exploration
- Search
- Analysis
- Testing
- Documentation
- Independent implementation scopes

Parallel execution is discouraged for:

- Integration into the same branch/worktree
- Simultaneous writes to the same files
- Work that depends on another unfinished claim's output

### 4. Integration

Executors do not land changes directly into a shared branch unless they are also the current integration authority.

Instead, executors SHOULD return one of:

- A patch or commit
- A worktree branch ready for merge
- A structured change summary
- A failure or blocker report

The integration authority then decides whether to land the change.

## Claim Lifecycle

### Claim states

A claim MUST move through these states:

- `proposed`
- `claimed`
- `in_progress`
- `blocked`
- `handoff_pending`
- `completed`
- `abandoned`
- `expired`
- `rejected`

### State meanings

- `proposed`: defined but not yet owned
- `claimed`: ownership assigned, work not yet started
- `in_progress`: active execution underway
- `blocked`: owner cannot proceed without input or dependency
- `handoff_pending`: owner requests transfer
- `completed`: executor finished the claim output
- `abandoned`: owner explicitly relinquished the claim
- `expired`: lease ended without heartbeat or completion
- `rejected`: output was not accepted by integration authority or orchestrator

## Lease Model

Claims MUST use leases rather than indefinite ownership.

Each active claim MUST have:

- `leased_at`
- `lease_expires_at`
- `last_heartbeat_at`

Rules:

- A claim owner MUST periodically heartbeat while working.
- If heartbeat stops and the lease expires, the claim becomes reclaimable.
- Lease renewal SHOULD be lightweight.
- Lease expiry MUST NOT silently discard produced work.

Expired claims SHOULD retain:

- Partial progress notes
- Last known outputs
- File or branch references
- Blocker reason if known

## Write Scope Rules

### Read operations

Multiple sessions MAY read the same files, work items, or event logs concurrently.

### Write operations

For implementation work:

- A claim MUST declare write scope before editing starts.
- The system SHOULD reject or warn on overlapping active write scopes.
- Worktree isolation SHOULD be preferred when write scope is broad or uncertain.

### Cooperative writes

Cooperative writes are allowed only when:

- They are explicitly declared
- A coordinating integration authority exists
- The merge strategy is known in advance

Default behavior MUST assume overlapping writes are unsafe.

## Branch and Worktree Authority

Each branch or worktree MUST have one current integration authority.

The integration authority record SHOULD include:

- Branch name
- Worktree path
- Session ID
- Assigned at
- Intended merge target

Rules:

- Only the integration authority SHOULD merge into that branch/worktree.
- Executors MAY prepare commits or patches in isolated branches/worktrees.
- Integration authority MAY change, but the handoff MUST be explicit and recorded.
- If an authority becomes stale, the track orchestrator MAY appoint a replacement.

## Parent-Child Sessions

Parent-child session hierarchy remains important for:

- Provenance
- Cost attribution
- Prompt lineage
- Debugging
- Session analytics

However:

- A child session is not automatically authorized to merge.
- A parent session is not automatically entitled to reclaim a child's write scope without claim transfer.
- Authority MUST come from current claim ownership and integration assignment.

## Handoff Protocol

A handoff MUST capture enough information for a different session to continue without re-exploration.

Required handoff fields:

- Source session ID
- Target session ID, if known
- Claim ID or work item ID
- Current status
- Summary of completed work
- Remaining work
- Blockers
- Relevant files, branches, or worktrees
- Recommended next action

Handoffs SHOULD be emitted when:

- A session is ending
- A model switch is needed
- A human takes over
- A subagent exceeds its scope
- Integration authority changes

## Failure and Recovery

### Session crash

If a session disappears unexpectedly:

- Its claims MUST remain visible
- Its leases MUST eventually expire
- Its last known outputs MUST remain inspectable
- The track orchestrator MAY reassign its claims

### Stalled execution

A claim SHOULD be considered stalled when:

- Heartbeats stop
- No new events appear for a threshold window
- The session reports blocked state repeatedly

The track orchestrator MAY:

- Extend the lease
- Request status
- Reassign the claim
- Split the claim

### Integration authority loss

If the integration authority becomes unavailable:

- No other session should land new changes automatically
- The track orchestrator SHOULD appoint a new integration authority
- The reassignment MUST be recorded as an authority handoff

## Recommended Event Types

This spec recommends, but does not yet require, event types such as:

- `track.orchestrator.assigned`
- `claim.proposed`
- `claim.claimed`
- `claim.heartbeat`
- `claim.blocked`
- `claim.completed`
- `claim.expired`
- `claim.abandoned`
- `claim.handoff`
- `integration.authority.assigned`
- `integration.authority.released`
- `integration.authority.handoff`

## Suggested Data Shape

Illustrative claim record:

```json
{
  "claim_id": "clm-1234",
  "work_item_id": "feat-abc123",
  "track_id": "trk-xyz789",
  "owner_session_id": "sess-01",
  "owner_agent": "codex",
  "status": "in_progress",
  "intended_output": "Implement feature flag parser and tests",
  "write_scope": {
    "paths": [
      "src/python/wipnote/flags.py",
      "tests/python/test_flags.py"
    ],
    "branch": "worktree-agent-1234",
    "worktree": ".claude/worktrees/agent-1234"
  },
  "leased_at": "2026-03-30T14:00:00Z",
  "lease_expires_at": "2026-03-30T14:30:00Z",
  "last_heartbeat_at": "2026-03-30T14:12:00Z",
  "dependencies": [
    "feat-parser-spec"
  ]
}
```

Illustrative integration authority record:

```json
{
  "branch": "main",
  "worktree": "/repo",
  "integration_authority_session_id": "sess-ctrl-01",
  "integration_authority_agent": "claude-code",
  "assigned_at": "2026-03-30T14:05:00Z",
  "target": "origin/main"
}
```

## Operational Guidance

### When to use a worktree

Use isolated worktrees when:

- The write scope spans many files
- The boundaries are uncertain
- The task may run for a long time
- Multiple executors are active in parallel

### When to avoid parallel execution

Do not parallelize when:

- Two tasks need the same files
- The second task depends on the exact output of the first
- The integration cost is higher than the expected savings

### When to reassign a claim

Reassign when:

- The lease expired
- The executor explicitly handed off
- A different model is better suited for the remaining work
- Integration authority requests a narrower or corrected implementation

## Relationship to Existing Wipnote Concepts

This spec extends, rather than replaces:

- Session hierarchies for lineage
- Orchestrator mode for context preservation
- Tracks for initiative-level planning
- Features, bugs, and spikes as work items

This spec adds a stronger coordination layer centered on:

- Explicit claims
- Lease-based ownership
- Branch/worktree integration authority
- Durable handoff records

## Open Questions

The following remain implementation questions:

- Should claims be first-class HTML nodes, SQLite-backed records, or both?
- Should integration authority be attached to branches, worktrees, or both?
- What is the default lease duration?
- How strict should overlap detection be for write scopes?
- Which CLI commands should expose claim and authority state first?

## Adoption Path

Recommended rollout order:

1. Add claim records and claim lifecycle events.
2. Add lease expiry and heartbeat support.
3. Add integration authority assignment per branch/worktree.
4. Add overlap detection for write scopes.
5. Add handoff records and recovery tooling.
6. Add dashboard visibility for claims, authorities, and stalled work.


# Event Tracing Patterns - Observability Platforms Research

## Executive Summary

Industry observability platforms (Logfire, Langfuse, OpenTelemetry) use a **Pre-Hook + Post-Hook pattern with correlation IDs** to track tool call lifecycles. This enables accurate duration measurement, context propagation, and parent-child span relationships. Wipnote currently uses PostToolUse only, missing critical pre-execution context and precise timing data.

---

## 1. Logfire Approach

### Tool Call Capture
- **Integration**: Automatically captures tool invocations when instrumenting supported LLM libraries
- **Display**: Shows tool name, arguments, and returned payload (objects or arrays)
- **Events**: Follows **OpenTelemetry semantic conventions for GenAI spans**
- **Metadata**: Captures latency, input/output tokens, and total cost

### Key Implementation Details
- Built on OpenTelemetry standard
- Uses `gen_ai.system` and `gen_ai.request.model` attributes
- Captures token counts: `gen_ai.usage.prompt_tokens`, `gen_ai.usage.completion_tokens`
- Measures cost with input/output pricing breakdown

### What Wipnote Can Learn
- Semantic conventions provide standardized attribute naming
- Cost tracking requires both token counts and pricing metadata
- Token usage (prompt vs completion) is essential for cost analysis

**Source**: [Logfire LLM Panels Documentation](https://logfire.pydantic.dev/docs/guides/web-ui/llm-panels/)

---

## 2. Langfuse Approach

### Data Model Hierarchy
- **Traces**: Represent a single request/operation (user question → response)
- **Observations**: Individual steps within a trace
  - Specializations: generations, toolcalls, RAG retrieval steps, etc.
- **Sessions**: Optional grouping of related traces

### Span Lifecycle & Duration
- **Observations**: Can be nested hierarchically (parent-child relationships)
- **Core Fields**:
  - `id`: Unique observation ID
  - `traceId`: Links to parent trace
  - `parentObservationId`: Creates hierarchy
  - `startTime`: ISO timestamp when observation begins
  - `endTime`: ISO timestamp when observation completes
  - `type`: observation type (e.g., "toolcall")
  - `name`: Human-readable name
  - `level`: Log level (debug, info, warn, error)

### Duration Calculation
- Duration = `endTime - startTime`
- Automatic calculation when using context managers
- Manual `.end()` required for explicit lifecycle management

### Parent-Child Relationships
- Tree structure within single trace
- `parentObservationId` establishes parent link
- Enables nested tool calls and retrieval steps

**Source**: [Langfuse Data Model](https://langfuse.com/docs/observability/data-model), [Observation Types](https://langfuse.com/docs/observability/features/observation-types)

---

## 3. OpenTelemetry Standard

### Trace Model
- **Definition**: Directed acyclic graph (DAG) of Spans
- **Trace ID**: Shared across all spans in a trace (top-level identifier)
- **Span Relationships**: Parent-child via `parent_span_id`
- **Lifetime**: From root request through all nested operations

### Span Structure
Each Span contains:
- **Timestamps**: `start_time` and `end_time` (nanosecond precision)
- **Duration**: Calculated as `end_time - start_time`
- **Context**: Trace ID, Span ID, Parent Span ID
- **Attributes**: Key-value pairs describing operation
  - Keys: non-null strings
  - Values: string, boolean, float, int, or arrays of these
- **Events**: Timestamped annotations marking significant moments
  - Display as offsets from span start (easy duration tracking)
- **Status**: Success (Unset/Ok) or error conditions
- **Kind**: Client, Server, Internal, Producer, Consumer

### Best Practices
1. **Set attributes at span creation** - Samplers only see attributes present at creation
2. **Use events for timestamps** - Event timestamps display as offsets from span start
3. **Prefer duration representation** - Allows high-resolution timing in all languages
4. **Use semantic conventions** - Standardizes span names, attributes, kinds

**Sources**:
- [OpenTelemetry Traces](https://opentelemetry.io/docs/concepts/signals/traces/)
- [Trace Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/general/trace/)
- [OpenTelemetry Spans Explained](https://last9.io/blog/opentelemetry-spans-events/)

---

## 4. Pre-Hook + Post-Hook Pattern

### Why It's Universal
- **Separation of Concerns**: Start logic separate from completion logic
- **Accurate Timing**: Timestamps captured at actual execution boundaries
- **Correlation**: Tool use ID enables matching start → end events
- **Error Handling**: PostToolUse fires only on success; errors caught separately
- **Metadata Collection**: Pre-hook captures input; post-hook captures output and duration

### Claude Code Implementation
- **PreToolUse Hook**:
  - Fires BEFORE tool execution begins
  - Has access to: tool name, input, tool_use_id
  - Can block tool execution (permissions, validation)
  - **Action**: Record start timestamp, input arguments

- **PostToolUse Hook**:
  - Fires AFTER tool completes successfully
  - Has access to: tool name, input, output, tool_use_id
  - **Action**: Record end timestamp, output, duration = end - start

- **Correlation**: `tool_use_id` (str) correlates PreToolUse → PostToolUse events
  - Deterministic matching across hook invocations
  - Survives async and parallel execution

### Data Flow
```
PreToolUse(tool_use_id="abc-123", start=t1, tool="search", input={...})
  ↓
[Tool Execution - latency measured here]
  ↓
PostToolUse(tool_use_id="abc-123", end=t2, output={...})
  ↓
Duration = t2 - t1
```

**Sources**:
- [Claude Code Hooks Guide](https://code.claude.com/docs/en/hooks-guide)
- [Feature Request: tool_use_id Correlation](https://github.com/anthropics/claude-code/issues/13938)

---

## 5. Correlation & Context Propagation

### Trace Context Standard
- **W3C Traceparent Header Format**: `${version}-${trace-id}-${parent-id}-${trace-flags}`
- **Example**: `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01`
  - `00`: Version
  - `4bf92f3577b34da6a3ce929d0e0e4736`: Trace ID (128-bit)
  - `00f067aa0ba902b7`: Parent Span ID (64-bit)
  - `01`: Trace flags (sampled)

### Parent-Child Relationships
- **Synchronous**: Parent-child relationship (cleaner, unified view)
  - Parent span ID explicitly set in child
  - Used when you control the call stack
  - Creates clear visual hierarchy

- **Asynchronous**: Span links (more flexible)
  - Used in message-driven systems
  - When full context cannot be guaranteed
  - When service ownership is distributed

### For Tool Calls
- **Tool Use ID**: Acts as correlation ID (matches PreToolUse ↔ PostToolUse)
- **Session ID**: Could link all tool calls in a session
- **Trace ID**: Could track tool call chain across agents
- **Parent Observation ID**: Creates hierarchy for nested tool calls

**Sources**:
- [OpenTelemetry Context Propagation](https://opentelemetry.io/docs/concepts/context-propagation/)
- [Parent-Child vs Span Links (Datadog)](https://www.datadoghq.com/blog/parent-child-vs-span-links-tracing/)

---

## 6. Wipnote Current State

### What We Track
- **PostToolUse only**: Tool name, input, output, success/error
- **One-shot capture**: Single event per tool call
- **Available fields**: Captured from PostToolUse hook payload

### What We're Missing
1. **Start Timestamp**: When did execution actually begin?
2. **End Timestamp**: When did it complete?
3. **Duration**: How long did it take?
4. **Input Metadata**: Arguments passed to tool
5. **Output Metadata**: Return value structure and size
6. **Token Usage**: If applicable (not all tools)
7. **Cost**: Pricing impact
8. **Error Context**: Why did it fail (if applicable)?
9. **Parent-Child Relationships**: How do tool calls relate?
10. **Correlation IDs**: How do we match events together?

### Current Limitations
- Cannot calculate duration (no pre/post pair)
- Cannot track timing patterns (start vs completion)
- Cannot identify slow tool calls without external timing
- Cannot measure context propagation latency
- Single PostToolUse event doesn't show lifecycle
- No correlation mechanism for related tool calls

---

## 7. Best Practices from Observability Leaders

### Timing Accuracy
1. **Nanosecond Precision**: Use high-resolution timestamps (not milliseconds)
2. **Clock Skew Handling**: Duration = end - start (handles small drifts naturally)
3. **Event Timestamps as Offsets**: Display event times relative to span start
4. **Server-Side Calculation**: Duration calculated at post-hook (not client-sent)

### Minimum Required Fields for Tool Call Span
```
- tool_use_id: Correlation ID (string, unique)
- trace_id: Trace this belongs to (string)
- session_id: Session grouping (string)
- parent_observation_id: Optional parent span (string, nullable)
- start_time: PreToolUse timestamp (ISO8601)
- end_time: PostToolUse timestamp (ISO8601)
- duration_ms: Calculated (end - start) milliseconds
- tool_name: Which tool was called (string)
- tool_input: Arguments passed (JSON object)
- tool_output: Return value (JSON object)
- type: "toolcall" observation type (string)
- status: "Ok" or "Error" (enum)
- error_message: If status=Error (string, nullable)
```

---

## 8. Recommended Pattern for Wipnote

### Phase 1: Dual Hook Capture (PreToolUse + PostToolUse)

**PreToolUse Hook**:
```
Capture:
  - tool_use_id (correlation)
  - tool_name
  - tool_input
  - start_time (timestamp)
  - session_id
```

**PostToolUse Hook**:
```
Capture:
  - tool_use_id (for matching)
  - tool_name
  - tool_input
  - tool_output
  - end_time (timestamp)
  - status
  - error_message (if failed)
```

**Correlation**:
- Match events via tool_use_id
- Calculate duration = end_time - start_time
- Link to session and trace context

### Phase 2: Parent-Child Relationships
- Add parent_observation_id for nested calls
- Track call hierarchy

### Phase 3: Enhanced Metadata
- Token usage (if applicable)
- Cost calculation
- Performance metrics

### Phase 4: Querying & Analytics
- Duration histograms by tool
- Success rate tracking
- Cost analysis
- Nested call patterns

---

## 9. What to Capture in PreToolUse Hook

### Essential Data
```json
{
  "event_type": "PreToolUse",
  "tool_use_id": "abc-123",
  "tool_name": "search",
  "timestamp": "2025-01-07T10:30:00.000000Z",
  "input": {
    "query": "...",
    "num_results": 5
  },
  "context": {
    "trace_id": "xyz-789",
    "session_id": "sess-abc",
    "parent_tool_use_id": null
  }
}
```

### Why This Data
- **tool_use_id**: Correlation key (matches PostToolUse)
- **timestamp**: Exact start (no guessing)
- **input**: Arguments for analysis and replay
- **trace_id**: Links to full request context
- **session_id**: Groups related operations
- **parent_tool_use_id**: Enables hierarchy

---

## 10. Data Schema for Proper Tracing

### Event-Centric Schema (JSONL)
```
PreToolUse event:
{
  "event_id": "evt-pre-123",
  "event_type": "PreToolUse",
  "timestamp": "2025-01-07T10:30:00.000000Z",
  "tool_use_id": "abc-123",
  "tool_name": "search",
  "input": {...},
  "session_id": "sess-abc",
  "trace_id": "trace-xyz"
}

PostToolUse event:
{
  "event_id": "evt-post-456",
  "event_type": "PostToolUse",
  "timestamp": "2025-01-07T10:30:00.150000Z",
  "tool_use_id": "abc-123",
  "tool_name": "search",
  "input": {...},
  "output": {...},
  "status": "Ok",
  "duration_ms": 150,
  "session_id": "sess-abc",
  "trace_id": "trace-xyz"
}
```

### Observation-Centric Schema (SQL/NoSQL)
```
toolcalls table:
- tool_call_id (PK)
- tool_use_id (unique, indexed)
- trace_id (indexed)
- session_id (indexed)
- tool_name (indexed)
- start_time (indexed DESC)
- end_time
- duration_ms (calculated)
- input (JSON)
- output (JSON)
- status (indexed)
- error_message
- parent_tool_use_id
- created_at
```

### Index Strategy
1. **Single Tool Call Lookup**: `(tool_use_id)` UNIQUE
2. **Trace Reconstruction**: `(trace_id, start_time DESC)`
3. **Session Analysis**: `(session_id, start_time DESC)`
4. **Performance Queries**: `(tool_name, duration_ms DESC)`
5. **Error Analysis**: `(status, start_time DESC)` where status != "Ok"
6. **Time Range**: `(start_time DESC)` for chronological queries

---

## 11. Handling Async & Concurrent Operations

### Challenge
- Multiple tools can execute in parallel
- Pre/Post hooks may not maintain order
- Context can be lost in async operations

### Solutions

**1. Tool Use ID Correlation**
- Each tool execution gets unique `tool_use_id`
- Survives async/parallel execution
- Deterministic matching: PreToolUse + PostToolUse with same ID = same execution

**2. Session Context**
- All tool calls in session share `session_id`
- Enables grouping even if execution order changes
- Can query "all tools called in session X"

**3. Trace Context**
- `trace_id` links all operations in request
- Enables reconstruction of execution graph
- Shows which tools ran in parallel

**4. Explicit Ordering**
- Store `sequence_number` or `order` field
- Allow reconstruction of actual execution order
- Independent of hook invocation order

**5. Timestamp-Based Reconstruction**
```python
# Find concurrent tools
SELECT tool_use_id, tool_name, start_time, end_time
FROM tool_calls
WHERE trace_id = 'xyz'
  AND start_time < OTHER.end_time
  AND end_time > OTHER.start_time
```

---

## 12. Query Patterns for Analysis

### Performance Analysis
```sql
-- Slowest tools
SELECT tool_name, AVG(duration_ms) as avg_duration, MAX(duration_ms) as max_duration, COUNT(*) as count
FROM tool_calls
GROUP BY tool_name
ORDER BY avg_duration DESC;

-- Duration distribution
SELECT tool_name,
  PERCENTILE(duration_ms, 0.5) as p50,
  PERCENTILE(duration_ms, 0.95) as p95,
  PERCENTILE(duration_ms, 0.99) as p99
FROM tool_calls
GROUP BY tool_name;
```

### Reliability Analysis
```sql
-- Error rate by tool
SELECT tool_name,
  COUNT(CASE WHEN status = 'Error' THEN 1 END) as errors,
  COUNT(*) as total,
  COUNT(CASE WHEN status = 'Error' THEN 1 END) * 100.0 / COUNT(*) as error_rate
FROM tool_calls
GROUP BY tool_name
ORDER BY error_rate DESC;
```

### Nested Call Analysis
```sql
-- Tools that call other tools
SELECT parent.tool_name as parent, child.tool_name as child, COUNT(*) as count
FROM tool_calls parent
JOIN tool_calls child ON child.parent_tool_use_id = parent.tool_use_id
GROUP BY parent.tool_name, child.tool_name
ORDER BY count DESC;
```

---

## 13. Implementation Roadmap for Wipnote

### Immediate (Sprint 1)
- [ ] Add PreToolUse hook capture
- [ ] Store pre-event and post-event with correlation
- [ ] Implement tool_use_id matching
- [ ] Add start_time and end_time fields
- [ ] Calculate duration_ms = end_time - start_time

### Short-Term (Sprint 2-3)
- [ ] Add parent-child relationships (parent_observation_id)
- [ ] Implement session-level grouping
- [ ] Add error status and error_message fields
- [ ] Implement indexing strategy

### Medium-Term (Sprint 4-5)
- [ ] Add optional token usage fields
- [ ] Add cost calculation (if applicable)
- [ ] Implement query patterns for analysis
- [ ] Create UI for performance analysis

### Long-Term (Future)
- [ ] Trace visualization
- [ ] Waterfall/Gantt chart for parallel execution
- [ ] Cost dashboards
- [ ] Anomaly detection (slow tools, high error rates)

---

## Key Differences from Current Wipnote

| Aspect | Current (PostToolUse Only) | Recommended (Pre+Post) |
|--------|--------------------------|----------------------|
| **Duration Measurement** | Not possible | Precise (start-end) |
| **Input Capture** | Only in PostToolUse | Dedicated PreToolUse |
| **Timing Accuracy** | Approximate (post only) | Precise (pre + post) |
| **Correlation** | N/A | tool_use_id |
| **Parent-Child Links** | Not supported | Supported |
| **Error Visibility** | Post-event only | Pre-event + error details |
| **Async Support** | Limited | Full (via unique IDs) |
| **Cost Tracking** | Not implemented | Can calculate |

---

## Conclusion

Industry observability platforms universally use a **Pre-Hook + Post-Hook pattern** because it:
1. ✅ Enables accurate duration measurement
2. ✅ Captures both input and output
3. ✅ Supports correlation via IDs
4. ✅ Handles async/concurrent execution
5. ✅ Maintains hierarchical relationships
6. ✅ Provides complete lifecycle visibility

Wipnote should implement this pattern to provide:
- Accurate performance metrics
- Better cost analysis
- Nested call relationships
- Comprehensive observability
- Industry-standard tracing model

**Next Step**: Implement PreToolUse hook capture and duration calculation.

---

## Research Sources

- [Logfire LLM Panels Documentation](https://logfire.pydantic.dev/docs/guides/web-ui/llm-panels/)
- [Langfuse Data Model](https://langfuse.com/docs/observability/data-model)
- [Langfuse Observation Types](https://langfuse.com/docs/observability/features/observation-types)
- [OpenTelemetry Traces](https://opentelemetry.io/docs/concepts/signals/traces/)
- [OpenTelemetry Trace Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/general/trace/)
- [OpenTelemetry Spans Explained](https://last9.io/blog/opentelemetry-spans-events/)
- [Claude Code Hooks Guide](https://code.claude.com/docs/en/hooks-guide)
- [OpenTelemetry Context Propagation](https://opentelemetry.io/docs/concepts/context-propagation/)
- [Parent-Child vs Span Links (Datadog)](https://www.datadoghq.com/blog/parent-child-vs-span-links-tracing/)
- [W3C Trace Context Standard](https://www.w3.org/TR/trace-context/)

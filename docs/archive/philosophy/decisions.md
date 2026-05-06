# Design Decisions

Key design decisions and their rationale.

## HTML as Primary Format

**Decision:** Use HTML files as the primary data format, not JSON or a database.

**Rationale:**
- HTML provides structure (nodes), relationships (hyperlinks), and presentation (CSS) in one format
- Browsers are universal graph viewers - no special tools needed
- Human readable and machine parseable
- Git-friendly text format

**Trade-offs:**
- Slightly more verbose than JSON
- Requires HTML parsing (vs JSON.parse)
- Not a "standard" database format

**Why it's worth it:** The ability to view any node in a browser with full styling is invaluable for debugging and understanding. Version control works perfectly. No special tools required.

## CSS Selectors for Queries

**Decision:** Use CSS selectors instead of a custom query language (like Cypher or GraphQL).

**Rationale:**
- Everyone already knows CSS selectors
- Powerful enough for most queries
- Native browser support
- Libraries available in every language

**Trade-offs:**
- Less expressive than Cypher for complex graph patterns
- No built-in graph traversal syntax

**Why it's worth it:** Zero learning curve. `'[data-status="blocked"]'` is immediately understandable. For complex queries, use the Python/JS graph algorithms.

## Pydantic for Validation

**Decision:** Use Pydantic models for all data structures.

**Rationale:**
- Type safety and validation
- Automatic serialization
- Excellent error messages
- IDE autocomplete support

**Trade-offs:**
- Adds dependency (but only on Python side)
- Schema changes require code updates

**Why it's worth it:** Catch errors early with validation. Type hints make the SDK easier to use. Documentation comes from type annotations.

## justhtml for Parsing

**Decision:** Use justhtml library for HTML parsing in Python.

**Rationale:**
- Pure Python, no C dependencies
- Simple API
- Works in restrictive environments
- Small footprint

**Trade-offs:**
- Slower than lxml for large files
- Less features than BeautifulSoup

**Why it's worth it:** Zero-dependency install works everywhere. Performance is fine for typical graph sizes (thousands of nodes).

## SQLite Index

**Decision:** Use SQLite as the operational query and event store.

**Rationale:**
- Fast local queries without an external server
- Central to the dashboard, hooks, and session tracking
- Standard library support via `sqlite3`
- Must be kept in sync with HTML artifacts (HTML is canonical)

**Trade-offs:**
- Adds a derived layer that must stay consistent with HTML files
- SQLite is not truly optional in practice — the dashboard and hooks depend on it

**Why it's worth it:** Zero-infrastructure relational queries. The HTML files remain the source of truth; SQLite is a read cache that can always be rebuilt.

## TrackBuilder Fluent API

**Decision:** Provide a fluent builder pattern for track creation.

**Rationale:**
- Reads like English
- Self-documenting
- Guides users through required fields
- Chainable for conciseness

**Example:**
```python
track = sdk.tracks.builder() \
    .title("Project") \
    .with_spec(overview="...") \
    .with_plan_phases([...]) \
    .create()
```

**Trade-offs:**
- More code than dictionary-based API
- Another pattern to learn

**Why it's worth it:** Discoverability through method chaining. IDE autocomplete shows what's available. Errors caught at build time, not creation time.

## Auto-generated IDs

**Decision:** Auto-generate hash-based IDs instead of user-provided or timestamp-based IDs.

**Rationale:**
- Collision-resistant across multiple concurrent agents
- No user decision required
- Works across time zones without ambiguity
- Short enough to read and type

**Format:** `{prefix}-{hash8}` — e.g., `feat-a1b2c3d4`, `bug-12345678`, `trk-abcdef12`

The 8-character hash is derived from SHA256 of (title + microsecond timestamp + random entropy). See [ID Generation API](../api/ids.md) for details.

**Trade-offs:**
- Less human-friendly than `"user-auth"`
- Not sortable by creation time (unlike timestamp IDs)

**Why it's worth it:** Multiple agents can create nodes simultaneously without conflicts. Opaque IDs also discourage treating IDs as semantic, which keeps the data model clean.

## Session Management via Hooks

**Decision:** Use hooks for automatic session management, not manual SDK calls.

**Rationale:**
- Zero user effort
- Can't forget to start/end session
- Consistent across all agents
- Framework-agnostic

**Trade-offs:**
- Requires hook configuration
- Less explicit than manual calls
- Might capture unwanted activity

**Why it's worth it:** Agent developers don't think about sessions. Attribution happens automatically. Context preserved across conversations.

## Phoenix LiveView Dashboard

**Decision:** Build the live observability dashboard with Phoenix LiveView (Elixir), not vanilla HTML/CSS/JS or React/Vue.

**Rationale:**
- Real-time server-push updates without writing custom WebSocket code
- HTMX-style DOM morphing (via LiveView's diff protocol) without a JS build step
- Elixir/OTP concurrency model handles many simultaneous agent sessions cleanly
- HTML artifact layer remains framework-independent; the dashboard is an optional layer on top

**Trade-offs:**
- Adds an Elixir/Erlang runtime dependency for users who want the live dashboard
- Increases operational complexity compared to a static HTML file
- Smaller community than React/Vue

**Why it's worth it:** The core HTML artifact layer still works in any browser without the dashboard. Phoenix LiveView gives the live dashboard production-grade real-time capabilities that vanilla JS approaches struggle with at scale. Users who only need static viewing never need to install Elixir.

## Git as Version Control

**Decision:** Design for Git from the start, not as an afterthought.

**Rationale:**
- Developers already use Git
- Perfect for text files
- Branching and merging work naturally
- History and diffs are meaningful

**Trade-offs:**
- Large graphs might have slow diffs
- Merge conflicts possible (though readable)

**Why it's worth it:** Real version control, not change logs. Branches for experimentation. History shows evolution of work.

## Python SDK First

**Decision:** Build Python SDK first, JavaScript second.

**Rationale:**
- AI agents primarily use Python
- Rich ecosystem (Pydantic, etc.)
- Type hints for documentation
- Easier to validate designs

**Trade-offs:**
- JavaScript users wait longer
- Some duplication of logic

**Why it's worth it:** Focus on primary users (agents) first. Get it right in one language, then port. JavaScript can still use HTML directly.

## MIT License

**Decision:** Use MIT license, not GPL or proprietary.

**Rationale:**
- Maximum freedom for users
- Commercial use allowed
- Compatible with everything
- Simple and clear

**Trade-offs:**
- Can't prevent proprietary forks
- No patent protection

**Why it's worth it:** Maximize adoption. No licensing worries. Good for community.

## Immutable by Default

**Decision:** Pydantic models are immutable by default.

**Rationale:**
- Prevents accidental modifications
- Thread-safe reads
- Explicit about changes (must call `.save()`)

**Trade-offs:**
- Must create new instances for changes
- More verbose update code

**Why it's worth it:** Explicit is better than implicit. Prevents bugs from unexpected mutations. Clear when data is being persisted.

## Decisions We Didn't Make

### Why not MongoDB?

- Binary format (not human-readable)
- Requires server
- Complex deployment
- Not a graph database

### Why not GraphQL?

- Too complex for this use case
- Schema definition overhead
- Requires server
- CSS selectors are simpler

### Why not Markdown?

- Can't represent structured properties
- No native relationship types
- Needs front matter (not standard)
- HTML is more flexible

### Why not RDF/Semantic Web?

- Too complex
- Poor tooling
- Steep learning curve
- Overkill for most use cases

## Next Steps

- [Why HTML?](why-html.md) - Core philosophy
- [Comparisons](comparisons.md) - vs alternatives
- [Contributing](../contributing/index.md) - Help improve Wipnote

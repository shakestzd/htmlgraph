# Why HTML?

The HTML-first design philosophy — an origin story

## The Problem

Modern AI agent systems are drowning in complexity:

- **Neo4j/Memgraph**: Requires Docker, JVM, learning Cypher query language
- **Redis**: Adds caching and state management overhead
- **PostgreSQL**: Heavy relational database setup and maintenance
- **Custom Protocols**: Proprietary agent coordination systems
- **JSON/YAML**: Manual reference management, no native graph structure
- **Separate UIs**: Additional observability tools and dashboards

Each component adds:

- Installation friction
- Learning curve
- Runtime dependencies
- Maintenance burden
- Integration complexity

## The Insight

**The web is already a giant graph database.**

Every HTML document has:

- **Nodes**: HTML files
- **Edges**: Hyperlinks (`<a href>`)
- **Properties**: `data-*` attributes
- **Query language**: CSS selectors
- **Presentation**: Built-in rendering with CSS
- **Portability**: Works everywhere
- **Version control**: Git-friendly text format

## Core Principles

### 1. Standards Over Invention

Use existing web standards instead of creating new ones:

- **HTML** for structure and content
- **CSS** for styling and presentation
- **JavaScript** for interactivity
- **HTTP** for serving
- **CSS Selectors** for querying

These standards are:

- Well-documented
- Universally supported
- Battle-tested
- Familiar to everyone

### 2. Human-Readable First

Optimizing for human readability has unexpected benefits:

- **Debugging**: View source in any browser
- **Version control**: Meaningful git diffs
- **Onboarding**: No special tools to learn
- **Trust**: See exactly what's stored
- **Portability**: Works in any environment

### 3. Minimal Infrastructure

**The HTML files themselves** work with just:

- A file system
- A web browser

**The SDK** has 14 runtime Python dependencies, including:

- `pydantic` - Data validation and models
- `justhtml` - HTML parsing
- `rich` - Terminal output
- `jinja2` - HTML templating
- `networkx` - Graph algorithms
- `sqlite3` - Indexing (Python standard library)
- ...and others

**What the core does not need:**

- Docker containers
- External database servers (Neo4j, Redis, PostgreSQL)
- Build tools or compilation
- Cloud services or API keys
- Daemon processes

**Note:** The optional Phoenix LiveView dashboard adds an Elixir/Erlang runtime for live observability features, but the core SDK and HTML artifact layer work without it.

### 4. Offline First

Wipnote works completely offline:

- No network required
- No authentication
- No cloud sync
- No external services

Copy the `.wipnote/` directory anywhere and it just works.

### 5. Git Native

HTML is plain text, which means:

- **Diffs show real changes**: See exactly what changed
- **Merge conflicts are readable**: Resolve conflicts easily
- **History is meaningful**: Understand evolution over time
- **Branches work naturally**: Experiment safely

### 6. AI Agent Friendly

HTML is ideal for AI agents:

- **Structured but flexible**: Easy to parse and generate
- **Self-documenting**: Content and metadata together
- **Hyperlinks are native**: Relationships are first-class
- **CSS selectors**: Powerful query language agents already know

## Benefits

### For Developers

- **Fast setup**: `pip install wipnote`, done
- **No configuration**: Works out of the box
- **View in browser**: Open any file to see it styled
- **Standard tools**: Git, text editors, browsers

### For AI Agents

- **Simple API**: SDK or direct HTML manipulation
- **Context-efficient**: Lightweight node representation
- **Clear attribution**: Session tracking built-in
- **Deterministic**: TrackBuilder for repeatable workflows

### For Teams

- **No infrastructure**: No databases to maintain
- **Easy sharing**: Commit to git, done
- **Transparent**: Everyone can view the graph
- **Accessible**: No special permissions or access

### For Projects

- **Low overhead**: Files on disk, that's it
- **Scalable**: Millions of nodes possible
- **Portable**: Move projects easily
- **Archivable**: HTML will outlive most databases

## Trade-offs

### What You Gain

- Simplicity
- Portability
- Human readability
- Minimal infrastructure (no Docker, databases, or daemons)
- Git integration
- Universal compatibility

### What You Give Up

- Sub-millisecond queries (add SQLite index if needed)
- Complex graph algorithms (implement in Python/JS)
- Concurrent writes (use file locking or optimistic concurrency)
- Database GUI tools (use the HTML dashboard instead)

**The trade-off is worth it** for most use cases. When you need advanced graph features, add them incrementally.

## Philosophy in Practice

### Start Simple

```python
# Just create a feature
feature = sdk.features.create("Add login")

# It's an HTML file
# Open it in a browser
# That's it
```

### Add Complexity Only When Needed

```python
# Need better query performance?
# Add SQLite index (optional)
sdk.rebuild_index()

# Need complex graph analysis?
# Use the graph algorithms
path = sdk.graph.shortest_path(start, end)
```

### Trust Web Standards

Don't reinvent what browsers already do:

- **Styling**: Use CSS, not custom renderers
- **Queries**: Use CSS selectors, not custom query language
- **Storage**: Use HTML files, not custom formats
- **Serving**: Use HTTP, not custom protocols

## Comparisons

### vs Neo4j

| Feature | Neo4j | Wipnote |
|---------|-------|-----------|
| Setup | Docker + JVM + Cypher | `pip install` |
| Query | Learn Cypher | CSS selectors |
| View data | Neo4j Browser | Any web browser |
| Version control | Binary exports | Git diff |
| Portability | Requires runtime | Just files |

### vs JSON/YAML

| Feature | JSON/YAML | Wipnote |
|---------|-----------|-----------|
| Structure | Manual references | Native hyperlinks |
| Presentation | Needs separate UI | Built-in rendering |
| Querying | jq or custom | CSS selectors |
| Validation | JSON Schema | HTML + Pydantic |

### vs Notion/Roam

| Feature | Notion/Roam | Wipnote |
|---------|-------------|-----------|
| Ownership | Cloud-hosted | Your filesystem |
| API | Rate-limited | Direct file access |
| Offline | Limited | Full functionality |
| Version control | Not supported | Git native |
| Agent access | API tokens | Direct SDK |

## The Future

HTML isn't going anywhere. By building on web standards, Wipnote will work:

- In 10 years
- On any platform
- With any tools
- For any use case

Built on web standards. Designed for AI-assisted development.

## Next Steps

- [Comparisons](comparisons.md) - Detailed comparisons with alternatives
- [Design Decisions](decisions.md) - Why specific choices were made
- [Getting Started](../getting-started/installation.md) - Try it yourself

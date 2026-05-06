# Initial Prompt: Build Wipnote - "HTML is All You Need"

## Project Overview

You are building **Wipnote**, a lightweight graph database framework that uses HTML files as nodes, hyperlinks as edges, and CSS selectors as the query language. The goal is to provide AI agents with a simple, human-readable, version-control-friendly alternative to traditional graph databases like Neo4j.

**Tagline**: "HTML is All You Need"

**Core Value Proposition**: Why use Docker + Neo4j + Cypher when HTML + hyperlinks + CSS selectors can do the job with zero dependencies?

## Your Mission

Build a Python library and JavaScript library that allows:

1. **Creating graph nodes as HTML files** with semantic structure
2. **Linking nodes with hyperlinks** (edges in the graph)
3. **Querying with CSS selectors** instead of custom query languages
4. **Converting between HTML and Pydantic models** for AI agents
5. **Rendering dashboards** in vanilla JavaScript (no frameworks)
6. **Optional SQLite indexing** for performance at scale

## Context Documents

You have access to:
- `CLAUDE.md` - Complete technical specification and architecture
- This prompt - Your starting instructions

Read CLAUDE.md carefully before beginning implementation.

## Phase 1 Tasks (Start Here)

### 1. Project Setup
```bash
# Create directory structure
mkdir -p wipnote/{src/python/wipnote,src/js,examples,tests,docs,dashboard}

# Initialize Python package
cd wipnote
cat > pyproject.toml << EOF
[project]
name = "wipnote"
version = "0.1.0"
description = "HTML is All You Need - Graph database on web standards"
authors = [{name = "Shakes", email = "your@email.com"}]
readme = "README.md"
requires-python = ">=3.10"
dependencies = [
    "justhtml>=0.6.0",
    "pydantic>=2.0.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0.0",
    "pytest-cov>=4.0.0",
    "black>=23.0.0",
    "mypy>=1.0.0",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
EOF
```

### 2. Core Python Implementation

**Priority 1: HTML Parser (src/python/wipnote/parser.py)**
```python
"""
Create a wrapper around justhtml that:
1. Parses HTML files into clean Python objects
2. Extracts graph structure (nodes, edges)
3. Handles data-* attributes
4. Provides CSS selector interface
"""
```

**Priority 2: Pydantic Models (src/python/wipnote/models.py)**
```python
"""
Define Pydantic models for:
1. Node - Basic graph node with id, title, type, status, properties
2. Edge - Graph edge with from_node, to_node, relationship type
3. Graph - Collection of nodes and edges
4. Step - Implementation step (for agent coordination)

Each model should have:
- to_html() method
- to_context() method (lightweight for AI agents)
- from_html() classmethod
"""
```

**Priority 3: Graph Operations (src/python/wipnote/graph.py)**
```python
"""
Implement core graph operations:
1. add(node) - Create HTML file for node
2. update(node) - Update existing HTML file
3. query(css_selector) - Find nodes matching selector
4. get(node_id) - Get specific node
5. shortest_path(from_id, to_id) - BFS shortest path
6. transitive_deps(node_id) - Get all dependencies recursively
7. find_bottlenecks() - Find nodes blocking most others
"""
```

**Priority 4: Converters (src/python/wipnote/converter.py)**
```python
"""
Bidirectional conversion:
1. html_to_node(filepath) - Parse HTML into Pydantic Node
2. node_to_html(node, filepath) - Write Node as HTML
3. Preserve all semantic information
4. Handle edge cases (missing fields, malformed HTML)
"""
```

### 3. JavaScript Implementation

**Priority 1: Core Library (src/js/wipnote.js)**
```javascript
"""
Create vanilla JS library with:
1. Wipnote class
2. loadFrom(directory) - Load all HTML files
3. query(css_selector) - Query nodes
4. getNode(id) - Get specific node
5. findPath(from, to) - Shortest path algorithm
6. Graph visualization utilities
"""
```

**Priority 2: Dashboard (dashboard/index.html)**
```html
"""
Create demo dashboard with:
1. Stats overview (total nodes, completion rate, etc.)
2. Node list with filtering
3. Dependency graph visualization
4. Search functionality
5. Pure vanilla JS, no frameworks
6. Responsive CSS
"""
```

### 4. Examples

**Priority 1: Todo List (examples/todo-list/)**
```
Simple example showing:
- 3-5 task HTML files with dependencies
- Status tracking (todo, in-progress, done)
- Basic dashboard
- Python script to create/update tasks
```

**Priority 2: Agent Coordination (examples/agent-coordination/)**
```
Replicate Ijoka pattern:
- Feature HTML files with dependencies
- Session HTML files tracking agent work
- Dashboard with real-time updates
- Agent interface example
```

### 5. Tests

**Priority 1: Python Tests (tests/python/)**
```python
"""
Test coverage for:
1. HTML parsing with various edge cases
2. Pydantic model validation
3. Graph operations (add, query, traverse)
4. Bidirectional conversion (HTML ↔ Pydantic)
5. Agent interface
"""
```

## Implementation Guidelines

### 1. HTML Format Standards

Follow this structure for all HTML files:
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Node Title</title>
    <link rel="stylesheet" href="../styles.css">
</head>
<body>
    <article id="node-id" 
             data-type="feature"
             data-status="todo"
             data-priority="medium">
        
        <header>
            <h1>Node Title</h1>
        </header>
        
        <nav data-graph-edges>
            <section data-edge-type="blocks">
                <h3>Blocked By:</h3>
                <ul>
                    <li><a href="other-node.html">Other Node</a></li>
                </ul>
            </section>
        </nav>
        
        <section data-content>
            <p>Node content here...</p>
        </section>
    </article>
</body>
</html>
```

### 2. CSS Selector Patterns

Support these common queries:
```python
# By status
graph.query("[data-status='in-progress']")

# By priority
graph.query("[data-priority='high']")

# Combined
graph.query("[data-status='blocked'][data-priority='high']")

# By type
graph.query("article[data-type='feature']")

# Complex
graph.query("article[data-status='todo'] nav[data-edge-type='blocks'] a")
```

### 3. Agent Interface Principles

Agents should receive **lightweight context**, not full HTML:
```python
# BAD - Agent sees full HTML
context = open('feature-001.html').read()  # 500+ tokens

# GOOD - Agent sees summary
context = node.to_context()  # ~50 tokens
"""
# feature-001: User Authentication
Status: in-progress | Priority: high
Progress: 2/5 steps
⚠️  Blocked by: feature-005
Next: Implement OAuth flow
"""
```

### 4. Performance Considerations

- **Small graphs (<100 nodes)**: Pure HTML parsing is fine
- **Medium graphs (100-1000 nodes)**: Add JSON manifest for discovery
- **Large graphs (>1000 nodes)**: Add SQLite index for queries

Start with small graph support, add optimizations later.

### 5. Error Handling

Be robust to:
- Malformed HTML
- Missing required attributes
- Broken links
- Circular dependencies
- Concurrent writes

Use Pydantic validation to catch issues early.

## Development Workflow

1. **Read CLAUDE.md** - Understand full architecture
2. **Set up project structure** - Create directories, pyproject.toml
3. **Implement Python core** - parser.py, models.py, graph.py
4. **Write tests** - Ensure core functionality works
5. **Create simple example** - Todo list to validate design
6. **Implement JavaScript** - Vanilla JS library + dashboard
7. **Build agent coordination example** - Migrate Ijoka pattern
8. **Write documentation** - README, quickstart, cookbook
9. **Polish and optimize** - Performance, edge cases
10. **Prepare for launch** - Blog post, social media

## Key Principles

1. **Zero dependencies** (except justhtml and Pydantic)
2. **Web standards only** (HTML, CSS, JavaScript)
3. **No build step required**
4. **Human-readable at all times**
5. **Version control friendly**
6. **AI agent optimized**
7. **Progressive enhancement** (start simple, add features)

## Success Criteria

You'll know this is working when:
- [ ] Can create HTML node files programmatically
- [ ] Can query nodes with CSS selectors
- [ ] Can find shortest path between nodes
- [ ] Dashboard renders and shows stats
- [ ] Agent can get lightweight context
- [ ] Todo example works end-to-end
- [ ] Python tests pass with >90% coverage
- [ ] JavaScript loads and queries in browser
- [ ] Documentation is clear and complete

## Questions to Answer During Implementation

1. **File watching**: Should we include file system watching for live updates?
2. **Concurrency**: How to handle multiple agents writing simultaneously?
3. **Schema validation**: Should HTML structure be strictly enforced?
4. **Search**: When to recommend SQLite index vs pure CSS selectors?
5. **Rendering**: Should we include graph visualization library or leave to user?

Document your decisions in CLAUDE.md.

## Next Steps After Phase 1

Once core is working:
- Add SQLite optional indexer
- Create more examples (knowledge base, docs site)
- Write comparison blog post
- Build launch materials
- Submit to HN/Reddit
- Iterate based on feedback

## Your First Task

Start by reading CLAUDE.md thoroughly, then:

1. Create the project directory structure
2. Write pyproject.toml
3. Implement models.py with basic Node/Edge Pydantic models
4. Write a simple test to validate the design

Show me your implementation plan before writing code.

---

**Remember**: The goal is to prove that "HTML is All You Need" for AI agent infrastructure. Keep it simple, keep it standard, keep it readable.

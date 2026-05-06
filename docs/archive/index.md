# Wipnote

<div class="hero-subtitle" style="text-align: center; margin: 2rem 0 3rem; font-size: 1.5rem; font-weight: 300; letter-spacing: 0.02em;">
Local-first observability and coordination platform for AI-assisted development
</div>

<div style="text-align: center; margin: 2rem 0;">
<img src="assets/graph-hero.png" alt="Wipnote - AI-assisted development observability" style="width: 100%; max-width: 1200px; border-radius: 8px; box-shadow: 0 4px 20px rgba(205, 255, 0, 0.2);">
</div>

<div class="quick-start">

## Install Wipnote

```bash
pip install wipnote
```

## Start Tracking Your Work

```python
from wipnote import SDK

# Initialize SDK (auto-discovers .wipnote directory)
sdk = SDK(agent="claude")

# Create a feature with fluent API
feature = sdk.features.create("User Authentication") \
    .set_priority("high") \
    .add_steps([
        "Create login endpoint",
        "Add JWT middleware",
        "Write tests"
    ]) \
    .save()

# Query with filters
high_priority = sdk.features.where(status="todo", priority="high")

# Create tracks with specs and plans
track = sdk.tracks.builder() \
    .title("OAuth Integration") \
    .with_spec(overview="Add OAuth 2.0 support") \
    .with_plan_phases([
        ("Phase 1", ["Setup OAuth (2h)", "Add JWT (3h)"])
    ]) \
    .create()
```

</div>

<div class="feature-grid">

<div class="feature-card">
<span class="feature-icon">&#128196;</span>
<div class="feature-title">HTML Work Items</div>
<div class="feature-desc">
Features, bugs, spikes, and tracks stored as HTML files. Git-diffable, browser-readable, and human-inspectable without any tooling.
</div>
</div>

<div class="feature-card">
<span class="feature-icon">&#128065;</span>
<div class="feature-title">Live Dashboard</div>
<div class="feature-desc">
Phoenix LiveView dashboard with real-time event feed, session tracking, and agent activity monitoring — all local, no external services.
</div>
</div>

<div class="feature-card">
<span class="feature-icon">&#9889;</span>
<div class="feature-title">Local-First Core</div>
<div class="feature-desc">
HTML files and SQLite require no external servers. Git-diffable, browser-readable, and fully offline. The optional Phoenix dashboard adds real-time observability.
</div>
</div>

<div class="feature-card">
<span class="feature-icon">&#128226;</span>
<div class="feature-title">Multi-AI Coordination</div>
<div class="feature-desc">
Works with Claude Code, Gemini CLI, Codex, and Copilot. Event-driven hook system captures every agent action automatically.
</div>
</div>

<div class="feature-card">
<span class="feature-icon">&#128200;</span>
<div class="feature-title">Git Native</div>
<div class="feature-desc">
Text-based storage means perfect version control. Diffs show what changed. Merge conflicts are human-readable.
</div>
</div>

<div class="feature-card">
<span class="feature-icon">&#128640;</span>
<div class="feature-title">SDK for Programmatic Access</div>
<div class="feature-desc">
Fluent Python SDK with Pydantic validation. TrackBuilder for deterministic workflows. Full type safety throughout.
</div>
</div>

</div>

---

## Why Wipnote?

AI-assisted development creates an observability gap: multiple agents running across sessions, no unified view of what was done, why, or by whom.

- ✅ **Purpose-built for Claude Code**: Understands Claude Code concepts natively — sessions, hooks, features, spikes, agent attribution. Not a generic APM tool adapted for AI; built from the ground up for how developers actually work with AI coding agents.
- ✅ **Local-first**: HTML files and SQLite require no servers or cloud services to configure or maintain
- ✅ **Observable**: every AI agent action is tracked and browsable in the dashboard
- ✅ **Multi-AI**: works with Claude Code, Gemini CLI, Codex, Copilot — not locked to one tool
- ✅ **Human-readable**: HTML files you can open in any browser, inspect with DevTools, and diff in git
- ✅ **Git-native**: all work items are diffable, versionable, and mergeable

---

## Core Philosophy

!!! quote "Local-first observability"
    Track, coordinate, and observe AI-assisted development workflows. HTML files as canonical work items, SQLite for operational queries, and an optional Phoenix LiveView dashboard for real-time observability — local-first, no external infrastructure required for the core.

---

## Quick Comparisons

### vs External Tracking Tools

| Feature | External Tools | Wipnote |
|---------|---------------|-----------|
| Setup | Accounts, APIs, cloud config | `pip install wipnote` |
| Offline | ❌ Requires internet | ✅ Fully offline |
| Human readable | 🟡 Web UI only | ✅ Any browser or text editor |
| Version control | ❌ Not git-native | ✅ Git diff works perfectly |
| Multi-AI support | ❌ Usually one tool | ✅ Claude, Gemini, Codex, Copilot |

### vs JSON/YAML Files

| Feature | JSON | Wipnote |
|---------|------|-----------|
| Human readable | 🟡 Text editor | ✅ Browser with styling |
| Query | ❌ jq or custom | ✅ SQLite + CSS selectors |
| Live dashboard | ❌ Needs UI | ✅ Optional Phoenix LiveView |
| Agent hooks | ❌ Manual | ✅ Automatic event capture |

---

## Next Steps

<div class="feature-grid">

<div class="feature-card">
<div class="feature-title">📚 Get Started</div>
<div class="feature-desc">
<a href="getting-started/">Installation guide, first project, and core concepts →</a>
</div>
</div>

<div class="feature-card">
<div class="feature-title">🔌 SDK Reference</div>
<div class="feature-desc">
<a href="api/sdk/">Complete SDK documentation with examples →</a>
</div>
</div>

<div class="feature-card">
<div class="feature-title">📖 User Guide</div>
<div class="feature-desc">
<a href="guide/">Learn tracks, features, and session management →</a>
</div>
</div>

<div class="feature-card">
<div class="feature-title">⚡ Examples</div>
<div class="feature-desc">
<a href="examples/">Real-world use cases and code samples →</a>
</div>
</div>

</div>

---

<div style="text-align: center; margin: 4rem 0 2rem; font-size: 0.875rem; color: var(--hg-text-muted);">
<p>Built with web standards. Designed for AI-assisted development.</p>
<p style="color: var(--hg-accent); font-weight: 600; margin-top: 1rem;">Local-first. Observable. Multi-AI.</p>
</div>

<script>
// Animated Graph Visualization
(function() {
  const container = document.getElementById('graph-viz');
  if (!container) return;

  const width = container.offsetWidth;
  const height = container.offsetHeight || 400;

  // Create nodes
  const nodes = [];
  const nodeCount = 25;
  for (let i = 0; i < nodeCount; i++) {
    const node = document.createElement('div');
    node.className = 'graph-node';
    node.style.left = Math.random() * (width - 20) + 'px';
    node.style.top = Math.random() * (height - 20) + 'px';
    node.style.animationDelay = Math.random() * 2 + 's';
    container.appendChild(node);
    nodes.push({
      element: node,
      x: parseFloat(node.style.left),
      y: parseFloat(node.style.top)
    });
  }

  // Create edges between nearby nodes
  for (let i = 0; i < nodes.length; i++) {
    for (let j = i + 1; j < nodes.length; j++) {
      const dx = nodes[j].x - nodes[i].x;
      const dy = nodes[j].y - nodes[i].y;
      const distance = Math.sqrt(dx * dx + dy * dy);

      if (distance < 150 && Math.random() > 0.7) {
        const edge = document.createElement('div');
        edge.className = 'graph-edge';
        edge.style.left = nodes[i].x + 6 + 'px';
        edge.style.top = nodes[i].y + 6 + 'px';
        edge.style.width = distance + 'px';
        edge.style.transform = `rotate(${Math.atan2(dy, dx)}rad)`;
        edge.style.animationDelay = Math.random() * 3 + 's';
        container.appendChild(edge);
      }
    }
  }

  // Slowly animate nodes
  setInterval(() => {
    nodes.forEach((node, i) => {
      const x = parseFloat(node.element.style.left);
      const y = parseFloat(node.element.style.top);
      const newX = x + (Math.random() - 0.5) * 2;
      const newY = y + (Math.random() - 0.5) * 2;

      // Boundary check
      if (newX > 0 && newX < width - 20) {
        node.element.style.left = newX + 'px';
        node.x = newX;
      }
      if (newY > 0 && newY < height - 20) {
        node.element.style.top = newY + 'px';
        node.y = newY;
      }
    });
  }, 100);
})();
</script>

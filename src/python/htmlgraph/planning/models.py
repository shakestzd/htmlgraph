"""
Planning utilities for HtmlGraph (Conductor-style workflow).

This module provides Pydantic models for comprehensive project planning and tracking.
It implements a Conductor-style planning workflow where complex work is organized
into Tracks, each containing a Spec (requirements) and Plan (implementation strategy).

Available Classes:
    - Track: Top-level container for a work stream with spec and plan
    - Spec: Requirements document with priorities and acceptance criteria
    - Plan: Implementation plan with phases, tasks, and multiple views (list/kanban/timeline/graph)
    - Phase: Logical grouping of related tasks within a plan
    - Task: Individual work item with estimates, blocking, and assignment
    - Requirement: A requirement within a spec with verification status
    - AcceptanceCriterion: An acceptance criterion for validating a spec

Conductor Workflow:
    1. Create Track: Define the work stream and its scope
    2. Write Spec: Document requirements and acceptance criteria
    3. Build Plan: Break work into phases and tasks
    4. Execute: Work through tasks, track progress
    5. Validate: Verify against acceptance criteria

Key Features:
    - Multi-view rendering: List, Kanban, Timeline, and Graph views
    - Dependency tracking: Tasks can be blocked by other tasks or features
    - Progress tracking: Automatic completion percentage calculation
    - HTML output: Rich, styled HTML documents with dashboard design system
    - Linking: Tracks link to features and sessions for traceability

Usage:
    from htmlgraph.planning import Track, Spec, Plan, Phase, Task
    from htmlgraph.sdk import SDK

    sdk = SDK(agent="claude")

    # Create a track
    track = Track(
        id="track-001",
        title="User Authentication System",
        description="Complete auth system with OAuth",
        status="active"
    )

    # Build a spec with requirements
    spec = Spec(
        id="spec-001",
        title="Auth System Requirements",
        track_id="track-001",
        overview="OAuth-based authentication with JWT sessions",
        requirements=[
            Requirement(
                id="req-001",
                description="Support Google and GitHub OAuth",
                priority="must-have"
            )
        ]
    )

    # Build a plan with phases and tasks
    plan = Plan(
        id="plan-001",
        title="Auth Implementation Plan",
        track_id="track-001",
        phases=[
            Phase(
                id="1",
                name="Foundation",
                tasks=[
                    Task(
                        id="task-001",
                        description="Set up OAuth providers",
                        estimate_hours=4.0,
                        priority="high"
                    )
                ]
            )
        ]
    )

    # Generate HTML outputs
    spec_html = spec.to_html()
    plan_html = plan.to_html()
"""

from datetime import datetime
from typing import Any, Literal

from pydantic import BaseModel, Field

from htmlgraph.models import Step


class Requirement(BaseModel):
    """A requirement within a spec with verification status."""

    id: str
    description: str
    priority: Literal["must-have", "should-have", "nice-to-have"] = "must-have"
    verified: bool = False
    notes: str = ""
    related_tech: list[str] = Field(default_factory=list)  # Links to tech stack
    feature_ids: list[str] = Field(
        default_factory=list
    )  # Features satisfying this requirement

    def to_html(self) -> str:
        """Convert requirement to HTML article element."""
        verified_attr = f' data-verified="{str(self.verified).lower()}"'
        priority_attr = f' data-priority="{self.priority}"'

        status = "✅" if self.verified else "⏳"

        tech_links = ""
        if self.related_tech:
            tech_items = "".join(
                f'<li><a href="../../project/tech-stack.html#{tech}">{tech}</a></li>'
                for tech in self.related_tech
            )
            tech_links = f"""
                <nav data-related>
                    <h4>Related Tech:</h4>
                    <ul>{tech_items}</ul>
                </nav>"""

        notes_html = f"<p>{self.notes}</p>" if self.notes else ""

        return f'''
        <article class="requirement"{priority_attr}{verified_attr} id="{self.id}">
            <h3>{status} {self.description}</h3>
            {notes_html}
            {tech_links}
        </article>'''


class AcceptanceCriterion(BaseModel):
    """An acceptance criterion for a spec."""

    description: str
    completed: bool = False
    test_case: str | None = None
    feature_ids: list[str] = Field(
        default_factory=list
    )  # Features satisfying this criterion

    def to_html(self) -> str:
        """Convert criterion to HTML list item."""
        completed_attr = f' data-completed="{str(self.completed).lower()}"'
        status = "✅" if self.completed else "⏳"

        test_html = ""
        if self.test_case:
            test_html = f"<code>{self.test_case}</code>"

        return f"<li{completed_attr}>{status} {self.description} {test_html}</li>"


class Spec(BaseModel):
    """
    A specification document for a track.

    Contains:
    - Overview and context
    - Requirements with priorities
    - Acceptance criteria
    - Links to related documents
    """

    id: str
    title: str
    track_id: str  # Parent track
    status: Literal["draft", "review", "approved", "outdated"] = "draft"
    author: str = "claude-code"
    created: datetime = Field(default_factory=datetime.now)
    updated: datetime = Field(default_factory=datetime.now)

    overview: str = ""
    context: str = ""  # Why we're building this
    requirements: list[Requirement] = Field(default_factory=list)
    acceptance_criteria: list[AcceptanceCriterion] = Field(default_factory=list)

    # Links
    product_links: list[str] = Field(default_factory=list)  # Links to product docs
    tech_stack_links: list[str] = Field(default_factory=list)  # Links to tech choices

    def to_html(self, stylesheet_path: str = "../../.htmlgraph/styles.css") -> str:
        """Convert spec to full HTML document with dashboard styling."""

        # Build requirements HTML
        req_html = ""
        if self.requirements:
            req_items = "\n".join(req.to_html() for req in self.requirements)
            req_html = f"""
        <section data-section="requirements">
            <h2>Requirements</h2>
            <div class="requirements-list">
                {req_items}
            </div>
        </section>"""

        # Build acceptance criteria HTML
        ac_html = ""
        if self.acceptance_criteria:
            ac_items = "\n                ".join(
                ac.to_html() for ac in self.acceptance_criteria
            )
            ac_html = f"""
        <section data-section="acceptance-criteria">
            <h2>Acceptance Criteria</h2>
            <ol class="criteria-list">
                {ac_items}
            </ol>
        </section>"""

        # Build links HTML with back navigation
        nav_html = """
        <div class="spec-nav">
            <a href="index.html" class="nav-link">← Track</a>
            <a href="plan.html" class="nav-link">Plan →</a>
        </div>"""

        overview_html = (
            f"<p>{self.overview}</p>"
            if self.overview
            else '<p class="muted">No overview provided</p>'
        )
        context_html = (
            f"<p>{self.context}</p>"
            if self.context
            else '<p class="muted">No context provided</p>'
        )

        return f'''<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="htmlgraph-version" content="1.0">
    <title>Spec: {self.title}</title>
    <link rel="stylesheet" href="{stylesheet_path}">
    <style>
        /* HtmlGraph Dashboard Design System */
        :root {{
            --bg-primary: #151518;
            --bg-secondary: #1C1C20;
            --bg-tertiary: #252528;
            --text-primary: #E0DED8;
            --text-secondary: #A0A0A0;
            --text-muted: #707070;
            --border: #333338;
            --border-strong: #606068;
            --accent: #CDFF00;
            --accent-text: #0A0A0A;
        }}

        * {{ box-sizing: border-box; }}

        body {{
            background: var(--bg-primary);
            color: var(--text-primary);
            font-family: 'JetBrains Mono', 'SF Mono', Monaco, 'Cascadia Code', monospace;
            font-size: 14px;
            line-height: 1.6;
            margin: 0;
            padding: 2rem;
            max-width: 1200px;
            margin-inline: auto;
        }}

        .spec-nav {{
            display: flex;
            gap: 1rem;
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 2px solid var(--border-strong);
        }}

        .nav-link {{
            color: var(--text-secondary);
            text-decoration: none;
            padding: 0.5rem 1rem;
            border: 2px solid var(--border-strong);
            border-radius: 0;
            transition: all 0.2s;
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.1em;
        }}

        .nav-link:hover {{
            color: var(--accent);
            border-color: var(--accent);
        }}

        article {{
            background: var(--bg-secondary);
            border: 2px solid var(--border-strong);
            padding: 0;
        }}

        header {{
            padding: 2rem;
            border-bottom: 2px solid var(--border-strong);
            background: var(--bg-tertiary);
        }}

        h1 {{
            margin: 0 0 1rem 0;
            font-size: 1.5rem;
            font-weight: 600;
            color: var(--accent);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }}

        h2 {{
            font-size: 0.875rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.1em;
            color: var(--text-secondary);
            margin: 0 0 1rem 0;
            border-bottom: 1px solid var(--border);
            padding-bottom: 0.5rem;
        }}

        .metadata {{
            display: flex;
            flex-wrap: wrap;
            gap: 0.75rem;
            font-size: 0.75rem;
        }}

        .badge {{
            background: var(--bg-primary);
            color: var(--text-secondary);
            padding: 0.25rem 0.75rem;
            border: 1px solid var(--border);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            font-size: 0.7rem;
        }}

        .status-draft {{
            color: var(--text-muted);
            border-color: var(--text-muted);
        }}

        .status-review {{
            color: #f59e0b;
            border-color: #f59e0b;
        }}

        .status-approved {{
            color: var(--accent);
            border-color: var(--accent);
        }}

        section {{
            padding: 2rem;
            border-bottom: 1px solid var(--border);
        }}

        section:last-child {{
            border-bottom: none;
        }}

        p {{
            margin: 0;
            color: var(--text-primary);
            line-height: 1.8;
        }}

        .muted {{
            color: var(--text-muted);
            font-style: italic;
        }}

        .requirements-list article,
        .criteria-list li {{
            background: var(--bg-tertiary);
            border: 1px solid var(--border);
            padding: 1rem;
            margin-bottom: 0.75rem;
        }}

        .criteria-list {{
            list-style: none;
            counter-reset: criteria;
            padding: 0;
            margin: 0;
        }}

        .criteria-list li {{
            counter-increment: criteria;
            position: relative;
            padding-left: 3rem;
        }}

        .criteria-list li::before {{
            content: counter(criteria);
            position: absolute;
            left: 1rem;
            top: 1rem;
            background: var(--accent);
            color: var(--accent-text);
            width: 1.5rem;
            height: 1.5rem;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 700;
            font-size: 0.75rem;
        }}

        a {{
            color: var(--accent);
            text-decoration: none;
        }}

        a:hover {{
            text-decoration: underline;
        }}
    </style>
</head>
<body>
    {nav_html}
    <article id="{self.id}" data-type="spec" data-status="{self.status}" data-track="{self.track_id}">
        <header>
            <h1>{self.title}</h1>
            <div class="metadata">
                <span class="badge status-{self.status}">{self.status.title()}</span>
                <span class="badge">Author: {self.author}</span>
                <span class="badge">Created: {self.created.strftime("%Y-%m-%d")}</span>
            </div>
        </header>

        <section data-section="overview">
            <h2>Overview</h2>
            {overview_html}
        </section>

        <section data-section="context">
            <h2>Context</h2>
            {context_html}
        </section>
        {req_html}{ac_html}
    </article>
</body>
</html>
'''


class Task(BaseModel):
    """
    A task within a plan phase.

    More detailed than a Step - includes estimates, blocking, assignment.
    """

    id: str
    description: str
    completed: bool = False
    assigned: str | None = None
    priority: Literal["low", "medium", "high"] = "medium"
    estimate_hours: float | None = None

    # Relationships
    blocked_by: list[str] = Field(default_factory=list)  # Task IDs or feature IDs
    subtasks: list[Step] = Field(default_factory=list)
    feature_ids: list[str] = Field(
        default_factory=list
    )  # Features implementing this task

    # Tracking
    started_at: datetime | None = None
    completed_at: datetime | None = None

    def to_html(self) -> str:
        """Convert task to HTML article element."""
        completed_attr = f' data-completed="{str(self.completed).lower()}"'
        priority_attr = f' data-priority="{self.priority}"'
        assigned_attr = f' data-assigned="{self.assigned}"' if self.assigned else ""
        blocked_attr = (
            f' data-blocked-by="{",".join(self.blocked_by)}"' if self.blocked_by else ""
        )

        status_icon = "✅" if self.completed else ("⏳" if self.started_at else "○")

        # Build blocking links
        blocking_html = ""
        if self.blocked_by:
            block_items = "".join(
                f'<li><a href="../../features/{bid}.html">Depends on: {bid}</a></li>'
                if bid.startswith("feature-")
                else f"<li>Depends on: {bid}</li>"
                for bid in self.blocked_by
            )
            blocking_html = f"""
                <nav data-task-links>
                    <ul>{block_items}</ul>
                </nav>"""

        # Build subtasks
        subtasks_html = ""
        if self.subtasks:
            subtask_items = "\n                    ".join(
                st.to_html() for st in self.subtasks
            )
            subtasks_html = f"""
                <ul data-subtasks>
                    {subtask_items}
                </ul>"""

        estimate_html = ""
        if self.estimate_hours:
            estimate_html = f' <span class="estimate">({self.estimate_hours}h)</span>'

        return f'''
        <article data-task="{self.id}"{completed_attr}{priority_attr}{assigned_attr}{blocked_attr}>
            <h3>
                <input type="checkbox" {"checked" if self.completed else ""} disabled>
                {status_icon} {self.description}{estimate_html}
            </h3>
            {blocking_html}{subtasks_html}
        </article>'''


class Phase(BaseModel):
    """A logical grouping of tasks within a plan."""

    id: str
    name: str
    description: str = ""
    status: Literal["not-started", "in-progress", "completed"] = "not-started"
    tasks: list[Task] = Field(default_factory=list)

    @property
    def completion_percentage(self) -> int:
        """Calculate completion percentage from tasks."""
        if not self.tasks:
            return 100 if self.status == "completed" else 0
        completed = sum(1 for t in self.tasks if t.completed)
        return int((completed / len(self.tasks)) * 100)

    @property
    def task_summary(self) -> str:
        """Get task summary (e.g., '2/5 tasks')."""
        completed = sum(1 for t in self.tasks if t.completed)
        return f"{completed}/{len(self.tasks)} tasks"

    def to_html(self) -> str:
        """Convert phase to HTML section."""
        status_attr = f' data-status="{self.status}"'
        completion = self.completion_percentage

        # Progress indicator
        progress_dots = "●" * (completion // 33) + "○" * (3 - (completion // 33))

        # Build tasks
        task_items = "\n".join(task.to_html() for task in self.tasks)

        desc_html = f"<p>{self.description}</p>" if self.description else ""

        # Collapsible section
        expanded = "▼" if self.status == "in-progress" else "▶"

        return f'''
        <section data-phase="{self.id}"{status_attr}>
            <h2>{expanded} Phase {self.id}: {self.name} ({self.task_summary}) {progress_dots}</h2>
            {desc_html}
            {task_items}
        </section>'''


class Plan(BaseModel):
    """
    An implementation plan for a track.

    Contains:
    - Phases with tasks
    - Progress tracking
    - Multiple views (list, kanban, timeline, graph)
    """

    id: str
    title: str
    track_id: str  # Parent track
    status: Literal["draft", "active", "completed", "abandoned"] = "draft"
    created: datetime = Field(default_factory=datetime.now)
    updated: datetime = Field(default_factory=datetime.now)

    phases: list[Phase] = Field(default_factory=list)

    # Milestones
    milestones: dict[str, str] = Field(default_factory=dict)  # {date: description}

    @property
    def total_tasks(self) -> int:
        """Total number of tasks across all phases."""
        return sum(len(phase.tasks) for phase in self.phases)

    @property
    def completed_tasks(self) -> int:
        """Number of completed tasks."""
        return sum(sum(1 for t in phase.tasks if t.completed) for phase in self.phases)

    @property
    def completion_percentage(self) -> int:
        """Overall completion percentage."""
        if self.total_tasks == 0:
            return 100 if self.status == "completed" else 0
        return int((self.completed_tasks / self.total_tasks) * 100)

    def to_html(self, stylesheet_path: str = "../../.htmlgraph/styles.css") -> str:
        """Convert plan to full HTML document with dashboard styling and multiple views."""

        # Build phases HTML
        phases_html = "\n".join(phase.to_html() for phase in self.phases)

        # Progress bar with dashboard styling
        completion = self.completion_percentage
        progress_html = f"""
        <div class="progress-container">
            <div class="progress-info">
                <span class="progress-label">{completion}% Complete</span>
                <span class="progress-count">({self.completed_tasks}/{self.total_tasks} tasks)</span>
            </div>
            <div class="progress-bar">
                <div class="progress-fill" style="width: {completion}%"></div>
            </div>
        </div>"""

        # Navigation for different views - matching dashboard view buttons exactly
        view_nav = """
        <div class="view-toggle">
            <button onclick="showView('list')" class="view-btn active" data-view="list">List</button>
            <button onclick="showView('kanban')" class="view-btn" data-view="kanban">Kanban</button>
            <button onclick="showView('timeline')" class="view-btn" data-view="timeline">Timeline</button>
            <button onclick="showView('graph')" class="view-btn" data-view="graph">Graph</button>
        </div>"""

        # JavaScript for view switching - updated for dashboard-style buttons
        js_code = """
        <script>
        function showView(view) {
            // Hide all view containers
            document.querySelectorAll('.view-container').forEach(v => {
                v.style.display = v.dataset.view === view ? 'block' : 'none';
            });
            // Update button active states
            document.querySelectorAll('.view-btn').forEach(btn => {
                btn.classList.toggle('active', btn.dataset.view === view);
            });
            // Render view-specific content
            if (view === 'kanban') renderKanban();
            if (view === 'timeline') renderTimeline();
            if (view === 'graph') renderGraph();
        }

        function filterTasks(query) {
            document.querySelectorAll('[data-task]').forEach(task => {
                const text = task.textContent.toLowerCase();
                task.style.display = text.includes(query.toLowerCase()) ? 'block' : 'none';
            });
        }

        function filterByAgent(agent) {
            if (!agent) {
                document.querySelectorAll('[data-task]').forEach(t => t.style.display = 'block');
                return;
            }
            document.querySelectorAll('[data-task]').forEach(task => {
                task.style.display = task.dataset.assigned === agent ? 'block' : 'none';
            });
        }

        function renderKanban() {
            const todo = document.getElementById('kanban-todo');
            const progress = document.getElementById('kanban-progress');
            const done = document.getElementById('kanban-done');

            todo.innerHTML = '';
            progress.innerHTML = '';
            done.innerHTML = '';

            document.querySelectorAll('[data-task]').forEach(task => {
                const clone = task.cloneNode(true);
                if (task.dataset.completed === 'true') {
                    done.appendChild(clone);
                } else if (task.dataset.assigned) {
                    progress.appendChild(clone);
                } else {
                    todo.appendChild(clone);
                }
            });
        }

        function renderTimeline() {
            // TODO: Implement timeline view with milestones
            console.log('Timeline view');
        }

        function renderGraph() {
            // TODO: Implement graph visualization of dependencies
            console.log('Graph view');
        }
        </script>"""

        # Search and filter controls with dashboard styling
        controls_html = """
        <div class="controls">
            <input type="search" class="search-input" placeholder="Search tasks..." oninput="filterTasks(this.value)">
            <select class="agent-filter" onchange="filterByAgent(this.value)">
                <option value="">All Agents</option>
                <option value="claude">Claude</option>
                <option value="copilot">Copilot</option>
            </select>
        </div>"""

        # Navigation links to connect with existing features
        nav_links = """
        <div class="plan-nav">
            <a href="../../index.html" class="nav-link">← Dashboard</a>
            <a href="index.html" class="nav-link">Track</a>
            <a href="spec.html" class="nav-link">Spec</a>
        </div>"""

        # List view (default)
        list_view = f"""
        <div class="view-container" data-view="list">
            {controls_html}
            <div class="phases-container">
                {phases_html}
            </div>
        </div>"""

        # Kanban view - matching dashboard kanban styling
        kanban_view = """
        <div class="view-container kanban-view" data-view="kanban" style="display: none;">
            <div class="kanban-board">
                <div class="kanban-column" data-status="todo">
                    <h3 class="column-header">To Do</h3>
                    <div id="kanban-todo" class="column-content"></div>
                </div>
                <div class="kanban-column" data-status="in-progress">
                    <h3 class="column-header">In Progress</h3>
                    <div id="kanban-progress" class="column-content"></div>
                </div>
                <div class="kanban-column" data-status="done">
                    <h3 class="column-header">Done</h3>
                    <div id="kanban-done" class="column-content"></div>
                </div>
            </div>
        </div>"""

        # Timeline view placeholder
        timeline_view = """
        <div class="view-container" data-view="timeline" style="display: none;">
            <div class="timeline-placeholder">
                <p class="muted">Timeline view - Coming soon</p>
                <p class="muted">Will visualize tasks on a timeline with milestones</p>
            </div>
        </div>"""

        # Graph view placeholder
        graph_view = """
        <div class="view-container" data-view="graph" style="display: none;">
            <div class="graph-placeholder">
                <p class="muted">Dependency graph - Coming soon</p>
                <p class="muted">Will visualize task dependencies and blocking relationships</p>
            </div>
            <svg id="dependency-graph" width="100%" height="600"></svg>
        </div>"""

        return f'''<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="htmlgraph-version" content="1.0">
    <title>{self.title}</title>
    <link rel="stylesheet" href="{stylesheet_path}">
    <style>
        /* HtmlGraph Dashboard Design System - Plan View */
        :root {{
            --bg-primary: #151518;
            --bg-secondary: #1C1C20;
            --bg-tertiary: #252528;
            --text-primary: #E0DED8;
            --text-secondary: #A0A0A0;
            --text-muted: #707070;
            --border: #333338;
            --border-strong: #606068;
            --accent: #CDFF00;
            --accent-dim: #B8E600;
            --accent-text: #0A0A0A;
        }}

        * {{ box-sizing: border-box; }}

        body {{
            background: var(--bg-primary);
            color: var(--text-primary);
            font-family: 'JetBrains Mono', 'SF Mono', Monaco, 'Cascadia Code', monospace;
            font-size: 14px;
            line-height: 1.6;
            margin: 0;
            padding: 2rem;
            max-width: 1400px;
            margin-inline: auto;
        }}

        /* Navigation */
        .plan-nav {{
            display: flex;
            gap: 1rem;
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 2px solid var(--border-strong);
        }}

        .nav-link {{
            color: var(--text-secondary);
            text-decoration: none;
            padding: 0.5rem 1rem;
            border: 2px solid var(--border-strong);
            transition: all 0.2s;
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.1em;
        }}

        .nav-link:hover {{
            color: var(--accent);
            border-color: var(--accent);
        }}

        /* Article/Container */
        article {{
            background: var(--bg-secondary);
            border: 2px solid var(--border-strong);
            padding: 0;
        }}

        header {{
            padding: 2rem;
            border-bottom: 2px solid var(--border-strong);
            background: var(--bg-tertiary);
        }}

        h1 {{
            margin: 0 0 1rem 0;
            font-size: 1.5rem;
            font-weight: 600;
            color: var(--accent);
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }}

        .metadata {{
            display: flex;
            flex-wrap: wrap;
            gap: 0.75rem;
            font-size: 0.75rem;
        }}

        .badge {{
            background: var(--bg-primary);
            color: var(--text-secondary);
            padding: 0.25rem 0.75rem;
            border: 1px solid var(--border);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            font-size: 0.7rem;
        }}

        .status-active {{ color: var(--accent); border-color: var(--accent); }}
        .status-draft {{ color: var(--text-muted); }}
        .status-completed {{ color: #16a34a; border-color: #16a34a; }}

        /* Progress Bar */
        .progress-container {{
            padding: 2rem;
            border-bottom: 2px solid var(--border-strong);
            background: var(--bg-tertiary);
        }}

        .progress-info {{
            display: flex;
            justify-content: space-between;
            margin-bottom: 0.75rem;
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }}

        .progress-label {{
            color: var(--accent);
            font-weight: 600;
        }}

        .progress-count {{
            color: var(--text-secondary);
        }}

        .progress-bar {{
            width: 100%;
            height: 0.5rem;
            background: var(--bg-primary);
            border: 2px solid var(--border-strong);
            position: relative;
            overflow: hidden;
        }}

        .progress-fill {{
            height: 100%;
            background: var(--accent);
            transition: width 0.3s ease;
        }}

        /* View Toggle - Exact Dashboard Match */
        .view-toggle {{
            display: flex;
            gap: 0;
            margin: 2rem;
            border: 2px solid var(--border-strong);
            width: fit-content;
        }}

        .view-btn {{
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.75rem;
            font-weight: 500;
            text-transform: uppercase;
            letter-spacing: 0.1em;
            padding: 0.75rem 1.5rem;
            background: var(--bg-secondary);
            color: var(--text-secondary);
            border: none;
            cursor: pointer;
            transition: all 0.15s;
        }}

        .view-btn:not(:last-child) {{
            border-right: 2px solid var(--border-strong);
        }}

        .view-btn:hover {{
            color: var(--text-primary);
            background: var(--bg-tertiary);
        }}

        .view-btn.active {{
            background: var(--accent);
            color: var(--accent-text);
        }}

        /* Controls */
        .controls {{
            margin: 2rem;
            margin-bottom: 1rem;
            display: flex;
            gap: 1rem;
        }}

        .search-input,
        .agent-filter {{
            background: var(--bg-tertiary);
            color: var(--text-primary);
            border: 1px solid var(--border);
            padding: 0.5rem 1rem;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.75rem;
        }}

        .search-input {{
            flex: 1;
        }}

        .search-input::placeholder {{
            color: var(--text-muted);
        }}

        /* Phases */
        .phases-container {{
            padding: 0 2rem 2rem;
        }}

        details {{
            border: 1px solid var(--border);
            margin-bottom: 1rem;
            background: var(--bg-tertiary);
        }}

        summary {{
            padding: 1rem;
            cursor: pointer;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            font-size: 0.875rem;
            background: var(--bg-primary);
            border-bottom: 1px solid var(--border);
        }}

        summary:hover {{
            background: var(--bg-tertiary);
            color: var(--accent);
        }}

        /* Tasks */
        [data-task] {{
            padding: 1rem;
            border-bottom: 1px solid var(--border);
            display: flex;
            align-items: flex-start;
            gap: 1rem;
        }}

        [data-task]:last-child {{
            border-bottom: none;
        }}

        .estimate {{
            color: var(--text-muted);
            font-size: 0.75rem;
            margin-left: auto;
        }}

        /* Kanban Board */
        .kanban-board {{
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 1.5rem;
            padding: 2rem;
        }}

        .kanban-column {{
            background: var(--bg-tertiary);
            border: 2px solid var(--border-strong);
            min-height: 400px;
        }}

        .column-header {{
            margin: 0;
            padding: 1rem;
            background: var(--bg-primary);
            border-bottom: 2px solid var(--border-strong);
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.1em;
            font-weight: 600;
        }}

        .column-content {{
            padding: 1rem;
        }}

        /* Placeholders */
        .timeline-placeholder,
        .graph-placeholder {{
            padding: 4rem 2rem;
            text-align: center;
        }}

        .muted {{
            color: var(--text-muted);
            font-size: 0.875rem;
        }}

        a {{
            color: var(--accent);
            text-decoration: none;
        }}

        a:hover {{
            text-decoration: underline;
        }}
    </style>
</head>
<body>
    {nav_links}
    <article id="{self.id}" data-type="plan" data-status="{self.status}" data-track="{self.track_id}">
        <header>
            <h1>{self.title}</h1>
            <div class="metadata">
                <span class="badge status-{self.status}">{self.status.title()}</span>
                <span class="badge">Created: {self.created.strftime("%Y-%m-%d")}</span>
            </div>
        </header>

        {progress_html}
        {view_nav}
        {list_view}
        {kanban_view}
        {timeline_view}
        {graph_view}
    </article>
    {js_code}
</body>
</html>
'''


class Track(BaseModel):
    """
    A track represents a complete work stream with spec and plan.

    Tracks organize related work and provide structure for planning,
    implementation, and tracking progress.
    """

    id: str
    title: str
    type: str = "track"
    description: str = ""
    status: Literal["planned", "active", "completed", "abandoned"] = "planned"
    priority: Literal["low", "medium", "high", "critical"] = "medium"

    created: datetime = Field(default_factory=datetime.now)
    updated: datetime = Field(default_factory=datetime.now)

    properties: dict[str, Any] = Field(default_factory=dict)
    edges: dict[str, list] = Field(default_factory=dict)
    content: str = ""

    # Component files
    has_spec: bool = False
    has_plan: bool = False

    # Links to features and sessions
    features: list[str] = Field(default_factory=list)  # Feature IDs
    sessions: list[str] = Field(default_factory=list)  # Session IDs

    def to_context(self) -> str:
        """Generate lightweight context for AI agents."""
        lines = [f"# Track: {self.title}"]
        lines.append(f"Status: {self.status} | Priority: {self.priority}")

        if self.has_spec:
            lines.append("Spec: Available")
        if self.has_plan:
            lines.append("Plan: Available")

        if self.features:
            lines.append(f"Features: {len(self.features)}")

        return "\n".join(lines)

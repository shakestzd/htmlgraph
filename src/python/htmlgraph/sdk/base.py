from __future__ import annotations

"""
Base SDK Core Class - Initialization and Core Properties

Extracted from sdk.py to reduce file size and improve modularity.
Contains the core SDK initialization logic and essential properties.
"""


import os
from pathlib import Path
from typing import TYPE_CHECKING, Any, cast

if TYPE_CHECKING:
    from htmlgraph import SDK

from htmlgraph.agent_detection import detect_agent_name
from htmlgraph.agents import AgentInterface
from htmlgraph.analytics import Analytics, CrossSessionAnalytics, DependencyAnalytics
from htmlgraph.collections import (
    BaseCollection,
    BugCollection,
    ChoreCollection,
    EpicCollection,
    FeatureCollection,
    PhaseCollection,
    SpikeCollection,
    TaskDelegationCollection,
    TodoCollection,
)
from htmlgraph.collections.insight import InsightCollection
from htmlgraph.collections.metric import MetricCollection
from htmlgraph.collections.pattern import PatternCollection
from htmlgraph.collections.session import SessionCollection
from htmlgraph.context_analytics import ContextAnalytics
from htmlgraph.db.schema import HtmlGraphDB
from htmlgraph.graph import HtmlGraph
from htmlgraph.session_manager import SessionManager
from htmlgraph.session_warning import check_and_show_warning
from htmlgraph.system_prompts import SystemPromptManager
from htmlgraph.track_builder import TrackCollection


class BaseSDK:
    """
    Core SDK class with initialization logic and essential properties.

    This class handles:
    - SDK initialization and auto-discovery
    - Database and graph initialization
    - Collection setup and configuration
    - Lazy-loaded properties (orchestrator, system_prompts)
    - Core utility methods (_log_event, _ensure_session_exists)

    Subclasses add domain-specific methods (analytics, planning, orchestration).
    """

    def __init__(
        self,
        directory: Path | str | None = None,
        agent: str | None = None,
        parent_session: str | None = None,
        db_path: str | None = None,
    ):
        """
        Initialize SDK.

        Args:
            directory: Path to .htmlgraph directory (auto-discovered if not provided)
            agent: REQUIRED - Agent identifier for operations.
                Used to attribute work items (features, spikes, bugs, etc) to the agent.
                Examples: agent='explorer', agent='coder', agent='tester'
                Critical for: Work attribution, result retrieval, orchestrator tracking
                Falls back to: CLAUDE_AGENT_NAME env var, then detect_agent_name()
                Raises ValueError if not provided and cannot be detected
            parent_session: Parent session ID to log activities to (for nested contexts)
            db_path: Path to SQLite database file (optional, defaults to ~/.htmlgraph/htmlgraph.db)
        """
        if directory is None:
            directory = self._discover_htmlgraph()

        if agent is None:
            # Try environment variable fallback
            agent = os.getenv("CLAUDE_AGENT_NAME")

        if agent is None:
            # Try automatic detection
            detected = detect_agent_name()
            if detected and detected != "cli":
                # Only accept detected if it's not the default fallback
                agent = detected
            else:
                # No valid agent found - fail fast with helpful error message
                raise ValueError(
                    "Agent identifier is required for work attribution. "
                    "Pass agent='name' to SDK() initialization. "
                    "Examples: SDK(agent='explorer'), SDK(agent='coder'), SDK(agent='tester')\n"
                    "Alternatively, set CLAUDE_AGENT_NAME environment variable.\n"
                    "Critical for: Work attribution, result retrieval, orchestrator tracking"
                )

        self._directory = Path(directory)
        self._agent_id = agent
        self._parent_session = parent_session or os.getenv("HTMLGRAPH_PARENT_SESSION")

        # Initialize SQLite database (Phase 2)
        # Use db_path if explicitly provided; otherwise place alongside the discovered
        # .htmlgraph directory so subagents always use the project-local database.
        self._db = HtmlGraphDB(db_path or str(self._directory / "htmlgraph.db"))
        self._db.connect()
        self._db.create_tables()

        # Initialize underlying HtmlGraphs first (for backward compatibility and sharing)
        # These are shared with SessionManager to avoid double-loading features
        self._graph = HtmlGraph(self._directory / "features")
        self._bugs_graph = HtmlGraph(self._directory / "bugs")

        # Initialize SessionManager with shared graph instances to avoid double-loading
        self.session_manager = SessionManager(
            self._directory,
            features_graph=self._graph,
            bugs_graph=self._bugs_graph,
        )

        # Agent interface (for backward compatibility)
        self._agent_interface = AgentInterface(
            self._directory / "features", agent_id=agent
        )

        # Cast self to SDK for type checking - BaseSDK is only used via SDK subclass
        sdk_self = cast("SDK", self)

        # Collection interfaces - all work item types (all with builder support)
        self.features = FeatureCollection(sdk_self)
        self.bugs = BugCollection(sdk_self)
        self.chores = ChoreCollection(sdk_self)
        self.spikes = SpikeCollection(sdk_self)
        self.epics = EpicCollection(sdk_self)
        self.phases = PhaseCollection(sdk_self)

        # Non-work collections
        self.sessions: SessionCollection = SessionCollection(sdk_self)
        self.tracks: TrackCollection = TrackCollection(
            sdk_self
        )  # Use specialized collection with builder support
        self.agents: BaseCollection[Any] = BaseCollection(sdk_self, "agents", "agent")

        # Learning collections (Active Learning Persistence)
        self.patterns = PatternCollection(sdk_self)
        self.insights = InsightCollection(sdk_self)
        self.metrics = MetricCollection(sdk_self)

        # Todo collection (persistent task tracking)
        self.todos = TodoCollection(sdk_self)

        # Task delegation collection (observability for spawned agents)
        self.task_delegations = TaskDelegationCollection(sdk_self)

        # Create learning directories if needed
        (self._directory / "patterns").mkdir(exist_ok=True)
        (self._directory / "insights").mkdir(exist_ok=True)
        (self._directory / "metrics").mkdir(exist_ok=True)
        (self._directory / "todos").mkdir(exist_ok=True)
        (self._directory / "task-delegations").mkdir(exist_ok=True)

        # Initialize RefManager and set on all collections
        from htmlgraph.refs import RefManager

        self.refs = RefManager(self._directory)

        # Set ref manager on all work item collections
        self.features.set_ref_manager(self.refs)
        self.bugs.set_ref_manager(self.refs)
        self.chores.set_ref_manager(self.refs)
        self.spikes.set_ref_manager(self.refs)
        self.epics.set_ref_manager(self.refs)
        self.phases.set_ref_manager(self.refs)
        self.tracks.set_ref_manager(self.refs)
        self.todos.set_ref_manager(self.refs)

        # Analytics interface (Phase 2: Work Type Analytics)
        self.analytics = Analytics(sdk_self)

        # Dependency analytics interface (Advanced graph analytics)
        self.dep_analytics = DependencyAnalytics(self._graph)

        # Cross-session analytics interface (Git commit-based analytics)
        self.cross_session_analytics = CrossSessionAnalytics(sdk_self)

        # Context analytics interface (Context usage tracking)
        self.context = ContextAnalytics(sdk_self)

        # Pattern learning interface (Phase 2: Behavior Pattern Learning)
        from htmlgraph.analytics.pattern_learning import PatternLearner

        self.pattern_learning = PatternLearner(self._directory)

        # Lazy-loaded orchestrator for subagent management
        self._orchestrator: Any = None

        # System prompt manager (lazy-loaded)
        self._system_prompts: SystemPromptManager | None = None

        # Session warning system (workaround for Claude Code hook bug #10373)
        # Shows orchestrator instructions on first SDK usage per session
        self._session_warning = check_and_show_warning(
            self._directory,
            agent=self._agent_id,
            session_id=None,  # Will be set by session manager if available
        )

    @staticmethod
    def _discover_htmlgraph() -> Path:
        """
        Auto-discover .htmlgraph directory.

        Delegates to discover_htmlgraph_dir() which checks environment
        variables (CLAUDE_PROJECT_DIR, HTMLGRAPH_PROJECT_DIR) first,
        then walks up from cwd. This ensures subagents spawned via
        Task() use the parent project's .htmlgraph directory.
        """
        from htmlgraph.sdk.discovery import discover_htmlgraph_dir

        return discover_htmlgraph_dir()

    @property
    def agent(self) -> str | None:
        """Get current agent ID."""
        return self._agent_id

    @property
    def system_prompts(self) -> SystemPromptManager:
        """
        Access system prompt management.

        Provides methods to:
        - Get active prompt (project override OR plugin default)
        - Create/delete project-level overrides
        - Validate token counts
        - Get prompt statistics

        Lazy-loaded on first access.

        Returns:
            SystemPromptManager instance

        Example:
            >>> sdk = SDK(agent="claude")

            # Get active prompt
            >>> prompt = sdk.system_prompts.get_active()

            # Create project override
            >>> sdk.system_prompts.create("## Custom prompt\\n...")

            # Validate token count
            >>> result = sdk.system_prompts.validate()
            >>> print(result['message'])

            # Get statistics
            >>> stats = sdk.system_prompts.get_stats()
            >>> print(f"Source: {stats['source']}")
        """
        if self._system_prompts is None:
            self._system_prompts = SystemPromptManager(self._directory)
        return self._system_prompts

    def dismiss_session_warning(self) -> bool:
        """
        Dismiss the session warning after reading it.

        IMPORTANT: Call this as your FIRST action after seeing the orchestrator
        warning. This confirms you've read the instructions.

        Returns:
            True if warning was dismissed, False if already dismissed

        Example:
            sdk = SDK(agent="claude")
            # Warning shown automatically...

            # First action: dismiss to confirm you read it
            sdk.dismiss_session_warning()

            # Now proceed with orchestration
            sdk.spawn_coder(feature_id="feat-123", ...)
        """
        if self._session_warning:
            return self._session_warning.dismiss(
                agent=self._agent_id,
                session_id=None,
            )
        return False

    def get_warning_status(self) -> dict[str, Any]:
        """
        Get current session warning status.

        Returns:
            Dict with dismissed status, timestamp, and show count
        """
        if self._session_warning:
            return self._session_warning.get_status()
        return {"dismissed": True, "show_count": 0}

    def db(self) -> HtmlGraphDB:
        """
        Get the SQLite database instance.

        Returns:
            HtmlGraphDB instance for executing queries

        Example:
            >>> sdk = SDK(agent="claude")
            >>> db = sdk.db()
            >>> events = db.get_session_events("sess-123")
            >>> features = db.get_features_by_status("todo")
        """
        return self._db

    def query(self, sql: str, params: tuple = ()) -> list[dict[str, Any]]:
        """
        Execute a raw SQL query on the SQLite database.

        Args:
            sql: SQL query string
            params: Query parameters (for safe parameterized queries)

        Returns:
            List of result dictionaries

        Example:
            >>> sdk = SDK(agent="claude")
            >>> results = sdk.query(
            ...     "SELECT * FROM features WHERE status = ? AND priority = ?",
            ...     ("todo", "high")
            ... )
            >>> for row in results:
            ...     print(row["title"])
        """
        if not self._db.connection:
            self._db.connect()

        cursor = self._db.connection.cursor()  # type: ignore[union-attr]
        cursor.execute(sql, params)
        rows = cursor.fetchall()
        return [dict(row) for row in rows]

    def execute_query_builder(
        self, sql: str, params: tuple = ()
    ) -> list[dict[str, Any]]:
        """
        Execute a query using the Queries builder.

        Args:
            sql: SQL query from Queries builder
            params: Parameters from Queries builder

        Returns:
            List of result dictionaries

        Example:
            >>> sdk = SDK(agent="claude")
            >>> sql, params = Queries.get_features_by_status("todo", limit=5)
            >>> results = sdk.execute_query_builder(sql, params)
        """
        return self.query(sql, params)

    def _log_event(
        self,
        event_type: str,
        tool_name: str | None = None,
        input_summary: str | None = None,
        output_summary: str | None = None,
        context: dict[str, Any] | None = None,
        cost_tokens: int = 0,
    ) -> bool:
        """
        Log an event to the SQLite database with parent-child linking.

        Internal method used by collections to track operations.
        Automatically creates a session if one doesn't exist.
        Reads parent event ID from HTMLGRAPH_PARENT_ACTIVITY env var for hierarchical tracking.

        Args:
            event_type: Type of event (tool_call, completion, error, etc.)
            tool_name: Tool that was called
            input_summary: Summary of input
            output_summary: Summary of output
            context: Additional context metadata
            cost_tokens: Token cost estimate

        Returns:
            True if logged successfully, False otherwise

        Example (internal use):
            >>> sdk._log_event(
            ...     event_type="tool_call",
            ...     tool_name="Edit",
            ...     input_summary="Edit file.py",
            ...     cost_tokens=100
            ... )
        """
        from uuid import uuid4

        event_id = f"evt-{uuid4().hex[:12]}"
        session_id = self._parent_session or "cli-session"

        # Read parent event ID from environment variable for hierarchical linking
        parent_event_id = os.getenv("HTMLGRAPH_PARENT_ACTIVITY")

        # Ensure session exists before logging event
        try:
            self._ensure_session_exists(session_id, parent_event_id=parent_event_id)
        except Exception as e:
            import logging

            logging.debug(f"Failed to ensure session exists: {e}")
            # Continue anyway - session creation failure shouldn't block event logging

        return self._db.insert_event(
            event_id=event_id,
            agent_id=self._agent_id,
            event_type=event_type,
            session_id=session_id,
            tool_name=tool_name,
            input_summary=input_summary,
            output_summary=output_summary,
            context=context,
            parent_event_id=parent_event_id,
            cost_tokens=cost_tokens,
        )

    def _ensure_session_exists(
        self, session_id: str, parent_event_id: str | None = None
    ) -> None:
        """
        Create a session record if it doesn't exist.

        Args:
            session_id: Session ID to ensure exists
            parent_event_id: Event that spawned this session (optional)
        """
        if not self._db.connection:
            self._db.connect()

        cursor = self._db.connection.cursor()  # type: ignore[union-attr]
        cursor.execute(
            "SELECT COUNT(*) FROM sessions WHERE session_id = ?", (session_id,)
        )
        exists = cursor.fetchone()[0] > 0

        if not exists:
            # Create session record
            self._db.insert_session(
                session_id=session_id,
                agent_assigned=self._agent_id,
                is_subagent=self._parent_session is not None,
                parent_session_id=self._parent_session,
                parent_event_id=parent_event_id,
            )

    def reload(self) -> None:
        """Reload all data from disk."""
        self._graph.reload()
        self._agent_interface.reload()
        # SessionManager reloads implicitly on access via its converters/graphs

    def summary(self, max_items: int = 10) -> str:
        """
        Get project summary.

        Returns:
            Compact overview for AI agent orientation
        """
        return self._agent_interface.get_summary(max_items)

"""
Type stub for htmlgraph.sdk.core module.

Contains the SDK class type definitions.
"""

from pathlib import Path
from typing import Any

from htmlgraph.agents import AgentInterface
from htmlgraph.analytics import Analytics, CrossSessionAnalytics, DependencyAnalytics
from htmlgraph.collections import (
    BugCollection,
    FeatureCollection,
    SpikeCollection,
)
from htmlgraph.collections.session import SessionCollection
from htmlgraph.context_analytics import ContextAnalytics
from htmlgraph.db.schema import HtmlGraphDB
from htmlgraph.graph import HtmlGraph
from htmlgraph.models import Node
from htmlgraph.refs import RefManager
from htmlgraph.sdk.analytics import AnalyticsEngine
from htmlgraph.session_manager import SessionManager
from htmlgraph.system_prompts import SystemPromptManager
from htmlgraph.track_builder import TrackCollection
from htmlgraph.types import (
    ActiveWorkItem,
    AggregateResultsDict,
    BottleneckDict,
    ImpactAnalysisDict,
    ParallelPlanResult,
    RiskAssessmentDict,
    SessionStartInfo,
    SmartPlanResult,
    TrackCreationResult,
    WorkQueueItem,
    WorkRecommendation,
)

class SDK:
    """
    Main SDK interface for AI agents.

    Type stub to provide static type information for mypy.
    The actual implementation is in sdk/core.py.
    """

    # Core attributes
    _directory: Path
    _agent_id: str | None
    _parent_session: str | None
    _db: HtmlGraphDB
    _graph: HtmlGraph
    _bugs_graph: HtmlGraph
    _agent_interface: AgentInterface
    _orchestrator: Any
    _system_prompts: SystemPromptManager | None
    _analytics_engine: AnalyticsEngine

    # Collection interfaces
    features: FeatureCollection
    bugs: BugCollection
    spikes: SpikeCollection
    sessions: SessionCollection
    tracks: TrackCollection

    # Session manager
    session_manager: SessionManager

    # Refs manager
    refs: RefManager

    def __init__(
        self,
        directory: Path | str | None = None,
        agent: str | None = None,
        parent_session: str | None = None,
        db_path: str | None = None,
    ) -> None: ...
    @property
    def agent(self) -> str | None: ...
    @property
    def system_prompts(self) -> SystemPromptManager: ...
    @property
    def analytics(self) -> Analytics: ...
    @property
    def dep_analytics(self) -> DependencyAnalytics: ...
    @property
    def cross_session_analytics(self) -> CrossSessionAnalytics: ...
    @property
    def context(self) -> ContextAnalytics: ...
    @property
    def pattern_learning(self) -> Any: ...
    @property
    def orchestrator(self) -> Any: ...
    def dismiss_session_warning(self) -> bool: ...
    def get_warning_status(self) -> dict[str, Any]: ...
    def ref(self, short_ref: str) -> Node | None: ...
    def db(self) -> HtmlGraphDB: ...
    def query(self, sql: str, params: tuple[Any, ...] = ()) -> list[dict[str, Any]]: ...
    def execute_query_builder(
        self, sql: str, params: tuple[Any, ...] = ()
    ) -> list[dict[str, Any]]: ...
    def export_to_html(
        self,
        output_dir: str | None = None,
        include_features: bool = True,
        include_sessions: bool = True,
        include_events: bool = False,
    ) -> dict[str, int]: ...
    def _log_event(
        self,
        event_type: str,
        tool_name: str | None = None,
        input_summary: str | None = None,
        output_summary: str | None = None,
        context: dict[str, Any] | None = None,
        cost_tokens: int = 0,
    ) -> bool: ...
    def reload(self) -> None: ...
    def summary(self, max_items: int = 10) -> str: ...
    def my_work(self) -> dict[str, Any]: ...
    def next_task(
        self, priority: str | None = None, auto_claim: bool = True
    ) -> Node | None: ...
    def get_status(self) -> dict[str, Any]: ...
    def dedupe_sessions(
        self,
        max_events: int = 1,
        move_dir_name: str = "_orphans",
        dry_run: bool = False,
        stale_extra_active: bool = True,
    ) -> dict[str, int]: ...
    def track_activity(
        self,
        tool: str,
        summary: str,
        file_paths: list[str] | None = None,
        success: bool = True,
        feature_id: str | None = None,
        session_id: str | None = None,
        parent_activity_id: str | None = None,
        payload: dict[str, Any] | None = None,
    ) -> Any: ...
    def spawn_explorer(
        self,
        task: str,
        scope: str | None = None,
        patterns: list[str] | None = None,
        questions: list[str] | None = None,
    ) -> dict[str, Any]: ...
    def spawn_coder(
        self,
        feature_id: str,
        context: str | None = None,
        files_to_modify: list[str] | None = None,
        test_command: str | None = None,
    ) -> dict[str, Any]: ...
    def orchestrate(
        self,
        feature_id: str,
        exploration_scope: str | None = None,
        test_command: str | None = None,
    ) -> dict[str, Any]: ...
    def help(self, topic: str | None = None) -> str: ...

    # Session management (from SessionManagerMixin)
    def start_session(
        self,
        session_id: str | None = None,
        title: str | None = None,
        agent: str | None = None,
    ) -> Any: ...
    def end_session(
        self,
        session_id: str,
        handoff_notes: str | None = None,
        recommended_next: str | None = None,
        blockers: list[str] | None = None,
    ) -> Any: ...
    def _ensure_session_exists(
        self, session_id: str, parent_event_id: str | None = None
    ) -> None: ...

    # Session handoff (from SessionHandoffMixin)
    def prepare_handoff(
        self,
        session_id: str | None = None,
        notes: str | None = None,
        recommended_next: str | None = None,
        blockers: list[str] | None = None,
    ) -> dict[str, Any]: ...
    def receive_handoff(self, handoff_context: dict[str, Any]) -> dict[str, Any]: ...

    # Session continuity (from SessionContinuityMixin)
    def get_session_continuity(
        self, session_id: str | None = None
    ) -> dict[str, Any]: ...
    def restore_session_context(self, continuity_data: dict[str, Any]) -> bool: ...

    # Planning (from PlanningMixin)
    def find_bottlenecks(self, top_n: int = 5) -> list[BottleneckDict]: ...
    def get_parallel_work(self, max_agents: int = 5) -> dict[str, Any]: ...
    def recommend_next_work(
        self,
        agent_count: int = 1,
        include_reasons: bool = True,
    ) -> list[WorkRecommendation]: ...
    def assess_risks(self) -> RiskAssessmentDict: ...
    def analyze_impact(self, node_id: str) -> ImpactAnalysisDict: ...
    def get_work_queue(
        self,
        max_items: int = 10,
        include_blocked: bool = False,
    ) -> list[WorkQueueItem]: ...
    def work_next(self) -> ActiveWorkItem | None: ...
    def start_planning_spike(
        self,
        title: str,
        questions: list[str] | None = None,
    ) -> Any: ...
    def create_track_from_plan(
        self,
        spike_id: str,
        track_title: str | None = None,
    ) -> TrackCreationResult: ...
    def smart_plan(
        self,
        goal: str,
        constraints: list[str] | None = None,
        max_features: int = 10,
    ) -> SmartPlanResult: ...
    def plan_parallel_work(
        self,
        available_agents: int = 3,
        work_items: list[str] | None = None,
    ) -> ParallelPlanResult: ...
    def aggregate_parallel_results(
        self,
        task_ids: list[str],
        timeout_seconds: int = 300,
    ) -> AggregateResultsDict: ...

    # Session start info
    def get_session_start_info(
        self,
        include_git_log: bool = True,
        git_log_count: int = 5,
        analytics_top_n: int = 3,
        analytics_max_agents: int = 3,
    ) -> SessionStartInfo: ...

    # Active work item
    def get_active_work_item(
        self,
        agent: str | None = None,
        filter_by_agent: bool = False,
        work_types: list[str] | None = None,
    ) -> ActiveWorkItem | None: ...

    # Operations
    def start_server(
        self,
        port: int = 8080,
        host: str = "localhost",
        watch: bool = True,
        auto_port: bool = False,
    ) -> Any: ...
    def stop_server(self, handle: Any) -> None: ...
    def get_server_status(self, handle: Any | None = None) -> Any: ...
    def install_hooks(self, use_copy: bool = False) -> Any: ...
    def list_hooks(self) -> Any: ...
    def validate_hook_config(self) -> Any: ...
    def export_sessions(self, overwrite: bool = False) -> Any: ...
    def rebuild_event_index(self) -> Any: ...
    def query_events(
        self,
        session_id: str | None = None,
        tool: str | None = None,
        feature_id: str | None = None,
        since: str | None = None,
        limit: int | None = None,
    ) -> Any: ...
    def get_event_stats(self) -> Any: ...
    def analyze_session(self, session_id: str) -> Any: ...
    def analyze_project(self) -> Any: ...
    def get_work_recommendations(self) -> Any: ...
    def get_task_attribution(self, task_id: str) -> dict[str, Any]: ...
    def get_subagent_work(self, session_id: str) -> dict[str, list[dict[str, Any]]]: ...

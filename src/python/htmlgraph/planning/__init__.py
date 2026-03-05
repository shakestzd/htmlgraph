"""HtmlGraph Planning Module - Track/plan models and parallel execution."""

# Core planning models (used by TrackBuilder and SDK)
# Parallel execution engine (Contextune port)
from htmlgraph.planning.builder import (
    ExecutionPlan,
    FileConflict,
    PlanBuilder,
    PlanTask,
    Wave,
)
from htmlgraph.planning.models import (
    AcceptanceCriterion,
    Phase,
    Plan,
    Requirement,
    Spec,
    Task,
    Track,
)

# Worktree management
from htmlgraph.planning.worktree import WorktreeInfo, WorktreeManager

__all__ = [
    # Core models
    "AcceptanceCriterion",
    "Phase",
    "Plan",
    "Requirement",
    "Spec",
    "Task",
    "Track",
    # Execution engine
    "ExecutionPlan",
    "FileConflict",
    "PlanBuilder",
    "PlanTask",
    "Wave",
    # Worktree management
    "WorktreeInfo",
    "WorktreeManager",
]

"""HtmlGraph PlanBuilder - Parallel execution planning with topological sort."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from htmlgraph.sdk import SDK


@dataclass
class PlanTask:
    """A task within a parallel execution plan."""

    id: str
    title: str
    description: str = ""
    priority: str = "medium"  # blocker, high, medium, low
    agent_type: str = "sonnet"  # haiku, sonnet, opus
    files: list[str] = field(default_factory=list)
    depends_on: list[str] = field(default_factory=list)
    wave: int = -1  # Computed by build()
    feature_id: str | None = None  # Set when HtmlGraph feature created


@dataclass
class FileConflict:
    """A file conflict detected between two tasks in the same wave."""

    file: str
    task_a: str
    task_b: str
    wave: int


@dataclass
class Wave:
    """A group of tasks that can run in parallel."""

    number: int
    tasks: list[PlanTask] = field(default_factory=list)

    def is_ready(self, completed_waves: set[int]) -> bool:
        """Check if all prerequisite waves are complete."""
        if self.number == 0:
            return True
        return all(w in completed_waves for w in range(self.number))


@dataclass
class ExecutionPlan:
    """
    A parallel execution plan computed from PlanBuilder.

    Distinct from htmlgraph.planning.models.Plan (the Pydantic track-plan model).
    This dataclass represents a dependency-resolved execution schedule.
    """

    name: str
    waves: list[Wave] = field(default_factory=list)
    conflicts: list[FileConflict] = field(default_factory=list)
    track_id: str | None = None

    @property
    def task_count(self) -> int:
        """Total number of tasks across all waves."""
        return sum(len(w.tasks) for w in self.waves)

    def detect_conflicts(self) -> list[FileConflict]:
        """Detect file conflicts within each wave."""
        conflicts: list[FileConflict] = []
        for wave in self.waves:
            file_owners: dict[str, str] = {}
            for task in wave.tasks:
                for f in task.files:
                    if f in file_owners:
                        conflicts.append(
                            FileConflict(
                                file=f,
                                task_a=file_owners[f],
                                task_b=task.id,
                                wave=wave.number,
                            )
                        )
                    else:
                        file_owners[f] = task.id
        self.conflicts = conflicts
        return conflicts

    def summary(self) -> str:
        """Generate human-readable plan summary."""
        lines = [f"Plan: {self.name}"]
        lines.append(f"Total tasks: {self.task_count} | Waves: {len(self.waves)}")
        if self.track_id:
            lines.append(f"Track: {self.track_id}")
        lines.append("")

        for wave in self.waves:
            lines.append(f"Wave {wave.number} ({len(wave.tasks)} tasks, parallel):")
            for task in wave.tasks:
                deps = ""
                if task.depends_on:
                    deps = f" (depends on: {', '.join(task.depends_on)})"
                lines.append(
                    f"  - {task.id} [{task.agent_type}] [{task.priority}]"
                    f" - {task.title}{deps}"
                )
            lines.append("")

        if self.conflicts:
            lines.append("Conflicts:")
            for c in self.conflicts:
                lines.append(f"  {c.task_a} <-> {c.task_b}: {c.file} (wave {c.wave})")
        else:
            lines.append("No file conflicts detected")

        return "\n".join(lines)

    def get_wave(self, number: int) -> Wave | None:
        """Get a wave by its number."""
        for w in self.waves:
            if w.number == number:
                return w
        return None


class PlanBuilder:
    """Fluent API for building parallel execution plans.

    Computes execution waves via topological sort (Kahn's algorithm) so that
    independent tasks can run in parallel while respecting declared dependencies.

    Usage:
        plan = (
            PlanBuilder(sdk, "My Plan")
            .add_task(id="t1", title="Task 1", files=["a.py"])
            .add_task(id="t2", title="Task 2", files=["b.py"], depends_on=["t1"])
            .build()
        )

        print(plan.summary())
    """

    def __init__(self, sdk: SDK | None = None, name: str = "Unnamed Plan"):
        self._sdk = sdk
        self._name = name
        self._tasks: dict[str, PlanTask] = {}

    def add_task(
        self,
        *,
        id: str,
        title: str,
        description: str = "",
        priority: str = "medium",
        agent_type: str = "sonnet",
        files: list[str] | None = None,
        depends_on: list[str] | None = None,
    ) -> PlanBuilder:
        """Add a task to the plan. Returns self for chaining."""
        self._tasks[id] = PlanTask(
            id=id,
            title=title,
            description=description,
            priority=priority,
            agent_type=agent_type,
            files=files or [],
            depends_on=depends_on or [],
        )
        return self

    def build(self, *, create_track: bool = True) -> ExecutionPlan:
        """Build the plan, computing waves via topological sort.

        Args:
            create_track: If True and SDK available, create Track + Features
                in HtmlGraph for observability tracking.

        Returns:
            ExecutionPlan with waves computed and conflicts detected.
        """
        waves = self._compute_waves()
        plan = ExecutionPlan(name=self._name, waves=waves)
        plan.detect_conflicts()

        if create_track and self._sdk is not None:
            self._create_htmlgraph_entities(plan)

        return plan

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    _PRIORITY_ORDER: dict[str, int] = {
        "blocker": 0,
        "high": 1,
        "medium": 2,
        "low": 3,
    }

    def _task_priority_key(self, task_id: str) -> int:
        return self._PRIORITY_ORDER.get(self._tasks[task_id].priority, 2)

    def _compute_waves(self) -> list[Wave]:
        """Compute execution waves using Kahn's topological sort algorithm."""
        # Build in-degree map and adjacency list
        in_degree: dict[str, int] = {tid: 0 for tid in self._tasks}
        dependents: dict[str, list[str]] = {tid: [] for tid in self._tasks}

        for tid, task in self._tasks.items():
            for dep in task.depends_on:
                if dep in self._tasks:
                    in_degree[tid] += 1
                    dependents[dep].append(tid)

        waves: list[Wave] = []
        remaining = set(self._tasks.keys())
        wave_num = 0

        while remaining:
            # Tasks with no unresolved dependencies are ready for this wave
            ready = [tid for tid in remaining if in_degree[tid] == 0]

            if not ready:
                # Circular dependency detected - break cycle by taking highest priority
                ready = [sorted(remaining, key=self._task_priority_key)[0]]

            wave = Wave(number=wave_num)
            for tid in ready:
                task = self._tasks[tid]
                task.wave = wave_num
                wave.tasks.append(task)
                remaining.discard(tid)

                # Reduce in-degree of dependents now that this task is scheduled
                for dep_tid in dependents.get(tid, []):
                    in_degree[dep_tid] -= 1

            # Sort tasks within wave by priority (blocker first)
            wave.tasks.sort(key=lambda t: self._PRIORITY_ORDER.get(t.priority, 2))
            waves.append(wave)
            wave_num += 1

        return waves

    def _create_htmlgraph_entities(self, plan: ExecutionPlan) -> None:
        """Create Track and Features in HtmlGraph for execution tracking."""
        if self._sdk is None:
            return

        try:
            track: Any = (
                self._sdk.tracks.builder().title(plan.name).priority("high").create()
            )
            plan.track_id = track.id if hasattr(track, "id") else str(track)
        except Exception:
            pass  # Track creation is non-critical

        for wave in plan.waves:
            for task in wave.tasks:
                try:
                    feature: Any = self._sdk.features.create(task.title)
                    if hasattr(feature, "save"):
                        feature.save()
                    task.feature_id = feature.id if hasattr(feature, "id") else None
                except Exception:
                    pass  # Feature creation is non-critical

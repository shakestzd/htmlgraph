from __future__ import annotations

"""Tests for htmlgraph.planning.builder module."""


from htmlgraph.planning.builder import (
    ExecutionPlan as Plan,
)
from htmlgraph.planning.builder import (
    FileConflict,
    PlanBuilder,
    PlanTask,
    Wave,
)


class TestPlanTask:
    """Tests for PlanTask dataclass."""

    def test_defaults(self):
        task = PlanTask(id="t1", title="Test task")
        assert task.id == "t1"
        assert task.title == "Test task"
        assert task.description == ""
        assert task.priority == "medium"
        assert task.agent_type == "sonnet"
        assert task.files == []
        assert task.depends_on == []
        assert task.wave == -1
        assert task.feature_id is None

    def test_custom_values(self):
        task = PlanTask(
            id="t2",
            title="Complex task",
            description="Do complex things",
            priority="blocker",
            agent_type="opus",
            files=["a.py", "b.py"],
            depends_on=["t1"],
            wave=1,
            feature_id="feat-123",
        )
        assert task.priority == "blocker"
        assert task.agent_type == "opus"
        assert len(task.files) == 2
        assert task.depends_on == ["t1"]


class TestWave:
    """Tests for Wave dataclass."""

    def test_empty_wave(self):
        wave = Wave(number=0)
        assert wave.number == 0
        assert wave.tasks == []

    def test_is_ready_wave_0(self):
        wave = Wave(number=0)
        assert wave.is_ready(set()) is True

    def test_is_ready_wave_1_no_prereqs(self):
        wave = Wave(number=1)
        assert wave.is_ready(set()) is False

    def test_is_ready_wave_1_with_prereqs(self):
        wave = Wave(number=1)
        assert wave.is_ready({0}) is True

    def test_is_ready_wave_2(self):
        wave = Wave(number=2)
        assert wave.is_ready({0}) is False
        assert wave.is_ready({0, 1}) is True


class TestPlan:
    """Tests for Plan dataclass."""

    def test_empty_plan(self):
        plan = Plan(name="Test Plan")
        assert plan.task_count == 0
        assert plan.waves == []
        assert plan.conflicts == []

    def test_task_count(self):
        plan = Plan(
            name="Test",
            waves=[
                Wave(
                    number=0,
                    tasks=[
                        PlanTask(id="t1", title="A"),
                        PlanTask(id="t2", title="B"),
                    ],
                ),
                Wave(
                    number=1,
                    tasks=[
                        PlanTask(id="t3", title="C"),
                    ],
                ),
            ],
        )
        assert plan.task_count == 3

    def test_detect_conflicts(self):
        plan = Plan(
            name="Test",
            waves=[
                Wave(
                    number=0,
                    tasks=[
                        PlanTask(id="t1", title="A", files=["shared.py", "a.py"]),
                        PlanTask(id="t2", title="B", files=["shared.py", "b.py"]),
                    ],
                ),
            ],
        )
        conflicts = plan.detect_conflicts()
        assert len(conflicts) == 1
        assert conflicts[0].file == "shared.py"
        assert conflicts[0].task_a == "t1"
        assert conflicts[0].task_b == "t2"

    def test_no_conflicts(self):
        plan = Plan(
            name="Test",
            waves=[
                Wave(
                    number=0,
                    tasks=[
                        PlanTask(id="t1", title="A", files=["a.py"]),
                        PlanTask(id="t2", title="B", files=["b.py"]),
                    ],
                ),
            ],
        )
        assert plan.detect_conflicts() == []

    def test_summary(self):
        plan = Plan(
            name="Test Plan",
            waves=[
                Wave(
                    number=0,
                    tasks=[
                        PlanTask(
                            id="t1",
                            title="Task One",
                            priority="high",
                            agent_type="haiku",
                        ),
                    ],
                ),
            ],
        )
        summary = plan.summary()
        assert "Test Plan" in summary
        assert "t1" in summary
        assert "Task One" in summary
        assert "haiku" in summary

    def test_get_wave(self):
        w0 = Wave(number=0)
        w1 = Wave(number=1)
        plan = Plan(name="Test", waves=[w0, w1])
        assert plan.get_wave(0) is w0
        assert plan.get_wave(1) is w1
        assert plan.get_wave(2) is None


class TestPlanBuilder:
    """Tests for PlanBuilder."""

    def test_empty_build(self):
        builder = PlanBuilder(name="Empty")
        plan = builder.build(create_track=False)
        assert plan.task_count == 0
        assert len(plan.waves) == 0

    def test_single_task(self):
        plan = (
            PlanBuilder(name="Single")
            .add_task(id="t1", title="Only task")
            .build(create_track=False)
        )

        assert plan.task_count == 1
        assert len(plan.waves) == 1
        assert plan.waves[0].tasks[0].id == "t1"

    def test_independent_tasks_same_wave(self):
        plan = (
            PlanBuilder(name="Parallel")
            .add_task(id="t1", title="A", files=["a.py"])
            .add_task(id="t2", title="B", files=["b.py"])
            .add_task(id="t3", title="C", files=["c.py"])
            .build(create_track=False)
        )

        assert len(plan.waves) == 1
        assert plan.task_count == 3

    def test_linear_dependencies(self):
        plan = (
            PlanBuilder(name="Linear")
            .add_task(id="t1", title="First")
            .add_task(id="t2", title="Second", depends_on=["t1"])
            .add_task(id="t3", title="Third", depends_on=["t2"])
            .build(create_track=False)
        )

        assert len(plan.waves) == 3
        assert plan.waves[0].tasks[0].id == "t1"
        assert plan.waves[1].tasks[0].id == "t2"
        assert plan.waves[2].tasks[0].id == "t3"

    def test_diamond_dependency(self):
        # t1 -> t2, t1 -> t3, t2 -> t4, t3 -> t4
        plan = (
            PlanBuilder(name="Diamond")
            .add_task(id="t1", title="Root")
            .add_task(id="t2", title="Left", depends_on=["t1"])
            .add_task(id="t3", title="Right", depends_on=["t1"])
            .add_task(id="t4", title="Join", depends_on=["t2", "t3"])
            .build(create_track=False)
        )

        assert len(plan.waves) == 3
        assert plan.waves[0].tasks[0].id == "t1"
        # t2 and t3 should be in same wave
        wave1_ids = {t.id for t in plan.waves[1].tasks}
        assert wave1_ids == {"t2", "t3"}
        assert plan.waves[2].tasks[0].id == "t4"

    def test_priority_ordering_within_wave(self):
        plan = (
            PlanBuilder(name="Priority")
            .add_task(id="low", title="Low", priority="low")
            .add_task(id="high", title="High", priority="high")
            .add_task(id="blocker", title="Blocker", priority="blocker")
            .build(create_track=False)
        )

        assert len(plan.waves) == 1
        task_ids = [t.id for t in plan.waves[0].tasks]
        assert task_ids[0] == "blocker"
        assert task_ids[1] == "high"
        assert task_ids[2] == "low"

    def test_method_chaining(self):
        builder = PlanBuilder(name="Chain")
        result = builder.add_task(id="t1", title="A")
        assert result is builder  # Returns self for chaining

    def test_file_conflict_detection(self):
        plan = (
            PlanBuilder(name="Conflict")
            .add_task(id="t1", title="A", files=["shared.py"])
            .add_task(id="t2", title="B", files=["shared.py"])
            .build(create_track=False)
        )

        assert len(plan.conflicts) == 1
        assert plan.conflicts[0].file == "shared.py"

    def test_no_sdk_build(self):
        """Build without SDK should work (no tracking)."""
        plan = (
            PlanBuilder(name="No SDK")
            .add_task(id="t1", title="Task")
            .build(create_track=True)  # True but no SDK
        )

        assert plan.task_count == 1
        assert plan.track_id is None

    def test_complex_plan(self):
        """Test a realistic multi-wave plan."""
        plan = (
            PlanBuilder(name="Session Ingestion")
            .add_task(
                id="base",
                title="Base Ingester",
                priority="blocker",
                agent_type="sonnet",
                files=["src/ingestion/base.py"],
            )
            .add_task(
                id="claude",
                title="Claude Adapter",
                priority="high",
                agent_type="haiku",
                files=["src/ingestion/claude.py"],
                depends_on=["base"],
            )
            .add_task(
                id="gemini",
                title="Gemini Adapter",
                priority="high",
                agent_type="haiku",
                files=["src/ingestion/gemini.py"],
                depends_on=["base"],
            )
            .add_task(
                id="codex",
                title="Codex Adapter",
                priority="medium",
                agent_type="haiku",
                files=["src/ingestion/codex.py"],
                depends_on=["base"],
            )
            .add_task(
                id="unified",
                title="Unified Query",
                priority="high",
                agent_type="sonnet",
                files=["src/ingestion/query.py"],
                depends_on=["claude", "gemini", "codex"],
            )
            .build(create_track=False)
        )

        assert len(plan.waves) == 3
        assert plan.waves[0].tasks[0].id == "base"
        wave1_ids = {t.id for t in plan.waves[1].tasks}
        assert wave1_ids == {"claude", "gemini", "codex"}
        assert plan.waves[2].tasks[0].id == "unified"
        assert plan.conflicts == []


class TestFileConflict:
    """Tests for FileConflict dataclass."""

    def test_creation(self):
        conflict = FileConflict(file="shared.py", task_a="t1", task_b="t2", wave=0)
        assert conflict.file == "shared.py"
        assert conflict.task_a == "t1"
        assert conflict.task_b == "t2"
        assert conflict.wave == 0

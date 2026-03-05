from __future__ import annotations

"""HtmlGraph CLI - Planning and parallel execution commands."""

import argparse
import json
from typing import TYPE_CHECKING

from htmlgraph.cli.base import BaseCommand, CommandError, CommandResult
from htmlgraph.cli.constants import DEFAULT_GRAPH_DIR

if TYPE_CHECKING:
    from argparse import _SubParsersAction


def register_commands(subparsers: _SubParsersAction) -> None:
    """Register planning commands with the argument parser."""

    # plan - top-level planning command
    plan_parser = subparsers.add_parser("plan", help="Manage parallel execution plans")
    plan_sub = plan_parser.add_subparsers(dest="plan_command", help="Plan subcommand")

    # plan show
    show_parser = plan_sub.add_parser("show", help="Show current plan")
    show_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    show_parser.set_defaults(func=PlanShowCommand.from_args)

    # plan create
    create_parser = plan_sub.add_parser("create", help="Create a plan from JSON")
    create_parser.add_argument("file", help="JSON file with plan definition")
    create_parser.add_argument("--name", default="Unnamed Plan", help="Plan name")
    create_parser.add_argument(
        "--no-track", action="store_true", help="Don't create HtmlGraph track"
    )
    create_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    create_parser.set_defaults(func=PlanCreateCommand.from_args)

    # worktree - top-level worktree command
    wt_parser = subparsers.add_parser(
        "worktree", help="Manage git worktrees for parallel development"
    )
    wt_sub = wt_parser.add_subparsers(dest="wt_command", help="Worktree subcommand")

    # worktree setup
    setup_parser = wt_sub.add_parser("setup", help="Set up worktrees for plan tasks")
    setup_parser.add_argument(
        "--base-dir", default="worktrees", help="Base directory for worktrees"
    )
    setup_parser.add_argument("--tasks", nargs="*", help="Specific task IDs to set up")
    setup_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    setup_parser.set_defaults(func=WorktreeSetupCommand.from_args)

    # worktree status
    status_parser = wt_sub.add_parser("status", help="Show worktree status")
    status_parser.add_argument(
        "--base-dir", default="worktrees", help="Base directory for worktrees"
    )
    status_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    status_parser.set_defaults(func=WorktreeStatusCommand.from_args)

    # worktree merge
    merge_parser = wt_sub.add_parser("merge", help="Merge a task branch back to main")
    merge_parser.add_argument("task_id", help="Task ID to merge")
    merge_parser.add_argument(
        "--base", default="main", help="Base branch to merge into"
    )
    merge_parser.add_argument(
        "--no-test", action="store_true", help="Skip tests before merge"
    )
    merge_parser.add_argument(
        "--base-dir", default="worktrees", help="Base directory for worktrees"
    )
    merge_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    merge_parser.set_defaults(func=WorktreeMergeCommand.from_args)

    # worktree cleanup
    cleanup_parser = wt_sub.add_parser("cleanup", help="Clean up worktrees")
    cleanup_parser.add_argument("--task", help="Specific task to clean up")
    cleanup_parser.add_argument(
        "--force", action="store_true", help="Force cleanup of unmerged branches"
    )
    cleanup_parser.add_argument(
        "--base-dir", default="worktrees", help="Base directory for worktrees"
    )
    cleanup_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    cleanup_parser.set_defaults(func=WorktreeCleanupCommand.from_args)


# ============================================================================
# Plan Commands
# ============================================================================


class PlanShowCommand(BaseCommand):
    """Show the current plan summary."""

    def __init__(self) -> None:
        super().__init__()

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> PlanShowCommand:
        cmd = cls()
        cmd.graph_dir = getattr(args, "graph_dir", DEFAULT_GRAPH_DIR)
        return cmd

    def execute(self) -> CommandResult:
        """Show latest track and features as a plan summary."""
        from htmlgraph.cli.base import TextOutputBuilder

        sdk = self.get_sdk()

        tracks = sdk.tracks.all()
        if not tracks:
            output = TextOutputBuilder()
            output.add_warning(
                "No plans found. Create one with: htmlgraph plan create <file>"
            )
            return CommandResult(text=output.build())

        latest = tracks[-1]
        output = TextOutputBuilder()
        output.add_info(f"Latest Track: {latest.id}")
        output.add_field("Title", getattr(latest, "title", "Untitled"))
        output.add_field("Status", getattr(latest, "status", "unknown"))

        features = sdk.features.all()
        output.add_blank()
        output.add_line(f"Features: {len(features)}")
        for f in features[:20]:
            status = getattr(f, "status", "unknown")
            title = getattr(f, "title", f.id)
            output.add_line(f"  [{status}] {f.id} - {title}")

        return CommandResult(
            text=output.build(),
            json_data={
                "track_id": latest.id,
                "title": getattr(latest, "title", "Untitled"),
                "status": getattr(latest, "status", "unknown"),
                "features": [
                    {
                        "id": f.id,
                        "title": getattr(f, "title", f.id),
                        "status": getattr(f, "status", "unknown"),
                    }
                    for f in features
                ],
            },
        )


class PlanCreateCommand(BaseCommand):
    """Create a plan from a JSON definition file."""

    def __init__(
        self,
        *,
        file: str,
        name: str,
        no_track: bool,
    ) -> None:
        super().__init__()
        self.file = file
        self.name = name
        self.no_track = no_track

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> PlanCreateCommand:
        return cls(
            file=args.file,
            name=args.name,
            no_track=args.no_track,
        )

    def validate(self) -> None:
        """Validate the plan file exists."""
        import os

        if not os.path.exists(self.file):
            raise CommandError(f"File not found: {self.file}")

    def execute(self) -> CommandResult:
        """Load JSON plan definition and create plan via PlanBuilder."""
        from htmlgraph.cli.base import TextOutputBuilder
        from htmlgraph.planning import PlanBuilder

        try:
            with open(self.file) as f:
                plan_def = json.load(f)
        except json.JSONDecodeError as e:
            raise CommandError(f"Invalid JSON: {e}")

        sdk = self.get_sdk() if not self.no_track else None
        builder = PlanBuilder(sdk=sdk, name=self.name)

        for task in plan_def.get("tasks", []):
            builder.add_task(
                id=task["id"],
                title=task["title"],
                description=task.get("description", ""),
                priority=task.get("priority", "medium"),
                agent_type=task.get("agent_type", "sonnet"),
                files=task.get("files", []),
                depends_on=task.get("depends_on", []),
            )

        plan = builder.build(create_track=not self.no_track)

        output = TextOutputBuilder()
        output.add_success(
            f"Plan created with {plan.task_count} tasks in {len(plan.waves)} waves"
        )
        output.add_blank()
        output.add_line(plan.summary())

        return CommandResult(
            text=output.build(),
            json_data={
                "task_count": plan.task_count,
                "wave_count": len(plan.waves),
            },
        )


# ============================================================================
# Worktree Commands
# ============================================================================


class WorktreeSetupCommand(BaseCommand):
    """Set up worktrees for parallel development."""

    def __init__(self, *, base_dir: str, task_ids: list[str] | None) -> None:
        super().__init__()
        self.base_dir = base_dir
        self.task_ids = task_ids

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> WorktreeSetupCommand:
        task_ids = args.tasks if getattr(args, "tasks", None) else None
        return cls(
            base_dir=args.base_dir,
            task_ids=task_ids,
        )

    def execute(self) -> CommandResult:
        """Set up worktrees for plan task IDs."""
        from htmlgraph.cli.base import TextOutputBuilder
        from htmlgraph.planning import WorktreeManager

        manager = WorktreeManager(base_dir=self.base_dir)

        task_ids = self.task_ids
        if not task_ids:
            sdk = self.get_sdk()
            features = sdk.features.where(status="todo") + sdk.features.where(
                status="in_progress"
            )
            task_ids = [f.id for f in features]
            if not task_ids:
                output = TextOutputBuilder()
                output.add_warning("No pending tasks found.")
                return CommandResult(text=output.build())

        created = manager.setup(task_ids=task_ids)

        output = TextOutputBuilder()
        output.add_success(f"Set up {len(created)} worktrees:")
        for wt in created:
            output.add_line(f"  {wt.task_id}: {wt.path} ({wt.branch})")

        return CommandResult(
            text=output.build(),
            json_data={
                "created": [
                    {"task_id": wt.task_id, "path": str(wt.path), "branch": wt.branch}
                    for wt in created
                ]
            },
        )


class WorktreeStatusCommand(BaseCommand):
    """Show status of all worktrees."""

    def __init__(self, *, base_dir: str) -> None:
        super().__init__()
        self.base_dir = base_dir

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> WorktreeStatusCommand:
        return cls(base_dir=args.base_dir)

    def execute(self) -> CommandResult:
        """Display worktree status table."""
        from htmlgraph.cli.base import TableBuilder, TextOutputBuilder
        from htmlgraph.planning import WorktreeManager

        manager = WorktreeManager(base_dir=self.base_dir)
        infos = manager.status()

        if not infos:
            output = TextOutputBuilder()
            output.add_warning("No active worktrees.")
            return CommandResult(text=output.build(), json_data={"worktrees": []})

        builder = (
            TableBuilder.create_list_table("Worktrees")
            .add_id_column("Task", max_width=25)
            .add_text_column("Branch", max_width=30)
            .add_numeric_column("Commits Ahead", width=14)
            .add_status_column("Changes", width=10)
        )

        for info in infos:
            changes = "yes" if info.has_changes else "no"
            builder.add_row(
                info.task_id,
                info.branch,
                str(info.commits_ahead),
                changes,
            )

        return CommandResult(
            data=builder.table,
            json_data={
                "worktrees": [
                    {
                        "task_id": info.task_id,
                        "branch": info.branch,
                        "commits_ahead": info.commits_ahead,
                        "has_changes": info.has_changes,
                    }
                    for info in infos
                ]
            },
        )


class WorktreeMergeCommand(BaseCommand):
    """Merge a task branch back to main."""

    def __init__(
        self,
        *,
        task_id: str,
        base_branch: str,
        run_tests: bool,
        base_dir: str,
    ) -> None:
        super().__init__()
        self.task_id = task_id
        self.base_branch = base_branch
        self.run_tests = run_tests
        self.base_dir = base_dir

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> WorktreeMergeCommand:
        return cls(
            task_id=args.task_id,
            base_branch=args.base,
            run_tests=not args.no_test,
            base_dir=args.base_dir,
        )

    def execute(self) -> CommandResult:
        """Merge the task branch."""
        from htmlgraph.cli.base import TextOutputBuilder
        from htmlgraph.planning import WorktreeManager

        manager = WorktreeManager(base_dir=self.base_dir)
        success = manager.merge(
            self.task_id,
            base_branch=self.base_branch,
            run_tests=self.run_tests,
        )

        output = TextOutputBuilder()
        if success:
            output.add_success(f"Successfully merged {self.task_id}")
            return CommandResult(
                text=output.build(),
                json_data={"task_id": self.task_id, "merged": True},
            )

        output.add_error(f"Merge failed for {self.task_id}")
        result = CommandResult(
            text=output.build(),
            json_data={"task_id": self.task_id, "merged": False},
        )
        result.exit_code = 1
        return result


class WorktreeCleanupCommand(BaseCommand):
    """Clean up worktrees."""

    def __init__(
        self,
        *,
        task_id: str | None,
        force: bool,
        base_dir: str,
    ) -> None:
        super().__init__()
        self.task_id = task_id
        self.force = force
        self.base_dir = base_dir

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> WorktreeCleanupCommand:
        return cls(
            task_id=getattr(args, "task", None),
            force=args.force,
            base_dir=args.base_dir,
        )

    def execute(self) -> CommandResult:
        """Remove worktrees."""
        from htmlgraph.cli.base import TextOutputBuilder
        from htmlgraph.planning import WorktreeManager

        manager = WorktreeManager(base_dir=self.base_dir)
        removed = manager.cleanup(
            task_id=self.task_id,
            force=self.force,
        )

        output = TextOutputBuilder()
        output.add_success(f"Removed {removed} worktrees")
        return CommandResult(
            text=output.build(),
            json_data={"removed": removed},
        )

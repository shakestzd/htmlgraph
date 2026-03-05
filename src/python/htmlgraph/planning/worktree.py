from __future__ import annotations

"""Git worktree management for parallel task execution."""

import subprocess
from dataclasses import dataclass
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from htmlgraph.planning.builder import ExecutionPlan


@dataclass
class WorktreeInfo:
    """Status information for a single worktree."""

    path: Path
    branch: str
    task_id: str
    commits_ahead: int = 0
    has_changes: bool = False


class WorktreeManager:
    """Manage git worktrees for parallel task execution.

    Usage:
        manager = WorktreeManager()
        manager.setup(task_ids=["feat-001", "feat-002"])
        infos = manager.status()
        manager.merge("feat-001")
        manager.cleanup()
    """

    def __init__(self, base_dir: str = "worktrees", branch_prefix: str = "feature"):
        self.base_dir = Path(base_dir)
        self.branch_prefix = branch_prefix
        self._project_root = self._find_project_root()

    def _find_project_root(self) -> Path:
        result = subprocess.run(
            ["git", "rev-parse", "--show-toplevel"],
            capture_output=True,
            text=True,
            check=True,
        )
        return Path(result.stdout.strip())

    def _run_git(
        self, *args: str, cwd: Path | None = None
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["git", *args],
            capture_output=True,
            text=True,
            cwd=cwd or self._project_root,
        )

    def setup(
        self,
        plan: ExecutionPlan | None = None,
        task_ids: list[str] | None = None,
    ) -> list[WorktreeInfo]:
        """Set up worktrees for plan tasks or explicit task IDs."""
        self.base_dir.mkdir(parents=True, exist_ok=True)

        ids: list[str] = []
        if plan:
            for wave in plan.waves:
                for task in wave.tasks:
                    ids.append(task.id)
        elif task_ids:
            ids = task_ids

        created: list[WorktreeInfo] = []
        for task_id in ids:
            short = task_id.replace("feat-", "").replace("task-", "")
            branch = f"{self.branch_prefix}/{short}"
            wt_path = self.base_dir / short

            if wt_path.exists():
                created.append(
                    WorktreeInfo(path=wt_path, branch=branch, task_id=task_id)
                )
                continue

            ref_exists = self._run_git(
                "show-ref", "--verify", "--quiet", f"refs/heads/{branch}"
            )
            if ref_exists.returncode == 0:
                self._run_git("worktree", "add", str(wt_path), branch)
            else:
                self._run_git("worktree", "add", str(wt_path), "-b", branch)

            created.append(WorktreeInfo(path=wt_path, branch=branch, task_id=task_id))

        return created

    def status(self) -> list[WorktreeInfo]:
        """Get status of all active worktrees."""
        if not self.base_dir.exists():
            return []

        infos: list[WorktreeInfo] = []
        for wt_dir in sorted(self.base_dir.iterdir()):
            if not wt_dir.is_dir():
                continue

            branch_result = self._run_git("branch", "--show-current", cwd=wt_dir)
            branch = (
                branch_result.stdout.strip()
                if branch_result.returncode == 0
                else "detached"
            )

            commits_result = self._run_git(
                "log", "--oneline", "origin/main..HEAD", cwd=wt_dir
            )
            commits = (
                len(commits_result.stdout.strip().splitlines())
                if commits_result.returncode == 0
                else 0
            )

            status_result = self._run_git("status", "--porcelain", cwd=wt_dir)
            has_changes = (
                bool(status_result.stdout.strip())
                if status_result.returncode == 0
                else False
            )

            infos.append(
                WorktreeInfo(
                    path=wt_dir,
                    branch=branch,
                    task_id=wt_dir.name,
                    commits_ahead=commits,
                    has_changes=has_changes,
                )
            )

        return infos

    def merge(
        self, task_id: str, base_branch: str = "main", *, run_tests: bool = True
    ) -> bool:
        """Merge a task branch back to base. Returns True on success."""
        short = task_id.replace("feat-", "").replace("task-", "")
        branch = f"{self.branch_prefix}/{short}"
        wt_path = self.base_dir / short

        if not wt_path.exists():
            return False

        if run_tests:
            test_result = subprocess.run(
                ["uv", "run", "pytest"],
                capture_output=True,
                text=True,
                cwd=wt_path,
            )
            if test_result.returncode != 0:
                return False

        self._run_git("checkout", base_branch)
        result = self._run_git(
            "merge", "--no-ff", branch, "-m", f"feat: merge {task_id}"
        )

        if result.returncode != 0:
            self._run_git("merge", "--abort")
            return False

        return True

    def cleanup(self, task_id: str | None = None, *, force: bool = False) -> int:
        """Clean up worktrees. Returns number removed."""
        removed = 0

        if task_id:
            short = task_id.replace("feat-", "").replace("task-", "")
            wt_path = self.base_dir / short
            branch = f"{self.branch_prefix}/{short}"

            if wt_path.exists():
                self._run_git(
                    "worktree", "remove", str(wt_path), *(["--force"] if force else [])
                )
                self._run_git("branch", "-D" if force else "-d", branch)
                removed = 1
        else:
            if self.base_dir.exists():
                for wt_dir in sorted(self.base_dir.iterdir()):
                    if not wt_dir.is_dir():
                        continue
                    name = wt_dir.name
                    branch = f"{self.branch_prefix}/{name}"

                    merged_result = self._run_git("branch", "--merged", "main")
                    is_merged = (
                        branch in merged_result.stdout
                        if merged_result.returncode == 0
                        else False
                    )

                    if is_merged or force:
                        self._run_git(
                            "worktree",
                            "remove",
                            str(wt_dir),
                            *(["--force"] if force else []),
                        )
                        self._run_git("branch", "-D" if force else "-d", branch)
                        removed += 1

        self._run_git("worktree", "prune")

        if self.base_dir.exists() and not any(self.base_dir.iterdir()):
            self.base_dir.rmdir()

        return removed

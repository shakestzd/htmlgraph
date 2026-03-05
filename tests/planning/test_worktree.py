from __future__ import annotations

"""Tests for htmlgraph.planning.worktree module."""

import subprocess
from pathlib import Path
from unittest.mock import MagicMock, patch

from htmlgraph.planning.worktree import WorktreeInfo, WorktreeManager


class TestWorktreeInfo:
    """Tests for WorktreeInfo dataclass."""

    def test_defaults(self):
        info = WorktreeInfo(
            path=Path("worktrees/task1"),
            branch="feature/task1",
            task_id="task1",
        )
        assert info.commits_ahead == 0
        assert info.has_changes is False

    def test_custom_values(self):
        info = WorktreeInfo(
            path=Path("worktrees/task1"),
            branch="feature/task1",
            task_id="task1",
            commits_ahead=3,
            has_changes=True,
        )
        assert info.commits_ahead == 3
        assert info.has_changes is True


class TestWorktreeManager:
    """Tests for WorktreeManager."""

    @patch("subprocess.run")
    def test_init(self, mock_run: MagicMock):
        mock_run.return_value = subprocess.CompletedProcess(
            args=[], returncode=0, stdout="/path/to/repo\n"
        )
        manager = WorktreeManager()
        assert manager.base_dir == Path("worktrees")
        assert manager.branch_prefix == "feature"

    @patch("subprocess.run")
    def test_custom_base_dir(self, mock_run: MagicMock):
        mock_run.return_value = subprocess.CompletedProcess(
            args=[], returncode=0, stdout="/path/to/repo\n"
        )
        manager = WorktreeManager(base_dir="my-worktrees", branch_prefix="task")
        assert manager.base_dir == Path("my-worktrees")
        assert manager.branch_prefix == "task"

    @patch("subprocess.run")
    def test_status_no_dir(self, mock_run: MagicMock):
        mock_run.return_value = subprocess.CompletedProcess(
            args=[], returncode=0, stdout="/path/to/repo\n"
        )
        manager = WorktreeManager(base_dir="/nonexistent/path")
        infos = manager.status()
        assert infos == []

    @patch("subprocess.run")
    def test_setup_creates_directory(self, mock_run: MagicMock, tmp_path: Path):
        mock_run.return_value = subprocess.CompletedProcess(
            args=[], returncode=0, stdout=str(tmp_path) + "\n"
        )

        base = tmp_path / "worktrees"
        manager = WorktreeManager(base_dir=str(base))

        # Setup with explicit task IDs
        created = manager.setup(task_ids=["task-001"])

        assert base.exists()
        assert len(created) == 1
        assert created[0].task_id == "task-001"

    @patch("subprocess.run")
    def test_cleanup_returns_zero_when_empty(self, mock_run: MagicMock, tmp_path: Path):
        mock_run.return_value = subprocess.CompletedProcess(
            args=[], returncode=0, stdout=str(tmp_path) + "\n"
        )

        base = tmp_path / "worktrees"
        base.mkdir()
        manager = WorktreeManager(base_dir=str(base))
        removed = manager.cleanup()
        assert removed == 0

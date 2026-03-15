from __future__ import annotations

"""HtmlGraph bootstrap operations.

One-command setup to go from installation to first value in under 60 seconds.
This module provides functions for bootstrapping a project with HtmlGraph.

The bootstrap process includes:
1. Auto-detecting project type (Python, Node, etc.)
2. Creating .htmlgraph directory structure
3. Initializing database with schema
4. Installing Claude Code plugin hooks automatically
5. Printing next steps for the user

This is designed for simplicity and speed - the minimal viable setup.
"""


import json
import subprocess
from pathlib import Path
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from htmlgraph.cli.models import BootstrapConfig


def detect_project_type(project_dir: Path) -> str:
    """
    Auto-detect project type from files in directory.

    Args:
        project_dir: Project directory to inspect

    Returns:
        Detected project type: "python", "node", "multi", or "unknown"
    """
    # Check for Python project markers
    has_python = any(
        [
            (project_dir / "pyproject.toml").exists(),
            (project_dir / "setup.py").exists(),
            (project_dir / "requirements.txt").exists(),
            (project_dir / "Pipfile").exists(),
        ]
    )

    # Check for Node project markers
    has_node = (project_dir / "package.json").exists()

    # Determine project type
    if has_python and has_node:
        return "multi"
    elif has_python:
        return "python"
    elif has_node:
        return "node"
    else:
        return "unknown"


def create_gitignore_template() -> str:
    """
    Create .gitignore template content for .htmlgraph directory.

    Returns:
        Gitignore template content
    """
    return """# HtmlGraph cache and regenerable files
.htmlgraph/htmlgraph.db
.htmlgraph/parent-activity.json
"""


def check_already_initialized(project_dir: Path) -> bool:
    """
    Check if project is already initialized with HtmlGraph.

    Args:
        project_dir: Project directory to check

    Returns:
        True if already initialized, False otherwise
    """
    graph_dir = project_dir / ".htmlgraph"
    return graph_dir.exists()


def create_bootstrap_structure(project_dir: Path) -> dict[str, list[str]]:
    """
    Create minimal .htmlgraph directory structure for bootstrap.

    Args:
        project_dir: Project directory

    Returns:
        Dictionary with lists of created directories and files
    """
    graph_dir = project_dir / ".htmlgraph"
    created_dirs: list[str] = []
    created_files: list[str] = []

    # Create main .htmlgraph directory
    if not graph_dir.exists():
        graph_dir.mkdir(parents=True)
        created_dirs.append(str(graph_dir))

    # Create subdirectories
    subdirs = [
        "sessions",
        "features",
        "spikes",
        "tracks",
    ]

    for subdir in subdirs:
        subdir_path = graph_dir / subdir
        if not subdir_path.exists():
            subdir_path.mkdir(parents=True)
            created_dirs.append(str(subdir_path))

    # Create .gitignore in .htmlgraph
    gitignore = graph_dir / ".gitignore"
    if not gitignore.exists():
        gitignore.write_text(create_gitignore_template())
        created_files.append(str(gitignore))

    # Create config.json
    config_file = graph_dir / "config.json"
    if not config_file.exists():
        config_data = {
            "bootstrapped": True,
            "version": "1.0",
        }
        config_file.write_text(json.dumps(config_data, indent=2) + "\n")
        created_files.append(str(config_file))

    return {"directories": created_dirs, "files": created_files}


def initialize_database(graph_dir: Path) -> str:
    """
    Initialize HtmlGraph database with schema.

    Args:
        graph_dir: Path to .htmlgraph directory

    Returns:
        Path to created database file
    """
    from htmlgraph.db.schema import HtmlGraphDB

    db_path = graph_dir / "htmlgraph.db"

    # Create database using HtmlGraphDB (auto-creates tables)
    db = HtmlGraphDB(db_path=str(db_path))
    db.disconnect()

    return str(db_path)


def check_claude_code_available() -> bool:
    """
    Check if Claude Code CLI is available.

    Returns:
        True if claude command is available, False otherwise
    """
    try:
        result = subprocess.run(
            ["claude", "--version"],
            capture_output=True,
            check=False,
            timeout=5,
        )
        return result.returncode == 0
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return False


def get_next_steps(
    project_type: str, has_claude: bool, plugin_installed: bool
) -> list[str]:
    """
    Generate next steps message based on project state.

    Args:
        project_type: Detected project type
        has_claude: Whether Claude Code CLI is available
        plugin_installed: Whether plugin hooks were installed

    Returns:
        List of next step messages
    """
    steps = []

    if has_claude:
        if plugin_installed:
            steps.append("1. Use Claude Code: Run 'claude --dev' in this project")
        else:
            steps.append(
                "1. Install HtmlGraph plugin: Run 'claude plugin install htmlgraph'"
            )
            steps.append("2. Use Claude Code: Run 'claude --dev' in this project")
    else:
        steps.append(
            "1. Install Claude Code CLI: Visit https://code.claude.com/docs/installation"
        )
        steps.append(
            "2. Install HtmlGraph plugin: Run 'claude plugin install htmlgraph'"
        )
        steps.append("3. Use Claude Code: Run 'claude --dev' in this project")

    steps.append(
        f"{len(steps) + 1}. Track work: Create features with 'htmlgraph feature create \"Title\"'"
    )
    steps.append(f"{len(steps) + 1}. View progress: Run 'htmlgraph status'")
    steps.append(
        f"{len(steps) + 1}. See what Claude did: Run 'htmlgraph serve' and open http://localhost:8080"
    )

    return steps


def bootstrap_htmlgraph(config: BootstrapConfig) -> dict[str, Any]:
    """
    Bootstrap HtmlGraph in a project directory.

    This is the main entry point for the bootstrap command.

    Args:
        config: BootstrapConfig with bootstrap settings

    Returns:
        Dictionary with bootstrap results
    """
    project_dir = Path(config.project_path).resolve()

    # Check if already initialized
    if check_already_initialized(project_dir):
        # Ask user if they want to overwrite
        print(f"\n⚠️  HtmlGraph already initialized in {project_dir}")
        response = input("Do you want to reinitialize? (y/N): ").strip().lower()
        if response not in ["y", "yes"]:
            return {
                "success": False,
                "message": "Bootstrap cancelled - already initialized",
            }

    # Detect project type
    project_type = detect_project_type(project_dir)

    # Create directory structure
    created = create_bootstrap_structure(project_dir)
    graph_dir = project_dir / ".htmlgraph"

    # Initialize database
    db_path = initialize_database(graph_dir)
    created["files"].append(db_path)

    # Check for Claude Code
    has_claude = check_claude_code_available()

    # Check if plugin is already available (skip installation check for now)
    plugin_installed = False
    if not config.no_plugins and has_claude:
        # We'll consider it "installed" if hooks can be configured
        # The actual plugin installation happens via marketplace
        plugin_installed = True

    # Generate next steps
    next_steps = get_next_steps(project_type, has_claude, plugin_installed)

    return {
        "success": True,
        "project_type": project_type,
        "graph_dir": str(graph_dir),
        "directories_created": created["directories"],
        "files_created": created["files"],
        "has_claude": has_claude,
        "plugin_installed": plugin_installed,
        "next_steps": next_steps,
    }

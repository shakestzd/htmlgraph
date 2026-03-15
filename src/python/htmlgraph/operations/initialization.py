from __future__ import annotations

"""HtmlGraph initialization operations.

This module provides functions for initializing the .htmlgraph directory structure,
creating necessary files, and optionally installing Git hooks.

The initialization process includes:
1. Directory structure creation (.htmlgraph with subdirectories)
2. Database initialization (htmlgraph.db)
3. Index creation (index.sqlite)
4. Configuration files
5. Optional Git hooks installation

Extracted from monolithic cmd_init() implementation for better maintainability.
"""


import json
import sqlite3
import subprocess
from pathlib import Path
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from htmlgraph.cli.models import InitConfig, InitResult, ValidationResult


# Default collections to create
DEFAULT_COLLECTIONS = [
    "features",
    "bugs",
    "chores",
    "spikes",
    "epics",
    "tracks",
    "sessions",
    "insights",
    "metrics",
    "cigs",
    "patterns",  # Learning collection (SDK patterns API)
    "todos",  # Persistent task tracking
    "task-delegations",  # Spawned agent observability
]

# Additional directories
ADDITIONAL_DIRECTORIES = [
    "archive-index",
    "archives",
]


def validate_directory(base_dir: Path) -> ValidationResult:
    """
    Validate that directory is ready for initialization.

    Args:
        base_dir: Directory to validate

    Returns:
        ValidationResult with validation status and details
    """
    from htmlgraph.cli.models import ValidationResult

    result = ValidationResult(exists=base_dir.exists())

    # Check if directory exists
    if not base_dir.exists():
        result.valid = False
        result.errors.append(f"Directory does not exist: {base_dir}")
        return result

    # Check if already initialized
    graph_dir = base_dir / ".htmlgraph"
    if graph_dir.exists():
        result.is_initialized = True

        # Check for nested .htmlgraph directory (initialization corruption bug)
        nested_graph = graph_dir / ".htmlgraph"
        if nested_graph.exists():
            result.errors.append(
                f"ERROR: Nested .htmlgraph directory detected at {nested_graph}\n"
                "  This indicates initialization corruption.\n"
                "  Fix: Remove nested directory with: rm -rf .htmlgraph/.htmlgraph/"
            )
            result.valid = False
            return result

        result.errors.append(
            f"Directory already initialized: {graph_dir}\n"
            "  Directory already contains .htmlgraph folder"
        )
        result.valid = False
        return result

    # Check if in git repository
    try:
        subprocess.run(
            ["git", "rev-parse", "--git-dir"],
            cwd=base_dir,
            capture_output=True,
            check=True,
        )
        result.has_git = True
    except (subprocess.CalledProcessError, FileNotFoundError):
        result.has_git = False

    return result


def create_directory_structure(base_dir: Path) -> list[str]:
    """
    Create the .htmlgraph directory structure.

    Args:
        base_dir: Base directory (usually current working directory)

    Returns:
        List of created directory paths
    """
    created = []
    graph_dir = base_dir / ".htmlgraph"

    # Create main graph directory
    if not graph_dir.exists():
        graph_dir.mkdir(parents=True)
        created.append(str(graph_dir))

    # Create collection directories
    for collection in DEFAULT_COLLECTIONS:
        coll_dir = graph_dir / collection
        if not coll_dir.exists():
            coll_dir.mkdir(parents=True)
            created.append(str(coll_dir))

    # Create additional directories
    for dirname in ADDITIONAL_DIRECTORIES:
        add_dir = graph_dir / dirname
        if not add_dir.exists():
            add_dir.mkdir(parents=True)
            created.append(str(add_dir))

    return created


def create_database(graph_dir: Path) -> str:
    """
    Create the SQLite database for agent events and sessions.

    Args:
        graph_dir: Path to .htmlgraph directory

    Returns:
        Path to created database file
    """
    db_path = graph_dir / "htmlgraph.db"

    # Create database with schema
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()

    # Sessions table
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS sessions (
            session_id TEXT PRIMARY KEY,
            agent TEXT NOT NULL,
            status TEXT DEFAULT 'active',
            started_at TEXT NOT NULL,
            ended_at TEXT,
            event_count INTEGER DEFAULT 0,
            created_at TEXT DEFAULT CURRENT_TIMESTAMP
        )
    """)

    # Agent events table
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS agent_events (
            event_id TEXT PRIMARY KEY,
            session_id TEXT NOT NULL,
            tool_name TEXT NOT NULL,
            timestamp TEXT NOT NULL,
            success INTEGER DEFAULT 1,
            feature_id TEXT,
            work_type TEXT,
            context TEXT,
            FOREIGN KEY (session_id) REFERENCES sessions(session_id)
        )
    """)

    # Create indexes for common queries
    cursor.execute("""
        CREATE INDEX IF NOT EXISTS idx_events_session
        ON agent_events(session_id)
    """)

    cursor.execute("""
        CREATE INDEX IF NOT EXISTS idx_events_timestamp
        ON agent_events(timestamp)
    """)

    cursor.execute("""
        CREATE INDEX IF NOT EXISTS idx_events_feature
        ON agent_events(feature_id)
    """)

    conn.commit()
    conn.close()

    return str(db_path)


def create_analytics_index(graph_dir: Path) -> str:
    """
    Create the analytics cache database (index.sqlite).

    Args:
        graph_dir: Path to .htmlgraph directory

    Returns:
        Path to created index file
    """
    index_path = graph_dir / "index.sqlite"

    try:
        # Use AnalyticsIndex if available to ensure proper schema
        from htmlgraph.analytics_index import AnalyticsIndex

        index = AnalyticsIndex(str(index_path))
        index.ensure_schema()
    except ImportError:
        # Fallback to simple creation if AnalyticsIndex not available
        conn = sqlite3.connect(index_path)
        cursor = conn.cursor()

        # Create basic cache table structure
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS cache_metadata (
                key TEXT PRIMARY KEY,
                value TEXT,
                updated_at TEXT DEFAULT CURRENT_TIMESTAMP
            )
        """)

        # Store schema version
        cursor.execute("""
            INSERT OR REPLACE INTO cache_metadata (key, value)
            VALUES ('schema_version', '1.0')
        """)

        conn.commit()
        conn.close()

    return str(index_path)


def create_config_files(graph_dir: Path) -> list[str]:
    """
    Create initial configuration files.

    Args:
        graph_dir: Path to .htmlgraph directory

    Returns:
        List of created config file paths
    """
    created = []

    # Create hooks-config.json
    hooks_config = graph_dir / "hooks-config.json"
    if not hooks_config.exists():
        hooks_config.write_text(
            json.dumps(
                {
                    "enabled": True,
                    "track_events": True,
                    "detect_drift": True,
                    "auto_spikes": False,
                },
                indent=2,
            )
        )
        created.append(str(hooks_config))

    # Create drift-queue.json
    drift_queue = graph_dir / "drift-queue.json"
    if not drift_queue.exists():
        drift_queue.write_text(json.dumps([], indent=2))
        created.append(str(drift_queue))

    # Create active-auto-spikes.json
    auto_spikes = graph_dir / "active-auto-spikes.json"
    if not auto_spikes.exists():
        auto_spikes.write_text(json.dumps({}, indent=2))
        created.append(str(auto_spikes))

    return created


def create_hook_scripts(graph_dir: Path) -> list[str]:
    """
    Copy hook scripts to .htmlgraph/hooks directory.

    Args:
        graph_dir: Path to .htmlgraph directory

    Returns:
        List of created hook file paths
    """
    created: list[str] = []

    try:
        # Get the path to the hooks directory in the package
        import htmlgraph

        htmlgraph_dir = Path(htmlgraph.__file__).parent
        hooks_src = htmlgraph_dir / "hooks"

        if not hooks_src.exists():
            return created

        # Create hooks directory in graph_dir
        hooks_dir = graph_dir / "hooks"
        if not hooks_dir.exists():
            hooks_dir.mkdir(parents=True)

        # Copy hook scripts
        hook_names = [
            "post-commit.sh",
            "post-checkout.sh",
            "post-merge.sh",
            "pre-push.sh",
        ]
        for hook_name in hook_names:
            src_hook = hooks_src / hook_name
            dest_hook = hooks_dir / hook_name

            if src_hook.exists() and not dest_hook.exists():
                # Copy the file
                dest_hook.write_text(src_hook.read_text(encoding="utf-8"))
                # Make it executable
                import stat

                dest_hook.chmod(
                    dest_hook.stat().st_mode
                    | stat.S_IXUSR
                    | stat.S_IXGRP
                    | stat.S_IXOTH
                )
                created.append(str(dest_hook))

    except Exception:
        # Silently fail if hooks can't be copied
        pass

    return created


def update_gitignore(base_dir: Path) -> str | None:
    """
    Update .gitignore to exclude HtmlGraph cache files.

    Args:
        base_dir: Base directory containing .gitignore

    Returns:
        Path to .gitignore if updated, None otherwise
    """
    gitignore_path = base_dir / ".gitignore"

    # Read existing .gitignore or create new
    existing_lines = []
    if gitignore_path.exists():
        existing_lines = gitignore_path.read_text().splitlines()

    # Check if HtmlGraph section already exists
    if any("# HtmlGraph" in line for line in existing_lines):
        return None

    # Add HtmlGraph section
    new_lines = [
        "",
        "# HtmlGraph cache and regenerable files",
        ".htmlgraph/index.sqlite",
        ".htmlgraph/index.sqlite.backup",
        ".htmlgraph/database.db",
        ".htmlgraph/parent-activity.json",
    ]

    # Append to existing .gitignore
    with gitignore_path.open("a") as f:
        f.write("\n".join(new_lines) + "\n")

    return str(gitignore_path)


def install_git_hooks(base_dir: Path) -> bool:
    """
    Install Git hooks for event logging.

    Args:
        base_dir: Base directory containing .git

    Returns:
        True if hooks installed successfully, False otherwise
    """
    # Check if .git directory exists
    git_dir = base_dir / ".git"
    if not git_dir.exists():
        return False

    # Import hook installation from operations.hooks
    try:
        from htmlgraph.operations.hooks import install_hooks

        result = install_hooks(project_dir=base_dir, use_copy=False)
        return bool(result.installed)
    except Exception:
        return False


def _run_interactive_setup(graph_dir: Path, result: InitResult) -> None:
    """
    Run interactive setup wizard.

    Args:
        graph_dir: Path to .htmlgraph directory
        result: InitResult to update with created files
    """
    print("\n=== HtmlGraph Interactive Setup ===\n")

    # Ask about project name
    project_name = input("Project name (optional, press Enter to skip): ").strip()

    # Ask about default agent
    default_agent = input("Default agent name (default: claude): ").strip()
    if not default_agent:
        default_agent = "claude"

    # Create config file
    config_file = graph_dir / "config.json"
    if not config_file.exists():
        config_data = {}
        if project_name:
            config_data["project_name"] = project_name
        config_data["default_agent"] = default_agent

        config_file.write_text(json.dumps(config_data, indent=2) + "\n")
        result.files_created.append(str(config_file))
        print(f"\n✓ Created config file: {config_file}")

    print("\n✓ Interactive setup complete!\n")


def create_dashboard_index(base_dir: Path) -> str | None:
    """
    Copy the dashboard HTML file to index.html at the root.

    Args:
        base_dir: Base directory where index.html should be created

    Returns:
        Path to created index.html, or None if not created
    """
    try:
        # Get the path to the dashboard template
        import htmlgraph

        htmlgraph_dir = Path(htmlgraph.__file__).parent
        dashboard_path = htmlgraph_dir / "dashboard.html"

        if not dashboard_path.exists():
            return None

        # Copy to root as index.html
        index_path = base_dir / "index.html"
        index_path.write_text(dashboard_path.read_text(encoding="utf-8"))
        return str(index_path)
    except Exception:
        return None


def initialize_htmlgraph(config: InitConfig) -> InitResult:
    """
    Initialize HtmlGraph directory structure.

    This is the main entry point for the init command.

    Args:
        config: InitConfig with initialization settings

    Returns:
        InitResult with initialization details
    """
    from htmlgraph.cli.models import InitResult

    base_dir = Path(config.dir).resolve()
    graph_dir = base_dir / ".htmlgraph"

    # Validate directory
    validation = validate_directory(base_dir)
    if not validation.valid:
        return InitResult(
            success=False,
            graph_dir=str(graph_dir),
            errors=validation.errors,
        )

    result = InitResult(graph_dir=str(graph_dir))

    try:
        # Create directory structure
        dirs_created = create_directory_structure(base_dir)
        result.directories_created.extend(dirs_created)

        # Create database
        db_path = create_database(graph_dir)
        result.files_created.append(db_path)

        # Create analytics index (unless disabled)
        if not config.no_index:
            index_path = create_analytics_index(graph_dir)
            result.files_created.append(index_path)
        else:
            result.warnings.append("Skipped analytics cache creation (--no-index)")

        # Create config files (unless --no-events-keep)
        if not config.no_events_keep:
            config_files = create_config_files(graph_dir)
            result.files_created.extend(config_files)
        else:
            result.warnings.append("Skipped .gitkeep creation (--no-events-keep)")

        # Create hook scripts
        hook_files = create_hook_scripts(graph_dir)
        result.files_created.extend(hook_files)

        # Create dashboard index.html at root
        dashboard_index = create_dashboard_index(base_dir)
        if dashboard_index:
            result.files_created.append(dashboard_index)

        # Update .gitignore (unless disabled)
        if not config.no_update_gitignore:
            gitignore_path = update_gitignore(base_dir)
            if gitignore_path:
                result.files_created.append(gitignore_path)
                result.warnings.append("Updated .gitignore with HtmlGraph cache rules")
        else:
            result.warnings.append("Skipped .gitignore update (--no-update-gitignore)")

        # Install Git hooks (if requested)
        if config.install_hooks:
            hooks_installed = install_git_hooks(base_dir)
            result.hooks_installed = hooks_installed
            if hooks_installed:
                result.warnings.append("Installed Git hooks for event logging")
            else:
                result.warnings.append(
                    "Failed to install Git hooks (not in git repository?)"
                )

        # Interactive setup wizard (if requested)
        if config.interactive:
            _run_interactive_setup(graph_dir, result)

        # Add Git reminder if not initialized
        if not validation.has_git and not config.install_hooks:
            result.warnings.append(
                "Not in a Git repository. Run 'htmlgraph init --install-hooks' "
                "if you want to track Git events."
            )

        result.success = True

    except Exception as e:
        result.success = False
        result.errors.append(f"Initialization failed: {e}")

    return result

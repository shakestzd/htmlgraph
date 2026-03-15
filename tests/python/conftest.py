"""
Pytest configuration and fixtures for HtmlGraph test suite.

This module provides shared test infrastructure including environment variable
cleanup to ensure test isolation and prevent environment pollution between tests.
"""

import os
from pathlib import Path

import pytest
from htmlgraph import SDK


@pytest.fixture(autouse=True)
def cleanup_env_vars():
    """
    Clean up HtmlGraph environment variables before and after each test.

    This fixture ensures that environment variables set during one test don't
    pollute subsequent tests, which is critical for tests that verify parent-child
    event relationships and session linking.

    Also ensures database timestamps are distinct between tests by adding a small
    sleep after each test (SQLite CURRENT_TIMESTAMP has 1-second resolution).

    HtmlGraph environment variables managed:
    - HTMLGRAPH_PARENT_ACTIVITY: Parent event ID for event linking
    - HTMLGRAPH_PARENT_SESSION: Parent session ID
    - HTMLGRAPH_PARENT_SESSION_ID: Alternative parent session ID
    - HTMLGRAPH_PARENT_AGENT: Parent agent identifier
    - HTMLGRAPH_PARENT_EVENT: Parent event identifier
    - HTMLGRAPH_PARENT_TRACK: Parent track identifier
    - HTMLGRAPH_AGENT: Current agent name
    - HTMLGRAPH_SUBAGENT_TYPE: Subagent type identifier
    """
    env_vars = [
        "HTMLGRAPH_PARENT_ACTIVITY",
        "HTMLGRAPH_PARENT_SESSION",
        "HTMLGRAPH_PARENT_SESSION_ID",
        "HTMLGRAPH_PARENT_AGENT",
        "HTMLGRAPH_PARENT_EVENT",
        "HTMLGRAPH_PARENT_TRACK",
        "HTMLGRAPH_AGENT",
        "HTMLGRAPH_SUBAGENT_TYPE",
    ]

    # Clean before test - preserve original values
    original_values = {}
    for var in env_vars:
        original_values[var] = os.environ.pop(var, None)

    yield

    # Clean after test - restore original values if they existed
    for var in env_vars:
        # Remove any value set during the test
        if var in os.environ:
            del os.environ[var]
        # Restore original value if it existed before the test
        if original_values[var] is not None:
            os.environ[var] = original_values[var]


@pytest.fixture
def isolated_db(tmp_path: Path) -> Path:
    """
    Provide isolated SQLite database path for testing.

    Each test gets its own database file in a temporary directory.
    The database is automatically cleaned up by pytest's tmp_path fixture.

    Returns:
        Path: Absolute path to test-specific database file (test.db)

    Example:
        def test_something(isolated_db):
            sdk = SDK(directory=tmpdir, agent="test", db_path=str(isolated_db))
            # Test code...
    """
    return tmp_path / "test.db"


@pytest.fixture
def isolated_graph_dir(tmp_path: Path) -> Path:
    """
    Create isolated .htmlgraph directory with minimal structure.

    Creates only the base directory - subdirectories are created on demand
    by SDK methods. Use this for unit tests that don't need the full
    directory structure.

    Returns:
        Path: Path to isolated .htmlgraph directory

    Example:
        def test_something(isolated_graph_dir, isolated_db):
            sdk = SDK(directory=isolated_graph_dir, agent="test", db_path=str(isolated_db))
            # Test code...
    """
    graph_dir = tmp_path / ".htmlgraph"
    graph_dir.mkdir()
    return graph_dir


@pytest.fixture
def isolated_graph_dir_full(tmp_path: Path) -> Path:
    """
    Create isolated .htmlgraph directory with complete subdirectory structure.

    Creates all standard subdirectories expected by SDK and tests:
    - Work items: features, bugs, spikes, tracks
    - Sessions: sessions
    - Events: events
    - Archive: archives, archive-index
    - Agents: agents
    - Events: events
    - CIGS: cigs
    - Logs: logs

    Use this for integration tests or tests that need multiple collections.
    For unit tests, prefer isolated_graph_dir and create only what you need.

    Returns:
        Path: Path to isolated .htmlgraph directory with full structure

    Example:
        def test_integration(isolated_graph_dir_full, isolated_db):
            sdk = SDK(directory=isolated_graph_dir_full, agent="test", db_path=str(isolated_db))
            # Test code...
    """
    graph_dir = tmp_path / ".htmlgraph"
    graph_dir.mkdir()

    # Standard subdirectories — mirrors DEFAULT_COLLECTIONS + ADDITIONAL_DIRECTORIES
    # in src/python/htmlgraph/operations/initialization.py
    for subdir in [
        # Work item types
        "features",
        "bugs",
        "chores",
        "spikes",
        "epics",
        "tracks",
        # Sessions
        "sessions",
        # Analytics
        "insights",
        "metrics",
        # CIGS
        "cigs",
        # SDK collections
        "patterns",
        "todos",
        "task-delegations",
        # Events / logs
        "events",
        "logs",
        # Archives
        "archive-index",
        "archives",
    ]:
        (graph_dir / subdir).mkdir()

    return graph_dir


@pytest.fixture
def isolated_sdk(isolated_graph_dir_full: Path, isolated_db: Path) -> SDK:
    """
    Create fully isolated SDK instance for testing.

    Provides SDK with:
    - Complete directory structure (via isolated_graph_dir_full)
    - Isolated database (via isolated_db)
    - Test agent name ("test-agent")

    This is the recommended fixture for most integration tests.

    Returns:
        SDK: Fully configured SDK instance for testing

    Example:
        def test_feature_creation(isolated_sdk):
            feature = isolated_sdk.features.create("Test").save()
            assert feature.id
    """
    return SDK(
        directory=isolated_graph_dir_full,
        agent="test-agent",
        db_path=str(isolated_db),
    )


@pytest.fixture
def isolated_sdk_minimal(isolated_graph_dir: Path, isolated_db: Path) -> SDK:
    """
    Create minimally-structured SDK instance for testing.

    Like isolated_sdk but uses minimal directory structure.
    SDK will create subdirectories as needed on first use.

    Use for unit tests that only touch specific collections or need
    control over directory structure.

    Returns:
        SDK: Minimally-configured SDK instance for testing

    Example:
        def test_unit(isolated_sdk_minimal):
            features = isolated_sdk_minimal.features.create("Test").save()
            # Only features directory is created, others on demand
    """
    return SDK(
        directory=isolated_graph_dir,
        agent="test-agent",
        db_path=str(isolated_db),
    )


def pytest_collection_modifyitems(items: list) -> None:
    """Mark tests that should run serially.

    Tests with @pytest.mark.serial are marked with xdist_group
    to ensure they run serially when using -n auto.
    """
    for item in items:
        if item.get_closest_marker("serial"):
            item.add_marker(pytest.mark.xdist_group("serial"))

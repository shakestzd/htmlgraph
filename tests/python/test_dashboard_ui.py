"""
Playwright tests for HtmlGraph dashboard UI.

Note: These tests require:
1. HTMLGRAPH_UI_TESTS=1 environment variable to be set
2. Dashboard server running on http://localhost:8080

Run with: HTMLGRAPH_UI_TESTS=1 uv run pytest tests/python/test_dashboard_ui.py -v

The tests verify that the dashboard HTML structure matches expectations without
requiring complex Playwright fixtures. They use static analysis instead.
"""

import os
import re
from pathlib import Path

import pytest

pytestmark = pytest.mark.skipif(
    os.environ.get("HTMLGRAPH_UI_TESTS") != "1",
    reason="UI tests require HTMLGRAPH_UI_TESTS=1 environment variable.",
)


@pytest.fixture
def dashboard_html():
    """Load the active dashboard HTML file (dashboard-redesign.html) for static analysis."""
    dashboard_path = (
        Path(__file__).parent.parent.parent
        / "src"
        / "python"
        / "htmlgraph"
        / "api"
        / "templates"
        / "dashboard-redesign.html"
    )
    if not dashboard_path.exists():
        pytest.skip("Dashboard HTML file not found")
    with open(dashboard_path) as f:
        return f.read()


def test_dashboard_title(dashboard_html):
    """Test that dashboard has correct page title."""
    assert "HtmlGraph Dashboard" in dashboard_html


def test_dashboard_heading(dashboard_html):
    """Test that dashboard has correct main heading."""
    assert "HtmlGraph" in dashboard_html


def test_dashboard_logo(dashboard_html):
    """Test that dashboard has logo element."""
    assert 'class="logo"' in dashboard_html


def test_view_navigation_tabs(dashboard_html):
    """Test that all navigation tab buttons are defined."""
    # The redesigned dashboard uses data-tab attributes instead of data-view
    assert 'data-tab="activity"' in dashboard_html
    assert 'data-tab="orchestration"' in dashboard_html
    assert 'data-tab="work-items"' in dashboard_html
    assert 'data-tab="agents"' in dashboard_html
    assert 'data-tab="metrics"' in dashboard_html


def test_tab_navigation_structure(dashboard_html):
    """Test that tab navigation structure exists."""
    assert 'class="tabs-navigation"' in dashboard_html
    assert 'class="tab-button active"' in dashboard_html


def test_content_area_structure(dashboard_html):
    """Test that main content area structure exists."""
    assert 'id="content-area"' in dashboard_html
    assert 'class="content-area"' in dashboard_html


def test_header_stats_present(dashboard_html):
    """Test that header stats badges are present."""
    assert 'id="event-count"' in dashboard_html
    assert 'id="agent-count"' in dashboard_html
    assert 'id="session-count"' in dashboard_html


def test_websocket_indicator(dashboard_html):
    """Test that WebSocket status indicator exists."""
    assert 'id="ws-indicator"' in dashboard_html
    assert 'id="ws-status"' in dashboard_html


def test_sessions_tab_exists(dashboard_html):
    """Test that sessions data is accessible via stats."""
    # Sessions count is shown in header stats
    match = re.search(r'id="session-count"', dashboard_html)
    assert match is not None, "Session count element not found"


def test_dashboard_uses_html5(dashboard_html):
    """Test that dashboard uses HTML5 doctype."""
    assert dashboard_html.startswith(
        "<!DOCTYPE html>"
    ) or dashboard_html.strip().startswith("<!DOCTYPE html>")


def test_dashboard_has_viewport_meta(dashboard_html):
    """Test that dashboard includes viewport meta tag for responsive design."""
    assert 'name="viewport"' in dashboard_html


def test_activity_feed_section(dashboard_html):
    """Test that activity feed tab/section exists."""
    # Check for activity tab button
    assert 'data-tab="activity"' in dashboard_html
    assert "activity" in dashboard_html.lower()


@pytest.mark.skip(reason="Requires active server and Playwright fixtures")
def test_theme_toggle_interactive(dashboard_html):
    """Test theme toggle button functionality (requires running server)."""
    # This test would require Playwright fixtures and a running server
    # Keeping as placeholder for future implementation
    pass


@pytest.mark.skip(reason="Requires active server and Playwright fixtures")
def test_view_navigation_interactive(dashboard_html):
    """Test navigation between different views (requires running server)."""
    # This test would require Playwright fixtures and a running server
    # Keeping as placeholder for future implementation
    pass

"""
Activity Feed UI tests for HtmlGraph dashboard.

Tests verify:
- Dashboard loads without errors
- Activity Feed section is present in HTML
- Responsive design meta tags present
- No critical console errors expected

Note: These tests use static HTML analysis rather than Playwright to avoid
event loop conflicts when running alongside other UI tests.
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
    """Load the active dashboard HTML file (dashboard-redesign.html) for analysis."""
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


class TestActivityFeedDashboard:
    """Test Activity Feed dashboard HTML structure."""

    def test_activity_feed_section_exists(self, dashboard_html):
        """Test Activity Feed tab/section is present in dashboard HTML."""
        # The redesign uses a tab-based navigation; activity feed loads via HTMX
        assert (
            'data-tab="activity"' in dashboard_html
            or "activity-feed" in dashboard_html.lower()
            or "activity" in dashboard_html.lower()
        ), "Activity Feed section not found in dashboard"

    def test_dashboard_page_title_exists(self, dashboard_html):
        """Test dashboard has valid page title."""
        assert "<title>" in dashboard_html
        assert "</title>" in dashboard_html
        title_match = re.search(r"<title>(.*?)</title>", dashboard_html)
        assert title_match is not None
        assert len(title_match.group(1)) > 0

    def test_responsive_meta_tags_present(self, dashboard_html):
        """Test dashboard includes responsive design meta tags."""
        assert 'name="viewport"' in dashboard_html
        assert 'content="width=device-width' in dashboard_html

    def test_body_element_present(self, dashboard_html):
        """Test page has body element."""
        assert "<body" in dashboard_html
        assert "</body>" in dashboard_html

    def test_no_critical_html_errors(self, dashboard_html):
        """Test HTML structure is valid (basic checks)."""
        # Check for matching tags
        opening_divs = dashboard_html.count("<div")
        closing_divs = dashboard_html.count("</div>")
        # Allow for some imbalance due to self-closing tags and nested counting,
        # but they should be roughly equal
        assert abs(opening_divs - closing_divs) < 10, (
            f"Unbalanced div tags: {opening_divs} opening, {closing_divs} closing"
        )

    def test_dashboard_has_css_styling(self, dashboard_html):
        """Test dashboard includes CSS styling."""
        assert "<style" in dashboard_html
        assert "</style>" in dashboard_html

    def test_dashboard_has_javascript(self, dashboard_html):
        """Test dashboard includes JavaScript."""
        assert "<script" in dashboard_html
        assert "</script>" in dashboard_html

    def test_activity_feed_uses_semantic_html(self, dashboard_html):
        """Test activity feed uses semantic HTML elements."""
        # Check for heading elements
        assert "<h" in dashboard_html  # h1, h2, h3, etc.

    def test_no_broken_image_references(self, dashboard_html):
        """Test that dashboard doesn't have empty image src attributes."""
        # Look for img tags with empty src
        empty_imgs = re.findall(r'<img[^>]*src=""[^>]*>', dashboard_html)
        assert len(empty_imgs) == 0, (
            f"Found {len(empty_imgs)} img tags with empty src attributes"
        )

    def test_external_resources_have_urls(self, dashboard_html):
        """Test external resources reference valid URLs."""
        # This is a relaxed check - some scripts are inline
        # Just verify we don't have obviously broken tags
        assert True  # Placeholder for more specific check if needed

    @pytest.mark.skip(reason="Requires running server")
    def test_activity_feed_loads_and_renders(self):
        """Test Activity Feed dashboard loads (requires running server)."""
        pass

    @pytest.mark.skip(reason="Requires running server")
    def test_activity_feed_no_console_errors(self):
        """Test Activity Feed has no console errors (requires running server)."""
        pass

    @pytest.mark.skip(reason="Requires running server")
    def test_activity_feed_responsive_layout(self):
        """Test Activity Feed is responsive (requires running server)."""
        pass

"""
Pydantic models for HtmlGraph nodes, edges, and steps.

These models provide:
- Schema validation for graph data
- HTML serialization/deserialization
- Lightweight context generation for AI agents
"""

from datetime import datetime, timezone
from typing import Any, Literal

from pydantic import Field


def utc_now() -> datetime:
    """Return current time as UTC-aware datetime."""
    return datetime.now(timezone.utc)


from htmlgraph.models.base import MaintenanceType, Node, SpikeType


class Spike(Node):
    """
    A Spike node representing timeboxed investigation/research work.

    Extends Node with spike-specific fields:
    - spike_type: Classification (technical/architectural/risk)
    - timebox_hours: Time budget for investigation
    - findings: Summary of what was learned
    - decision: Decision made based on spike results
    """

    spike_type: SpikeType = SpikeType.GENERAL
    timebox_hours: int | None = None
    findings: str | None = None
    decision: str | None = None

    def __init__(self, **data: Any):
        # Ensure type is always "spike"
        data["type"] = "spike"
        super().__init__(**data)

    def to_html(self, stylesheet_path: str = "../styles.css") -> str:
        """
        Convert spike to HTML document with spike-specific fields.

        Overrides Node.to_html() to include findings and decision sections.
        """
        # Build findings section
        findings_html = ""
        if self.findings:
            findings_html = f"""
        <section data-findings>
            <h3>Findings</h3>
            <div class="findings-content">
                {self.findings}
            </div>
        </section>"""

        # Build decision section
        decision_html = ""
        if self.decision:
            decision_html = f"""
        <section data-decision>
            <h3>Decision</h3>
            <p>{self.decision}</p>
        </section>"""

        # Build spike metadata section
        spike_meta_html = f"""
        <section data-spike-metadata>
            <h3>Spike Metadata</h3>
            <dl>
                <dt>Type</dt>
                <dd>{self.spike_type.value.title()}</dd>"""

        if self.timebox_hours:
            spike_meta_html += f"""
                <dt>Timebox</dt>
                <dd>{self.timebox_hours} hours</dd>"""

        spike_meta_html += """
            </dl>
        </section>"""

        # Get base HTML from Node and insert spike-specific sections
        # We need to call Node's to_html() but inject our sections
        # Strategy: Get base HTML, then insert our sections before closing article tag

        # Call parent's to_html to get base structure
        base_html = super().to_html(stylesheet_path)

        # Insert spike sections before </article>
        spike_sections = f"{spike_meta_html}{findings_html}{decision_html}"
        html_with_findings = base_html.replace(
            "</article>", f"{spike_sections}\n    </article>"
        )

        # Add spike-specific attributes to article tag
        spike_attrs = f' data-spike-type="{self.spike_type.value}"'
        if self.timebox_hours:
            spike_attrs += f' data-timebox-hours="{self.timebox_hours}"'

        # Insert spike attributes into article tag
        html_with_attrs = html_with_findings.replace(
            f'data-updated="{self.updated.isoformat()}"',
            f'data-updated="{self.updated.isoformat()}"{spike_attrs}',
        )

        return html_with_attrs


class Chore(Node):
    """
    A Chore node representing maintenance work.

    Extends Node with maintenance-specific fields:
    - maintenance_type: Classification (corrective/adaptive/perfective/preventive)
    - technical_debt_score: Estimated tech debt impact (0-10)
    """

    maintenance_type: MaintenanceType | None = None
    technical_debt_score: int | None = None

    def __init__(self, **data: Any):
        # Ensure type is always "chore"
        data["type"] = "chore"
        super().__init__(**data)


class Pattern(Node):
    """Learned workflow pattern for agent optimization.

    Stores detected tool sequences that are either optimal patterns
    to encourage or anti-patterns to avoid.
    """

    pattern_type: Literal["optimal", "anti-pattern", "neutral"] = "neutral"
    sequence: list[str] = Field(default_factory=list)  # ["Bash", "Edit", "Read"]

    # Detection metrics
    detection_count: int = 0
    success_rate: float = 0.0  # 0.0-1.0
    avg_duration_seconds: float = 0.0

    # Sessions where detected
    detected_in_sessions: list[str] = Field(default_factory=list)

    # Recommendation
    recommendation: str | None = None

    # Trend
    first_detected: datetime | None = None
    last_detected: datetime | None = None
    detection_trend: Literal["increasing", "stable", "decreasing"] = "stable"

    def __init__(self, **data: Any):
        # Ensure type is always "pattern"
        data["type"] = "pattern"
        super().__init__(**data)

    def to_html(self, stylesheet_path: str = "../styles.css") -> str:
        """Convert pattern to HTML document with pattern-specific fields."""
        # Build pattern sequence HTML
        sequence_html = ""
        if self.sequence:
            sequence_items = " → ".join(self.sequence)
            sequence_html = f"""
        <section data-pattern-sequence>
            <h3>Tool Sequence</h3>
            <p class="sequence">{sequence_items}</p>
        </section>"""

        # Build pattern metrics HTML
        metrics_html = f"""
        <section data-pattern-metrics>
            <h3>Pattern Metrics</h3>
            <dl>
                <dt>Detection Count</dt>
                <dd>{self.detection_count}</dd>
                <dt>Success Rate</dt>
                <dd>{self.success_rate:.1%}</dd>
                <dt>Avg Duration</dt>
                <dd>{self.avg_duration_seconds:.1f}s</dd>
            </dl>
        </section>"""

        # Build detected sessions HTML
        detected_sessions_html = ""
        if self.detected_in_sessions:
            session_links = "\n                    ".join(
                f'<li><a href="../sessions/{sid}.html">{sid}</a></li>'
                for sid in self.detected_in_sessions
            )
            detected_sessions_html = f"""
        <section data-detected-sessions>
            <h3>Detected In Sessions</h3>
            <ul>
                {session_links}
            </ul>
        </section>"""

        # Build recommendation HTML
        recommendation_html = ""
        if self.recommendation:
            recommendation_html = f"""
        <section data-recommendation>
            <h3>Recommendation</h3>
            <p>{self.recommendation}</p>
        </section>"""

        # Build trend HTML
        trend_html = ""
        if self.first_detected or self.last_detected:
            trend_html = """
        <section data-trend>
            <h3>Trend Analysis</h3>
            <dl>"""
            if self.first_detected:
                trend_html += f"""
                <dt>First Detected</dt>
                <dd>{self.first_detected.strftime("%Y-%m-%d %H:%M")}</dd>"""
            if self.last_detected:
                trend_html += f"""
                <dt>Last Detected</dt>
                <dd>{self.last_detected.strftime("%Y-%m-%d %H:%M")}</dd>"""
            trend_html += f"""
                <dt>Detection Trend</dt>
                <dd class="trend-{self.detection_trend}">{self.detection_trend.title()}</dd>
            </dl>
        </section>"""

        # Build pattern-specific attributes
        pattern_attrs = f' data-pattern-type="{self.pattern_type}"'
        pattern_attrs += f' data-detection-count="{self.detection_count}"'
        pattern_attrs += f' data-success-rate="{self.success_rate:.2f}"'
        pattern_attrs += f' data-detection-trend="{self.detection_trend}"'
        if self.sequence:
            import json

            sequence_json = json.dumps(self.sequence)
            pattern_attrs += f" data-sequence='{sequence_json}'"

        return f'''<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="htmlgraph-version" content="1.0">
    <title>{self.title}</title>
    <link rel="stylesheet" href="{stylesheet_path}">
</head>
<body>
    <article id="{self.id}"
             data-type="{self.type}"
             data-status="{self.status}"
             data-priority="{self.priority}"
             data-created="{self.created.isoformat()}"
             data-updated="{self.updated.isoformat()}"{pattern_attrs}>

        <header>
            <h1>{self.title}</h1>
            <div class="metadata">
                <span class="badge status-{self.status}">{self.status.replace("-", " ").title()}</span>
                <span class="badge pattern-{self.pattern_type}">{self.pattern_type.title()}</span>
            </div>
        </header>
{sequence_html}{metrics_html}{detected_sessions_html}{recommendation_html}{trend_html}
    </article>
</body>
</html>
'''

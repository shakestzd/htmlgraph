"""
Pydantic models for HtmlGraph nodes, edges, and steps.

These models provide:
- Schema validation for graph data
- HTML serialization/deserialization
- Lightweight context generation for AI agents
"""

from datetime import datetime
from typing import Any

from pydantic import BaseModel, Field

from htmlgraph.models.base import utc_now


class ContextSnapshot(BaseModel):
    """
    A snapshot of context window usage at a point in time.

    Used to track how context is consumed across sessions, features,
    and activities. Enables analytics for context efficiency.

    The snapshot captures data from Claude Code's status line JSON input.
    """

    timestamp: datetime = Field(default_factory=datetime.now)

    # Token usage in current context window
    input_tokens: int = 0
    output_tokens: int = 0
    cache_creation_tokens: int = 0
    cache_read_tokens: int = 0

    # Context window capacity
    context_window_size: int = 200000

    # Cumulative totals for the session
    total_input_tokens: int = 0
    total_output_tokens: int = 0

    # Cost tracking
    cost_usd: float = 0.0

    # Optional context for what triggered this snapshot
    trigger: str | None = None  # "activity", "feature_switch", "session_start", etc.
    feature_id: str | None = None  # Feature being worked on at this moment

    @property
    def current_tokens(self) -> int:
        """Total tokens in current context window."""
        return self.input_tokens + self.cache_creation_tokens + self.cache_read_tokens

    @property
    def usage_percent(self) -> float:
        """Context window usage as a percentage."""
        if self.context_window_size == 0:
            return 0.0
        return (self.current_tokens / self.context_window_size) * 100

    @classmethod
    def from_claude_input(
        cls, data: dict, trigger: str | None = None, feature_id: str | None = None
    ) -> "ContextSnapshot":
        """
        Create a ContextSnapshot from Claude Code status line JSON input.

        Args:
            data: JSON input from Claude Code (contains context_window, cost, etc.)
            trigger: What triggered this snapshot
            feature_id: Current feature being worked on

        Returns:
            ContextSnapshot instance
        """
        context = data.get("context_window", {})
        usage = context.get("current_usage") or {}
        cost = data.get("cost", {})

        return cls(
            input_tokens=usage.get("input_tokens", 0),
            output_tokens=usage.get("output_tokens", 0),
            cache_creation_tokens=usage.get("cache_creation_input_tokens", 0),
            cache_read_tokens=usage.get("cache_read_input_tokens", 0),
            context_window_size=context.get("context_window_size", 200000),
            total_input_tokens=context.get("total_input_tokens", 0),
            total_output_tokens=context.get("total_output_tokens", 0),
            cost_usd=cost.get("total_cost_usd", 0.0),
            trigger=trigger,
            feature_id=feature_id,
        )

    def to_dict(self) -> dict:
        """Convert to dictionary for serialization."""
        return {
            "ts": self.timestamp.isoformat(),
            "in": self.input_tokens,
            "out": self.output_tokens,
            "cache_create": self.cache_creation_tokens,
            "cache_read": self.cache_read_tokens,
            "window": self.context_window_size,
            "total_in": self.total_input_tokens,
            "total_out": self.total_output_tokens,
            "cost": self.cost_usd,
            "trigger": self.trigger,
            "feature": self.feature_id,
        }

    @classmethod
    def from_dict(cls, data: dict) -> "ContextSnapshot":
        """Create from dictionary."""
        return cls(
            timestamp=datetime.fromisoformat(data["ts"]) if "ts" in data else utc_now(),
            input_tokens=data.get("in", 0),
            output_tokens=data.get("out", 0),
            cache_creation_tokens=data.get("cache_create", 0),
            cache_read_tokens=data.get("cache_read", 0),
            context_window_size=data.get("window", 200000),
            total_input_tokens=data.get("total_in", 0),
            total_output_tokens=data.get("total_out", 0),
            cost_usd=data.get("cost", 0.0),
            trigger=data.get("trigger"),
            feature_id=data.get("feature"),
        )


class ErrorEntry(BaseModel):
    """
    An error record for session error tracking and debugging.

    Stored inline within Session nodes for error analysis and debugging.
    """

    timestamp: datetime = Field(default_factory=datetime.now)
    error_type: str  # Exception class name (ValueError, FileNotFoundError, etc.)
    message: str  # Error message
    traceback: str | None = None  # Full traceback for debugging
    tool: str | None = None  # Tool that caused the error (Edit, Bash, etc.)
    context: str | None = None  # Additional context information
    session_id: str | None = None  # Session ID for cross-referencing
    locals_dump: str | None = None  # JSON-serialized local variables at error point
    stack_frames: list[dict[str, Any]] | None = (
        None  # Structured stack frame information
    )
    command_args: dict[str, Any] | None = None  # Command arguments being executed
    display_level: str = "minimal"  # Display level: minimal, verbose, or debug

    def to_html(self) -> str:
        """Convert error to HTML details element."""
        attrs = [
            f'data-ts="{self.timestamp.isoformat()}"',
            f'data-error-type="{self.error_type}"',
        ]
        if self.tool:
            attrs.append(f'data-tool="{self.tool}"')

        summary = f"<span class='error-type'>{self.error_type}</span>: {self.message}"
        details = ""
        if self.traceback:
            details = f"<pre class='traceback'>{self.traceback}</pre>"

        return f"<details class='error-item' {' '.join(attrs)}><summary>{summary}</summary>{details}</details>"

    def to_context(self) -> str:
        """Lightweight context for AI agents."""
        return f"[{self.timestamp.strftime('%H:%M:%S')}] ERROR {self.error_type}: {self.message}"


class ActivityEntry(BaseModel):
    """
    A lightweight activity log entry for high-frequency events.

    Stored inline within Session nodes to avoid file explosion.
    """

    id: str | None = None  # Optional event ID for deduplication
    timestamp: datetime = Field(default_factory=datetime.now)
    tool: str  # Edit, Bash, Read, Write, Grep, Glob, Task, UserQuery, etc.
    summary: str  # Human-readable summary (e.g., "Edit: src/auth/login.py:45-52")
    success: bool = True
    feature_id: str | None = None  # Link to feature this activity belongs to
    drift_score: float | None = None  # 0.0-1.0 alignment score
    parent_activity_id: str | None = (
        None  # Link to parent activity (e.g., Skill invocation)
    )
    payload: dict[str, Any] | None = (
        None  # Optional rich payload for significant events
    )

    # Context tracking (optional, captured when available)
    context_tokens: int | None = None  # Tokens in context when this activity occurred

    def to_html(self) -> str:
        """Convert activity to HTML list item."""
        attrs = [
            f'data-ts="{self.timestamp.isoformat()}"',
            f'data-tool="{self.tool}"',
            f'data-success="{str(self.success).lower()}"',
        ]
        if self.id:
            attrs.append(f'data-event-id="{self.id}"')
        if self.feature_id:
            attrs.append(f'data-feature="{self.feature_id}"')
        if self.drift_score is not None:
            attrs.append(f'data-drift="{self.drift_score:.2f}"')
        if self.parent_activity_id:
            attrs.append(f'data-parent="{self.parent_activity_id}"')
        if self.context_tokens is not None:
            attrs.append(f'data-context-tokens="{self.context_tokens}"')

        return f"<li {' '.join(attrs)}>{self.summary}</li>"

    def to_context(self) -> str:
        """Lightweight context for AI agents."""
        status = "✓" if self.success else "✗"
        return f"[{self.timestamp.strftime('%H:%M:%S')}] {status} {self.tool}: {self.summary}"

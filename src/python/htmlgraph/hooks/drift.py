"""
Drift Detection for HtmlGraph Hooks.

This module provides utilities for detecting when activities drift away from
the current feature context, and managing the classification queue.

Public Functions:
    load_drift_config() -> dict[str, Any]
        Load drift configuration from plugin config or project .claude directory

    load_drift_queue(graph_dir: Path, max_age_hours: int = 48) -> dict[str, Any]
        Load the drift queue from file and clean up stale entries

    save_drift_queue(graph_dir: Path, queue: dict[str, Any]) -> None
        Save the drift queue to file

    clear_drift_queue_activities(graph_dir: Path) -> None
        Clear activities from the drift queue after successful classification

    add_to_drift_queue(graph_dir: Path, activity: dict[str, Any], config: dict[str, Any]) -> dict[str, Any]
        Add a high-drift activity to the queue

    should_trigger_classification(queue: dict[str, Any], config: dict[str, Any]) -> bool
        Check if we should trigger auto-classification

    build_classification_prompt(queue: dict[str, Any], feature_id: str) -> str
        Build the prompt for the classification agent
"""

import json
import logging
import os
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any, cast

from htmlgraph.hooks.constants import DRIFT_QUEUE_FILE

logger = logging.getLogger(__name__)


def load_drift_config() -> dict[str, Any]:
    """Load drift configuration from plugin config or project .claude directory."""
    config_paths = [
        Path(__file__).parent.parent.parent.parent.parent
        / ".claude"
        / "config"
        / "drift-config.json",
        Path(os.environ.get("CLAUDE_PROJECT_DIR", ""))
        / ".claude"
        / "config"
        / "drift-config.json",
        Path(os.environ.get("CLAUDE_PLUGIN_ROOT", "")) / "config" / "drift-config.json",
    ]

    for config_path in config_paths:
        if config_path.exists():
            try:
                with open(config_path) as f:
                    return cast(dict[Any, Any], json.load(f))
            except Exception:
                pass

    # Default config
    return {
        "drift_detection": {
            "enabled": True,
            "warning_threshold": 0.7,
            "auto_classify_threshold": 0.85,
            "min_activities_before_classify": 3,
            "cooldown_minutes": 10,
        },
        "classification": {"enabled": True, "use_haiku_agent": True},
        "queue": {
            "max_pending_classifications": 5,
            "max_age_hours": 48,
            "process_on_stop": True,
            "process_on_threshold": True,
        },
    }


def load_drift_queue(graph_dir: Path, max_age_hours: int = 48) -> dict[str, Any]:
    """
    Load the drift queue from file and clean up stale entries.

    Args:
        graph_dir: Path to .htmlgraph directory
        max_age_hours: Maximum age in hours before activities are removed (default: 48)

    Returns:
        Drift queue dict with only recent activities
    """
    queue_path = graph_dir / DRIFT_QUEUE_FILE
    if queue_path.exists():
        try:
            with open(queue_path) as f:
                queue = json.load(f)

            # Filter out stale activities
            cutoff_time = datetime.now() - timedelta(hours=max_age_hours)
            original_count = len(queue.get("activities", []))

            fresh_activities = []
            for activity in queue.get("activities", []):
                try:
                    activity_time = datetime.fromisoformat(
                        activity.get("timestamp", "")
                    )
                    if activity_time >= cutoff_time:
                        fresh_activities.append(activity)
                except (ValueError, TypeError):
                    # Keep activities with invalid timestamps to avoid data loss
                    fresh_activities.append(activity)

            # Update queue if we removed stale entries
            if len(fresh_activities) < original_count:
                queue["activities"] = fresh_activities
                save_drift_queue(graph_dir, queue)
                removed = original_count - len(fresh_activities)
                logger.warning(
                    f"Cleaned {removed} stale drift queue entries (older than {max_age_hours}h)"
                )

            return cast(dict[Any, Any], queue)
        except Exception:
            pass
    return {"activities": [], "last_classification": None}


def save_drift_queue(graph_dir: Path, queue: dict[str, Any]) -> None:
    """Save the drift queue to file."""
    queue_path = graph_dir / DRIFT_QUEUE_FILE
    try:
        with open(queue_path, "w") as f:
            json.dump(queue, f, indent=2, default=str)
    except Exception as e:
        logger.warning(f"Warning: Could not save drift queue: {e}")


def clear_drift_queue_activities(graph_dir: Path) -> None:
    """
    Clear activities from the drift queue after successful classification.

    This removes stale entries that have been processed, preventing indefinite accumulation.
    """
    queue_path = graph_dir / DRIFT_QUEUE_FILE
    try:
        # Load existing queue to preserve last_classification timestamp
        queue = {"activities": [], "last_classification": datetime.now().isoformat()}
        if queue_path.exists():
            with open(queue_path) as f:
                existing = json.load(f)
                # Preserve the classification timestamp if it exists
                if existing.get("last_classification"):
                    queue["last_classification"] = existing["last_classification"]

        # Save cleared queue
        with open(queue_path, "w") as f:
            json.dump(queue, f, indent=2)
    except Exception as e:
        logger.warning(f"Warning: Could not clear drift queue: {e}")


def add_to_drift_queue(
    graph_dir: Path, activity: dict[str, Any], config: dict[str, Any]
) -> dict[str, Any]:
    """Add a high-drift activity to the queue."""
    max_age_hours = config.get("queue", {}).get("max_age_hours", 48)
    queue = load_drift_queue(graph_dir, max_age_hours=max_age_hours)
    max_pending = config.get("queue", {}).get("max_pending_classifications", 5)

    queue["activities"].append(
        {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "tool": activity.get("tool"),
            "summary": activity.get("summary"),
            "file_paths": activity.get("file_paths", []),
            "drift_score": activity.get("drift_score"),
            "feature_id": activity.get("feature_id"),
        }
    )

    # Keep only recent activities
    queue["activities"] = queue["activities"][-max_pending:]
    save_drift_queue(graph_dir, queue)
    return queue


def should_trigger_classification(
    queue: dict[str, Any], config: dict[str, Any]
) -> bool:
    """Check if we should trigger auto-classification."""
    drift_config = config.get("drift_detection", {})

    if not config.get("classification", {}).get("enabled", True):
        return False

    min_activities = drift_config.get("min_activities_before_classify", 3)
    cooldown_minutes = drift_config.get("cooldown_minutes", 10)

    # Check minimum activities threshold
    if len(queue.get("activities", [])) < min_activities:
        return False

    # Check cooldown
    last_classification = queue.get("last_classification")
    if last_classification:
        try:
            last_time = datetime.fromisoformat(last_classification)
            if datetime.now() - last_time < timedelta(minutes=cooldown_minutes):
                return False
        except Exception:
            pass

    return True


def build_classification_prompt(queue: dict[str, Any], feature_id: str) -> str:
    """Build the prompt for the classification agent."""
    activities = queue.get("activities", [])

    activity_lines = []
    for act in activities:
        line = f"- {act.get('tool', 'unknown')}: {act.get('summary', 'no summary')}"
        if act.get("file_paths"):
            line += f" (files: {', '.join(act['file_paths'][:2])})"
        line += f" [drift: {act.get('drift_score', 0):.2f}]"
        activity_lines.append(line)

    return f"""Classify these high-drift activities into a work item.

Current feature context: {feature_id}

Recent activities with high drift:
{chr(10).join(activity_lines)}

Based on the activity patterns:
1. Determine the work item type (bug, feature, spike, chore, or hotfix)
2. Create an appropriate title and description
3. Create the work item HTML file in .htmlgraph/

Use the classification rules:
- bug: fixing errors, incorrect behavior
- feature: new functionality, additions
- spike: research, exploration, investigation
- chore: maintenance, refactoring, cleanup
- hotfix: urgent production issues

Create the work item now using Write tool."""


__all__ = [
    "load_drift_config",
    "load_drift_queue",
    "save_drift_queue",
    "clear_drift_queue_activities",
    "add_to_drift_queue",
    "should_trigger_classification",
    "build_classification_prompt",
]

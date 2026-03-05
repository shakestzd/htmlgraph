from __future__ import annotations

"""HtmlGraph CLI - Activity tracking commands."""

import argparse
import json
import os
from typing import TYPE_CHECKING

from htmlgraph.cli.base import BaseCommand, CommandError, CommandResult
from htmlgraph.cli.constants import DEFAULT_GRAPH_DIR

if TYPE_CHECKING:
    from argparse import _SubParsersAction


def register_activity_commands(subparsers: _SubParsersAction) -> None:
    """Register activity tracking commands.

    Args:
        subparsers: Subparser action from ArgumentParser.add_subparsers()
    """
    activity_parser = subparsers.add_parser("activity", help="Track an activity event")
    activity_parser.add_argument("tool", help="Tool name (e.g. Edit, Bash, Read)")
    activity_parser.add_argument("summary", help="Human-readable summary")
    activity_parser.add_argument(
        "--file", "-f", action="append", dest="files", help="File involved"
    )
    activity_parser.add_argument(
        "--success", choices=["true", "false"], default="true", help="Success flag"
    )
    activity_parser.add_argument("--feature-id", help="Explicit feature ID")
    activity_parser.add_argument("--session-id", help="Explicit session ID")
    activity_parser.add_argument("--payload", help="JSON payload")
    activity_parser.add_argument("--payload-file", help="Path to JSON payload file")
    activity_parser.add_argument("--agent", help="Agent name override")
    activity_parser.add_argument(
        "--graph-dir", "-g", default=DEFAULT_GRAPH_DIR, help="Graph directory"
    )
    activity_parser.set_defaults(func=ActivityCommand.from_args)


class ActivityCommand(BaseCommand):
    """Track an activity event."""

    def __init__(
        self,
        *,
        tool: str,
        summary: str,
        files: list[str] | None = None,
        success: bool = True,
        feature_id: str | None = None,
        session_id: str | None = None,
        payload: dict | None = None,
        agent: str | None = None,
    ) -> None:
        super().__init__()
        self.tool = tool
        self.summary = summary
        self.files = files
        self.success = success
        self.feature_id = feature_id
        self.session_id = session_id
        self.payload = payload
        self.agent_name = agent

    @classmethod
    def from_args(cls, args: argparse.Namespace) -> ActivityCommand:
        # Parse success
        success = args.success.lower() == "true"

        # Parse payload
        payload = None
        if args.payload:
            try:
                payload = json.loads(args.payload)
            except json.JSONDecodeError as e:
                raise CommandError(f"Invalid JSON payload: {e}")

        if args.payload_file:
            if not os.path.exists(args.payload_file):
                raise CommandError(f"Payload file not found: {args.payload_file}")
            try:
                with open(args.payload_file) as f:
                    payload = json.load(f)
            except json.JSONDecodeError as e:
                raise CommandError(f"Invalid JSON in payload file: {e}")
            except Exception as e:
                raise CommandError(f"Error reading payload file: {e}")

        return cls(
            tool=args.tool,
            summary=args.summary,
            files=args.files,
            success=success,
            feature_id=args.feature_id,
            session_id=args.session_id,
            payload=payload,
            agent=args.agent,
        )

    def execute(self) -> CommandResult:
        """Track the activity."""
        sdk = self.get_sdk()

        # track_activity will handle session_id discovery and attribution
        entry = sdk.track_activity(
            tool=self.tool,
            summary=self.summary,
            file_paths=self.files,
            success=self.success,
            feature_id=self.feature_id,
            session_id=self.session_id,
            payload=self.payload,
        )

        return CommandResult(
            text=f"Tracked activity: [{entry.tool}] {entry.summary}",
            data={
                "id": entry.id,
                "tool": entry.tool,
                "summary": entry.summary,
                "feature_id": entry.feature_id,
            },
        )

#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "htmlgraph>=0.34.13",
# ]
# ///
"""
PreToolUse: Attribution Check — Warn when tools are called without attribution.

Fires before ALL tool calls (Agent, Task, Bash, Read, Edit, Write, etc.).
Checks whether sdk.features.start() was called since the last UserQuery
in this session. Uses two signals:

1. Primary (canonical): sdk.get_active_work_item() — reads HTML files
   (the source of truth). If any work item is in-progress, no warning.

2. Secondary (belt-and-suspenders): scan agent_events for a Bash event
   after the last UserQuery whose input_summary mentions 'features.start'.
   Catches cases where active_feature_id was set mid-turn via SDK.

Skips internal/system tools that don't represent actual work (TodoRead,
TodoWrite, AskUserQuestion).

If neither signal fires, inject a warning reminding Claude to verify the
active work item before proceeding.

Decision is always "allow" — this hook warns, never blocks.
"""

import json
import os
import sqlite3
import sys

try:
    from htmlgraph.hooks.version_check import check_hook_version

    check_hook_version("0.34.14")
except Exception:
    pass


def _db_path(project_dir: str) -> str:
    return os.path.join(project_dir, ".htmlgraph", "htmlgraph.db")


def _resolve_project_dir() -> str:
    env_dir = os.environ.get("CLAUDE_PROJECT_DIR")
    if env_dir:
        return env_dir
    return os.getcwd()


def _attribution_verified(conn: sqlite3.Connection, session_id: str) -> bool:
    """
    Return True if attribution has been set/verified this turn.

    Checks two signals:
    1. sdk.get_active_work_item() finds an in-progress item in HTML (canonical).
    2. A Bash event after the last UserQuery references 'features.start'.
    """
    # Signal 1: Check HTML files for active work item (canonical source)
    try:
        from htmlgraph import SDK

        sdk = SDK()
        active = sdk.get_active_work_item()
        if active:
            return True
    except Exception:
        pass

    # Signal 2: Bash event after last UserQuery mentioning features.start
    uq_row = conn.execute(
        "SELECT MAX(timestamp) FROM agent_events WHERE session_id = ? AND tool_name = 'UserQuery'",
        (session_id,),
    ).fetchone()

    if not uq_row or not uq_row[0]:
        # No UserQuery yet — can't verify turn boundary; allow without warning.
        # Callers that need to warn on Task regardless of UserQuery state
        # should check the tool_name before calling this function.
        return True

    last_userquery_ts = uq_row[0]

    count_row = conn.execute(
        """
        SELECT COUNT(*) FROM agent_events
        WHERE session_id = ?
          AND tool_name = 'Bash'
          AND timestamp > ?
          AND (
              input_summary LIKE '%features.start%'
              OR input_summary LIKE '%sdk.features.start%'
              OR input_summary LIKE '%bugs.start%'
              OR input_summary LIKE '%sdk.bugs.start%'
              OR input_summary LIKE '%spikes.start%'
              OR input_summary LIKE '%sdk.spikes.start%'
          )
        """,
        (session_id, last_userquery_ts),
    ).fetchone()

    return bool(count_row and count_row[0] > 0)


_WARNING_MESSAGE = (
    "ATTRIBUTION CHECK: You are about to execute a tool without "
    "verifying the active work item this turn.\n\n"
    "All tool events will be attributed to whatever feature is "
    "currently active — which may be wrong if the user's request changed.\n\n"
    "Before proceeding:\n"
    "  1. Check the CIGS 'ACTIVE:' item in the system reminder.\n"
    "  2. If it does not match the current task, call "
    "sdk.features.start('correct-id') first.\n"
    "  3. Then execute your tool.\n\n"
    "To suppress this warning: call sdk.features.start() before your tool call."
)

_ESCALATED_ATTRIBUTION_WARNING = (
    "⚠️ REPEATED UNATTRIBUTED TOOL CALLS ({count} tools without work item)\n\n"
    "You have executed {count} tool calls without an active work item. "
    "All are attributed to NULL.\n\n"
    "REQUIRED NOW:\n"
    "  sdk.bugs.create('title').save() → sdk.bugs.start('id')\n"
    "  OR sdk.features.start('existing-id')\n\n"
    "Do this BEFORE your next tool call."
)

_STRONG_ATTRIBUTION_WARNING = (
    "🛑 ATTRIBUTION ENFORCEMENT: {count} unattributed tool calls this turn\n\n"
    "You are consistently executing tools without attribution. "
    "This defeats work tracking.\n\n"
    "STOP and run: sdk.bugs.start('id') or sdk.features.start('id')\n\n"
    "Check CIGS OPEN line for existing work items, or create one now."
)

_AGENT_DELEGATION_WARNING = (
    "STOP: AGENT DELEGATION WITHOUT ACTIVE WORK ITEM\n\n"
    "You are about to delegate to a subagent, but no work item is active. "
    "All subagent tool calls will be attributed to NULL — orphaned in the dashboard.\n\n"
    "REQUIRED before delegating:\n"
    "  1. Find or create the right work item:\n"
    "     sdk.bugs.create('title').save() or sdk.features.create('title').save()\n"
    "  2. Start it: sdk.bugs.start('id') or sdk.features.start('id')\n"
    "  3. THEN delegate with Task/Agent\n\n"
    "This is mandatory — delegation without attribution defeats the purpose of work tracking."
)

_STEP_REMINDER_MESSAGE = (
    "STEP TRACKING REMINDER: The active feature has steps that haven't been "
    "updated this session.\n\n"
    "The dashboard shows all steps as 'pending' because no step completions "
    "have been recorded.\n\n"
    "When you finish a step, mark it:\n"
    "  with sdk.features.edit('feat-xxx') as f:\n"
    "      f.steps[N].completed = True\n\n"
    "Or check the CIGS 'Steps:' line to see which steps remain."
)


def _count_unattributed_events(conn: sqlite3.Connection, session_id: str) -> int:
    """Count tool calls since the last UserQuery that have no feature attribution.

    Returns the number of unattributed events, used to escalate warning severity.
    Query is intentionally simple for speed (< 100ms with session_id index).
    """
    uq_row = conn.execute(
        "SELECT MAX(timestamp) FROM agent_events WHERE session_id = ? AND tool_name = 'UserQuery'",
        (session_id,),
    ).fetchone()

    if not uq_row or not uq_row[0]:
        return 0

    last_userquery_ts = uq_row[0]

    count_row = conn.execute(
        """
        SELECT COUNT(*) FROM agent_events
        WHERE session_id = ?
          AND feature_id IS NULL
          AND tool_name NOT IN ('UserQuery', 'TodoRead', 'TodoWrite')
          AND timestamp > ?
        """,
        (session_id, last_userquery_ts),
    ).fetchone()

    return count_row[0] if count_row else 0


def _check_stale_steps(
    conn: sqlite3.Connection, session_id: str, feature_id: str
) -> bool:
    """Return True if active feature has many tool calls but no step updates.

    Looks for 10+ tool calls in this turn with no Bash commands referencing
    step completion patterns. Only fires once per turn boundary.
    """
    # Find the last UserQuery timestamp (turn boundary)
    uq_row = conn.execute(
        "SELECT MAX(timestamp) FROM agent_events WHERE session_id = ? AND tool_name = 'UserQuery'",
        (session_id,),
    ).fetchone()

    if not uq_row or not uq_row[0]:
        return False

    last_userquery_ts = uq_row[0]

    # Count tool calls since the last UserQuery
    tool_count_row = conn.execute(
        "SELECT COUNT(*) FROM agent_events WHERE session_id = ? AND timestamp > ?",
        (session_id, last_userquery_ts),
    ).fetchone()

    tool_count = tool_count_row[0] if tool_count_row else 0
    if tool_count < 10:
        return False

    # Check if any Bash event since last UserQuery references step completion
    step_updates_row = conn.execute(
        """SELECT COUNT(*) FROM agent_events
           WHERE session_id = ?
             AND tool_name = 'Bash'
             AND timestamp > ?
             AND (
                 input_summary LIKE '%complete_step%'
                 OR input_summary LIKE '%.completed = True%'
                 OR input_summary LIKE '%steps[%completed%'
                 OR input_summary LIKE '%mark_step%'
             )""",
        (session_id, last_userquery_ts),
    ).fetchone()

    step_updates = step_updates_row[0] if step_updates_row else 0
    return step_updates == 0


def main() -> None:
    try:
        hook_input = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError):
        print(json.dumps({"decision": "allow"}))
        return

    # Skip tools that don't represent actual work (internal/system tools)
    skip_tools = {"TodoRead", "TodoWrite", "AskUserQuestion"}
    tool_name = hook_input.get("tool_name", "")
    if tool_name in skip_tools:
        print(json.dumps({"decision": "allow"}))
        return

    is_delegation = tool_name == "Task"

    session_id = hook_input.get("session_id", "") or os.environ.get(
        "HTMLGRAPH_SESSION_ID", ""
    )
    if not session_id:
        # For Task calls with no session context, still warn — delegation
        # without attribution is always wrong regardless of session state.
        if is_delegation:
            # Check HTML for active work item before warning (fast path)
            try:
                from htmlgraph import SDK

                sdk = SDK()
                if not sdk.get_active_work_item():
                    print(
                        json.dumps(
                            {"decision": "allow", "message": _AGENT_DELEGATION_WARNING}
                        )
                    )
                    return
            except Exception:
                pass
        print(json.dumps({"decision": "allow"}))
        return

    project_dir = _resolve_project_dir()
    db_file = _db_path(project_dir)
    if not os.path.exists(db_file):
        print(json.dumps({"decision": "allow"}))
        return

    try:
        conn = sqlite3.connect(db_file, timeout=3)
        conn.row_factory = sqlite3.Row
        try:
            # For Task calls, check active work item directly first so we warn
            # even when there is no UserQuery yet (first tool call of session).
            if is_delegation:
                try:
                    from htmlgraph import SDK

                    sdk = SDK()
                    active = sdk.get_active_work_item()
                except Exception:
                    active = None
                if not active:
                    print(
                        json.dumps(
                            {"decision": "allow", "message": _AGENT_DELEGATION_WARNING}
                        )
                    )
                    return
                # Active item found — fall through to stale-step check below
                verified = True
                feature_id = active.get("id") if active else None
            else:
                verified = _attribution_verified(conn, session_id)
                feature_id = None

            if verified:
                # Check whether step tracking has gone stale
                if feature_id is None:
                    # Get active feature ID from HTML (canonical source)
                    try:
                        from htmlgraph import SDK

                        sdk = SDK()
                        active_item = sdk.get_active_work_item()
                        feature_id = active_item.get("id") if active_item else None
                    except Exception:
                        feature_id = None
                if feature_id and _check_stale_steps(conn, session_id, feature_id):
                    print(
                        json.dumps(
                            {"decision": "allow", "message": _STEP_REMINDER_MESSAGE}
                        )
                    )
                    return
        finally:
            conn.close()
    except Exception:
        # Never block Claude on DB errors
        print(json.dumps({"decision": "allow"}))
        return

    if verified:
        print(json.dumps({"decision": "allow"}))
    else:
        # Use stronger warning for delegation — most impactful case
        if is_delegation:
            print(
                json.dumps({"decision": "allow", "message": _AGENT_DELEGATION_WARNING})
            )
        else:
            try:
                conn = sqlite3.connect(db_file, timeout=3)
                conn.row_factory = sqlite3.Row
                try:
                    unattributed_count = _count_unattributed_events(conn, session_id)
                finally:
                    conn.close()
            except Exception:
                unattributed_count = 0

            if unattributed_count >= 8:
                message = _STRONG_ATTRIBUTION_WARNING.format(count=unattributed_count)
            elif unattributed_count >= 4:
                message = _ESCALATED_ATTRIBUTION_WARNING.format(
                    count=unattributed_count
                )
            else:
                message = _WARNING_MESSAGE

            print(json.dumps({"decision": "allow", "message": message}))


if __name__ == "__main__":
    main()

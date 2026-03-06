"""
OpenTelemetry Export for HtmlGraph.

Exports HtmlGraph sessions and events as OTLP traces/spans.

Mapping:
    HtmlGraph session  -> OTLP trace  (TraceId derived from session_id)
    HtmlGraph event    -> OTLP span   (SpanId derived from event_id)

Usage:
    htmlgraph export otel [--endpoint http://localhost:4317] [--graph-dir .htmlgraph]

The exporter sends data via HTTP/JSON to the OTLP HTTP endpoint (port 4318 by default
for HTTP/JSON; port 4317 is typically used for gRPC).  No opentelemetry-sdk dependency
is required — raw OTLP/JSON is assembled and sent via the stdlib ``urllib.request``.
"""

from __future__ import annotations

import hashlib
import json
import logging
import sqlite3
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# ID helpers
# ---------------------------------------------------------------------------


def _to_trace_id(session_id: str) -> str:
    """Derive a 32-hex-char TraceId from a session_id string."""
    return hashlib.md5(session_id.encode(), usedforsecurity=False).hexdigest()  # noqa: S324


def _to_span_id(event_id: str) -> str:
    """Derive a 16-hex-char SpanId from an event_id string."""
    return hashlib.md5(event_id.encode(), usedforsecurity=False).hexdigest()[:16]  # noqa: S324


def _ts_to_unix_nano(ts_str: str) -> int:
    """Convert ISO-8601 timestamp string to Unix nanoseconds."""
    ts_str = ts_str.replace("Z", "+00:00")
    try:
        dt = datetime.fromisoformat(ts_str)
    except ValueError:
        dt = datetime.now(timezone.utc)
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return int(dt.timestamp() * 1_000_000_000)


# ---------------------------------------------------------------------------
# OTLP JSON builders
# ---------------------------------------------------------------------------


def _build_span(
    event: sqlite3.Row,
    trace_id: str,
    root_span_id: str,
) -> dict[str, Any]:
    """Build a single OTLP span dict from an agent_events row."""
    event_id: str = event["event_id"]
    span_id = _to_span_id(event_id)

    start_ns = _ts_to_unix_nano(
        event["timestamp"] or datetime.now(timezone.utc).isoformat()
    )
    # Spans with no explicit end: default to 100 ms duration
    end_ns = start_ns + 100_000_000

    tool_name: str = event["tool_name"] or "unknown"
    input_summary: str = event["input_summary"] or ""
    output_summary: str = event["output_summary"] or ""
    status: str = event["status"] or "completed"
    is_error: bool = status == "error"

    parent_event_id: str | None = (
        event["parent_event_id"] if "parent_event_id" in event.keys() else None
    )
    parent_span_id = _to_span_id(parent_event_id) if parent_event_id else root_span_id

    attributes = [
        {"key": "htmlgraph.tool_name", "value": {"stringValue": tool_name}},
        {
            "key": "htmlgraph.input_summary",
            "value": {"stringValue": input_summary[:256]},
        },
        {
            "key": "htmlgraph.output_summary",
            "value": {"stringValue": output_summary[:256]},
        },
        {"key": "htmlgraph.status", "value": {"stringValue": status}},
        {"key": "htmlgraph.event_id", "value": {"stringValue": event_id}},
    ]

    # Optional fields
    for col, attr_key in [
        ("agent_id", "htmlgraph.agent_id"),
        ("subagent_type", "htmlgraph.subagent_type"),
        ("model", "htmlgraph.model"),
        ("feature_id", "htmlgraph.feature_id"),
    ]:
        if col in event.keys() and event[col]:
            attributes.append(
                {"key": attr_key, "value": {"stringValue": str(event[col])}}
            )

    span: dict[str, Any] = {
        "traceId": trace_id,
        "spanId": span_id,
        "parentSpanId": parent_span_id,
        "name": tool_name,
        "kind": 1,  # SPAN_KIND_INTERNAL
        "startTimeUnixNano": str(start_ns),
        "endTimeUnixNano": str(end_ns),
        "attributes": attributes,
        "status": {
            "code": 2 if is_error else 1,  # STATUS_CODE_ERROR / OK
            "message": output_summary[:256] if is_error else "",
        },
    }
    return span


def _build_root_span(
    session: sqlite3.Row,
    trace_id: str,
) -> tuple[str, dict[str, Any]]:
    """Build a synthetic root span representing the session itself."""
    session_id: str = session["session_id"]
    root_span_id = _to_span_id(session_id + "-root")

    start_ns = _ts_to_unix_nano(
        session["started_at"] or datetime.now(timezone.utc).isoformat()
    )
    ended_at = session["ended_at"] if "ended_at" in session.keys() else None
    end_ns = (
        _ts_to_unix_nano(ended_at) if ended_at else int(time.time() * 1_000_000_000)
    )

    attributes = [
        {"key": "htmlgraph.session_id", "value": {"stringValue": session_id}},
        {
            "key": "htmlgraph.agent_id",
            "value": {"stringValue": session["agent_id"] or ""},
        },
        {
            "key": "htmlgraph.status",
            "value": {"stringValue": session["status"] or "active"},
        },
    ]

    root_span: dict[str, Any] = {
        "traceId": trace_id,
        "spanId": root_span_id,
        "name": f"session:{session_id}",
        "kind": 1,
        "startTimeUnixNano": str(start_ns),
        "endTimeUnixNano": str(end_ns),
        "attributes": attributes,
        "status": {"code": 1, "message": ""},
    }
    return root_span_id, root_span


def build_otlp_payload(
    sessions: list[sqlite3.Row],
    events_by_session: dict[str, list[sqlite3.Row]],
    service_name: str = "htmlgraph",
) -> dict[str, Any]:
    """
    Build a complete OTLP ExportTraceServiceRequest JSON payload.

    Args:
        sessions: List of session rows from SQLite.
        events_by_session: Mapping of session_id -> list of event rows.
        service_name: Service name attribute for the resource.

    Returns:
        Dict ready to be JSON-serialised and POST-ed to /v1/traces.
    """
    scope_spans_list = []

    for session in sessions:
        session_id: str = session["session_id"]
        trace_id = _to_trace_id(session_id)
        root_span_id, root_span = _build_root_span(session, trace_id)

        child_spans = [
            _build_span(ev, trace_id, root_span_id)
            for ev in events_by_session.get(session_id, [])
        ]

        all_spans = [root_span, *child_spans]

        scope_spans_list.append(
            {
                "resource": {
                    "attributes": [
                        {"key": "service.name", "value": {"stringValue": service_name}},
                        {
                            "key": "htmlgraph.session_id",
                            "value": {"stringValue": session_id},
                        },
                    ]
                },
                "scopeSpans": [
                    {
                        "scope": {"name": "htmlgraph", "version": "1.0"},
                        "spans": all_spans,
                    }
                ],
            }
        )

    return {"resourceSpans": scope_spans_list}


# ---------------------------------------------------------------------------
# Database queries
# ---------------------------------------------------------------------------


def _query_sessions(conn: sqlite3.Connection, limit: int = 100) -> list[sqlite3.Row]:
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    cursor.execute(
        """
        SELECT session_id, agent_id, status, started_at, ended_at, created_at
        FROM sessions
        ORDER BY created_at DESC
        LIMIT ?
        """,
        (limit,),
    )
    return cursor.fetchall()


def _query_events(
    conn: sqlite3.Connection, session_ids: list[str]
) -> dict[str, list[sqlite3.Row]]:
    if not session_ids:
        return {}
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    placeholders = ",".join("?" * len(session_ids))
    cursor.execute(
        f"""
        SELECT event_id, session_id, agent_id, tool_name, input_summary, output_summary,
               status, timestamp, parent_event_id, subagent_type, model, feature_id
        FROM agent_events
        WHERE session_id IN ({placeholders})
        ORDER BY timestamp ASC
        """,
        session_ids,
    )
    rows = cursor.fetchall()
    result: dict[str, list[sqlite3.Row]] = {}
    for row in rows:
        sid = row["session_id"]
        result.setdefault(sid, []).append(row)
    return result


# ---------------------------------------------------------------------------
# HTTP export
# ---------------------------------------------------------------------------


def export_to_otlp(
    endpoint: str = "http://localhost:4318",
    graph_dir: str = ".htmlgraph",
    session_limit: int = 100,
    service_name: str = "htmlgraph",
    dry_run: bool = False,
) -> int:
    """
    Export HtmlGraph sessions/events as OTLP traces to an OTLP HTTP collector.

    Args:
        endpoint: OTLP HTTP base URL (e.g. ``http://localhost:4318``).
                  The path ``/v1/traces`` is appended automatically.
        graph_dir: Path to ``.htmlgraph`` directory.
        session_limit: Maximum number of recent sessions to export.
        service_name: OTLP ``service.name`` resource attribute.
        dry_run: If True, print the payload instead of sending it.

    Returns:
        Number of sessions exported.
    """
    db_path = Path(graph_dir) / "htmlgraph.db"
    if not db_path.exists():
        raise FileNotFoundError(f"Database not found: {db_path}")

    conn = sqlite3.connect(str(db_path))
    conn.row_factory = sqlite3.Row
    try:
        sessions = _query_sessions(conn, limit=session_limit)
        session_ids = [s["session_id"] for s in sessions]
        events_by_session = _query_events(conn, session_ids)
    finally:
        conn.close()

    if not sessions:
        print("No sessions found to export.")
        return 0

    payload = build_otlp_payload(sessions, events_by_session, service_name=service_name)
    payload_json = json.dumps(payload).encode()

    if dry_run:
        print(json.dumps(payload, indent=2))
        print(
            f"\n[dry-run] Would POST {len(payload_json)} bytes to {endpoint}/v1/traces"
        )
        return len(sessions)

    url = endpoint.rstrip("/") + "/v1/traces"
    req = urllib.request.Request(
        url,
        data=payload_json,
        headers={
            "Content-Type": "application/json",
            "Content-Length": str(len(payload_json)),
        },
        method="POST",
    )

    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            status = resp.status
            body = resp.read().decode(errors="replace")
            logger.info("OTLP export response: %s %s", status, body[:200])
            print(f"Exported {len(sessions)} sessions to {url} (HTTP {status})")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode(errors="replace")
        raise RuntimeError(f"OTLP HTTP error {exc.code}: {body[:400]}") from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(
            f"Could not reach OTLP endpoint {url}: {exc.reason}"
        ) from exc

    return len(sessions)


__all__ = [
    "build_otlp_payload",
    "export_to_otlp",
    "ingest_cloud_event",
]

# Re-export for convenience
from htmlgraph.http_hook import ingest_cloud_event  # noqa: E402

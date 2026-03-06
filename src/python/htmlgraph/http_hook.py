"""
HTTP Hook Server for HtmlGraph.

Provides a simple HTTP endpoint that accepts CloudEvent JSON payloads via POST
and records them as HtmlGraph events in the SQLite database.

Start via CLI:
    htmlgraph serve-hooks [--port 8081] [--host 0.0.0.0]

Endpoints:
    POST /events  - Accept CloudEvent JSON, store as HtmlGraph event
    GET  /health  - Health check
"""

from __future__ import annotations

import json
import logging
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, HTTPServer
from typing import Any

from htmlgraph.db.schema import HtmlGraphDB
from htmlgraph.ids import generate_id

logger = logging.getLogger(__name__)


class CloudEventHandler(BaseHTTPRequestHandler):
    """HTTP request handler for CloudEvent ingestion."""

    # Injected by server factory
    db: HtmlGraphDB

    def log_message(self, format: str, *args: Any) -> None:  # noqa: A002
        """Suppress default request logging; use Python logging instead."""
        logger.debug(format % args)

    def do_GET(self) -> None:
        """Handle GET /health."""
        if self.path == "/health":
            self._send_json(200, {"status": "ok", "service": "htmlgraph-http-hook"})
        else:
            self._send_json(404, {"error": "not found"})

    def do_POST(self) -> None:
        """Handle POST /events - accept CloudEvent JSON."""
        if self.path != "/events":
            self._send_json(404, {"error": "not found"})
            return

        # Read body
        content_length = int(self.headers.get("Content-Length", 0))
        if content_length == 0:
            self._send_json(400, {"error": "empty body"})
            return

        try:
            raw = self.rfile.read(content_length)
            payload = json.loads(raw)
        except (json.JSONDecodeError, Exception) as exc:
            self._send_json(400, {"error": f"invalid JSON: {exc}"})
            return

        try:
            event_id = ingest_cloud_event(self.db, payload)
        except Exception as exc:
            logger.exception("Failed to ingest CloudEvent")
            self._send_json(500, {"error": str(exc)})
            return

        self._send_json(202, {"event_id": event_id, "accepted": True})

    def _send_json(self, status: int, body: dict[str, Any]) -> None:
        """Write a JSON response."""
        encoded = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)


def ingest_cloud_event(db: HtmlGraphDB, payload: dict[str, Any]) -> str:
    """
    Parse a CloudEvent payload and persist it as an agent_events row.

    Minimal CloudEvent v1.0 fields supported:
        id, source, type, time, data

    The ``data`` field may contain:
        session_id, tool_name, tool_input, tool_output, agent_id
    """
    data: dict[str, Any] = payload.get("data") or {}

    event_id = payload.get("id") or generate_id("event")
    session_id: str = data.get("session_id") or payload.get("source") or "http-hook"
    tool_name: str = data.get("tool_name") or payload.get("type") or "HttpEvent"
    agent_id: str = data.get("agent_id") or "http-hook"
    tool_input: dict[str, Any] = data.get("tool_input") or {}
    tool_output: dict[str, Any] = data.get("tool_output") or {}
    is_error: bool = bool(data.get("is_error", False))

    input_summary = data.get("input_summary") or _truncate(str(tool_input), 200)
    output_summary = data.get("output_summary") or _truncate(str(tool_output), 200)

    context: dict[str, Any] = {
        "source": "http-hook",
        "cloud_event_type": payload.get("type"),
        "cloud_event_source": payload.get("source"),
        "is_error": is_error,
    }

    # Ensure session exists
    _ensure_session(db, session_id, agent_id)

    db.insert_event(
        event_id=event_id,
        agent_id=agent_id,
        event_type="tool_call",
        session_id=session_id,
        tool_name=tool_name,
        input_summary=input_summary,
        tool_input=tool_input,
        output_summary=output_summary,
        context=context,
        parent_event_id=None,
        cost_tokens=0,
    )

    logger.info(
        "Ingested CloudEvent event_id=%s tool=%s session=%s",
        event_id,
        tool_name,
        session_id,
    )
    return event_id


def _ensure_session(db: HtmlGraphDB, session_id: str, agent_id: str) -> None:
    """Create session row if it does not already exist."""
    if not db.connection:
        return
    cursor = db.connection.cursor()
    cursor.execute(
        "SELECT session_id FROM sessions WHERE session_id = ? LIMIT 1", (session_id,)
    )
    if cursor.fetchone() is None:
        now = datetime.now(timezone.utc).isoformat()
        try:
            cursor.execute(
                """
                INSERT INTO sessions (session_id, agent_id, status, started_at, created_at, updated_at)
                VALUES (?, ?, 'active', ?, ?, ?)
                """,
                (session_id, agent_id, now, now, now),
            )
            db.connection.commit()
        except Exception:
            db.connection.rollback()


def _truncate(text: str, max_len: int) -> str:
    return text[:max_len] if len(text) > max_len else text


def make_handler_class(db: HtmlGraphDB) -> type[CloudEventHandler]:
    """Return a CloudEventHandler subclass with ``db`` bound as a class attribute."""

    class BoundHandler(CloudEventHandler):
        pass

    BoundHandler.db = db
    return BoundHandler


def run_http_hook_server(
    host: str = "0.0.0.0",
    port: int = 8081,
    graph_dir: str = ".htmlgraph",
) -> None:
    """
    Start the HTTP hook server.

    Args:
        host: Bind address (default 0.0.0.0).
        port: Port to listen on (default 8081).
        graph_dir: Path to .htmlgraph directory containing the SQLite database.
    """
    import os

    db_path = os.path.join(graph_dir, "htmlgraph.db")
    db = HtmlGraphDB(db_path=db_path)

    handler_cls = make_handler_class(db)
    server = HTTPServer((host, port), handler_cls)

    print(f"HtmlGraph HTTP Hook server listening on http://{host}:{port}")
    print(f"  POST http://{host}:{port}/events   - ingest CloudEvents")
    print(f"  GET  http://{host}:{port}/health   - health check")
    print("Press Ctrl+C to stop.")

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down HTTP hook server.")
    finally:
        db.disconnect()
        server.server_close()


__all__ = [
    "CloudEventHandler",
    "ingest_cloud_event",
    "run_http_hook_server",
]

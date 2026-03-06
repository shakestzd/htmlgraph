from __future__ import annotations

"""Session ingestion from external AI tool formats into HtmlGraph."""

from htmlgraph.ingest.claude_code import ClaudeCodeIngester, IngestResult
from htmlgraph.ingest.codex import CodexSession, ingest_codex_sessions
from htmlgraph.ingest.copilot import CopilotSession, ingest_copilot_sessions
from htmlgraph.ingest.cursor import ingest_cursor_sessions
from htmlgraph.ingest.gemini import ingest_gemini_sessions
from htmlgraph.ingest.opencode import ingest_opencode_sessions

__all__ = [
    "ClaudeCodeIngester",
    "CodexSession",
    "CopilotSession",
    "IngestResult",
    "ingest_codex_sessions",
    "ingest_copilot_sessions",
    "ingest_cursor_sessions",
    "ingest_gemini_sessions",
    "ingest_opencode_sessions",
]

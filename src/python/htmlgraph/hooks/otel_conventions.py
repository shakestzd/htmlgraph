"""
OTel GenAI Semantic Convention Span Names.

Maps HtmlGraph internal tool names to OpenTelemetry GenAI semantic convention
display names for observability tooling (Jaeger, Tempo, OTLP collectors, etc.).

The stored ``tool_name`` column in ``agent_events`` and ``tool_traces`` is NOT
modified — this module provides a computed *display* name only.

References:
    https://opentelemetry.io/docs/specs/semconv/gen-ai/
    https://opentelemetry.io/docs/specs/semconv/gen-ai/gen-ai-spans/
"""

from __future__ import annotations

# ---------------------------------------------------------------------------
# Mapping: HtmlGraph tool_name → OTel GenAI span name
# ---------------------------------------------------------------------------
# Convention used here:
#   - Orchestrator delegations   → "invoke_agent"
#   - Tool executions            → "execute_tool <tool_snake_case>"
#   - Session / conversation     → "user_turn" / "system_event"
# ---------------------------------------------------------------------------

OTEL_SPAN_NAMES: dict[str, str] = {
    # --- Orchestrator delegations -------------------------------------------
    "Agent": "invoke_agent",
    "Task": "invoke_agent",
    # --- File operations ----------------------------------------------------
    "Read": "execute_tool read",
    "Write": "execute_tool write",
    "Edit": "execute_tool edit",
    "MultiEdit": "execute_tool multi_edit",
    # --- Search / discovery -------------------------------------------------
    "Grep": "execute_tool grep",
    "Glob": "execute_tool glob",
    # --- Shell execution ----------------------------------------------------
    "Bash": "execute_tool bash",
    # --- Web tools ----------------------------------------------------------
    "WebSearch": "execute_tool web_search",
    "WebFetch": "execute_tool web_fetch",
    # --- Notebook tools -----------------------------------------------------
    "NotebookRead": "execute_tool notebook_read",
    "NotebookEdit": "execute_tool notebook_edit",
    # --- Session / conversation events --------------------------------------
    "UserQuery": "user_turn",
    "UserPromptSubmit": "user_turn",
    # --- Skill invocation ---------------------------------------------------
    "Skill": "invoke_skill",
    # --- Todo management ----------------------------------------------------
    "TodoWrite": "execute_tool todo_write",
    "TodoRead": "execute_tool todo_read",
    # --- Task management ----------------------------------------------------
    "TaskCreate": "execute_tool task_create",
    "TaskUpdate": "execute_tool task_update",
    "TaskList": "execute_tool task_list",
    "TaskGet": "execute_tool task_get",
    # --- User interaction ---------------------------------------------------
    "AskUserQuestion": "user_interaction",
}


def get_otel_span_name(tool_name: str) -> str:
    """Return the OTel GenAI span name for *tool_name*.

    For unknown tools the convention ``execute_tool <tool_snake_case>`` is
    applied automatically, which keeps all spans discoverable in tracing UIs
    even for custom or MCP tools.

    Args:
        tool_name: The raw tool name as stored in ``agent_events.tool_name``
                   (e.g. ``"Bash"``, ``"mcp__plugin_x__navigate"``).

    Returns:
        OTel-compatible span name string, e.g. ``"execute_tool bash"``.

    Examples:
        >>> get_otel_span_name("Bash")
        'execute_tool bash'
        >>> get_otel_span_name("Agent")
        'invoke_agent'
        >>> get_otel_span_name("UserQuery")
        'user_turn'
        >>> get_otel_span_name("mcp__plugin_x__navigate")
        'execute_tool mcp__plugin_x__navigate'
        >>> get_otel_span_name("UnknownTool")
        'execute_tool unknowntool'
    """
    if tool_name in OTEL_SPAN_NAMES:
        return OTEL_SPAN_NAMES[tool_name]

    # MCP tools follow the pattern ``mcp__<plugin>__<action>`` — keep the
    # full name in snake_case so they remain identifiable in tracing UIs.
    return f"execute_tool {tool_name.lower()}"

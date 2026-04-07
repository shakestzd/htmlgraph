# /// script
# requires-python = ">=3.9"
# dependencies = [
#     "marimo>=0.22.4",
#     "pyyaml>=6.0",
#     "anywidget>=0.9",
#     "traitlets>=5.0",
# ]
# ///

import marimo

__generated_with = "0.22.4"
app = marimo.App(width="medium", app_title="CRISPI Plan Archive")


@app.cell
def _():
    import marimo as mo
    from pathlib import Path
    import yaml
    import sqlite3
    from plan_ui import (
        stat_card, effort_badge, risk_badge,
        render_plan_header, render_slice_cards,
        render_questions, render_chat_history_bubbles,
    )
    from critique_renderer import render_critique
    from dagre_widget import DependencyGraphWidget
    return (
        DependencyGraphWidget, Path, effort_badge,
        mo, render_critique, render_plan_header, render_questions,
        render_slice_cards, render_chat_history_bubbles,
        risk_badge, sqlite3, stat_card, yaml,
    )


@app.cell
def _(Path, mo, yaml):
    import os as _os
    # Plan path is always provided via CLI arg or PLAN_YAML_PATH env var.
    _cli_plan = mo.cli_args().get("plan")
    _env_plan = _os.environ.get("PLAN_YAML_PATH", "")
    _selected = _cli_plan or _env_plan
    mo.stop(not _selected, mo.md("**No plan specified.** Pass `--plan /path/to/plan.yaml`."))
    plan_path = Path(_selected)
    mo.stop(not plan_path.exists(), mo.md(f"File not found: `{plan_path}`"))
    plan = yaml.safe_load(plan_path.read_text())
    plan_id = plan["meta"]["id"]
    return plan, plan_id, plan_path


@app.cell
def _(Path, plan_id, sqlite3):
    import os as _os
    # Load feedback from SQLite — same lookup as plan_notebook.
    _env_hg = _os.environ.get("HTMLGRAPH_DIR", "")
    if _env_hg and Path(_env_hg).exists():
        _hg = Path(_env_hg)
    else:
        _cwd = Path.cwd()
        _candidates = [
            _cwd / ".htmlgraph",
            _cwd.parent / ".htmlgraph",
            _cwd.parent.parent / ".htmlgraph",
        ]
        _hg = next((p for p in _candidates if p.exists()), None)

    saved_feedback = {}
    _db_path = _hg / "htmlgraph.db" if _hg else None
    if _db_path and _db_path.exists():
        _conn = sqlite3.connect(str(_db_path))
        _conn.row_factory = sqlite3.Row
        _rows = _conn.execute(
            "SELECT section, action, value, question_id FROM plan_feedback WHERE plan_id = ?",
            (plan_id,),
        ).fetchall()
        for _r in _rows:
            _key = f"{_r['section']}:{_r['action']}"
            if _r["question_id"]:
                _key += f":{_r['question_id']}"
            saved_feedback[_key] = _r["value"]
        _conn.close()
    return saved_feedback,


@app.cell
def _(DependencyGraphWidget, mo, plan, plan_id, plan_path, render_plan_header, stat_card):
    # --- Plan header + dependency graph (always visible, outside accordion) ---
    _meta = {**plan["meta"], "_slices_count": len(plan.get("slices", []))}
    _header = render_plan_header(_meta, mo, stat_card)

    _slices = plan.get("slices", [])
    _nodes = [
        {"id": s["id"], "num": s["num"], "name": s["title"],
         "status": "approved" if s.get("approved") else "todo",
         "deps": ",".join(str(d) for d in s.get("deps", []))}
        for s in _slices
    ]
    _graph = mo.ui.anywidget(DependencyGraphWidget(nodes=_nodes, approved_ids=[
        s["id"] for s in _slices if s.get("approved")
    ]))

    header_output = mo.vstack([
        _header,
        mo.md("### Dependency Graph"),
        _graph,
        mo.accordion({f"**ID:** `{plan_id}` | **SOURCE:** `{plan_path}`": mo.md(f"`{plan_path}`")}),
    ])
    return header_output,


@app.cell
def _(mo, plan, saved_feedback):
    # --- Section A — Design Discussion (static) ---
    _design = plan.get("design", {})
    _design_ok = saved_feedback.get("design:approve", "false").lower() == "true"
    _comment = saved_feedback.get("design:comment", _design.get("comment", ""))

    _sections = []
    if _design.get("problem"):
        _sections.append(mo.md(f"### Problem\n\n{_design['problem']}"))
    if _design.get("goals"):
        _goals = "\n".join(f"- {g}" for g in _design["goals"])
        _sections.append(mo.md(f"### Goals\n\n{_goals}"))
    if _design.get("constraints"):
        _constraints = "\n".join(f"- {c}" for c in _design["constraints"])
        _sections.append(mo.md(f"### Constraints\n\n{_constraints}"))
    if _design.get("content"):
        _sections.append(mo.md(_design["content"]))
    if not _sections:
        _sections.append(mo.md("_No design content._"))

    _approval_kind = "success" if _design_ok else "warn"
    _approval_text = "Design: Approved" if _design_ok else "Design: Pending"
    _approval_badge = mo.callout(mo.md(f"**{_approval_text}**"), kind=_approval_kind)

    _comment_section = []
    if _comment:
        _comment_section = [mo.callout(mo.md(f"**Reviewer comment:** {_comment}"), kind="info")]

    design_output = mo.vstack(
        _sections
        + [_approval_badge]
        + _comment_section
    )
    return design_output,


@app.cell
def _(effort_badge, mo, plan, render_slice_cards, risk_badge, saved_feedback):
    # --- Section B — Vertical Slices (static, no checkboxes) ---
    _slices = plan.get("slices", [])
    _cards = render_slice_cards(
        _slices, saved_feedback,
        effort_badge_fn=effort_badge,
        risk_badge_fn=risk_badge,
        mo=mo,
        slice_approvals=None,  # None = static mode: reads from saved_feedback
    )
    slices_output = mo.vstack([
        mo.md("Slice approvals (archived)"),
        mo.accordion(_cards, multiple=True),
    ])
    return slices_output,


@app.cell
def _(mo, plan, render_questions, saved_feedback):
    # --- Section C — Open Questions (static, no radio buttons) ---
    _questions = plan.get("questions", [])
    if _questions:
        questions_output = render_questions(
            _questions, saved_feedback, mo,
            question_inputs=None,  # None = static mode: reads from saved_feedback
        )
    else:
        questions_output = mo.md("_No questions defined._")
    return questions_output,


@app.cell
def _(plan, render_critique):
    # --- Section D — AI Critique (already handles None gracefully) ---
    critique_output = render_critique(plan.get("critique"))
    return critique_output,


@app.cell
def _(mo, plan, saved_feedback):
    # --- Amendments (static, no dropdowns) ---
    import json as _json

    # Load amendments from saved_feedback directly (section='amendment').
    _raw = {k: v for k, v in saved_feedback.items() if k.startswith("amendment:")}
    # Rebuild amendment list from plan_feedback keys: "amendment:<action>:<id>"
    _amendments = []
    for _key, _val in _raw.items():
        _parts = _key.split(":", 2)
        if len(_parts) == 3:
            _, _action, _aid = _parts
            try:
                _aval = _json.loads(_val)
            except (_json.JSONDecodeError, TypeError):
                _aval = {}
            _amendments.append({"id": _aid, "action": _action, "value": _aval})

    if not _amendments:
        amendments_output = mo.md("_No amendments recorded._")
    else:
        _pending = sum(1 for a in _amendments if a["action"] == "proposed")
        _accepted = sum(1 for a in _amendments if a["action"] == "accepted")
        _rejected = sum(1 for a in _amendments if a["action"] == "rejected")
        _status = (
            f"**{_pending}** pending | "
            f"**{_accepted}** accepted | "
            f"**{_rejected}** rejected"
        )
        _action_to_kind = {"accepted": "success", "rejected": "danger", "proposed": "neutral", "applied": "success"}
        _action_to_display = {"proposed": "Pending", "accepted": "Accepted", "rejected": "Rejected", "applied": "Applied"}
        _rows = []
        for _a in _amendments:
            _kind = _action_to_kind.get(_a["action"], "neutral")
            _display = _action_to_display.get(_a["action"], "Pending")
            _label = (
                f"Slice {_a['value'].get('slice_num', '?')}: "
                f"{_a['value'].get('operation', '?')} {_a['value'].get('field', '?')} "
                f"— {_a['value'].get('content', '')[:60]}"
            )
            _rows.append(mo.callout(mo.md(f"**{_display}** — {_label}"), kind=_kind))
        amendments_output = mo.vstack([
            mo.md(_status),
            *_rows,
        ])
    return amendments_output,


@app.cell
def _(mo, plan, saved_feedback, stat_card):
    # --- Section E — Feedback Summary (static, no finalize button) ---
    _slices = plan.get("slices", [])
    _questions = plan.get("questions", [])

    _design_ok = saved_feedback.get("design:approve", "false").lower() == "true"
    _approved_slices = sum(
        1 for s in _slices
        if saved_feedback.get(f"slice-{s['num']}:approve", "false").lower() == "true"
    )
    _answered_qs = sum(
        1 for q in _questions
        if saved_feedback.get(f"questions:answer:{q['id']}") or q.get("answer")
    )

    _total = 1 + len(_slices) + len(_questions)
    _approved = (1 if _design_ok else 0) + _approved_slices + _answered_qs
    _pct = round(_approved / _total * 100) if _total > 0 else 0
    _all_ok = _approved == _total and _total > 0
    _bar_color = "#16a34a" if _all_ok else "#3b82f6"

    _d_bg, _d_fg, _d_bd = (
        ("#dcfce7", "#166534", "#86efac") if _design_ok
        else ("#fef3c7", "#92400e", "#f59e0b")
    )
    _q_bg, _q_fg, _q_bd = (
        ("#dcfce7", "#166534", "#86efac") if _answered_qs == len(_questions)
        else ("#fef3c7", "#92400e", "#f59e0b")
    )

    _progress_bar = mo.Html(
        f'<div style="display:flex;justify-content:space-between;font-size:0.8rem;'
        f'margin-bottom:4px"><span><strong>Review Progress</strong></span>'
        f'<span>{_approved} of {_total} completed &middot; {_total - _approved} remaining'
        f'</span></div>'
        f'<div style="background:var(--marimo-monochrome-100,#e0e0e0);'
        f'border-radius:6px;height:14px;overflow:hidden">'
        f'<div style="background:{_bar_color};height:100%;width:{_pct}%;'
        f'border-radius:6px"></div></div>'
    )

    _status_callout = (
        mo.callout(mo.md("**Plan finalized**"), kind="success")
        if plan.get("meta", {}).get("status") == "finalized"
        else (
            mo.callout(mo.md("**Review complete** — all sections approved"), kind="success")
            if _all_ok
            else mo.callout(mo.md("Review in progress — not all sections approved"), kind="warn")
        )
    )

    # Static decisions table via HTML (not mo.md markdown — marimo renders markdown tables as dataframe UI).
    _decision_rows_html = "".join(
        f"<tr><td style='padding:8px 12px;border-bottom:1px solid #e5e7eb'>{q['text']}</td>"
        f"<td style='padding:8px 12px;border-bottom:1px solid #e5e7eb;font-weight:600'>"
        f"{saved_feedback.get('questions:answer:' + q['id']) or q.get('answer') or '<em>pending</em>'}</td></tr>"
        for q in _questions
    )
    _decisions_table = mo.Html(
        f"<div style='margin:8px 0'><strong>Decisions Made</strong>"
        f"<table style='width:100%;border-collapse:collapse;margin-top:8px'>"
        f"<thead><tr style='border-bottom:2px solid #d1d5db'>"
        f"<th style='text-align:left;padding:8px 12px'>Question</th>"
        f"<th style='text-align:left;padding:8px 12px'>Decision</th></tr></thead>"
        f"<tbody>{_decision_rows_html}</tbody></table></div>"
    ) if _questions else mo.md("_No questions._")

    feedback_output = mo.vstack([
        mo.hstack([
            stat_card("Slices", f"{_approved_slices}/{len(_slices)}", "#f0f4ff", "#1e3a5f", "#93c5fd"),
            stat_card("Design", "Approved" if _design_ok else "Pending", _d_bg, _d_fg, _d_bd),
            stat_card("Questions", f"{_answered_qs}/{len(_questions)}", _q_bg, _q_fg, _q_bd),
            stat_card(
                "Progress", f"{_pct}%",
                "#dcfce7" if _all_ok else "#f0f4ff",
                "#166534" if _all_ok else "#1e3a5f",
                "#86efac" if _all_ok else "#93c5fd",
            ),
        ], justify="space-between", gap=0.75),
        _progress_bar,
        _decisions_table,
        _status_callout,
    ])
    return feedback_output,


@app.cell
def _(mo, plan_id, render_chat_history_bubbles, saved_feedback, sqlite3):
    # --- Section F — Plan Discussion (static bubbles) ---
    _history = []
    _chat_raw = saved_feedback.get("chat:messages")
    if _chat_raw:
        import json as _json2
        try:
            _history = _json2.loads(_chat_raw)
        except (_json2.JSONDecodeError, TypeError):
            _history = []

    # Also try loading from ClaudeChatBackend's storage pattern if available.
    if not _history:
        import os as _os2
        from pathlib import Path as _Path
        _env_hg = _os2.environ.get("HTMLGRAPH_DIR", "")
        if _env_hg and _Path(_env_hg).exists():
            _hg2 = _Path(_env_hg)
        else:
            _cwd2 = _Path.cwd()
            _candidates2 = [
                _cwd2 / ".htmlgraph",
                _cwd2.parent / ".htmlgraph",
                _cwd2.parent.parent / ".htmlgraph",
            ]
            _hg2 = next((p for p in _candidates2 if p.exists()), None)
        if _hg2:
            _db2 = str(_hg2 / "htmlgraph.db")
            try:
                _conn2 = sqlite3.connect(_db2)
                _conn2.row_factory = sqlite3.Row
                _row = _conn2.execute(
                    "SELECT value FROM plan_feedback "
                    "WHERE plan_id = ? AND section = 'chat' AND action = 'messages' "
                    "ORDER BY updated_at DESC LIMIT 1",
                    (plan_id,),
                ).fetchone()
                _conn2.close()
                if _row:
                    import json as _json3
                    _history = _json3.loads(_row["value"])
            except Exception:
                pass

    if _history:
        _bubbles = render_chat_history_bubbles(_history, mo)
        chat_output = mo.vstack([
            mo.md(f"*{len(_history)} messages*"),
            *_bubbles,
        ])
    else:
        chat_output = mo.md("_No chat history recorded._")
    return chat_output,


@app.cell
def _(
    amendments_output,
    chat_output,
    critique_output,
    design_output,
    feedback_output,
    header_output,
    mo,
    questions_output,
    slices_output,
):
    # --- Final assembly: header always visible, sections in accordion ---
    mo.vstack([
        header_output,
        mo.accordion(
            {
                "## A. Design Discussion": design_output,
                "## B. Vertical Slices": slices_output,
                "## C. Open Questions": questions_output,
                "## D. AI Critique Results": critique_output,
                "## E. Amendments": amendments_output,
                "## F. Feedback Summary": feedback_output,
                "## G. Plan Discussion": chat_output,
            },
            multiple=True,
            lazy=True,
        ),
    ])
    return


if __name__ == "__main__":
    app.run()

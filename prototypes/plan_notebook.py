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
app = marimo.App(width="medium", app_title="CRISPI Plan")


@app.cell
def _():
    import marimo as mo
    from pathlib import Path
    import yaml
    import sqlite3
    from plan_ui import (
        stat_card, status_badge, priority_badge, effort_badge,
        risk_badge, render_feedback_summary, STATUS_COLORS,
    )
    from plan_persistence import persist_feedback, finalize_plan, persist_amendment, get_amendments, update_amendment_status
    from amendment_parser import parse_amendments
    from critique_renderer import render_critique
    from dagre_widget import DependencyGraphWidget
    from claude_chat import ClaudeChatBackend
    return (
        ClaudeChatBackend, DependencyGraphWidget, Path,
        STATUS_COLORS, effort_badge, finalize_plan, get_amendments, mo,
        parse_amendments, persist_amendment, persist_feedback,
        render_critique, render_feedback_summary, risk_badge, sqlite3,
        stat_card, status_badge, update_amendment_status, yaml,
    )


@app.cell
def _(Path, mo, yaml):
    import os as _os
    # Check env var first (set by `htmlgraph plan review` when running from embedded temp dir).
    _env_hg = _os.environ.get("HTMLGRAPH_DIR", "")
    if _env_hg and Path(_env_hg).exists():
        htmlgraph_dir = Path(_env_hg)
    else:
        _cwd = Path.cwd()
        _candidates = [
            _cwd / ".htmlgraph",
            _cwd.parent / ".htmlgraph",
            _cwd.parent.parent / ".htmlgraph",
        ]
        htmlgraph_dir = next((p for p in _candidates if p.exists()), None)

    # Scan for available YAML plans and build a dropdown.
    _plans = {}
    if htmlgraph_dir:
        _plans_dir = htmlgraph_dir / "plans"
        if _plans_dir.exists():
            for _f in sorted(_plans_dir.glob("*.yaml")):
                try:
                    _p = yaml.safe_load(_f.read_text())
                    _label = f"{_p['meta']['id']} — {_p['meta']['title']}"
                    _plans[_label] = str(_f)
                except Exception:
                    pass
    # Also include sample_plan.yaml if it exists locally.
    _sample = Path.cwd() / "sample_plan.yaml"
    if _sample.exists() and str(_sample) not in _plans.values():
        _plans["sample_plan.yaml — Sample"] = str(_sample)

    # Check env var override from `plan review` command.
    _env_path = mo.cli_args().get("plan") or _os.environ.get("PLAN_YAML_PATH", "")

    if len(_plans) > 0:
        plan_yaml_input = mo.ui.dropdown(options=_plans, value=None, label="Select Plan")
    else:
        plan_yaml_input = mo.ui.text(value=_env_path or str(_sample), label="Plan YAML path", full_width=True)
    # Hide selector in export mode (CLI arg provides the plan path).
    if not mo.cli_args().get("plan"):
        mo.output.replace(plan_yaml_input)
    return htmlgraph_dir, plan_yaml_input


@app.cell
def _(Path, htmlgraph_dir, mo, plan_yaml_input, sqlite3, yaml):
    # --- Load plan content from YAML + feedback from SQLite ---
    _cli_plan = mo.cli_args().get("plan")
    _selected = _cli_plan or plan_yaml_input.value or ""
    mo.stop(not _selected, mo.md("**Select a plan from the dropdown above.**"))
    _path = Path(_selected)
    mo.stop(not _path.exists(), mo.md(f"File not found: `{_path}`"))
    plan_yaml_text = _path.read_text()
    plan = yaml.safe_load(plan_yaml_text)
    plan_path = _path
    plan_id = plan["meta"]["id"]

    # Load existing feedback from SQLite (restores state across sessions).
    saved_feedback = {}
    _db_path = htmlgraph_dir / "htmlgraph.db" if htmlgraph_dir else None
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
    return plan, plan_id, plan_path, plan_yaml_text, saved_feedback


@app.cell
def _(mo, plan_yaml_text):
    editor = mo.ui.code_editor(value=plan_yaml_text, language="yaml", disabled=True)
    return (editor,)


@app.cell
def _(STATUS_COLORS, editor, mo, plan, plan_id, plan_yaml_input, stat_card):
    # --- Header ---
    _meta = plan["meta"]
    _slices = plan.get("slices", [])
    _status = _meta["status"].capitalize()
    _sb, _sf, _sc = STATUS_COLORS.get(_meta["status"], STATUS_COLORS["todo"])
    mo.vstack([
        mo.md(f"# Plan: {_meta['title']}"),
        mo.md(f"### {_meta.get('description', '')}"),
        mo.hstack([
            stat_card("Status", _status, _sb, _sf, _sc),
            stat_card("Slices", len(_slices), "#f0f4ff", "#1e3a5f", "#93c5fd"),
            stat_card("Created", _meta.get("created_at", ""), "#f0f4ff", "#1e3a5f", "#93c5fd"),
            stat_card("Version", f"v{_meta.get('version', 1)}", "#f5f3ff", "#4c1d95", "#a78bfa"),
        ], justify="space-between", gap=0.75),
        mo.accordion({f"**ID:** `{plan_id}` | **SOURCE:** `{plan_path}`": editor}),
    ])
    return


@app.cell
def _(DependencyGraphWidget, mo, plan):
    # --- Dependency Graph ---
    _slices = plan.get("slices", [])
    _nodes = [
        {"id": s["id"], "num": s["num"], "name": s["title"],
         "status": "approved" if s.get("approved") else "todo",
         "deps": ",".join(str(d) for d in s.get("deps", []))}
        for s in _slices
    ]
    graph_widget = mo.ui.anywidget(DependencyGraphWidget(nodes=_nodes, approved_ids=[]))
    mo.vstack([mo.md("### Dependency Graph"), graph_widget])
    return (graph_widget,)


@app.cell
def _(mo, plan, saved_feedback):
    # --- A. Design Discussion (structured subsections from YAML) ---
    _design = plan.get("design", {})
    _saved_design = saved_feedback.get("design:approve", "false").lower() == "true"
    _saved_comment = saved_feedback.get("design:comment", _design.get("comment", ""))
    design_approved = mo.ui.checkbox(label="Approve design", value=_saved_design)
    design_comment = mo.ui.text_area(
        placeholder="Comments on design...", full_width=True, value=_saved_comment
    )
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
        _sections.append(mo.md("_No design content yet._"))
    mo.vstack(
        [mo.md("## A. Design Discussion")]
        + _sections
        + [design_approved, mo.accordion({"Add Comments": design_comment})]
    )
    return design_approved, design_comment


@app.cell
def _(design_approved, design_comment, persist_feedback, plan_id):
    persist_feedback(plan_id, "design", "approve", design_approved.value)
    if design_comment.value:
        persist_feedback(plan_id, "design", "comment", design_comment.value)
    return


@app.cell
def _(effort_badge, mo, plan, risk_badge, saved_feedback):
    # --- B. Vertical Slices ---
    _slices = plan.get("slices", [])
    _num_to_title = {s["num"]: s["title"] for s in _slices}
    slice_approvals = mo.ui.dictionary({
        s["id"]: mo.ui.checkbox(
            label="Approve",
            value=saved_feedback.get(f"slice-{s['num']}:approve", "false").lower() == "true",
        )
        for s in _slices
    })
    _cards = {}
    for _s in _slices:
        _effort = effort_badge(_s["effort"]) if _s.get("effort") else None
        _risk = risk_badge(_s["risk"]) if _s.get("risk") else None
        _badges = mo.hstack([b for b in [_effort, _risk] if b], justify="start", gap=0.25)
        _top_row = mo.hstack([slice_approvals[_s["id"]], _badges], justify="space-between")
        _body = [_top_row]
        if _s.get("what"):
            _body.append(mo.md(f"**What:** {_s['what']}"))
        if _s.get("why"):
            _body.append(mo.md(f"**Why:** {_s['why']}"))
        if _s.get("files"):
            _body.append(mo.md(f"**Files:** {', '.join(f'`{f}`' for f in _s['files'])}"))
        if _s.get("done_when"):
            _body.append(mo.md("**Done when:**\n" + "\n".join(f"- {d}" for d in _s["done_when"])))
        if _s.get("deps"):
            _body.append(mo.md(
                f"**Depends on:** {', '.join(_num_to_title.get(d, f'#{d}') for d in _s['deps'])}"
            ))
        if _s.get("tests"):
            _body.append(mo.md(f"**Tests:**\n```\n{_s['tests'].strip()}\n```"))
        _cards[f"Slice {_s['num']}: {_s['title']}"] = mo.vstack(_body)
    mo.vstack([
        mo.md("## B. Vertical Slices\n\nApprove slices individually."),
        mo.accordion(_cards, multiple=True),
    ])
    return (slice_approvals,)


@app.cell
def _(graph_widget, persist_feedback, plan, plan_id, slice_approvals):
    _slices = plan.get("slices", [])
    _id_to_num = {s["id"]: s["num"] for s in _slices}
    _approved_ids = []
    for _fid, _val in slice_approvals.value.items():
        _num = _id_to_num.get(_fid)
        if _num is not None:
            persist_feedback(plan_id, f"slice-{_num}", "approve", _val)
        if _val:
            _approved_ids.append(_fid)
    graph_widget.approved_ids = _approved_ids
    return


@app.cell
def _(mo, plan, saved_feedback):
    # --- C. Open Questions ---
    _questions = plan.get("questions", [])

    def _restore_answer(q):
        _rec = q.get("recommended", "")
        _saved = saved_feedback.get(f"questions:answer:{q['id']}") or q.get("answer")
        _key = _saved or _rec
        if _key:
            for opt in q["options"]:
                if opt["key"] == _key:
                    lbl = opt["label"]
                    if _rec and opt["key"] == _rec:
                        lbl += " ⭐ recommended"
                    return lbl
        return None

    def _build_options(q):
        _rec = q.get("recommended", "")
        _opts = {}
        for opt in q["options"]:
            lbl = opt["label"]
            if _rec and opt["key"] == _rec:
                lbl += " ⭐ recommended"
            _opts[lbl] = opt["key"]
        return _opts

    question_inputs = mo.ui.dictionary({
        q["id"]: mo.ui.radio(options=_build_options(q), value=_restore_answer(q))
        for i, q in enumerate(_questions)
    })
    _parts = []
    for _i, _q in enumerate(_questions):
        _desc = _q.get("description", "")
        _parts.append(mo.md(f"**Q{_i + 1}. {_q['text']}**"))
        _parts.append(mo.md((f" `{_desc}`" if _desc else "")))
        _parts.append(question_inputs[_q["id"]])
        _parts.append(mo.md("---"))
    mo.vstack([mo.md("## C. Open Questions")] + _parts[:-1])
    return (question_inputs,)


@app.cell
def _(persist_feedback, plan_id, question_inputs):
    for _qid, _val in question_inputs.value.items():
        if _val is not None:
            persist_feedback(plan_id, "questions", "answer", _val, question_id=_qid)
    return


@app.cell
def _(plan, render_critique):
    # --- D. AI Critique ---
    render_critique(plan.get("critique"))
    return


@app.cell
def _(design_approved, mo, plan, question_inputs, render_feedback_summary, slice_approvals):
    # --- E. Feedback Summary + Finalize ---
    _slices = plan.get("slices", [])
    _questions = plan.get("questions", [])
    _approved_slices = sum(1 for v in slice_approvals.value.values() if v)
    _answers = question_inputs.value
    _answered_qs = sum(1 for v in _answers.values() if v is not None)
    _summary, finalize_btn = render_feedback_summary(
        plan, design_approved.value, _approved_slices, len(_slices),
        _answered_qs, len(_questions), _answers, _questions,
    )
    mo.output.replace(_summary)
    return (finalize_btn,)


@app.cell
def _(finalize_btn, finalize_plan, mo, plan, plan_path, question_inputs, slice_approvals, yaml):
    # --- Finalize → update YAML status + export summary ---
    mo.stop(not finalize_btn.value)
    _result = finalize_plan(plan, plan_path, slice_approvals.value, question_inputs.value, yaml)
    mo.callout(mo.md(_result), kind="success")
    return


@app.cell
def _(get_amendments, mo, plan_id, update_amendment_status):
    # --- E2. Amendments from Chat ---
    _amendments = get_amendments(plan_id)
    if not _amendments:
        mo.output.replace(mo.md(""))
    else:
        _pending = [a for a in _amendments if a["action"] == "proposed"]
        _accepted = [a for a in _amendments if a["action"] == "accepted"]
        _rejected = [a for a in _amendments if a["action"] == "rejected"]

        _status = (
            f"**{len(_pending)}** pending | "
            f"**{len(_accepted)}** accepted | "
            f"**{len(_rejected)}** rejected"
        )

        _action_to_label = {"proposed": "Pending", "accepted": "Accept", "rejected": "Reject"}
        amendment_decisions = mo.ui.dictionary({
            a["id"]: mo.ui.dropdown(
                options={"Pending": "proposed", "Accept": "accepted", "Reject": "rejected"},
                value=_action_to_label.get(a["action"], "Pending"),
                label=f"Slice {a['value'].get('slice_num', '?')}: "
                      f"{a['value'].get('operation', '?')} {a['value'].get('field', '?')} "
                      f"— {a['value'].get('content', '')[:60]}",
            )
            for a in _amendments
        })

        mo.vstack([
            mo.md(f"## Amendments\n\n{_status}"),
            amendment_decisions,
        ])
    return


@app.cell
def _(get_amendments, plan_id, update_amendment_status):
    # --- Persist amendment decisions ---
    _amendments = get_amendments(plan_id)
    _by_id = {a["id"]: a for a in _amendments}
    _mod = __import__("sys").modules.get(__name__)
    _decisions = getattr(_mod, "amendment_decisions", None) if _mod else None
    if _decisions is not None:
        for _aid, _new_action in _decisions.value.items():
            _current = _by_id.get(_aid, {}).get("action")
            if _current and _new_action != _current:
                update_amendment_status(plan_id, _aid, _new_action)
    return


@app.cell
def _(ClaudeChatBackend, htmlgraph_dir, mo, parse_amendments, persist_amendment, plan_id, plan_yaml_text):
    # --- F. Plan Discussion (sidebar chat or static transcript) ---
    import os as _os
    _is_export = _os.environ.get("PLAN_EXPORT_MODE") == "1"

    def _render_history_bubbles(history):
        """Render chat messages as styled bubbles."""
        _bubbles = []
        for _m in history:
            _role = _m.get("role", "user")
            _text = _m.get("content", "")
            _preview = _text[:500] + ("\n\n..." if len(_text) > 500 else "")
            if _role == "user":
                _esc = _preview.replace("<", "&lt;").replace(">", "&gt;")
                _bubbles.append(mo.Html(
                    f'<div style="margin:6px 0;padding:8px 12px;background:#3b82f6;'
                    f'color:#fff;border-radius:12px 12px 4px 12px;font-size:13px;'
                    f'line-height:1.4;margin-left:20%">{_esc}</div>'
                ))
            else:
                _bubbles.append(mo.callout(mo.md(_preview), kind="neutral"))
        return _bubbles

    _db = str(htmlgraph_dir / "htmlgraph.db") if htmlgraph_dir else None
    _project_dir = str(htmlgraph_dir.parent) if htmlgraph_dir else None
    _backend = ClaudeChatBackend(plan_context=plan_yaml_text, db_path=_db, plan_id=plan_id, project_dir=_project_dir)
    _history = _backend.load_messages()

    if _is_export:
        # Static HTML export: render chat history as a read-only transcript.
        if _history:
            _bubbles = _render_history_bubbles(_history)
            mo.sidebar([
                mo.md(f"## Plan Discussion\n\n*{len(_history)} messages*"),
                *_bubbles,
            ], width="360px")
        mo.stop(_is_export)  # Skip interactive chat widget in export mode.

    # Interactive mode: full chat widget.
    _available, _avail_msg = ClaudeChatBackend.is_available()
    _has_fallback = ClaudeChatBackend.has_api_fallback()

    _items = []
    if not _available and not _has_fallback:
        _items.append(mo.callout(mo.md(
            "**AI Chat unavailable.** Install [Claude Code](https://claude.ai/download) "
            "and ensure `claude` is on PATH, or set `ANTHROPIC_API_KEY`."), kind="warn"))
    else:
        if _history:
            _count = len(_history)
            _items.append(mo.accordion({
                f"Prior conversation ({_count} messages)": mo.vstack(
                    _render_history_bubbles(_history)),
            }))

        def _chat_model(messages, config):
            """Streaming model: yield text deltas, persist + extract amendments."""
            _user_msg = messages[-1].content if messages else ""
            _full = ""
            for chunk in _backend.send(_user_msg):
                _full += chunk
                yield chunk
            _all = [{"role": getattr(m, "role", "user"),
                     "content": str(getattr(m, "content", ""))}
                    for m in messages]
            _all.append({"role": "assistant", "content": _full})
            _backend.save_messages(_all)
            try:
                for _a in parse_amendments(_full):
                    _aid = f"amend-{hash((_a['slice_num'], _a['field'], _a['content'])) & 0xFFFFFF:06x}"
                    persist_amendment(plan_id, _a, _aid)
            except Exception:
                pass

        if not _available and _has_fallback:
            _items.append(mo.callout(mo.md(
                "Using **Anthropic API** fallback (claude CLI not found)."), kind="info"))
        _items.append(mo.ui.chat(
            _chat_model,
            prompts=["What are the main risks?", "Summarize the design decisions"],
        ))

    mo.sidebar(_items, width="360px")


if __name__ == "__main__":
    app.run()

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
        stat_card,
        status_badge,
        priority_badge,
        effort_badge,
        risk_badge,
        STATUS_COLORS,
    )
    from critique_renderer import render_critique
    from dagre_widget import DependencyGraphWidget

    return (
        DependencyGraphWidget,
        Path,
        STATUS_COLORS,
        effort_badge,
        mo,
        render_critique,
        risk_badge,
        sqlite3,
        stat_card,
        status_badge,
        yaml,
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


@app.function
# --- Persistence helper: write one feedback row to SQLite ---
def persist_feedback(plan_id, section, action, value, question_id=""):
    """Write a single feedback entry to plan_feedback table."""
    import sqlite3 as _sql
    from pathlib import Path as _P

    _cwd = _P.cwd()
    _candidates = [
        _cwd / ".htmlgraph",
        _cwd.parent / ".htmlgraph",
        _cwd.parent.parent / ".htmlgraph",
    ]
    _hg = next((p for p in _candidates if p.exists()), None)
    if not _hg:
        return
    _db = _hg / "htmlgraph.db"
    if not _db.exists():
        return
    _conn = _sql.connect(str(_db))
    _conn.execute(
        """INSERT OR REPLACE INTO plan_feedback (plan_id, section, action, value, question_id, updated_at)
           VALUES (?, ?, ?, ?, ?, datetime('now'))""",
        (plan_id, section, action, str(value), question_id),
    )
    _conn.commit()
    _conn.close()


@app.cell
def _(mo, plan_yaml_text):
    editor = mo.ui.code_editor(
        value=plan_yaml_text,
        language="yaml",
        disabled=True,
    )
    return (editor,)


@app.cell
def _(STATUS_COLORS, editor, mo, plan, plan_id, plan_yaml_input, stat_card):
    # --- Header ---
    _meta = plan["meta"]
    _slices = plan.get("slices", [])
    _status = _meta["status"].capitalize()
    _sb, _sf, _sc = STATUS_COLORS.get(_meta["status"], STATUS_COLORS["todo"])
    mo.vstack(
        [
            mo.md(f"# Plan: {_meta['title']}"),
            mo.md(f"### {_meta.get('description', '')}"),
            mo.hstack(
                [
                    stat_card("Status", _status, _sb, _sf, _sc),
                    stat_card(
                        "Slices", len(_slices), "#f0f4ff", "#1e3a5f", "#93c5fd"
                    ),
                    stat_card(
                        "Created",
                        _meta.get("created_at", ""),
                        "#f0f4ff",
                        "#1e3a5f",
                        "#93c5fd",
                    ),
                ],
                justify="space-between",
                gap=0.75,
            ),
            mo.accordion(
                {
                    f"**ID:** `{plan_id}` | **SOURCE:** `{plan_path}`": editor,
                }
            ),
        ]
    )
    return


@app.cell
def _(DependencyGraphWidget, mo, plan):
    # --- Dependency Graph ---
    _slices = plan.get("slices", [])
    _nodes = [
        {
            "id": s["id"],
            "num": s["num"],
            "name": s["title"],
            "status": "approved" if s.get("approved") else "todo",
            "deps": ",".join(str(d) for d in s.get("deps", [])),
        }
        for s in _slices
    ]
    graph_widget = mo.ui.anywidget(
        DependencyGraphWidget(nodes=_nodes, approved_ids=[])
    )
    mo.vstack([mo.md("### Dependency Graph"), graph_widget])
    return (graph_widget,)


@app.cell
def _(mo, plan, saved_feedback):
    # --- A. Design Discussion (structured subsections from YAML) ---
    _design = plan.get("design", {})
    _saved_design = saved_feedback.get("design:approve", "false").lower() == "true"
    _saved_comment = saved_feedback.get(
        "design:comment", _design.get("comment", "")
    )
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
def _(design_approved, design_comment, plan_id):
    # Persist design approval on every change.
    persist_feedback(plan_id, "design", "approve", design_approved.value)
    if design_comment.value:
        persist_feedback(plan_id, "design", "comment", design_comment.value)
    return


@app.cell
def _(effort_badge, mo, plan, risk_badge, saved_feedback):
    # --- B. Vertical Slices ---
    _slices = plan.get("slices", [])
    _num_to_title = {s["num"]: s["title"] for s in _slices}
    slice_approvals = mo.ui.dictionary(
        {
            s["id"]: mo.ui.checkbox(
                label="Approve",
                value=saved_feedback.get(f"slice-{s['num']}:approve", "false").lower()
                == "true",
            )
            for s in _slices
        }
    )
    _cards = {}
    for _s in _slices:
        _effort = effort_badge(_s["effort"]) if _s.get("effort") else None
        _risk = risk_badge(_s["risk"]) if _s.get("risk") else None
        _badges = mo.hstack(
            [b for b in [_effort, _risk] if b], justify="start", gap=0.25
        )
        _top_row = mo.hstack(
            [slice_approvals[_s["id"]], _badges], justify="space-between"
        )
        _body = [_top_row]
        if _s.get("what"):
            _body.append(mo.md(f"**What:** {_s['what']}"))
        if _s.get("why"):
            _body.append(mo.md(f"**Why:** {_s['why']}"))
        if _s.get("files"):
            _body.append(
                mo.md(f"**Files:** {', '.join(f'`{f}`' for f in _s['files'])}")
            )
        if _s.get("done_when"):
            _body.append(
                mo.md(
                    "**Done when:**\n"
                    + "\n".join(f"- {d}" for d in _s["done_when"])
                )
            )
        if _s.get("deps"):
            _body.append(
                mo.md(
                    f"**Depends on:** {', '.join(_num_to_title.get(d, f'#{d}') for d in _s['deps'])}"
                )
            )
        if _s.get("tests"):
            _body.append(mo.md(f"**Tests:**\n```\n{_s['tests'].strip()}\n```"))
        _label = f"Slice {_s['num']}: {_s['title']}"
        _cards[_label] = mo.vstack(_body)
    mo.vstack(
        [
            mo.md("## B. Vertical Slices\n\nApprove slices individually."),
            mo.accordion(_cards, multiple=True),
        ],
    )
    return (slice_approvals,)


@app.cell
def _(graph_widget, plan, plan_id, slice_approvals):
    # Persist slice approvals + sync graph widget.
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


    # Restore saved answers from SQLite — must match _build_options labels.
    def _restore_answer(q):
        _rec = q.get("recommended", "")
        _saved = saved_feedback.get(f"questions:answer:{q['id']}") or q.get("answer")
        _key_to_find = _saved or _rec  # saved > yaml answer > recommended
        if _key_to_find:
            for opt in q["options"]:
                if opt["key"] == _key_to_find:
                    _lbl = opt["label"]
                    if _rec and opt["key"] == _rec:
                        _lbl += " ⭐ recommended"
                    return _lbl
        return None


    # Mark recommended option in label.
    def _build_options(q):
        _rec = q.get("recommended", "")
        _opts = {}
        for opt in q["options"]:
            _lbl = opt["label"]
            if _rec and opt["key"] == _rec:
                _lbl += " ⭐ recommended"
            _opts[_lbl] = opt["key"]
        return _opts


    question_inputs = mo.ui.dictionary(
        {
            q["id"]: mo.ui.radio(
                options=_build_options(q), value=_restore_answer(q)
            )
            for i, q in enumerate(_questions)
        }
    )
    _parts = []
    for _i, _q in enumerate(_questions):
        _desc = _q.get("description", "")
        _heading = f"**Q{_i + 1}. {_q['text']}**"
        _parts.append(mo.md(_heading))
        _parts.append(mo.md((f" `{_desc}`" if _desc else "")))
        _parts.append(question_inputs[_q["id"]])
        _parts.append(mo.md("---"))
    mo.vstack([mo.md("## C. Open Questions")] + _parts[:-1])
    return (question_inputs,)


@app.cell
def _(plan_id, question_inputs):
    # Persist question answers on every change.
    for _qid, _val in question_inputs.value.items():
        if _val is not None:
            persist_feedback(
                plan_id, "questions", "answer", _val, question_id=_qid
            )
    return


@app.cell
def _(plan, render_critique):
    # --- D. AI Critique ---
    render_critique(plan.get("critique"))
    return


@app.cell
def _(
    design_approved,
    mo,
    plan,
    question_inputs,
    slice_approvals,
    stat_card,
    status_badge,
):
    # --- E. Feedback Summary + Finalize ---
    _slices = plan.get("slices", [])
    _questions = plan.get("questions", [])
    _approved_slices = sum(1 for v in slice_approvals.value.values() if v)
    _total_slices = len(_slices)
    _design_ok = design_approved.value
    _answers = question_inputs.value
    _answered_qs = sum(1 for v in _answers.values() if v is not None)
    _total_qs = len(_questions)
    _total = 1 + _total_slices + _total_qs
    _approved = (1 if _design_ok else 0) + _approved_slices + _answered_qs
    _pct = round(_approved / _total * 100) if _total > 0 else 0
    _all_ok = _approved == _total and _total > 0
    _remaining = _total - _approved
    _bar_color = "#16a34a" if _all_ok else "#3b82f6"

    _progress_bar = mo.Html(
        f'<div style="display:flex;justify-content:space-between;font-size:0.8rem;margin-bottom:4px">'
        f"<span><strong>Review Progress</strong></span>"
        f"<span>{_approved} of {_total} completed &middot; {_remaining} remaining</span></div>"
        f'<div style="background:var(--marimo-monochrome-100,#e0e0e0);border-radius:6px;height:14px;overflow:hidden">'
        f'<div style="background:{_bar_color};height:100%;width:{_pct}%;border-radius:6px;transition:width .3s"></div></div>'
    )
    _d_bg, _d_fg, _d_bd = (
        ("#dcfce7", "#166534", "#86efac")
        if _design_ok
        else ("#fef3c7", "#92400e", "#f59e0b")
    )
    _q_bg, _q_fg, _q_bd = (
        ("#dcfce7", "#166534", "#86efac")
        if _answered_qs == _total_qs
        else ("#fef3c7", "#92400e", "#f59e0b")
    )

    finalize_btn = mo.ui.run_button(label="Finalize Plan")


    def _decision_display(answer):
        return mo.md(f"**{answer}**") if answer else status_badge("unanswered")


    _decisions_table = mo.ui.table(
        [
            {
                "Question": q["text"],
                "Decision": _decision_display(_answers.get(q["id"])),
            }
            for q in _questions
        ],
        selection=None,
        label="Decisions Made",
    )

    mo.vstack(
        [
            mo.md("## E. Feedback Summary"),
            mo.hstack(
                [
                    stat_card(
                        "Slices",
                        f"{_approved_slices}/{_total_slices}",
                        "#f0f4ff",
                        "#1e3a5f",
                        "#93c5fd",
                    ),
                    stat_card(
                        "Design",
                        "Approved" if _design_ok else "Pending",
                        _d_bg,
                        _d_fg,
                        _d_bd,
                    ),
                    stat_card(
                        "Questions",
                        f"{_answered_qs}/{_total_qs}",
                        _q_bg,
                        _q_fg,
                        _q_bd,
                    ),
                    stat_card(
                        "Progress",
                        f"{_pct}%",
                        "#dcfce7" if _all_ok else "#f0f4ff",
                        "#166534" if _all_ok else "#1e3a5f",
                        "#86efac" if _all_ok else "#93c5fd",
                    ),
                ],
                justify="space-between",
                gap=0.75,
            ),
            _progress_bar,
            _decisions_table,
            (mo.callout(mo.md("**Plan finalized** — exported as static HTML"), kind="success")
            if plan.get("meta", {}).get("status") == "finalized" else
            (finalize_btn if _all_ok else mo.callout(
                mo.md("All sections must be approved before finalizing."), kind="warn"))),
        ]
    )
    return (finalize_btn,)


@app.cell
def _(
    finalize_btn,
    mo,
    plan,
    plan_path,
    question_inputs,
    slice_approvals,
    yaml,
):
    # --- Finalize → update YAML status + export summary ---
    mo.stop(not finalize_btn.value)
    _plan = dict(plan)
    _plan["meta"]["status"] = "finalized"
    _plan["design"]["approved"] = True
    for _s in _plan.get("slices", []):
        _s["approved"] = slice_approvals.value.get(_s["id"], False)
    for _q in _plan.get("questions", []):
        _q["answer"] = question_inputs.value.get(_q["id"])
    plan_path.write_text(
        yaml.dump(
            _plan, sort_keys=False, allow_unicode=True, default_flow_style=False
        )
    )

    _approved = [s for s in _plan["slices"] if s["approved"]]
    _feature_lines = "\n".join(f"  - `{s['id']}` {s['title']}" for s in _approved)
    _decision_lines = "\n".join(
        f"- {q['text']}: **{q.get('answer', 'pending')}**"
        for q in _plan.get("questions", [])
    )
    # Export static HTML archive via marimo export with CLI args.
    import subprocess as _sp
    _plan_id = _plan["meta"]["id"]
    _export_path = plan_path.parent / f"{_plan_id}.html"
    _notebook_dir = Path.cwd()
    _notebook_file = _notebook_dir / "plan_notebook.py"
    if not _notebook_file.exists():
        _notebook_file = plan_path.parent.parent / "prototypes" / "plan_notebook.py"
    _export_result = ""
    try:
        _r = _sp.run(
            ["marimo", "export", "html", str(_notebook_file),
             "-o", str(_export_path), "--no-include-code",
             "--", "--plan", str(plan_path)],
            capture_output=True, text=True, timeout=120,
            cwd=str(_notebook_file.parent),
        )
        if _export_path.exists() and _export_path.stat().st_size > 1000:
            _export_result = f"\n\n**Exported:** `{_export_path}` ({_export_path.stat().st_size // 1024}KB)"
        else:
            _export_result = f"\n\n**Export warning:** file created but may be incomplete"
    except Exception as _e:
        _export_result = f"\n\n**Export skipped:** {_e}"

    mo.callout(
        mo.md(
            f"## Plan Finalized\n\n**{_plan['meta']['title']}** — {len(_approved)} slices approved.\n\n"
            f"**Features:**\n{_feature_lines}\n\n**Decisions:**\n{_decision_lines}\n\n"
            f"Saved to `{plan_path}` with status: **finalized**"
            f"{_export_result}\n\n"
            f"> Next: run `/htmlgraph:execute` to dispatch approved slices."
        ),
        kind="success",
    )
    return


if __name__ == "__main__":
    app.run()

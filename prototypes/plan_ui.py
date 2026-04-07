"""Shared UI helpers for CRISPI plan notebooks."""

import marimo as mo


def stat_card(label, value, bg, fg, border):
    """Render a colored stat card."""
    return mo.Html(
        f'<div style="background:{bg};border:2px solid {border};border-radius:10px;'
        f'padding:14px 20px;min-width:120px;text-align:center">'
        f'<div style="font-size:0.75rem;font-weight:600;color:{fg};opacity:0.7;'
        f'text-transform:uppercase;letter-spacing:0.05em;margin-bottom:4px">{label}</div>'
        f'<div style="font-size:1.5rem;font-weight:800;color:{fg}">{value}</div></div>'
    )


STATUS_COLORS = {
    "todo": ("#fef3c7", "#92400e", "#f59e0b"),
    "in-progress": ("#dbeafe", "#1e40af", "#3b82f6"),
    "done": ("#dcfce7", "#166534", "#22c55e"),
    "blocked": ("#fee2e2", "#991b1b", "#ef4444"),
}


def status_badge(s):
    """Render a colored status pill badge."""
    colors = {
        "done": ("rgba(22,163,74,0.15)", "#16a34a"),
        "in-progress": ("rgba(245,158,11,0.15)", "#f59e0b"),
        "todo": ("rgba(107,114,128,0.15)", "#6b7280"),
        "blocked": ("rgba(220,38,38,0.15)", "#dc2626"),
    }
    bg, fg = colors.get(s, colors["todo"])
    return mo.Html(
        f'<span style="background:{bg};color:{fg};padding:2px 10px;'
        f'border-radius:9999px;font-size:0.75rem;font-weight:600">{s}</span>'
    )


def priority_badge(p):
    """Render a colored priority pill badge."""
    colors = {
        "critical": ("rgba(220,38,38,0.15)", "#dc2626"),
        "high": ("rgba(245,158,11,0.15)", "#f59e0b"),
        "medium": ("rgba(107,114,128,0.15)", "#6b7280"),
        "low": ("rgba(139,92,246,0.15)", "#8b5cf6"),
    }
    bg, fg = colors.get(p, colors["medium"])
    return mo.Html(
        f'<span style="background:{bg};color:{fg};padding:2px 10px;'
        f'border-radius:9999px;font-size:0.75rem;font-weight:600">{p}</span>'
    )


def id_badge(fid):
    """Render a monospace ID badge."""
    return mo.Html(f'<code style="font-size:0.8rem">{fid}</code>')


def effort_badge(effort):
    """Render an effort badge with size-based colors. S=green, M=amber, L=red."""
    colors = {
        "S": ("rgba(22,163,74,0.15)", "#16a34a"),
        "M": ("rgba(245,158,11,0.15)", "#f59e0b"),
        "L": ("rgba(220,38,38,0.15)", "#dc2626"),
    }
    bg, fg = colors.get(effort, ("rgba(107,114,128,0.15)", "#6b7280"))
    return mo.Html(
        f'<span style="background:{bg};color:{fg};padding:2px 10px;'
        f'border-radius:9999px;font-size:0.75rem;font-weight:600">Effort: {effort}</span>'
    )


def risk_badge(risk):
    """Render a risk badge with severity-based colors. Low=green, Med=amber, High=red."""
    colors = {
        "Low": ("rgba(22,163,74,0.15)", "#16a34a"),
        "Med": ("rgba(245,158,11,0.15)", "#f59e0b"),
        "Medium": ("rgba(245,158,11,0.15)", "#f59e0b"),
        "High": ("rgba(220,38,38,0.15)", "#dc2626"),
    }
    bg, fg = colors.get(risk, ("rgba(107,114,128,0.15)", "#6b7280"))
    return mo.Html(
        f'<span style="background:{bg};color:{fg};padding:2px 10px;'
        f'border-radius:9999px;font-size:0.75rem;font-weight:600">Risk: {risk}</span>'
    )


def render_plan_header(meta, mo, stat_card_fn):
    """Render the plan header: title, description, stat cards, and ID/SOURCE accordion."""
    _slices_count = meta.get("_slices_count", 0)
    _status = meta["status"].capitalize()
    _sb, _sf, _sc = STATUS_COLORS.get(meta["status"], STATUS_COLORS["todo"])
    return mo.vstack([
        mo.md(f"# Plan: {meta['title']}"),
        mo.md(f"### {meta.get('description', '')}"),
        mo.hstack([
            stat_card_fn("Status", _status, _sb, _sf, _sc),
            stat_card_fn("Slices", _slices_count, "#f0f4ff", "#1e3a5f", "#93c5fd"),
            stat_card_fn("Created", meta.get("created_at", ""), "#f0f4ff", "#1e3a5f", "#93c5fd"),
            stat_card_fn("Version", f"v{meta.get('version', 1)}", "#f5f3ff", "#4c1d95", "#a78bfa"),
        ], justify="space-between", gap=0.75),
    ])


def render_slice_cards(slices, saved_feedback, effort_badge_fn, risk_badge_fn, mo,
                       slice_approvals=None):
    """Render accordion cards for all slices. Returns a dict of accordion items.

    slice_approvals: a mo.ui.dictionary of checkboxes (interactive mode) or None.
    When None, approval state is read from saved_feedback for static display.
    """
    _num_to_title = {s["num"]: s["title"] for s in slices}
    _cards = {}
    for _s in slices:
        _effort = effort_badge_fn(_s["effort"]) if _s.get("effort") else None
        _risk = risk_badge_fn(_s["risk"]) if _s.get("risk") else None
        _badges = mo.hstack([b for b in [_effort, _risk] if b], justify="start", gap=0.25)

        if slice_approvals is not None:
            # Interactive mode: checkbox widget
            _top_row = mo.hstack([slice_approvals[_s["id"]], _badges], justify="space-between")
        else:
            # Static mode: read from saved_feedback
            _approved_val = saved_feedback.get(f"slice-{_s['num']}:approve", "false").lower() == "true"
            _a_bg, _a_fg = ("#dcfce7", "#166534") if _approved_val else ("#f3f4f6", "#6b7280")
            _a_label = "✓ Approved" if _approved_val else "○ Not approved"
            _approval_badge = mo.Html(
                f'<span style="display:inline-block;padding:2px 10px;border-radius:12px;'
                f'font-size:0.8rem;font-weight:500;background:{_a_bg};color:{_a_fg}">'
                f'{_a_label}</span>'
            )
            _top_row = mo.hstack([_approval_badge, _badges], justify="space-between")

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
        _cards[f"### Slice {_s['num']}: {_s['title']}"] = mo.vstack(_body)
    return _cards


def render_questions(questions, saved_feedback, mo, question_inputs=None):
    """Render section C: Open Questions.

    question_inputs: a mo.ui.dictionary of radio widgets (interactive mode) or None.
    When None, answers are read from saved_feedback for static display.
    Returns rendered vstack output.
    """
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

    _parts = []
    for _i, _q in enumerate(questions):
        _desc = _q.get("description", "")
        _parts.append(mo.md(f"**Q{_i + 1}. {_q['text']}**"))
        _parts.append(mo.md((f" `{_desc}`" if _desc else "")))
        if question_inputs is not None:
            # Interactive mode: radio widget
            _parts.append(question_inputs[_q["id"]])
        else:
            # Static mode: read answer from saved_feedback
            _answer = _restore_answer(_q)
            _parts.append(mo.callout(
                mo.md(f"Answer: {_answer}" if _answer else "_No answer recorded._"),
                kind="neutral",
            ))
        _parts.append(mo.md("---"))
    return mo.vstack([mo.md("## C. Open Questions")] + _parts[:-1])


def render_chat_history_bubbles(history, mo):
    """Render chat messages as styled bubbles.

    User messages appear in blue on the right; assistant messages as neutral callouts.
    Returns a list of mo.Html / mo.callout elements.
    """
    _bubbles = []
    for _m in history:
        _role = _m.get("role", "user")
        _text = _m.get("content", "")
        # Unescape double-encoded JSON strings (\\n -> \n, \\" -> ")
        if isinstance(_text, str):
            _text = _text.replace("\\n", "\n").replace('\\"', '"')
        _preview = _text
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


def render_feedback_summary(plan, design_ok, approved_slices, total_slices,
                            answered_qs, total_qs, answers, questions):
    """Render the feedback summary section (E) and return (vstack, finalize_btn, all_ok)."""
    total = 1 + total_slices + total_qs
    approved = (1 if design_ok else 0) + approved_slices + answered_qs
    pct = round(approved / total * 100) if total > 0 else 0
    all_ok = approved == total and total > 0
    remaining = total - approved
    bar_color = "#16a34a" if all_ok else "#3b82f6"

    progress_bar = mo.Html(
        f'<div style="display:flex;justify-content:space-between;font-size:0.8rem;'
        f'margin-bottom:4px"><span><strong>Review Progress</strong></span>'
        f'<span>{approved} of {total} completed &middot; {remaining} remaining'
        f'</span></div>'
        f'<div style="background:var(--marimo-monochrome-100,#e0e0e0);'
        f'border-radius:6px;height:14px;overflow:hidden">'
        f'<div style="background:{bar_color};height:100%;width:{pct}%;'
        f'border-radius:6px;transition:width .3s"></div></div>'
    )
    d_bg, d_fg, d_bd = (
        ("#dcfce7", "#166534", "#86efac") if design_ok
        else ("#fef3c7", "#92400e", "#f59e0b")
    )
    q_bg, q_fg, q_bd = (
        ("#dcfce7", "#166534", "#86efac") if answered_qs == total_qs
        else ("#fef3c7", "#92400e", "#f59e0b")
    )

    finalize_btn = mo.ui.run_button(label="Finalize Plan")

    _decision_rows = "".join(
        f"<tr><td style='padding:8px 12px;border-bottom:1px solid #e5e7eb'>{q['text']}</td>"
        f"<td style='padding:8px 12px;border-bottom:1px solid #e5e7eb;font-weight:600'>"
        f"{answers.get(q['id']) or '<em>pending</em>'}</td></tr>"
        for q in questions
    )
    decisions_table = mo.Html(
        f"<div style='margin:8px 0'><strong>Decisions Made</strong>"
        f"<table style='width:100%;border-collapse:collapse;margin-top:8px'>"
        f"<thead><tr style='border-bottom:2px solid #d1d5db'>"
        f"<th style='text-align:left;padding:8px 12px'>Question</th>"
        f"<th style='text-align:left;padding:8px 12px'>Decision</th></tr></thead>"
        f"<tbody>{_decision_rows}</tbody></table></div>"
    )

    summary = mo.vstack([
        mo.md("## E. Feedback Summary"),
        mo.hstack([
            stat_card("Slices", f"{approved_slices}/{total_slices}",
                      "#f0f4ff", "#1e3a5f", "#93c5fd"),
            stat_card("Design", "Approved" if design_ok else "Pending",
                      d_bg, d_fg, d_bd),
            stat_card("Questions", f"{answered_qs}/{total_qs}",
                      q_bg, q_fg, q_bd),
            stat_card("Progress", f"{pct}%",
                      "#dcfce7" if all_ok else "#f0f4ff",
                      "#166534" if all_ok else "#1e3a5f",
                      "#86efac" if all_ok else "#93c5fd"),
        ], justify="space-between", gap=0.75),
        progress_bar,
        decisions_table,
        (mo.callout(mo.md("**Plan finalized** — exported as static HTML"),
                    kind="success")
         if plan.get("meta", {}).get("status") == "finalized" else
         (finalize_btn if all_ok else mo.callout(
             mo.md("All sections must be approved before finalizing."),
             kind="warn"))),
    ])
    return summary, finalize_btn

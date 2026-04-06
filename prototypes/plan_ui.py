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

    def decision_display(answer):
        return mo.md(f"**{answer}**") if answer else status_badge("unanswered")

    decisions_table = mo.ui.table(
        [{"Question": q["text"], "Decision": decision_display(answers.get(q["id"]))}
         for q in questions],
        selection=None, label="Decisions Made",
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

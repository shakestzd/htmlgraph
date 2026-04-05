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

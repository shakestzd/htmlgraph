"""Render AI critique results for CRISPI plan notebooks."""

import marimo as mo


def _badge(text, bg, fg):
    return (f'<span style="background:{bg};color:{fg};padding:2px 8px;border-radius:9999px;'
            f'font-size:0.75rem;font-weight:600;white-space:nowrap">{text}</span>')


_STATUS = {
    "verified":  ("#dcfce7", "#166534", "Verified"),
    "unknown":   ("#fef3c7", "#92400e", "Unknown"),
    "falsified": ("#fee2e2", "#991b1b", "Falsified"),
}

_ITEM_KIND = {
    "success": ("#dcfce7", "#166534"),
    "warn":    ("#fef3c7", "#92400e"),
    "danger":  ("#fee2e2", "#991b1b"),
    "info":    ("#dbeafe", "#1e40af"),
}

_SEV = {"High": ("#fee2e2", "#991b1b"), "Medium": ("#fef3c7", "#92400e"), "Low": ("#dbeafe", "#1e40af")}


def render_critique(data):
    if not data:
        return mo.callout(mo.md(
            "**AI Critique Results** — _not yet run. Dispatching critique agents will populate this section._"
        ), kind="neutral")

    _reviewers = " + ".join(data.get("reviewers", []))
    _date = data.get("reviewed_at", "")

    # Assumptions.
    _assumption_html = ""
    for a in data.get("assumptions", []):
        bg, fg, label = _STATUS.get(a["status"], _STATUS["unknown"])
        ev = f' — <code>{a["evidence"]}</code>' if a.get("evidence") else ""
        _assumption_html += (
            f'<div style="display:flex;align-items:center;gap:8px;padding:6px 0">'
            f'{_badge(label, bg, fg)} <span><strong>{a["id"]}:</strong> {a["text"]}{ev}</span></div>\n'
        )

    # Critics side-by-side.
    _critic_cols = []
    for critic in data.get("critics", [])[:2]:
        _col = f'<div><h4 style="text-transform:uppercase;letter-spacing:.04em;font-size:.75rem;margin:0 0 8px">{critic["title"]}</h4>\n'
        for section in critic.get("sections", []):
            _col += f'<h4 style="font-size:.9rem;margin:12px 0 6px">{section["heading"]}</h4>\n'
            for item in section.get("items", []):
                bg, fg = _ITEM_KIND.get(item.get("kind", ""), ("#f0f4ff", "#1e3a5f"))
                _col += (f'<div style="margin:4px 0;font-size:.85rem">'
                         f'{_badge(item["badge"], bg, fg)} {item["text"]}</div>\n')
        _col += '</div>'
        _critic_cols.append(_col)

    _critics_html = ""
    if len(_critic_cols) == 2:
        _critics_html = f'<div style="display:grid;grid-template-columns:1fr 1fr;gap:20px;margin:12px 0">{_critic_cols[0]}{_critic_cols[1]}</div>'
    elif len(_critic_cols) == 1:
        _critics_html = _critic_cols[0]

    # Risks.
    _risk_html = ""
    for r in data.get("risks", []):
        bg, fg = _SEV.get(r["severity"], _SEV["Medium"])
        _risk_html += f'<tr><td>{r["risk"]}</td><td>{_badge(r["severity"], bg, fg)}</td><td>{r["mitigation"]}</td></tr>\n'
    _risk_table = ""
    if _risk_html:
        _risk_table = (
            '<h4 style="text-transform:uppercase;letter-spacing:.04em;font-size:.75rem;margin:16px 0 8px">RISK ASSESSMENT</h4>'
            '<table style="width:100%;font-size:.85rem;border-collapse:collapse">'
            '<thead><tr><th style="text-align:left;padding:6px 8px;border-bottom:1px solid var(--marimo-monochrome-200,#ccc)">Risk</th>'
            '<th style="text-align:left;padding:6px 8px;border-bottom:1px solid var(--marimo-monochrome-200,#ccc)">Severity</th>'
            '<th style="text-align:left;padding:6px 8px;border-bottom:1px solid var(--marimo-monochrome-200,#ccc)">Mitigation</th></tr></thead>'
            f'<tbody>{_risk_html}</tbody></table>'
        )

    # Synthesis.
    _syn = data.get("synthesis", "")
    _syn_html = ""
    if _syn:
        _syn_html = (f'<h4 style="text-transform:uppercase;letter-spacing:.04em;font-size:.75rem;margin:16px 0 8px">SYNTHESIS</h4>'
                     f'<p style="font-size:.85rem">{_syn}</p>')

    return mo.vstack([
        mo.md("### D. AI Critique Results"),
        mo.Html(
            f'<div style="border:1px solid var(--marimo-monochrome-200,#333);border-radius:8px;padding:16px">'
            f'<p style="color:#16a34a;font-size:.85rem;margin:0 0 16px">&#10003; Critique complete — {_reviewers} reviewed {_date}</p>'
            f'<h4 style="text-transform:uppercase;letter-spacing:.04em;font-size:.75rem;margin:0 0 8px">ASSUMPTION VERIFICATION</h4>'
            f'{_assumption_html}'
            f'{_critics_html}'
            f'{_risk_table}'
            f'{_syn_html}'
            f'</div>'
        ),
    ])

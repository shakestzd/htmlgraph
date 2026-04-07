"""Persistence helpers for CRISPI plan notebooks.

Handles SQLite feedback storage, amendment tracking, and plan finalization.
"""

from __future__ import annotations

import json
import os
import subprocess
from pathlib import Path
from typing import Any


def persist_feedback(
    plan_id: str,
    section: str,
    action: str,
    value: Any,
    question_id: str = "",
) -> None:
    """Write a single feedback entry to plan_feedback table."""
    import os
    import sqlite3

    env_hg = os.environ.get("HTMLGRAPH_DIR", "")
    if env_hg and Path(env_hg).exists():
        hg = Path(env_hg)
    else:
        cwd = Path.cwd()
        candidates = [
            cwd / ".htmlgraph",
            cwd.parent / ".htmlgraph",
            cwd.parent.parent / ".htmlgraph",
        ]
        hg = next((p for p in candidates if p.exists()), None)
    if not hg:
        return
    db = hg / "htmlgraph.db"
    if not db.exists():
        return
    conn = sqlite3.connect(str(db))
    conn.execute(
        """INSERT OR REPLACE INTO plan_feedback
           (plan_id, section, action, value, question_id, updated_at)
           VALUES (?, ?, ?, ?, ?, datetime('now'))""",
        (plan_id, section, action, str(value), question_id),
    )
    conn.commit()
    conn.close()


def _get_db():
    """Return (sqlite3, db_path) or (None, None) if DB not found."""
    import os
    import sqlite3

    env_hg = os.environ.get("HTMLGRAPH_DIR", "")
    if env_hg and Path(env_hg).exists():
        hg = Path(env_hg)
    else:
        cwd = Path.cwd()
        candidates = [
            cwd / ".htmlgraph",
            cwd.parent / ".htmlgraph",
            cwd.parent.parent / ".htmlgraph",
        ]
        hg = next((p for p in candidates if p.exists()), None)
    if not hg:
        return None, None
    db = hg / "htmlgraph.db"
    if not db.exists():
        return None, None
    return sqlite3, str(db)


def persist_amendment(plan_id: str, amendment: dict, amendment_id: str) -> None:
    """Store a proposed amendment from chat."""
    persist_feedback(plan_id, "amendment", "proposed", json.dumps(amendment), question_id=amendment_id)


def get_amendments(plan_id: str) -> list[dict]:
    """Retrieve all amendments for a plan with their current status."""
    sqlite3_mod, db_path = _get_db()
    if sqlite3_mod is None:
        return []
    conn = sqlite3_mod.connect(db_path)
    conn.row_factory = sqlite3_mod.Row
    rows = conn.execute(
        "SELECT action, value, question_id FROM plan_feedback "
        "WHERE plan_id = ? AND section = 'amendment' ORDER BY created_at ASC",
        (plan_id,),
    ).fetchall()
    conn.close()
    results = []
    for r in rows:
        try:
            val = json.loads(r["value"])
        except (json.JSONDecodeError, TypeError):
            val = {}
        results.append({"id": r["question_id"], "action": r["action"], "value": val})
    return results


def update_amendment_status(plan_id: str, amendment_id: str, new_action: str) -> None:
    """Change an amendment's status (proposed -> accepted/rejected)."""
    sqlite3_mod, db_path = _get_db()
    if sqlite3_mod is None:
        return
    conn = sqlite3_mod.connect(db_path)
    conn.execute(
        "UPDATE plan_feedback SET action = ?, updated_at = datetime('now') "
        "WHERE plan_id = ? AND section = 'amendment' AND question_id = ?",
        (new_action, plan_id, amendment_id),
    )
    conn.commit()
    conn.close()


def finalize_plan(
    plan: dict,
    plan_path: Path,
    slice_approvals: dict[str, bool],
    question_answers: dict[str, Any],
    yaml_module: Any,
) -> str:
    """Finalize a plan: update YAML status, export HTML, return summary markdown."""
    updated = dict(plan)
    updated["meta"]["status"] = "finalized"
    updated["design"]["approved"] = True
    for s in updated.get("slices", []):
        s["approved"] = slice_approvals.get(s["id"], False)
    for q in updated.get("questions", []):
        q["answer"] = question_answers.get(q["id"])

    # Apply accepted amendments to slices.
    plan_id = updated["meta"]["id"]
    amendments = get_amendments(plan_id)
    applied_count = 0
    for amendment in amendments:
        if amendment["action"] != "accepted":
            continue
        amendment_id = amendment["id"]
        val = amendment["value"]
        slice_num = val.get("slice_num")
        operation = val.get("operation")
        field = val.get("field")
        content = val.get("content")
        if slice_num is None or operation is None or field is None:
            continue

        # Find the matching slice.
        slice_obj = None
        for s in updated.get("slices", []):
            if s.get("id") == slice_num or (isinstance(slice_num, int) and updated["slices"].index(s) == slice_num):
                slice_obj = s
                break
        if slice_obj is None:
            continue

        # Apply the amendment based on operation type.
        if operation == "add":
            # Append to list fields.
            if field not in slice_obj:
                slice_obj[field] = []
            if isinstance(slice_obj[field], list):
                if isinstance(content, list):
                    slice_obj[field].extend(content)
                else:
                    slice_obj[field].append(content)
        elif operation == "set":
            # Replace scalar fields.
            slice_obj[field] = content
        elif operation == "remove":
            # Remove from list fields.
            if field in slice_obj and isinstance(slice_obj[field], list) and isinstance(content, list):
                slice_obj[field] = [x for x in slice_obj[field] if x not in content]

        update_amendment_status(plan_id, amendment_id, "applied")
        applied_count += 1

    plan_path.write_text(
        yaml_module.dump(
            updated, sort_keys=False, allow_unicode=True, default_flow_style=False
        )
    )

    approved = [s for s in updated["slices"] if s["approved"]]
    feature_lines = "\n".join(f"  - `{s['id']}` {s['title']}" for s in approved)
    decision_lines = "\n".join(
        f"- {q['text']}: **{q.get('answer', 'pending')}**"
        for q in updated.get("questions", [])
    )
    amendments_note = f"\n- {applied_count} amendments applied" if applied_count > 0 else ""

    # Export static HTML archive.
    export_path = plan_path.parent / f"{plan_id}.html"
    notebook_dir = Path.cwd()
    notebook_file = notebook_dir / "plan_notebook.py"
    if not notebook_file.exists():
        notebook_file = plan_path.parent.parent / "prototypes" / "plan_notebook.py"
    export_result = ""
    try:
        export_env = {**os.environ, "PLAN_EXPORT_MODE": "1"}
        subprocess.run(
            [
                "marimo", "export", "html", str(notebook_file),
                "-o", str(export_path), "--no-include-code",
                "--", "--plan", str(plan_path),
            ],
            capture_output=True, text=True, timeout=120,
            cwd=str(notebook_file.parent),
            env=export_env,
        )
        if export_path.exists() and export_path.stat().st_size > 1000:
            export_result = (
                f"\n\n**Exported:** `{export_path}` "
                f"({export_path.stat().st_size // 1024}KB)"
            )
        else:
            export_result = "\n\n**Export warning:** file created but may be incomplete"
    except Exception as e:
        export_result = f"\n\n**Export skipped:** {e}"

    return (
        f"## Plan Finalized\n\n"
        f"**{updated['meta']['title']}** — {len(approved)} slices approved.\n\n"
        f"**Features:**\n{feature_lines}\n\n"
        f"**Decisions:**\n{decision_lines}{amendments_note}\n\n"
        f"Saved to `{plan_path}` with status: **finalized**"
        f"{export_result}\n\n"
        f"> **Next steps:**\n>\n"
        f"> 1. `htmlgraph plan finalize-yaml {plan_id}` — create track and features from approved slices\n>\n"
        f"> 2. `/htmlgraph:execute` — dispatch agents to implement features"
    )

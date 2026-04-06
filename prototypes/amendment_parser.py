"""Amendment parser — extracts structured AMEND directives from chat text.

Syntax:
    AMEND slice-N: <operation> <field> "<content>"
    AMEND slice-N: <operation> <field> `content`
    AMEND slice-N: <operation> <field> content-to-end-of-line

Operations: add, remove, set
Fields: done_when, files, title, what, why, effort, risk
"""

from __future__ import annotations

import re

# Match: AMEND slice-N: operation field "content" or `content` or bare content
_AMEND_RE = re.compile(
    r"AMEND\s+slice-(\d+)\s*:\s*"       # AMEND slice-N:
    r"(add|remove|set)\s+"               # operation
    r"(done_when|files|title|what|why|effort|risk)\s+"  # field
    r'(?:"([^"]+)"|`([^`]+)`|(.+?))\s*$',  # content (quoted, backtick, or bare)
    re.IGNORECASE | re.MULTILINE,
)


def parse_amendments(text: str) -> list[dict]:
    """Extract AMEND directives from text.

    Returns list of dicts with keys: slice_num, field, operation, content.
    """
    results = []
    for m in _AMEND_RE.finditer(text):
        content = m.group(4) or m.group(5) or m.group(6) or ""
        results.append({
            "slice_num": int(m.group(1)),
            "operation": m.group(2).lower(),
            "field": m.group(3).lower(),
            "content": content.strip(),
        })
    return results

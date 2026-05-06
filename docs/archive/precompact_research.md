# PreCompact Hook Research Findings

## Hook Signature & Parameters

**Input Format (JSON via stdin):**
```json
{
  "session_id": "abc123",
  "transcript_path": "~/.claude/projects/.../session-id.jsonl",
  "permission_mode": "default",
  "hook_event_name": "PreCompact",
  "trigger": "manual" | "auto",
  "custom_instructions": ""
}
```

**Key Parameters:**
- `session_id` - Unique session identifier
- `transcript_path` - Path to JSONL transcript file (⚠️ KNOWN BUG: Often empty)
- `trigger` - "manual" (user ran /compact) or "auto" (context limit reached)
- `custom_instructions` - Instructions passed with manual /compact command
- `permission_mode` - Permission settings for session

**Return Behavior:**
- Exit code 0 = success, compaction proceeds
- Exit code non-zero = error (compaction still proceeds)
- Cannot block or prevent compaction
- No ability to modify compaction output

**Execution Constraints:**
- 60-second timeout limit
- Runs in project directory context
- Access to file system and environment variables
- No interactive capabilities

---

## When It Fires

**Automatic Trigger:**
- Context window approaches capacity (typically ~80-90% full)
- Claude Code automatically initiates compaction
- `trigger` = "auto"
- `custom_instructions` = ""

**Manual Trigger:**
- User runs `/compact` command
- User runs `/compact [custom instructions]`
- `trigger` = "manual"
- `custom_instructions` = user-provided text

---

## Use Cases & Capabilities

**1. Transcript Backup (Primary Use Case)**
```python
# Before compaction, save full conversation history
backup_dir = Path("logs/transcript_backups")
timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
shutil.copy2(transcript_path, backup_dir / f"backup_{timestamp}.jsonl")
```

**2. Context Preservation**
- Extract in-progress work from transcript
- Save critical decisions/state before summarization
- Prepare context refresh instructions for post-compaction
- Log compaction events for audit trail

**3. Work Detection**
```python
# Detect uncommitted git changes
result = subprocess.run(['git', 'status', '--porcelain'], capture_output=True)
has_uncommitted = len(result.stdout.strip()) > 0

# Detect in-progress Wipnote features
sdk = SDK()
in_progress = [f for f in sdk.features.list() if f.status == 'in-progress']
```

**4. User Warnings**
```python
# Warn about data loss risk
if has_uncommitted or in_progress:
    print("⚠️  WARNING: Uncommitted work detected before compaction!")
    print("   - Uncommitted git changes")
    print(f"   - {len(in_progress)} in-progress features")
    print("   Consider saving your work before continuing.")
```

**5. Automatic Work Preservation**
```python
# Auto-save in-progress features to snapshot
for feature in in_progress:
    snapshot_path = f".wipnote/snapshots/pre-compact-{timestamp}/{feature.id}.json"
    save_snapshot(feature, snapshot_path)
```

---

## Known Issues & Limitations

**🐛 Critical Bugs (as of Dec 2025):**

1. **Empty transcript_path (#13668)**
   - `transcript_path` often receives empty string instead of actual path
   - Blocks transcript backup functionality
   - Workaround: Use session discovery or fixed paths

2. **Hook Not Triggered (#13572)**
   - PreCompact hook sometimes doesn't fire for `/compact` command
   - Works manually but not in automation
   - No consistent reproduction steps

**⚠️ Design Limitations:**

1. **Cannot Block Compaction**
   - Hook runs *before* but can't prevent compaction
   - Exit codes don't affect compaction behavior
   - Informational/preparatory only, not preventive

2. **Cannot Modify Compaction**
   - Cannot change what gets summarized
   - Cannot inject additional context
   - Cannot customize compaction algorithm

3. **Timing Uncertainty**
   - Auto-compaction timing not predictable
   - May fire mid-operation
   - Race conditions possible with concurrent file access

---

## Implementation Examples

**Example 1: Basic Logging (from disler/claude-code-hooks-mastery)**
```python
#!/usr/bin/env -S uv run --script
import json, sys
from pathlib import Path
from datetime import datetime

# Read hook input
input_data = json.loads(sys.stdin.read())

# Log event
log_dir = Path("logs")
log_dir.mkdir(exist_ok=True)
with open(log_dir / "pre_compact.json", "a") as f:
    json.dump({
        **input_data,
        "logged_at": datetime.now().isoformat()
    }, f)
    f.write("\n")

sys.exit(0)
```

**Example 2: Context Preservation (from webdevtodayjason/claude-hooks)**
```python
# Detect Context Forge project
has_claude_md = Path("CLAUDE.md").exists()
has_docs = Path("Docs").is_dir()
has_prps = Path("PRPs").is_dir()

if has_claude_md or has_docs or has_prps:
    # Track implementation stage from transcript
    stage = extract_current_stage(input_data['transcript_path'])

    # Prepare refresh instructions
    refresh_instructions = f'''
After compaction, re-read:
- CLAUDE.md (project rules)
- Docs/ (specifications)
- PRPs/ (implementation stages)

Current stage: {stage}
Resume from: [last checkpoint]
'''

    # Save for post-compact recovery
    Path(".context-refresh").write_text(refresh_instructions)
```

**Example 3: Wipnote Integration**
```python
from wipnote import SDK

sdk = SDK(agent='pre-compact-hook')
input_data = json.loads(sys.stdin.read())

# Find in-progress work
in_progress = [f for f in sdk.features.list() if f.status == 'in-progress']

if in_progress and input_data['trigger'] == 'auto':
    # Auto-compaction during active work - create snapshot
    snapshot = sdk.spikes.create(f"Pre-Compact Snapshot {datetime.now()}")
    snapshot.set_findings(f'''
## Work State Before Auto-Compaction

**In-Progress Features:**
{chr(10).join(f"- {f.title} ({f.completion}% complete)" for f in in_progress)}

**Session:** {input_data['session_id'][:8]}
**Trigger:** Auto-compaction (context full)
**Time:** {datetime.now().isoformat()}
''').save()

    print(f"✅ Created snapshot: {snapshot.id}")
```

---

## Best Practices

**1. Graceful Error Handling**
```python
try:
    # Hook logic here
    sys.exit(0)
except Exception as e:
    # Always exit 0 - don't block compaction on errors
    sys.exit(0)
```

**2. Fast Execution**
- Stay under 60-second timeout
- Avoid heavy processing
- Use async/background tasks for slow operations
- Don't wait for user input

**3. Idempotent Operations**
- Handle multiple invocations safely
- Use unique filenames (timestamps, session IDs)
- Check for existing backups before creating

**4. Robust Path Handling**
```python
# Handle empty transcript_path bug
transcript_path = input_data.get('transcript_path', '')
if not transcript_path or not Path(transcript_path).exists():
    # Fallback: Find session file manually
    project_dir = Path.cwd()
    # Implementation specific to project structure
```

**5. Clear User Communication**
```python
if input_data['trigger'] == 'manual':
    # Verbose output for manual compaction
    print("📦 Compacting conversation...")
    print("✅ Transcript backed up")
elif input_data['trigger'] == 'auto':
    # Minimal output for auto
    print("⚡ Auto-compact triggered")
```

---

## Workaround: /compact Custom Instructions

**Problem:** PreCompact hooks have bugs and limitations.

**Solution:** Use `/compact` with structured instructions (no hooks needed):

```
/compact In addition to the default summary, explicitly include:

0) COMPACT NUMBER - This is compact #[N]

1) IMMEDIATE NEXT ACTION - [Specific imperative with file paths]

2) SETTLED DECISIONS - [Key decisions with rationale]

3) DEAD ENDS - [What failed and WHY]

4) TRUST ANCHORS - [What's verified working]

5) USER PREFERENCES - [Lasting preferences]

6) TASK QUEUE - [Pending tasks with dependencies]

7) BREAKTHROUGHS - [Key insights and why they matter]
```

**Advantages:**
- ✅ No hook infrastructure needed
- ✅ AI extracts context (more accurate)
- ✅ Works reliably (no trigger bugs)
- ✅ Self-documenting

---

## Integration Plan for Wipnote

**Approach 1: PreCompact Hook (When Bugs Fixed)**
```python
# .claude/hooks/wipnote_precompact.py
from wipnote import SDK

sdk = SDK(agent='pre-compact-hook')
input_data = json.loads(sys.stdin.read())

# 1. Detect uncommitted work
git_status = subprocess.run(['git', 'status', '--porcelain'],
                           capture_output=True, text=True)
has_uncommitted = bool(git_status.stdout.strip())

# 2. Find in-progress Wipnote items
in_progress_features = [f for f in sdk.features.list()
                       if f.status == 'in-progress']
in_progress_spikes = [s for s in sdk.spikes.list()
                     if s.status == 'in-progress']

# 3. Warn or auto-save based on trigger
if input_data['trigger'] == 'manual':
    # User-initiated - just warn
    if has_uncommitted or in_progress_features or in_progress_spikes:
        print("⚠️  WARNING: Active work detected!")
        if has_uncommitted:
            print("   - Uncommitted git changes")
        if in_progress_features:
            print(f"   - {len(in_progress_features)} in-progress features")
        if in_progress_spikes:
            print(f"   - {len(in_progress_spikes)} in-progress spikes")
        print("   Consider saving before compacting.")

elif input_data['trigger'] == 'auto':
    # Auto-compaction - create snapshot
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    snapshot = sdk.spikes.create(f"Auto-Compact Snapshot {timestamp}")

    snapshot_data = {
        'git_uncommitted': has_uncommitted,
        'features': [{'id': f.id, 'title': f.title, 'status': f.status}
                    for f in in_progress_features],
        'spikes': [{'id': s.id, 'title': s.title}
                  for s in in_progress_spikes],
        'session_id': input_data['session_id'],
        'timestamp': datetime.now().isoformat()
    }

    snapshot.set_findings(f"```json\n{json.dumps(snapshot_data, indent=2)}\n```")
    snapshot.save()

    print(f"✅ Work snapshot created: {snapshot.id}")
```

**Approach 2: /compact Instruction Template (Immediate)**

Add to Wipnote CLI: `wipnote compact-prep`

Outputs template for manual /compact command with current work state.

---

## References

**Official Documentation:**
- [Hooks Reference](https://code.claude.com/docs/en/hooks)
- [Hooks Guide](https://code.claude.com/docs/en/hooks-guide)
- [Blog: How to Configure Hooks](https://claude.com/blog/how-to-configure-hooks)

**GitHub Examples:**
- [disler/claude-code-hooks-mastery](https://github.com/disler/claude-code-hooks-mastery) - Production-ready examples
- [webdevtodayjason/claude-hooks](https://github.com/webdevtodayjason/claude-hooks) - Context Forge integration
- [GowayLee/cchooks](https://github.com/GowayLee/cchooks) - Python SDK for hooks

**Known Issues:**
- [Issue #13668](https://github.com/anthropics/claude-code/issues/13668) - Empty transcript_path
- [Issue #13572](https://github.com/anthropics/claude-code/issues/13572) - Hook not triggered

---

## Recommendations for Wipnote

**Immediate (Use Now):**
1. ✅ Implement `/compact` instruction template via CLI
2. ✅ Add `wipnote compact-prep` command
3. ✅ Document manual compaction workflow

**Short-term (When Bugs Fixed):**
1. ⏳ Implement PreCompact hook for auto-save
2. ⏳ Add snapshot creation before auto-compaction
3. ⏳ Integrate with git status checks

**Long-term (Future Enhancement):**
1. 🔮 PostCompact hook for context restoration
2. 🔮 Automatic work resumption after compaction
3. 🔮 Compaction analytics (frequency, triggers, impact)

**Decision:** Start with Approach 2 (instruction template) since it works reliably now, then add Approach 1 (hook) when bugs are resolved in Claude Code v1.1+.

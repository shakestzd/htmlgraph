# ID Generation

Wipnote provides collision-resistant, hash-based ID generation for multi-agent collaboration. This system prevents conflicts when multiple agents create tasks concurrently.

> **📖 Design Document:** For detailed architecture, implementation details, and rationale, see [Hash-Based IDs Design Document](../design/hash-based-ids.md).

## Overview

Traditional timestamp-based IDs (`feature-20241222-143022`) can collide when two agents create features at the same second. Hash-based IDs eliminate this problem by combining:

- **Title** (content-addressability)
- **Timestamp** (microsecond precision)
- **Random entropy** (4 bytes by default)

## ID Format

IDs follow the format `{prefix}-{hash}`:

| Node Type | Prefix | Example |
|-----------|--------|---------|
| Feature | `feat-` | `feat-a1b2c3d4` |
| Bug | `bug-` | `bug-12345678` |
| Chore | `chr-` | `chr-deadbeef` |
| Spike | `spk-` | `spk-87654321` |
| Epic | `epc-` | `epc-abcdef12` |
| Session | `sess-` | `sess-7890abcd` |
| Track | `trk-` | `trk-fedcba98` |
| Phase | `phs-` | `phs-11223344` |
| Spec | `spec-` | `spec-55667788` |
| Plan | `plan-` | `plan-99aabbcc` |

## Basic Usage

### Generating IDs

```python
from wipnote import generate_id

# Generate a feature ID
feature_id = generate_id("feature", "User Authentication")
# → "feat-a1b2c3d4"

# Generate a bug ID
bug_id = generate_id("bug", "Login fails on Safari")
# → "bug-12345678"

# Generate a track ID
track_id = generate_id("track", "OAuth Integration")
# → "trk-abcdef12"
```

### Automatic Generation

When using the SDK or CLI, IDs are generated automatically:

```python
from wipnote import SDK

sdk = SDK(agent="claude")

# ID generated automatically
feature = sdk.features.create(
    title="User Authentication"
).set_priority("high").save()

print(feature.id)  # → "feat-7f3a2b1c"
```

```bash
# CLI also generates hash-based IDs
wipnote feature create "User Authentication" --priority high
# Created: feat-9e8d7c6b
```

## Hierarchical IDs

For sub-tasks, use hierarchical IDs:

```python
from wipnote import generate_hierarchical_id

# Create parent feature
parent_id = generate_id("feature", "Auth System")  # → "feat-a1b2c3d4"

# Create sub-tasks
subtask1 = generate_hierarchical_id(parent_id, 1)  # → "feat-a1b2c3d4.1"
subtask2 = generate_hierarchical_id(parent_id, 2)  # → "feat-a1b2c3d4.2"

# Nested sub-tasks
nested = generate_hierarchical_id(subtask1, 1)     # → "feat-a1b2c3d4.1.1"
```

## Parsing IDs

Extract components from any ID:

```python
from wipnote import parse_id

# Parse a hash-based ID
result = parse_id("feat-a1b2c3d4.1.2")
# {
#     'prefix': 'feat',
#     'node_type': 'feature',
#     'hash': 'a1b2c3d4',
#     'hierarchy': [1, 2],
#     'is_legacy': False
# }

# Parse a legacy ID
result = parse_id("feature-20241222-143022")
# {
#     'prefix': 'feature',
#     'node_type': 'feature',
#     'hash': '20241222-143022',
#     'hierarchy': [],
#     'is_legacy': True
# }
```

## Validation

Check if IDs are valid:

```python
from wipnote import is_valid_id, is_legacy_id

# Hash-based IDs
is_valid_id("feat-a1b2c3d4")      # → True
is_valid_id("feat-a1b2c3d4.1.2")  # → True

# Legacy IDs (still valid)
is_valid_id("feature-20241222-143022")  # → True

# Invalid IDs
is_valid_id("invalid")            # → False
is_valid_id("feat-xyz")           # → False (not hex)

# Check legacy format
is_legacy_id("feature-20241222-143022")  # → True
is_legacy_id("feat-a1b2c3d4")            # → False
```

## Hierarchy Helpers

Navigate hierarchical IDs:

```python
from wipnote.ids import get_parent_id, get_root_id, get_depth

id = "feat-a1b2c3d4.1.2"

get_parent_id(id)  # → "feat-a1b2c3d4.1"
get_root_id(id)    # → "feat-a1b2c3d4"
get_depth(id)      # → 2

# Root IDs have no parent
get_parent_id("feat-a1b2c3d4")  # → None
get_depth("feat-a1b2c3d4")      # → 0
```

## Collision Resistance

With 4 bytes of random entropy (default), the probability of collision is approximately 1 in 4 billion per ID generated. Combined with microsecond timestamps and title hashing, collisions are effectively impossible even with thousands of concurrent agents.

```python
# Generate 1000 IDs with identical titles
ids = [generate_id("feature", "Same Title") for _ in range(1000)]
unique = len(set(ids))
print(f"Generated {unique} unique IDs")  # → "Generated 1000 unique IDs"
```

## Backward Compatibility

Legacy timestamp-based IDs remain fully supported:

- Existing features with old IDs continue to work
- `parse_id()` correctly identifies legacy format
- `is_valid_id()` accepts both formats
- No migration required

## API Reference

::: wipnote.ids.generate_id

::: wipnote.ids.generate_hierarchical_id

::: wipnote.ids.parse_id

::: wipnote.ids.is_valid_id

::: wipnote.ids.is_legacy_id

::: wipnote.ids.get_parent_id

::: wipnote.ids.get_root_id

::: wipnote.ids.get_depth

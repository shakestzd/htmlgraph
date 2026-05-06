# Atomic Operations Quick Start Guide

## Import

```python
from wipnote import (
    AtomicFileWriter,
    DirectoryLocker,
    atomic_rename,
    cleanup_orphaned_temp_files,
    safe_temp_file,
    validate_atomic_write,
)
from pathlib import Path
```

## Basic Usage

### Atomic Text Write

```python
from wipnote import AtomicFileWriter
from pathlib import Path

# Context manager (streaming)
target = Path("data.txt")
with AtomicFileWriter(target) as f:
    f.write("Hello, World!")

# Or static method (one-shot)
AtomicFileWriter.atomic_write(target, "Hello, World!")

# Verify
assert validate_atomic_write(target)
```

### Atomic JSON Write

```python
from wipnote import AtomicFileWriter

data = {
    "session_id": "sess-abc123",
    "created": "2026-01-08T12:00:00Z",
    "status": "active",
}

AtomicFileWriter.atomic_json_write(Path("session.json"), data)
```

### Safe Read with Retry

```python
from wipnote import AtomicFileWriter

# Retries up to 3 times with exponential backoff
content = AtomicFileWriter.safe_read_with_retry(
    Path("data.txt"),
    max_retries=3,
    retry_delay=0.1  # seconds
)
```

## Advanced Usage

### Custom Temp Directory

```python
from wipnote import AtomicFileWriter

target = Path("/data/important.txt")
temp_dir = Path("/tmp")  # Use different temp location

with AtomicFileWriter(target, temp_dir=temp_dir) as f:
    f.write("content")
```

### Directory Locking

```python
from wipnote import DirectoryLocker
from pathlib import Path

lock_dir = Path(".locks")
locker = DirectoryLocker(lock_dir)

# Exclusive lock (single writer)
if locker.acquire_exclusive_lock(timeout=5.0):
    try:
        # Critical section - only one process here
        with AtomicFileWriter(Path("registry.json")) as f:
            # Write registration
            pass
    finally:
        locker.release_lock()

# Shared lock (multiple readers)
if locker.acquire_shared_lock(timeout=5.0):
    try:
        # Multiple processes can be here
        content = Path("registry.json").read_text()
    finally:
        locker.release_lock()
```

### Cleanup Orphaned Files

```python
from wipnote import cleanup_orphaned_temp_files
from pathlib import Path

# Remove temp files older than 24 hours
deleted_count = cleanup_orphaned_temp_files(
    Path(".wipnote/sessions"),
    age_hours=24
)
print(f"Cleaned up {deleted_count} orphaned temp files")
```

### Generate Temp File Paths

```python
from wipnote import safe_temp_file
from pathlib import Path

# Generate unique temp path (doesn't create file)
temp_path = safe_temp_file(Path("/tmp"), prefix="session")
print(f"Temp path: {temp_path}")
# Output: Temp path: /tmp/.session-1234567890123456-abc123de.tmp
```

### Platform-Aware Atomic Rename

```python
from wipnote import atomic_rename
from pathlib import Path

src = Path("temp.txt")
dst = Path("final.txt")

atomic_rename(src, dst)  # Atomic on all platforms
# On POSIX: uses os.rename()
# On Windows: uses os.replace()
```

## Crash Safety Demonstration

### Scenario: Write with Exception

```python
from wipnote import AtomicFileWriter
from pathlib import Path

target = Path("data.txt")
target.write_text("original")

try:
    with AtomicFileWriter(target) as f:
        f.write("partial...")
        raise RuntimeError("Simulated crash!")
except RuntimeError:
    pass

# Original file is completely untouched!
assert target.read_text() == "original"
```

### Scenario: Cleanup After Crash

```python
from wipnote import cleanup_orphaned_temp_files
from pathlib import Path

# If process crashed, temp files might remain
# Cleanup removes them after age threshold

cleanup_orphaned_temp_files(Path(".wipnote"), age_hours=24)
```

## Integration with SessionRegistry

```python
from wipnote import AtomicFileWriter
from pathlib import Path
import json

class SessionRegistry:
    def register_session(self, instance_id, session_data):
        registry_file = self.active_dir / f"{instance_id}.json"

        # Atomic write - safe even if multiple instances
        with AtomicFileWriter(registry_file) as f:
            json.dump(session_data, f, indent=2)
```

## Error Handling

### File Not Found

```python
from wipnote import AtomicFileWriter

try:
    content = AtomicFileWriter.safe_read_with_retry(
        Path("nonexistent.txt"),
        max_retries=1
    )
except FileNotFoundError:
    print("File not found!")
```

### Permission Denied

```python
from wipnote import AtomicFileWriter
from pathlib import Path

try:
    with AtomicFileWriter(Path("/root/protected.txt")) as f:
        f.write("content")
except OSError as e:
    print(f"Permission denied: {e}")
```

### Validation

```python
from wipnote import validate_atomic_write
from pathlib import Path

path = Path("data.json")

if validate_atomic_write(path):
    print("File is valid and readable")
else:
    print("File is missing, corrupted, or unreadable")
```

## Performance Considerations

### Large Files
```python
from wipnote import AtomicFileWriter

# Streaming write (doesn't load entire file in memory)
with AtomicFileWriter(Path("large.bin")) as f:
    with open("source.bin", "rb") as src:
        while chunk := src.read(65536):  # 64KB chunks
            f.write(chunk.decode())
```

### Concurrent Writes
```python
from wipnote import AtomicFileWriter
from pathlib import Path

# Safe to do concurrently (different files)
with AtomicFileWriter(Path("file1.txt")) as f1:
    with AtomicFileWriter(Path("file2.txt")) as f2:
        f1.write("data1")
        f2.write("data2")
        # Both files committed atomically
```

## Testing

### Unit Test Example

```python
import pytest
from wipnote import AtomicFileWriter, validate_atomic_write
from pathlib import Path

def test_atomic_write_rollback(tmp_path):
    """Test that write is rolled back on exception."""
    target = tmp_path / "target.txt"
    target.write_text("original")

    try:
        with AtomicFileWriter(target) as f:
            f.write("new")
            raise ValueError("Error")
    except ValueError:
        pass

    # Original unchanged
    assert target.read_text() == "original"

    # Temp file cleaned up
    temp_files = list(tmp_path.glob(".tmp-*"))
    assert len(temp_files) == 0
```

## Best Practices

1. **Always use context managers** for exception safety
   ```python
   with AtomicFileWriter(path) as f:
       f.write(data)
   ```

2. **Validate after write** for critical files
   ```python
   AtomicFileWriter.atomic_json_write(path, data)
   assert validate_atomic_write(path)
   ```

3. **Use same directory for temp files** (same filesystem)
   ```python
   # Good: temp_dir omitted, uses same dir as target
   with AtomicFileWriter(target) as f:
       f.write(data)

   # Risky: temp on different filesystem (not atomic on cross-fs)
   with AtomicFileWriter(target, temp_dir=Path("/tmp")) as f:
       f.write(data)
   ```

4. **Cleanup orphaned files periodically**
   ```python
   # Run on app startup
   from wipnote import cleanup_orphaned_temp_files
   cleanup_orphaned_temp_files(Path(".wipnote"), age_hours=24)
   ```

5. **Use locks for multi-process writes**
   ```python
   locker = DirectoryLocker(lock_dir)
   if locker.acquire_exclusive_lock():
       try:
           # Safe to write
           pass
       finally:
           locker.release_lock()
   ```

## Troubleshooting

### Files Keep Getting Corrupted
- Check: Are you calling `.close()` on file objects? ✗
- Solution: Use context manager instead ✓
  ```python
  # Wrong
  f = open(path, "w")
  f.write(data)
  # File may be corrupted if crash here

  # Right
  with AtomicFileWriter(path) as f:
      f.write(data)  # Safe!
  ```

### Temp Files Accumulating
- Check: Are processes crashing without cleanup?
- Solution: Run cleanup periodically
  ```python
  cleanup_orphaned_temp_files(Path(".wipnote"), age_hours=24)
  ```

### Locks Timing Out
- Check: Is another process holding the lock?
- Solution: Increase timeout or improve lock granularity
  ```python
  locker.acquire_exclusive_lock(timeout=10.0)  # Longer timeout
  ```

## See Also

- **PHASE_1_3_IMPLEMENTATION.md** - Complete design documentation
- **src/python/wipnote/atomic_ops.py** - Source code with docstrings
- **tests/python/test_atomic_ops.py** - Test examples

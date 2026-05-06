# Gemini 3 Upgrade Guide

This document describes the migration from deprecated Gemini models to the latest Gemini 3 preview and beyond.

## Summary

**Key Change:** GeminiSpawner now uses `model=None` by default, which is the **RECOMMENDED** approach.

When `model=None`, the Gemini CLI automatically selects the best available model based on the task and current availability.

## Why model=None is Best

### 1. Automatic Model Updates

When you use `model=None`, you automatically benefit from Google's latest model improvements:

```python
from wipnote.orchestration.spawners import GeminiSpawner

spawner = GeminiSpawner()
result = spawner.spawn(
    prompt="Analyze this codebase",
    # model=None is the default - uses latest Gemini models
    track_in_wipnote=True
)
```

### 2. Current Default Models (as of Gemini CLI v0.22+)

When `model=None`, the CLI can use:
- **gemini-2.5-flash-lite**: Fast, efficient model for most tasks
- **gemini-3-flash-preview**: Preview of Gemini 3 with enhanced capabilities

### 3. Avoid Deprecation Issues

Older models may fail with newer CLI versions due to "thinking mode" incompatibility:

```python
# BAD - May fail with "thinking mode not supported" error
result = spawner.spawn(
    prompt="...",
    model="gemini-2.0-flash"  # DEPRECATED
)

# GOOD - Uses latest compatible models
result = spawner.spawn(
    prompt="..."
    # model=None (default)
)
```

## Migration Guide

### Before (Deprecated)

```python
result = spawner.spawn(
    prompt="Analyze code quality",
    model="gemini-2.0-flash",  # DEPRECATED
    track_in_wipnote=True
)
```

### After (Recommended)

```python
result = spawner.spawn(
    prompt="Analyze code quality",
    # model parameter omitted - uses best available models
    track_in_wipnote=True
)
```

## Model Reference

### Recommended

| Model | Description |
|-------|-------------|
| `None` (default) | **RECOMMENDED** - CLI chooses best available model |

### Available (if you must specify)

| Model | Description |
|-------|-------------|
| `gemini-2.5-flash-lite` | Fast, efficient |
| `gemini-3-flash-preview` | Gemini 3 with enhanced capabilities |
| `gemini-2.5-pro` | More capable, slower |

### Deprecated (may cause errors)

| Model | Status |
|-------|--------|
| `gemini-2.0-flash` | DEPRECATED - May fail with CLI v0.22+ |
| `gemini-1.5-flash` | DEPRECATED - May fail with CLI v0.22+ |
| `gemini-1.5-pro` | DEPRECATED - May fail with CLI v0.22+ |

## Gemini 3 Preview Features

The Gemini 3 preview model offers enhanced capabilities:

1. **Improved Reasoning**: Better at complex analysis tasks
2. **Enhanced Context Understanding**: More accurate codebase analysis
3. **Faster Response Times**: Optimized for interactive workflows
4. **Better Code Understanding**: Improved comprehension of code patterns

## Performance Characteristics

### gemini-2.5-flash-lite
- **Latency**: Very fast (sub-second for simple queries)
- **Context Window**: Large (suitable for codebase analysis)
- **Best For**: Quick exploration, simple queries, fast iterations

### gemini-3-flash-preview
- **Latency**: Fast
- **Context Window**: Large
- **Best For**: Complex analysis, detailed code review, advanced reasoning

## Testing the Upgrade

To verify the upgrade works correctly:

```python
from wipnote.orchestration.spawners import GeminiSpawner

spawner = GeminiSpawner()
result = spawner.spawn(
    prompt="Return a simple confirmation: 'Gemini spawner working correctly'",
    track_in_wipnote=False,
    timeout=30
)

if result.success:
    print(f"SUCCESS: {result.response}")
else:
    print(f"ERROR: {result.error}")
```

## Troubleshooting

### "Thinking mode not supported" Error

**Cause**: Using a deprecated model that doesn't support the CLI's thinking mode feature.

**Solution**: Remove the `model` parameter or set it to `None`:

```python
# Fix: Remove model parameter
result = spawner.spawn(prompt="...")
```

### CLI Not Found

**Cause**: Gemini CLI not installed.

**Solution**: Install from https://github.com/google/gemini-cli

### Model Not Available

**Cause**: Specified model not available in your region or API tier.

**Solution**: Use `model=None` to let the CLI choose an available model.

## Related Documentation

- [GeminiSpawner API](/src/python/wipnote/orchestration/spawners/gemini.py)
- [Gemini Skill](/packages/claude-plugin/.claude-plugin/skills/gemini/skill.md)
- [Orchestration Patterns](/src/python/wipnote/docs/ORCHESTRATION_PATTERNS.md)

## Changelog

- **2026-01-12**: Initial documentation for Gemini 3 upgrade
  - GeminiSpawner defaults to `model=None`
  - Updated all documentation examples
  - Deprecated `gemini-2.0-flash`, `gemini-1.5-flash`, `gemini-1.5-pro` references

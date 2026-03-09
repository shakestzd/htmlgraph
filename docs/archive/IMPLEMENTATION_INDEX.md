# Retry Decorator Implementation - Complete Index

## Quick Navigation

### Documentation
- **Getting Started**: [RETRY_DECORATOR.md](./docs/RETRY_DECORATOR.md) - Comprehensive guide (500+ lines)
- **Implementation Summary**: [RETRY_DECORATOR_SUMMARY.md](./RETRY_DECORATOR_SUMMARY.md) - What was built and why

### Source Code
- **Core Implementation**: [src/python/htmlgraph/decorators.py](./src/python/htmlgraph/decorators.py) (280 lines)
  - `@retry()` - Main synchronous decorator
  - `@retry_async()` - Async variant
  - `RetryError` - Exception class

### Tests
- **Test Suite**: [tests/python/test_decorators.py](./tests/python/test_decorators.py) (400+ lines, 32 tests)

### Examples
- **Practical Examples**: [examples/retry_decorator_examples.py](./examples/retry_decorator_examples.py) (500+ lines)
  - 10 real-world usage examples
  - Runnable with: `uv run python examples/retry_decorator_examples.py`

## Key Files At A Glance

| File | Purpose | Lines | Status |
|------|---------|-------|--------|
| `src/python/htmlgraph/decorators.py` | Core implementation | 280 | ✓ Complete |
| `tests/python/test_decorators.py` | Test suite (32 tests) | 400+ | ✓ All passing |
| `examples/retry_decorator_examples.py` | Practical examples | 500+ | ✓ Runnable |
| `docs/RETRY_DECORATOR.md` | Full documentation | 500+ | ✓ Comprehensive |
| `src/python/htmlgraph/__init__.py` | Package exports | Updated | ✓ Ready |

## Usage Quick Start

### Import
```python
from htmlgraph import retry, retry_async, RetryError
```

### Basic Example
```python
@retry()
def fetch_data():
    """Retries with defaults: 3 attempts, 1s initial delay, 2x backoff."""
    response = requests.get('https://api.example.com/data')
    response.raise_for_status()
    return response.json()
```

### Advanced Example
```python
@retry(
    max_attempts=5,
    initial_delay=0.5,
    max_delay=30.0,
    exponential_base=1.5,
    exceptions=(ConnectionError, TimeoutError),
    on_retry=log_retry_attempt,
)
def resilient_operation():
    pass
```

### Async Example
```python
@retry_async(max_attempts=3)
async def async_operation():
    async with aiohttp.ClientSession() as session:
        async with session.get('https://api.example.com') as resp:
            return await resp.json()
```

## Feature Overview

### Core Features
- ✓ Exponential backoff with configurable base
- ✓ Jitter support (prevents thundering herd)
- ✓ Exception filtering (retry specific exceptions)
- ✓ Custom callbacks for monitoring
- ✓ Async/await support
- ✓ Full type hints and validation

### Quality Metrics
- ✓ Type checking: mypy 100% PASS
- ✓ Linting: ruff 0 ERRORS
- ✓ Tests: 32/32 PASSING
- ✓ Documentation: Comprehensive
- ✓ Examples: 10 detailed scenarios

## Test Coverage

### Test Categories (32 tests, all passing)
- Basic functionality (6 tests)
- Exception handling (3 tests)
- Backoff timing (4 tests)
- Callbacks (4 tests)
- Input validation (6 tests)
- Async support (5 tests)
- Integration scenarios (4 tests)

### Run Tests
```bash
uv run pytest tests/python/test_decorators.py -v
# Output: 32 passed in 5.13s
```

## Design Decisions

### Why Exponential Backoff?
- Prevents overwhelming failing services
- Gives services time to recover
- Reduces cascading failures
- Industry standard approach

### Why Jitter?
- Prevents "thundering herd" problem
- Spreads retry attempts across time
- Improves success rate under high load
- Enabled by default for safety

### Why Exception Filtering?
- Distinguishes transient from permanent errors
- Fail fast on application errors
- Reduce latency for non-recoverable failures
- Better debugging with clear error messages

### Why Custom Callbacks?
- Flexibility for monitoring/metrics
- Non-intrusive instrumentation
- Extensible without code changes
- Support for advanced patterns

## API Reference

### @retry Decorator
```python
@retry(
    max_attempts: int = 3,
    initial_delay: float = 1.0,
    max_delay: float = 60.0,
    exponential_base: float = 2.0,
    jitter: bool = True,
    exceptions: tuple[Type[Exception], ...] = (Exception,),
    on_retry: Optional[Callable[[int, Exception, float], None]] = None,
) -> Callable[[Callable[..., T]], Callable[..., T]]
```

### @retry_async Decorator
```python
@retry_async(
    # Same parameters as @retry
) -> Callable[[Callable[..., Any]], Callable[..., Any]]
```

### RetryError Exception
```python
class RetryError(Exception):
    function_name: str        # Name of failed function
    attempts: int             # Number of attempts
    last_exception: Exception # Final exception
```

## Backoff Calculation

### Formula
```
delay = min(initial_delay * (base ^ attempt), max_delay)
```

### Example (base=2.0, initial_delay=1.0)
- Attempt 1: 1s
- Attempt 2: 2s
- Attempt 3: 4s
- Attempt 4: 8s
- Attempt 5: 16s
- Attempt 6: 32s

### Jitter Effect
- Delay multiplied by random value in [0.5, 1.5]
- Prevents synchronized retries in distributed systems

## Common Patterns

### API Retry
```python
@retry(exceptions=(ConnectionError, TimeoutError))
def api_call():
    pass
```

### Database Connection
```python
@retry(max_attempts=5, initial_delay=1.0)
def connect_db():
    pass
```

### File Operations
```python
@retry(exceptions=(IOError, OSError))
def read_file():
    pass
```

### Critical Operations
```python
@retry(
    max_attempts=10,
    exponential_base=2.0,
    jitter=True,  # Important!
)
def critical_op():
    pass
```

## Troubleshooting

### Function Keeps Retrying
**Problem**: Retries on all exceptions (not desired)
**Solution**: Specify exception types
```python
@retry(exceptions=(ConnectionError, TimeoutError))
```

### Delays Too Long
**Problem**: Waiting too long between retries
**Solution**: Reduce delays
```python
@retry(initial_delay=0.1, max_delay=5.0)
```

### Tests Taking Long
**Problem**: Retry delays slow down tests
**Solution**: Disable jitter and reduce delays
```python
@retry(initial_delay=0.01, jitter=False)
```

### Need Custom Logic
**Problem**: Standard backoff not sufficient
**Solution**: Use custom callback
```python
def custom_callback(attempt, exc, delay):
    # Custom logic
    pass

@retry(on_retry=custom_callback)
```

## Performance Profile

### Time Complexity
- Success case: O(1)
- Failure case: O(n) where n = max_attempts
- Delays: Exponential growth between retries

### Space Complexity
- O(1) - minimal memory overhead

### Network Impact
- Beneficial: Prevents hammering failing services
- Jitter spreads load across time
- Exponential backoff allows service recovery

## Next Steps

1. **Explore documentation**: Read [docs/RETRY_DECORATOR.md](./docs/RETRY_DECORATOR.md)
2. **Review examples**: Check [examples/retry_decorator_examples.py](./examples/retry_decorator_examples.py)
3. **Run tests**: Execute `uv run pytest tests/python/test_decorators.py -v`
4. **Use in code**: Import and decorate your functions
5. **Monitor**: Use callbacks for metrics/logging

## Related Resources

- [Exponential Backoff (Wikipedia)](https://en.wikipedia.org/wiki/Exponential_backoff)
- [Thundering Herd Problem](https://en.wikipedia.org/wiki/Thundering_herd_problem)
- [Circuit Breaker Pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
- [Rate Limiting Strategies](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)

## Support

For issues or questions:
1. Check [RETRY_DECORATOR.md](./docs/RETRY_DECORATOR.md) for detailed documentation
2. Review [test_decorators.py](./tests/python/test_decorators.py) for usage examples
3. Check [retry_decorator_examples.py](./examples/retry_decorator_examples.py) for patterns
4. Review source code in [decorators.py](./src/python/htmlgraph/decorators.py)

---

**Status**: ✓ Production Ready
**Quality**: ✓ 100% Type Checked | 0 Lint Errors | 32/32 Tests Passing
**Documentation**: ✓ Complete | ✓ Comprehensive | ✓ Examples Included

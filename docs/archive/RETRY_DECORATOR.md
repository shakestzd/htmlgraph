# Retry Decorator with Exponential Backoff

A production-grade Python decorator for adding automatic retry logic with exponential backoff to any function. Handles transient failures gracefully in distributed systems, I/O operations, and network requests.

## Quick Start

```python
from wipnote import retry

@retry()
def fetch_data_from_api():
    """Retries with sensible defaults: 3 attempts, 1s initial delay, 2x backoff."""
    response = requests.get('https://api.example.com/data')
    response.raise_for_status()
    return response.json()
```

## Table of Contents

- [Installation](#installation)
- [Core Features](#core-features)
- [Basic Usage](#basic-usage)
- [Advanced Configuration](#advanced-configuration)
- [Async Support](#async-support)
- [Exception Filtering](#exception-filtering)
- [Custom Callbacks](#custom-callbacks)
- [Real-World Examples](#real-world-examples)
- [API Reference](#api-reference)
- [Performance Considerations](#performance-considerations)
- [Testing](#testing)

## Installation

The retry decorator is included in the wipnote package:

```bash
uv pip install wipnote
```

Or use directly in your project:

```python
from wipnote import retry, retry_async, RetryError
```

## Core Features

### Exponential Backoff

Automatically increases delay between retries to prevent overwhelming failing services:

```
Attempt 1: immediate
Attempt 2: wait 1s, then retry
Attempt 3: wait 2s, then retry
Attempt 4: wait 4s, then retry
```

Formula: `delay = min(initial_delay * (base ** attempt), max_delay)`

### Jitter

Adds randomness to delays to prevent thundering herd in distributed systems:

```python
@retry(jitter=True)  # Delays vary ±50%
def my_function():
    pass
```

### Exception Filtering

Retry only on specific exceptions, fail fast on others:

```python
@retry(exceptions=(ConnectionError, TimeoutError))
def api_call():
    # Retries on network errors
    # Fails immediately on auth errors or 404s
    pass
```

### Custom Callbacks

Monitor retry attempts with custom logging or metrics:

```python
def on_retry(attempt: int, exc: Exception, delay: float):
    logger.warning(f"Retry #{attempt} after {delay}s: {exc}")

@retry(on_retry=on_retry)
def critical_operation():
    pass
```

## Basic Usage

### Default Behavior

```python
@retry()
def unstable_function():
    """Uses defaults: 3 attempts, 1s initial delay, 2x exponential backoff."""
    return expensive_operation()
```

Default parameters:
- `max_attempts=3` - Total number of attempts
- `initial_delay=1.0` - First retry delay in seconds
- `max_delay=60.0` - Maximum delay cap
- `exponential_base=2.0` - Backoff multiplier
- `jitter=True` - Add randomness to delays
- `exceptions=(Exception,)` - Catch all exceptions

### Success on First Attempt

```python
# If function succeeds, no delays
result = fetch_data()  # Returns immediately if successful
```

### Automatic Retries on Failure

```python
# If function fails with configured exception type, retries automatically
@retry(max_attempts=3)
def failing_operation():
    if some_condition():
        raise ConnectionError("Network error")
    return "success"
```

### Exhausted Retries

```python
from wipnote import RetryError

@retry(max_attempts=2)
def always_fails():
    raise ValueError("Persistent error")

try:
    always_fails()
except RetryError as e:
    print(f"Failed after {e.attempts} attempts")
    print(f"Last error: {e.last_exception}")
```

## Advanced Configuration

### Custom Backoff Parameters

```python
@retry(
    max_attempts=5,
    initial_delay=0.5,      # Start with 0.5s
    max_delay=30.0,         # Cap at 30s
    exponential_base=1.5,   # Slower growth than 2.0
)
def conservative_retry():
    """Gentler backoff: 0.5s, 0.75s, 1.125s, 1.688s, 2.5s"""
    pass
```

Backoff progression for above config:
- Attempt 1 fails: wait 0.5s
- Attempt 2 fails: wait 0.75s (0.5 * 1.5)
- Attempt 3 fails: wait 1.125s (0.75 * 1.5)
- Attempt 4 fails: wait 1.688s (1.125 * 1.5)
- Attempt 5 fails: wait 2.5s (1.688 * 1.5)

### Aggressive Retry Strategy

```python
@retry(
    max_attempts=10,
    initial_delay=0.05,
    max_delay=30.0,
    exponential_base=2.0,
    jitter=True,
)
def critical_operation():
    """Many attempts, fast initial retries, gradual backoff."""
    pass
```

This is useful for:
- Critical operations that must succeed
- High-traffic distributed systems (jitter prevents congestion)
- Short-lived transient failures

### Disable Jitter for Deterministic Testing

```python
@retry(
    max_attempts=3,
    initial_delay=0.1,
    jitter=False,  # Exact timing for reproducibility
)
def test_function():
    """Delays are exactly: 0.1s, 0.2s, 0.4s"""
    pass
```

## Async Support

The `retry_async` decorator works with async/await without blocking:

```python
import asyncio
from wipnote import retry_async

@retry_async(max_attempts=3, initial_delay=0.1)
async def async_api_call():
    """Non-blocking retry with asyncio.sleep instead of time.sleep."""
    async with aiohttp.ClientSession() as session:
        async with session.get('https://api.example.com') as resp:
            return await resp.json()

# Run it
result = asyncio.run(async_api_call())
```

### Concurrent Operations

```python
@retry_async(max_attempts=3)
async def fetch_user(user_id: int):
    async with aiohttp.ClientSession() as session:
        async with session.get(f'https://api.example.com/users/{user_id}') as resp:
            return await resp.json()

# Run multiple operations in parallel
users = await asyncio.gather(
    fetch_user(1),
    fetch_user(2),
    fetch_user(3),
)
```

## Exception Filtering

### Retry on Multiple Exception Types

```python
@retry(
    max_attempts=5,
    exceptions=(ConnectionError, TimeoutError, OSError),
)
def resilient_network_call():
    """Retries on network errors, fails immediately on auth errors."""
    pass
```

### Fail Fast on Application Errors

```python
@retry(
    exceptions=(ConnectionError,),  # Only retry network errors
)
def api_with_fallback():
    """
    Retries on ConnectionError (transient)
    Fails immediately on ValueError (application error)
    """
    response = requests.get(url)
    if response.status_code == 404:
        raise ValueError("Resource not found")  # Don't retry this
    response.raise_for_status()  # ConnectionError will retry
    return response.json()
```

### Only Retry Specific Errors

```python
import asyncio

@retry(exceptions=(asyncio.TimeoutError,))
async def timeout_sensitive_operation():
    """Only retries on timeout, not on other exceptions."""
    pass
```

## Custom Callbacks

### Basic Logging

```python
def log_retry(attempt: int, exc: Exception, delay: float) -> None:
    """Log each retry with full details."""
    print(f"Attempt {attempt} failed with {type(exc).__name__}: {exc}")
    print(f"Retrying in {delay:.2f} seconds...")

@retry(max_attempts=4, on_retry=log_retry)
def traced_operation():
    pass
```

### Structured Logging

```python
import logging

logger = logging.getLogger(__name__)

def structured_log_retry(attempt: int, exc: Exception, delay: float) -> None:
    logger.warning(
        "Function retry",
        extra={
            "attempt": attempt,
            "exception_type": type(exc).__name__,
            "exception_message": str(exc),
            "delay_seconds": delay,
        }
    )

@retry(on_retry=structured_log_retry)
def important_function():
    pass
```

### Metrics Collection

```python
from prometheus_client import Counter

retry_attempts = Counter('function_retries', 'Number of retries', ['function_name'])

def collect_metrics(attempt: int, exc: Exception, delay: float) -> None:
    retry_attempts.labels(function_name='api_call').inc()

@retry(on_retry=collect_metrics)
def monitored_operation():
    pass
```

### Alert on Threshold

```python
def alert_on_many_retries(attempt: int, exc: Exception, delay: float) -> None:
    if attempt >= 3:  # Alert after 3rd attempt
        send_alert(f"Operation failing: {exc}")

@retry(max_attempts=5, on_retry=alert_on_many_retries)
def critical_with_alerting():
    pass
```

## Real-World Examples

### Database Connection

```python
@retry(
    max_attempts=5,
    initial_delay=1.0,
    max_delay=10.0,
    exceptions=(ConnectionError, TimeoutError),
)
def get_database_connection(db_url: str):
    """Connect to database with exponential backoff."""
    return psycopg2.connect(db_url)
```

### API Client

```python
class APIClient:
    @retry(
        max_attempts=3,
        initial_delay=0.5,
        exceptions=(ConnectionError, requests.Timeout),
    )
    def get(self, endpoint: str):
        response = requests.get(
            f"{self.base_url}/{endpoint}",
            timeout=5
        )
        response.raise_for_status()
        return response.json()

    @retry(max_attempts=2)
    def post(self, endpoint: str, data: dict):
        response = requests.post(
            f"{self.base_url}/{endpoint}",
            json=data,
            timeout=5
        )
        response.raise_for_status()
        return response.json()
```

### File I/O

```python
@retry(
    max_attempts=3,
    initial_delay=0.1,
    exceptions=(IOError, OSError),
)
def read_file_safe(filepath: str) -> str:
    """Read file, retrying if locked or temporarily inaccessible."""
    with open(filepath, 'r') as f:
        return f.read()
```

### Cache Warming

```python
@retry(max_attempts=3)
def warm_cache():
    """Initialize cache, retrying on network errors."""
    for key in important_keys:
        data = fetch_and_cache(key)
    return True
```

### Distributed System Coordination

```python
@retry(
    max_attempts=10,
    initial_delay=0.1,
    max_delay=30.0,
    exponential_base=2.0,
    jitter=True,  # Important: prevent thundering herd
)
def acquire_distributed_lock(lock_name: str):
    """Acquire distributed lock with exponential backoff + jitter."""
    return redis_client.set(
        f"lock:{lock_name}",
        uuid.uuid4(),
        ex=60,
        nx=True  # Only if doesn't exist
    )
```

## API Reference

### `@retry` Decorator

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

**Parameters:**

- **max_attempts** (int, default=3)
  - Maximum number of attempts
  - Must be >= 1
  - Example: `max_attempts=5` tries up to 5 times

- **initial_delay** (float, default=1.0)
  - Initial delay before first retry in seconds
  - Must be >= 0
  - Example: `initial_delay=0.5` starts with 0.5s delay

- **max_delay** (float, default=60.0)
  - Maximum delay cap between retries
  - Must be >= initial_delay
  - Example: `max_delay=10.0` caps delays at 10 seconds

- **exponential_base** (float, default=2.0)
  - Base for exponential backoff calculation
  - Must be > 0
  - Typical values: 1.5 (slow), 2.0 (normal), 3.0 (fast)
  - Example: `exponential_base=2.0` doubles delay each time

- **jitter** (bool, default=True)
  - Whether to add random jitter to delays
  - Helps prevent thundering herd in distributed systems
  - Jitter range: [0.5 * delay, 1.5 * delay]
  - Example: `jitter=False` for deterministic testing

- **exceptions** (tuple[Type[Exception], ...], default=(Exception,))
  - Exception types to retry on
  - Other exceptions propagate immediately
  - Example: `exceptions=(ConnectionError, TimeoutError)`

- **on_retry** (Optional[Callable], default=None)
  - Callback invoked before each retry
  - Signature: `(attempt: int, exc: Exception, delay: float) -> None`
  - Example: `on_retry=custom_logger`

**Returns:**
- Decorated function with retry logic

**Raises:**
- **RetryError**: When all retry attempts are exhausted
  - Contains: `function_name`, `attempts`, `last_exception`
  - Other configured exceptions propagate immediately

### `@retry_async` Decorator

Identical to `@retry` but for async functions, using `asyncio.sleep` instead of `time.sleep`.

```python
@retry_async(
    max_attempts: int = 3,
    initial_delay: float = 1.0,
    max_delay: float = 60.0,
    exponential_base: float = 2.0,
    jitter: bool = True,
    exceptions: tuple[Type[Exception], ...] = (Exception,),
    on_retry: Optional[Callable[[int, Exception, float], None]] = None,
) -> Callable[[Callable[..., Any]], Callable[..., Any]]
```

All parameters identical to `@retry`.

### `RetryError` Exception

```python
class RetryError(Exception):
    """Raised when function exhausts all retry attempts."""

    def __init__(
        self,
        function_name: str,
        attempts: int,
        last_exception: Exception,
    ):
        self.function_name = str    # Name of function that failed
        self.attempts = int         # Number of attempts made
        self.last_exception = Exception  # Final exception before giving up
```

## Performance Considerations

### Memory Usage

- Minimal overhead: One wrapper function per decorated function
- Jitter uses `random.random()` (negligible)
- No persistent state maintained

### CPU Usage

- Negligible: Just time.sleep or asyncio.sleep during delays
- No busy-waiting or polling

### Network Impact

**Beneficial:**
- Prevents hammering failing services
- Jitter prevents thundering herd
- Exponential backoff allows service recovery time

**Example impact (with jitter):**
- 100 clients hitting failing service
- Without retry: 100 requests per second
- With jitter: Spread across seconds due to random delays
- Result: Reduced load allows faster service recovery

### Timing Example

```python
@retry(
    max_attempts=4,
    initial_delay=1.0,
    exponential_base=2.0,
    jitter=False,
)
def worst_case():
    """All attempts fail."""
    raise ValueError("Error")
```

Total time: 1s + 2s + 4s = 7 seconds to exhaust retries

## Testing

### Unit Testing

```python
import pytest
from wipnote import retry, RetryError

def test_retry_succeeds_immediately():
    @retry()
    def success():
        return "ok"

    assert success() == "ok"

def test_retry_after_failures():
    call_count = 0

    @retry(max_attempts=3, initial_delay=0.01)
    def eventually_succeeds():
        nonlocal call_count
        call_count += 1
        if call_count < 3:
            raise ValueError("Not yet")
        return "success"

    assert eventually_succeeds() == "success"
    assert call_count == 3

def test_retry_exhaustion():
    @retry(max_attempts=2, initial_delay=0.01)
    def always_fails():
        raise ValueError("Persistent error")

    with pytest.raises(RetryError) as exc_info:
        always_fails()

    assert exc_info.value.attempts == 2
```

### Integration Testing

```python
def test_with_external_service():
    @retry(
        max_attempts=3,
        initial_delay=0.5,
        exceptions=(ConnectionError,),
    )
    def call_external_api():
        response = requests.get("https://api.example.com")
        response.raise_for_status()
        return response.json()

    # Will retry on network errors
    result = call_external_api()
    assert result is not None
```

### Async Testing

```python
import pytest

@pytest.mark.asyncio
async def test_async_retry():
    call_count = 0

    @retry_async(max_attempts=3, initial_delay=0.01)
    async def async_eventually_succeeds():
        nonlocal call_count
        call_count += 1
        if call_count < 3:
            raise ValueError("Not yet")
        return "success"

    result = await async_eventually_succeeds()
    assert result == "success"
    assert call_count == 3
```

## Common Patterns

### Fail Fast with Fallback

```python
@retry(exceptions=(ConnectionError, TimeoutError))
def primary_source():
    """Retries network errors, falls through for logic errors."""
    pass

def get_data():
    try:
        return primary_source()
    except (ValueError, KeyError):
        # Application errors - return fallback
        return fallback_data()
    except RetryError:
        # Network errors after retries
        return fallback_data()
```

### Cascading Timeouts

```python
@retry(
    max_attempts=3,
    initial_delay=5.0,
    max_delay=15.0,
    exceptions=(TimeoutError,),
)
def operation_with_cascading_timeouts():
    """First attempt: 0s timeout, Second: 5s, Third: 15s"""
    pass
```

### Circuit Breaker Pattern

```python
failure_count = 0

def monitor_failures(attempt, exc, delay):
    global failure_count
    failure_count += 1
    if failure_count > 3:
        # Open circuit - don't retry
        raise CircuitBreakerOpen("Too many failures")

@retry(on_retry=monitor_failures)
def protected_operation():
    pass
```

## Troubleshooting

### Function Keeps Retrying Unexpectedly

```python
# Wrong: Retries on ALL exceptions
@retry()
def my_function():
    pass

# Correct: Only retry on transient errors
@retry(exceptions=(ConnectionError, TimeoutError))
def my_function():
    pass
```

### Delays Are Too Long

```python
# Reduce max_delay or initial_delay
@retry(
    initial_delay=0.1,    # Was 1.0
    max_delay=5.0,        # Was 60.0
)
def faster_retries():
    pass
```

### Tests Taking Too Long

```python
# Disable delays for unit tests
@retry(initial_delay=0.0, jitter=False)
def test_function():
    pass
```

### Need Fine-Grained Control

```python
def custom_backoff(attempt, exc, delay):
    # Custom logic
    if attempt > 2:
        # Give up early on specific errors
        if isinstance(exc, ValueError):
            raise RetryError(...)

@retry(on_retry=custom_backoff)
def custom_logic():
    pass
```

## See Also

- [exponential backoff](https://en.wikipedia.org/wiki/Exponential_backoff)
- [jitter (thundering herd)](https://en.wikipedia.org/wiki/Thundering_herd_problem)
- [circuit breaker pattern](https://martinfowler.com/bliki/CircuitBreaker.html)
- [timeout patterns](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)

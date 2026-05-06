# Server

Development server for the dashboard.

## Overview

The server module runs the Wipnote FastAPI application, which provides the REST API, WebSocket and SSE endpoints, and the activity feed backend used by the dashboard. The primary interactive dashboard is served by a separate Phoenix LiveView application at `http://localhost:4000`; the FastAPI server (default port 8080) handles the data layer.

## Usage

### Command Line

```bash
# Start server on default port (8080)
wipnote serve

# Custom port
wipnote serve --port 3000

# Custom host
wipnote serve --host 0.0.0.0 --port 8080

# Auto-reload on file changes
wipnote serve --watch
```

### Python API

```python
from wipnote.server import serve

# Start server
serve(
    graph_dir=".wipnote",
    port=8080,
    host="localhost",
    watch=False
)
```

## Features

- FastAPI application with REST endpoints
- WebSocket and SSE support for live dashboard updates
- CORS headers for local development
- Gzip compression
- Cache headers

## Complete API Reference

For detailed API documentation with method signatures and server configuration, see the Python source code in `src/python/wipnote/api/`.

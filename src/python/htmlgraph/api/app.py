"""
FastAPI application factory and configuration.

Handles app creation, lifespan management, router registration,
and dependency injection setup.
"""

import logging
import sqlite3
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles

from htmlgraph.api.cache import QueryCache, init_cache_backend
from htmlgraph.api.db import DatabaseManager
from htmlgraph.api.templates import create_jinja_environment

logger = logging.getLogger(__name__)


def _ensure_database_initialized(db_path: str) -> None:
    """Ensure SQLite database exists and has correct schema.

    Args:
        db_path: Path to SQLite database file
    """
    db_file = Path(db_path)
    db_file.parent.mkdir(parents=True, exist_ok=True)

    # Check if database exists and has tables
    try:
        conn = sqlite3.connect(db_path)
        cursor = conn.cursor()

        # Query existing tables
        cursor.execute("SELECT name FROM sqlite_master WHERE type='table'")
        tables = cursor.fetchall()
        table_names = [t[0] for t in tables]

        # Always run create_tables() to apply migrations and add any missing
        # tables introduced by newer versions.
        logger.info(f"Ensuring database schema at {db_path}")
        from htmlgraph.db.schema import HtmlGraphDB

        db = HtmlGraphDB(db_path)
        db.connect()
        db.create_tables()
        db.disconnect()
        if not table_names:
            logger.info("Database schema created successfully")
        else:
            logger.debug(
                "Database schema verified and migrated (existing tables: %s)",
                table_names,
            )

        conn.close()

    except sqlite3.Error as e:
        logger.warning(f"Database check warning: {e}")
        # Try to create anyway
        try:
            from htmlgraph.db.schema import HtmlGraphDB

            db = HtmlGraphDB(db_path)
            db.connect()
            db.create_tables()
            db.disconnect()
        except Exception as create_error:
            logger.error(f"Failed to create database: {create_error}")
            raise


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None, None]:
    """Manage application lifespan (startup and shutdown).

    Args:
        app: FastAPI application instance.

    Yields:
        Control to application.
    """
    # Startup
    logger.info("HtmlGraph API starting up...")

    # Initialize cache backend (in-memory by default)
    redis_url = getattr(app.state, "redis_url", None)
    await init_cache_backend(redis_url)
    logger.info("Cache backend initialized")

    # Initialize database manager
    db_manager: DatabaseManager = app.state.db_manager
    db_manager.initialize()
    logger.info("Database manager initialized")

    yield

    # Shutdown
    logger.info("HtmlGraph API shutting down...")
    await db_manager.close()
    logger.info("Cleanup complete")


def get_app(db_path: str, redis_url: str | None = None) -> FastAPI:
    """Create and configure FastAPI application.

    Args:
        db_path: Path to SQLite database file.
        redis_url: Optional Redis URL for cache backend.

    Returns:
        Configured FastAPI application instance.
    """
    # Ensure database is initialized
    _ensure_database_initialized(db_path)

    # Create app with lifespan handler
    app = FastAPI(
        title="HtmlGraph Dashboard API",
        description="Real-time agent observability dashboard",
        version="0.1.0",
    )

    # Configure app state
    app.state.db_path = db_path
    app.state.redis_url = redis_url
    app.state.query_cache = QueryCache(ttl_seconds=1.0)  # Short TTL for real-time data
    app.state.db_manager = DatabaseManager(db_path)

    # Setup Jinja2 templates
    app.state.templates = create_jinja_environment()

    # Mount static files
    static_dir = Path(__file__).parent / "static"
    if static_dir.exists():
        app.mount("/static", StaticFiles(directory=str(static_dir)), name="static")
        logger.info(f"Mounted static files from {static_dir}")

    # Register lifespan
    app.router.lifespan_context = lifespan

    # TODO: Register routers here as they are created in Phase 2+
    # from htmlgraph.api.routers import activity, features, sessions, orchestration
    # app.include_router(activity.router)
    # app.include_router(features.router)
    # app.include_router(sessions.router)
    # app.include_router(orchestration.router)

    logger.info("FastAPI application created and configured")

    return app

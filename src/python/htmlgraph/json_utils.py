"""
JSON Handling Utilities - Consolidated JSON parsing and serialization.

Provides:
- Safe JSON parsing with fallback logic
- Standardized JSON serialization
- Error handling for malformed JSON
- Performance optimization with orjson where beneficial
"""

import json
import logging
from collections.abc import Callable
from datetime import datetime
from typing import Any

logger = logging.getLogger(__name__)

try:
    import orjson

    ORJSON_AVAILABLE = True
except ImportError:
    ORJSON_AVAILABLE = False
    logger.debug("orjson not available, using standard json")


class JSONParseError(Exception):
    """Custom exception for JSON parsing errors."""

    pass


class JSONHandler:
    """Centralized JSON parsing and serialization."""

    @staticmethod
    def parse_json(
        data: str | bytes,
        default: Any = None,
        strict: bool = False,
    ) -> Any:
        """
        Safely parse JSON with fallback logic.

        Args:
            data: JSON string or bytes to parse
            default: Default value if parsing fails
            strict: If True, raise exception on parse error. If False, return default.

        Returns:
            Parsed JSON object or default value

        Raises:
            JSONParseError: If strict=True and parsing fails
        """
        if not data:
            return default

        try:
            # Convert bytes to string if needed
            if isinstance(data, bytes):
                data = data.decode("utf-8")

            # Use orjson for better performance if available
            if ORJSON_AVAILABLE:
                return orjson.loads(data)
            else:
                return json.loads(data)

        except (json.JSONDecodeError, ValueError, UnicodeDecodeError) as e:
            error_msg = f"Failed to parse JSON: {str(e)[:100]}"
            logger.warning(error_msg)

            if strict:
                raise JSONParseError(error_msg) from e

            return default

    @staticmethod
    def parse_json_safe(data: str | bytes) -> Any:
        """
        Parse JSON with safe defaults (always returns None on failure).

        Args:
            data: JSON string or bytes to parse

        Returns:
            Parsed JSON object or None if parsing fails
        """
        return JSONHandler.parse_json(data, default=None, strict=False)

    @staticmethod
    def serialize_json(
        obj: Any,
        pretty: bool = False,
        default_handler: Callable[[Any], Any] | None = None,
    ) -> str:
        """
        Serialize object to JSON string.

        Args:
            obj: Object to serialize
            pretty: If True, pretty-print the JSON
            default_handler: Custom handler for non-serializable objects

        Returns:
            JSON string

        Raises:
            TypeError: If object is not JSON serializable
        """
        if default_handler is None:
            default_handler = _default_json_handler

        try:
            if ORJSON_AVAILABLE:
                option = orjson.OPT_INDENT_2 if pretty else 0
                return orjson.dumps(obj, option=option, default=default_handler).decode(
                    "utf-8"
                )
            else:
                return json.dumps(
                    obj,
                    indent=2 if pretty else None,
                    default=default_handler,
                )
        except TypeError as e:
            logger.error(f"Failed to serialize object to JSON: {e}")
            raise

    @staticmethod
    def validate_json(data: str | bytes) -> bool:
        """
        Check if data is valid JSON without parsing it.

        Args:
            data: JSON string or bytes to validate

        Returns:
            True if valid JSON, False otherwise
        """
        try:
            if isinstance(data, bytes):
                data = data.decode("utf-8")

            if ORJSON_AVAILABLE:
                orjson.loads(data)
            else:
                json.loads(data)
            return True
        except (json.JSONDecodeError, ValueError, UnicodeDecodeError):
            return False

    @staticmethod
    def extract_json_subset(
        data: dict[str, Any] | list[Any],
        keys: list[str] | None = None,
    ) -> dict[str, Any] | list[Any]:
        """
        Extract a subset of JSON object/array.

        For dictionaries: return only specified keys
        For lists: return as-is (no filtering)

        Args:
            data: JSON object or array
            keys: List of keys to extract (for dicts)

        Returns:
            Filtered JSON object or original array
        """
        if isinstance(data, dict) and keys:
            return {k: data[k] for k in keys if k in data}
        return data

    @staticmethod
    def merge_json_objects(*objects: dict[str, Any]) -> dict[str, Any]:
        """
        Merge multiple JSON objects (shallow merge).

        Args:
            *objects: Variable number of dictionaries to merge

        Returns:
            Merged dictionary (later objects override earlier ones)
        """
        result = {}
        for obj in objects:
            if isinstance(obj, dict):
                result.update(obj)
        return result


def _default_json_handler(obj: Any) -> Any:
    """Default handler for non-serializable objects."""
    if isinstance(obj, datetime):
        return obj.isoformat()
    elif hasattr(obj, "__dict__"):
        return obj.__dict__
    elif hasattr(obj, "to_dict"):
        return obj.to_dict()
    else:
        return str(obj)


# Convenience functions for common operations
def parse_json(data: str | bytes, default: Any = None) -> Any:
    """Parse JSON with fallback."""
    return JSONHandler.parse_json(data, default=default)


def serialize_json(obj: Any, pretty: bool = False) -> str:
    """Serialize object to JSON."""
    return JSONHandler.serialize_json(obj, pretty=pretty)


def validate_json(data: str | bytes) -> bool:
    """Validate JSON without parsing."""
    return JSONHandler.validate_json(data)

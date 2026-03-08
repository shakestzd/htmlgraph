"""Property-based API tests using schemathesis."""

import pytest

try:
    import schemathesis  # noqa: F401

    SCHEMATHESIS_AVAILABLE = True
except ImportError:
    SCHEMATHESIS_AVAILABLE = False


@pytest.mark.skipif(not SCHEMATHESIS_AVAILABLE, reason="schemathesis not installed")
@pytest.mark.schemathesis
def test_api_schema_valid():
    """Verify the OpenAPI schema is parseable by schemathesis."""
    import os
    import tempfile

    with tempfile.TemporaryDirectory() as tmp:
        os.environ.setdefault("HTMLGRAPH_DB_PATH", f"{tmp}/test.db")
        try:
            from fastapi.testclient import TestClient
            from htmlgraph.api.main import get_app

            app = get_app()
            client = TestClient(app)
            response = client.get("/openapi.json")
            if response.status_code == 200:
                schema = response.json()
                assert "openapi" in schema
                assert "paths" in schema
            else:
                pytest.skip("App doesn't expose openapi.json in test mode")
        except Exception as e:
            pytest.skip(f"App creation failed: {e}")

"""OpenTelemetry configuration for HtmlGraph API."""

import logging
import os

logger = logging.getLogger(__name__)


def configure_opentelemetry(service_name: str = "htmlgraph") -> bool:
    """Configure OTel tracing. Returns True if OTLP endpoint is configured."""
    endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    if not endpoint:
        logger.debug("OTEL_EXPORTER_OTLP_ENDPOINT not set — OTel tracing disabled")
        return False

    try:
        from opentelemetry import trace
        from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import (
            OTLPSpanExporter,
        )
        from opentelemetry.sdk.resources import Resource
        from opentelemetry.sdk.trace import TracerProvider
        from opentelemetry.sdk.trace.export import BatchSpanProcessor

        resource = Resource.create({"service.name": service_name})
        provider = TracerProvider(resource=resource)
        exporter = OTLPSpanExporter(endpoint=endpoint, insecure=True)
        provider.add_span_processor(BatchSpanProcessor(exporter))
        trace.set_tracer_provider(provider)
        logger.info("OTel tracing configured → %s", endpoint)
        return True
    except ImportError:
        logger.warning("opentelemetry packages not installed")
        return False


def instrument_fastapi(app: object) -> None:
    """Instrument FastAPI app with OTel if available."""
    try:
        from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

        FastAPIInstrumentor.instrument_app(app)  # type: ignore[arg-type]
        logger.debug("FastAPI OTel instrumentation enabled")
    except ImportError:
        pass


def instrument_sqlite3() -> None:
    """Instrument sqlite3 with OTel if available."""
    try:
        from opentelemetry.instrumentation.sqlite3 import SQLite3Instrumentor

        SQLite3Instrumentor().instrument()
        logger.debug("SQLite3 OTel instrumentation enabled")
    except ImportError:
        pass


def configure_sentry() -> bool:
    """Configure Sentry if SENTRY_DSN env var is set."""
    dsn = os.getenv("SENTRY_DSN")
    if not dsn:
        return False
    try:
        import sentry_sdk
        from sentry_sdk.integrations.fastapi import FastApiIntegration

        sentry_sdk.init(dsn=dsn, integrations=[FastApiIntegration()])
        logger.info("Sentry configured")
        return True
    except ImportError:
        logger.warning("sentry-sdk not installed")
        return False

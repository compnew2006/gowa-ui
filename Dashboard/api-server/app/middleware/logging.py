"""
Structured JSON logging and request correlation middleware.

Configures logging with JSON formatter for production.
Adds correlation ID middleware and request logging.
"""
import json
import time
import uuid
import logging

from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response


class JSONFormatter(logging.Formatter):
    """Structured JSON log formatter."""

    def format(self, record: logging.LogRecord) -> str:
        log_entry = {
            "timestamp": self.formatTime(record, self.datefmt),
            "level": record.levelname,
            "logger": record.name,
            "message": record.getMessage(),
        }
        if hasattr(record, "correlation_id"):
            log_entry["correlation_id"] = record.correlation_id
        if record.exc_info and record.exc_info[1]:
            log_entry["exception"] = self.formatException(record.exc_info)
        # Include any extra fields passed via logging extras
        for key in ("method", "path", "status_code", "duration_ms", "page_id", "user_id"):
            value = getattr(record, key, None)
            if value is not None:
                log_entry[key] = value
        return json.dumps(log_entry, ensure_ascii=False)


def configure_logging(environment: str = "development") -> None:
    """
    Configure structured logging.

    Uses JSON formatter in production/staging, default formatter in development.
    """
    root_logger = logging.getLogger()
    root_logger.setLevel(logging.INFO)

    # Remove existing handlers to avoid duplicates
    root_logger.handlers.clear()

    handler = logging.StreamHandler()
    if environment in ("production", "staging"):
        handler.setFormatter(JSONFormatter(datefmt="%Y-%m-%dT%H:%M:%S"))
    else:
        handler.setFormatter(
            logging.Formatter(
                "[%(asctime)s] %(levelname)s %(name)s: %(message)s",
                datefmt="%Y-%m-%dT%H:%M:%S",
            )
        )
    root_logger.addHandler(handler)

    # Quiet down noisy libraries
    logging.getLogger("uvicorn.access").setLevel(logging.WARNING)
    logging.getLogger("sqlalchemy.engine").setLevel(logging.WARNING)


class CorrelationIDMiddleware(BaseHTTPMiddleware):
    """Generate a unique correlation ID for each request."""

    async def dispatch(self, request: Request, call_next):
        correlation_id = request.headers.get("X-Correlation-ID", str(uuid.uuid4()))
        request.state.correlation_id = correlation_id
        response = await call_next(request)
        response.headers["X-Correlation-ID"] = correlation_id
        return response


class RequestLoggingMiddleware(BaseHTTPMiddleware):
    """Log method, path, status, and duration for each request."""

    async def dispatch(self, request: Request, call_next):
        start_time = time.monotonic()
        response: Response = await call_next(request)
        duration_ms = round((time.monotonic() - start_time) * 1000, 2)

        logger = logging.getLogger("app.request")
        extra = {
            "method": request.method,
            "path": request.url.path,
            "status_code": response.status_code,
            "duration_ms": duration_ms,
            "correlation_id": getattr(request.state, "correlation_id", None),
        }

        # Skip health check noise
        if request.url.path == "/api/health":
            return response

        if response.status_code >= 500:
            logger.error("Request completed", extra=extra)
        elif response.status_code >= 400:
            logger.warning("Request completed", extra=extra)
        else:
            logger.info("Request completed", extra=extra)

        return response

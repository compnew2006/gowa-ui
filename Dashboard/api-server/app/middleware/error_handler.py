from fastapi import Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException
import json
import logging

logger = logging.getLogger(__name__)


async def http_exception_handler(request: Request, exc: StarletteHTTPException):
    """Handle HTTP exceptions with proper error responses."""
    return JSONResponse(
        status_code=exc.status_code,
        content={"detail": exc.detail},
    )


async def validation_exception_handler(request: Request, exc: RequestValidationError):
    """Handle request validation errors with 400 status code."""
    errors = []
    for error in exc.errors():
        errors.append({
            "field": ".".join(str(loc) for loc in error["loc"] if loc != "body"),
            "message": error["msg"],
            "type": error["type"],
        })

    logger.warning(f"Validation error on {request.url.path}: {errors}")

    return JSONResponse(
        status_code=status.HTTP_400_BAD_REQUEST,
        content={
            "detail": "Validation error",
            "errors": errors,
        },
    )


async def json_decode_exception_handler(request: Request, exc: json.JSONDecodeError):
    """Handle JSON decode errors with 400 status code."""
    logger.warning(f"Invalid JSON on {request.url.path}: {exc.msg}")

    return JSONResponse(
        status_code=status.HTTP_400_BAD_REQUEST,
        content={"detail": "Invalid JSON format"},
    )


async def generic_exception_handler(request: Request, exc: Exception):
    """Handle unexpected exceptions with 500 status code."""
    logger.error(f"Unexpected error on {request.url.path}: {exc}", exc_info=True)

    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={"detail": "Internal server error"},
    )

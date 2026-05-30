"""
PII Detection & Masking Service.

Detects and masks PII in AI replies before they are sent:
  - Phone numbers (Arabic/international formats)
  - Email addresses
  - Credit card numbers
  - National ID / SSN patterns
  - Arabic/international bank account patterns
"""
from __future__ import annotations
import re
import logging
from typing import NamedTuple

logger = logging.getLogger(__name__)


class PiiResult(NamedTuple):
    detected: bool
    masked_text: str
    pii_types: list[str]


# Pattern registry: (name, regex, replacement)
_PATTERNS: list[tuple[str, re.Pattern, str]] = [
    (
        "phone",
        re.compile(
            r"(?<!\d)"
            r"(\+?966|00966|05|01|07|\+2|002)?"
            r"[\s\-\.]?"
            r"[0-9]{3}[\s\-\.]?[0-9]{3,4}[\s\-\.]?[0-9]{3,4}"
            r"(?!\d)",
            re.IGNORECASE,
        ),
        "[رقم الهاتف محذوف]",
    ),
    (
        "email",
        re.compile(r"[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}"),
        "[البريد الإلكتروني محذوف]",
    ),
    (
        "credit_card",
        re.compile(r"\b(?:\d[ \-]?){13,19}\b"),
        "[رقم البطاقة محذوف]",
    ),
    (
        "national_id",
        re.compile(r"\b[12]\d{9}\b"),  # Saudi National ID (10 digits starting 1 or 2)
        "[رقم الهوية محذوف]",
    ),
    (
        "iban",
        re.compile(r"\bSA\d{2}[\s]?\d{4}[\s]?\d{4}[\s]?\d{4}[\s]?\d{4}[\s]?\d{4}\b", re.IGNORECASE),
        "[رقم الحساب محذوف]",
    ),
]


def scan_text(text: str) -> PiiResult:
    """
    Scan text for PII and return masked version.
    Returns PiiResult(detected, masked_text, pii_types).
    """
    if not text:
        return PiiResult(False, text, [])

    masked = text
    found_types: list[str] = []

    for name, pattern, replacement in _PATTERNS:
        new_masked, count = pattern.subn(replacement, masked)
        if count > 0:
            found_types.append(name)
            masked = new_masked

    return PiiResult(
        detected=len(found_types) > 0,
        masked_text=masked,
        pii_types=found_types,
    )


def mask_reply(reply: str) -> tuple[str, bool]:
    """
    Mask PII in an AI reply before it is sent.
    Returns (masked_reply, pii_was_detected).
    """
    result = scan_text(reply)
    if result.detected:
        logger.info("[PII] Masked types: %s", result.pii_types)
    return result.masked_text, result.detected


def scan_comment(comment: str) -> tuple[bool, list[str]]:
    """
    Scan an incoming comment for PII (for compliance audit).
    Returns (detected, pii_types).
    """
    result = scan_text(comment)
    return result.detected, result.pii_types

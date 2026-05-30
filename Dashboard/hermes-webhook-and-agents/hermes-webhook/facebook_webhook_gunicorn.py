#!/usr/bin/env python3
"""
Production Facebook & Instagram Webhook Server for Hermes
Run with: gunicorn --bind 0.0.0.0:5000 --workers 4 facebook_webhook_gunicorn:app

Supports:
- Facebook Page comments
- Instagram Business comments (connected to Facebook Pages)
- Multi-page/location support
- Shared knowledge base between Facebook and Instagram
"""

import fcntl
from facebook_db import get_page_db
import logging
import json
import os
from pathlib import Path
import re
import shutil
import hashlib
import hmac
from string import Formatter
import subprocess
import sys
import threading
import time
import uuid
from datetime import datetime, timedelta, timezone

from flask import Flask, jsonify, request
import requests
import redis

# Import sibling modules safely in both /opt deployment and local workspace runs.
CURRENT_DIR = Path(__file__).resolve().parent
if str(CURRENT_DIR) not in sys.path:
    sys.path.insert(0, str(CURRENT_DIR))

from notification_system import notification_system
from responder import UNKNOWN_RESPONSES, build_response, detect_language
from runtime_paths import (
    get_interaction_log_dir,
    get_log_dir,
    get_pages_config_file,
    is_llm_rephrase_enabled,
    is_test_mode,
    resolve_configured_data_path,
)


def configure_logging():
    handlers = [logging.StreamHandler()]
    try:
        handlers.insert(0, logging.FileHandler(get_log_dir() / "facebook_webhook.log"))
    except OSError as exc:
        handlers = [logging.StreamHandler()]
        logging.basicConfig(level=logging.INFO, force=True)
        logging.getLogger(__name__).warning("Using stdout logging only: %s", exc)

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        handlers=handlers,
        force=True,
    )


configure_logging()
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Configuration
GRAPH_URL = "https://graph.facebook.com/v19.0"
ADMIN_TOKEN = os.environ.get("HERMES_ADMIN_TOKEN", "")
HERMES_API_URL = os.environ.get("HERMES_API_URL", "http://localhost:8080")
REDIS_URL = os.environ.get("REDIS_URL", "redis://localhost:6379/0")

VERIFY_TOKEN = os.environ.get("FB_WEBHOOK_VERIFY_TOKEN", "")
APP_SECRET = os.environ.get("FB_APP_SECRET", "")

if not VERIFY_TOKEN:
    raise RuntimeError("FB_WEBHOOK_VERIFY_TOKEN must be set")
if not APP_SECRET:
    logger = logging.getLogger(__name__)
    logger.error(
        "SECURITY WARNING: FB_APP_SECRET is not set. "
        "Webhook will REJECT all incoming events until FB_APP_SECRET is configured in .env. "
        "Get your App Secret from https://developers.facebook.com/apps/ -> Settings -> Basic"
    )


def sanitize_for_log(data):
    if isinstance(data, dict):
        redacted = {}
        for k, v in data.items():
            kl = k.lower()
            if any(t in kl for t in ("token", "secret", "password", "key", "access")):
                val = str(v)
                redacted[k] = val[:4] + "..." if len(val) > 4 else "***"
            else:
                redacted[k] = sanitize_for_log(v)
        return redacted
    if isinstance(data, str):
        for pattern in (r"EAA[a-zA-Z0-9]+", r"[0-9]{10,}_[a-zA-Z0-9]+"):
            data = re.sub(pattern, "***REDACTED***", data)
        return data
    return data
ALLOWED_PRIVATE_REPLY_QUESTION_TYPES = {"pricing", "location", "hours", "services", "contact", "greeting"}
DEFAULT_PRIVATE_REPLY_QUESTION_TYPES = ["pricing", "location", "hours", "services", "contact", "greeting"]
ALLOWED_TEMPLATE_FIELDS = {"customer_name", "public_reply"}
DEFAULT_HUMAN_REPLY_LEARNING_MAX_AGE_DAYS = 90
EXACT_NORMALIZED_MATCH_MODE = "exact_normalized"
DEFAULT_PUBLIC_WEBHOOK_BASE_URL = os.environ.get("HERMES_PUBLIC_WEBHOOK_BASE_URL", "https://fbwebhook.ofuqalmadenah.com")
ARABIC_DIACRITICS_PATTERN = re.compile(r"[\u064B-\u065F]")
UNKNOWN_FALLBACK_RESPONSES = frozenset(UNKNOWN_RESPONSES.values())
LEARNED_REPLY_SOURCES = {"unknown_fallback_learning", "telegram_teach"}
LEARNED_REPLY_SOURCE_SECTIONS = {
    "unknown_fallback_learning": "Human Reply Learning",
    "telegram_teach": "Telegram Teach",
}
LEARNED_REPLY_PUBLIC_SOURCES = set(LEARNED_REPLY_SOURCE_SECTIONS.values())
SUSPICIOUS_REPLY_PATTERNS = (
    re.compile(r"https?://|www\.", re.IGNORECASE),
    re.compile(r"\b(?:\+?\d[\d\-\s]{8,}\d)\b"),
    re.compile(r"\b(احتيال|محتال|نصاب|نصابين|scam|fraud|fake|cheat|steal|thief)\b", re.IGNORECASE),
    re.compile(r"(?:\bلا تتعامل\b|\bلا تشتري\b|\bdon't buy\b|\bdon't deal\b)", re.IGNORECASE),
)
CASE_SPECIFIC_REPLY_PATTERNS = (
    re.compile(r"\b(سنرسل|سنتواصل|سنقوم|سوف نرسل|ساعطيك|سأرسل|سنعطي)\b", re.IGNORECASE),
    re.compile(r"\b(طلبك|حالتك|رقم طلبك|شحنتك|حسابك|رصيدك|اشتراكك)\b", re.IGNORECASE),
    re.compile(r"\b(we'll send|we'll contact|we will send|we will contact|i'll send|we'll give)\b", re.IGNORECASE),
    re.compile(r"\b(your order|your case|ticket|your shipment|your account|your balance|your subscription)\b", re.IGNORECASE),
)
TELEGRAM_TEACH_COMMAND_RE = re.compile(r"^/teach(?:@[A-Za-z0-9_]+)?(?:\s+|$)", re.IGNORECASE)
TELEGRAM_FIELD_PATTERNS = (
    ("question", re.compile(r"^(?:q|question|word|message|msg|س|سؤال)\s*:\s*(.+)$", re.IGNORECASE)),
    ("answer", re.compile(r"^(?:a|answer|reply|response|رد|ج|جواب)\s*:\s*(.+)$", re.IGNORECASE)),
)
DEFAULT_PRIVATE_REPLY_TEMPLATES = {
    "pricing": {
        "en": "Hi {customer_name}, thanks for your comment. {public_reply} Send us a message here if you'd like to continue.",
        "ar": "أهلاً {customer_name}، شكراً على تعليقك. {public_reply} ابعت لنا رسالة هنا لو تحب نكمل التفاصيل.",
    },
    "location": {
        "en": "Hi {customer_name}, thanks for your comment. {public_reply} Message us here if you'd like directions or to continue.",
        "ar": "أهلاً {customer_name}، شكراً على تعليقك. {public_reply} ابعت لنا رسالة هنا لو تحب الموقع أو نكمل التفاصيل.",
    },
    "hours": {
        "en": "Hi {customer_name}, thanks for your comment. {public_reply} Message us here if you'd like anything else.",
        "ar": "أهلاً {customer_name}، شكراً على تعليقك. {public_reply} ابعت لنا رسالة هنا لو تحتاج أي تفاصيل إضافية.",
    },
    "services": {
        "en": "Hi {customer_name}, thanks for your comment. {public_reply} Send us a message here if you'd like to discuss your project.",
        "ar": "أهلاً {customer_name}، شكراً على تعليقك. {public_reply} ابعت لنا رسالة هنا لو تحب نتكلم عن مشروعك.",
    },
    "contact": {
        "en": "Hi {customer_name}, thanks for your comment. {public_reply} You can continue with us here in Messenger anytime.",
        "ar": "أهلاً {customer_name}، شكراً على تعليقك. {public_reply} تقدر تكمل معنا هنا على ماسنجر في أي وقت.",
    },
    "greeting": {
        "en": "Hi {customer_name}, thanks for your comment. {public_reply} Send us a message here if you'd like to continue.",
        "ar": "أهلاً {customer_name}، شكراً على تعليقك. {public_reply} ابعت لنا رسالة هنا لو تحب نكمل على الخاص.",
    },
}

# Track replied comments to prevent duplicates
REPLIED_COMMENTS = set()
PRIVATE_REPLIED_COMMENTS = set()
COMMENT_CONTEXTS = {}
LEARNED_REPLY_RECORDS = {}
ACTIVE_LEARNED_REPLIES = {}
AUTO_PUBLIC_REPLY_SIGNATURES = set()
LOADED_LEARNING_PAGES = set()
LEARNING_STATE_SIGNATURES = {}

_redis_client = None
_redis_lock = threading.Lock()


def get_redis():
    global _redis_client
    if _redis_client is not None:
        return _redis_client
    with _redis_lock:
        if _redis_client is not None:
            return _redis_client
        try:
            client = redis.from_url(REDIS_URL, socket_timeout=5, socket_connect_timeout=5)
            client.ping()
            _redis_client = client
            logger.info("Connected to Redis for dedup store")
            return _redis_client
        except Exception as exc:
            logger.warning("Redis unavailable, falling back to in-memory dedup: %s", exc)
            return None


def _redis_set_add(key, value):
    r = get_redis()
    if r is not None:
        try:
            r.sadd(key, value)
            r.expire(key, 86400 * 30)
            return
        except Exception:
            pass
    if key.endswith(":private"):
        PRIVATE_REPLIED_COMMENTS.add(value)
    else:
        REPLIED_COMMENTS.add(value)


def _redis_set_contains(key, value):
    r = get_redis()
    if r is not None:
        try:
            return bool(r.sismember(key, value))
        except Exception:
            pass
    if key.endswith(":private"):
        return value in PRIVATE_REPLIED_COMMENTS
    return value in REPLIED_COMMENTS


def get_webhook_log_file():
    return get_log_dir() / "webhook_events.log"


def get_replied_comments_file():
    return get_log_dir() / "replied_comments.txt"


def get_private_replied_comments_file():
    return get_log_dir() / "private_replied_comments.txt"


def get_comment_context_file(page_id):
    return get_interaction_log_dir() / f"{page_id}_comment_contexts.jsonl"


def get_learned_replies_file(page_id):
    return get_interaction_log_dir() / f"{page_id}_learned_replies.jsonl"


def get_auto_public_reply_file(page_id):
    return get_interaction_log_dir() / f"{page_id}_auto_public_replies.jsonl"

SAFE_ID_RE = re.compile(r"^[A-Za-z0-9_-]{1,128}$")


def validate_id(value, name="id"):
    if not value or not SAFE_ID_RE.fullmatch(str(value)):
        logger.warning("Invalid %s rejected: %s", name, repr(value)[:40])
        return False
    return True


def load_pages_config():
    pages_config_file = get_pages_config_file()
    try:
        with open(pages_config_file, "r", encoding="utf-8") as f:
            config = json.load(f)
            logger.info("Loaded %d pages", len(config.get('pages', {})))
            return config
    except Exception as e:
        logger.error("Failed to load pages config: %s", e)
        return {"pages": {}}

def get_page_config(page_id):
    if not validate_id(page_id, "page_id"):
        return None
    pages_config = load_pages_config()
    page_config = pages_config.get("pages", {}).get(page_id)
    if not page_config:
        return None

    resolved_config = dict(page_config)
    if resolved_config.get("knowledge_file"):
        resolved_config["knowledge_file"] = str(
            resolve_configured_data_path(resolved_config["knowledge_file"])
        )
    return resolved_config

# Load already replied comments on startup
try:
    replied_comments_file = get_replied_comments_file()
    if replied_comments_file.exists():
        with open(replied_comments_file, "r", encoding="utf-8") as f:
            REPLIED_COMMENTS = set(line.strip() for line in f if line.strip())
        logger.info(f"Loaded {len(REPLIED_COMMENTS)} previously replied comments")
except Exception as e:
    logger.error(f"Failed to load replied comments: {e}")
    REPLIED_COMMENTS = set()

try:
    private_replied_comments_file = get_private_replied_comments_file()
    if private_replied_comments_file.exists():
        with open(private_replied_comments_file, "r", encoding="utf-8") as f:
            PRIVATE_REPLIED_COMMENTS = set(line.strip() for line in f if line.strip())
        logger.info(f"Loaded {len(PRIVATE_REPLIED_COMMENTS)} previously private-replied comments")
except Exception as e:
    logger.error(f"Failed to load private replied comments: {e}")
    PRIVATE_REPLIED_COMMENTS = set()

def has_already_replied(comment_id):
    r = get_redis()
    if r is not None:
        try:
            return bool(r.sismember("hermes:replied_comments", comment_id))
        except Exception:
            pass
    return comment_id in REPLIED_COMMENTS

def mark_as_replied(comment_id):
    REPLIED_COMMENTS.add(comment_id)
    r = get_redis()
    if r is not None:
        try:
            r.sadd("hermes:replied_comments", comment_id)
            return
        except Exception:
            pass
    try:
        with open(get_replied_comments_file(), "a", encoding="utf-8") as f:
            fcntl.flock(f.fileno(), fcntl.LOCK_EX)
            f.write(f"{comment_id}\n")
            fcntl.flock(f.fileno(), fcntl.LOCK_UN)
    except Exception as e:
        logger.error(f"Failed to save replied comment: {e}")


def has_private_reply(comment_id):
    r = get_redis()
    if r is not None:
        try:
            return bool(r.sismember("hermes:private_replied_comments", comment_id))
        except Exception:
            pass
    return comment_id in PRIVATE_REPLIED_COMMENTS


def mark_private_reply(comment_id):
    PRIVATE_REPLIED_COMMENTS.add(comment_id)
    r = get_redis()
    if r is not None:
        try:
            r.sadd("hermes:private_replied_comments", comment_id)
            return
        except Exception:
            pass
    try:
        with open(get_private_replied_comments_file(), "a", encoding="utf-8") as f:
            fcntl.flock(f.fileno(), fcntl.LOCK_EX)
            f.write(f"{comment_id}\n")
            fcntl.flock(f.fileno(), fcntl.LOCK_UN)
    except Exception as e:
        logger.error(f"Failed to save private replied comment: {e}")


def utc_now():
    return datetime.now(timezone.utc)


def datetime_to_iso(value):
    return value.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")


def parse_datetime(value):
    if not value:
        return None
    normalized = value
    if normalized.endswith("Z"):
        normalized = normalized[:-1] + "+00:00"
    try:
        return datetime.fromisoformat(normalized)
    except ValueError:
        return None


def get_human_reply_learning_settings(page_config):
    max_age_days = page_config.get("human_reply_learning_max_age_days", DEFAULT_HUMAN_REPLY_LEARNING_MAX_AGE_DAYS)
    try:
        max_age_days = int(max_age_days)
    except (TypeError, ValueError):
        max_age_days = DEFAULT_HUMAN_REPLY_LEARNING_MAX_AGE_DAYS
    if max_age_days <= 0:
        max_age_days = DEFAULT_HUMAN_REPLY_LEARNING_MAX_AGE_DAYS

    match_mode = page_config.get("human_reply_learning_match_mode", EXACT_NORMALIZED_MATCH_MODE)
    if match_mode != EXACT_NORMALIZED_MATCH_MODE:
        match_mode = EXACT_NORMALIZED_MATCH_MODE

    return {
        "enabled": bool(page_config.get("human_reply_learning_enabled", False)),
        "max_age_days": max_age_days,
        "match_mode": match_mode,
    }


def normalize_learning_text(text):
    normalized = (text or "").strip().lower()
    for source, target in (
        ("أ", "ا"),
        ("إ", "ا"),
        ("آ", "ا"),
        ("ة", "ه"),
        ("ى", "ي"),
    ):
        normalized = normalized.replace(source, target)
    normalized = ARABIC_DIACRITICS_PATTERN.sub("", normalized)
    normalized = re.sub(r"[^\w\s]", " ", normalized, flags=re.UNICODE)
    normalized = normalized.replace("_", " ")
    return " ".join(normalized.split())


def is_exact_unknown_fallback(reply_text):
    return (reply_text or "").strip() in UNKNOWN_FALLBACK_RESPONSES


def is_learned_reply_expired(payload, reference_time=None):
    expires_at = parse_datetime(payload.get("expires_at"))
    if expires_at is None:
        return True
    now = reference_time or utc_now()
    return expires_at <= now


def contains_suspicious_human_reply(reply_text):
    return any(pattern.search(reply_text or "") for pattern in SUSPICIOUS_REPLY_PATTERNS)


def is_case_specific_human_reply(reply_text):
    return any(pattern.search(reply_text or "") for pattern in CASE_SPECIFIC_REPLY_PATTERNS)


def get_public_webhook_base_url(page_config=None):
    configured = ""
    if isinstance(page_config, dict):
        configured = (page_config.get("public_webhook_base_url") or "").strip()
    return (configured or DEFAULT_PUBLIC_WEBHOOK_BASE_URL).rstrip("/")


def get_telegram_teach_settings(page_config):
    return {
        "enabled": bool((page_config or {}).get("telegram_teach_enabled", False)),
    }


def get_telegram_credentials(page_id):
    page_config = get_page_config(page_id) or {}
    notification_config = notification_system.get_page_notification_config(page_id) or {}

    telegram_token = (
        notification_config.get("telegram_bot_token")
        or page_config.get("telegram_bot_token")
        or ""
    ).strip()
    telegram_chat_id = str(
        notification_config.get("telegram_chat_id")
        or page_config.get("telegram_chat_id")
        or ""
    ).strip()
    return telegram_token, telegram_chat_id


def build_telegram_webhook_secret(page_id, telegram_token):
    raw = f"{page_id}:{telegram_token}".encode("utf-8")
    return hashlib.sha256(raw).hexdigest()[:32]


def build_telegram_webhook_url(page_id, page_config=None):
    return f"{get_public_webhook_base_url(page_config)}/telegram/webhook/{page_id}"


def parse_telegram_teach_command(text):
    raw = (text or "").strip()
    if not raw or not TELEGRAM_TEACH_COMMAND_RE.match(raw):
        return None

    body = TELEGRAM_TEACH_COMMAND_RE.sub("", raw, count=1).strip()
    if not body:
        return {"question": "", "answer": ""}

    if "=>" in body:
        question, answer = body.split("=>", 1)
        question = question.strip()
        answer = answer.strip()
        return {"question": question, "answer": answer}

    lines = [line.strip() for line in body.splitlines() if line.strip()]
    question = ""
    answer = ""
    for line in lines:
        for field_name, pattern in TELEGRAM_FIELD_PATTERNS:
            match = pattern.match(line)
            if not match:
                continue
            if field_name == "question":
                question = match.group(1).strip()
            else:
                answer = match.group(1).strip()
            break

    if question or answer:
        return {"question": question, "answer": answer}

    if len(lines) >= 2:
        return {
            "question": lines[0].strip(),
            "answer": "\n".join(lines[1:]).strip(),
        }

    return {"question": "", "answer": ""}


def get_telegram_teach_usage(page_id):
    webhook_url = build_telegram_webhook_url(page_id, get_page_config(page_id))
    return (
        "Use /teach in one of these formats:\n"
        "/teach سلام عليكم => وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك\n\n"
        "/teach\n"
        "q: سلام عليكم\n"
        "a: وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك\n\n"
        "Use /learned to list active taught phrases.\n"
        "Use /remove سلام عليكم to delete an active taught reply.\n\n"
        f"Webhook: {webhook_url}"
    )


def parse_telegram_remove_command(text):
    raw = (text or "").strip()
    if not raw:
        return ""
    for prefix in ("/remove", "/forget", "/delete"):
        if raw.lower().startswith(prefix):
            return raw[len(prefix):].strip()
    return ""


def append_jsonl(file_path, payload):
    with open(file_path, "a", encoding="utf-8") as f:
        json.dump(payload, f, ensure_ascii=False)
        f.write("\n")


def get_file_signature(file_path):
    try:
        stats = file_path.stat()
    except OSError:
        return None
    return (stats.st_mtime_ns, stats.st_size)


def get_page_learning_state_signature(page_id):
    return {
        "comment_contexts": get_file_signature(get_comment_context_file(page_id)),
        "learned_replies": get_file_signature(get_learned_replies_file(page_id)),
        "auto_public_replies": get_file_signature(get_auto_public_reply_file(page_id)),
    }


def clear_page_learning_state(page_id):
    for comment_id, payload in list(COMMENT_CONTEXTS.items()):
        if payload.get("page_id") == page_id:
            COMMENT_CONTEXTS.pop(comment_id, None)

    for signature in list(AUTO_PUBLIC_REPLY_SIGNATURES):
        if signature[0] == page_id:
            AUTO_PUBLIC_REPLY_SIGNATURES.discard(signature)

    LEARNED_REPLY_RECORDS[page_id] = {}
    ACTIVE_LEARNED_REPLIES[page_id] = {}


def rebuild_active_learned_replies(page_id):
    active_replies = {}
    now = utc_now()

    for payload in LEARNED_REPLY_RECORDS.get(page_id, {}).values():
        if payload.get("page_id") != page_id:
            continue
        if payload.get("source") not in LEARNED_REPLY_SOURCES:
            continue
        if not payload.get("is_active", True):
            continue
        normalized_message = payload.get("source_question_normalized")
        if not normalized_message:
            continue
        if is_learned_reply_expired(payload, now):
            continue

        current = active_replies.get(normalized_message)
        if current is None:
            active_replies[normalized_message] = payload
            continue

        current_time = parse_datetime(current.get("learned_at")) or datetime.min.replace(tzinfo=timezone.utc)
        payload_time = parse_datetime(payload.get("learned_at")) or datetime.min.replace(tzinfo=timezone.utc)
        if payload_time >= current_time:
            active_replies[normalized_message] = payload

    ACTIVE_LEARNED_REPLIES[page_id] = active_replies


def ensure_learning_state_loaded(page_id):
    current_signature = get_page_learning_state_signature(page_id)
    if page_id in LOADED_LEARNING_PAGES and LEARNING_STATE_SIGNATURES.get(page_id) == current_signature:
        return

    clear_page_learning_state(page_id)

    comment_context_file = get_comment_context_file(page_id)
    if comment_context_file.exists():
        try:
            with open(comment_context_file, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    payload = json.loads(line)
                    comment_id = payload.get("comment_id")
                    if comment_id:
                        COMMENT_CONTEXTS[comment_id] = payload
        except Exception as exc:
            logger.error("Failed loading comment contexts for %s: %s", page_id, exc)

    LEARNED_REPLY_RECORDS.setdefault(page_id, {})
    ACTIVE_LEARNED_REPLIES.setdefault(page_id, {})
    learned_replies_file = get_learned_replies_file(page_id)
    if learned_replies_file.exists():
        try:
            with open(learned_replies_file, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    payload = json.loads(line)
                    record_id = payload.get("id")
                    if record_id:
                        LEARNED_REPLY_RECORDS[page_id][record_id] = payload
        except Exception as exc:
            logger.error("Failed loading learned replies for %s: %s", page_id, exc)

    rebuild_active_learned_replies(page_id)

    auto_reply_file = get_auto_public_reply_file(page_id)
    if auto_reply_file.exists():
        try:
            with open(auto_reply_file, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    payload = json.loads(line)
                    parent_comment_id = payload.get("parent_comment_id")
                    normalized_reply = payload.get("normalized_reply")
                    if parent_comment_id and normalized_reply:
                        AUTO_PUBLIC_REPLY_SIGNATURES.add((page_id, parent_comment_id, normalized_reply))
        except Exception as exc:
            logger.error("Failed loading auto reply signatures for %s: %s", page_id, exc)

    LOADED_LEARNING_PAGES.add(page_id)
    LEARNING_STATE_SIGNATURES[page_id] = current_signature




def save_reply_to_sqlite(page_id, comment_id, reply_text, reply_id):
    """Save reply information to SQLite database"""
    try:
        db = get_page_db(page_id)
        db.mark_as_replied(comment_id, reply_id, reply_text, ai_model="auto", reply_type="public")
        logger.info("✅ Saved reply to SQLite for comment %s", comment_id[:20])
        return True
    except Exception as e:
        logger.error("Error saving reply to SQLite: %s", e)
        return False

def record_comment_context_and_save_reply(page_id, comment_id, message, from_name, sender_id, post_id, parent_id, reply_result, bot_reply_text):
    if not page_id or not comment_id or not message:
        return

    ensure_learning_state_loaded(page_id)
    payload = {
        "created_at": datetime_to_iso(utc_now()),
        "page_id": page_id,
        "comment_id": comment_id,
        "original_message": message,
        "normalized_message": normalize_learning_text(message),
        "customer_name": from_name,
        "sender_id": sender_id,
        "post_id": post_id,
        "parent_id": parent_id,
        "language": reply_result.language if reply_result else "",
        "question_type": reply_result.question_type if reply_result else "",
        "found_in_kb": bool(reply_result.found_in_kb) if reply_result else False,
        "bot_reply_text": bot_reply_text or "",
        "used_unknown_fallback": is_exact_unknown_fallback(bot_reply_text),
    }
    payload["message"] = payload["original_message"]
    COMMENT_CONTEXTS[comment_id] = payload
    try:
        append_jsonl(get_comment_context_file(page_id), payload)
    except Exception as exc:
        logger.error("Failed saving comment context: %s", exc)


def record_auto_public_reply_signature(page_id, parent_comment_id, reply_message):
    """Record auto reply signature (simplified - using SQLite)"""
    # This is now handled by save_reply_to_sqlite
    pass


def is_known_auto_public_reply(page_id, parent_comment_id, reply_message):
    """Check if this is a known auto reply (simplified)"""
    # This can be implemented with SQLite if needed
    return False


def build_learned_reply_payload(page_id, parent_context, parent_comment_id, reply_message, learned_from_human_id, max_age_days):
    learned_at = utc_now()
    return {
        "id": "learned_{suffix}".format(suffix=uuid.uuid4().hex),
        "page_id": page_id,
        "source_question": parent_context.get("original_message", ""),
        "source_question_normalized": parent_context.get("normalized_message", ""),
        "learned_reply": reply_message,
        "parent_comment_id": parent_comment_id,
        "learned_from_human_id": learned_from_human_id,
        "language": parent_context.get("language", ""),
        "source": "unknown_fallback_learning",
        "learned_at": datetime_to_iso(learned_at),
        "expires_at": datetime_to_iso(learned_at + timedelta(days=max_age_days)),
        "is_active": True,
        "superseded_by": None,
    }


def build_telegram_teach_payload(page_id, source_question, learned_reply, taught_by_chat_id, telegram_message_id, max_age_days):
    learned_at = utc_now()
    return {
        "id": "learned_{suffix}".format(suffix=uuid.uuid4().hex),
        "page_id": page_id,
        "source_question": source_question,
        "source_question_normalized": normalize_learning_text(source_question),
        "learned_reply": learned_reply,
        "parent_comment_id": "",
        "learned_from_human_id": str(taught_by_chat_id or ""),
        "language": detect_language(source_question),
        "source": "telegram_teach",
        "learned_at": datetime_to_iso(learned_at),
        "expires_at": datetime_to_iso(learned_at + timedelta(days=max_age_days)),
        "is_active": True,
        "superseded_by": None,
        "telegram_message_id": telegram_message_id,
    }


def supersede_active_learned_reply(page_id, normalized_message, new_record_id):
    active_payload = ACTIVE_LEARNED_REPLIES.get(page_id, {}).get(normalized_message)
    if not active_payload:
        return

    superseded_payload = dict(active_payload)
    superseded_payload["is_active"] = False
    superseded_payload["superseded_by"] = new_record_id
    LEARNED_REPLY_RECORDS.setdefault(page_id, {})[superseded_payload["id"]] = superseded_payload
    try:
        append_jsonl(get_learned_replies_file(page_id), superseded_payload)
    except Exception as exc:
        logger.error("Failed superseding learned reply: %s", exc)


def deactivate_active_learned_reply(page_id, normalized_message, removed_by="", reason="removed"):
    active_payload = ACTIVE_LEARNED_REPLIES.get(page_id, {}).get(normalized_message)
    if not active_payload:
        return False

    removed_payload = dict(active_payload)
    removed_payload["is_active"] = False
    removed_payload["superseded_by"] = reason
    removed_payload["removed_at"] = datetime_to_iso(utc_now())
    removed_payload["removed_by"] = removed_by
    LEARNED_REPLY_RECORDS.setdefault(page_id, {})[removed_payload["id"]] = removed_payload
    ACTIVE_LEARNED_REPLIES.get(page_id, {}).pop(normalized_message, None)
    try:
        append_jsonl(get_learned_replies_file(page_id), removed_payload)
    except Exception as exc:
        logger.error("Failed deactivating learned reply: %s", exc)
        return False
    return True


def list_active_learned_replies(page_id):
    ensure_learning_state_loaded(page_id)
    active = list(ACTIVE_LEARNED_REPLIES.get(page_id, {}).values())
    active.sort(
        key=lambda payload: parse_datetime(payload.get("learned_at")) or datetime.min.replace(tzinfo=timezone.utc),
        reverse=True,
    )
    return active


def learn_from_human_page_reply(page_id, comment_id, parent_comment_id, reply_message, learned_from_human_id):
    if not page_id or not parent_comment_id or not reply_message:
        return False

    page_config = get_page_config(page_id)
    if not page_config:
        return False
    settings = get_human_reply_learning_settings(page_config)
    if not settings["enabled"]:
        return False

    ensure_learning_state_loaded(page_id)
    if is_known_auto_public_reply(page_id, parent_comment_id, reply_message):
        return False

    parent_context = COMMENT_CONTEXTS.get(parent_comment_id)
    if not parent_context:
        return False
    if parent_context.get("page_id") != page_id:
        return False
    if not parent_context.get("used_unknown_fallback"):
        return False
    if not is_exact_unknown_fallback(parent_context.get("bot_reply_text", "")):
        return False
    if contains_suspicious_human_reply(reply_message):
        return False
    if is_case_specific_human_reply(reply_message):
        return False

    normalized_message = parent_context.get("normalized_message") or normalize_learning_text(parent_context.get("original_message", ""))
    if not normalized_message:
        return False

    payload = build_learned_reply_payload(
        page_id=page_id,
        parent_context=parent_context,
        parent_comment_id=parent_comment_id,
        reply_message=reply_message,
        learned_from_human_id=learned_from_human_id,
        max_age_days=settings["max_age_days"],
    )
    supersede_active_learned_reply(page_id, normalized_message, payload["id"])
    LEARNED_REPLY_RECORDS.setdefault(page_id, {})[payload["id"]] = payload
    ACTIVE_LEARNED_REPLIES.setdefault(page_id, {})[normalized_message] = payload
    try:
        append_jsonl(get_learned_replies_file(page_id), payload)
    except Exception as exc:
        logger.error("Failed saving learned reply: %s", exc)
        return False

    logger.info("Learned human reply for '%s' on page %s", parent_context.get("original_message", ""), page_id)
    try:
        notification_system.notify_learning(
            page_id,
            parent_context.get("original_message", ""),
            reply_message,
            learned_from_human_id=learned_from_human_id,
            parent_comment_id=parent_comment_id,
        )
    except Exception as exc:
        logger.error("Failed learned-reply notification: %s", exc)
    return True


def learn_from_telegram_teach(page_id, source_question, learned_reply, taught_by_chat_id, telegram_message_id=""):
    if not page_id or not source_question or not learned_reply:
        return False, "Both question and answer are required."

    page_config = get_page_config(page_id)
    if not page_config:
        return False, "Page config was not found."

    teach_settings = get_telegram_teach_settings(page_config)
    if not teach_settings["enabled"]:
        return False, "Telegram teaching is disabled for this page."

    source_question = source_question.strip()
    learned_reply = learned_reply.strip()
    normalized_message = normalize_learning_text(source_question)
    if not normalized_message:
        return False, "The taught question is empty after normalization."
    if not learned_reply:
        return False, "The taught answer is empty."
    if is_exact_unknown_fallback(learned_reply):
        return False, "The fallback reply cannot be taught as a learned answer."

    settings = get_human_reply_learning_settings(page_config)
    ensure_learning_state_loaded(page_id)

    payload = build_telegram_teach_payload(
        page_id=page_id,
        source_question=source_question,
        learned_reply=learned_reply,
        taught_by_chat_id=taught_by_chat_id,
        telegram_message_id=telegram_message_id,
        max_age_days=settings["max_age_days"],
    )
    supersede_active_learned_reply(page_id, normalized_message, payload["id"])
    LEARNED_REPLY_RECORDS.setdefault(page_id, {})[payload["id"]] = payload
    ACTIVE_LEARNED_REPLIES.setdefault(page_id, {})[normalized_message] = payload
    try:
        append_jsonl(get_learned_replies_file(page_id), payload)
    except Exception as exc:
        logger.error("Failed saving Telegram-taught reply: %s", exc)
        return False, "Failed to save the taught reply."

    logger.info("Telegram taught reply for '%s' on page %s", source_question, page_id)
    return True, "Learned reply saved and active now."


def get_active_learned_reply_payload(page_id, message):
    ensure_learning_state_loaded(page_id)
    normalized_message = normalize_learning_text(message)
    if not normalized_message:
        return None

    payload = ACTIVE_LEARNED_REPLIES.get(page_id, {}).get(normalized_message)
    if not payload:
        return None
    if is_learned_reply_expired(payload):
        ACTIVE_LEARNED_REPLIES.get(page_id, {}).pop(normalized_message, None)
        return None
    return payload


def get_learned_human_reply(page_id, message):
    payload = get_active_learned_reply_payload(page_id, message)
    if not payload:
        return None
    return payload.get("learned_reply")

def save_interaction_for_learning(page_id, commenter_name, comment, reply):
    """Save interaction to an append-only learning log outside the KB."""
    try:
        interaction_file = get_interaction_log_dir() / "{page_id}.jsonl".format(page_id=page_id)
        payload = {
            "timestamp": int(time.time()),
            "page_id": page_id,
            "customer": commenter_name,
            "comment": comment,
            "response": reply,
        }
        with open(interaction_file, "a", encoding="utf-8") as f:
            json.dump(payload, f, ensure_ascii=False)
            f.write("\n")
        logger.info("Saved interaction to %s", interaction_file.name)
    except Exception as e:
        logger.error("Failed to save interaction for learning: %s", e)


def build_page_messaging_payload(entry, event):
    """Normalize a Facebook Page messaging event for downstream processing."""
    recipient_id = event.get("recipient", {}).get("id", "")
    page_id = recipient_id or entry.get("id", "")
    message = event.get("message", {})
    return {
        "type": "page_messaging",
        "page_id": page_id,
        "sender_id": event.get("sender", {}).get("id", ""),
        "recipient_id": recipient_id,
        "message": message.get("text", ""),
        "timestamp": event.get("timestamp"),
        "raw": event,
    }


def get_private_reply_settings(page_config):
    """Return normalized private-reply settings for a page."""
    configured_types = page_config.get("messenger_private_reply_question_types") or DEFAULT_PRIVATE_REPLY_QUESTION_TYPES
    allowed_types = [
        question_type
        for question_type in configured_types
        if question_type in ALLOWED_PRIVATE_REPLY_QUESTION_TYPES
    ]
    if not allowed_types:
        allowed_types = list(DEFAULT_PRIVATE_REPLY_QUESTION_TYPES)

    templates = {
        question_type: dict(languages)
        for question_type, languages in DEFAULT_PRIVATE_REPLY_TEMPLATES.items()
    }
    configured_templates = page_config.get("messenger_private_reply_templates") or {}
    if isinstance(configured_templates, dict):
        for question_type, languages in configured_templates.items():
            if question_type not in ALLOWED_PRIVATE_REPLY_QUESTION_TYPES or not isinstance(languages, dict):
                continue
            templates[question_type] = {}
            for language, template in languages.items():
                if language in {"en", "ar"} and isinstance(template, str) and template.strip():
                    templates[question_type][language] = template.strip()

    return {
        "enabled": bool(page_config.get("messenger_private_replies_enabled", False)),
        "public_and_private": bool(page_config.get("messenger_private_reply_public_and_private", True)),
        "question_types": allowed_types,
        "templates": templates,
    }


def validate_private_reply_template(template):
    """Allow only the supported template placeholders."""
    try:
        for _, field_name, _, _ in Formatter().parse(template):
            if not field_name:
                continue
            if field_name not in ALLOWED_TEMPLATE_FIELDS:
                return False
    except ValueError:
        return False
    return True


def fallback_customer_name(language, customer_name):
    cleaned_name = (customer_name or "").strip()
    if cleaned_name:
        return cleaned_name
    return "حضرتك" if language == "ar" else "there"


def render_private_reply_message(page_config, question_type, language, customer_name, public_reply, source_section=""):
    """Render the configured private reply template for a supported intent."""
    if source_section in LEARNED_REPLY_PUBLIC_SOURCES:
        mirrored_reply = (public_reply or "").strip()
        return mirrored_reply or None

    settings = get_private_reply_settings(page_config)
    templates = settings["templates"].get(question_type)
    if not templates:
        return None

    template = templates.get(language) or templates.get("en")
    if not template or not validate_private_reply_template(template):
        return None

    try:
        return template.format(
            customer_name=fallback_customer_name(language, customer_name),
            public_reply=(public_reply or "").strip(),
        ).strip()
    except (KeyError, ValueError):
        return None


def should_send_private_reply(page_config, reply_result, comment_id, sender_id, page_id):
    """Check whether a comment is eligible for a private Messenger reply."""
    if not page_config or not comment_id or not reply_result:
        return False
    if sender_id == page_id or has_private_reply(comment_id):
        return False

    settings = get_private_reply_settings(page_config)
    if not settings["enabled"] or not settings["public_and_private"]:
        return False
    if is_exact_unknown_fallback(reply_result.response):
        return False
    if not (reply_result.found_in_kb or reply_result.source_section in LEARNED_REPLY_PUBLIC_SOURCES):
        return False
    if reply_result.question_type not in settings["question_types"]:
        return False
    return True

def log_event(event_data):
    try:
        with open(get_webhook_log_file(), "a", encoding="utf-8") as f:
            f.write(f"{event_data}\n")
    except Exception as e:
        logger.error(f"Failed to log event: {e}")


def verify_facebook_signature(request_obj):
    if not APP_SECRET:
        logger.error("FB_APP_SECRET is not configured — rejecting webhook request")
        return False
    signature_header = request_obj.headers.get("X-Hub-Signature-256", "")
    if not signature_header:
        logger.warning("Missing X-Hub-Signature-256 header")
        return False
    if not signature_header.startswith("sha256="):
        logger.warning("Invalid signature format")
        return False
    expected_signature = signature_header[7:]
    computed = hmac.new(
        APP_SECRET.encode("utf-8"),
        request_obj.get_data(),
        hashlib.sha256,
    ).hexdigest()
    if not hmac.compare_digest(computed, expected_signature):
        logger.warning("Webhook signature mismatch")
        return False
    return True


RETRY_INTERVAL_SECONDS = 60
MAX_RETRY_ATTEMPTS = 5
_retry_thread = None
_retry_stop_event = threading.Event()


def _retry_worker_loop():
    while not _retry_stop_event.is_set():
        try:
            _process_retry_queue()
        except Exception as exc:
            logger.error("Retry worker error: %s", exc)
        _retry_stop_event.wait(RETRY_INTERVAL_SECONDS)


def _process_retry_queue():
    queue_dir = Path.home() / ".hermes" / "webhook_queue"
    if not queue_dir.exists():
        return
    for queue_file in sorted(queue_dir.glob("*.json")):
        try:
            with open(queue_file, "r", encoding="utf-8") as f:
                payload = json.load(f)
            attempts = payload.get("_retry_attempts", 0) + 1
            if attempts > MAX_RETRY_ATTEMPTS:
                logger.warning("Max retries reached for %s, giving up", queue_file.name)
                dead_dir = queue_dir / "../dead_letter"
                dead_dir.mkdir(parents=True, exist_ok=True)
                queue_file.rename(dead_dir / queue_file.name)
                continue
            payload["_retry_attempts"] = attempts
            response = requests.post(
                f"{HERMES_API_URL}/webhook/facebook",
                json=payload,
                timeout=5,
            )
            if response.status_code == 200:
                logger.info("Retry succeeded for %s (attempt %d)", queue_file.name, attempts)
                queue_file.unlink(missing_ok=True)
            else:
                logger.warning("Retry failed for %s: HTTP %d", queue_file.name, response.status_code)
                with open(queue_file, "w", encoding="utf-8") as f:
                    json.dump(payload, f)
        except Exception as exc:
            logger.error("Error processing retry file %s: %s", queue_file.name, exc)


def start_retry_worker():
    global _retry_thread
    if _retry_thread is not None and _retry_thread.is_alive():
        return
    _retry_stop_event.clear()
    _retry_thread = threading.Thread(target=_retry_worker_loop, daemon=True, name="retry-worker")
    _retry_thread.start()
    logger.info("Retry worker started (interval=%ds, max_attempts=%d)", RETRY_INTERVAL_SECONDS, MAX_RETRY_ATTEMPTS)

def require_admin():
    if not ADMIN_TOKEN:
        return True
    token = request.headers.get("Authorization", "")
    return token == f"Bearer {ADMIN_TOKEN}"


@app.route('/health', methods=['GET'])
def health_check():
    if not require_admin():
        return jsonify({"status": "healthy"}), 200
    pages_config = load_pages_config()
    pages = pages_config.get("pages", {})

    return jsonify({
        "status": "healthy",
        "service": "facebook-webhook",
        "pages_configured": len(pages),
    }), 200


@app.route("/debug/reply-preview", methods=["POST"])
def debug_reply_preview():
    if not is_test_mode() or not require_admin():
        return jsonify({"error": "Not found"}), 404

    payload = request.get_json(silent=True) or {}
    page_id = payload.get("page_id")
    message = (payload.get("message") or "").strip()

    if not page_id or not message:
        return jsonify({"error": "page_id and message are required"}), 400

    result = build_reply_result(page_id, message)
    return jsonify({
        "response": result.response,
        "language": result.language,
        "question_type": result.question_type,
        "found_in_kb": result.found_in_kb,
        "source_section": result.source_section,
    }), 200


def is_valid_telegram_webhook_request(page_id):
    telegram_token, _ = get_telegram_credentials(page_id)
    if not telegram_token:
        return False

    expected_secret = build_telegram_webhook_secret(page_id, telegram_token)
    provided_secret = request.headers.get("X-Telegram-Bot-Api-Secret-Token", "")
    if not provided_secret:
        return False
    return hmac.compare_digest(provided_secret, expected_secret)


def format_learned_reply_list(page_id, limit=20):
    items = list_active_learned_replies(page_id)
    if not items:
        return "No active taught replies for this page yet."

    lines = ["📚 Active taught replies:"]
    for index, payload in enumerate(items[:limit], start=1):
        source_question = (payload.get("source_question") or "").strip()
        learned_reply = (payload.get("learned_reply") or "").strip()
        source_label = LEARNED_REPLY_SOURCE_SECTIONS.get(payload.get("source"), payload.get("source", "learned"))
        lines.append(f"{index}. {source_question} => {learned_reply}")
        lines.append(f"   source: {source_label}")
    if len(items) > limit:
        lines.append(f"... and {len(items) - limit} more")
    return "\n".join(lines)


def handle_telegram_teach_update(page_id, update):
    message = update.get("message") or update.get("edited_message") or {}
    message_text = (message.get("text") or "").strip()
    if not message_text:
        return

    _, expected_chat_id = get_telegram_credentials(page_id)
    chat_id = str(message.get("chat", {}).get("id", "")).strip()
    if not expected_chat_id or chat_id != expected_chat_id:
        logger.warning("Ignoring Telegram teach update from unauthorized chat for page %s", page_id)
        return

    message_id = message.get("message_id")
    lower_text = message_text.lower()
    if lower_text.startswith("/help") or lower_text.startswith("/start") or lower_text.startswith("/teachhelp"):
        notification_system.send_telegram_text(page_id, get_telegram_teach_usage(page_id), reply_to_message_id=message_id)
        return
    if lower_text.startswith("/learned") or lower_text.startswith("/list"):
        notification_system.send_telegram_text(
            page_id,
            format_learned_reply_list(page_id),
            reply_to_message_id=message_id,
        )
        return
    remove_target = parse_telegram_remove_command(message_text)
    if any(lower_text.startswith(prefix) for prefix in ("/remove", "/forget", "/delete")) and not remove_target:
        notification_system.send_telegram_text(
            page_id,
            "❌ Missing question to remove.\n\nUse /remove سلام عليكم",
            reply_to_message_id=message_id,
        )
        return
    if remove_target:
        normalized_message = normalize_learning_text(remove_target)
        if not normalized_message:
            notification_system.send_telegram_text(
                page_id,
                "❌ Missing question to remove.\n\nUse /remove سلام عليكم",
                reply_to_message_id=message_id,
            )
            return
        removed = deactivate_active_learned_reply(
            page_id,
            normalized_message,
            removed_by=chat_id,
            reason="telegram_remove",
        )
        if removed:
            notification_system.send_telegram_text(
                page_id,
                f"🗑️ Removed taught reply for: {remove_target}",
                reply_to_message_id=message_id,
            )
        else:
            notification_system.send_telegram_text(
                page_id,
                f"❌ No active taught reply found for: {remove_target}",
                reply_to_message_id=message_id,
            )
        return

    parsed = parse_telegram_teach_command(message_text)
    if parsed is None:
        return

    question = (parsed.get("question") or "").strip()
    answer = (parsed.get("answer") or "").strip()
    if not question or not answer:
        notification_system.send_telegram_text(
            page_id,
            "❌ Missing question or answer.\n\n" + get_telegram_teach_usage(page_id),
            reply_to_message_id=message_id,
        )
        return

    success, status_message = learn_from_telegram_teach(
        page_id,
        question,
        answer,
        taught_by_chat_id=chat_id,
        telegram_message_id=str(message_id or ""),
    )
    if success:
        confirmation = (
            "✅ Learned and active now.\n\n"
            f"Question: {question}\n"
            f"Answer: {answer}"
        )
    else:
        confirmation = f"❌ {status_message}\n\n{get_telegram_teach_usage(page_id)}"
    notification_system.send_telegram_text(page_id, confirmation, reply_to_message_id=message_id)


@app.route("/telegram/webhook/<page_id>", methods=["POST"])
def telegram_webhook(page_id):
    page_config = get_page_config(page_id)
    if not page_config:
        return jsonify({"error": "Not found"}), 404
    if not get_telegram_teach_settings(page_config)["enabled"]:
        return "OK", 200
    if not is_valid_telegram_webhook_request(page_id):
        return jsonify({"error": "Forbidden"}), 403

    update = request.get_json(silent=True) or {}
    log_event("TELEGRAM_UPDATE: {}".format(json.dumps(sanitize_for_log(update), ensure_ascii=False)))
    try:
        handle_telegram_teach_update(page_id, update)
    except Exception as exc:
        logger.error("Failed processing Telegram teach update for %s: %s", page_id, exc)
    return "OK", 200

@app.route('/webhook', methods=['GET'])
def verify_webhook():
    mode = request.args.get('hub.mode')
    token = request.args.get('hub.verify_token')
    challenge = request.args.get('hub.challenge')

    if mode == 'subscribe' and token == VERIFY_TOKEN:
        logger.info("Webhook verified successfully")
        return challenge, 200
    else:
        logger.warning("Webhook verification failed: mode=%s", mode)
        return "Forbidden", 403

@app.route('/webhook', methods=['POST'])
def webhook():
    if not verify_facebook_signature(request):
        return jsonify({"error": "Invalid signature"}), 403

    data = request.get_json(silent=True) or {}

    log_event("EVENT: {}".format(json.dumps(sanitize_for_log(data), ensure_ascii=False)))

    if data.get('object') == 'page':
        for entry in data.get('entry', []):
            for change in entry.get('changes', []):
                field = change.get('field')

                if field == 'comments':
                    handle_comment_event(change.get('value'))
                elif field == 'feed':
                    handle_feed_event(change.get('value'))
                else:
                    logger.info(f"Unhandled field: {field}")

            if 'messaging' in entry or 'standby' in entry:
                handle_page_messaging_event(entry)

    # Check if this is an Instagram update
    elif data.get('object') == 'instagram':
        for entry in data.get('entry', []):
            # Handle Instagram comments
            if 'changes' in entry:
                for change in entry.get('changes', []):
                    field = change.get('field')

                    if field == 'comments':
                        handle_instagram_comment_event(change.get('value'))
                    else:
                        logger.info(f"Unhandled Instagram field: {field}")

            # Handle Instagram mentions/comments directly
            elif 'messaging' in entry or 'standby' in entry:
                handle_instagram_direct_event(entry)

    return "OK", 200

def handle_comment_event(value):
    comment_id = value.get('id')
    post_id = value.get('post_id')
    message = value.get('message', '')
    from_name = value.get('from', {}).get('name', 'Unknown')
    verb = value.get('verb', 'added')

    logger.info("Comment %s: %s - %s", verb, from_name, message[:80])
    log_event("COMMENT: {}: {}".format(from_name, message[:200]))

    payload = {
        "type": "new_comment",
        "comment_id": comment_id,
        "post_id": post_id,
        "message": message,
        "from": from_name,
        "verb": verb,
        "raw": value
    }

    forward_to_hermes(payload)


def handle_page_messaging_event(entry):
    try:
        logger.info("Facebook Page messaging event received")
        log_event("PAGE_MESSAGING: {}".format(json.dumps(sanitize_for_log(entry), ensure_ascii=False)))

        for event in entry.get("messaging", []) + entry.get("standby", []):
            forward_to_hermes(build_page_messaging_payload(entry, event))
    except Exception as e:
        logger.error("Error processing Facebook Page messaging event: %s", e)

def handle_feed_event(value):
    """Process new post/feed event"""
    if value.get('item') == 'comment':
        comment_id = value.get('comment_id')
        message = value.get('message', '')
        post_id = value.get('post_id')
        parent_id = value.get('parent_id')

        # Extract page ID from post_id (format: PAGEID_POSTID)
        page_id = post_id.split('_')[0] if post_id else ""

        # Try different fields for the sender name and ID
        from_name = (value.get('sender_name') or
                     value.get('from', {}).get('name') or
                     value.get('author_name') or
                     'Unknown')

        sender_id = (value.get('sender_id') or
                     value.get('from', {}).get('id') or
                     '')

        logger.info("Feed comment on page %s: %s - %s", page_id, from_name, message[:80])
        log_event("COMMENT_EVENT: {} said: {}".format(from_name, message[:200]))

        # Get page configuration
        page_config = get_page_config(page_id)

        # Try to auto-reply if this is a comment (and NOT from the page itself!)
        try:
            # Check if comment is from the page itself - prevent infinite loop!
            if sender_id == page_id:
                if message and parent_id:
                    learned = learn_from_human_page_reply(page_id, comment_id, parent_id, message, sender_id)
                    if learned:
                        logger.info("Learned human page reply for parent comment %s", parent_id)
                logger.info("Skipping reply - comment is from the page itself")
            elif has_already_replied(comment_id):
                logger.info("Skipping reply - already replied to this comment")
            elif from_name != 'Unknown' and sender_id != page_id and page_config:
                reply_result = build_reply_result(page_id, message)
                reply = reply_result.response
                record_comment_context_and_save_reply(
                    page_id,
                    comment_id,
                    message,
                    from_name,
                    sender_id,
                    post_id,
                    parent_id,
                    reply_result,
                    reply,
                )
                if reply:
                    # Attempt to reply
                    logger.info("📤 Attempting to reply to %s: %s", from_name, message[:50])
                    
                    reply_response = reply_to_facebook_comment(page_id, comment_id, reply)
                    
                    # Check if reply was successful
                    if reply_response.get('id'):
                        logger.info("✅ AI-replied to %s (reply ID: %s)", from_name, reply[:50])
                        
                        # Save to SQLite database
                        save_reply_to_sqlite(page_id, comment_id, reply, reply_response.get('id', ''))
                        record_auto_public_reply_signature(page_id, comment_id, reply)
                        
                        # Save interaction for learning
                        save_interaction_for_learning(page_id, from_name, message, reply)

                        # Send a private Messenger reply for specific intents
                        if should_send_private_reply(page_config, reply_result, comment_id, sender_id, page_id):
                            private_reply = render_private_reply_message(
                                page_config,
                                reply_result.question_type,
                                reply_result.language,
                                from_name,
                                reply,
                                reply_result.source_section,
                            )
                            if private_reply:
                                private_reply_response = send_private_reply_to_commenter(
                                    page_id,
                                    comment_id,
                                    sender_id,
                                    private_reply,
                                )
                                if private_reply_response.get("id"):
                                    logger.info(
                                        "✅ Sent private reply to %s for %s intent",
                                        from_name,
                                        reply_result.question_type,
                                    )
                                    mark_private_reply(comment_id)
                                else:
                                    logger.warning("⚠️  Failed private reply: %s", private_reply_response)

                        # Notify page owner
                        try:
                            notification_system.notify_owner(page_id, from_name, message, reply)
                        except Exception as notify_error:
                            logger.warning("Failed to send notification: %s", notify_error)
                            
                    else:
                        # Reply failed - log detailed error
                        error_code = reply_response.get('error_code', 'UNKNOWN')
                        error_msg = reply_response.get('error', 'Unknown error')
                        logger.error("❌ Failed to reply to %s: [%s] %s", 
                                   from_name, error_code, error_msg)
                        
                        # Log additional context for debugging
                        logger.warning("Comment ID: %s, Page: %s", comment_id, page_id)
                        logger.warning("Original message: %s", message[:100])
        except Exception as e:
            logger.error("Auto-reply failed: %s", e)

        # Still try to forward to Hermes (optional)
        payload = {
            "type": "new_comment",
            "page_id": page_id,
            "comment_id": comment_id,
            "post_id": post_id,
            "message": message,
            "from": from_name,
            "raw": value
        }

        # Forward to Hermes for processing and learning
        forward_to_hermes(payload)

def generate_auto_reply(page_id, message):
    """Generate a grounded reply using KB extraction and optional LLM rephrasing."""
    if not message:
        return None
    return build_reply_result(page_id, message).response


def _extract_hermes_response(output_text):
    cleaned_response = []
    in_response = False

    for line in output_text.splitlines():
        if any(skip in line for skip in ["Hermes Agent", "Session:", "Duration:", "Messages:", "Available", "╭", "╰", "│", "═", "Resume this"]):
            continue
        if "⚕ Hermes" in line or in_response:
            in_response = True
            if line.strip() and not any(skip in line for skip in ["⚕ Hermes", "───"]):
                cleaned_response.append(line.strip())

    if cleaned_response:
        return "\n".join(cleaned_response).strip()
    return ""


def _run_hermes_rephrase(prompt, message):
    if shutil.which("hermes") is None:
        return ""

    try:
        result = subprocess.run(
            ["hermes", "chat", "--query", prompt],
            input=message,
            capture_output=True,
            text=True,
            timeout=30,
            cwd="/tmp",
        )
    except subprocess.TimeoutExpired:
        logger.error("Hermes CLI timed out")
        return ""
    except Exception as exc:
        logger.error("Failed to use Hermes AI: %s", exc)
        return ""

    if result.returncode != 0:
        logger.warning("Hermes CLI returned non-zero: %s", result.returncode)
        return ""

    return _extract_hermes_response(result.stdout.strip())


def build_reply_result(page_id, message):
    page_config = get_page_config(page_id)
    if not page_config:
        logger.warning("No config found for page %s", page_id)
        return build_response(message, "", "this business")

    page_name = page_config.get("page_name", "this business")
    knowledge = load_knowledge_base(page_id)
    llm_enabled = is_llm_rephrase_enabled()
    result = build_response(
        message=message,
        knowledge=knowledge,
        page_name=page_name,
        llm_runner=_run_hermes_rephrase if llm_enabled else None,
        enable_llm_rephrase=llm_enabled,
    )
    learning_settings = get_human_reply_learning_settings(page_config)
    telegram_teach_settings = get_telegram_teach_settings(page_config)
    learned_payload = None
    if (learning_settings["enabled"] or telegram_teach_settings["enabled"]) and is_exact_unknown_fallback(result.response):
        learned_payload = get_active_learned_reply_payload(page_id, message)
    if learned_payload:
        result.response = learned_payload.get("learned_reply", "")
        result.found_in_kb = False
        result.source_section = LEARNED_REPLY_SOURCE_SECTIONS.get(
            learned_payload.get("source"),
            "Human Reply Learning",
        )
    return result


def load_knowledge_base(page_id):
    """Load business knowledge from file for a specific page."""
    try:
        page_config = get_page_config(page_id)
        if not page_config:
            logger.error("No config found for page %s", page_id)
            return ""

        knowledge_file = resolve_configured_data_path(page_config.get("knowledge_file", ""))
        if not knowledge_file.exists():
            logger.error("Knowledge file not found: %s", knowledge_file)
            return ""

        with open(knowledge_file, "r", encoding="utf-8") as f:
            return f.read()
    except Exception as exc:
        logger.error("Failed to load knowledge base: %s", exc)
        return ""


def get_fallback_reply(page_id, message):
    """Return the deterministic grounded fallback reply."""
    return build_reply_result(page_id, message).response


def reply_to_facebook_comment(page_id, comment_id, message):
    """
    Post a reply comment to Facebook
    Returns dict with 'id' on success, or 'error' on failure
    """
    try:
        # Get page configuration
        page_config = get_page_config(page_id)
        if not page_config:
            logger.error("No config found for page %s", page_id)
            return {"error": "Page not configured", "error_code": "NO_CONFIG"}

        # Get access token
        access_token = page_config.get('access_token')
        if not access_token:
            logger.error("No access token found for page %s", page_id)
            return {"error": "No access token", "error_code": "NO_TOKEN"}

        # Log reply attempt
        logger.info("📤 Attempting to reply to comment %s on page %s", comment_id[:30], page_id)
        
        # Make API request
        res = requests.post(
            f"{GRAPH_URL}/{comment_id}/comments",
            data={
                "message": message,
                "access_token": access_token
            },
            timeout=30,
        )
        
        # Check HTTP status
        if res.status_code != 200:
            logger.error("Facebook API returned HTTP %d for comment %s", res.status_code, comment_id[:30])
            try:
                error_data = res.json()
                return {
                    "error": error_data.get("error", {}).get("message", f"HTTP {res.status_code}"),
                    "error_code": error_data.get("error", {}).get("code"),
                    "http_status": res.status_code,
                    "details": error_data
                }
            except:
                return {
                    "error": f"HTTP {res.status_code}: {res.text[:100]}",
                    "error_code": f"HTTP_{res.status_code}",
                    "http_status": res.status_code
                }
        
        # Parse JSON response
        try:
            result = res.json()
            
            # Check for Facebook API errors
            if 'error' in result:
                error = result['error']
                logger.error("Facebook API error: %s (code: %s)", 
                           error.get('message'), error.get('code'))
                return {
                    "error": error.get('message', 'Unknown error'),
                    "error_code": error.get('code'),
                    "error_type": error.get('type'),
                    "details": error
                }
            
            # Success - return the comment ID
            reply_id = result.get('id')
            if reply_id:
                logger.info("✅ Successfully replied to comment %s (reply ID: %s)", comment_id[:30], reply_id[:30])
            
            return result
            
        except ValueError:
            logger.error("Invalid JSON response from Facebook for comment %s", comment_id[:30])
            return {
                "error": f"Invalid JSON response: {res.text[:100]}",
                "error_code": "INVALID_JSON",
                "http_status": res.status_code
            }
            
    except requests.exceptions.Timeout:
        logger.error("Timeout while replying to comment %s", comment_id[:30])
        return {
            "error": "Request timeout - Facebook API is slow",
            "error_code": "TIMEOUT",
            "suggestion": "Try again later or increase timeout"
        }
    except Exception as e:
        logger.error("Failed to reply to %s: %s", comment_id, e, exc_info=True)
        return {
            "error": str(e),
            "error_code": "EXCEPTION",
            "exception_type": type(e).__name__
        }


def send_private_reply_to_facebook_comment(page_id, comment_id, message):
    """
    Send a private Messenger reply to a commenter
    Returns dict with 'id' on success, or 'error' on failure
    """
    try:
        page_config = get_page_config(page_id)
        if not page_config:
            logger.error("No config found for page %s", page_id)
            return {"error": "Page not configured", "error_code": "NO_CONFIG"}

        access_token = page_config.get("access_token")
        if not access_token:
            logger.error("No access token found for page %s", page_id)
            return {"error": "No access token", "error_code": "NO_TOKEN"}

        logger.info("📤 Sending private reply to %s via Messenger", comment_id[:30])
        
        response = requests.post(
            f"{GRAPH_URL}/{page_id}/messages",
            params={"access_token": access_token},
            json={
                "recipient": {"comment_id": comment_id},
                "message": {"text": message},
            },
            timeout=30,
        )

        if response.status_code != 200:
            logger.error("Messenger API returned HTTP %d", response.status_code)
            try:
                error_data = response.json()
                return {
                    "error": error_data.get("error", {}).get("message", f"HTTP {response.status_code}"),
                    "error_code": error_data.get("error", {}).get("code"),
                    "http_status": response.status_code,
                    "details": error_data
                }
            except:
                return {
                    "error": f"HTTP {response.status_code}: {response.text[:100]}",
                    "error_code": f"HTTP_{response.status_code}",
                    "http_status": response.status_code
                }

        try:
            result = response.json()
        except ValueError:
            return {
                "error": f"Invalid JSON: {response.text[:100]}",
                "error_code": "INVALID_JSON",
                "http_status": response.status_code
            }
            
        if response.status_code == 200:
            logger.info("✅ Private reply sent successfully to %s", comment_id[:30])
            return result
        else:
            logger.error("Messenger API error for page %s: HTTP %d", page_id, response.status_code)
            return {"error": result.get("error") or response.text, "http_status": response.status_code}
            
    except requests.exceptions.Timeout:
        logger.error("Timeout while sending private reply to %s", comment_id[:30])
        return {
            "error": "Request timeout - Messenger API is slow",
            "error_code": "TIMEOUT"
        }
    except Exception as e:
        logger.error("Failed to send private reply to %s: %s", comment_id, e, exc_info=True)
        return {"error": str(e), "error_code": "EXCEPTION"}
        return {"error": str(e)}


def send_direct_messenger_reply(page_id, recipient_id, message):
    try:
        page_config = get_page_config(page_id)
        if not page_config:
            logger.error("No config found for page %s", page_id)
            return {"error": "Page not configured"}

        access_token = page_config.get("access_token")
        if not access_token:
            logger.error("No access token found for page %s", page_id)
            return {"error": "No access token"}

        response = requests.post(
            f"{GRAPH_URL}/me/messages",
            params={"access_token": access_token},
            json={
                "messaging_type": "RESPONSE",
                "recipient": {"id": recipient_id},
                "message": {"text": message},
            },
            timeout=30,
        )
        result = response.json()
        if response.status_code == 200 and (result.get("message_id") or result.get("recipient_id")):
            return {
                "id": result.get("message_id") or result.get("recipient_id"),
                "delivery": "direct_messenger",
                "raw": result,
            }
        logger.error("Direct Messenger API error for page %s: HTTP %d", page_id, response.status_code)
        return {"error": result.get("error") or response.text}
    except Exception as e:
        logger.error("Failed to send direct Messenger reply: %s", e)
        return {"error": str(e)}


def _private_reply_needs_direct_fallback(private_reply_response):
    if not private_reply_response or private_reply_response.get("id"):
        return False

    error = private_reply_response.get("error")
    if isinstance(error, dict):
        return error.get("type") == "GraphMethodException" or (
            error.get("code") == 100 and str(error.get("error_subcode")) == "33"
        )

    text = str(error or "")
    return "GraphMethodException" in text or '"error_subcode":33' in text or "'error_subcode': 33" in text


def send_private_reply_to_commenter(page_id, comment_id, sender_id, message):
    """Try the comment private-reply edge, then fall back to direct Messenger if Meta rejects it."""
    private_reply_response = send_private_reply_to_facebook_comment(page_id, comment_id, message)
    if private_reply_response.get("id"):
        return private_reply_response
    if not sender_id or not _private_reply_needs_direct_fallback(private_reply_response):
        return private_reply_response

    logger.info("↪ Trying direct Messenger fallback for comment %s", comment_id)
    direct_response = send_direct_messenger_reply(page_id, sender_id, message)
    if direct_response.get("id"):
        return direct_response
    return {
        "error": {
            "comment_private_reply": private_reply_response,
            "direct_messenger": direct_response,
        }
    }

def save_to_sqlite(payload):
    """Save comment to SQLite database for this page"""
    try:
        page_id = payload.get("page_id") or payload.get("post_id", "").split("_")[0] if payload.get("post_id") else None
        
        if not page_id:
            logger.warning("No page_id in payload, cannot save to SQLite")
            return False
        
        # Get the database for this page
        db = get_page_db(page_id)
        
        # Prepare comment data
        comment_data = {
            'comment_id': payload.get("comment_id"),
            'post_id': payload.get("post_id"),
            'message': payload.get("message", ""),
            'from': payload.get("from", "Unknown"),
            'sender_id': payload.get("sender_id", ""),
            'parent_id': payload.get("parent_id", ""),
            'verb': payload.get("verb", "add"),
            'created_time': payload.get("created_time", int(datetime.now().timestamp())),
            'raw': payload.get("raw", {})
        }
        
        # Save to database
        db_id = db.add_comment(comment_data)
        
        if db_id > 0:
            logger.info("✅ Saved comment %s to page %s SQLite DB (ID: %d)", 
                       comment_data['comment_id'][:20], page_id, db_id)
            return True
        else:
            logger.info("Comment %s already exists in database", 
                       comment_data['comment_id'][:20])
            return True
            
    except Exception as e:
        logger.error("Error saving to SQLite: %s", e)
        return False

def forward_to_hermes(payload):
    """Forward comment to Hermes (disabled - using SQLite instead)"""
    # Save to SQLite instead of Hermes
    save_to_sqlite(payload)

def queue_for_retry(payload):
    queue_dir = Path.home() / ".hermes" / "webhook_queue"
    queue_dir.mkdir(parents=True, exist_ok=True)

    timestamp = int(time.time())
    comment_id = payload.get("comment_id", "unknown")
    queue_file = queue_dir / f"{timestamp}_{comment_id}.json"

    try:
        with open(queue_file, "w", encoding="utf-8") as f:
            fcntl.flock(f.fileno(), fcntl.LOCK_EX)
            json.dump(payload, f, ensure_ascii=False)
            fcntl.flock(f.fileno(), fcntl.LOCK_UN)
        logger.info("Queued for retry: %s", queue_file.name)
    except Exception as exc:
        logger.error("Failed to queue retry: %s", exc)

# ═════════════════════════════════════════════════════════════════════════
# INSTAGRAM HANDLING FUNCTIONS
# ═════════════════════════════════════════════════════════════════════════

def handle_instagram_comment_event(value):
    """Process Instagram comment event"""
    try:
        comment_id = value.get('id')
        media_id = value.get('media_id') or value.get('media', {}).get('id', '')
        message = value.get('text', '')
        from_user = value.get('from', {})
        from_name = from_user.get('username', 'Unknown')
        from_id = from_user.get('id', '')
        timestamp = value.get('timestamp', '')

        logger.info("Instagram comment from %s: %s", from_name, message[:80])
        log_event("INSTAGRAM_COMMENT: {} (@{}): {}".format(from_name, from_id, message[:200]))

        # Extract Instagram business account ID from media_id or comment_id
        # Instagram accounts are connected to Facebook pages
        instagram_account_id = comment_id.split('_')[0] if '_' in comment_id else None

        # Find the Facebook page ID this Instagram account is connected to
        page_id = get_facebook_page_for_instagram(instagram_account_id, media_id)

        if not page_id:
            logger.warning("Could not find connected Facebook page for Instagram account %s", instagram_account_id)
            # Forward to Hermes for processing
            payload = {
                "type": "instagram_comment",
                "instagram_account_id": instagram_account_id,
                "comment_id": comment_id,
                "media_id": media_id,
                "message": message,
                "from": from_name,
                "from_id": from_id,
                "timestamp": timestamp,
                "raw": value
            }
            forward_to_hermes(payload)
            return

        # Get page configuration
        page_config = get_page_config(page_id)

        # Try to auto-reply (not from our own Instagram account)
        try:
            # Check if comment is from the account itself
            if from_id == instagram_account_id:
                logger.info("Skipping reply - comment is from the Instagram account itself")
            elif has_already_replied(comment_id):
                logger.info("Skipping reply - already replied to this Instagram comment")
            elif page_config and page_config.get('instagram_enabled', True):
                # Generate AI-powered reply using Hermes (shares same knowledge base!)
                reply = generate_auto_reply(page_id, message)
                if reply:
                    reply_response = reply_to_instagram_comment(comment_id, reply)
                    if reply_response.get('id'):
                        logger.info("AI-replied to Instagram comment from %s: %s...", from_name, reply[:50])
                        mark_as_replied(comment_id)
                        # Save interaction for learning (same knowledge base!)
                        save_interaction_for_learning(page_id, from_name, message, reply)

                        # 🔔 Send notification to page owner (includes Instagram indicator)
                        notification_text = f"📸 Instagram Comment\n\n{message}"
                        notification_system.notify_owner(page_id, from_name, notification_text, reply)
                    else:
                        logger.warning("Failed to reply to Instagram comment: %s", reply_response)
        except Exception as e:
            logger.error("Instagram auto-reply failed: %s", e)

        # Forward to Hermes for processing and learning
        payload = {
            "type": "instagram_comment",
            "page_id": page_id,
            "instagram_account_id": instagram_account_id,
            "comment_id": comment_id,
            "media_id": media_id,
            "message": message,
            "from": from_name,
            "from_id": from_id,
            "timestamp": timestamp,
            "raw": value
        }

        forward_to_hermes(payload)

    except Exception as e:
        logger.error("Error processing Instagram comment event: %s", e)


def handle_instagram_direct_event(entry):
    try:
        logger.info("Instagram direct event received")
        log_event("INSTAGRAM_DIRECT: {}".format(json.dumps(sanitize_for_log(entry), ensure_ascii=False)))

        # Extract messaging data
        messaging = entry.get('messaging', [])
        standby = entry.get('standby', [])

        # Process messaging events (mentions, comments on posts)
        for event in messaging + standby:
            if 'message' in event:
                message = event['message']
                from_user = message.get('from', {})
                from_name = from_user.get('username', 'Unknown')
                from_id = from_user.get('id', '')
                text = message.get('text', '')

                logger.info("Instagram message from %s: %s", from_name, text[:80])

                # Extract Instagram business account ID
                instagram_account_id = from_id

                # Find connected Facebook page
                page_id = get_facebook_page_for_instagram(instagram_account_id, None)

                if page_id:
                    # Use same knowledge base as Facebook page
                    page_config = get_page_config(page_id)
                    if page_config and page_config.get('instagram_enabled', True):
                        reply = generate_auto_reply(page_id, text)
                        if reply:
                            logger.info("Generated Instagram reply for %s", from_name)

                log_event("INSTAGRAM_MESSAGE: {}: {}".format(from_name, text[:200]))

    except Exception as e:
        logger.error("Error processing Instagram direct event: %s", e)


def get_facebook_page_for_instagram(instagram_account_id, media_id):
    try:
        # Load all page configurations
        pages_config = load_pages_config()
        pages = pages_config.get("pages", {})

        # Search for page that has this Instagram account connected
        for page_id, page_config in pages.items():
            # Check if page has Instagram account ID configured
            if page_config.get('instagram_account_id') == instagram_account_id:
                logger.info("Found Instagram account %s connected to page %s", instagram_account_id, page_id)
                return page_id

        # If not found in config, try to look it up via Graph API
        # (This would require the instagram_basic permission)
        if instagram_account_id:
            try:
                # Try to get the Instagram business account info
                page_config = list(pages.values())[0] if pages else None
                if page_config:
                    access_token = page_config.get('access_token')
                    if access_token:
                        # Query Instagram account info
                        url = f"{GRAPH_URL}/{instagram_account_id}"
                        params = {
                            'fields': 'connected_page',
                            'access_token': access_token
                        }

                        response = requests.get(url, params=params, timeout=10)

                        if response.status_code == 200:
                            data = response.json()
                            connected_page = data.get('connected_page', {})
                            if connected_page:
                                connected_page_id = connected_page.get('id')
                                logger.info("Found connected page %s for Instagram %s", connected_page_id, instagram_account_id)
                                return connected_page_id

            except Exception as e:
                logger.debug("Could not lookup Instagram connection via API: %s", e)

        logger.warning("No Facebook page found for Instagram account %s", instagram_account_id)
        return None

    except Exception as e:
        logger.error("Error finding Facebook page for Instagram: %s", e)
        return None


def reply_to_instagram_comment(comment_id, message):
    try:
        pages_config = load_pages_config()
        pages = pages_config.get("pages", {})

        if not pages:
            logger.error("No pages configured for Instagram reply")
            return {"error": "No pages configured"}

        page_config = list(pages.values())[0]
        access_token = page_config.get('access_token')

        if not access_token:
            logger.error("No access token found for Instagram reply")
            return {"error": "No access token"}

        url = f"{GRAPH_URL}/{comment_id}/replies"

        response = requests.post(
            url,
            data={
                "message": message,
                "access_token": access_token
            },
            timeout=30,
        )

        result = response.json()

        if response.status_code == 200:
            logger.info("Successfully posted Instagram reply")
            return result
        else:
            logger.error("Instagram API error: HTTP %d", response.status_code)
            return {"error": response.text}

    except Exception as e:
        logger.error("Failed to reply to Instagram comment: %s", e)
        return {"error": str(e)}

@app.errorhandler(500)
def server_error(e):
    logger.error("Server error: %s", e)
    return jsonify({"error": "Internal server error"}), 500

if os.environ.get("START_RETRY_WORKER", "1") == "1":
    try:
        start_retry_worker()
    except Exception:
        pass

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 5000))
    logger.info("Facebook Webhook Server starting on port %d", port)
    start_retry_worker()
    app.run(host='0.0.0.0', port=port, debug=False)

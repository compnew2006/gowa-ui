#!/usr/bin/env python3
"""
Runtime path helpers for the Hermes webhook deployment.

Production still prefers ``/opt/hermes-webhook`` and ``/var/log/hermes``.
When those paths are unavailable, local workspace paths are used instead.
"""

from pathlib import Path
import os


DEPLOY_ROOT = Path("/opt/hermes-webhook")
LOCAL_ROOT = Path(__file__).resolve().parent
DEPLOY_LOG_DIR = Path("/var/log/hermes")
TRUTHY_VALUES = {"1", "true", "yes", "on"}


def _expand_path(value):
    return Path(value).expanduser()


def env_flag(name, default=False):
    value = os.environ.get(name)
    if value is None:
        return default
    return value.strip().lower() in TRUTHY_VALUES


def get_webhook_root():
    env_value = os.environ.get("HERMES_WEBHOOK_ROOT")
    if env_value:
        return _expand_path(env_value)
    if DEPLOY_ROOT.exists():
        return DEPLOY_ROOT
    return LOCAL_ROOT


def resolve_runtime_path(value, default_relative=None):
    if value:
        candidate = _expand_path(value)
        if not candidate.is_absolute():
            return get_webhook_root() / candidate
        if candidate.exists():
            return candidate
        try:
            relative_to_deploy = candidate.relative_to(DEPLOY_ROOT)
        except ValueError:
            return candidate
        return get_webhook_root() / relative_to_deploy

    if default_relative is None:
        return get_webhook_root()
    return get_webhook_root() / default_relative


def ensure_directory(path):
    path.mkdir(parents=True, exist_ok=True)
    return path


def get_pages_config_file():
    return resolve_runtime_path(os.environ.get("HERMES_PAGES_CONFIG"), "pages_config.json")


def get_notification_config_file():
    return resolve_runtime_path(
        os.environ.get("HERMES_NOTIFICATION_CONFIG"),
        "notification_config.json",
    )


def get_log_dir():
    env_value = os.environ.get("HERMES_LOG_DIR")
    if env_value:
        return ensure_directory(resolve_runtime_path(env_value))

    for candidate in (DEPLOY_LOG_DIR, get_webhook_root() / "logs"):
        try:
            return ensure_directory(candidate)
        except OSError:
            continue

    return ensure_directory(Path.cwd() / "hermes-logs")


def get_interaction_log_dir():
    env_value = os.environ.get("HERMES_INTERACTION_LOG_DIR")
    if env_value:
        return ensure_directory(resolve_runtime_path(env_value))
    return ensure_directory(get_log_dir() / "interactions")


def resolve_configured_data_path(value):
    return resolve_runtime_path(value)


def is_test_mode():
    return env_flag("HERMES_TEST_MODE", default=False)


def is_llm_rephrase_enabled():
    return env_flag("HERMES_ENABLE_LLM_REPHRASE", default=False)

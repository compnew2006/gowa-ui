import importlib
import json
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest


WEBHOOK_DIR = Path(__file__).resolve().parents[1]
PAGE_ID = "895247390337022"
KNOWLEDGE_BASENAME = "895247390337022_knowledge.md"


@pytest.fixture
def runtime_env(tmp_path, monkeypatch):
    runtime_root = tmp_path / "webhook-root"
    knowledge_dir = runtime_root / "knowledge"
    knowledge_dir.mkdir(parents=True)

    source_knowledge = WEBHOOK_DIR / "knowledge" / KNOWLEDGE_BASENAME
    knowledge_text = source_knowledge.read_text(encoding="utf-8")
    knowledge_file = knowledge_dir / KNOWLEDGE_BASENAME
    knowledge_file.write_text(knowledge_text, encoding="utf-8")

    pages_config = {
        "pages": {
            PAGE_ID: {
                "page_name": "Ofuqalmadenah",
                "page_id": PAGE_ID,
                "access_token": "test-access-token",
                "knowledge_file": "/opt/hermes-webhook/knowledge/{name}".format(name=KNOWLEDGE_BASENAME),
                "instagram_enabled": True,
                "instagram_account_id": "",
                "instagram_username": "",
                "human_reply_learning_enabled": True,
                "human_reply_learning_max_age_days": 90,
                "human_reply_learning_match_mode": "exact_normalized",
                "telegram_bot_token": "test-telegram-token",
                "telegram_chat_id": "8092716067",
                "telegram_teach_enabled": True,
                "messenger_private_replies_enabled": False,
                "messenger_private_reply_question_types": [
                    "pricing",
                    "location",
                    "hours",
                    "services",
                    "contact",
                    "greeting",
                ],
                "messenger_private_reply_public_and_private": True,
                "messenger_private_reply_templates": {},
            }
        }
    }
    notification_config = {
        "pages": {
            PAGE_ID: {
                "page_name": "Ofuqalmadenah",
                "telegram_bot_token": "test-telegram-token",
                "telegram_chat_id": "8092716067",
            }
        }
    }

    pages_config_file = runtime_root / "pages_config.json"
    notification_config_file = runtime_root / "notification_config.json"
    pages_config_file.write_text(json.dumps(pages_config), encoding="utf-8")
    notification_config_file.write_text(json.dumps(notification_config), encoding="utf-8")

    monkeypatch.setenv("HERMES_WEBHOOK_ROOT", str(runtime_root))
    monkeypatch.setenv("HERMES_PAGES_CONFIG", str(pages_config_file))
    monkeypatch.setenv("HERMES_NOTIFICATION_CONFIG", str(notification_config_file))
    monkeypatch.setenv("HERMES_LOG_DIR", str(runtime_root / "logs"))
    monkeypatch.setenv("HERMES_INTERACTION_LOG_DIR", str(runtime_root / "interaction-logs"))
    monkeypatch.setenv("HERMES_TEST_MODE", "1")
    monkeypatch.delenv("HERMES_ENABLE_LLM_REPHRASE", raising=False)

    if str(WEBHOOK_DIR) not in sys.path:
        sys.path.insert(0, str(WEBHOOK_DIR))

    for module_name in [
        "runtime_paths",
        "notification_system",
        "responder",
        "facebook_webhook_gunicorn",
        "test_agent",
    ]:
        sys.modules.pop(module_name, None)

    runtime_paths = importlib.import_module("runtime_paths")
    responder = importlib.import_module("responder")
    notification_system = importlib.import_module("notification_system")
    app_module = importlib.import_module("facebook_webhook_gunicorn")

    return SimpleNamespace(
        page_id=PAGE_ID,
        knowledge_text=knowledge_text,
        knowledge_file=knowledge_file,
        pages_config_file=pages_config_file,
        runtime_root=runtime_root,
        responder=responder,
        runtime_paths=runtime_paths,
        notification_system=notification_system,
        app_module=app_module,
    )

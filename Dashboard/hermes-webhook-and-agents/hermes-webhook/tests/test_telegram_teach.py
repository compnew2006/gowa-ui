import json


UNKNOWN_FALLBACK = "لا أملك هذه المعلومة المحددة حالياً. دعني أتحقق مع فريقنا وأعود إليك قريباً."


def telegram_headers(runtime_env):
    secret = runtime_env.app_module.build_telegram_webhook_secret(runtime_env.page_id, "test-telegram-token")
    return {"X-Telegram-Bot-Api-Secret-Token": secret}


def telegram_update(message_id, text, chat_id="8092716067"):
    return {
        "update_id": message_id,
        "message": {
            "message_id": message_id,
            "chat": {"id": int(chat_id), "type": "private"},
            "text": text,
        },
    }


def test_telegram_teach_command_learns_reply(runtime_env, monkeypatch):
    sent_messages = []
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_learning", lambda *args, **kwargs: True)
    monkeypatch.setattr(
        runtime_env.app_module.notification_system,
        "send_telegram_text",
        lambda page_id, text, reply_to_message_id=None: sent_messages.append((page_id, text, reply_to_message_id)) or True,
    )

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        f"/telegram/webhook/{runtime_env.page_id}",
        data=json.dumps(telegram_update(101, "/teach سلام عليكم => وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك")),
        content_type="application/json",
        headers=telegram_headers(runtime_env),
    )

    assert response.status_code == 200
    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "سلام عليكم")
    assert result.response == "وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك"
    assert result.source_section == "Telegram Teach"
    assert sent_messages[-1] == (
        runtime_env.page_id,
        "✅ Learned and active now.\n\nQuestion: سلام عليكم\nAnswer: وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك",
        101,
    )


def test_telegram_learned_command_lists_active_replies(runtime_env, monkeypatch):
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_learning", lambda *args, **kwargs: True)
    sent_messages = []
    monkeypatch.setattr(
        runtime_env.app_module.notification_system,
        "send_telegram_text",
        lambda page_id, text, reply_to_message_id=None: sent_messages.append((page_id, text, reply_to_message_id)) or True,
    )
    runtime_env.app_module.learn_from_telegram_teach(
        runtime_env.page_id,
        "سلام عليكم",
        "وعليكم السلام ورحمة الله وبركاته",
        taught_by_chat_id="8092716067",
        telegram_message_id="201",
    )

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        f"/telegram/webhook/{runtime_env.page_id}",
        data=json.dumps(telegram_update(202, "/learned")),
        content_type="application/json",
        headers=telegram_headers(runtime_env),
    )

    assert response.status_code == 200
    assert sent_messages[-1][0] == runtime_env.page_id
    assert "📚 Active taught replies:" in sent_messages[-1][1]
    assert "سلام عليكم => وعليكم السلام ورحمة الله وبركاته" in sent_messages[-1][1]
    assert sent_messages[-1][2] == 202


def test_telegram_remove_command_deactivates_reply(runtime_env, monkeypatch):
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_learning", lambda *args, **kwargs: True)
    sent_messages = []
    monkeypatch.setattr(
        runtime_env.app_module.notification_system,
        "send_telegram_text",
        lambda page_id, text, reply_to_message_id=None: sent_messages.append((page_id, text, reply_to_message_id)) or True,
    )
    runtime_env.app_module.learn_from_telegram_teach(
        runtime_env.page_id,
        "سلام عليكم",
        "وعليكم السلام ورحمة الله وبركاته",
        taught_by_chat_id="8092716067",
        telegram_message_id="301",
    )

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        f"/telegram/webhook/{runtime_env.page_id}",
        data=json.dumps(telegram_update(302, "/remove سلام عليكم")),
        content_type="application/json",
        headers=telegram_headers(runtime_env),
    )

    assert response.status_code == 200
    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "سلام عليكم")
    assert result.response == UNKNOWN_FALLBACK
    assert sent_messages[-1] == (
        runtime_env.page_id,
        "🗑️ Removed taught reply for: سلام عليكم",
        302,
    )


def test_telegram_learned_command_refreshes_from_disk_after_external_teach(runtime_env, monkeypatch):
    sent_messages = []
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_learning", lambda *args, **kwargs: True)
    monkeypatch.setattr(
        runtime_env.app_module.notification_system,
        "send_telegram_text",
        lambda page_id, text, reply_to_message_id=None: sent_messages.append((page_id, text, reply_to_message_id)) or True,
    )

    runtime_env.app_module.ensure_learning_state_loaded(runtime_env.page_id)
    learned_file = runtime_env.runtime_paths.get_interaction_log_dir() / f"{runtime_env.page_id}_learned_replies.jsonl"
    payload = runtime_env.app_module.build_telegram_teach_payload(
        runtime_env.page_id,
        "السلام عليكم فين موقعكم ؟",
        "وعليكم السلام ورحمة الله وبركاته نحن موجودون في قنا، مصر، ونعمل عن بُعد مع العملاء في جميع أنحاء مصر والعالم.",
        taught_by_chat_id="8092716067",
        telegram_message_id="401",
        max_age_days=90,
    )
    runtime_env.app_module.append_jsonl(learned_file, payload)

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        f"/telegram/webhook/{runtime_env.page_id}",
        data=json.dumps(telegram_update(402, "/learned")),
        content_type="application/json",
        headers=telegram_headers(runtime_env),
    )

    assert response.status_code == 200
    assert "السلام عليكم فين موقعكم ؟ => وعليكم السلام ورحمة الله وبركاته نحن موجودون في قنا، مصر، ونعمل عن بُعد مع العملاء في جميع أنحاء مصر والعالم." in sent_messages[-1][1]

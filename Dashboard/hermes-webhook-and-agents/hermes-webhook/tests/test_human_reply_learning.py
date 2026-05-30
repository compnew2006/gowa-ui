import importlib
import json


UNKNOWN_FALLBACK = "لا أملك هذه المعلومة المحددة حالياً. دعني أتحقق مع فريقنا وأعود إليك قريباً."


def update_pages_config(runtime_env, updater):
    config = json.loads(runtime_env.pages_config_file.read_text(encoding="utf-8"))
    updater(config)
    runtime_env.pages_config_file.write_text(json.dumps(config), encoding="utf-8")


def configure_webhook_mocks(runtime_env, monkeypatch):
    public_replies = []

    monkeypatch.setattr(
        runtime_env.app_module,
        "reply_to_facebook_comment",
        lambda page_id, comment_id, message: public_replies.append((page_id, comment_id, message)) or {"id": f"reply-{comment_id}"},
    )
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", lambda payload: None)
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_owner", lambda *args, **kwargs: True)
    return public_replies


def post_page_comment_event(client, page_id, comment_id, message, sender_id, sender_name, parent_id=None):
    payload = {
        "object": "page",
        "entry": [
            {
                "changes": [
                    {
                        "field": "feed",
                        "value": {
                            "item": "comment",
                            "comment_id": comment_id,
                            "message": message,
                            "post_id": f"{page_id}_post-1",
                            "parent_id": parent_id or f"root_{page_id}",
                            "from": {
                                "id": sender_id,
                                "name": sender_name,
                            },
                        },
                    }
                ]
            }
        ],
    }
    return client.post("/webhook", data=json.dumps(payload), content_type="application/json")


def create_unknown_context(runtime_env, comment_id, message, page_id=None, bot_reply_text=UNKNOWN_FALLBACK):
    page = page_id or runtime_env.page_id
    reply_result = runtime_env.app_module.build_response(message, runtime_env.knowledge_text, "Ofuqalmadenah")
    runtime_env.app_module.record_comment_context(
        page,
        comment_id,
        message,
        "Customer",
        "customer-ctx",
        f"{page}_post-1",
        f"root_{page}",
        reply_result,
        bot_reply_text,
    )
    return reply_result


def reload_app_module(runtime_env):
    reloaded = importlib.reload(runtime_env.app_module)
    runtime_env.app_module = reloaded
    return reloaded


def test_first_unknown_comment_gets_exact_fallback(runtime_env, monkeypatch):
    public_replies = configure_webhook_mocks(runtime_env, monkeypatch)
    client = runtime_env.app_module.app.test_client()

    response = post_page_comment_event(
        client,
        runtime_env.page_id,
        "comment-unknown-1",
        "سلام عليكم",
        "customer-1",
        "First Customer",
    )

    assert response.status_code == 200
    assert public_replies[-1] == (
        runtime_env.page_id,
        "comment-unknown-1",
        UNKNOWN_FALLBACK,
    )


def test_human_page_reply_is_learned_immediately(runtime_env, monkeypatch):
    public_replies = configure_webhook_mocks(runtime_env, monkeypatch)
    learned_notifications = []
    monkeypatch.setattr(
        runtime_env.app_module.notification_system,
        "notify_learning",
        lambda page_id, source_question, learned_reply, learned_from_human_id="", parent_comment_id="": learned_notifications.append(
            (page_id, source_question, learned_reply, learned_from_human_id, parent_comment_id)
        ) or True,
    )
    client = runtime_env.app_module.app.test_client()

    first = post_page_comment_event(
        client,
        runtime_env.page_id,
        "comment-learn-1",
        "سلام عليكم",
        "customer-1",
        "First Customer",
    )
    assert first.status_code == 200
    assert public_replies[-1][2] == UNKNOWN_FALLBACK
    context_file = runtime_env.runtime_paths.get_interaction_log_dir() / f"{runtime_env.page_id}_comment_contexts.jsonl"
    context_payload = json.loads(context_file.read_text(encoding="utf-8").strip().splitlines()[-1])
    assert context_payload["bot_reply_text"] == UNKNOWN_FALLBACK
    assert context_payload["used_unknown_fallback"] is True

    learned = post_page_comment_event(
        client,
        runtime_env.page_id,
        "comment-page-1",
        "وعليكم السلام ورحمة الله وبركاته ، ازاي اقدر اساعدك",
        runtime_env.page_id,
        "Ofuqalmadenah",
        parent_id="comment-learn-1",
    )
    assert learned.status_code == 200

    second = post_page_comment_event(
        client,
        runtime_env.page_id,
        "comment-learn-2",
        "سلام عليكم",
        "customer-2",
        "Second Customer",
    )
    assert second.status_code == 200
    assert public_replies[-1] == (
        runtime_env.page_id,
        "comment-learn-2",
        "وعليكم السلام ورحمة الله وبركاته ، ازاي اقدر اساعدك",
    )

    learned_file = runtime_env.runtime_paths.get_interaction_log_dir() / f"{runtime_env.page_id}_learned_replies.jsonl"
    assert learned_file.exists()
    assert "unknown_fallback_learning" in learned_file.read_text(encoding="utf-8")
    assert learned_notifications == [
        (
            runtime_env.page_id,
            "سلام عليكم",
            "وعليكم السلام ورحمة الله وبركاته ، ازاي اقدر اساعدك",
            runtime_env.page_id,
            "comment-learn-1",
        )
    ]


def test_exact_normalized_match_reuses_learned_reply(runtime_env):
    create_unknown_context(runtime_env, "ctx-normalized-1", "إزايكم؟")

    learned = runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-normalized-1",
        "ctx-normalized-1",
        "أهلاً بيك، ازاي اقدر اساعدك؟",
        runtime_env.page_id,
    )

    assert learned is True
    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "ازايكم")
    assert result.response == "أهلاً بيك، ازاي اقدر اساعدك؟"


def test_reordered_message_does_not_match_learned_reply(runtime_env):
    create_unknown_context(runtime_env, "ctx-order-1", "هل عندكم فرع في طنطا")
    runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-order-1",
        "ctx-order-1",
        "حالياً لا يوجد فرع في طنطا.",
        runtime_env.page_id,
    )

    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "فرع في طنطا هل عندكم")
    assert result.response == UNKNOWN_FALLBACK


def test_kb_grounded_answer_overrides_learned_reply(runtime_env):
    create_unknown_context(
        runtime_env,
        "ctx-kb-1",
        "How much for a website?",
        bot_reply_text=UNKNOWN_FALLBACK,
    )
    runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-kb-1",
        "ctx-kb-1",
        "Website design starts from 3,000 EGP.",
        runtime_env.page_id,
    )

    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "How much for a website?")
    assert result.response == "Website design starts from 5,000 EGP."
    assert result.found_in_kb is True


def test_learned_reply_expires_after_90_days(runtime_env, monkeypatch):
    create_unknown_context(runtime_env, "ctx-expire-1", "سلام عليكم")
    runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-expire-1",
        "ctx-expire-1",
        "وعليكم السلام، تحت أمرك.",
        runtime_env.page_id,
    )

    real_now = runtime_env.app_module.utc_now
    future_time = real_now() + runtime_env.app_module.timedelta(days=91)
    monkeypatch.setattr(runtime_env.app_module, "utc_now", lambda: future_time)

    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "سلام عليكم")
    assert result.response == UNKNOWN_FALLBACK


def test_latest_human_reply_replaces_previous_one(runtime_env):
    create_unknown_context(runtime_env, "ctx-latest-1", "سلام عليكم")
    first = runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-latest-1",
        "ctx-latest-1",
        "وعليكم السلام، تحت أمرك.",
        runtime_env.page_id,
    )
    assert first is True

    create_unknown_context(runtime_env, "ctx-latest-2", "سلام عليكم")
    second = runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-latest-2",
        "ctx-latest-2",
        "وعليكم السلام ورحمة الله وبركاته ، ازاي اقدر اساعدك",
        runtime_env.page_id,
    )
    assert second is True

    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "سلام عليكم")
    assert result.response == "وعليكم السلام ورحمة الله وبركاته ، ازاي اقدر اساعدك"

    learned_file = runtime_env.runtime_paths.get_interaction_log_dir() / f"{runtime_env.page_id}_learned_replies.jsonl"
    learned_text = learned_file.read_text(encoding="utf-8")
    assert '"is_active": false' in learned_text
    assert '"superseded_by"' in learned_text


def test_bot_auto_reply_is_not_learned_as_human_reply(runtime_env, monkeypatch):
    public_replies = configure_webhook_mocks(runtime_env, monkeypatch)
    client = runtime_env.app_module.app.test_client()

    first = post_page_comment_event(
        client,
        runtime_env.page_id,
        "comment-bot-1",
        "سلام عليكم",
        "customer-1",
        "First Customer",
    )
    assert first.status_code == 200
    assert public_replies[-1][2] == UNKNOWN_FALLBACK

    auto_reply_event = post_page_comment_event(
        client,
        runtime_env.page_id,
        "comment-bot-echo",
        UNKNOWN_FALLBACK,
        runtime_env.page_id,
        "Ofuqalmadenah",
        parent_id="comment-bot-1",
    )
    assert auto_reply_event.status_code == 200

    second = post_page_comment_event(
        client,
        runtime_env.page_id,
        "comment-bot-2",
        "سلام عليكم",
        "customer-2",
        "Second Customer",
    )
    assert second.status_code == 200
    assert public_replies[-1] == (
        runtime_env.page_id,
        "comment-bot-2",
        UNKNOWN_FALLBACK,
    )


def test_suspicious_human_reply_is_rejected(runtime_env):
    learned_notifications = []
    runtime_env.app_module.notification_system.notify_learning = (
        lambda page_id, source_question, learned_reply, learned_from_human_id="", parent_comment_id="": learned_notifications.append(
            (page_id, source_question, learned_reply, learned_from_human_id, parent_comment_id)
        ) or True
    )
    create_unknown_context(runtime_env, "ctx-suspicious-1", "هل عندكم فرع في طنطا؟")
    learned = runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-suspicious-1",
        "ctx-suspicious-1",
        "نعم، ادخل على https://bad.example الآن.",
        runtime_env.page_id,
    )

    assert learned is False
    assert learned_notifications == []
    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "هل عندكم فرع في طنطا؟")
    assert result.response == UNKNOWN_FALLBACK


def test_case_specific_human_reply_is_rejected(runtime_env):
    create_unknown_context(runtime_env, "ctx-case-1", "المنتج متأخر")
    learned = runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-case-1",
        "ctx-case-1",
        "سنتواصل معك بخصوص طلبك اليوم.",
        runtime_env.page_id,
    )

    assert learned is False
    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "المنتج متأخر")
    assert result.response == UNKNOWN_FALLBACK


def test_learned_reply_persists_across_reload(runtime_env):
    create_unknown_context(runtime_env, "ctx-reload-1", "سلام عليكم")
    runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-reload-1",
        "ctx-reload-1",
        "وعليكم السلام، اهلاً بيك.",
        runtime_env.page_id,
    )

    reloaded = reload_app_module(runtime_env)
    result = reloaded.build_reply_result(runtime_env.page_id, "سلام عليكم")
    assert result.response == "وعليكم السلام، اهلاً بيك."


def test_learned_replies_are_isolated_per_page(runtime_env):
    second_page_id = "999999999999999"

    def add_second_page(config):
        config["pages"][second_page_id] = {
            "page_name": "Second Page",
            "page_id": second_page_id,
            "access_token": "second-access-token",
            "knowledge_file": f"/opt/hermes-webhook/knowledge/{runtime_env.knowledge_file.name}",
            "instagram_enabled": False,
            "instagram_account_id": "",
            "instagram_username": "",
            "human_reply_learning_enabled": True,
            "human_reply_learning_max_age_days": 90,
            "human_reply_learning_match_mode": "exact_normalized",
            "messenger_private_replies_enabled": False,
            "messenger_private_reply_question_types": ["pricing", "location", "hours", "services", "contact"],
            "messenger_private_reply_public_and_private": True,
            "messenger_private_reply_templates": {},
        }

    update_pages_config(runtime_env, add_second_page)

    create_unknown_context(runtime_env, "ctx-page1-1", "سلام عليكم")
    runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "reply-page1-1",
        "ctx-page1-1",
        "وعليكم السلام، صفحة أولى.",
        runtime_env.page_id,
    )

    result_first_page = runtime_env.app_module.build_reply_result(runtime_env.page_id, "سلام عليكم")
    result_second_page = runtime_env.app_module.build_reply_result(second_page_id, "سلام عليكم")

    assert result_first_page.response == "وعليكم السلام، صفحة أولى."
    assert result_second_page.response == UNKNOWN_FALLBACK


def test_build_reply_result_refreshes_external_learned_reply_from_disk(runtime_env):
    runtime_env.app_module.ensure_learning_state_loaded(runtime_env.page_id)
    learned_file = runtime_env.runtime_paths.get_interaction_log_dir() / f"{runtime_env.page_id}_learned_replies.jsonl"
    payload = runtime_env.app_module.build_telegram_teach_payload(
        runtime_env.page_id,
        "السلام عليكم هل عندكم فرع في اسوان ؟",
        "وعليكم السلام ورحمة الله وبركاته حالياً لا يوجد لدينا فرع في أسوان.",
        taught_by_chat_id="8092716067",
        telegram_message_id="disk-refresh-1",
        max_age_days=90,
    )
    runtime_env.app_module.append_jsonl(learned_file, payload)

    result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "السلام عليكم هل عندكم فرع في اسوان ؟")
    assert result.response == "وعليكم السلام ورحمة الله وبركاته حالياً لا يوجد لدينا فرع في أسوان."
    assert result.source_section == "Telegram Teach"

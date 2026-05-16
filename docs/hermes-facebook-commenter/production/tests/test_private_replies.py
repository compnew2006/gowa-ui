import json


def update_page_config(runtime_env, **updates):
    config = json.loads(runtime_env.pages_config_file.read_text(encoding="utf-8"))
    page_config = config["pages"][runtime_env.page_id]
    page_config.update(updates)
    runtime_env.pages_config_file.write_text(json.dumps(config), encoding="utf-8")


def test_render_private_reply_uses_detected_language(runtime_env):
    update_page_config(
        runtime_env,
        messenger_private_reply_templates={
            "pricing": {
                "ar": "مرحباً {customer_name}. {public_reply}",
                "en": "Hello {customer_name}. {public_reply}",
            }
        },
    )

    page_config = runtime_env.app_module.get_page_config(runtime_env.page_id)
    message = runtime_env.app_module.render_private_reply_message(
        page_config,
        "pricing",
        "ar",
        "أحمد",
        "تصميم المواقع يبدأ من 5,000 جنيه مصري.",
    )

    assert message == "مرحباً أحمد. تصميم المواقع يبدأ من 5,000 جنيه مصري."


def test_render_private_reply_falls_back_to_english(runtime_env):
    update_page_config(
        runtime_env,
        messenger_private_reply_templates={
            "location": {
                "en": "Hello {customer_name}. {public_reply}",
            }
        },
    )

    page_config = runtime_env.app_module.get_page_config(runtime_env.page_id)
    message = runtime_env.app_module.render_private_reply_message(
        page_config,
        "location",
        "ar",
        "Sara",
        "We're based in Qena, Egypt.",
    )

    assert message == "Hello Sara. We're based in Qena, Egypt."


def test_render_private_reply_rejects_unsupported_placeholders(runtime_env):
    update_page_config(
        runtime_env,
        messenger_private_reply_templates={
            "pricing": {
                "en": "Hello {customer_name}. {unsupported}",
            }
        },
    )

    page_config = runtime_env.app_module.get_page_config(runtime_env.page_id)

    assert runtime_env.app_module.validate_private_reply_template("Hello {customer_name}. {unsupported}") is False
    assert (
        runtime_env.app_module.render_private_reply_message(
            page_config,
            "pricing",
            "en",
            "Sara",
            "Website design starts from 5,000 EGP.",
        )
        is None
    )


def test_render_private_reply_mirrors_learned_human_reply(runtime_env):
    page_config = runtime_env.app_module.get_page_config(runtime_env.page_id)
    message = runtime_env.app_module.render_private_reply_message(
        page_config,
        "greeting",
        "ar",
        "أحمد",
        "وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك",
        "Human Reply Learning",
    )

    assert message == "وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك"


def test_should_send_private_reply_rejects_page_self_comments(runtime_env):
    update_page_config(runtime_env, messenger_private_replies_enabled=True)
    page_config = runtime_env.app_module.get_page_config(runtime_env.page_id)
    reply_result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "How much for a website?")

    assert (
        runtime_env.app_module.should_send_private_reply(
            page_config,
            reply_result,
            "comment-self",
            runtime_env.page_id,
            runtime_env.page_id,
        )
        is False
    )


def test_should_send_private_reply_allows_learned_greeting(runtime_env):
    update_page_config(runtime_env, messenger_private_replies_enabled=True)
    page_config = runtime_env.app_module.get_page_config(runtime_env.page_id)

    reply_result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "سلام عليكم")
    runtime_env.app_module.record_comment_context(
        runtime_env.page_id,
        "comment-learn-greeting-1",
        "سلام عليكم",
        "Customer",
        "customer-1",
        f"{runtime_env.page_id}_post-1",
        None,
        reply_result,
        reply_result.response,
    )
    runtime_env.app_module.learn_from_human_page_reply(
        runtime_env.page_id,
        "comment-learn-greeting-human",
        "comment-learn-greeting-1",
        "وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك",
        runtime_env.page_id,
    )

    learned_result = runtime_env.app_module.build_reply_result(runtime_env.page_id, "سلام عليكم")

    assert learned_result.source_section == "Human Reply Learning"
    assert (
        runtime_env.app_module.should_send_private_reply(
            page_config,
            learned_result,
            "comment-learn-greeting-2",
            "customer-2",
            runtime_env.page_id,
        )
        is True
    )


def test_feed_comment_webhook_sends_public_and_private_reply(runtime_env, monkeypatch):
    update_page_config(runtime_env, messenger_private_replies_enabled=True)
    public_replies = []
    private_replies = []
    forwarded = []

    monkeypatch.setattr(
        runtime_env.app_module,
        "reply_to_facebook_comment",
        lambda page_id, comment_id, message: public_replies.append((page_id, comment_id, message)) or {"id": "reply-1"},
    )
    monkeypatch.setattr(
        runtime_env.app_module,
        "send_private_reply_to_facebook_comment",
        lambda page_id, comment_id, message: private_replies.append((page_id, comment_id, message)) or {"id": "dm-1"},
    )
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", lambda payload: forwarded.append(payload))
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_owner", lambda *args, **kwargs: True)

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        "/webhook",
        data=json.dumps(
            {
                "object": "page",
                "entry": [
                    {
                        "changes": [
                            {
                                "field": "feed",
                                "value": {
                                    "item": "comment",
                                    "comment_id": "comment-private-1",
                                    "message": "How much for a website?",
                                    "post_id": f"{runtime_env.page_id}_post-1",
                                    "sender_name": "Test Customer",
                                    "sender_id": "customer-1",
                                },
                            }
                        ]
                    }
                ],
            }
        ),
        content_type="application/json",
    )

    assert response.status_code == 200
    assert public_replies == [
        (
            runtime_env.page_id,
            "comment-private-1",
            "Website design starts from 5,000 EGP.",
        )
    ]
    assert private_replies == [
        (
            runtime_env.page_id,
            "comment-private-1",
            "Hi Test Customer, thanks for your comment. Website design starts from 5,000 EGP. Send us a message here if you'd like to continue.",
        )
    ]
    private_log = runtime_env.runtime_paths.get_log_dir() / "private_replied_comments.txt"
    assert private_log.exists()
    assert private_log.read_text(encoding="utf-8").strip() == "comment-private-1"
    assert len(forwarded) == 1


def test_feed_comment_webhook_skips_private_reply_for_unsupported_answer(runtime_env, monkeypatch):
    update_page_config(runtime_env, messenger_private_replies_enabled=True)
    public_replies = []
    private_replies = []

    monkeypatch.setattr(
        runtime_env.app_module,
        "reply_to_facebook_comment",
        lambda page_id, comment_id, message: public_replies.append((page_id, comment_id, message)) or {"id": "reply-1"},
    )
    monkeypatch.setattr(
        runtime_env.app_module,
        "send_private_reply_to_facebook_comment",
        lambda page_id, comment_id, message: private_replies.append((page_id, comment_id, message)) or {"id": "dm-1"},
    )
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", lambda payload: None)
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_owner", lambda *args, **kwargs: True)

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        "/webhook",
        data=json.dumps(
            {
                "object": "page",
                "entry": [
                    {
                        "changes": [
                            {
                                "field": "feed",
                                "value": {
                                    "item": "comment",
                                    "comment_id": "comment-private-2",
                                    "message": "Do you have a branch in Alexandria?",
                                    "post_id": f"{runtime_env.page_id}_post-1",
                                    "sender_name": "Test Customer",
                                    "sender_id": "customer-2",
                                },
                            }
                        ]
                    }
                ],
            }
        ),
        content_type="application/json",
    )

    assert response.status_code == 200
    assert public_replies == [
        (
            runtime_env.page_id,
            "comment-private-2",
            "I don't have that specific information. Let me check with our team and get back to you shortly.",
        )
    ]
    assert private_replies == []


def test_feed_comment_webhook_falls_back_to_direct_messenger(runtime_env, monkeypatch):
    update_page_config(runtime_env, messenger_private_replies_enabled=True)
    public_replies = []
    direct_replies = []

    monkeypatch.setattr(
        runtime_env.app_module,
        "reply_to_facebook_comment",
        lambda page_id, comment_id, message: public_replies.append((page_id, comment_id, message)) or {"id": "reply-1"},
    )
    monkeypatch.setattr(
        runtime_env.app_module,
        "send_private_reply_to_facebook_comment",
        lambda page_id, comment_id, message: {
            "error": {
                "message": "Unsupported post request",
                "type": "GraphMethodException",
                "code": 100,
                "error_subcode": 33,
            }
        },
    )
    monkeypatch.setattr(
        runtime_env.app_module,
        "send_direct_messenger_reply",
        lambda page_id, recipient_id, message: direct_replies.append((page_id, recipient_id, message)) or {"id": "mid.1"},
    )
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", lambda payload: None)
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_owner", lambda *args, **kwargs: True)

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        "/webhook",
        data=json.dumps(
            {
                "object": "page",
                "entry": [
                    {
                        "changes": [
                            {
                                "field": "feed",
                                "value": {
                                    "item": "comment",
                                    "comment_id": "comment-private-fallback-1",
                                    "message": "How much for a website?",
                                    "post_id": f"{runtime_env.page_id}_post-1",
                                    "sender_name": "Test Customer",
                                    "sender_id": "customer-dm-1",
                                },
                            }
                        ]
                    }
                ],
            }
        ),
        content_type="application/json",
    )

    assert response.status_code == 200
    assert public_replies == [
        (
            runtime_env.page_id,
            "comment-private-fallback-1",
            "Website design starts from 5,000 EGP.",
        )
    ]
    assert direct_replies == [
        (
            runtime_env.page_id,
            "customer-dm-1",
            "Hi Test Customer, thanks for your comment. Website design starts from 5,000 EGP. Send us a message here if you'd like to continue.",
        )
    ]


def test_send_private_reply_uses_page_messages_endpoint(runtime_env, monkeypatch):
    calls = []

    class FakeResponse:
        status_code = 200
        text = '{"message_id":"mid.123"}'

        def json(self):
            return {"message_id": "mid.123"}

    def fake_post(url, params=None, json=None, timeout=10):
        calls.append(
            {
                "url": url,
                "params": dict(params or {}),
                "json": json,
                "timeout": timeout,
            }
        )
        return FakeResponse()

    monkeypatch.setattr(runtime_env.app_module.requests, "post", fake_post)

    result = runtime_env.app_module.send_private_reply_to_facebook_comment(
        runtime_env.page_id,
        "comment-private-live-1",
        "Private hello",
    )

    assert result == {"message_id": "mid.123"}
    assert calls == [
        {
            "url": "https://graph.facebook.com/v19.0/895247390337022/messages",
            "params": {"access_token": "test-access-token"},
            "json": {
                "recipient": {"comment_id": "comment-private-live-1"},
                "message": {"text": "Private hello"},
            },
            "timeout": 10,
        }
    ]


def test_feed_comment_webhook_sends_private_reply_for_learned_greeting(runtime_env, monkeypatch):
    update_page_config(runtime_env, messenger_private_replies_enabled=True)
    public_replies = []
    private_replies = []

    monkeypatch.setattr(
        runtime_env.app_module,
        "reply_to_facebook_comment",
        lambda page_id, comment_id, message: public_replies.append((page_id, comment_id, message)) or {"id": f"reply-{comment_id}"},
    )
    monkeypatch.setattr(
        runtime_env.app_module,
        "send_private_reply_to_facebook_comment",
        lambda page_id, comment_id, message: private_replies.append((page_id, comment_id, message)) or {"id": f"dm-{comment_id}"},
    )
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", lambda payload: None)
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_owner", lambda *args, **kwargs: True)

    client = runtime_env.app_module.app.test_client()

    first = client.post(
        "/webhook",
        data=json.dumps(
            {
                "object": "page",
                "entry": [
                    {
                        "changes": [
                            {
                                "field": "feed",
                                "value": {
                                    "item": "comment",
                                    "comment_id": "comment-greeting-1",
                                    "message": "سلام عليكم",
                                    "post_id": f"{runtime_env.page_id}_post-1",
                                    "sender_name": "Greeting Customer",
                                    "sender_id": "customer-greeting-1",
                                },
                            }
                        ]
                    }
                ],
            }
        ),
        content_type="application/json",
    )

    learned = client.post(
        "/webhook",
        data=json.dumps(
            {
                "object": "page",
                "entry": [
                    {
                        "changes": [
                            {
                                "field": "feed",
                                "value": {
                                    "item": "comment",
                                    "comment_id": "comment-greeting-human-1",
                                    "message": "وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك",
                                    "post_id": f"{runtime_env.page_id}_post-1",
                                    "parent_id": "comment-greeting-1",
                                    "from": {
                                        "id": runtime_env.page_id,
                                        "name": "Ofuqalmadenah",
                                    },
                                },
                            }
                        ]
                    }
                ],
            }
        ),
        content_type="application/json",
    )

    second = client.post(
        "/webhook",
        data=json.dumps(
            {
                "object": "page",
                "entry": [
                    {
                        "changes": [
                            {
                                "field": "feed",
                                "value": {
                                    "item": "comment",
                                    "comment_id": "comment-greeting-2",
                                    "message": "سلام عليكم",
                                    "post_id": f"{runtime_env.page_id}_post-1",
                                    "sender_name": "Greeting Customer Two",
                                    "sender_id": "customer-greeting-2",
                                },
                            }
                        ]
                    }
                ],
            }
        ),
        content_type="application/json",
    )

    assert first.status_code == 200
    assert learned.status_code == 200
    assert second.status_code == 200
    assert public_replies[0] == (
        runtime_env.page_id,
        "comment-greeting-1",
        "لا أملك هذه المعلومة المحددة حالياً. دعني أتحقق مع فريقنا وأعود إليك قريباً.",
    )
    assert public_replies[-1] == (
        runtime_env.page_id,
        "comment-greeting-2",
        "وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك",
    )
    assert private_replies == [
        (
            runtime_env.page_id,
            "comment-greeting-2",
            "وعليكم السلام ورحمة الله وبركاته كيف اقدر اساعدك",
        )
    ]


def test_feed_comment_webhook_does_not_send_private_reply_twice(runtime_env, monkeypatch):
    update_page_config(runtime_env, messenger_private_replies_enabled=True)
    private_replies = []

    monkeypatch.setattr(runtime_env.app_module, "reply_to_facebook_comment", lambda *args, **kwargs: {"id": "reply-1"})
    monkeypatch.setattr(
        runtime_env.app_module,
        "send_private_reply_to_facebook_comment",
        lambda page_id, comment_id, message: private_replies.append((page_id, comment_id, message)) or {"id": "dm-1"},
    )
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", lambda payload: None)
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_owner", lambda *args, **kwargs: True)

    client = runtime_env.app_module.app.test_client()
    payload = {
        "object": "page",
        "entry": [
            {
                "changes": [
                    {
                        "field": "feed",
                        "value": {
                            "item": "comment",
                            "comment_id": "comment-private-3",
                            "message": "How much for a website?",
                            "post_id": f"{runtime_env.page_id}_post-1",
                            "sender_name": "Test Customer",
                            "sender_id": "customer-3",
                        },
                    }
                ]
            }
        ],
    }

    first = client.post("/webhook", data=json.dumps(payload), content_type="application/json")
    second = client.post("/webhook", data=json.dumps(payload), content_type="application/json")

    assert first.status_code == 200
    assert second.status_code == 200
    assert len(private_replies) == 1


def test_feed_comment_webhook_skips_private_reply_when_public_reply_fails(runtime_env, monkeypatch):
    update_page_config(runtime_env, messenger_private_replies_enabled=True)
    private_replies = []

    monkeypatch.setattr(runtime_env.app_module, "reply_to_facebook_comment", lambda *args, **kwargs: {"error": "reply failed"})
    monkeypatch.setattr(
        runtime_env.app_module,
        "send_private_reply_to_facebook_comment",
        lambda page_id, comment_id, message: private_replies.append((page_id, comment_id, message)) or {"id": "dm-1"},
    )
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", lambda payload: None)
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_owner", lambda *args, **kwargs: True)

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        "/webhook",
        data=json.dumps(
            {
                "object": "page",
                "entry": [
                    {
                        "changes": [
                            {
                                "field": "feed",
                                "value": {
                                    "item": "comment",
                                    "comment_id": "comment-private-4",
                                    "message": "How much for a website?",
                                    "post_id": f"{runtime_env.page_id}_post-1",
                                    "sender_name": "Test Customer",
                                    "sender_id": "customer-4",
                                },
                            }
                        ]
                    }
                ],
            }
        ),
        content_type="application/json",
    )

    assert response.status_code == 200
    assert private_replies == []


def test_page_messaging_events_are_forwarded(runtime_env, monkeypatch):
    forwarded = []
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", lambda payload: forwarded.append(payload))

    client = runtime_env.app_module.app.test_client()
    response = client.post(
        "/webhook",
        data=json.dumps(
            {
                "object": "page",
                "entry": [
                    {
                        "id": runtime_env.page_id,
                        "messaging": [
                            {
                                "sender": {"id": "customer-5"},
                                "recipient": {"id": runtime_env.page_id},
                                "timestamp": 1234567890,
                                "message": {"text": "I sent you a DM"},
                            }
                        ],
                    }
                ],
            }
        ),
        content_type="application/json",
    )

    assert response.status_code == 200
    assert forwarded == [
        {
            "type": "page_messaging",
            "page_id": runtime_env.page_id,
            "sender_id": "customer-5",
            "recipient_id": runtime_env.page_id,
            "message": "I sent you a DM",
            "timestamp": 1234567890,
            "raw": {
                "sender": {"id": "customer-5"},
                "recipient": {"id": runtime_env.page_id},
                "timestamp": 1234567890,
                "message": {"text": "I sent you a DM"},
            },
        }
    ]

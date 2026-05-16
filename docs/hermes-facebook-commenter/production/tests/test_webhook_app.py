import json


def test_health_route(runtime_env):
    client = runtime_env.app_module.app.test_client()

    response = client.get("/health")

    assert response.status_code == 200
    payload = response.get_json()
    assert payload["status"] == "healthy"
    assert payload["pages_configured"] == 1
    assert payload["pages"] == [runtime_env.page_id]


def test_debug_reply_preview_route(runtime_env):
    client = runtime_env.app_module.app.test_client()

    response = client.post(
        "/debug/reply-preview",
        json={"page_id": runtime_env.page_id, "message": "كم سعر تصميم موقع؟"},
    )

    assert response.status_code == 200
    payload = response.get_json()
    assert payload["response"] == "تصميم المواقع يبدأ من 5,000 جنيه مصري."
    assert payload["language"] == "ar"
    assert payload["question_type"] == "pricing"
    assert payload["found_in_kb"] is True


def test_debug_reply_preview_hidden_without_test_mode(runtime_env, monkeypatch):
    monkeypatch.setenv("HERMES_TEST_MODE", "0")
    client = runtime_env.app_module.app.test_client()

    response = client.post(
        "/debug/reply-preview",
        json={"page_id": runtime_env.page_id, "message": "How much for a website?"},
    )

    assert response.status_code == 404


def test_feed_comment_webhook_replies_and_logs_interaction(runtime_env, monkeypatch):
    replies = []
    forwarded = []
    notified = []

    def fake_reply_to_facebook_comment(page_id, comment_id, message):
        replies.append((page_id, comment_id, message))
        return {"id": "reply-123"}

    def fake_forward(payload):
        forwarded.append(payload)

    def fake_notify(page_id, customer_name, question, agent_response):
        notified.append((page_id, customer_name, question, agent_response))
        return True

    monkeypatch.setattr(runtime_env.app_module, "reply_to_facebook_comment", fake_reply_to_facebook_comment)
    monkeypatch.setattr(runtime_env.app_module, "forward_to_hermes", fake_forward)
    monkeypatch.setattr(runtime_env.app_module.notification_system, "notify_owner", fake_notify)

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
                            "comment_id": "comment-1",
                            "message": "How much for a website?",
                            "post_id": "{page_id}_post-1".format(page_id=runtime_env.page_id),
                            "sender_name": "Test Customer",
                            "sender_id": "customer-1",
                        },
                    }
                ]
            }
        ],
    }

    response = client.post("/webhook", data=json.dumps(payload), content_type="application/json")

    assert response.status_code == 200
    assert replies == [
        (
            runtime_env.page_id,
            "comment-1",
            "Website design starts from 5,000 EGP.",
        )
    ]
    assert len(forwarded) == 1
    assert forwarded[0]["message"] == "How much for a website?"
    assert notified[0][3] == "Website design starts from 5,000 EGP."

    interaction_log = runtime_env.runtime_paths.get_interaction_log_dir() / "{page_id}.jsonl".format(page_id=runtime_env.page_id)
    assert interaction_log.exists()
    lines = interaction_log.read_text(encoding="utf-8").strip().splitlines()
    assert len(lines) == 1
    payload = json.loads(lines[0])
    assert payload["comment"] == "How much for a website?"
    assert payload["response"] == "Website design starts from 5,000 EGP."

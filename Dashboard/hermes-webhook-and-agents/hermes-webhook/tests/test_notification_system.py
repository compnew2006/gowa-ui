from types import SimpleNamespace


def test_send_telegram_text_retries_without_reply_target(runtime_env, monkeypatch):
    calls = []

    class FakeResponse:
        def __init__(self, status_code, payload):
            self.status_code = status_code
            self._payload = payload
            self.text = str(payload)

        def json(self):
            return self._payload

    def fake_post(url, json=None, timeout=10):
        calls.append({"url": url, "json": dict(json or {}), "timeout": timeout})
        if len(calls) == 1:
            return FakeResponse(400, {"ok": False, "description": "Bad Request: message to be replied not found"})
        return FakeResponse(200, {"ok": True, "result": {"message_id": 1}})

    monkeypatch.setattr(runtime_env.notification_system.requests, "post", fake_post)
    notification_service = runtime_env.notification_system.notification_system

    result = notification_service.send_telegram_text(
        runtime_env.page_id,
        "hello",
        reply_to_message_id=123,
    )

    assert result is True
    assert len(calls) == 2
    assert calls[0]["json"]["reply_to_message_id"] == 123
    assert "reply_to_message_id" not in calls[1]["json"]

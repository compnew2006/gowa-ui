from flask import Flask, request, jsonify
import os
import requests
import json
import logging

log_format = logging.Formatter('%(asctime)s - %(levelname)s - %(message)s')

stream_handler = logging.StreamHandler()
stream_handler.setFormatter(log_format)

file_handler = logging.FileHandler(os.path.expanduser('~/hermes_webhook_basic.log'))
file_handler.setFormatter(log_format)

logging.basicConfig(level=logging.INFO, handlers=[stream_handler, file_handler])

logger = logging.getLogger(__name__)

app = Flask(__name__)

GRAPH_URL = "https://graph.facebook.com/v19.0"
PAGE_TOKEN = os.environ.get("FB_PAGE_ACCESS_TOKEN")
VERIFY_TOKEN = os.environ.get("FB_WEBHOOK_VERIFY_TOKEN", "hermes_facebook_verify")
HERMES_API_URL = os.environ.get("HERMES_API_URL", "http://localhost:8080")


@app.route('/webhook', methods=['GET'])
def verify_webhook():
    mode = request.args.get('hub.mode')
    token = request.args.get('hub.verify_token')
    challenge = request.args.get('hub.challenge')

    if mode == 'subscribe' and token == VERIFY_TOKEN:
        logger.info("✓ Webhook verified successfully")
        return challenge, 200
    else:
        logger.warning("✗ Webhook verification failed")
        return "Forbidden", 403


@app.route('/webhook', methods=['POST'])
def webhook():
    data = request.json

    if data.get('object') == 'page':
        for entry in data.get('entry', []):
            for change in entry.get('changes', []):
                if change.get('field') == 'comments':
                    handle_comment_event(change.get('value'))
                elif change.get('field') == 'feed':
                    handle_feed_event(change.get('value'))

    return "OK", 200


def handle_comment_event(value):
    if value is None:
        logger.warning("⚠ handle_comment_event called with None value")
        return

    comment_id = value.get('id')
    post_id = value.get('post_id')
    message = value.get('message', '')
    from_name = value.get('from', {}).get('name', 'Unknown')

    logger.info(f"📩 New comment from {from_name}: {message}")

    try:
        requests.post(f"{HERMES_API_URL}/webhook/facebook", json={
            "type": "new_comment",
            "comment_id": comment_id,
            "post_id": post_id,
            "message": message,
            "from": from_name,
            "raw": value
        })
        logger.info("✓ Forwarded to Hermes")
    except Exception as e:
        logger.error(f"✗ Failed to forward to Hermes: {e}")


def handle_feed_event(value):
    if value is None:
        logger.warning("⚠ handle_feed_event called with None value")
        return

    if value.get('item') == 'comment':
        comment_id = value.get('comment_id')
        message = value.get('message', '')
        post_id = value.get('post_id')

        logger.info(f"📩 New comment (feed event): {message}")

        try:
            requests.post(f"{HERMES_API_URL}/webhook/facebook", json={
                "type": "new_comment",
                "comment_id": comment_id,
                "post_id": post_id,
                "message": message,
                "raw": value
            })
        except Exception as e:
            logger.error(f"✗ Failed to forward: {e}")


if __name__ == '__main__':
    port = int(os.environ.get('PORT', 5000))
    logger.info(f"🚀 Facebook Webhook Server running on port {port}")
    logger.info(f"📡 Webhook URL: https://your-domain.com/webhook")
    logger.info(f"🔑 Verify Token: {VERIFY_TOKEN}")
    app.run(host='0.0.0.0', port=port, debug=False)

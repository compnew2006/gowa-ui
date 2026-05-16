#!/usr/bin/env python3
"""
Production Facebook Webhook Server for Hermes
Run with: gunicorn --bind 0.0.0.0:5000 --workers 4 facebook_webhook_gunicorn:app
"""

from flask import Flask, request, jsonify
import os
import requests
import logging
from pathlib import Path

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/var/log/hermes/facebook_webhook.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

app = Flask(__name__)

# Configuration
GRAPH_URL = "https://graph.facebook.com/v19.0"
PAGE_TOKEN = os.environ.get("FB_PAGE_ACCESS_TOKEN")
VERIFY_TOKEN = os.environ.get("FB_WEBHOOK_VERIFY_TOKEN", "hermes_facebook_verify")
HERMES_API_URL = os.environ.get("HERMES_API_URL", "http://localhost:8080")

# Webhook log file
WEBHOOK_LOG = Path.home() / ".hermes" / "webhook_events.log"

def log_event(event_data):
    """Log webhook events to file"""
    WEBHOOK_LOG.parent.mkdir(parents=True, exist_ok=True)
    with open(WEBHOOK_LOG, 'a') as f:
        f.write(f"{event_data}\n")

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({
        "status": "healthy",
        "service": "facebook-webhook",
        "page_configured": bool(PAGE_TOKEN)
    }), 200

@app.route('/webhook', methods=['GET'])
def verify_webhook():
    """Meta webhook verification"""
    mode = request.args.get('hub.mode')
    token = request.args.get('hub.verify_token')
    challenge = request.args.get('hub.challenge')

    if mode == 'subscribe' and token == VERIFY_TOKEN:
        logger.info("✓ Webhook verified successfully")
        return challenge, 200
    else:
        logger.warning(f"✗ Webhook verification failed: mode={mode}, token={token}")
        return "Forbidden", 403

@app.route('/webhook', methods=['POST'])
def webhook():
    """Receive webhook events from Meta"""
    data = request.json

    # Log the raw event
    log_event(f"EVENT: {json.dumps(data)}")

    # Check if this is a page update
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

    return "OK", 200

def handle_comment_event(value):
    """Process new comment event"""
    comment_id = value.get('id')
    post_id = value.get('post_id')
    message = value.get('message', '')
    from_name = value.get('from', {}).get('name', 'Unknown')
    verb = value.get('verb', 'added')

    logger.info(f"📩 Comment {verb}: {from_name} - {message}")
    log_event(f"COMMENT: {from_name}: {message}")

    # Forward to Hermes for processing
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

def handle_feed_event(value):
    """Process new post/feed event"""
    if value.get('item') == 'comment':
        comment_id = value.get('comment_id')
        message = value.get('message', '')
        post_id = value.get('post_id')
        from_name = value.get('sender_name', 'Unknown')

        logger.info(f"📩 Feed comment: {from_name} - {message}")

        payload = {
            "type": "new_comment",
            "comment_id": comment_id,
            "post_id": post_id,
            "message": message,
            "from": from_name,
            "raw": value
        }

        forward_to_hermes(payload)

def forward_to_hermes(payload):
    """Forward event to Hermes API"""
    try:
        response = requests.post(
            f"{HERMES_API_URL}/webhook/facebook",
            json=payload,
            timeout=5
        )
        if response.status_code == 200:
            logger.info("✓ Forwarded to Hermes successfully")
        else:
            logger.warning(f"⚠ Hermes returned: {response.status_code}")
    except requests.exceptions.RequestException as e:
        logger.error(f"✗ Failed to forward to Hermes: {e}")
        # Save to queue for retry
        queue_for_retry(payload)

def queue_for_retry(payload):
    """Save failed events for retry"""
    queue_dir = Path.home() / ".hermes" / "webhook_queue"
    queue_dir.mkdir(parents=True, exist_ok=True)

    import time
    timestamp = int(time.time())
    queue_file = queue_dir / f"{timestamp}_{payload['comment_id']}.json"

    with open(queue_file, 'w') as f:
        json.dump(payload, f)

    logger.info(f"📦 Queued for retry: {queue_file}")

@app.errorhandler(500)
def server_error(e):
    logger.error(f"Server error: {e}")
    return jsonify({"error": "Internal server error"}), 500

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 5000))
    logger.info(f"🚀 Facebook Webhook Server starting on port {port}")
    logger.info(f"🔑 Verify Token: {VERIFY_TOKEN}")
    logger.info(f"📡 Webhook URL: https://your-subdomain.ofuqalmadenah.com/webhook")
    app.run(host='0.0.0.0', port=port, debug=False)

#!/usr/bin/env python3
"""
Multi-Business Facebook Webhook Server
Handles multiple Facebook pages with separate business contexts
"""

from flask import Flask, request, jsonify
import os
import sys
import json
import logging
import requests
from pathlib import Path

# Add plugins directory to path
sys.path.insert(0, str(Path.home() / ".hermes" / "plugins"))

from multi_business_facebook import manager

app = Flask(__name__)

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/var/log/hermes/multi_business_webhook.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

# Configuration
VERIFY_TOKEN = os.environ.get("FB_WEBHOOK_VERIFY_TOKEN", "hermes_multi_business_verify")
HERMES_API_URL = os.environ.get("HERMES_API_URL", "http://localhost:8080")

# Webhook log file
WEBHOOK_LOG_DIR = Path.home() / ".hermes" / "webhook_events"
WEBHOOK_LOG_DIR.mkdir(parents=True, exist_ok=True)

@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    businesses = manager.list_businesses()
    return jsonify({
        "status": "healthy",
        "service": "multi-business-facebook-webhook",
        "total_businesses": len(businesses),
        "businesses": businesses
    }), 200

@app.route('/businesses', methods=['GET'])
def list_businesses():
    """List all configured businesses"""
    businesses = manager.list_businesses()
    return jsonify({
        "total": len(businesses),
        "businesses": businesses
    }), 200

@app.route('/businesses/<business_id>', methods=['GET'])
def get_business(business_id):
    """Get business details"""
    business_info = manager.get_business_info(business_id)
    if not business_info:
        return jsonify({"error": "Business not found"}), 404

    return jsonify(business_info), 200

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
    log_event("raw", data)

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
    page_id = value.get('page_id')

    # Find which business this page belongs to
    business = manager.get_business_by_page_id(page_id)

    if not business:
        logger.error(f"✗ No business found for page_id: {page_id}")
        return

    comment_id = value.get('id')
    post_id = value.get('post_id')
    message = value.get('message', '')
    from_name = value.get('from', {}).get('name', 'Unknown')
    verb = value.get('verb', 'added')

    logger.info(f"📩 [{business.name}] Comment {verb}: {from_name} - {message}")
    log_event(f"business_{business.business_id}", {
        "type": "comment",
        "from": from_name,
        "message": message,
        "timestamp": value.get('created_time')
    })

    # Auto-reply if enabled
    if business.auto_reply:
        try:
            # Generate contextual reply based on business knowledge
            reply = manager.generate_reply(
                business.business_id,
                message,
                from_name
            )

            # Send reply
            response = manager.reply_to_comment(
                business.business_id,
                comment_id,
                reply
            )

            if 'id' in response:
                logger.info(f"✓ [{business.name}] Replied to {from_name}")
                log_event(f"business_{business.business_id}", {
                    "type": "reply",
                    "to": from_name,
                    "message": reply,
                    "success": True
                })
            else:
                logger.error(f"✗ [{business.name}] Failed to reply: {response}")
                log_event(f"business_{business.business_id}", {
                    "type": "reply",
                    "to": from_name,
                    "success": False,
                    "error": response
                })

            # Learn from this interaction
            manager.learn_from_interaction(business.business_id, {
                "message": message,
                "from": from_name,
                "timestamp": value.get('created_time')
            })

        except Exception as e:
            logger.error(f"✗ [{business.name}] Error processing comment: {e}")

    # Forward to Hermes for additional processing
    forward_to_hermes(business.business_id, {
        "type": "new_comment",
        "business_id": business.business_id,
        "business_name": business.name,
        "comment_id": comment_id,
        "post_id": post_id,
        "message": message,
        "from": from_name,
        "verb": verb
    })

def handle_feed_event(value):
    """Process new post/feed event"""
    if value.get('item') == 'comment':
        page_id = value.get('page_id')

        # Find business
        business = manager.get_business_by_page_id(page_id)
        if not business:
            logger.error(f"✗ No business found for page_id: {page_id}")
            return

        comment_id = value.get('comment_id')
        message = value.get('message', '')
        post_id = value.get('post_id')
        from_name = value.get('sender_name', 'Unknown')

        logger.info(f"📩 [{business.name}] Feed comment: {from_name} - {message}")

        # Auto-reply if enabled
        if business.auto_reply:
            try:
                reply = manager.generate_reply(
                    business.business_id,
                    message,
                    from_name
                )

                response = manager.reply_to_comment(
                    business.business_id,
                    comment_id,
                    reply
                )

                if 'id' in response:
                    logger.info(f"✓ [{business.name}] Replied via feed webhook")
                else:
                    logger.error(f"✗ [{business.name}] Feed reply failed: {response}")

            except Exception as e:
                logger.error(f"✗ [{business.name}] Feed webhook error: {e}")

def forward_to_hermes(business_id, payload):
    """Forward event to Hermes API"""
    try:
        response = requests.post(
            f"{HERMES_API_URL}/webhook/facebook",
            json=payload,
            timeout=5
        )
        if response.status_code == 200:
            logger.info(f"✓ Forwarded to Hermes for {business_id}")
        else:
            logger.warning(f"⚠ Hermes returned: {response.status_code}")
    except Exception as e:
        logger.error(f"✗ Failed to forward to Hermes: {e}")

def log_event(event_type, data):
    """Log webhook events to file"""
    import time
    timestamp = int(time.time())
    log_file = WEBHOOK_LOG_DIR / f"{event_type}_{timestamp}.json"

    with open(log_file, 'w') as f:
        json.dump(data, f, indent=2)

@app.errorhandler(500)
def server_error(e):
    logger.error(f"Server error: {e}")
    return jsonify({"error": "Internal server error"}), 500

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 5000))
    logger.info(f"🚀 Multi-Business Facebook Webhook Server starting on port {port}")
    logger.info(f"🔑 Verify Token: {VERIFY_TOKEN}")
    logger.info(f"📊 Managing {len(manager.businesses)} businesses")
    logger.info(f"📡 Webhook URL: https://your-subdomain.ofuqalmadenah.com/webhook")
    app.run(host='0.0.0.0', port=port, debug=False)

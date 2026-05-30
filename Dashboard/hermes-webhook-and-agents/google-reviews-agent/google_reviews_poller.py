#!/usr/bin/env python3
"""
Google Reviews Poller - Main Service
Polls for new reviews and generates auto-replies
Multi-business/locations support
"""

import os
import sys
import json
import logging
import time
import subprocess
from pathlib import Path
from datetime import datetime, timedelta

# Add notification system to path
sys.path.insert(0, '/opt/hermes-webhook')
from notification_system import notification_system

from google_api_client import gmb_client

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/var/log/hermes/google_reviews_poller.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

# Configuration
CONFIG_FILE = Path('/opt/google-reviews-agent/locations_config.json')
LOG_DIR = Path('/opt/google-reviews-agent/logs')
REPLIED_REVIEWS_FILE = LOG_DIR / 'replied_reviews.txt'
POLLING_INTERVAL_MINUTES = int(os.environ.get('POLLING_INTERVAL_MINUTES', '10'))

# Track replied reviews
REPLIED_REVIEWS = set()

def load_replied_reviews():
    """Load already replied reviews"""
    global REPLIED_REVIEWS
    try:
        if REPLIED_REVIEWS_FILE.exists():
            with open(REPLIED_REVIEWS_FILE, 'r') as f:
                REPLIED_REVIEWS = set(line.strip() for line in f if line.strip())
            logger.info(f"✅ Loaded {len(REPLIED_REVIEWS)} previously replied reviews")
    except Exception as e:
        logger.error(f"❌ Failed to load replied reviews: {e}")
        REPLIED_REVIEWS = set()

def save_replied_review(review_id):
    """Save a replied review"""
    global REPLIED_REVIEWS
    REPLIED_REVIEWS.add(review_id)
    try:
        with open(REPLIED_REVIEWS_FILE, 'a') as f:
            f.write(f"{review_id}\n")
    except Exception as e:
        logger.error(f"❌ Failed to save replied review: {e}")

def has_already_replied(review_id):
    """Check if we've already replied to this review"""
    return review_id in REPLIED_REVIEWS

def load_locations_config():
    """Load multi-location configuration"""
    try:
        with open(CONFIG_FILE, 'r') as f:
            config = json.load(f)
            logger.info(f"✅ Loaded {len(config.get('locations', {}))} locations")
            return config
    except Exception as e:
        logger.error(f"❌ Failed to load locations config: {e}")
        return {"locations": {}}

def get_location_config(location_id):
    """Get configuration for specific location"""
    config = load_locations_config()
    return config.get("locations", {}).get(location_id)

def load_knowledge_base(location_id):
    """Load business knowledge for specific location - checks Facebook first if linked"""
    try:
        location_config = get_location_config(location_id)
        if not location_config:
            logger.error(f"❌ No config found for location {location_id}")
            return ""

        # Check if this location is linked to a Facebook page
        facebook_page_id = location_config.get('facebook_page_id')

        if facebook_page_id:
            logger.info(f"🔗 Location {location_id} linked to Facebook page {facebook_page_id}")

            # Load Facebook page's knowledge base
            fb_knowledge_file = Path(f"/opt/hermes-webhook/knowledge/{facebook_page_id}_knowledge.md")

            if fb_knowledge_file.exists():
                with open(fb_knowledge_file, 'r', encoding='utf-8') as f:
                    content = f.read()
                logger.info(f"✅ Loaded Facebook knowledge base for {location_id} (shared with FB page {facebook_page_id})")
                return content
            else:
                logger.warning(f"⚠️ Facebook knowledge file not found: {fb_knowledge_file}")
                logger.info(f"💡 Falling back to location-specific knowledge file")

        # Use location-specific knowledge file (fallback or primary)
        knowledge_file = Path(location_config.get('knowledge_file', ''))
        if not knowledge_file.exists():
            logger.warning(f"⚠️ Knowledge file not found: {knowledge_file}")
            return ""

        with open(knowledge_file, 'r', encoding='utf-8') as f:
            content = f.read()

        if facebook_page_id:
            logger.info(f"✅ Loaded location-specific knowledge for {location_id} (fallback from FB)")
        else:
            logger.info(f"✅ Loaded knowledge for {location_id}")

        return content

    except Exception as e:
        logger.error(f"❌ Failed to load knowledge base: {e}")
        return ""

def generate_auto_reply(location_id, review):
    """Generate automatic reply using Hermes AI"""
    try:
        # Get location config
        location_config = get_location_config(location_id)
        if not location_config:
            logger.warning(f"⚠️ No config for location {location_id}")
            return None

        business_name = location_config.get('business_name', 'this business')

        # Load knowledge
        knowledge = load_knowledge_base(location_id)

        # Extract review info
        reviewer_name = review.get('reviewer', {}).get('displayName', 'Customer')
        rating = review.get('starRating', 0)
        comment = review.get('comment', '')
        review_text = f"Rating: {rating}/5 stars - {comment}"

        # Create system prompt
        system_prompt = f"""You are a professional customer service representative for {business_name}.

BUSINESS KNOWLEDGE:
{knowledge}

REVIEW TO REPLY TO:
- Customer: {reviewer_name}
- Rating: {rating}/5 stars
- Review: {comment}

INSTRUCTIONS:
- Generate a polite, professional reply to this review
- If rating is 4-5 stars: Thank them and mention specific points if applicable
- If rating is 1-3 stars: Apologize, address concerns, and offer to make it right
- Be authentic and human-sounding (not robotic)
- Keep it under 100 words
- Detect language and reply in the same language
- Use ONE emoji at most (and only if it fits naturally)
- Don't mention you're an AI
- Sign off with the business name naturally

Generate your reply:"""

        # Call Hermes CLI
        result = subprocess.run(
            ['hermes', 'chat', '--query', system_prompt],
            capture_output=True,
            text=True,
            timeout=30,
            cwd='/tmp'
        )

        if result.returncode == 0:
            response_text = result.stdout.strip()

            # Clean up Hermes output
            lines = response_text.split('\n')
            cleaned_response = []

            for line in lines:
                # Skip UI elements
                if any(skip in line for skip in ['Hermes Agent', 'Session:', 'Duration:', 'Available', '╭', '╰', '│', '═']):
                    continue
                if '⚕ Hermes' in line or cleaned_response:
                    if line.strip() and not any(skip in line for skip in ['⚕ Hermes', '───']):
                        cleaned_response.append(line.strip())

            if cleaned_response:
                final_response = '\n'.join(cleaned_response).strip()
                logger.info(f"✅ Hermes AI generated reply for {reviewer_name}")
                return final_response

        logger.warning(f"Hermes CLI returned non-zero: {result.returncode}")
    except subprocess.TimeoutExpired:
        logger.error("Hermes CLI timed out")
    except Exception as e:
        logger.error(f"❌ Failed to use Hermes AI: {e}")

    # Fallback replies
    rating = review.get('starRating', 0)
    reviewer_name = review.get('reviewer', {}).get('displayName', 'Customer')
    business_name = location_config.get('business_name', 'us')

    if rating >= 4:
        return f"Thank you {reviewer_name} for the wonderful review! We're so glad you had a great experience with {business_name}. We look forward to serving you again soon! 😊"

    return f"Thank you {reviewer_name} for your feedback. We're sorry to hear about your experience and would love the chance to make it right. Please contact us directly so we can address your concerns."

def save_interaction_for_learning(location_id, reviewer_name, rating, comment, reply):
    """Save interaction to business knowledge (both location and Facebook if linked)"""
    try:
        location_config = get_location_config(location_id)
        if not location_config:
            return

        learning_entry = f"""
## Customer Review Interaction (Google Maps)

**Date:** {json.dumps({"timestamp": int(time.time())})}

**Customer:** {reviewer_name}

**Rating:** {rating}/5 stars

**Review:** {comment}

**Response:** {reply}

---

"""

        # Check if linked to Facebook page
        facebook_page_id = location_config.get('facebook_page_id')

        if facebook_page_id:
            # Save to Facebook knowledge base too!
            fb_knowledge_file = Path(f"/opt/hermes-webhook/knowledge/{facebook_page_id}_knowledge.md")

            if fb_knowledge_file.exists():
                with open(fb_knowledge_file, 'a') as f:
                    f.write(learning_entry)
                logger.info(f"💾 Saved interaction to Facebook knowledge base (shared)")
            else:
                logger.warning(f"⚠️ Facebook knowledge file not found: {fb_knowledge_file}")

        # Also save to location-specific knowledge file
        knowledge_file = Path(location_config.get('knowledge_file', ''))
        if knowledge_file.exists():
            with open(knowledge_file, 'a') as f:
                f.write(learning_entry)
            logger.info(f"💾 Saved interaction to location knowledge base")
        else:
            logger.debug(f"Location knowledge file not found (using FB only): {knowledge_file}")

    except Exception as e:
        logger.error(f"❌ Failed to save interaction: {e}")

def process_review(location_id, review):
    """Process a single review - generate and post reply"""
    try:
        review_id = review.get('name')

        # Check if already replied
        if gmb_client.has_replied(review):
            logger.info(f"⏭️ Review already has a reply")
            return

        if has_already_replied(review_id):
            logger.info(f"⏭️ Already processed this review")
            return

        # Get location config
        location_config = get_location_config(location_id)
        if not location_config or not location_config.get('enabled', True):
            logger.info(f"⏭️ Location {location_id} disabled")
            return

        # Extract review info
        reviewer_name = review.get('reviewer', {}).get('displayName', 'Customer')
        rating = review.get('starRating', 0)
        comment = review.get('comment', '')

        logger.info(f"📩 New review from {reviewer_name}: {rating}/5 stars")

        # Generate reply
        reply = generate_auto_reply(location_id, review)

        if not reply:
            logger.warning(f"⚠️ Failed to generate reply")
            return

        # Post reply
        if gmb_client.post_reply(review_id, reply):
            logger.info(f"✅ Posted reply for {reviewer_name}")
            save_replied_review(review_id)

            # Save for learning
            save_interaction_for_learning(
                location_id,
                reviewer_name,
                rating,
                comment,
                reply
            )

            # Send notification
            business_name = location_config.get('business_name', 'Business')
            notification_text = f"Rating: {rating}/5 stars\n\n{comment}\n\n🤖 Our Reply:\n{reply}"

            # Use existing notification system (compatible format)
            notification_system.notify_owner(
                location_id,  # Use location_id as page_id
                reviewer_name,
                notification_text,
                reply
            )

        else:
            logger.error(f"❌ Failed to post reply")

    except Exception as e:
        logger.error(f"❌ Error processing review: {e}")

def poll_location(location_id):
    """Poll a single location for new reviews"""
    try:
        location_config = get_location_config(location_id)
        if not location_config:
            logger.error(f"❌ No config for location {location_id}")
            return

        if not location_config.get('enabled', True):
            logger.info(f"⏭️ Location {location_id} disabled, skipping")
            return

        location_id_api = location_config.get('location_id')
        business_name = location_config.get('business_name', 'Business')

        logger.info(f"🔍 Polling {business_name} (Location: {location_id_api})")

        # Get reviews from last polling period
        reviews = gmb_client.get_new_reviews(
            location_id_api,
            since_hours=POLLING_INTERVAL_MINUTES / 60 + 1  # Buffer
        )

        if not reviews:
            logger.info(f"✅ No new reviews for {business_name}")
            return

        logger.info(f"📩 Found {len(reviews)} new reviews for {business_name}")

        # Process each review
        for review in reviews:
            process_review(location_id, review)

    except Exception as e:
        logger.error(f"❌ Error polling location {location_id}: {e}")

def poll_all_locations():
    """Poll all configured locations"""
    try:
        config = load_locations_config()
        locations = config.get('locations', {})

        if not locations:
            logger.warning("⚠️ No locations configured")
            return

        logger.info(f"🚀 Starting poll for {len(locations)} location(s)")

        # Authenticate with Google
        if not gmb_client.authenticate():
            logger.error("❌ Authentication failed")
            return

        # Poll each location
        for location_id, location_config in locations.items():
            try:
                poll_location(location_id)
            except Exception as e:
                logger.error(f"❌ Error with location {location_id}: {e}")

        logger.info(f"✅ Polling cycle complete")

    except Exception as e:
        logger.error(f"❌ Error in polling cycle: {e}")

def main():
    """Main polling loop"""
    logger.info("🚀 Google Reviews Poller starting...")
    logger.info(f"📊 Polling interval: {POLLING_INTERVAL_MINUTES} minutes")

    # Load replied reviews
    load_replied_reviews()

    # Ensure log directory exists
    LOG_DIR.mkdir(parents=True, exist_ok=True)

    # Main loop
    while True:
        try:
            logger.info("=" * 60)
            logger.info(f"🔄 Starting polling cycle at {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
            logger.info("=" * 60)

            poll_all_locations()

            logger.info(f"💤 Next poll in {POLLING_INTERVAL_MINUTES} minutes...")
            logger.info("=" * 60)

            # Wait for next poll
            time.sleep(POLLING_INTERVAL_MINUTES * 60)

        except KeyboardInterrupt:
            logger.info("⛔ Poller stopped by user")
            break
        except Exception as e:
            logger.error(f"❌ Error in main loop: {e}")
            logger.info(f"🔄 Retrying in {POLLING_INTERVAL_MINUTES} minutes...")
            time.sleep(POLLING_INTERVAL_MINUTES * 60)

if __name__ == '__main__':
    main()

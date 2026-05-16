#!/usr/bin/env python3
"""
Multi-tenant notification system for Facebook Agent
Sends notifications to respective page owners based on page ID
"""

import os
import requests
import logging
import json
from datetime import datetime
from pathlib import Path

from runtime_paths import get_notification_config_file

logger = logging.getLogger(__name__)


def _clean_preview(text, limit=200):
    compact = " ".join((text or "").split())
    if len(compact) <= limit:
        return compact
    return compact[: limit - 3] + "..."


def _telegram_reply_target_missing(response):
    if response.status_code != 400:
        return False
    try:
        payload = response.json()
    except ValueError:
        return False
    description = str(payload.get("description") or "").lower()
    return "message to be replied not found" in description

class MultiTenantNotificationSystem:
    def __init__(self):
        self.notifications = self.load_notification_config()

    @property
    def config_file(self):
        return get_notification_config_file()
    
    def load_notification_config(self):
        """Load notification config for all pages"""
        try:
            config_file = self.config_file
            if config_file.exists():
                with open(config_file, 'r', encoding='utf-8') as f:
                    return json.load(f)
            return {"pages": {}}
        except Exception as e:
            logger.error(f"Failed to load notification config: {e}")
            return {"pages": {}}
    
    def save_notification_config(self, config):
        """Save notification config to file"""
        try:
            config_file = self.config_file
            config_file.parent.mkdir(parents=True, exist_ok=True)
            with open(config_file, 'w', encoding='utf-8') as f:
                json.dump(config, f, indent=2)
            logger.info("✅ Notification config saved")
        except Exception as e:
            logger.error(f"Failed to save notification config: {e}")
    
    def get_page_notification_config(self, page_id):
        """Get notification config for specific page"""
        return self.notifications.get("pages", {}).get(page_id, {})
    
    def detect_uncertainty(self, response):
        """
        Detect if the agent is uncertain about the answer
        Returns True if the response shows uncertainty
        """
        uncertainty_phrases = [
            "للأسف معلومة",
            "ليس لدي معلومة",
            "لا أعلم",
            "أعتذر",
            "للأسف",
            "معذرة",
            "ليس لدي",
            "غير متوفر",
            "Unfortunately",
            "I don't have",
            "I'm not sure",
            "I don't know",
            "Sorry",
            "Unfortunately",
            "not available",
            "لا أستطيع",
            "أعتذر عن",
            "مش متوفر",
            "معلش"
        ]
        
        response_lower = response.lower()
        return any(phrase.lower() in response_lower for phrase in uncertainty_phrases)
    
    def send_telegram_notification(self, page_id, customer_name, question, agent_response):
        """Send notification via Telegram to page owner"""
        config = self.get_page_notification_config(page_id)
        
        telegram_token = config.get('telegram_bot_token')
        telegram_chat_id = config.get('telegram_chat_id')
        
        if not telegram_token or not telegram_chat_id:
            logger.warning(f"⚠️ Telegram not configured for page {page_id}")
            return False
        
        try:
            # Get page name
            page_name = config.get('page_name', f'Page {page_id}')
            
            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            
            message = f"""
💬 *New Facebook Comment*

📄 *Page:* {page_name} (ID: {page_id})
👤 *Customer:* {customer_name}
❓ *Question:* {question}
🤖 *Agent Replied:* {agent_response}

⏰ *Time:* {timestamp}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔔 All comments are monitored for quality
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
"""
            
            url = f"https://api.telegram.org/bot{telegram_token}/sendMessage"
            
            response = requests.post(
                url,
                json={
                    "chat_id": telegram_chat_id,
                    "text": message,
                    "parse_mode": "Markdown"
                },
                timeout=10
            )
            
            if response.status_code == 200:
                logger.info(f"✅ Telegram notification sent to page {page_id} owner for: {question[:30]}...")
                return True
            else:
                logger.error(f"❌ Telegram API error for page {page_id}: {response.text}")
                return False
                
        except Exception as e:
            logger.error(f"❌ Failed to send Telegram notification for page {page_id}: {e}")
            return False

    def send_telegram_text(self, page_id, text, reply_to_message_id=None):
        """Send a plain Telegram text message to the configured page owner chat."""
        config = self.get_page_notification_config(page_id)

        telegram_token = config.get('telegram_bot_token')
        telegram_chat_id = config.get('telegram_chat_id')

        if not telegram_token or not telegram_chat_id:
            logger.warning(f"⚠️ Telegram not configured for page {page_id}")
            return False

        try:
            payload = {
                "chat_id": telegram_chat_id,
                "text": text,
            }
            if reply_to_message_id:
                payload["reply_to_message_id"] = reply_to_message_id

            url = f"https://api.telegram.org/bot{telegram_token}/sendMessage"
            response = requests.post(url, json=payload, timeout=10)

            if reply_to_message_id and _telegram_reply_target_missing(response):
                payload.pop("reply_to_message_id", None)
                response = requests.post(url, json=payload, timeout=10)

            if response.status_code == 200:
                logger.info(f"✅ Telegram text sent to page {page_id} owner")
                return True

            logger.error(f"❌ Telegram text API error for page {page_id}: {response.text}")
            return False
        except Exception as e:
            logger.error(f"❌ Failed to send Telegram text for page {page_id}: {e}")
            return False

    def send_telegram_learning_notification(
        self,
        page_id,
        source_question,
        learned_reply,
        learned_from_human_id="",
        parent_comment_id="",
    ):
        """Send Telegram notification when the system learns a new human reply."""
        config = self.get_page_notification_config(page_id)

        telegram_token = config.get('telegram_bot_token')
        telegram_chat_id = config.get('telegram_chat_id')

        if not telegram_token or not telegram_chat_id:
            logger.warning(f"⚠️ Telegram not configured for learned-reply notification on page {page_id}")
            return False

        try:
            page_name = config.get('page_name', f'Page {page_id}')
            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            learned_from = learned_from_human_id or "unknown"
            parent_ref = parent_comment_id or "unknown"

            message = (
                "🧠 New Learned Reply\n\n"
                f"📄 Page: {page_name} (ID: {page_id})\n"
                f"❓ Original Comment: {_clean_preview(source_question)}\n"
                f"💬 Learned Reply: {_clean_preview(learned_reply)}\n"
                f"👤 Learned From: {learned_from}\n"
                f"🔗 Parent Comment ID: {parent_ref}\n"
                f"⏰ Time: {timestamp}\n"
            )

            url = f"https://api.telegram.org/bot{telegram_token}/sendMessage"
            response = requests.post(
                url,
                json={
                    "chat_id": telegram_chat_id,
                    "text": message,
                },
                timeout=10
            )

            if response.status_code == 200:
                logger.info(
                    "✅ Telegram learned-reply notification sent to page %s owner for: %s",
                    page_id,
                    _clean_preview(source_question, limit=30),
                )
                return True

            logger.error(f"❌ Telegram learned-reply API error for page {page_id}: {response.text}")
            return False
        except Exception as e:
            logger.error(f"❌ Failed learned-reply Telegram notification for page {page_id}: {e}")
            return False
    
    def send_whatsapp_notification(self, page_id, customer_name, question, agent_response):
        """Send notification via WhatsApp to page owner"""
        config = self.get_page_notification_config(page_id)
        
        whatsapp_phone = config.get('whatsapp_phone')
        whatsapp_api_key = config.get('whatsapp_api_key')
        
        if not whatsapp_phone or not whatsapp_api_key:
            logger.warning(f"⚠️ WhatsApp not configured for page {page_id}")
            return False
        
        try:
            # Get page name
            page_name = config.get('page_name', f'Page {page_id}')
            
            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            
            message = f"""
💬 New Facebook Comment

📄 Page: {page_name} (ID: {page_id})
👤 Customer: {customer_name}
❓ Question: {question}
🤖 Agent Replied: {agent_response}

⏰ Time: {timestamp}

All comments monitored for quality
"""
            
            # CallMeBot API
            url = f"https://api.callmebot.com/whatsapp.php"
            
            response = requests.post(
                url,
                data={
                    "phone": whatsapp_phone,
                    "text": message,
                    "apikey": whatsapp_api_key
                },
                timeout=10
            )
            
            if response.status_code == 200:
                logger.info(f"✅ WhatsApp notification sent to page {page_id} owner for: {question[:30]}...")
                return True
            else:
                logger.error(f"❌ WhatsApp API error for page {page_id}: {response.text}")
                return False
                
        except Exception as e:
            logger.error(f"❌ Failed to send WhatsApp notification for page {page_id}: {e}")
            return False
    
    def notify_owner(self, page_id, customer_name, question, agent_response):
        """
        Send notification to page owner based on page ID
        NOW SENDS NOTIFICATION FOR EVERY COMMENT AND ANSWER
        Returns True if at least one notification was sent
        """
        logger.info(f"🔔 New comment on page {page_id}: {question[:30]}...")

        # Check if this page has notification config
        config = self.get_page_notification_config(page_id)
        if not config:
            logger.warning(f"⚠️ No notification config found for page {page_id}")
            return False

        # REMOVED uncertainty check - now sending for ALL comments
        logger.info(f"📤 Sending notification to page {page_id} owner...")

        success = False

        # Try Telegram
        if config.get('telegram_bot_token') and config.get('telegram_chat_id'):
            if self.send_telegram_notification(page_id, customer_name, question, agent_response):
                success = True

        # Try WhatsApp
        if config.get('whatsapp_phone') and config.get('whatsapp_api_key'):
            if self.send_whatsapp_notification(page_id, customer_name, question, agent_response):
                success = True

        return success

    def notify_learning(self, page_id, source_question, learned_reply, learned_from_human_id="", parent_comment_id=""):
        """Send a Telegram notification when a new learned reply is created."""
        logger.info(f"🧠 New learned reply on page {page_id}: {_clean_preview(source_question, limit=30)}")

        config = self.get_page_notification_config(page_id)
        if not config:
            logger.warning(f"⚠️ No notification config found for learned reply on page {page_id}")
            return False

        if config.get('telegram_bot_token') and config.get('telegram_chat_id'):
            return self.send_telegram_learning_notification(
                page_id,
                source_question,
                learned_reply,
                learned_from_human_id=learned_from_human_id,
                parent_comment_id=parent_comment_id,
            )

        logger.warning(f"⚠️ Telegram not configured for learned reply notification on page {page_id}")
        return False

# Singleton instance
notification_system = MultiTenantNotificationSystem()

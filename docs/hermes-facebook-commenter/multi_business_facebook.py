"""
Multi-Business Facebook Plugin for Hermes
Supports multiple Facebook pages with separate memory and learning
"""

import os
import re
import json
import requests
from pathlib import Path
from typing import Dict, List, Optional
from collections import Counter
from datetime import datetime

from rate_limiter import rate_limited, rate_limiter

GRAPH_URL = "https://graph.facebook.com/v19.0"

class BusinessConfig:
    """Configuration for a single business"""

    def __init__(self, business_id: str, config: dict):
        self.business_id = business_id
        self.name = config.get('name', 'Unknown Business')
        self.page_id = config.get('page_id')
        self.page_access_token = config.get('page_access_token')
        self.page_name = config.get('page_name', '')

        # Business-specific settings
        self.auto_reply = config.get('auto_reply', True)
        self.auto_post = config.get('auto_post', False)
        self.post_schedule = config.get('post_schedule', {})
        self.reply_language = config.get('reply_language', 'auto')  # auto, en, ar

        # Business knowledge (learned info)
        self.services = config.get('services', [])
        self.prices = config.get('prices', {})
        self.location = config.get('location', {})
        self.hours = config.get('hours', {})
        self.faqs = config.get('faqs', [])
        self.tone = config.get('tone', 'professional')

class MultiBusinessFacebookManager:
    """Manages multiple Facebook pages for different businesses"""

    def __init__(self):
        self.config_dir = Path.home() / ".hermes" / "businesses"
        self.config_dir.mkdir(parents=True, exist_ok=True)
        self.memory_dir = self.config_dir / "memory"
        self.memory_dir.mkdir(parents=True, exist_ok=True)
        self.businesses: Dict[str, BusinessConfig] = {}
        self.analytics: Dict[str, dict] = {}
        self.load_all_businesses()

    def load_all_businesses(self):
        """Load all business configurations"""
        for config_file in self.config_dir.glob("*.json"):
            business_id = config_file.stem
            try:
                with open(config_file) as f:
                    config = json.load(f)
                self.businesses[business_id] = BusinessConfig(business_id, config)
                print(f"✓ Loaded business: {config.get('name', business_id)}")
            except Exception as e:
                print(f"✗ Failed to load {business_id}: {e}")

    def get_business(self, business_id: str) -> Optional[BusinessConfig]:
        """Get a business by ID"""
        return self.businesses.get(business_id)

    def get_business_by_page_id(self, page_id: str) -> Optional[BusinessConfig]:
        """Find business by Facebook Page ID"""
        for business in self.businesses.values():
            if business.page_id == page_id:
                return business
        return None

    def add_business(self, business_id: str, config: dict) -> bool:
        """Add a new business"""
        config_file = self.config_dir / f"{business_id}.json"
        try:
            with open(config_file, 'w') as f:
                json.dump(config, f, indent=2)

            self.businesses[business_id] = BusinessConfig(business_id, config)
            print(f"✓ Added business: {config.get('name', business_id)}")
            return True
        except Exception as e:
            print(f"✗ Failed to add business: {e}")
            return False

    def publish_post(self, business_id: str, message: str) -> dict:
        """Publish a post for a specific business"""
        business = self.get_business(business_id)
        if not business:
            return {"error": "Business not found"}

        if not rate_limiter.wait_and_acquire(f"business:{business_id}"):
            return {"error": "Rate limit exceeded. Try again later."}

        res = requests.post(
            f"{GRAPH_URL}/{business.page_id}/feed",
            data={
                "message": message,
                "access_token": business.page_access_token
            }
        )

        # Save to business memory
        self.save_business_memory(business_id, "post", {
            "message": message,
            "timestamp": datetime.now().isoformat(),
            "response": res.json()
        })

        return res.json()

    def get_comments(self, business_id: str, post_id: str) -> list:
        """Get comments for a business's post"""
        business = self.get_business(business_id)
        if not business:
            return []

        if not rate_limiter.wait_and_acquire(f"business:{business_id}"):
            return {"error": "Rate limit exceeded. Try again later."}

        res = requests.get(
            f"{GRAPH_URL}/{post_id}/comments",
            params={
                "fields": "id,message,from,created_time",
                "access_token": business.page_access_token
            }
        )

        return res.json().get("data", [])

    def reply_to_comment(self, business_id: str, comment_id: str, message: str) -> dict:
        """Reply to a comment for a specific business"""
        business = self.get_business(business_id)
        if not business:
            return {"error": "Business not found"}

        if not rate_limiter.wait_and_acquire(f"business:{business_id}"):
            return {"error": "Rate limit exceeded. Try again later."}

        res = requests.post(
            f"{GRAPH_URL}/{comment_id}/replies",
            data={
                "message": message,
                "access_token": business.page_access_token
            }
        )

        # Save to business memory
        self.save_business_memory(business_id, "reply", {
            "comment_id": comment_id,
            "message": message,
            "timestamp": datetime.now().isoformat(),
            "response": res.json()
        })

        return res.json()

    def get_all_posts(self, business_id: str) -> list:
        """Get all posts for a business"""
        business = self.get_business(business_id)
        if not business:
            return []

        if not rate_limiter.wait_and_acquire(f"business:{business_id}"):
            return {"error": "Rate limit exceeded. Try again later."}

        res = requests.get(
            f"{GRAPH_URL}/{business.page_id}/posts",
            params={
                "fields": "id,message,created_time,comments{message,from}",
                "access_token": business.page_access_token
            }
        )

        return res.json().get("data", [])

    def save_business_memory(self, business_id: str, memory_type: str, data: dict):
        """Save learned information to business memory"""
        memory_file = self.memory_dir / f"{business_id}_{memory_type}s.jsonl"

        with open(memory_file, 'a') as f:
            f.write(json.dumps(data) + "\n")

    def get_business_memory(self, business_id: str, memory_type: str = None) -> list:
        """Retrieve business memory"""
        if memory_type:
            memory_file = self.memory_dir / f"{business_id}_{memory_type}s.jsonl"
            if not memory_file.exists():
                return []

            with open(memory_file) as f:
                return [json.loads(line) for line in f]
        else:
            # Get all memory types
            memories = {}
            for memory_file in self.memory_dir.glob(f"{business_id}_*.jsonl"):
                memory_type = memory_file.stem.split('_', 1)[1]
                with open(memory_file) as f:
                    memories[memory_type] = [json.loads(line) for line in f]
            return memories

    def _extract_keywords(self, text: str) -> Dict[str, int]:
        """Extract meaningful keywords from text with frequency"""
        STOP_WORDS = {
            'the', 'a', 'an', 'is', 'are', 'was', 'were', 'be', 'been',
            'have', 'has', 'had', 'do', 'does', 'did', 'will', 'would',
            'could', 'should', 'may', 'might', 'can', 'shall', 'to', 'of',
            'in', 'for', 'on', 'with', 'at', 'by', 'from', 'as', 'into',
            'through', 'during', 'before', 'after', 'above', 'below',
            'between', 'and', 'but', 'or', 'nor', 'not', 'so', 'yet',
            'i', 'you', 'he', 'she', 'it', 'we', 'they', 'me', 'him',
            'her', 'us', 'them', 'my', 'your', 'his', 'its', 'our',
            'their', 'this', 'that', 'these', 'those', 'am', 'if', 'no',
            'up', 'out', 'off', 'over', 'just', 'very', 'too', 'then',
            'than', 'also', 'about', 'how', 'what', 'why', 'when',
            'where', 'which', 'who', 'whom', 'there', 'here',
            'من', 'في', 'الى', 'على', 'عن', 'مع', 'كان', 'هذا', 'هذه',
            'ذلك', 'تلك', 'هو', 'هي', 'هم', 'هن', 'انا', 'نحن', 'انت',
            'انتم', 'انتن', 'كانت', 'كانوا', 'كانو', 'ليس', 'لا', 'لم',
            'لن', 'ما', 'ماذا', 'كيف', 'اين', 'متى', 'هل', 'اذا', 'ثم',
            'او', 'ام', 'قد', 'لقد', 'ان', 'إن', 'أن', 'بان', 'بأن',
        }
        words = re.findall(r'\w{3,}', text.lower())
        return dict(Counter(w for w in words if w not in STOP_WORDS))

    def learn_from_interaction(self, business_id: str, interaction: dict):
        """Learn from customer interactions with full analytics"""
        business = self.get_business(business_id)
        if not business:
            return

        message = interaction.get('message', '')
        message_lower = message.lower()
        timestamp = interaction.get('timestamp', datetime.now().isoformat())

        # Extract keywords
        keywords = self._extract_keywords(message)

        # Detect question type
        question_type = 'other'
        if any(w in message_lower for w in ['price', 'cost', 'how much', 'سعر', 'كم', 'ثمن', 'قيمة']):
            question_type = 'price'
        elif any(w in message_lower for w in ['hours', 'open', 'time', 'ساعة', 'مواعيد', 'مفتوح']):
            question_type = 'hours'
        elif any(w in message_lower for w in ['location', 'where', 'address', 'موقع', 'عنوان', 'مكان']):
            question_type = 'location'
        elif any(w in message_lower for w in ['service', 'offer', 'what do', 'what can', 'خدمات', 'ماذا', 'تقدمون']):
            question_type = 'service'
        elif any(w in message_lower for w in ['bad', 'terrible', 'awful', 'slow', 'break', 'broken', 'سيء', 'سيئ', 'رديء', 'بطيء', 'مكسور', 'شكوى', 'مشكلة']):
            question_type = 'complaint'
        elif any(w in message_lower for w in ['good', 'great', 'excellent', 'amazing', 'nice', 'beautiful', 'جميل', 'رائع', 'ممتاز', 'ممتازه']):
            question_type = 'praise'

        # Detect new service mentions (words not in current services)
        service_words = set()
        for s in business.services:
            service_words.update(re.findall(r'\w+', s.lower()))
        new_services = [w for w in keywords if w not in service_words]

        # Detect price mentions (digits near currency)
        price_pattern = re.compile(r'(\d+[\d,.\s]*\s*(?:ريال|جنيه|دولار|EGP|USD|SAR|LE|£|\$|€))|((?:ريال|جنيه|دولار|EGP|USD|SAR|LE|£|\$|€)\s*[\d,.\s]+\d)', re.IGNORECASE)
        price_mentions = price_pattern.findall(message)

        # Update analytics
        if business_id not in self.analytics:
            self.analytics[business_id] = {
                'total_interactions': 0,
                'question_type_counts': Counter(),
                'keywords': Counter(),
                'suggested_services': Counter(),
                'price_mentions': [],
                'recent_interactions': [],
            }

        a = self.analytics[business_id]
        a['total_interactions'] += 1
        a['question_type_counts'][question_type] += 1
        a['keywords'].update(keywords)
        for ns in new_services:
            if ns not in service_words:
                a['suggested_services'][ns] += 1
        if price_mentions:
            a['price_mentions'].append({
                'raw': message[:100],
                'matches': price_mentions,
                'timestamp': timestamp,
            })
        a['recent_interactions'].append({
            'message': message[:200],
            'type': question_type,
            'timestamp': timestamp,
        })
        if len(a['recent_interactions']) > 100:
            a['recent_interactions'] = a['recent_interactions'][-100:]

        # Save learning memory
        self.save_business_memory(business_id, "learning", {
            "interaction": interaction,
            "question_type": question_type,
            "keywords": list(keywords.keys())[:10],
            "timestamp": timestamp
        })

        # Save analytics
        self.save_business_memory(business_id, "analytics", {
            "question_type": question_type,
            "keywords": keywords,
            "new_services": new_services[:10],
            "price_mentions": bool(price_mentions),
            "timestamp": timestamp
        })

    def get_business_analytics(self, business_id: str) -> dict:
        """Retrieve aggregated analytics for a business"""
        if business_id in self.analytics:
            a = self.analytics[business_id]
            return {
                'total_interactions': a['total_interactions'],
                'question_type_breakdown': dict(a['question_type_counts']),
                'top_keywords': dict(a['keywords'].most_common(20)),
                'suggested_new_services': {
                    word: count for word, count in a['suggested_services'].most_common(10)
                    if count >= 2
                },
                'recent_price_mentions': a['price_mentions'][-10:],
                'recent_interactions': a['recent_interactions'][-10:],
            }

        # Fall back to filesystem
        memories = self.get_business_memory(business_id, 'analytics')
        if not memories:
            return {
                'total_interactions': 0,
                'question_type_breakdown': {},
                'top_keywords': {},
                'suggested_new_services': {},
                'recent_price_mentions': [],
                'recent_interactions': [],
            }

        counts = Counter()
        keywords = Counter()
        price_mentions = []
        for m in memories:
            counts[m.get('question_type', 'other')] += 1
            if 'keywords' in m:
                keywords.update(m['keywords'])
            if m.get('price_mentions'):
                price_mentions.append(m)

        return {
            'total_interactions': len(memories),
            'question_type_breakdown': dict(counts),
            'top_keywords': dict(keywords.most_common(20)),
            'recent_price_mentions': price_mentions[-10:],
        }

    def generate_reply(self, business_id: str, comment: str, customer_name: str = None) -> str:
        """Generate contextual reply based on business knowledge"""
        business = self.get_business(business_id)
        if not business:
            return "Thank you for your message!"

        comment_lower = comment.lower()
        language = self.detect_language(comment)

        # Check for common queries
        if any(word in comment_lower for word in ['price', 'cost', 'how much', 'سعر', 'كم']):
            return self.generate_price_reply(business, language)

        if any(word in comment_lower for word in ['hours', 'open', 'time', 'ساعة', 'مواعيد']):
            return self.generate_hours_reply(business, language)

        if any(word in comment_lower for word in ['location', 'where', 'address', 'موقع', 'عنوان']):
            return self.generate_location_reply(business, language)

        if any(word in comment_lower for word in ['service', 'offer', 'ماذا', 'خدمات']):
            return self.generate_services_reply(business, language)

        # Default friendly reply
        if language == 'ar':
            greeting = f"أهلاً وسهلاً {customer_name or ''}" if customer_name else "أهلاً وسهلاً"
            return f"{greeting}! شكراً لتواصلك مع {business.page_name}. كيف يمكننا مساعدتك اليوم؟"
        else:
            greeting = f"Hi {customer_name}" if customer_name else "Hi"
            return f"{greeting}! Thank you for contacting {business.page_name}. How can we help you today?"

    def detect_language(self, text: str) -> str:
        """Detect if text is Arabic or English"""
        arabic_chars = sum(1 for c in text if '\u0600' <= c <= '\u06FF')
        return 'ar' if arabic_chars > len(text) * 0.3 else 'en'

    def generate_price_reply(self, business: BusinessConfig, language: str) -> str:
        """Generate reply about prices"""
        if language == 'ar':
            reply = f"📍 أسعارنا في {business.page_name}:\n\n"
            for item, price in business.prices.items():
                reply += f"• {item}: {price}\n"
            reply += "\nللاستفسار عن خدمات أخرى، تواصل معنا! 📞"
        else:
            reply = f"📍 Our prices at {business.page_name}:\n\n"
            for item, price in business.prices.items():
                reply += f"• {item}: {price}\n"
            reply += "\nFor other services, contact us! 📞"

        return reply

    def generate_hours_reply(self, business: BusinessConfig, language: str) -> str:
        """Generate reply about hours"""
        if language == 'ar':
            return f"⏰ ساعات العمل في {business.page_name}: {business.hours.get('general', 'يومياً ٩ ص - ٩ م')}"
        else:
            return f"⏰ Business hours at {business.page_name}: {business.hours.get('general', 'Daily 9am - 9pm')}"

    def generate_location_reply(self, business: BusinessConfig, language: str) -> str:
        """Generate reply about location"""
        if language == 'ar':
            return f"📍 موقعنا: {business.location.get('address', 'تواصل معنا للعنوان الكامل')}"
        else:
            return f"📍 Our location: {business.location.get('address', 'Contact us for full address')}"

    def generate_services_reply(self, business: BusinessConfig, language: str) -> str:
        """Generate reply about services"""
        if language == 'ar':
            reply = f"✨ خدماتنا في {business.page_name}:\n\n"
            for service in business.services:
                reply += f"• {service}\n"
        else:
            reply = f"✨ Our services at {business.page_name}:\n\n"
            for service in business.services:
                reply += f"• {service}\n"

        return reply

    def list_businesses(self) -> List[dict]:
        """List all configured businesses"""
        return [
            {
                "id": b.business_id,
                "name": b.name,
                "page_name": b.page_name,
                "page_id": b.page_id,
                "auto_reply": b.auto_reply,
                "auto_post": b.auto_post
            }
            for b in self.businesses.values()
        ]

# Global instance
manager = MultiBusinessFacebookManager()

# Convenience functions for each business
def publish_post(business_id: str, message: str) -> dict:
    """Publish a post for a specific business"""
    return manager.publish_post(business_id, message)

def reply_to_comment(business_id: str, comment_id: str, message: str) -> dict:
    """Reply to a comment for a specific business"""
    return manager.reply_to_comment(business_id, comment_id, message)

def get_comments(business_id: str, post_id: str) -> list:
    """Get comments for a business's post"""
    return manager.get_comments(business_id, post_id)

def get_all_posts(business_id: str) -> list:
    """Get all posts for a business"""
    return manager.get_all_posts(business_id)

def add_business(business_id: str, config: dict) -> bool:
    """Add a new business"""
    return manager.add_business(business_id, config)

def generate_reply(business_id: str, comment: str, customer_name: str = None) -> str:
    """Generate contextual reply for a business"""
    return manager.generate_reply(business_id, comment, customer_name)

def list_businesses() -> list:
    """List all businesses"""
    return manager.list_businesses()

def get_business_info(business_id: str) -> dict:
    """Get business information"""
    business = manager.get_business(business_id)
    if not business:
        return {}

    return {
        "name": business.name,
        "page_name": business.page_name,
        "services": business.services,
        "prices": business.prices,
        "location": business.location,
        "hours": business.hours,
        "tone": business.tone
    }

def get_business_analytics(business_id: str) -> dict:
    """Get aggregated analytics for a business"""
    return manager.get_business_analytics(business_id)

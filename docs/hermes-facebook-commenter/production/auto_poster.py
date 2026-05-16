#!/usr/bin/env python3
"""
Facebook Auto-Poster - Complete System
Features:
- Instant posting to page
- Scheduling system
- AI content generation
- Multi-group support
- Content management
"""

import os
import json
import requests
import subprocess
from pathlib import Path
from datetime import datetime, timedelta
import logging

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('/var/log/hermes/auto_poster.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

# Configuration
GRAPH_URL = "https://graph.facebook.com/v19.0"
PAGE_TOKEN = os.environ.get("FB_PAGE_ACCESS_TOKEN")
PAGE_ID = os.environ.get("FB_PAGE_ID", "895247390337022")
KNOWLEDGE_FILE = Path("/opt/hermes-webhook/knowledge/895247390337022_knowledge.md")
CONTENT_DB = Path("/opt/hermes-webhook/content_queue.json")
SCHEDULE_DB = Path("/opt/hermes-webhook/scheduled_posts.json")

class FacebookAutoPoster:
    def __init__(self):
        self.content_queue = self.load_content_queue()
        self.scheduled_posts = self.load_scheduled_posts()
    
    def load_content_queue(self):
        """Load content queue from file"""
        try:
            if CONTENT_DB.exists():
                with open(CONTENT_DB, 'r') as f:
                    return json.load(f)
            return {"queue": [], "posted": []}
        except Exception as e:
            logger.error(f"Failed to load content queue: {e}")
            return {"queue": [], "posted": []}
    
    def save_content_queue(self):
        """Save content queue to file"""
        try:
            with open(CONTENT_DB, 'w') as f:
                json.dump(self.content_queue, f, indent=2)
        except Exception as e:
            logger.error(f"Failed to save content queue: {e}")
    
    def load_scheduled_posts(self):
        """Load scheduled posts from file"""
        try:
            if SCHEDULE_DB.exists():
                with open(SCHEDULE_DB, 'r') as f:
                    return json.load(f)
            return {"scheduled": [], "completed": []}
        except Exception as e:
            logger.error(f"Failed to load scheduled posts: {e}")
            return {"scheduled": [], "completed": []}
    
    def save_scheduled_posts(self):
        """Save scheduled posts to file"""
        try:
            with open(SCHEDULE_DB, 'w') as f:
                json.dump(self.scheduled_posts, f, indent=2)
        except Exception as e:
            logger.error(f"Failed to save scheduled posts: {e}")
    
    def post_to_page(self, message, page_id=None):
        """Post content to Facebook page"""
        try:
            page_id = page_id or PAGE_ID
            url = f"{GRAPH_URL}/{page_id}/feed"
            
            response = requests.post(
                url,
                data={
                    "message": message,
                    "access_token": PAGE_TOKEN
                },
                timeout=10
            )
            
            result = response.json()
            
            if 'id' in result:
                logger.info(f"✅ Posted successfully: {result['id']}")
                return {
                    "success": True,
                    "post_id": result['id'],
                    "page_id": page_id
                }
            else:
                logger.error(f"❌ Failed to post: {result}")
                return {
                    "success": False,
                    "error": result.get('error', 'Unknown error')
                }
        except Exception as e:
            logger.error(f"❌ Exception while posting: {e}")
            return {
                "success": False,
                "error": str(e)
            }
    
    def generate_ai_content(self, topic=None, tone="professional"):
        """Generate content using AI"""
        try:
            knowledge = ""
            if KNOWLEDGE_FILE.exists():
                with open(KNOWLEDGE_FILE, 'r') as f:
                    knowledge = f.read()
            
            if topic:
                prompt = f"""Write a Facebook post about: {topic}

Tone: {tone}
Length: 1-3 paragraphs
Include emojis
Engaging and professional

Business Knowledge:
{knowledge}

Write the post:"""
            else:
                prompt = f"""Write an engaging Facebook post for a software company.

Tone: {tone}
Length: 1-3 paragraphs
Include emojis
Make it engaging and professional

Business Knowledge:
{knowledge}

Write the post:"""
            
            result = subprocess.run(
                ['hermes', 'chat', '--query', prompt],
                input="Generate a Facebook post",
                capture_output=True,
                text=True,
                timeout=30,
                cwd='/tmp'
            )
            
            if result.returncode == 0:
                # Extract response
                lines = result.stdout.split('\n')
                content = []
                for line in lines:
                    if line.strip() and not any(skip in line for skip in ['Resume', 'Session:', 'Duration:', '╰', '│', '═']):
                        if not any(x in line for x in ['Resume this', 'Session:', 'Duration:']):
                            content.append(line.strip())
                
                if content:
                    post_text = '\n'.join(content).strip()
                    return post_text
            
            return None
        except Exception as e:
            logger.error(f"Failed to generate AI content: {e}")
            return None
    
    def add_to_queue(self, message, post_type="page", scheduled_time=None):
        """Add content to posting queue"""
        post = {
            "id": int(datetime.now().timestamp()),
            "message": message,
            "type": post_type,
            "status": "pending",
            "created_at": datetime.now().isoformat(),
            "scheduled_time": scheduled_time
        }
        
        if scheduled_time:
            self.scheduled_posts["scheduled"].append(post)
            self.save_scheduled_posts()
        else:
            self.content_queue["queue"].append(post)
            self.save_content_queue()
        
        logger.info(f"✅ Added to queue: {post['id']}")
        return post
    
    def process_queue(self):
        """Process all pending posts in queue"""
        processed = 0
        
        # Process instant posts
        for post in self.content_queue["queue"][:]:
            if post["status"] == "pending":
                result = self.post_to_page(post["message"])
                if result["success"]:
                    post["status"] = "posted"
                    post["posted_at"] = datetime.now().isoformat()
                    post["post_id"] = result["post_id"]
                    self.content_queue["queue"].remove(post)
                    self.content_queue["posted"].append(post)
                    processed += 1
        
        # Process scheduled posts
        now = datetime.now()
        for post in self.scheduled_posts["scheduled"][:]:
            if post["status"] == "pending":
                scheduled_time = datetime.fromisoformat(post["scheduled_time"])
                if now >= scheduled_time:
                    result = self.post_to_page(post["message"])
                    if result["success"]:
                        post["status"] = "posted"
                        post["posted_at"] = datetime.now().isoformat()
                        post["post_id"] = result["post_id"]
                        self.scheduled_posts["scheduled"].remove(post)
                        self.scheduled_posts["completed"].append(post)
                        processed += 1
        
        if processed > 0:
            self.save_content_queue()
            self.save_scheduled_posts()
        
        return processed

# Singleton instance
auto_poster = FacebookAutoPoster()

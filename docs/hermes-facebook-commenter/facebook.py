import os
import logging
import requests

from rate_limiter import rate_limited, rate_limiter

logger = logging.getLogger(__name__)

GRAPH_URL = "https://graph.facebook.com/v19.0"
PAGE_TOKEN = os.environ["FB_PAGE_ACCESS_TOKEN"]
PAGE_ID = os.environ["FB_PAGE_ID"]


@rate_limited(key=f"page:{PAGE_ID}")
def publish_post(message: str) -> dict:
    """Publish a post to the Facebook page"""
    try:
        res = requests.post(f"{GRAPH_URL}/{PAGE_ID}/feed", data={
            "message": message,
            "access_token": PAGE_TOKEN
        })
        if res.status_code != 200:
            logger.error("publish_post failed: HTTP %d: %s", res.status_code, res.text)
            return {"error": f"HTTP {res.status_code}: {res.text}"}
        data = res.json()
        logger.info("publish_post succeeded: %s", data)
        return data
    except requests.exceptions.RequestException as e:
        logger.error("publish_post network error: %s", e)
        return {"error": str(e)}


@rate_limited(key=f"page:{PAGE_ID}")
def get_comments(post_id: str) -> list:
    """Fetch all comments on a post"""
    try:
        res = requests.get(f"{GRAPH_URL}/{post_id}/comments", params={
            "fields": "id,message,from,created_time",
            "access_token": PAGE_TOKEN
        })
        if res.status_code != 200:
            logger.error("get_comments failed: HTTP %d: %s", res.status_code, res.text)
            return {"error": f"HTTP {res.status_code}: {res.text}"}
        data = res.json().get("data", [])
        logger.info("get_comments succeeded: %d comments", len(data))
        return data
    except requests.exceptions.RequestException as e:
        logger.error("get_comments network error: %s", e)
        return {"error": str(e)}


@rate_limited(key=f"page:{PAGE_ID}")
def reply_to_comment(comment_id: str, message: str) -> dict:
    """Reply to a specific comment"""
    try:
        res = requests.post(f"{GRAPH_URL}/{comment_id}/replies", data={
            "message": message,
            "access_token": PAGE_TOKEN
        })
        if res.status_code != 200:
            logger.error("reply_to_comment failed: HTTP %d: %s", res.status_code, res.text)
            return {"error": f"HTTP {res.status_code}: {res.text}"}
        data = res.json()
        logger.info("reply_to_comment succeeded: %s", data)
        return data
    except requests.exceptions.RequestException as e:
        logger.error("reply_to_comment network error: %s", e)
        return {"error": str(e)}


@rate_limited(key=f"page:{PAGE_ID}")
def get_all_posts() -> list:
    """Fetch recent posts from the page"""
    try:
        res = requests.get(f"{GRAPH_URL}/{PAGE_ID}/posts", params={
            "fields": "id,message,created_time,comments{message,from}",
            "access_token": PAGE_TOKEN
        })
        if res.status_code != 200:
            logger.error("get_all_posts failed: HTTP %d: %s", res.status_code, res.text)
            return {"error": f"HTTP {res.status_code}: {res.text}"}
        data = res.json().get("data", [])
        logger.info("get_all_posts succeeded: %d posts", len(data))
        return data
    except requests.exceptions.RequestException as e:
        logger.error("get_all_posts network error: %s", e)
        return {"error": str(e)}

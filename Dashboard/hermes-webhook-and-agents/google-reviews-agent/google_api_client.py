#!/usr/bin/env python3
"""
Google My Business API Client
Fetches reviews and posts replies
"""

import logging
import requests
from datetime import datetime, timedelta
from pathlib import Path

try:
    from googleapiclient.discovery import build
    from googleapiclient.errors import HttpError
    import google.auth
except ImportError:
    print("❌ Google libraries not installed")
    exit(1)

from google_oauth import oauth_manager

logger = logging.getLogger(__name__)

class GoogleMyBusinessClient:
    """Client for Google My Business API"""

    API_NAME = 'mybusiness'
    API_VERSION = 'v4'
    DISCOVERY_SERVICE_URL = 'https://mybusiness.googleapis.com/$discovery/rest'

    def __init__(self):
        self.service = None
        self.locations_cache = {}

    def authenticate(self):
        """Authenticate with Google My Business API"""
        try:
            credentials = oauth_manager.get_credentials()

            if not credentials:
                logger.error("❌ Not authenticated")
                return False

            # Build API service
            self.service = build(
                self.API_NAME,
                self.API_VERSION,
                credentials=credentials,
                discoveryServiceUrl=self.DISCOVERY_SERVICE_URL
            )

            logger.info("✅ Authenticated with Google My Business API")
            return True

        except Exception as e:
            logger.error(f"❌ Authentication failed: {e}")
            return False

    def get_accounts(self):
        """Get all accounts"""
        try:
            if not self.service:
                if not self.authenticate():
                    return None

            accounts = self.service.accounts().list().execute()

            logger.info(f"✅ Found {len(accounts.get('accounts', []))} accounts")
            return accounts.get('accounts', [])

        except HttpError as e:
            logger.error(f"❌ API error: {e}")
            return None
        except Exception as e:
            logger.error(f"❌ Failed to get accounts: {e}")
            return None

    def get_locations(self, account_id):
        """Get all locations for an account"""
        try:
            if not self.service:
                if not self.authenticate():
                    return None

            locations = self.service.accounts().locations().list(
                parent=account_id,
                pageSize=100
            ).execute()

            locations_list = locations.get('locations', [])

            # Cache locations
            for location in locations_list:
                self.locations_cache[location['name']] = location

            logger.info(f"✅ Found {len(locations_list)} locations for {account_id}")
            return locations_list

        except HttpError as e:
            logger.error(f"❌ API error: {e}")
            return None
        except Exception as e:
            logger.error(f"❌ Failed to get locations: {e}")
            return None

    def get_location_name(self, location_id):
        """Convert location ID to full resource name"""
        return f'locations/{location_id}'

    def get_reviews(self, location_id, page_size=100):
        """
        Get all reviews for a location

        Args:
            location_id: Location ID (can be short form or full name)
            page_size: Number of reviews to fetch

        Returns:
            List of reviews
        """
        try:
            if not self.service:
                if not self.authenticate():
                    return None

            # Handle both short ID and full name
            if not location_id.startswith('locations/'):
                location_name = self.get_location_name(location_id)
            else:
                location_name = location_id

            # Fetch reviews
            reviews_response = self.service.accounts().locations().reviews().list(
                parent=location_name,
                pageSize=page_size,
                orderBy='timestamp_desc'
            ).execute()

            reviews = reviews_response.get('reviews', [])

            logger.info(f"✅ Fetched {len(reviews)} reviews for {location_id}")
            return reviews

        except HttpError as e:
            logger.error(f"❌ API error fetching reviews: {e}")
            return None
        except Exception as e:
            logger.error(f"❌ Failed to get reviews: {e}")
            return None

    def get_new_reviews(self, location_id, since_hours=24):
        """
        Get reviews from the last N hours

        Args:
            location_id: Location ID
            since_hours: Only get reviews from last N hours

        Returns:
            List of new reviews
        """
        try:
            all_reviews = self.get_reviews(location_id)

            if not all_reviews:
                return []

            # Filter by timestamp
            cutoff_time = datetime.now() - timedelta(hours=since_hours)
            new_reviews = []

            for review in all_reviews:
                # Convert timestamp (milliseconds)
                timestamp = review.get('createTime', '')
                if timestamp:
                    try:
                        # Parse ISO timestamp
                        review_time = datetime.fromisoformat(timestamp.replace('Z', '+00:00'))
                        if review_time > cutoff_time:
                            new_reviews.append(review)
                    except:
                        # If can't parse, include it
                        new_reviews.append(review)

            logger.info(f"✅ Found {len(new_reviews)} new reviews (last {since_hours} hours)")
            return new_reviews

        except Exception as e:
            logger.error(f"❌ Failed to get new reviews: {e}")
            return []

    def post_reply(self, review_id, comment):
        """
        Post a reply to a review

        Args:
            review_id: Full review resource name
            comment: Reply text

        Returns:
            True if successful, False otherwise
        """
        try:
            if not self.service:
                if not self.authenticate():
                    return False

            reply_body = {
                'comment': comment
            }

            result = self.service.accounts().locations().reviews().reply(
                name=review_id,
                body=reply_body
            ).execute()

            logger.info(f"✅ Replied to review {review_id}")
            return True

        except HttpError as e:
            logger.error(f"❌ API error posting reply: {e}")
            return False
        except Exception as e:
            logger.error(f"❌ Failed to post reply: {e}")
            return False

    def get_location_info(self, location_id):
        """Get detailed information about a location"""
        try:
            if not self.service:
                if not self.authenticate():
                    return None

            # Handle both short ID and full name
            if not location_id.startswith('locations/'):
                location_name = self.get_location_name(location_id)
            else:
                location_name = location_id

            location = self.service.accounts().locations().get(
                name=location_name
            ).execute()

            logger.info(f"✅ Fetched location info for {location_id}")
            return location

        except HttpError as e:
            logger.error(f"❌ API error fetching location: {e}")
            return None
        except Exception as e:
            logger.error(f"❌ Failed to get location info: {e}")
            return None

    def has_replied(self, review):
        """Check if a review has been replied to"""
        return 'reply' in review and review['reply'].get('comment')


# Singleton instance
gmb_client = GoogleMyBusinessClient()

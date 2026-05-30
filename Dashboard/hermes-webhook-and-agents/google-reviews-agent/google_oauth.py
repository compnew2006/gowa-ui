#!/usr/bin/env python3
"""
Google OAuth Authentication for Google My Business API
Handles OAuth 2.0 flow and token management
"""

import os
import json
import logging
from pathlib import Path
from datetime import datetime, timedelta

try:
    from google.oauth2.credentials import Credentials
    from google_auth_oauthlib.flow import InstalledAppFlow
    from google.auth.transport.requests import Request
    import google.auth
except ImportError:
    print("❌ Google libraries not installed. Run:")
    print("pip install google-auth google-auth-oauthlib google-auth-httplib2 google-api-python-client")
    exit(1)

logger = logging.getLogger(__name__)

class GoogleOAuthManager:
    """Manage Google OAuth authentication"""

    SCOPES = ['https://www.googleapis.com/auth/business.manage']

    def __init__(self, config_dir='/opt/google-reviews-agent'):
        self.config_dir = Path(config_dir)
        self.credentials_file = self.config_dir / 'credentials.json'
        self.token_file = self.config_dir / 'token.json'
        self.client_secrets_file = self.config_dir / 'client_secrets.json'

    def setup_oauth(self, credentials_json=None):
        """
        Setup OAuth flow for Google My Business API

        Args:
            credentials_json: OAuth client secrets JSON string

        Returns:
            Credentials object if successful, None otherwise
        """
        try:
            # Save credentials if provided
            if credentials_json:
                with open(self.client_secrets_file, 'w') as f:
                    f.write(credentials_json)
                logger.info("✅ Client secrets saved")

            # Check if client secrets exist
            if not self.client_secrets_file.exists():
                logger.error("❌ Client secrets not found")
                logger.info("💡 Run setup first or provide credentials_json")
                return None

            # Load client secrets
            flow = InstalledAppFlow.from_client_secrets_file(
                self.client_secrets_file,
                self.SCOPES
            )

            # Run OAuth flow
            logger.info("🔐 Starting OAuth flow...")
            credentials = flow.run_local_server(port=0)

            # Save credentials
            self.save_credentials(credentials)
            logger.info("✅ OAuth credentials saved")

            return credentials

        except Exception as e:
            logger.error(f"❌ OAuth setup failed: {e}")
            return None

    def load_credentials(self):
        """Load saved credentials, refresh if needed"""
        try:
            if not self.token_file.exists():
                logger.warning("⚠️ No token file found")
                return None

            credentials = Credentials.from_authorized_user_file(
                self.token_file,
                self.SCOPES
            )

            # Refresh if expired
            if credentials.expired and credentials.refresh_token:
                logger.info("🔄 Refreshing credentials...")
                credentials.refresh(Request())
                self.save_credentials(credentials)
                logger.info("✅ Credentials refreshed")

            return credentials

        except Exception as e:
            logger.error(f"❌ Failed to load credentials: {e}")
            return None

    def save_credentials(self, credentials):
        """Save credentials to token file"""
        try:
            self.config_dir.mkdir(parents=True, exist_ok=True)

            token_data = {
                'token': credentials.token,
                'refresh_token': credentials.refresh_token,
                'token_uri': credentials.token_uri,
                'client_id': credentials.client_id,
                'client_secret': credentials.client_secret,
                'scopes': credentials.scopes,
                'expiry': credentials.expiry.isoformat() if credentials.expiry else None
            }

            with open(self.token_file, 'w') as f:
                json.dump(token_data, f, indent=2)

            logger.info("✅ Credentials saved")

        except Exception as e:
            logger.error(f"❌ Failed to save credentials: {e}")

    def get_credentials(self):
        """Get valid credentials (load or refresh)"""
        credentials = self.load_credentials()

        if not credentials:
            logger.warning("⚠️ No credentials found, run setup first")
            return None

        if credentials.expired:
            if not credentials.refresh_token:
                logger.error("❌ Credentials expired and no refresh token")
                return None

            try:
                credentials.refresh(Request())
                self.save_credentials(credentials)
                logger.info("✅ Credentials refreshed")
            except Exception as e:
                logger.error(f"❌ Failed to refresh credentials: {e}")
                return None

        return credentials

    def is_authenticated(self):
        """Check if user is authenticated"""
        credentials = self.get_credentials()
        return credentials is not None and credentials.valid

    def revoke_credentials(self):
        """Revoke and delete credentials"""
        try:
            if self.token_file.exists():
                self.token_file.unlink()
                logger.info("✅ Credentials revoked")

        except Exception as e:
            logger.error(f"❌ Failed to revoke credentials: {e}")


# Singleton instance
oauth_manager = GoogleOAuthManager()

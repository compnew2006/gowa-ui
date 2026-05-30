#!/bin/bash
#
# Setup Google Reviews Poller as systemd service
#

echo "🔧 Creating Google Reviews Poller service..."

# Create systemd service file
sudo tee /etc/systemd/system/google-reviews-poller.service > /dev/null << EOF
[Unit]
Description=Google Reviews Poller - Auto-reply to Google Maps reviews
After=network.target hermes-facebook-webhook.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/google-reviews-agent
Environment="PATH=/opt/google-reviews-agent/venv/bin:/usr/local/bin:/usr/bin:/bin"
Environment="POLLING_INTERVAL_MINUTES=10"
ExecStart=/opt/google-reviews-agent/venv/bin/python3 /opt/google-reviews-agent/google_reviews_poller.py
Restart=always
RestartSec=60
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Create log directory
sudo mkdir -p /var/log/hermes
sudo chown www-data:www-data /var/log/hermes

# Create Python virtual environment
echo "🐍 Creating Python virtual environment..."
cd /opt/google-reviews-agent
sudo -u www-data python3 -m venv venv

# Install Google libraries
echo "📦 Installing Google API libraries..."
sudo -u www-data /opt/google-reviews-agent/venv/bin/pip install --upgrade pip > /dev/null 2>&1
sudo -u www-data /opt/google-reviews-agent/venv/bin/pip install \
    google-auth \
    google-auth-oauthlib \
    google-auth-httplib2 \
    google-api-python-client \
    requests \
    > /dev/null 2>&1

# Reload systemd
sudo systemctl daemon-reload

# Enable service (but don't start yet - need OAuth first)
sudo systemctl enable google-reviews-poller

echo ""
echo "✅ Google Reviews Poller service created!"
echo ""
echo "📋 Service Details:"
echo "   • Name: google-reviews-poller"
echo "   • Polling Interval: 10 minutes (configurable)"
echo "   • Log: /var/log/hermes/google_reviews_poller.log"
echo ""
echo "⚠️  Before starting, you need to:"
echo "   1. Authenticate with Google:"
echo "      python3 -c \"from google_oauth import oauth_manager; oauth_manager.setup_oauth()\""
echo ""
echo "   2. Add at least one location:"
echo "      sudo /opt/google-reviews-agent/add-location.sh"
echo ""
echo "   3. Then start the service:"
echo "      sudo systemctl start google-reviews-poller"
echo ""
echo "📊 Management Commands:"
echo "   • Start:   sudo systemctl start google-reviews-poller"
echo "   • Stop:    sudo systemctl stop google-reviews-poller"
echo "   • Status:  sudo systemctl status google-reviews-poller"
echo "   • Logs:    tail -f /var/log/hermes/google_reviews_poller.log"
echo ""

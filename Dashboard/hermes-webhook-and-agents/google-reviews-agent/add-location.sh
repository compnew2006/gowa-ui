#!/bin/bash
#
# add-location.sh - Add a new Google business location
#

LOCATIONS_CONFIG="/opt/google-reviews-agent/locations_config.json"
KNOWLEDGE_DIR="/opt/google-reviews-agent/knowledge"
NOTIFICATION_CONFIG="/opt/hermes-webhook/notification_config.json"

echo "╔════════════════════════════════════════════════════════╗"
echo "║       Add New Google Business Location                  ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

# Check if authenticated
echo "🔐 Checking Google authentication..."
python3 << 'PYTHON'
import sys
sys.path.insert(0, '/opt/google-reviews-agent')
from google_oauth import oauth_manager

if not oauth_manager.is_authenticated():
    print("❌ Not authenticated with Google!")
    print("💡 Run: setup-google-oauth first")
    exit(1)
else:
    print("✅ Authenticated")
PYTHON

if [ $? -ne 0 ]; then
    echo ""
    echo "Please authenticate first:"
    echo "  cd /opt/google-reviews-agent"
    echo "  python3 -c \"from google_oauth import oauth_manager; oauth_manager.setup_oauth()\""
    exit 1
fi

echo ""
echo "Available locations:"
python3 << 'PYTHON'
import sys
sys.path.insert(0, '/opt/google-reviews-agent')
from google_api_client import gmb_client

if gmb_client.authenticate():
    accounts = gmb_client.get_accounts()
    if accounts:
        for acc in accounts:
            print(f"\n📊 Account: {acc.get('name', 'Unknown')}")
            print(f"   Type: {acc.get('type', 'Unknown')}")
            print(f"   Name: {acc.get('accountName', 'Unknown')}")

            locations = gmb_client.get_locations(acc['name'])
            if locations:
                print(f"\n   📍 Locations ({len(locations)}):")
                for loc in locations[:10]:  # Show first 10
                    loc_id = loc['name'].split('/')[-1]
                    loc_name = loc.get('locationName', 'Unknown')
                    print(f"      • {loc_name}")
                    print(f"        ID: {loc_id}")
                    print(f"        Address: {loc.get('address', {}).get('addressLines', ['N/A'])[0]}")
PYTHON

echo ""
echo ""
read -p "🆔 Enter Location ID (from list above): " LOCATION_ID
read -p "📝 Enter Location Name (display name): " LOCATION_NAME
read -p "🏢 Enter Business Name: " BUSINESS_NAME
read -p "📧 Enter Account ID (e.g., accounts/1234567890): " ACCOUNT_ID

# Ask about Facebook page linking
echo ""
echo "🔗 Link to Facebook Page? (Optional - allows sharing knowledge base)"
echo "   If you have a Facebook page for this business, enter the Page ID"
echo "   Leave empty to use separate knowledge file"
read -p "📘 Facebook Page ID (optional, press Enter to skip): " FACEBOOK_PAGE_ID

# Validate inputs
if [ -z "$LOCATION_ID" ] || [ -z "$LOCATION_NAME" ] || [ -z "$BUSINESS_NAME" ]; then
    echo "❌ Error: Location ID, Name, and Business Name are required!"
    exit 1
fi

# Generate config ID
CONFIG_ID="${BUSINESS_NAME// /_}_${LOCATION_ID}"
CONFIG_ID=$(echo "$CONFIG_ID" | tr '[:upper:]' '[:lower:]')

# Create knowledge file
KNOWLEDGE_FILE="$KNOWLEDGE_DIR/${CONFIG_ID}_knowledge.md"
echo "Creating knowledge file: $KNOWLEDGE_FILE"

cat > "$KNOWLEDGE_FILE" << EOF
# Business Knowledge for $BUSINESS_NAME

## Business Overview
- **Name**: $BUSINESS_NAME
- **Location**: $LOCATION_NAME
- **Location ID**: $LOCATION_ID
- **Account ID**: $ACCOUNT_ID
- **Type**: Local Business
- **Added**: $(date +%Y-%m-%d)

## Services
Add your services here...

## Products
Add your products here...

## Team & Contact
- **Location**: $LOCATION_NAME
- **Response Time**: Within 24 hours
- **Contact**: Add contact info

## Location Details
- **Address**: Add address
- **Phone**: Add phone
- **Hours**: Add business hours

## Customer Service Approach
- Professional and friendly
- Local community focus
- Quick response to reviews
- Quality guarantee

## Common Review Themes & Responses

### 5-Star Reviews
Thank customers for positive reviews, mention specific aspects

### Critical Reviews
- Apologize sincerely
- Address specific concerns
- Offer to make it right
- Take conversation offline if needed

## Common Questions & Answers

**Q: What are your hours?**
A: Add your hours

**Q: Do you offer [service]?**
A: Add info about services

## Special Offers

## Success Stories

## Tips & Tricks

## Product Highlights

---
*This knowledge base is automatically updated as the AI learns from review interactions.*
EOF

# Update locations_config.json
echo "Updating locations configuration..."
python3 << PYTHON
import json

# Load existing config
try:
    with open('$LOCATIONS_CONFIG', 'r') as f:
        config = json.load(f)
except:
    config = {"locations": {}}

# Add new location
config['locations']['$CONFIG_ID'] = {
    "location_id": "$LOCATION_ID",
    "location_name": "$LOCATION_NAME",
    "account_id": "$ACCOUNT_ID",
    "business_name": "$BUSINESS_NAME",
    "knowledge_file": "$KNOWLEDGE_FILE",
    "facebook_page_id": "$FACEBOOK_PAGE_ID",
    "telegram_bot_token": "",
    "telegram_chat_id": "",
    "whatsapp_phone": "",
    "whatsapp_api_key": "",
    "enabled": True,
    "auto_reply": True,
    "reply_delay_hours": 0,
    "language": "auto"
}

# Save config
with open('$LOCATIONS_CONFIG', 'w') as f:
    json.dump(config, f, indent=2)

print("✅ Location configuration updated")
PYTHON

# Update notification_config.json (for compatibility with existing system)
echo "Updating notification configuration..."
python3 << PYTHON
import json

# Load existing config
try:
    with open('$NOTIFICATION_CONFIG', 'r') as f:
        config = json.load(f)
except:
    config = {"pages": {}}

# Add notifications for new location
config['pages']['$CONFIG_ID'] = {
    "page_name": "$BUSINESS_NAME",
    "page_id": "$CONFIG_ID",
    "telegram_bot_token": "",
    "telegram_chat_id": "",
    "whatsapp_phone": "",
    "whatsapp_api_key": ""
}

# Save config
with open('$NOTIFICATION_CONFIG', 'w') as f:
    json.dump(config, f, indent=2)

print("✅ Notification configuration updated")
PYTHON

# Set proper permissions
sudo chmod 644 "$KNOWLEDGE_FILE"
sudo chown www-data:www-data "$KNOWLEDGE_FILE"
sudo chmod 644 "$LOCATIONS_CONFIG"
sudo chown www-data:www-data "$LOCATIONS_CONFIG"

echo ""
echo "✅ Location added successfully!"
echo ""
echo "📋 Summary:"
echo "   • Config ID: $CONFIG_ID"
echo "   • Business: $BUSINESS_NAME"
echo "   • Location: $LOCATION_NAME"
echo "   • Location ID: $LOCATION_ID"
echo "   • Knowledge File: $KNOWLEDGE_FILE"

if [ ! -z "$FACEBOOK_PAGE_ID" ]; then
    echo "   • 🔗 Linked to Facebook Page: $FACEBOOK_PAGE_ID"
    echo "   ✅ Will use Facebook knowledge base!"
    echo ""
    echo "   💡 This location will share knowledge with:"
    echo "      • Facebook comments"
    echo "      • Instagram comments"
    echo "      • Google reviews"
    echo "      All use the SAME knowledge base!"
else
    echo "   • 🔗 Facebook: Not linked (separate knowledge)"
fi

echo ""
echo "📝 Next Steps:"

if [ ! -z "$FACEBOOK_PAGE_ID" ]; then
    echo "   1. Edit the Facebook knowledge base (shared across platforms):"
    echo "      sudo nano /opt/hermes-webhook/knowledge/${FACEBOOK_PAGE_ID}_knowledge.md"
    echo ""
    echo "   💡 Any updates to this file will apply to:"
    echo "      • Facebook comments"
    echo "      • Instagram comments"
    echo "      • Google reviews"
else
    echo "   1. Edit the knowledge file:"
    echo "      sudo nano $KNOWLEDGE_FILE"
fi

echo ""
echo "   2. Setup notifications (optional):"
echo "      setup-google-notifications"
echo ""
echo "   3. Start the poller:"
echo "      sudo systemctl start google-reviews-poller"
echo ""
echo "   4. Check logs:"
echo "      tail -f /var/log/hermes/google_reviews_poller.log"
echo ""

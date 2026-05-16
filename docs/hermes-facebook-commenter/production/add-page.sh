#!/bin/bash
#
# add-page.sh - Add a new Facebook page to the multi-tenant system
#

PAGES_CONFIG="/opt/hermes-webhook/pages_config.json"
KNOWLEDGE_DIR="/opt/hermes-webhook/knowledge"
NOTIFICATION_CONFIG="/opt/hermes-webhook/notification_config.json"

echo "╔════════════════════════════════════════════════════════╗"
echo "║     Add New Facebook Page to Multi-Tenant System        ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

# Prompt for page details
read -p "📄 Page ID: " PAGE_ID
read -p "📝 Page Name: " PAGE_NAME
read -p "🔑 Page Access Token: " PAGE_TOKEN
read -p "📱 Telegram Bot Token (optional, press Enter to skip): " TG_TOKEN
read -p "💬 Telegram Chat ID (optional, press Enter to skip): " TG_CHAT_ID

# Validate inputs
if [ -z "$PAGE_ID" ] || [ -z "$PAGE_NAME" ] || [ -z "$PAGE_TOKEN" ]; then
    echo "❌ Error: Page ID, Name, and Token are required!"
    exit 1
fi

# Create knowledge file
KNOWLEDGE_FILE="$KNOWLEDGE_DIR/${PAGE_ID}_knowledge.md"
echo "Creating knowledge file: $KNOWLEDGE_FILE"

cat > "$KNOWLEDGE_FILE" << EOF
# Business Knowledge for $PAGE_NAME

## Business Overview
- **Name**: $PAGE_NAME
- **Page ID**: $PAGE_ID
- **Type**: Business
- **Founded**: $(date +%Y)

## Services
Add your services here...

## Team & Contact
- **Page**: $PAGE_NAME
- **Location**: Add your location
- **Response Time: Within 2 hours

## Location
Add your location details...

## Pricing
Add your pricing information...

## Business Hours
Add your business hours...

## Contact
Add contact information...

## Customer Service Approach
Add your customer service approach...

## Common Questions & Answers

**Q: Who owns/manages this page?**
A: This is the official page of $PAGE_NAME.

## Promotions

## Success Stories

## Tips & Tricks

## Portfolio Examples

---
*This knowledge base is automatically updated as the AI learns from customer interactions.*
EOF

# Update pages_config.json
echo "Updating pages configuration..."
python3 << PYTHON
import json

# Load existing config
try:
    with open('$PAGES_CONFIG', 'r') as f:
        config = json.load(f)
except:
    config = {"pages": {}}

# Add new page
config['pages']['$PAGE_ID'] = {
    "page_name": "$PAGE_NAME",
    "page_id": "$PAGE_ID",
    "access_token": "$PAGE_TOKEN",
    "knowledge_file": "$KNOWLEDGE_FILE"
}

# Add Telegram if provided
if '$TG_TOKEN' and '$TG_CHAT_ID':
    config['pages']['$PAGE_ID']['telegram_bot_token'] = '$TG_TOKEN'
    config['pages']['$PAGE_ID']['telegram_chat_id'] = '$TG_CHAT_ID'

# Save config
with open('$PAGES_CONFIG', 'w') as f:
    json.dump(config, f, indent=2)

print("✅ Pages configuration updated")
PYTHON

# Update notification_config.json
echo "Updating notification configuration..."
python3 << PYTHON
import json

# Load existing config
try:
    with open('$NOTIFICATION_CONFIG', 'r') as f:
        config = json.load(f)
except:
    config = {"pages": {}}

# Add notifications for new page
config['pages']['$PAGE_ID'] = {
    "page_name": "$PAGE_NAME",
    "page_id": "$PAGE_ID"
}

# Add Telegram if provided
if '$TG_TOKEN' and '$TG_CHAT_ID':
    config['pages']['$PAGE_ID']['telegram_bot_token'] = '$TG_TOKEN'
    config['pages']['$PAGE_ID']['telegram_chat_id'] = '$TG_CHAT_ID'

# Save config
with open('$NOTIFICATION_CONFIG', 'w') as f:
    json.dump(config, f, indent=2)

print("✅ Notification configuration updated")
PYTHON

# Set proper permissions
sudo chmod 644 "$KNOWLEDGE_FILE"
sudo chown www-data:www-data "$KNOWLEDGE_FILE"
sudo chmod 644 "$PAGES_CONFIG"
sudo chown www-data:www-data "$PAGES_CONFIG"
sudo chmod 644 "$NOTIFICATION_CONFIG"
sudo chown www-data:www-data "$NOTIFICATION_CONFIG"

echo ""
echo "✅ Page added successfully!"
echo ""
echo "📋 Summary:"
echo "   • Page ID: $PAGE_ID"
echo "   • Page Name: $PAGE_NAME"
echo "   • Knowledge File: $KNOWLEDGE_FILE"
echo ""
echo "📝 Next Steps:"
echo "   1. Edit the knowledge file to add your business info:"
echo "      sudo nano $KNOWLEDGE_FILE"
echo "   2. Restart the webhook service:"
echo "      sudo systemctl restart hermes-facebook-webhook"
echo "   3. Subscribe to the webhook:"
echo "      https://fbwebhook.ofuqalmadenah.com/webhook"
echo ""
echo "🔧 Don't forget to:"
echo "   - Add your business information to the knowledge file"
echo "   - Test the webhook is receiving events for this page"
echo "   - Configure Telegram notifications if needed"
echo ""

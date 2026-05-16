#!/bin/bash
#
# list-pages.sh - List all configured Facebook pages
#

echo "╔════════════════════════════════════════════════════════╗"
echo "║           Configured Facebook Pages                     ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

python3 << 'PYTHON'
import json
from pathlib import Path

PAGES_CONFIG = Path("/opt/hermes-webhook/pages_config.json")

try:
    with open(PAGES_CONFIG, 'r') as f:
        config = json.load(f)

    pages = config.get("pages", {})

    if not pages:
        print("⚠️  No pages configured yet.")
        print("\n💡 Add a page with: sudo /opt/hermes-webhook/add-page.sh")
    else:
        print(f"📊 Total Pages Configured: {len(pages)}\n")

        for page_id, page_info in pages.items():
            print(f"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
            print(f"📄 Page Name: {page_info.get('page_name', 'Unknown')}")
            print(f"🆔 Page ID: {page_id}")
            print(f"📁 Knowledge File: {page_info.get('knowledge_file', 'Not set')}")

            # Check if knowledge file exists
            knowledge_file = Path(page_info.get('knowledge_file', ''))
            if knowledge_file.exists():
                size = knowledge_file.stat().st_size
                print(f"   ✅ Knowledge file exists ({size} bytes)")
            else:
                print(f"   ⚠️  Knowledge file NOT found!")

            # Check Telegram
            if page_info.get('telegram_bot_token') and page_info.get('telegram_chat_id'):
                print(f"📱 Telegram: ✅ Configured (@bot)")
            else:
                print(f"📱 Telegram: ❌ Not configured")

            # Check WhatsApp
            if page_info.get('whatsapp_phone'):
                print(f"💬 WhatsApp: ✅ Configured")
            else:
                print(f"💬 WhatsApp: ❌ Not configured")

            print()

except Exception as e:
    print(f"❌ Error: {e}")
PYTHON

echo "════════════════════════════════════════════════════════"
echo ""
echo "💡 Commands:"
echo "   • Add page:    sudo /opt/hermes-webhook/add-page.sh"
echo "   • Check health: curl http://localhost:5000/health"
echo "   • View logs:   tail -f /var/log/hermes/facebook_webhook.log"
echo ""

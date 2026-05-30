#!/bin/bash
#
# list-locations.sh - List all configured Google business locations
#

LOCATIONS_CONFIG="/opt/google-reviews-agent/locations_config.json"

echo "╔════════════════════════════════════════════════════════╗"
echo "║         Configured Google Business Locations             ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

python3 << 'PYTHON'
import json
from pathlib import Path

LOCATIONS_CONFIG = Path("/opt/google-reviews-agent/locations_config.json")

try:
    with open(LOCATIONS_CONFIG, 'r') as f:
        config = json.load(f)

    locations = config.get("locations", {})

    if not locations:
        print("⚠️  No locations configured yet.")
        print("\n💡 Add a location with:")
        print("   sudo /opt/google-reviews-agent/add-location.sh")
    else:
        print(f"📊 Total Locations Configured: {len(locations)}\n")

        for config_id, loc_info in locations.items():
            print(f"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
            print(f"🏢 Business: {loc_info.get('business_name', 'Unknown')}")
            print(f"📍 Location: {loc_info.get('location_name', 'Unknown')}")
            print(f"🆔 Location ID: {loc_info.get('location_id', 'Unknown')}")
            print(f"🆔 Config ID: {config_id}")

            # Check Facebook page linking
            fb_page_id = loc_info.get('facebook_page_id')
            if fb_page_id:
                print(f"🔗 Facebook Page: ✅ Linked to {fb_page_id}")
                print(f"   📁 Uses Facebook knowledge base!")
                print(f"   📁 Location knowledge: {loc_info.get('knowledge_file', 'Not set')} (fallback)")

                # Check if Facebook knowledge exists
                fb_knowledge = Path(f"/opt/hermes-webhook/knowledge/{fb_page_id}_knowledge.md")
                if fb_knowledge.exists():
                    size = fb_knowledge.stat().st_size
                    print(f"   ✅ FB knowledge exists ({size} bytes)")
                else:
                    print(f"   ⚠️  FB knowledge NOT found!")
            else:
                print(f"🔗 Facebook: ❌ Not linked (separate knowledge)")
                print(f"📁 Knowledge File: {loc_info.get('knowledge_file', 'Not set')}")

                # Check if knowledge file exists
                knowledge_file = Path(loc_info.get('knowledge_file', ''))
                if knowledge_file.exists():
                    size = knowledge_file.stat().st_size
                    print(f"   ✅ Knowledge file exists ({size} bytes)")
                else:
                    print(f"   ⚠️  Knowledge file NOT found!")

            # Check status
            enabled = loc_info.get('enabled', True)
            auto_reply = loc_info.get('auto_reply', True)

            if enabled:
                print(f"🟢 Status: Enabled (Auto-reply: {'Yes' if auto_reply else 'No'})")
            else:
                print(f"🔴 Status: Disabled")

            # Check Telegram
            if loc_info.get('telegram_bot_token') and loc_info.get('telegram_chat_id'):
                print(f"📱 Telegram: ✅ Configured")
            else:
                print(f"📱 Telegram: ❌ Not configured")

            # Check WhatsApp
            if loc_info.get('whatsapp_phone'):
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
echo "   • Add location:    sudo /opt/google-reviews-agent/add-location.sh"
echo "   • Test poller:     sudo /opt/google-reviews-agent/google_reviews_poller.py"
echo "   • View logs:       tail -f /var/log/hermes/google_reviews_poller.log"
echo "   • Check service:   sudo systemctl status google-reviews-poller"
echo ""

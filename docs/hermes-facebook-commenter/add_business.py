#!/usr/bin/env python3
"""
Add a new business to multi-business Facebook manager
"""

import json
import sys
from pathlib import Path

def add_business_interactive():
    """Interively add a new business"""
    print("🏪 Add New Business to Multi-Business Facebook Manager")
    print("=" * 60)
    print()

    # Business ID
    business_id = input("Business ID (unique identifier, e.g., maktabat_al_arkan): ").strip()
    if not business_id:
        print("❌ Business ID is required")
        return

    # Business info
    name = input("Business name (e.g., Maktabat Al-Arkan): ").strip()
    page_name = input("Facebook page name: ").strip()
    page_id = input("Facebook Page ID: ").strip()
    page_access_token = input("Facebook Page Access Token: ").strip()

    # Basic settings
    auto_reply = input("Enable auto-reply? (y/n): ").strip().lower() == 'y'
    auto_post = input("Enable auto-post? (y/n): ").strip().lower() == 'y'

    # Services
    print("\n📋 Enter your services (one per line, empty line to finish):")
    services = []
    while True:
        service = input(f"Service {len(services) + 1}: ").strip()
        if not service:
            break
        services.append(service)

    # Prices
    print("\n💰 Enter your prices (format: item_name - price, empty line to finish):")
    prices = {}
    while True:
        price_entry = input(f"Price {len(prices) + 1}: ").strip()
        if not price_entry:
            break
        if '-' in price_entry:
            item, price = price_entry.split('-', 1)
            prices[item.strip()] = price.strip()

    # Location
    print("\n📍 Location information:")
    address = input("Address: ").strip()
    landmark = input("Landmark (optional): ").strip()
    phone = input("Phone: ").strip()

    # Hours
    print("\n⏰ Business hours:")
    hours_general = input("General hours (e.g., Daily 9am - 9pm): ").strip()

    # Create config
    config = {
        "name": name,
        "page_id": page_id,
        "page_name": page_name,
        "page_access_token": page_access_token,
        "auto_reply": auto_reply,
        "auto_post": auto_post,
        "services": services,
        "prices": prices,
        "location": {
            "address": address,
            "landmark": landmark,
            "phone": phone
        },
        "hours": {
            "general": hours_general
        },
        "tone": "friendly_professional",
        "reply_language": "auto"
    }

    # Save config
    config_dir = Path.home() / ".hermes" / "businesses"
    config_dir.mkdir(parents=True, exist_ok=True)

    config_file = config_dir / f"{business_id}.json"
    with open(config_file, 'w', encoding='utf-8') as f:
        json.dump(config, f, indent=2, ensure_ascii=False)

    print(f"\n✅ Business '{name}' added successfully!")
    print(f"📁 Config saved to: {config_file}")
    print("\n📊 To test, run:")
    print(f"   python3 -c \"from multi_business_facebook import manager; print(manager.list_businesses())\"")

def main():
    if len(sys.argv) > 1 and sys.argv[1] == '--list':
        # List all businesses
        from multi_business_facebook import manager
        businesses = manager.list_businesses()

        print(f"📊 Total businesses: {len(businesses)}\n")
        for business in businesses:
            print(f"🏪 {business['name']}")
            print(f"   ID: {business['id']}")
            print(f"   Page: {business['page_name']}")
            print(f"   Auto-reply: {'✅' if business['auto_reply'] else '❌'}")
            print(f"   Auto-post: {'✅' if business['auto_post'] else '❌'}")
            print()

    elif len(sys.argv) > 1 and sys.argv[1] == '--test':
        # Test a business
        from multi_business_facebook import manager

        business_id = sys.argv[2] if len(sys.argv) > 2 else None
        if not business_id:
            print("❌ Please provide business ID: --test <business_id>")
            return

        business = manager.get_business(business_id)
        if not business:
            print(f"❌ Business '{business_id}' not found")
            return

        print(f"🏪 Testing business: {business.name}")
        print(f"   Services: {', '.join(business.services[:3])}...")
        print(f"   Prices: {len(business.prices)} items")

        # Test reply generation
        test_comments = [
            "كم سعر البطاقة؟",
            "متى تفتحون؟",
            "أين تقعون؟"
        ]

        print("\n🤖 Testing auto-replies:")
        for comment in test_comments:
            reply = manager.generate_reply(business_id, comment, "Customer")
            print(f"\n   Question: {comment}")
            print(f"   Reply: {reply}")

    else:
        # Interactive mode
        add_business_interactive()

if __name__ == '__main__':
    main()

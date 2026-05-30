#!/bin/bash

echo "╔════════════════════════════════════════════════════════════╗"
echo "║     🤖 FACEBOOK AGENT - INTERACTIVE TESTING               ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "Choose a test mode:"
echo ""
echo "1) 💬 Interactive chat (type your own questions)"
echo "2) 🧪 Quick test (ask 5 common questions)"
echo "3) 📚 View current knowledge base"
echo "4) 🔄 Update knowledge base"
echo "5) ❌ Exit"
echo ""
read -p "Enter choice (1-5): " choice

case $choice in
    1)
        echo ""
        echo "Starting interactive chat..."
        echo "Press Ctrl+C to exit"
        echo ""
        python3 /opt/hermes-webhook/test_agent.py
        ;;
    2)
        echo ""
        echo "🧪 Running quick test with 5 questions..."
        echo ""
        python3 /opt/hermes-webhook/test_agent.py << 'INPUT'
كم سعر تصميم موقع؟
exit
INPUT
        echo ""
        sleep 2
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        python3 /opt/hermes-webhook/test_agent.py << 'INPUT'
ما هي الخدمات المقدمة؟
exit
INPUT
        echo ""
        sleep 2
        echo "✅ Quick test completed!"
        ;;
    3)
        echo ""
        echo "📚 Current Business Knowledge:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        cat /opt/hermes-webhook/knowledge/895247390337022_knowledge.md
        ;;
    4)
        echo ""
        echo "🔄 Opening knowledge base for editing..."
        sudo nano /opt/hermes-webhook/knowledge/895247390337022_knowledge.md
        echo ""
        echo "✅ Knowledge base updated!"
        echo "🔄 Restarting webhook..."
        sudo systemctl restart hermes-facebook-webhook
        echo "✅ Done!"
        ;;
    5)
        echo "👋 Goodbye!"
        exit 0
        ;;
    *)
        echo "❌ Invalid choice"
        exit 1
        ;;
esac

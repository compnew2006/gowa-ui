#!/usr/bin/env python3
"""Interactive testing tool for the KB-constrained Facebook agent."""

import os
import sys
from pathlib import Path

CURRENT_DIR = Path(__file__).resolve().parent
if str(CURRENT_DIR) not in sys.path:
    sys.path.insert(0, str(CURRENT_DIR))

from facebook_webhook_gunicorn import build_reply_result, load_knowledge_base


PAGE_ID = os.environ.get("HERMES_TEST_PAGE_ID", "895247390337022")

def load_knowledge():
    """Load business knowledge"""
    try:
        return load_knowledge_base(PAGE_ID)
    except Exception as e:
        print(f"⚠️  Warning: Could not load knowledge file: {e}")
        return ""

def generate_response(message):
    """Generate the same grounded response used by the live webhook."""
    try:
        return build_reply_result(PAGE_ID, message).response
    except Exception as e:
        return f"❌ Error: {e}"

def print_header():
    """Print welcome header"""
    print("\n" + "="*60)
    print("🤖 FACEBOOK AGENT TESTING CONSOLE")
    print("="*60)
    print("\n💬 Chat with your Facebook agent to test its responses")
    print("📝 Type 'quit' or 'exit' to stop")
    print("📚 Type 'knowledge' to see current business knowledge")
    print("🔄 Type 'reload' to reload knowledge base")
    print("\n" + "="*60 + "\n")

def print_response(label, text):
    """Print formatted message"""
    colors = {
        'customer': '\033[94m',    # Blue
        'agent': '\033[92m',       # Green
        'system': '\033[93m',      # Yellow
        'reset': '\033[0m'
    }
    
    color = colors.get(label, colors['reset'])
    icon = {
        'customer': '👤 Customer:',
        'agent': '🤖 Agent:',
        'system': 'ℹ️  System:'
    }.get(label, f'{label}:')
    
    print(f"{color}{icon}{colors['reset']}")
    print(f"{color}{text}{colors['reset']}\n")

def main():
    """Main interactive loop"""
    print_header()
    
    print_response('system', 
        "Agent is ready! You are now acting as a customer.\n"
        "Ask questions in Arabic or English to test the agent."
    )
    
    conversation_count = 0
    
    while True:
        try:
            # Get user input
            user_input = input("👤 Your message: ").strip()
            
            if not user_input:
                continue
                
            # Handle commands
            if user_input.lower() in ['quit', 'exit', 'q']:
                print("\n👋 Goodbye! Testing session ended.\n")
                break
                
            if user_input.lower() == 'knowledge':
                print_response('system', "Current Business Knowledge:\n" + load_knowledge())
                continue
                
            if user_input.lower() == 'reload':
                print_response('system', "🔄 Knowledge base reloaded!")
                continue
            
            # Generate response
            print_response('customer', user_input)
            print("\n⏳ Agent is thinking...", end='\r')
            
            response = generate_response(user_input)
            
            # Clear "thinking" message and show response
            print(" " * 50 + '\r')  # Clear line
            print_response('agent', response)
            
            conversation_count += 1
            
        except KeyboardInterrupt:
            print("\n\n👋 Goodbye! Testing session ended.\n")
            break
        except Exception as e:
            print(f"\n❌ Error: {e}\n")
    
    print(f"✅ Total conversations tested: {conversation_count}")

if __name__ == '__main__':
    main()

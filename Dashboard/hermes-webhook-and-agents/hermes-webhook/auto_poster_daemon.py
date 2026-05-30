#!/usr/bin/env python3
"""
Auto-Poster Daemon - Runs in background to process scheduled posts
"""

import time
import sys
from pathlib import Path
sys.path.insert(0, '/opt/hermes-webhook')
from auto_poster import auto_poster

def main():
    print("🚀 Auto-Poster Daemon Started")
    print("⏰ Processing scheduled posts every 60 seconds...")
    print("Press Ctrl+C to stop")
    
    while True:
        try:
            processed = auto_poster.process_queue()
            if processed > 0:
                print(f"✅ Processed {processed} posts")
            time.sleep(60)  # Check every minute
        except KeyboardInterrupt:
            print("\n👋 Auto-Poster Daemon Stopped")
            break
        except Exception as e:
            print(f"❌ Error: {e}")
            time.sleep(60)

if __name__ == '__main__':
    main()

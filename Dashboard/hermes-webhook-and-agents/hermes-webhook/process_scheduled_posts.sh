#!/bin/bash
# Process scheduled posts - runs every minute via cron

cd /opt/hermes-webhook
python3 << PYTHON
import sys
sys.path.insert(0, '/opt/hermes-webhook')
from auto_poster import auto_poster
processed = auto_poster.process_queue()
print(f"Processed {processed} scheduled posts")
PYTHON

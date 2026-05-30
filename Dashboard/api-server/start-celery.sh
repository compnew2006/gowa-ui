#!/bin/bash
cd /opt/ai-dashboard/api-server
source venv/bin/activate
exec celery -A app.workers.celery_app worker --loglevel=info --queues=high,normal,low --concurrency=2

#!/bin/bash
set -e
cd /home/runner/workspace/artifacts/api-server
echo "Installing Python dependencies..."
pip install -r requirements.txt -q 2>&1 | tail -3
echo "Starting FastAPI server on port ${PORT:-8080}..."
exec uvicorn app.main:app --host 0.0.0.0 --port "${PORT:-8080}" --reload --log-level info

#!/bin/sh
export PYTHONPATH=/opt/homebrew/lib/python3.11/site-packages${PYTHONPATH:+:$PYTHONPATH}
exec /opt/homebrew/opt/python@3.11/bin/python3.11 -m desloppify.cli "$@"

#!/bin/sh
export PYTHONPATH=/opt/homebrew/lib/python3.13/site-packages${PYTHONPATH:+:$PYTHONPATH}
exec /opt/homebrew/opt/python@3.13/bin/python3.13 -m desloppify.cli "$@"

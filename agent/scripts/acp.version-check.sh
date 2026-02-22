#!/bin/bash

# ACP Version Check Script
# Extracts and displays the current ACP version from AGENT.md

set -e

# Colors for output using tput (more reliable than ANSI codes)
if command -v tput >/dev/null 2>&1 && [ -t 1 ]; then
    RED=$(tput setaf 1)
    GREEN=$(tput setaf 2)
    YELLOW=$(tput setaf 3)
    BLUE=$(tput setaf 4)
    BOLD=$(tput bold)
    NC=$(tput sgr0)
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    NC=''
fi

# Check if AGENT.md exists
if [ ! -f "AGENT.md" ]; then
    echo "Error: AGENT.md not found in current directory"
    echo "This script should be run from your project root where AGENT.md is located."
    exit 1
fi

# Extract version from AGENT.md
VERSION=$(grep -m 1 "^\*\*Version\*\*:" AGENT.md | sed 's/.*: //')
CREATED=$(grep -m 1 "^\*\*Created\*\*:" AGENT.md | sed 's/.*: //')
STATUS=$(grep -m 1 "^\*\*Status\*\*:" AGENT.md | sed 's/.*: //')

# Display version information
echo "${BLUE}📦 ACP Version Information${NC}"
echo ""
echo "Version: $VERSION"
echo "Created: $CREATED"
echo "Status: $STATUS"
echo ""
echo "${GREEN}✓${NC} ACP is installed"
echo ""
echo "To check for updates: ./agent/scripts/acp.version-check-for-updates.sh"
echo "To update ACP: ./agent/scripts/acp.version-update.sh"

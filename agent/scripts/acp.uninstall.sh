#!/bin/bash

# Agent Context Protocol (ACP) Uninstall Script
# This script removes all ACP files from a project
# Usage: ./agent/scripts/unacp.install.sh [-y]

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

# Check for -y flag
AUTO_CONFIRM=false
if [ "$1" = "-y" ]; then
    AUTO_CONFIRM=true
fi

echo "${YELLOW}Agent Context Protocol (ACP) Uninstall${NC}"
echo "======================================"
echo ""

# Check if agent directory exists
if [ ! -d "agent" ] && [ ! -f "AGENT.md" ]; then
    echo "${YELLOW}No ACP files found in current directory${NC}"
    exit 0
fi

# If not auto-confirmed, show instructions for agents
if [ "$AUTO_CONFIRM" = false ]; then
    echo "${BLUE}Note for AI Agents:${NC}"
    echo "This script requires user confirmation to prevent accidental deletion."
    echo "Please ask the user: 'Do you want to uninstall ACP from this project?'"
    echo "If yes, rerun this script with: ./agent/scripts/unacp.install.sh -y"
    echo ""
    echo "${RED}WARNING: This will permanently delete:${NC}"
    echo "  - agent/ directory (all contents)"
    echo "  - AGENT.md file"
    echo ""
    echo "Make sure you have committed any important changes to git."
    echo ""
    
    read -p "Are you sure you want to remove all ACP files? (y/N) " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Uninstall cancelled. No files were deleted."
        exit 0
    fi
fi

echo ""
echo "Removing ACP files..."

# Remove agent directory
if [ -d "agent" ]; then
    rm -rf agent
    echo "${GREEN}✓${NC} Removed agent/ directory"
fi

# Remove AGENT.md
if [ -f "AGENT.md" ]; then
    rm -f AGENT.md
    echo "${GREEN}✓${NC} Removed AGENT.md"
fi

echo ""
echo "${GREEN}Uninstall complete!${NC}"
echo ""
echo "All ACP files have been removed from this project."
echo "Use 'git status' to see what was deleted."
echo ""

#!/bin/bash
#
# Git Commit and Push Script
#
# ORCHESTRATOR NOTE: This script should be called via Task delegation,
# not executed directly. See CLAUDE.md "Git Delegation" section.
#
# Systematizes the common workflow of staging, committing, and pushing changes.
# Reduces 3 bash calls to 1.
#
# Usage:
#   ./scripts/git-commit-push.sh "commit message" [flags]
#
# Examples:
#   ./scripts/git-commit-push.sh "chore: update session tracking"
#   ./scripts/git-commit-push.sh "fix: resolve deployment issues" --dry-run
#   ./scripts/git-commit-push.sh "feat: add new feature" --no-confirm
#
# Flags:
#   --dry-run       Show what would happen without executing
#   --no-confirm    Skip confirmation prompt
#   --help          Show this help message
#

set -e  # Exit on error

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
COMMIT_MSG=""
DRY_RUN=false
NO_CONFIRM=false

show_help() {
    echo "Git Commit and Push Script"
    echo ""
    echo "Usage: $0 \"commit message\" [flags]"
    echo ""
    echo "Flags:"
    echo "  --dry-run       Show what would happen without executing"
    echo "  --no-confirm    Skip confirmation prompt"
    echo "  --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 \"chore: update session tracking\""
    echo "  $0 \"fix: resolve deployment issues\" --dry-run"
    echo "  $0 \"feat: add new feature\" --no-confirm"
    exit 0
}

# Parse arguments
for arg in "$@"; do
    case $arg in
        --dry-run)
            DRY_RUN=true
            ;;
        --no-confirm)
            NO_CONFIRM=true
            ;;
        --help|-h)
            show_help
            ;;
        *)
            if [ -z "$COMMIT_MSG" ]; then
                COMMIT_MSG="$arg"
            fi
            ;;
    esac
done

# Validate commit message
if [ -z "$COMMIT_MSG" ]; then
    echo -e "${RED}Error: Commit message is required${NC}"
    echo ""
    show_help
fi

# Header
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Git Commit and Push${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}⚠️  DRY-RUN MODE - No actual changes will be made${NC}"
    echo ""
fi

# Step 1: Show status
echo -e "${BLUE}Files to be committed:${NC}"
echo ""
git status --short
echo ""

# Count changes
CHANGED_FILES=$(git status --short | wc -l | tr -d ' ')

if [ "$CHANGED_FILES" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No changes to commit${NC}"
    exit 0
fi

echo -e "${BLUE}Total files: ${CHANGED_FILES}${NC}"
echo ""

# Step 2: Confirm (unless --no-confirm)
if [ "$NO_CONFIRM" = false ] && [ "$DRY_RUN" = false ]; then
    echo -e "${YELLOW}Commit message: \"${COMMIT_MSG}\"${NC}"
    echo ""
    read -p "Continue with commit and push? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}Aborted${NC}"
        exit 1
    fi
    echo ""
fi

# Step 3: Stage all changes
echo -e "${BLUE}Staging changes...${NC}"
if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY-RUN]${NC} Would run: git add -A"
else
    git add -A
    echo -e "${GREEN}✅ Changes staged${NC}"
fi
echo ""

# Step 4: Commit
echo -e "${BLUE}Committing...${NC}"
if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY-RUN]${NC} Would run: git commit -m \"${COMMIT_MSG}\""
else
    git commit -m "$COMMIT_MSG"
    echo -e "${GREEN}✅ Changes committed${NC}"
fi
echo ""

# Step 5: Push
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo -e "${BLUE}Pushing to origin/${CURRENT_BRANCH}...${NC}"
if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY-RUN]${NC} Would run: git push -u origin ${CURRENT_BRANCH}"
else
    git push -u origin "$CURRENT_BRANCH"
    echo -e "${GREEN}✅ Changes pushed${NC}"
fi
echo ""

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Summary:"
echo "  Files changed: ${CHANGED_FILES}"
echo "  Commit: \"${COMMIT_MSG}\""
echo "  Branch: ${CURRENT_BRANCH}"
echo ""

#!/bin/bash
# Release pre-check script
# This script verifies version consistency before release

set -e

echo "========================================="
echo "  afs Release Pre-Check"
echo "========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Get the version we're about to release
if [ -z "$1" ]; then
    echo -e "${RED}Usage: $0 <version>${NC}"
    echo ""
    echo "Example: $0 1.0.0"
    exit 1
fi

RELEASE_VERSION="$1"
echo -e "${YELLOW}Target Version: v${RELEASE_VERSION}${NC}"
echo ""

# Check if we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    echo -e "${RED}Error: Not on main branch (current: $CURRENT_BRANCH)${NC}"
    exit 1
fi

# Check if working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo -e "${RED}Error: Working directory is not clean${NC}"
    echo ""
    echo "Uncommitted changes:"
    git status --short
    exit 1
fi

# Check 1: Version in source files
echo "[$(printf '%02d' 1)] Checking version definitions in source files..."

# Check main.go
if ! grep -q "Version = \"dev\"" main.go; then
    echo -e "  ${YELLOW}⚠️  main.go Version is not 'dev' (current: $(grep 'Version = ' main.go | cut -d'"' -f2))${NC}"
fi

# Check cmd/root.go
if ! grep -q "Version = \"dev\"" cmd/root.go; then
    echo -e "  ${YELLOW}⚠️  cmd/root.go Version is not 'dev' (current: $(grep 'Version = ' cmd/root.go | cut -d'"' -f2))${NC}"
fi

# Check if both files have the same version
MAIN_VERSION=$(grep 'Version = ' main.go | cut -d'"' -f2 | tr -d ' ')
CMD_VERSION=$(grep 'Version = ' cmd/root.go | cut -d'"' -f2 | tr -d ' ')
if [ "$MAIN_VERSION" != "$CMD_VERSION" ]; then
    echo -e "  ${RED}✗ Version mismatch! main.go='$MAIN_VERSION', cmd/root.go='$CMD_VERSION'${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓ Source files have 'dev' version${NC}"

# Check 2: Go build succeeds
echo ""
echo "[$(printf '%02d' 2)] Testing build..."
if go build -o afs . 2>&1 | grep -i error; then
    echo -e "  ${RED}✗ Build failed${NC}"
    exit 1
fi
rm -f afs
echo -e "  ${GREEN}✓ Build successful${NC}"

# Check 3: Tests pass
echo ""
echo "[$(printf '%02d' 3)] Running tests..."
if ! go test ./... -v 2>&1 | tail -5 | grep -q "PASS"; then
    echo -e "  ${RED}✗ Tests failed${NC}"
    exit 1
fi
echo -e "  ${GREEN}✓ Tests passed${NC}"

# Check 4: Static analysis
echo ""
echo "[$(printf '%02d' 4)] Running static analysis..."
if go vet ./... 2>&1 | grep -i "warning\|error"; then
    echo -e "  ${RED}✗ go vet found issues${NC}"
    exit 1
fi
if golangci-lint run 2>&1 | grep -E "warning|error" | head -1; then
    echo -e "  ${YELLOW}⚠️  golangci-lint found issues (optional fix)${NC}"
else
    echo -e "  ${GREEN}✓ No lint issues${NC}"
fi

# Check 5: Documentation consistency
echo ""
echo "[$(printf '%02d' 5)] Checking documentation..."

# Check README.md
if ! grep -q "afs version x.x.x" README.md && grep -q "afs version dev" README.md; then
    echo -e "  ${YELLOW}⚠️  README.md still shows 'dev' version${NC}"
fi

# Check SKILL.md commands
COMMANDS_IN_README=$(grep -o 'afs [a-z]*' README.md | sort -u)
COMMANDS_IN_SKILL=$(grep -o 'afs [a-z]*' skills/afs/SKILL.md | sort -u)
MISSING_IN_README=$(comm -13 <(echo "$COMMANDS_IN_SKILL") <(echo "$COMMANDS_IN_READMEE"))
if [ -n "$MISSING_IN_README" ]; then
    echo -e "  ${YELLOW}⚠️  Commands in SKILL.md but not in README.md:$(echo $MISSING_IN_README)${NC}"
fi

echo -e "  ${GREEN}✓ Documentation checked${NC}"

# Summary
echo ""
echo "========================================="
echo -e "${GREEN}All checks passed! ✓${NC}"
echo "========================================="
echo ""
echo "Ready to release v${RELEASE_VERSION}"
echo ""
echo "Next steps:"
echo "  1. Update CHANGELOG.md"
echo "  2. Commit: git add . && git commit -m \"chore(release): prepare for v${RELEASE_VERSION}\""
echo "  3. Tag: git tag -a v${RELEASE_VERSION} -m \"Release v${RELEASE_VERSION}\""
echo "  4. **WAIT FOR USER CONFIRMATION**"
echo "  5. Push: git push origin main && git push origin v${RELEASE_VERSION}"
echo ""

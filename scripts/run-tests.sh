#!/bin/bash
set -e

echo "Task Orchestrator Test Suite"
echo "============================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Run unit tests
echo -e "${YELLOW}Running unit tests...${NC}"
go test -v -race -coverprofile=coverage.out ./internal/...
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Unit tests passed${NC}"
else
    echo -e "${RED}✗ Unit tests failed${NC}"
    exit 1
fi
echo ""

# Run integration tests
echo -e "${YELLOW}Running integration tests...${NC}"
go test -v -race ./test/integration/...
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Integration tests passed${NC}"
else
    echo -e "${RED}✗ Integration tests failed${NC}"
    exit 1
fi
echo ""

# Generate coverage report
echo -e "${YELLOW}Generating coverage report...${NC}"
coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo -e "${GREEN}Total coverage: ${coverage}${NC}"

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
echo -e "${GREEN}Coverage report: coverage.html${NC}"
echo ""

echo -e "${GREEN}All tests passed!${NC}"

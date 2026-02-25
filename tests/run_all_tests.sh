#!/bin/bash

################################################################################
# CHOP - Test Runner
#
# Executes all tests with proper configuration and reporting
# Usage: ./tests/run_all_tests.sh [OPTIONS]
# Options:
#   -v, --verbose     Verbose output
#   -m, --markers     Run specific marker (security, unit, integration, etc)
#   -k, --keyword     Run tests matching keyword
#   --coverage        Generate coverage report
#   --html           Generate HTML report
################################################################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="$SCRIPT_DIR/run_all_tests.log"

# Truncate log on each run
: > "$LOG_FILE"

# Redirect all output to log file (and stdout)
exec > >(tee -a "$LOG_FILE") 2>&1

# Load .env file if it exists, otherwise warn the user
if [ -f "$PROJECT_ROOT/tests/.env" ]; then
    echo -e "${BLUE}ℹ️ Sourcing environment variables from tests/.env${NC}"
    export $(grep -v '^#' "$PROJECT_ROOT/tests/.env" | xargs)
else
    echo -e "${YELLOW}⚠️ No tests/.env file found! Using default variables or system environment vars.${NC}"
    # Fallback to sensible defaults for local docker
    export API_URL=${API_URL:-"http://localhost:63000/api"}
    export TEST_API_URL=${TEST_API_URL:-"http://localhost:63000/api"}
    export DB_HOST=${DB_HOST:-"localhost"}
    export DB_PORT=${DB_PORT:-"65432"}
    export DB_USER=${DB_USER:-"test_user"}
    export DB_PASSWORD=${DB_PASSWORD:-"test_password"}
    export DB_NAME=${DB_NAME:-"contracts_test"}
fi

TEST_COMPOSE_FILE="$PROJECT_ROOT/tests/docker-compose.test.yml"

cleanup_docker() {
    docker compose -f "$TEST_COMPOSE_FILE" down -v --remove-orphans
}
trap cleanup_docker EXIT

# Parse arguments
VERBOSE=""
MARKER=""
KEYWORD=""
COVERAGE=""
HTML_REPORT=""

show_help() {
    cat << EOF
CHOP Test Runner - Run all tests with proper configuration

USAGE:
    ./run_all_tests.sh [OPTIONS]

OPTIONS:
    -h, --help          Show this help message
    -v, --verbose       Verbose output (very verbose)
    -m, --markers       Run tests with specific marker (e.g., security, login_blocking)
    -k, --keyword       Run tests matching keyword (e.g., blocking, jwt)
    --coverage          Generate coverage report (HTML and terminal)
    --html              Generate HTML test report

EXAMPLES:
    ./run_all_tests.sh                          # Run all tests
    ./run_all_tests.sh -v                       # Verbose output
    ./run_all_tests.sh -m security              # Security tests only
    ./run_all_tests.sh -k blocking              # Tests matching "blocking"
    ./run_all_tests.sh --coverage               # With coverage report
    ./run_all_tests.sh -v -m security --coverage  # Combined options

EOF
}

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE="-vv"
            shift
            ;;
        -m|--markers)
            MARKER="-m $2"
            shift 2
            ;;
        -k|--keyword)
            KEYWORD="-k $2"
            shift 2
            ;;
        --coverage)
            COVERAGE="--cov=backend --cov-report=html --cov-report=term-missing"
            shift
            ;;
        --html)
            HTML_REPORT="--html=tests/report.html --self-contained-html"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run with -h or --help for usage information"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║${NC}         CHOP - Comprehensive Test Suite Runner            ${BLUE}║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Create/use project-root venv
VENV_DIR="$PROJECT_ROOT/.venv"
PYTHON_BIN=""
if command -v python >/dev/null 2>&1; then
    PYTHON_BIN="python"
elif command -v python3 >/dev/null 2>&1; then
    PYTHON_BIN="python3"
else
    echo -e "${RED}❌ Python not found on PATH${NC}"
    exit 1
fi
if [ ! -d "$VENV_DIR" ]; then
    "$PYTHON_BIN" -m venv "$VENV_DIR"
fi
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"

# Check if pytest is installed
if ! command -v pytest &> /dev/null; then
    echo -e "${RED}❌ pytest not found. Installing dependencies...${NC}"
    cd "$PROJECT_ROOT"
    "$PYTHON_BIN" -m pip install -q -r tests/requirements.txt
fi

echo -e "${YELLOW}📋 Test Configuration:${NC}"
echo "   Project Root: $PROJECT_ROOT"
echo "   Test Directory: $SCRIPT_DIR"
echo "   Verbose: ${VERBOSE:- no}"
echo "   Marker Filter: ${MARKER:- none}"
echo "   Keyword Filter: ${KEYWORD:- none}"
echo "   Coverage: ${COVERAGE:- no}"
echo ""

# Change to project root
cd "$PROJECT_ROOT"

# Start test containers
echo -e "${BLUE}🐳 Starting test containers...${NC}"
docker compose -f "$TEST_COMPOSE_FILE" up -d --build postgres_test backend_test frontend_test

# Wait for backend health before running tests
echo -e "${BLUE}⏳ Waiting for backend healthcheck...${NC}"
for _ in {1..30}; do
    backend_status=$(docker inspect -f '{{.State.Health.Status}}' chop_test_backend 2>/dev/null || true)
    if [ "$backend_status" = "healthy" ]; then
        echo -e "${GREEN}✅ Backend is healthy${NC}"
        break
    fi
    sleep 2
done

if [ "$backend_status" != "healthy" ]; then
    echo -e "${RED}❌ Backend did not become healthy in time${NC}"
    exit 1
fi

# Run pytest with configuration
echo -e "${BLUE}🚀 Running test suite...${NC}"
echo ""

"$PYTHON_BIN" -m pytest \
    -v \
    $VERBOSE \
    $MARKER \
    $KEYWORD \
    $COVERAGE \
    $HTML_REPORT \
    --tb=short \
    --strict-markers \
    --disable-warnings \
    -c "$SCRIPT_DIR/pytest.ini" \
    tests/

TEST_EXIT_CODE=$?

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
else
    echo -e "${RED}❌ Some tests failed (exit code: $TEST_EXIT_CODE)${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
fi

if [ -n "$COVERAGE" ]; then
    echo ""
    echo -e "${YELLOW}📊 Coverage report generated in htmlcov/index.html${NC}"
fi

if [ -n "$HTML_REPORT" ]; then
    echo -e "${YELLOW}📊 HTML report generated in tests/report.html${NC}"
fi

echo ""
if [ -n "${VIRTUAL_ENV:-}" ]; then
    deactivate
fi
exit $TEST_EXIT_CODE

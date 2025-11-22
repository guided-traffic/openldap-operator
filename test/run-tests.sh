#!/bin/bash

# Test runner script for OpenLDAP Operator integration tests
# This script sets up the test environment and runs all tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEST_DIR="${PROJECT_ROOT}/test"
DOCKER_COMPOSE_FILE="${TEST_DIR}/docker-compose.yml"

# Default values
SKIP_DOCKER=false
SKIP_UNIT=false
SKIP_INTEGRATION=false
VERBOSE=false
COVERAGE=false
LDAP_HOST="localhost"
LDAP_PORT="389"
DOCKER_COMPOSE_CMD="docker-compose"  # Default, will be detected in check_prerequisites

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to print usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
    -h, --help              Show this help message
    -v, --verbose           Enable verbose output
    -c, --coverage          Generate detailed coverage reports
    --skip-docker          Skip Docker setup (use existing LDAP server)
    --skip-unit            Skip unit tests
    --skip-integration     Skip integration tests
    --ldap-host HOST       LDAP server host (default: localhost)
    --ldap-port PORT       LDAP server port (default: 389)

Environment Variables:
    LDAP_HOST              LDAP server host
    LDAP_PORT              LDAP server port
    LDAP_BIND_DN           LDAP bind DN (default: cn=admin,dc=example,dc=com)
    LDAP_BASE_DN           LDAP base DN (default: dc=example,dc=com)
    LDAP_PASSWORD          LDAP bind password (default: admin123)

Examples:
    $0                          # Run all tests with Docker LDAP server
    $0 --coverage              # Run all tests with detailed coverage reports
    $0 --skip-docker           # Run tests against existing LDAP server
    $0 --skip-unit             # Run only integration tests
    $0 -c -v                   # Run with coverage and verbose output
    $0 --ldap-host ldap.example.com --ldap-port 636

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        --skip-docker)
            SKIP_DOCKER=true
            shift
            ;;
        --skip-unit)
            SKIP_UNIT=true
            shift
            ;;
        --skip-integration)
            SKIP_INTEGRATION=true
            shift
            ;;
        --ldap-host)
            LDAP_HOST="$2"
            shift 2
            ;;
        --ldap-port)
            LDAP_PORT="$2"
            shift 2
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."

    if ! command_exists go; then
        print_error "Go is not installed"
        exit 1
    fi

    if [[ "$SKIP_DOCKER" == "false" ]]; then
        if ! command_exists docker; then
            print_error "Docker is not installed"
            exit 1
        fi

        # Check for Docker Compose V2 (docker compose) or V1 (docker-compose)
        if docker compose version >/dev/null 2>&1; then
            DOCKER_COMPOSE_CMD="docker compose"
        elif command_exists docker-compose; then
            DOCKER_COMPOSE_CMD="docker-compose"
        else
            print_error "Docker Compose is not installed (tried 'docker compose' and 'docker-compose')"
            exit 1
        fi
        print_status "Using Docker Compose: $DOCKER_COMPOSE_CMD"
    fi

    print_status "Prerequisites check passed"
}

# Function to setup Docker environment
setup_docker() {
    if [[ "$SKIP_DOCKER" == "true" ]]; then
        print_status "Skipping Docker setup"
        return
    fi

    print_status "Setting up Docker environment..."

    cd "$TEST_DIR"

    # Stop any existing containers
    $DOCKER_COMPOSE_CMD -f "$DOCKER_COMPOSE_FILE" down --volumes --remove-orphans 2>/dev/null || true

    # Start the services
    print_status "Starting OpenLDAP server..."
    $DOCKER_COMPOSE_CMD -f "$DOCKER_COMPOSE_FILE" up -d openldap

    # Wait for LDAP server to be ready
    print_status "Waiting for LDAP server to be ready..."
    local max_attempts=30
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if $DOCKER_COMPOSE_CMD -f "$DOCKER_COMPOSE_FILE" exec -T openldap ldapsearch -x -H ldap://localhost:1389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w "admin123" >/dev/null 2>&1; then
            print_status "LDAP server is ready"
            break
        fi

        if [[ $attempt -eq $max_attempts ]]; then
            print_error "LDAP server failed to start within timeout"
            $DOCKER_COMPOSE_CMD -f "$DOCKER_COMPOSE_FILE" logs openldap
            exit 1
        fi

        print_status "Waiting for LDAP server... (attempt $attempt/$max_attempts)"
        sleep 5
        ((attempt++))
    done

    # Initialize test data
    print_status "Initializing test data..."
    if [[ -f "${TEST_DIR}/ldap-config/init.ldif" ]]; then
        $DOCKER_COMPOSE_CMD -f "$DOCKER_COMPOSE_FILE" exec -T openldap ldapadd -x -H ldap://localhost:1389 -D "cn=admin,dc=example,dc=com" -w "admin123" -f /ldifs/init.ldif 2>/dev/null || true
    fi
}

# Function to run unit tests
run_unit_tests() {
    if [[ "$SKIP_UNIT" == "true" ]]; then
        print_status "Skipping unit tests"
        return
    fi

    print_status "Running unit tests..."
    cd "$PROJECT_ROOT"

    if [[ "$COVERAGE" == "true" ]]; then
        print_status "Running unit tests with detailed coverage..."

        # Create coverage directory
        mkdir -p coverage

        # Run unit tests with coverage for each package
        print_status "Testing API types with coverage..."
        go test -v ./api/v1/... -coverprofile coverage/api.out -covermode=atomic

        print_status "Testing LDAP client with coverage..."
        go test -v ./internal/ldap/... -coverprofile coverage/ldap.out -covermode=atomic

        print_status "Testing controllers with coverage..."
        go test -v ./internal/controller/... -coverprofile coverage/controller.out -covermode=atomic

        # Combine coverage files
        print_status "Combining coverage reports..."
        echo "mode: atomic" > coverage/unit.out
        grep -h -v "^mode:" coverage/api.out coverage/ldap.out coverage/controller.out >> coverage/unit.out 2>/dev/null || true

        # Generate HTML and text reports
        go tool cover -html=coverage/unit.out -o coverage/unit-coverage.html
        go tool cover -func=coverage/unit.out | tee coverage/unit-coverage.txt

        print_status "Unit test coverage report generated: coverage/unit-coverage.html"

        # Print coverage summary
        echo ""
        print_status "Unit Test Coverage Summary:"
        tail -1 coverage/unit-coverage.txt
        echo ""
    else
        # Run unit tests for API types
        go test -v ./api/v1/... -cover

        # Run unit tests for LDAP client
        go test -v ./internal/ldap/... -cover

        # Run unit tests for controllers
        go test -v ./internal/controller/... -cover
    fi

    print_status "Unit tests completed"
}

# Function to run integration tests
run_integration_tests() {
    if [[ "$SKIP_INTEGRATION" == "true" ]]; then
        print_status "Skipping integration tests"
        return
    fi

    print_status "Running integration tests..."
    cd "$PROJECT_ROOT"

    # Set environment variables for integration tests
    export LDAP_HOST="$LDAP_HOST"
    export LDAP_PORT="$LDAP_PORT"
    export LDAP_BIND_DN="${LDAP_BIND_DN:-cn=admin,dc=example,dc=com}"
    export LDAP_BASE_DN="${LDAP_BASE_DN:-dc=example,dc=com}"
    export LDAP_PASSWORD="${LDAP_PASSWORD:-admin123}"

    if [[ "$VERBOSE" == "true" ]]; then
        print_status "Using LDAP configuration:"
        print_status "  Host: $LDAP_HOST"
        print_status "  Port: $LDAP_PORT"
        print_status "  Bind DN: $LDAP_BIND_DN"
        print_status "  Base DN: $LDAP_BASE_DN"
    fi

    # Run the integration test binary
    if [[ -f "${TEST_DIR}/integration/runner/main.go" ]]; then
        cd "${TEST_DIR}/integration/runner"
        go run main.go \
            --ldap-host "$LDAP_HOST" \
            --ldap-port "$LDAP_PORT" \
            --ldap-bind-dn "$LDAP_BIND_DN" \
            --ldap-base-dn "$LDAP_BASE_DN" \
            --ldap-password "$LDAP_PASSWORD"
    else
        print_warning "Integration test binary not found, skipping"
    fi

    print_status "Integration tests completed"
}

# Function to cleanup
cleanup() {
    if [[ "$SKIP_DOCKER" == "false" ]]; then
        print_status "Cleaning up Docker environment..."
        cd "$TEST_DIR"
        $DOCKER_COMPOSE_CMD -f "$DOCKER_COMPOSE_FILE" down --volumes --remove-orphans 2>/dev/null || true
    fi
}

# Function to generate final coverage report
generate_final_coverage_report() {
    if [[ "$COVERAGE" != "true" ]]; then
        return
    fi

    print_status "Generating final coverage report..."
    cd "$PROJECT_ROOT"

    # Create reports directory
    mkdir -p coverage/reports

    # Check if we have unit test coverage
    if [[ -f "coverage/unit.out" ]]; then
        # Generate comprehensive coverage report
        print_status "Creating comprehensive coverage analysis..."

        cat > coverage/reports/summary.txt << EOF
=== OpenLDAP Operator Test Coverage Report ===
Generated: $(date)

=== Unit Test Coverage Summary ===

EOF

        # Add unit test coverage summary
        if [[ -f "coverage/unit-coverage.txt" ]]; then
            echo "Overall Unit Test Coverage:" >> coverage/reports/summary.txt
            tail -1 coverage/unit-coverage.txt >> coverage/reports/summary.txt
            echo "" >> coverage/reports/summary.txt
            echo "Package-by-Package Coverage:" >> coverage/reports/summary.txt
            cat coverage/unit-coverage.txt >> coverage/reports/summary.txt
        fi

        cat >> coverage/reports/summary.txt << EOF

=== Coverage Thresholds ===

Recommended coverage targets:
- Overall: >= 80%
- API types: >= 90% (validation logic is critical)
- LDAP client: >= 85% (core functionality)
- Controllers: >= 75% (complex reconciliation logic)

=== Coverage Files ===

- HTML Report: coverage/unit-coverage.html
- Text Report: coverage/unit-coverage.txt
- Raw Data: coverage/unit.out

Open coverage/unit-coverage.html in your browser for detailed analysis.

EOF

        print_status "Coverage analysis completed!"
        echo ""
        print_status "=== COVERAGE SUMMARY ==="
        if [[ -f "coverage/unit-coverage.txt" ]]; then
            echo "Unit Test Coverage: $(tail -1 coverage/unit-coverage.txt | awk '{print $3}')"
        fi
        echo ""
        print_status "Detailed reports available:"
        print_status "  - HTML: coverage/unit-coverage.html"
        print_status "  - Summary: coverage/reports/summary.txt"
        echo ""

        # Display coverage summary
        if [[ -f "coverage/reports/summary.txt" ]] && [[ "$VERBOSE" == "true" ]]; then
            cat coverage/reports/summary.txt
        fi
    else
        print_warning "No coverage data found to generate final report"
    fi
}

# Function to run all tests
run_all_tests() {
    local exit_code=0

    print_status "Starting OpenLDAP Operator test suite..."

    if [[ "$COVERAGE" == "true" ]]; then
        print_status "Coverage reporting enabled"
    fi

    # Setup
    check_prerequisites
    setup_docker

    # Run tests
    if ! run_unit_tests; then
        exit_code=1
    fi

    if ! run_integration_tests; then
        exit_code=1
    fi

    # Generate final coverage report if enabled
    if [[ "$COVERAGE" == "true" ]]; then
        generate_final_coverage_report
    fi

    # Cleanup
    cleanup

    if [[ $exit_code -eq 0 ]]; then
        print_status "All tests passed!"
        if [[ "$COVERAGE" == "true" ]]; then
            print_status "Coverage reports available in coverage/ directory"
        fi
    else
        print_error "Some tests failed!"
    fi

    return $exit_code
}

# Main execution
main() {
    # Set up signal handlers for cleanup
    trap cleanup EXIT
    trap cleanup INT
    trap cleanup TERM

    # Run tests
    run_all_tests
    exit $?
}

# Run main function
main "$@"

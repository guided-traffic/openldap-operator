#!/bin/bash

# Coverage Analysis Script for OpenLDAP Operator
# This script provides detailed coverage analysis and recommendations

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COVERAGE_DIR="${PROJECT_ROOT}/coverage"

# Thresholds
OVERALL_THRESHOLD=80
API_THRESHOLD=90
LDAP_THRESHOLD=85
CONTROLLER_THRESHOLD=75

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

print_highlight() {
    echo -e "${BLUE}[HIGHLIGHT]${NC} $1"
}

# Function to extract coverage percentage from a line
extract_coverage() {
    echo "$1" | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//' || echo "0.0"
}

# Function to compare coverage with threshold
check_threshold() {
    local coverage=$1
    local threshold=$2
    local package_name=$3

    # Convert percentage to number for comparison
    local coverage_num=$(echo "$coverage" | sed 's/%//')

    if (( $(echo "$coverage_num >= $threshold" | bc -l) )); then
        print_status "$package_name: $coverage (✓ above threshold $threshold%)"
        return 0
    else
        print_warning "$package_name: $coverage (⚠ below threshold $threshold%)"
        return 1
    fi
}

# Function to analyze coverage by package
analyze_package_coverage() {
    local coverage_file=$1

    if [[ ! -f "$coverage_file" ]]; then
        print_error "Coverage file not found: $coverage_file"
        return 1
    fi

    print_highlight "Package Coverage Analysis"
    echo "----------------------------------------"

    local failed_packages=0
    local total_packages=0

    # Parse coverage by package
    while IFS= read -r line; do
        if [[ $line == *"/api/v1"* ]]; then
            local coverage=$(extract_coverage "$line")
            if ! check_threshold "$coverage%" $API_THRESHOLD "API Types (api/v1)"; then
                ((failed_packages++))
            fi
            ((total_packages++))
        elif [[ $line == *"/internal/ldap"* ]]; then
            local coverage=$(extract_coverage "$line")
            if ! check_threshold "$coverage%" $LDAP_THRESHOLD "LDAP Client (internal/ldap)"; then
                ((failed_packages++))
            fi
            ((total_packages++))
        elif [[ $line == *"/internal/controller"* ]]; then
            local coverage=$(extract_coverage "$line")
            if ! check_threshold "$coverage%" $CONTROLLER_THRESHOLD "Controllers (internal/controller)"; then
                ((failed_packages++))
            fi
            ((total_packages++))
        fi
    done < <(grep -E "(api/v1|internal/ldap|internal/controller)" "$coverage_file" | grep -v "total:")

    echo ""
    print_highlight "Package Threshold Summary"
    echo "----------------------------------------"
    echo "Packages meeting thresholds: $((total_packages - failed_packages))/$total_packages"

    if [[ $failed_packages -gt 0 ]]; then
        print_warning "$failed_packages packages below recommended thresholds"
        return 1
    else
        print_status "All packages meet recommended coverage thresholds!"
        return 0
    fi
}

# Function to find uncovered code
find_uncovered_code() {
    local coverage_file=$1

    print_highlight "Identifying Uncovered Code (excluding auto-generated files)"
    echo "----------------------------------------"

    # Create a temporary coverage profile for analysis
    local temp_profile="/tmp/coverage_analysis.out"
    cp "$coverage_file" "$temp_profile"

    # Generate line-by-line coverage data, excluding auto-generated files
    go tool cover -func="$temp_profile" | grep -v "zz_generated.deepcopy.go" | grep "0.0%" | head -20 | while read -r line; do
        local func_name=$(echo "$line" | awk '{print $2}')
        local file_name=$(echo "$line" | awk '{print $1}')
        echo "  ⚠ $file_name: $func_name (0% coverage)"
    done

    rm -f "$temp_profile"
}

# Function to generate coverage recommendations
generate_recommendations() {
    local overall_coverage=$1
    local failed_packages=$2

    print_highlight "Coverage Improvement Recommendations"
    echo "----------------------------------------"

    if (( $(echo "$overall_coverage < $OVERALL_THRESHOLD" | bc -l) )); then
        echo "📈 Overall coverage ($overall_coverage%) is below target ($OVERALL_THRESHOLD%)"
        echo "   • Focus on testing core business logic"
        echo "   • Add integration tests for end-to-end workflows"
        echo "   • Consider property-based testing for complex functions"
        echo ""
    fi

    echo "🎯 Specific Recommendations by Package:"
    echo ""
    echo "API Types (target: $API_THRESHOLD%+):"
    echo "  • Test all validation functions"
    echo "  • Test custom marshaling/unmarshaling logic"
    echo "  • Test default value assignment"
    echo "  • Test edge cases in spec validation"
    echo ""

    echo "LDAP Client (target: $LDAP_THRESHOLD%+):"
    echo "  • Test error handling for connection failures"
    echo "  • Test all CRUD operations with different inputs"
    echo "  • Test connection retry logic"
    echo "  • Mock LDAP server responses for edge cases"
    echo ""

    echo "Controllers (target: $CONTROLLER_THRESHOLD%+):"
    echo "  • Test reconciliation logic for all resource states"
    echo "  • Test error conditions and retry scenarios"
    echo "  • Test status updates and condition handling"
    echo "  • Test finalizer logic and resource cleanup"
    echo ""

    echo "🔧 Testing Best Practices:"
    echo "  • Use table-driven tests for multiple scenarios"
    echo "  • Mock external dependencies (LDAP server, Kubernetes API)"
    echo "  • Test both success and failure paths"
    echo "  • Include boundary condition tests"
    echo "  • Add benchmarks for performance-critical code"
    echo ""

    echo "📊 Tools to Consider:"
    echo "  • gocov for alternative coverage reporting"
    echo "  • gocov-html for enhanced HTML reports"
    echo "  • gomock for generating mocks"
    echo "  • testify for assertion helpers"
}

# Function to create coverage badge data
create_coverage_badge() {
    local overall_coverage=$1
    local badge_file="${COVERAGE_DIR}/badge.json"

    # Determine badge color based on coverage
    local color="red"
    if (( $(echo "$overall_coverage >= 80" | bc -l) )); then
        color="brightgreen"
    elif (( $(echo "$overall_coverage >= 60" | bc -l) )); then
        color="yellow"
    elif (( $(echo "$overall_coverage >= 40" | bc -l) )); then
        color="orange"
    fi

    cat > "$badge_file" << EOF
{
  "schemaVersion": 1,
  "label": "coverage",
  "message": "${overall_coverage}%",
  "color": "$color"
}
EOF

    print_status "Coverage badge data created: $badge_file"
}

# Main analysis function
main() {
    cd "$PROJECT_ROOT"

    print_highlight "OpenLDAP Operator Coverage Analysis"
    echo "========================================"
    echo ""

    # Check if coverage data exists
    if [[ ! -d "$COVERAGE_DIR" ]]; then
        print_error "Coverage directory not found. Run tests with coverage first:"
        print_error "  make test-coverage"
        print_error "  or"
        print_error "  ./test/run-tests.sh --coverage"
        exit 1
    fi

    local coverage_file="$COVERAGE_DIR/unit-coverage.txt"
    if [[ ! -f "$coverage_file" ]]; then
        coverage_file="$COVERAGE_DIR/coverage.txt"
    fi

    if [[ ! -f "$coverage_file" ]]; then
        print_error "Coverage report not found. Run tests with coverage first."
        exit 1
    fi

    # Extract overall coverage
    local overall_line=$(tail -1 "$coverage_file")
    local overall_coverage=$(extract_coverage "$overall_line")

    print_highlight "Overall Coverage: $overall_coverage%"
    echo ""

    # Check overall threshold
    local overall_status=0
    if ! check_threshold "$overall_coverage%" $OVERALL_THRESHOLD "Overall Project"; then
        overall_status=1
    fi
    echo ""

    # Analyze package coverage
    local package_status=0
    if ! analyze_package_coverage "$coverage_file"; then
        package_status=1
    fi
    echo ""

    # Find uncovered code
    if [[ -f "$COVERAGE_DIR/unit.out" ]]; then
        find_uncovered_code "$COVERAGE_DIR/unit.out"
        echo ""
    fi

    # Generate recommendations
    generate_recommendations "$overall_coverage" $((overall_status + package_status))

    # Create coverage badge
    create_coverage_badge "$overall_coverage"

    echo ""
    print_highlight "Analysis Complete"
    echo "========================================"

    # Exit with appropriate code
    if [[ $overall_status -eq 0 ]] && [[ $package_status -eq 0 ]]; then
        print_status "All coverage targets met! 🎉"
        exit 0
    else
        print_warning "Some coverage targets not met. See recommendations above."
        exit 1
    fi
}

# Check if bc is available for floating point arithmetic
if ! command -v bc >/dev/null 2>&1; then
    print_error "bc (basic calculator) is required for this script"
    print_error "Install with: brew install bc (macOS) or apt-get install bc (Ubuntu)"
    exit 1
fi

# Run main function
main "$@"

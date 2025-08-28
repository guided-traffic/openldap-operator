# Code Coverage Documentation

This document describes the comprehensive code coverage system implemented for the OpenLDAP Operator project.

## Overview

The project includes multiple layers of coverage analysis:

1. **Unit Test Coverage** - Tests for individual packages and functions
2. **Integration Test Coverage** - End-to-end testing with live LDAP servers
3. **Combined Coverage** - Overall project coverage metrics
4. **Coverage Analysis** - Detailed analysis with recommendations

## Coverage Targets

| Component | Target Coverage | Rationale |
|-----------|----------------|-----------|
| Overall Project | ≥ 80% | Industry standard for good coverage |
| API Types (`api/v1`) | ≥ 90% | Validation logic is critical |
| LDAP Client (`internal/ldap`) | ≥ 85% | Core functionality requires high confidence |
| Controllers (`internal/controller`) | ≥ 75% | Complex reconciliation logic, some paths hard to test |

## Running Coverage Tests

### Basic Coverage

```bash
# Run all tests with basic coverage
make test-coverage

# Run only unit tests with coverage
make test-unit-coverage
```

### Detailed Coverage Analysis

```bash
# Run comprehensive coverage analysis
make test-all-coverage

# Run test runner with coverage
./test/run-tests.sh --coverage

# Run detailed analysis on existing coverage data
make coverage-analysis
```

### Integration with Test Runner

The test runner script (`test/run-tests.sh`) supports coverage reporting:

```bash
# Run all tests with coverage
./test/run-tests.sh --coverage

# Run with coverage and verbose output
./test/run-tests.sh --coverage --verbose

# Skip Docker setup, run with coverage
./test/run-tests.sh --coverage --skip-docker
```

## Coverage Reports

The system generates multiple types of coverage reports:

### HTML Reports
- `coverage/unit-coverage.html` - Interactive HTML report for unit tests
- `coverage/coverage.html` - Combined coverage report

### Text Reports
- `coverage/unit-coverage.txt` - Text summary of unit test coverage
- `coverage/coverage.txt` - Combined coverage summary

### Analysis Reports
- `coverage/reports/summary.txt` - Comprehensive coverage summary
- `coverage/badge.json` - Coverage badge data for README

## Coverage Analysis Script

The `scripts/coverage-analysis.sh` script provides detailed analysis:

### Features

1. **Threshold Checking** - Compares coverage against targets
2. **Package Analysis** - Per-package coverage breakdown
3. **Uncovered Code Detection** - Identifies functions with 0% coverage
4. **Recommendations** - Specific suggestions for improvement
5. **Badge Generation** - Creates coverage badge data

### Usage

```bash
# Basic analysis
./scripts/coverage-analysis.sh

# Prerequisites check
# The script requires 'bc' for floating-point arithmetic:
# macOS: brew install bc
# Ubuntu: apt-get install bc
```

### Output Example

```
OpenLDAP Operator Coverage Analysis
========================================

Overall Coverage: 87.3%

[INFO] Overall Project: 87.3% (✓ above threshold 80%)

Package Coverage Analysis
----------------------------------------
[INFO] API Types (api/v1): 92.1% (✓ above threshold 90%)
[INFO] LDAP Client (internal/ldap): 88.7% (✓ above threshold 85%)
[WARN] Controllers (internal/controller): 72.4% (⚠ below threshold 75%)

Package Threshold Summary
----------------------------------------
Packages meeting thresholds: 2/3
[WARN] 1 packages below recommended thresholds
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Tests and Coverage
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Install dependencies
      run: sudo apt-get install -y bc

    - name: Run tests with coverage
      run: make test-all-coverage

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage/coverage.out
        flags: unittests
        name: codecov-umbrella
```

### Makefile Integration

Key make targets for CI/CD:

```bash
# Quick unit tests with coverage
make test-unit-coverage

# Full coverage analysis
make test-all-coverage

# Coverage analysis only (requires existing data)
make coverage-analysis
```

## Coverage Best Practices

### Writing Testable Code

1. **Dependency Injection** - Make external dependencies injectable
2. **Interface Abstraction** - Use interfaces for mockable components
3. **Small Functions** - Break complex logic into testable units
4. **Error Handling** - Ensure all error paths are testable

### Test Organization

```go
// Example: Table-driven tests for high coverage
func TestLDAPClient_CreateUser(t *testing.T) {
    tests := []struct {
        name      string
        userSpec  *v1.LDAPUserSpec
        mockSetup func(*MockLDAPConn)
        wantErr   bool
    }{
        {
            name: "valid user creation",
            userSpec: &v1.LDAPUserSpec{
                Username: "testuser",
                Email:    "test@example.com",
            },
            mockSetup: func(m *MockLDAPConn) {
                m.EXPECT().Add(gomock.Any()).Return(nil)
            },
            wantErr: false,
        },
        {
            name: "connection error",
            userSpec: &v1.LDAPUserSpec{
                Username: "testuser",
            },
            mockSetup: func(m *MockLDAPConn) {
                m.EXPECT().Add(gomock.Any()).Return(errors.New("connection failed"))
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Mocking External Dependencies

```go
//go:generate mockgen -source=client.go -destination=mocks/mock_client.go

type LDAPConnection interface {
    Bind(username, password string) error
    Add(addRequest *ldap.AddRequest) error
    Del(delRequest *ldap.DelRequest) error
    Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error)
}
```

## Coverage Exclusions

Some code paths are intentionally excluded from coverage requirements:

1. **Generated Code** - `zz_generated.deepcopy.go` files
2. **Main Functions** - Entry points and bootstrapping code
3. **Test Utilities** - Helper functions in test files
4. **Debug Code** - Logging and debugging statements

## Troubleshooting

### Common Issues

1. **Missing Coverage Data**
   ```bash
   # Solution: Run tests with coverage first
   make test-unit-coverage
   ```

2. **bc Command Not Found**
   ```bash
   # macOS
   brew install bc

   # Ubuntu/Debian
   sudo apt-get install bc
   ```

3. **Permission Denied on Scripts**
   ```bash
   chmod +x scripts/coverage-analysis.sh
   chmod +x test/run-tests.sh
   ```

### Coverage Not Updating

1. Clean previous coverage data:
   ```bash
   rm -rf coverage/
   make test-unit-coverage
   ```

2. Ensure tests are actually running:
   ```bash
   ./test/run-tests.sh --coverage --verbose
   ```

## Advanced Usage

### Custom Coverage Thresholds

Edit `scripts/coverage-analysis.sh` to adjust thresholds:

```bash
# Configuration
OVERALL_THRESHOLD=85    # Increase overall target
API_THRESHOLD=95        # Increase API target
LDAP_THRESHOLD=90       # Increase LDAP target
CONTROLLER_THRESHOLD=80 # Increase controller target
```

### Profile-Based Analysis

```bash
# Generate detailed profile
go test ./... -coverprofile=profile.out -covermode=atomic

# Analyze specific functions
go tool cover -func=profile.out | grep "MyFunction"

# HTML report for specific package
go test ./internal/ldap/... -coverprofile=ldap.out
go tool cover -html=ldap.out -o ldap-coverage.html
```

### Integration with External Tools

1. **SonarQube Integration**
   ```bash
   # Convert coverage for SonarQube
   gocover-cobertura < coverage/coverage.out > coverage.xml
   ```

2. **Codecov Integration**
   ```bash
   # Upload to Codecov
   bash <(curl -s https://codecov.io/bash) -f coverage/coverage.out
   ```

## Continuous Improvement

### Monthly Coverage Review

1. Run comprehensive coverage analysis
2. Identify packages below targets
3. Plan testing improvements
4. Update test coverage in sprint planning

### Coverage Trends

Track coverage over time:

```bash
# Generate historical coverage data
echo "$(date): $(make coverage-analysis | grep 'Overall Coverage' | awk '{print $3}')" >> coverage-history.txt
```

This coverage system ensures high-quality, well-tested code while providing clear visibility into test coverage across the entire OpenLDAP Operator project.

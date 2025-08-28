# OpenLDAP Operator Test Suite

This directory contains comprehensive tests for the OpenLDAP Operator, including unit tests, integration tests, and test utilities.

## Test Structure

```
test/
├── run-tests.sh              # Main test runner script
├── docker-compose.yml        # Docker Compose for test LDAP server
├── ldap-config/
│   └── init.ldif             # Initial LDAP test data
└── integration/
    ├── main.go               # Integration test runner
    └── integration_test.go   # Full integration test suite
```

## Test Types

### 1. Unit Tests

Unit tests are located alongside the source code and test individual components in isolation:

- **API Types Tests** (`api/v1/types_test.go`): Tests for custom resource validation and defaults
- **LDAP Client Tests** (`internal/ldap/client_test.go`): Tests for LDAP client functionality
- **Controller Tests** (`internal/controller/controller_test.go`): Tests for controller logic

### 2. Integration Tests

Integration tests verify the complete operator functionality with a real LDAP server:

- **LDAP Server Management**: Connection, health checks, error handling
- **User Management**: CRUD operations for LDAP users
- **Group Management**: CRUD operations for LDAP groups
- **Group Membership**: Adding/removing users from groups
- **Error Scenarios**: Invalid configurations, connection failures

### 3. End-to-End Tests

End-to-end tests use the full Kubernetes operator with a real LDAP server to test complete workflows.

## Running Tests

### Prerequisites

- Go 1.21+
- Docker and Docker Compose (for integration tests)
- Access to a Kubernetes cluster (for e2e tests)

### Quick Start

Run all tests with a temporary LDAP server:

```bash
./test/run-tests.sh
```

### Test Options

```bash
# Run only unit tests
./test/run-tests.sh --skip-integration

# Run only integration tests
./test/run-tests.sh --skip-unit

# Use an existing LDAP server
./test/run-tests.sh --skip-docker --ldap-host ldap.example.com --ldap-port 389

# Verbose output
./test/run-tests.sh --verbose
```

### Manual Test Execution

#### Unit Tests

```bash
# Run all unit tests
go test ./... -v

# Run specific package tests
go test ./api/v1/... -v
go test ./internal/ldap/... -v
go test ./internal/controller/... -v

# Run with coverage
go test ./... -cover
```

#### Integration Tests

1. Start the test LDAP server:
```bash
cd test
docker-compose up -d openldap
```

2. Wait for the server to be ready:
```bash
docker-compose exec openldap ldapsearch -x -H ldap://localhost:1389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w "admin123"
```

3. Run integration tests:
```bash
cd test/integration
go run main.go --ldap-host localhost --ldap-port 389
```

4. Clean up:
```bash
cd test
docker-compose down --volumes
```

## LDAP Test Server

The test suite uses a Bitnami OpenLDAP Docker image with the following configuration:

- **Host**: localhost
- **Port**: 389 (LDAP), 636 (LDAPS)
- **Bind DN**: `cn=admin,dc=example,dc=com`
- **Base DN**: `dc=example,dc=com`
- **Password**: `admin123`

### Initial Test Data

The server is initialized with:
- Organizational units: `ou=users,dc=example,dc=com` and `ou=groups,dc=example,dc=com`
- Test user: `uid=admin-test,ou=users,dc=example,dc=com`
- Test groups: `cn=administrators,ou=groups,dc=example,dc=com` and `cn=users,ou=groups,dc=example,dc=com`

### Management Interface

A phpLDAPadmin interface is available at http://localhost:8080 for manual inspection and debugging.

## Environment Variables

The following environment variables can be used to configure tests:

| Variable | Description | Default |
|----------|-------------|---------|
| `LDAP_HOST` | LDAP server hostname | `localhost` |
| `LDAP_PORT` | LDAP server port | `389` |
| `LDAP_BIND_DN` | LDAP bind DN | `cn=admin,dc=example,dc=com` |
| `LDAP_BASE_DN` | LDAP base DN | `dc=example,dc=com` |
| `LDAP_PASSWORD` | LDAP bind password | `admin123` |

## Test Scenarios

### LDAP Server Tests

- [x] Basic connection to LDAP server
- [x] TLS/LDAPS connection
- [x] Authentication with bind DN and password
- [x] Connection timeout handling
- [x] Health check validation
- [x] Invalid server configuration error handling

### User Management Tests

- [x] Create user with all attributes (uid, cn, sn, mail, etc.)
- [x] Create user with POSIX attributes (uidNumber, gidNumber, homeDirectory, loginShell)
- [x] Update user attributes
- [x] Delete user
- [x] Check user existence
- [x] Handle duplicate user creation
- [x] Validate user attribute constraints

### Group Management Tests

- [x] Create group (groupOfNames, groupOfUniqueNames, posixGroup)
- [x] Create group with description and gidNumber
- [x] Delete group
- [x] Check group existence
- [x] Handle duplicate group creation

### Group Membership Tests

- [x] Add user to group (different group types)
- [x] Remove user from group
- [x] List group members
- [x] Handle non-existent user/group references
- [x] Manage multiple users in groups

### Controller Tests

- [x] LDAPServer reconciliation
- [x] LDAPUser reconciliation
- [x] LDAPGroup reconciliation
- [x] Status updates and conditions
- [x] Finalizer handling
- [x] Error state management
- [x] Resource deletion

### Validation Tests

- [x] API type validation (required fields, formats)
- [x] LDAP DN construction
- [x] Username and group name validation
- [x] Email format validation
- [x] POSIX ID validation

## Benchmarks

Performance benchmarks are included for critical path operations:

```bash
go test -bench=. ./internal/ldap/...
go test -bench=. ./api/v1/...
```

## Troubleshooting

### Common Issues

1. **LDAP Server Connection Failed**
   - Verify Docker is running
   - Check if port 389 is available
   - Wait longer for server startup

2. **Test Data Conflicts**
   - Clean up Docker volumes: `docker-compose down --volumes`
   - Use unique test names with timestamps

3. **Permission Errors**
   - Ensure test user has sufficient LDAP permissions
   - Check LDAP server logs: `docker-compose logs openldap`

### Debug Mode

Enable verbose logging:

```bash
export LDAP_DEBUG=true
./test/run-tests.sh --verbose
```

### Manual LDAP Queries

Test LDAP connectivity manually:

```bash
# Basic search
ldapsearch -x -H ldap://localhost:389 -b "dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w "admin123"

# Search for users
ldapsearch -x -H ldap://localhost:389 -b "ou=users,dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w "admin123" "(objectClass=person)"

# Search for groups
ldapsearch -x -H ldap://localhost:389 -b "ou=groups,dc=example,dc=com" -D "cn=admin,dc=example,dc=com" -w "admin123" "(objectClass=group*)"
```

## Contributing

When adding new tests:

1. Add unit tests for new functionality
2. Update integration tests for new LDAP operations
3. Document test scenarios in this README
4. Ensure tests clean up after themselves
5. Use descriptive test names and error messages

## CI/CD Integration

The test suite is designed for integration with CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Run Tests
  run: |
    ./test/run-tests.sh
  env:
    LDAP_HOST: localhost
    LDAP_PORT: 389
```

For environments without Docker, tests can run against an external LDAP server:

```bash
./test/run-tests.sh --skip-docker --ldap-host $EXTERNAL_LDAP_HOST
```

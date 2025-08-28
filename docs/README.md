# OpenLDAP Operator Documentation

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Custom Resources](#custom-resources)
4. [Installation](#installation)
5. [Usage Examples](#usage-examples)
6. [Configuration](#configuration)
7. [Security](#security)
8. [Troubleshooting](#troubleshooting)
9. [Development](#development)

## Overview

The OpenLDAP Operator is a Kubernetes operator designed to manage external LDAP servers, users, groups, and Access Control Lists (ACLs). It provides a declarative way to manage LDAP resources using Kubernetes Custom Resources.

### Key Features

- **External LDAP Management**: Connect to and manage existing LDAP servers
- **User Management**: Create, update, and delete LDAP users with full attribute support
- **Group Management**: Manage LDAP groups and group memberships
- **Connection Monitoring**: Real-time monitoring of LDAP server connections
- **Namespace Isolation**: All resources are namespaced for multi-tenancy
- **ACL Support**: Configure search users and access controls
- **TLS Support**: Secure connections with optional TLS/SSL

## Architecture

The operator consists of three main controllers:

1. **LDAPServer Controller**: Manages connections to external LDAP servers
2. **LDAPUser Controller**: Manages individual LDAP users
3. **LDAPGroup Controller**: Manages LDAP groups and memberships

Each controller watches for changes to its respective Custom Resource and reconciles the desired state with the actual LDAP server state.

## Custom Resources

### LDAPServer

The `LDAPServer` resource represents a connection to an external LDAP server.

**Key Specifications:**
- `host`: LDAP server hostname or IP
- `port`: LDAP server port (default: 389)
- `bindDN`: Distinguished name for binding
- `bindPasswordSecret`: Reference to secret containing bind password
- `baseDN`: Base distinguished name for operations
- `tls`: TLS/SSL configuration
- `connectionTimeout`: Connection timeout in seconds
- `healthCheckInterval`: How often to check connection health

**Status Fields:**
- `connectionStatus`: Current connection status (Connected/Disconnected/Error/Unknown)
- `lastChecked`: Timestamp of last connection check
- `message`: Additional status information
- `conditions`: Detailed condition information

### LDAPUser

The `LDAPUser` resource represents an LDAP user entry.

**Key Specifications:**
- `ldapServerRef`: Reference to the LDAPServer
- `username`: LDAP username (uid)
- `email`: User's email address
- `firstName`: User's first name (givenName)
- `lastName`: User's last name (sn)
- `passwordSecret`: Reference to secret containing user password
- `groups`: List of groups the user should belong to
- `organizationalUnit`: OU for the user (default: "users")
- `userID`: Numeric user ID (uidNumber)
- `groupID`: Primary group ID (gidNumber)
- `homeDirectory`: User's home directory
- `loginShell`: User's login shell
- `enabled`: Whether the account is enabled
- `additionalAttributes`: Custom LDAP attributes

**Status Fields:**
- `phase`: Current lifecycle phase (Pending/Ready/Error/Deleting)
- `message`: Status message
- `dn`: Full distinguished name in LDAP
- `groups`: Current group memberships
- `lastModified`: Last modification timestamp

### LDAPGroup

The `LDAPGroup` resource represents an LDAP group entry.

**Key Specifications:**
- `ldapServerRef`: Reference to the LDAPServer
- `groupName`: LDAP group name (cn)
- `description`: Group description
- `members`: List of group members
- `organizationalUnit`: OU for the group (default: "groups")
- `groupID`: Numeric group ID (gidNumber)
- `groupType`: Type of group (posixGroup/groupOfNames/groupOfUniqueNames)
- `additionalAttributes`: Custom LDAP attributes

**Status Fields:**
- `phase`: Current lifecycle phase
- `dn`: Full distinguished name in LDAP
- `members`: Current group members
- `memberCount`: Number of members

## Installation

### Prerequisites

- Kubernetes cluster (version 1.19+)
- kubectl configured to access the cluster
- RBAC enabled

### Install the Operator

1. **Install CRDs:**
   ```bash
   kubectl apply -f config/crd/bases/
   ```

2. **Create namespace:**
   ```bash
   kubectl apply -f config/default/namespace.yaml
   ```

3. **Install RBAC:**
   ```bash
   kubectl apply -f config/rbac/
   ```

4. **Deploy the operator:**
   ```bash
   kubectl apply -f config/manager/
   ```

### Verify Installation

```bash
kubectl get pods -n openldap-operator-system
kubectl get crd | grep ldap
```

## Usage Examples

### Basic LDAP Server Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ldap-admin-secret
  namespace: default
type: Opaque
data:
  password: YWRtaW5QYXNzd29yZA== # base64 encoded password
---
apiVersion: ldap.example.com/v1alpha1
kind: LDAPServer
metadata:
  name: my-ldap-server
  namespace: default
spec:
  host: "ldap.company.com"
  port: 389
  bindDN: "cn=admin,dc=company,dc=com"
  bindPasswordSecret:
    name: ldap-admin-secret
    key: password
  baseDN: "dc=company,dc=com"
  tls:
    enabled: false
  connectionTimeout: 30
  healthCheckInterval: 5m
```

### Creating an LDAP User

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: user-password
  namespace: default
type: Opaque
data:
  password: dXNlclBhc3N3b3Jk # base64 encoded password
---
apiVersion: ldap.example.com/v1alpha1
kind: LDAPUser
metadata:
  name: john-doe
  namespace: default
spec:
  ldapServerRef:
    name: my-ldap-server
  username: johndoe
  email: john.doe@company.com
  firstName: John
  lastName: Doe
  displayName: John Doe
  passwordSecret:
    name: user-password
    key: password
  groups:
    - developers
    - employees
  organizationalUnit: users
  userID: 1001
  groupID: 1000
  homeDirectory: /home/johndoe
  loginShell: /bin/bash
  enabled: true
```

### Creating an LDAP Group

```yaml
apiVersion: ldap.example.com/v1alpha1
kind: LDAPGroup
metadata:
  name: developers
  namespace: default
spec:
  ldapServerRef:
    name: my-ldap-server
  groupName: developers
  description: Development team
  organizationalUnit: groups
  groupID: 2000
  groupType: groupOfNames
  members:
    - johndoe
    - janedoe
```

### TLS Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ldap-ca-cert
  namespace: default
type: Opaque
data:
  ca.crt: LS0tLS1CRUdJTi... # base64 encoded CA certificate
---
apiVersion: ldap.example.com/v1alpha1
kind: LDAPServer
metadata:
  name: secure-ldap-server
  namespace: default
spec:
  host: "ldaps.company.com"
  port: 636
  bindDN: "cn=admin,dc=company,dc=com"
  bindPasswordSecret:
    name: ldap-admin-secret
    key: password
  baseDN: "dc=company,dc=com"
  tls:
    enabled: true
    insecureSkipVerify: false
    caCertSecret:
      name: ldap-ca-cert
      key: ca.crt
```

## Configuration

### Environment Variables

The operator supports the following environment variables:

- `METRICS_BIND_ADDRESS`: Address for metrics endpoint (default: ":8080")
- `HEALTH_PROBE_BIND_ADDRESS`: Address for health probes (default: ":8081")
- `LEADER_ELECT`: Enable leader election (default: false)

### Resource Limits

Configure resource limits in the manager deployment:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 10m
    memory: 64Mi
```

## Security

### RBAC

The operator requires the following permissions:

- `ldapservers`, `ldapusers`, `ldapgroups`: full access
- `secrets`: read access for password retrieval
- `events`: create access for event recording

### Secrets Management

- Store all passwords in Kubernetes secrets
- Use base64 encoding for secret data
- Reference secrets by name and key in CRDs
- Rotate passwords regularly

### TLS Best Practices

- Always use TLS in production environments
- Validate server certificates (`insecureSkipVerify: false`)
- Store CA certificates in Kubernetes secrets
- Consider client certificate authentication

## Troubleshooting

### Common Issues

1. **Connection Failed**
   - Check LDAP server accessibility
   - Verify credentials in secrets
   - Check network policies and firewall rules

2. **User Creation Failed**
   - Verify LDAP server permissions
   - Check organizational unit exists
   - Validate user attributes

3. **Group Membership Issues**
   - Ensure group exists before adding members
   - Check group type compatibility
   - Verify member DN format

### Debugging

1. **Check operator logs:**
   ```bash
   kubectl logs -n openldap-operator-system deployment/controller-manager
   ```

2. **Check resource status:**
   ```bash
   kubectl describe ldapserver my-ldap-server
   kubectl describe ldapuser john-doe
   ```

3. **View events:**
   ```bash
   kubectl get events --sort-by=.metadata.creationTimestamp
   ```

### Log Levels

Set log verbosity using the `--zap-log-level` flag:
- `info`: Standard logging
- `debug`: Verbose logging
- `error`: Error-only logging

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/guided-traffic/openldap-operator.git
cd openldap-operator

# Install dependencies
go mod download

# Build the operator
make build

# Run tests
make test

# Run locally (requires kubectl access)
make run
```

### Building Docker Image

```bash
make docker-build IMG=my-registry/openldap-operator:latest
make docker-push IMG=my-registry/openldap-operator:latest
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

### Project Structure

```
.
├── api/v1alpha1/          # API type definitions
├── cmd/                   # Main application entry point
├── config/                # Kubernetes manifests
├── internal/controller/   # Controller implementations
├── hack/                  # Development scripts
└── docs/                  # Documentation
```

### Testing

The operator includes comprehensive tests:

- Unit tests for controllers
- Integration tests with real LDAP servers
- End-to-end tests with Kubernetes

Run tests with:
```bash
make test
```

### Custom Resource Development

When modifying CRDs:

1. Update the type definitions in `api/v1alpha1/`
2. Run `make generate` to update generated code
3. Run `make manifests` to update CRD manifests
4. Update controllers in `internal/controller/`
5. Add tests and documentation

For more detailed development information, see the [Development Guide](DEVELOPMENT.md).

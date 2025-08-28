# LDAPGroup Controller Documentation

## Overview

The LDAPGroup Controller manages LDAP groups in external LDAP servers. It supports three different group types with proper member management, status tracking, and comprehensive logging.

## Features

### ✅ Implemented Features

- **Multiple Group Types**: Supports `posixGroup`, `groupOfNames`, and `groupOfUniqueNames`
- **Member Management**: Add/remove members with automatic DN resolution
- **Status Tracking**: Real-time status updates with phase tracking
- **Comprehensive Logging**: Detailed logging for debugging and monitoring
- **Finalizer Handling**: Proper cleanup when groups are deleted
- **Error Handling**: Graceful handling of LDAP connection issues
- **Flexible Configuration**: Support for custom attributes and organizational units

### 🔧 Group Types

#### 1. posixGroup
- Object class: `posixGroup`
- Required attributes: `cn`, `gidNumber`
- Member attribute: `memberUid` (stores usernames only)
- Use case: Unix/Linux system groups

```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: unix-admins
spec:
  ldapServerRef:
    name: my-ldap-server
  groupName: admins
  groupType: posixGroup
  groupID: 1000
  members:
    - john
    - jane
```

#### 2. groupOfNames
- Object class: `groupOfNames`
- Required attributes: `cn`, `member`
- Member attribute: `member` (stores full DNs)
- Use case: Application groups, role-based access

```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: app-developers
spec:
  ldapServerRef:
    name: my-ldap-server
  groupName: developers
  groupType: groupOfNames
  members:
    - uid=john,ou=users,dc=example,dc=com
    - jane  # Auto-converted to DN
```

#### 3. groupOfUniqueNames
- Object class: `groupOfUniqueNames`
- Required attributes: `cn`, `uniqueMember`
- Member attribute: `uniqueMember` (stores full DNs)
- Use case: Groups requiring unique membership

```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: unique-managers
spec:
  ldapServerRef:
    name: my-ldap-server
  groupName: managers
  groupType: groupOfUniqueNames
  members:
    - uid=boss,ou=users,dc=example,dc=com
```

## Status Phases

The controller tracks the following phases:

- **Pending**: Group is being created or LDAP server is not available
- **Ready**: Group successfully synchronized with LDAP
- **Error**: Failed to create/update group (check logs for details)
- **Deleting**: Group is being deleted from LDAP

## Logging and Debugging

The controller provides comprehensive logging for troubleshooting:

### Log Levels

- **Info**: Normal operations (group creation, updates, deletions)
- **Error**: LDAP connection issues, authentication failures
- **Debug**: Detailed LDAP operations and member resolution

### Example Log Output

```
2024-08-28T10:15:30Z INFO Starting reconciliation for LDAPGroup {"ldapgroup": "default/developers"}
2024-08-28T10:15:30Z INFO Retrieved LDAPGroup {"groupName": "developers", "groupType": "groupOfNames"}
2024-08-28T10:15:30Z INFO Retrieved LDAP server {"server": "my-ldap", "connectionStatus": "Connected"}
2024-08-28T10:15:30Z INFO Successfully connected to LDAP server
2024-08-28T10:15:30Z INFO Group does not exist, creating {"dn": "cn=developers,ou=groups,dc=example,dc=com"}
2024-08-28T10:15:30Z INFO Creating new LDAP group {"dn": "cn=developers,ou=groups,dc=example,dc=com", "type": "groupOfNames"}
2024-08-28T10:15:30Z INFO Successfully created LDAP group
2024-08-28T10:15:30Z INFO Updated group status {"memberCount": 2}
2024-08-28T10:15:30Z INFO Updating LDAPGroup status {"phase": "Ready", "message": "Group successfully synchronized"}
2024-08-28T10:15:30Z INFO Successfully reconciled LDAPGroup {"groupName": "developers"}
```

## Member Management

### Automatic DN Resolution

The controller automatically converts usernames to full DNs:

- Input: `john` → Output: `uid=john,ou=users,dc=example,dc=com`
- Full DNs are passed through unchanged: `uid=john,ou=people,dc=example,dc=com`

### Member Types by Group Type

| Group Type | Member Attribute | Format | Example |
|------------|------------------|--------|---------|
| posixGroup | memberUid | Username only | `john` |
| groupOfNames | member | Full DN | `uid=john,ou=users,dc=example,dc=com` |
| groupOfUniqueNames | uniqueMember | Full DN | `uid=john,ou=users,dc=example,dc=com` |

## Error Handling

### Common Issues and Solutions

#### 1. LDAP Server Not Connected
```yaml
status:
  phase: Pending
  message: "LDAP server is not connected"
```
**Solution**: Check LDAPServer resource status and connection details.

#### 2. Authentication Failure
```yaml
status:
  phase: Error
  message: "Failed to connect to LDAP: LDAP Result Code 49"
```
**Solution**: Verify bind DN and password in secret.

#### 3. Missing Organizational Unit
```yaml
status:
  phase: Error
  message: "Failed to reconcile group: LDAP Result Code 32"
```
**Solution**: Ensure the specified OU exists in LDAP.

## Configuration Examples

### Basic Group
```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: basic-group
spec:
  ldapServerRef:
    name: my-ldap-server
  groupName: users
  groupType: groupOfNames
```

### Advanced Group with Custom Attributes
```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: advanced-group
spec:
  ldapServerRef:
    name: my-ldap-server
  groupName: developers
  description: "Software Development Team"
  groupType: posixGroup
  groupID: 2000
  organizationalUnit: teams
  members:
    - alice
    - bob
    - uid=charlie,ou=contractors,dc=example,dc=com
  additionalAttributes:
    businessCategory: ["Engineering"]
    departmentNumber: ["IT-DEV"]
```

### Cross-Namespace Server Reference
```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: cross-namespace-group
  namespace: app-namespace
spec:
  ldapServerRef:
    name: shared-ldap
    namespace: ldap-infrastructure
  groupName: app-users
  groupType: groupOfNames
```

## Testing

### Unit Tests
```bash
# Run LDAPGroup controller tests
go test ./internal/controller/... -v -run TestLDAPGroup

# Run helper function tests
go test ./internal/controller/... -v -run TestLDAPGroupControllerHelper
```

### Integration Tests
```bash
# Run with Docker LDAP server (if available)
go test ./internal/ldap/... -v

# Demo script
./scripts/demo-ldapgroup.sh
```

### Manual Testing
```bash
# Create test resources
kubectl apply -f deploy/samples/

# Watch group reconciliation
kubectl get ldapgroup -w

# Check detailed status
kubectl describe ldapgroup my-group

# View operator logs
kubectl logs -f deployment/openldap-operator-controller-manager
```

## Troubleshooting

### Debug Mode
Enable debug logging by setting log level in operator deployment:
```yaml
args:
  - --zap-log-level=debug
```

### Common kubectl Commands
```bash
# List all groups
kubectl get ldapgroup -o wide

# Get group status
kubectl get ldapgroup my-group -o jsonpath='{.status}'

# Watch for changes
kubectl get ldapgroup my-group -w

# Debug events
kubectl describe ldapgroup my-group

# Check finalizers
kubectl get ldapgroup my-group -o jsonpath='{.metadata.finalizers}'
```

### Status Conditions
Check the conditions array for detailed status information:
```bash
kubectl get ldapgroup my-group -o jsonpath='{.status.conditions[*]}'
```

## Performance Considerations

- **Reconciliation Frequency**: Groups are reconciled when changed or every 5 minutes on error
- **LDAP Connection**: New connection per reconciliation (consider connection pooling for high volume)
- **Member Limit**: No hard limit, but large groups may impact performance
- **Concurrent Groups**: Controller handles multiple groups concurrently

## Security Considerations

- **LDAP Credentials**: Store in Kubernetes secrets with proper RBAC
- **TLS**: Enable TLS for production LDAP connections
- **Namespace Isolation**: Groups are namespaced for multi-tenancy
- **Finalizers**: Ensure proper cleanup on deletion

## Future Enhancements

- [ ] LDAP connection pooling
- [ ] Group membership reconciliation (sync existing groups)
- [ ] Bulk member operations
- [ ] Group hierarchy support
- [ ] Metrics and monitoring
- [ ] Webhook validation

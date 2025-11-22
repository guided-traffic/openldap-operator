# OpenLDAP Operator - AI Coding Agent Instructions

## Project Overview

This is a Kubernetes operator built with Kubebuilder v3 that manages **external** LDAP servers (not LDAP-as-a-Service). The operator creates/updates/deletes LDAP users and groups on existing LDAP infrastructure via the go-ldap/ldap library.

**API Group**: `openldap.guided-traffic.com/v1`
**CRDs**: LDAPServer, LDAPUser, LDAPGroup

## Architecture Principles

### Controller Pattern
- Each controller follows standard Kubernetes reconciliation: fetch resource → check finalizers → handle deletion OR reconcile desired state → update status
- **LDAPServer Controller** (`internal/controller/ldapserver_controller.go`): Tests LDAP connections periodically (default 5min), updates ConnectionStatus in status
- **LDAPUser Controller** (`internal/controller/ldapuser_controller.go`): Manages LDAP user entries + group memberships via `spec.groups` field. Sets Phase to Warning if groups are missing but user synced
- **LDAPGroup Controller** (`internal/controller/ldapgroup_controller.go`): Creates LDAP groups. Membership is **managed by LDAPUser** resources, not by LDAPGroup directly

### LDAP Client Abstraction
- `internal/ldap/client.go` wraps go-ldap/ldap with automatic reconnection logic
- TLS is **enabled by default** - only disabled if `spec.tls.enabled: false` explicitly set
- Controllers fetch bind passwords from Secrets using `getSecretValue()` helper
- Connection pattern: create client → perform operations → defer close

### Automatic Home Directory Convention
- If `LDAPUser.spec.homeDirectory` is empty, auto-generates `/home/<username>` for POSIX compliance
- Actual value stored in `status.actualHomeDirectory` for visibility
- See `internal/controller/ldapuser_controller.go:createLDAPUser()` for implementation

## Development Workflows

### Code Generation (Required After API Changes)
```bash
make generate  # DeepCopy methods via controller-gen
make manifests # CRDs to deploy/helm/openldap-operator/crds/
```

### Testing
```bash
make test              # Unit tests (90.6% coverage target)
make test-integration  # Starts Docker osixia/openldap container, runs integration tests
make test-all          # Both unit + integration
```

**Docker-based Integration Tests**: Tests in `internal/ldap/*_test.go` use Ginkgo/Gomega with helper `docker_test_utils.go` that starts osixia/openldap:1.5.0 on port 1389. Skip logic: `if !IsDockerAvailable() { Skip(...) }`

### Build & Run
```bash
make build            # Builds bin/manager
make run              # Runs operator locally (requires kubectl context)
make docker-build     # Builds container image
```

### Helm Deployment
- Chart location: `deploy/helm/openldap-operator/`
- CRDs updated via `make manifests` → committed to `deploy/helm/openldap-operator/crds/`
- Version in `Chart.yaml` should match operator version

## Project-Specific Patterns

### Status Updates
All controllers use `updateStatus()` helper that sets Phase/Message and updates via `r.Status().Update(ctx, resource)`. Always update status as final reconciliation step.

### Finalizers
All resources use finalizer `openldap.guided-traffic.com/finalizer`. Add in first reconcile if missing, handle deletion if `DeletionTimestamp != nil`, remove finalizer after cleanup.

### Validation
- **Declarative**: Kubebuilder markers in `api/v1/*_types.go` (e.g., `+kubebuilder:validation:Enum`)
- **Programmatic**: `api/v1/validation.go` functions called by webhooks or admission logic

### LDAP DN Construction
Pattern: `cn=<name>,ou=<ou>,<baseDN>` for users/groups. See `ldapuser_controller.go:reconcileUser()` for examples.

### Group Membership Management
- **LDAPUser** controls memberships: `spec.groups: [dev, ops]` → controller adds user to those groups
- **LDAPGroup** is passive: `status.members` reflects current state (read-only), actual membership managed by LDAPUser reconciliation
- Missing groups logged in `LDAPUser.status.missingGroups` with Phase=Warning

### Error Handling
Controllers return `ctrl.Result{}` on success, `ctrl.Result{Requeue: true}` or `ctrl.Result{RequeueAfter: duration}` for retries. Connection errors update status but don't block reconciliation.

## Key Files Reference

- **Main entry**: `cmd/main.go` - Manager setup, controller registration
- **API types**: `api/v1/ldap{server,user,group}_types.go` - CRD specs/status
- **Controllers**: `internal/controller/ldap{server,user,group}_controller.go`
- **LDAP client**: `internal/ldap/client.go` - Connection/CRUD operations
- **Test samples**: `deploy/samples/*.yaml` - Example CR manifests
- **Project config**: `PROJECT` file (Kubebuilder metadata), `Makefile` (all commands)

## Common Tasks

### Adding New LDAP Attribute to LDAPUser
1. Add field to `LDAPUserSpec` in `api/v1/ldapuser_types.go` with kubebuilder markers
2. Update `createLDAPUser()` or `updateLDAPUser()` in `internal/controller/ldapuser_controller.go` to map field to LDAP attribute
3. Run `make generate manifests` to update DeepCopy + CRDs
4. Add validation to `api/v1/validation.go` if needed
5. Add tests to `internal/controller/ldapuser_controller_test.go`

### Debugging Connection Issues
Check `LDAPServer.status.connectionStatus` and `status.message`. Enable TLS skip verify for testing: `spec.tls.insecureSkipVerify: true`. Controllers log to stderr - view with `kubectl logs`.

### Extending Group Types
Currently supports posixGroup, groupOfNames, groupOfUniqueNames. See `internal/ldap/client.go:CreateGroup()` for objectClass mappings.

## Conventions

- **Naming**: Controllers named `LDAP{Resource}Reconciler`, client functions PascalCase (e.g., `CreateUser`)
- **Logging**: Use `log.FromContext(ctx)` in controllers
- **Namespacing**: All CRDs are namespaced; LDAPUser can reference LDAPServer in another namespace via `spec.ldapServerRef.namespace`
- **Secrets**: Always use SecretReference type (`name`, `key` fields) for sensitive data

## Important Notes

- **External LDAP Only**: This operator does NOT deploy/manage LDAP servers themselves - it connects to existing ones
- **TLS Default**: TLS enabled unless explicitly disabled - affects both controller connections and client library
- **Test Coverage Goal**: Maintain a high test coverage
- **No Webhooks Currently**: Validation is type-level + programmatic, no admission webhooks deployed
- **Version control**: Never commit to git yourself
- **Test-files**: There should be only one test-file pro code-file.
- **Language**: Write code, comments, variable names, function names, and everything else in English.

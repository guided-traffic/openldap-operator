# E2E Test Configuration

This directory contains all configuration files for end-to-end testing of the OpenLDAP Operator.

## Files

- **openldap.yaml**: Kubernetes manifests for deploying an OpenLDAP test server
  - Service and Deployment for osixia/openldap:1.5.0
  - Configured for testing with TLS disabled
  - Uses `ldap-admin-pass` secret for admin credentials

- **openldap-operator-values.yaml**: Minimal Helm values for operator installation in E2E tests
  - Overrides image settings for local testing
  - Disables webhooks for simplified testing

- **test-resources.yaml**: Sample CRs for E2E validation
  - LDAPServer resource pointing to test LDAP server
  - LDAPGroup resource for testing group management
  - LDAPUser resource for testing user management with group membership

## Usage in CI

These files are used by the E2E test job in `.github/workflows/release.yml`:

1. Create namespace and secret for LDAP admin password
2. Apply `openldap.yaml` to deploy LDAP server
3. Install operator via Helm using `values.yaml`
4. Apply `test-resources.yaml` to create test resources
5. Verify resources are synced and user exists in LDAP

## Local Testing

To run E2E tests locally with Kind:

```bash
# Create Kind cluster
kind create cluster --name openldap-operator-test

# Create namespace and secret
kubectl create namespace ldap-test
kubectl -n ldap-test create secret generic ldap-admin-pass \
  --from-literal=adminpassword=admin

# Deploy LDAP server
kubectl -n ldap-test apply -f test/e2e/openldap.yaml

# Wait for LDAP to be ready
kubectl -n ldap-test wait --for=condition=available --timeout=120s deployment/openldap

# Build and load operator image
make docker-build IMG=openldap-operator:test
kind load docker-image openldap-operator:test --name openldap-operator-test

# Install operator
helm install openldap-operator deploy/helm/openldap-operator \
  --namespace openldap-operator-system \
  --create-namespace \
  --values test/e2e/openldap-operator-values.yaml \
  --wait --timeout 120s

# Create test resources
kubectl -n ldap-test apply -f test/e2e/test-resources.yaml

# Verify
kubectl -n ldap-test get ldapserver,ldapgroup,ldapuser

# Cleanup
kind delete cluster --name openldap-operator-test
```

# OpenLDAP Operator Helm Chart Installation Guide

This guide provides detailed instructions for installing the OpenLDAP Operator using Helm.

## Quick Start

### 1. Install from Source

```bash
# Cl# Manually install CRDs if needed
kubectl apply -f deploy/helm/openldap-operator/crds/e the repository
git clone https://github.com/guided-traffic/openldap-operator.git
cd openldap-operator

# Install the operator
helm install openldap-operator deploy/helm/openldap-operator

# Verify installation
kubectl get pods -l app.kubernetes.io/name=openldap-operator
```

### 2. Install from Package

```bash
# Download the chart package
wget https://github.com/guided-traffic/openldap-operator/releases/download/v0.1.0/openldap-operator-0.1.0.tgz

# Install from package
helm install openldap-operator openldap-operator-0.1.0.tgz
```

## Configuration Examples

### Basic Installation

```bash
helm install openldap-operator deploy/helm/openldap-operator
```

### Production Installation

```bash
helm install openldap-operator deploy/helm/openldap-operator \
  --set image.repository=my-registry/openldap-operator \
  --set image.tag=v1.0.0 \
  --set operator.replicaCount=2 \
  --set operator.resources.limits.cpu=1000m \
  --set operator.resources.limits.memory=256Mi \
  --set metrics.serviceMonitor.enabled=true
```

### Installation with Examples

```bash
helm install openldap-operator deploy/helm/openldap-operator \
  --set examples.enabled=true \
  --set examples.ldapServer.host=ldap.mycompany.com \
  --set examples.ldapServer.baseDN="dc=mycompany,dc=com"
```

### Installation in Custom Namespace

```bash
# Create namespace
kubectl create namespace ldap-operator

# Install with custom namespace
helm install openldap-operator deploy/helm/openldap-operator \
  --namespace ldap-operator \
  --create-namespace
```

## Configuration Values

Create a `values.yaml` file for complex configurations:

```yaml
# Custom values for production deployment
image:
  repository: my-registry/openldap-operator
  tag: "v1.0.0"
  pullPolicy: Always

imagePullSecrets:
  - name: my-registry-secret

operator:
  replicaCount: 2
  resources:
    limits:
      cpu: 1000m
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi

  nodeSelector:
    node-type: operator

  tolerations:
    - key: "operator"
      operator: "Equal"
      value: "true"
      effect: "NoSchedule"

metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    namespace: monitoring

podDisruptionBudget:
  enabled: true
  minAvailable: 1

# Custom LDAP server for examples
examples:
  enabled: true
  ldapServer:
    name: company-ldap
    namespace: ldap-operator
    host: ldap.company.com
    port: 636
    baseDN: "dc=company,dc=com"
    bindDN: "cn=admin,dc=company,dc=com"
    tls:
      enabled: true
      insecureSkipVerify: false
```

Install with custom values:

```bash
helm install openldap-operator deploy/helm/openldap-operator -f values.yaml
```

## Upgrade and Rollback

### CRD Updates

Starting with version 1.0.2, the chart includes automatic CRD updates during Helm upgrades. This ensures that CustomResourceDefinitions are always synchronized with the operator version.

**How it works:**
- Before each `helm upgrade`, a pre-upgrade hook job runs
- The job applies the latest CRD definitions from the chart
- The job is automatically cleaned up after successful completion

**Configuration:**
```bash
# Enable CRD updates (default: true)
helm upgrade openldap-operator deploy/helm/openldap-operator \
  --set crdUpdate.enabled=true

# Disable CRD updates if you manage them separately
helm upgrade openldap-operator deploy/helm/openldap-operator \
  --set crdUpdate.enabled=false

# Use custom kubectl image for CRD updates
helm upgrade openldap-operator deploy/helm/openldap-operator \
  --set crdUpdate.image.repository=my-registry/kubectl \
  --set crdUpdate.image.tag=1.28.4
```

**Manual CRD Updates:**
If you prefer to manage CRDs manually or the automatic update fails:
```bash
# Apply CRDs directly
kubectl apply -f deploy/helm/openldap-operator/crds/

# Or use kubectl with the chart
helm template openldap-operator deploy/helm/openldap-operator | \
  grep -A 1000 "kind: CustomResourceDefinition" | \
  kubectl apply -f -
```

### Upgrade Installation

```bash
# Upgrade with new values
helm upgrade openldap-operator deploy/helm/openldap-operator \
  --set image.tag=v1.1.0

# Upgrade with values file
helm upgrade openldap-operator deploy/helm/openldap-operator -f values-v1.1.yaml
```

### Rollback Installation

```bash
# List revisions
helm history openldap-operator

# Rollback to previous version
helm rollback openldap-operator

# Rollback to specific revision
helm rollback openldap-operator 2
```

## Uninstalling

```bash
# Uninstall the release
helm uninstall openldap-operator

# Uninstall from custom namespace
helm uninstall openldap-operator --namespace ldap-operator
```

**Note**: CRDs may remain after uninstallation. To remove them:

```bash
kubectl delete crd ldapservers.openldap.guided-traffic.com
kubectl delete crd ldapusers.openldap.guided-traffic.com
kubectl delete crd ldapgroups.openldap.guided-traffic.com
```

## Troubleshooting

### Common Issues

1. **Image Pull Errors**
   ```bash
   # Check if image exists and credentials are correct
   kubectl describe pod -l app.kubernetes.io/name=openldap-operator
   ```

2. **RBAC Permission Issues**
   ```bash
   # Check if service account has required permissions
   kubectl auth can-i "*" "openldap.guided-traffic.com/*" \
     --as=system:serviceaccount:default:openldap-operator-openldap-operator
   ```

3. **CRD Installation Issues**
   ```bash
   # Manually install CRDs if needed
      ```bash
   kubectl apply -f deploy/helm/openldap-operator/crds/
   ```
   ```

### Debug Commands

```bash
# Check all resources created by Helm
helm get all openldap-operator

# View computed values
helm get values openldap-operator

# Check operator logs
kubectl logs -l app.kubernetes.io/name=openldap-operator -f

# Check operator status
kubectl get deployment openldap-operator

# Test with dry-run
helm install test-operator deploy/helm/openldap-operator --dry-run --debug
```

## Using the Operator

After successful installation, you can start managing LDAP resources:

### 1. Create an LDAP Server

```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPServer
metadata:
  name: my-ldap-server
  namespace: default
spec:
  host: ldap.example.com
  port: 389
  baseDN: "dc=example,dc=com"
  bindDN: "cn=admin,dc=example,dc=com"
  bindPasswordSecret:
    name: ldap-admin-secret
    key: password
```

### 2. Create Admin Secret

```bash
kubectl create secret generic ldap-admin-secret \
  --from-literal=password='your-admin-password'
```

### 3. Create LDAP Users

```yaml
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPUser
metadata:
  name: john-doe
  namespace: default
spec:
  ldapServerRef:
    name: my-ldap-server
  username: johndoe
  firstName: John
  lastName: Doe
  email: john.doe@example.com
  organizationalUnit: users
  userID: 1001
  groupID: 1001
  homeDirectory: /home/johndoe
  loginShell: /bin/bash
  passwordSecret:
    name: johndoe-password
    key: password
```

For more examples and detailed usage, see the main [README](../../README.md).

## Support

- **Documentation**: [Project README](../../README.md)
- **Issues**: [GitHub Issues](https://github.com/guided-traffic/openldap-operator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/guided-traffic/openldap-operator/discussions)

# OpenLDAP Operator Helm Chart

This Helm chart installs the OpenLDAP Operator on a Kubernetes cluster using the Helm package manager.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.2.0+
- PV provisioner support in the underlying infrastructure (optional, for persistent storage)

## Installing the Chart

To install the chart with the release name `openldap-operator`:

```bash
helm install openldap-operator ./deploy/helm/openldap-operator
```

The command deploys the OpenLDAP Operator on the Kubernetes cluster in the default configuration. The [Parameters](#parameters) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `openldap-operator` deployment:

```bash
helm delete openldap-operator
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Parameters

### Global parameters

| Name                      | Description                                     | Value |
| ------------------------- | ----------------------------------------------- | ----- |
| `global.imageRegistry`    | Global Docker image registry                    | `""`  |
| `global.imagePullSecrets` | Global Docker registry secret names as an array| `[]`  |

### Operator Configuration

| Name                                    | Description                                           | Value                    |
| --------------------------------------- | ----------------------------------------------------- | ------------------------ |
| `image.registry`                        | OpenLDAP Operator image registry                      | `""`                     |
| `image.repository`                      | OpenLDAP Operator image repository                    | `openldap-operator`      |
| `image.tag`                             | OpenLDAP Operator image tag                           | `latest`                 |
| `image.pullPolicy`                      | OpenLDAP Operator image pull policy                   | `IfNotPresent`           |
| `operator.replicaCount`                 | Number of operator replicas                           | `1`                      |
| `operator.leaderElection.enabled`       | Enable leader election                                | `true`                   |
| `operator.resources.limits.cpu`         | The resources limits for the operator containers     | `500m`                   |
| `operator.resources.limits.memory`      | The resources limits for the operator containers     | `128Mi`                  |
| `operator.resources.requests.cpu`       | The requested resources for the operator containers  | `10m`                    |
| `operator.resources.requests.memory`    | The requested resources for the operator containers  | `64Mi`                   |

### RBAC Configuration

| Name                | Description                                         | Value  |
| ------------------- | --------------------------------------------------- | ------ |
| `rbac.create`       | Specifies whether RBAC resources should be created | `true` |
| `rbac.annotations`  | Annotations to add to the ClusterRole              | `{}`   |

### Service Account

| Name                         | Description                                            | Value  |
| ---------------------------- | ------------------------------------------------------ | ------ |
| `serviceAccount.create`      | Specifies whether a service account should be created | `true` |
| `serviceAccount.annotations` | Annotations to add to the service account             | `{}`   |
| `serviceAccount.name`        | The name of the service account to use                | `""`   |

### Monitoring

| Name                                        | Description                                  | Value     |
| ------------------------------------------- | -------------------------------------------- | --------- |
| `metrics.enabled`                           | Enable metrics endpoint                      | `true`    |
| `metrics.service.type`                      | Metrics service type                         | `ClusterIP` |
| `metrics.service.port`                      | Metrics service port                         | `8080`    |
| `metrics.serviceMonitor.enabled`            | Enable ServiceMonitor creation               | `false`   |
| `metrics.serviceMonitor.interval`           | Scrape interval                              | `30s`     |
| `metrics.serviceMonitor.scrapeTimeout`      | Scrape timeout                               | `10s`     |

### Health Checks

| Name                                          | Description                           | Value |
| --------------------------------------------- | ------------------------------------- | ----- |
| `healthCheck.livenessProbe.initialDelaySeconds` | Initial delay for liveness probe   | `15`  |
| `healthCheck.livenessProbe.periodSeconds`       | Period for liveness probe          | `20`  |
| `healthCheck.readinessProbe.initialDelaySeconds`| Initial delay for readiness probe  | `5`   |
| `healthCheck.readinessProbe.periodSeconds`      | Period for readiness probe         | `10`  |

### Examples

| Name                                      | Description                                | Value                      |
| ----------------------------------------- | ------------------------------------------ | -------------------------- |
| `examples.enabled`                        | Enable creation of example resources       | `false`                    |
| `examples.ldapServer.name`                | Name of the example LDAP server           | `example-ldap`             |
| `examples.ldapServer.namespace`           | Namespace for the example LDAP server     | `default`                  |
| `examples.ldapServer.host`                | Host of the example LDAP server           | `ldap.example.com`         |
| `examples.ldapServer.port`                | Port of the example LDAP server           | `389`                      |
| `examples.ldapServer.baseDN`              | Base DN of the example LDAP server        | `dc=example,dc=com`        |
| `examples.ldapServer.bindDN`              | Bind DN of the example LDAP server        | `cn=admin,dc=example,dc=com` |

## Configuration and Installation Details

### Custom Resource Definitions (CRDs)

The chart includes the following CRDs:
- `LDAPServer` - Represents an external LDAP server connection
- `LDAPUser` - Represents an LDAP user to be managed
- `LDAPGroup` - Represents an LDAP group to be managed

#### CRD Upgrades

By default, Helm does not upgrade CRDs during chart upgrades. This chart includes an automatic CRD update mechanism using Helm hooks that runs before each upgrade to ensure your CRDs are always up-to-date.

The CRD update feature can be controlled via the following parameters:

| Name                              | Description                                    | Value           |
| --------------------------------- | ---------------------------------------------- | --------------- |
| `crdUpdate.enabled`               | Enable automatic CRD updates during upgrades  | `true`          |
| `crdUpdate.image.repository`      | Image for the CRD update job                   | `bitnami/kubectl` |
| `crdUpdate.image.tag`             | Image tag for the CRD update job               | `1.28`          |
| `crdUpdate.resources.limits.cpu`  | CPU limit for the CRD update job               | `100m`          |
| `crdUpdate.resources.limits.memory` | Memory limit for the CRD update job          | `64Mi`          |

To disable automatic CRD updates:
```bash
helm upgrade openldap-operator ./deploy/helm/openldap-operator --set crdUpdate.enabled=false
```

**Note**: The CRD update job requires cluster-admin permissions to update CustomResourceDefinitions. The chart creates the necessary RBAC resources automatically.

### RBAC

By default, the chart creates RBAC resources including:
- ClusterRole with permissions for managing LDAP resources and secrets
- ClusterRoleBinding to bind the ClusterRole to the service account
- Role and RoleBinding for leader election (if enabled)

### Monitoring

The operator exposes metrics on port 8080 by default. You can enable Prometheus monitoring by:

1. Setting `metrics.serviceMonitor.enabled=true`
2. Ensuring you have Prometheus Operator installed in your cluster

### Examples

To deploy with example resources:

```bash
helm install openldap-operator ./deploy/helm/openldap-operator --set examples.enabled=true
```

This will create:
- An example LDAPServer resource
- A secret with default admin credentials (change in production!)

### Custom Values

Create a custom values file:

```yaml
# values-production.yaml
image:
  repository: my-registry/openldap-operator
  tag: "v1.0.0"
  pullPolicy: Always

operator:
  replicaCount: 2
  resources:
    limits:
      cpu: 1000m
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi

metrics:
  serviceMonitor:
    enabled: true
    interval: 15s

examples:
  enabled: true
  ldapServer:
    host: "ldap.mycompany.com"
    baseDN: "dc=mycompany,dc=com"
```

Install with custom values:

```bash
helm install openldap-operator ./deploy/helm/openldap-operator -f values-production.yaml
```

## Troubleshooting

### Check operator status
```bash
kubectl get deployment openldap-operator
kubectl logs -l app.kubernetes.io/name=openldap-operator -f
```

### Verify CRDs are installed
```bash
kubectl get crd | grep openldap
```

### Check RBAC permissions
```bash
kubectl auth can-i "*" "openldap.guided-traffic.com/*" --as=system:serviceaccount:default:openldap-operator
```

## Contributing

Please see the main project repository for contribution guidelines: https://github.com/guided-traffic/openldap-operator

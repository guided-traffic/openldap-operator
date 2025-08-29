# CRD Update Helm Hook Implementation

## Übersicht

Diese Implementierung löst das bekannte Problem, dass Helm CRDs nach der ersten Installation nicht automatisch aktualisiert. Die Lösung verwendet Helm Hooks, um CRDs vor jedem Upgrade zu aktualisieren.

## Komponenten

### 1. CRD Update Job Template (`templates/crd-update-job.yaml`)

**Funktionalität:**
- Wird als Pre-Upgrade Hook (`helm.sh/hook: pre-upgrade`) ausgeführt
- Führt `kubectl apply` für alle CRDs aus
- Verwendet Hook-Weight `-5` für korrekte Reihenfolge
- Bereinigt sich selbst nach erfolgreichem Abschluss

**Sicherheit:**
- Eigener ServiceAccount für minimale Berechtigungen
- ClusterRole nur für CRD-Management
- Non-root Container mit Security Context
- ReadOnlyRootFilesystem (false nur für temporäre CRD-Dateien)

**Robustheit:**
- `set -e` für sofortigen Stopp bei Fehlern
- Temporäre Dateien für CRD-Inhalte
- Detaillierte Logging-Ausgaben

### 2. CRD Template (`templates/crds.yaml`)

**Funktionalität:**
- Inkludiert alle CRD-Dateien aus dem `crds/` Verzeichnis
- Ermöglicht Installation bei erstem Deployment
- Kontrolliert durch `crds.install` Flag

### 3. Konfiguration (`values.yaml`)

```yaml
crdUpdate:
  enabled: true                    # Aktiviert/deaktiviert CRD-Updates
  image:
    repository: bitnami/kubectl    # Kubectl-Image für CRD-Updates
    tag: "1.28"                    # Kubernetes-kompatible Version
    pullPolicy: IfNotPresent
  resources:                       # Ressourcen-Limits für Update-Job
    limits:
      cpu: 100m
      memory: 64Mi
    requests:
      cpu: 10m
      memory: 32Mi
```

## Funktionsweise

### Installation (Erstmalig)
1. Helm installiert CRDs über `templates/crds.yaml`
2. Standard-Deployment wird erstellt
3. Operator startet und ist betriebsbereit

### Upgrade
1. **Pre-Upgrade Hook** (Weight: -10):
   - ServiceAccount, ClusterRole, ClusterRoleBinding werden erstellt
2. **Pre-Upgrade Hook** (Weight: -5):
   - CRD-Update-Job wird gestartet
   - Job lädt aktuelle CRDs aus Chart und wendet sie an
   - Job beendet sich erfolgreich
3. **Standard Helm Upgrade**:
   - Operator-Deployment wird aktualisiert
   - Alle anderen Ressourcen werden aktualisiert
4. **Cleanup**:
   - CRD-Update-Ressourcen werden automatisch gelöscht

### Vorteile

1. **Automatisch**: Keine manuellen Schritte erforderlich
2. **Sicher**: Minimale RBAC-Berechtigungen
3. **Robust**: Fehlerbehandlung und Cleanup
4. **Konfigurierbar**: Kann deaktiviert werden
5. **Kompatibel**: Funktioniert mit Standard-Helm-Workflows

### Verwendung

```bash
# Standard-Upgrade (CRD-Update automatisch aktiviert)
helm upgrade openldap-operator ./deploy/helm/openldap-operator

# Upgrade ohne CRD-Update
helm upgrade openldap-operator ./deploy/helm/openldap-operator \
  --set crdUpdate.enabled=false

# Upgrade mit benutzerdefiniertem kubectl-Image
helm upgrade openldap-operator ./deploy/helm/openldap-operator \
  --set crdUpdate.image.repository=my-registry/kubectl \
  --set crdUpdate.image.tag=1.29
```

## Problemlösung

### Hook-Logs anzeigen
```bash
# Job-Status prüfen
kubectl get jobs -l app.kubernetes.io/component=crd-update

# Job-Logs anzeigen
kubectl logs -l app.kubernetes.io/component=crd-update
```

### Manuelle CRD-Updates
```bash
# Bei Hook-Fehlern CRDs manuell anwenden
kubectl apply -f deploy/helm/openldap-operator/crds/
```

### Berechtigungsprobleme
```bash
# RBAC-Berechtigungen prüfen
kubectl auth can-i create customresourcedefinitions \
  --as=system:serviceaccount:default:release-name-crd-update
```

## Kompatibilität

- **Helm**: 3.2.0+
- **Kubernetes**: 1.19+
- **kubectl Image**: Muss zur Kubernetes-Version kompatibel sein

## Implementierungsdetails

Die Lösung folgt Helm-Best-Practices:
- Hooks haben korrekte Weights für Ausführungsreihenfolge
- Cleanup-Policy verhindert Ressourcen-Anhäufung
- Template-Conditionals ermöglichen Feature-Toggles
- Resource-Limits verhindern uneingeschränkten Ressourcenverbrauch

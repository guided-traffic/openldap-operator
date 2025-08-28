#!/bin/bash

# Demo script for testing LDAPGroup controller functionality
# This script demonstrates how to create and manage LDAP groups

set -e

echo "=== LDAPGroup Controller Demo ==="
echo

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "kubectl is required but not found"
    exit 1
fi

NAMESPACE="ldap-operator-demo"

echo "1. Creating demo namespace..."
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

echo "2. Creating LDAP server secret..."
kubectl create secret generic ldap-admin-secret \
    --namespace=${NAMESPACE} \
    --from-literal=password='admin123' \
    --dry-run=client -o yaml | kubectl apply -f -

echo "3. Creating LDAP server resource..."
cat <<EOF | kubectl apply -f -
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPServer
metadata:
  name: demo-ldap-server
  namespace: ${NAMESPACE}
spec:
  host: openldap.${NAMESPACE}.svc.cluster.local
  port: 389
  bindDN: "cn=admin,dc=example,dc=com"
  bindPasswordSecret:
    name: ldap-admin-secret
    key: password
  baseDN: "dc=example,dc=com"
  tls:
    enabled: false
EOF

echo "4. Creating LDAP groups..."

echo "   4.1. Creating posixGroup..."
cat <<EOF | kubectl apply -f -
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: developers-posix
  namespace: ${NAMESPACE}
spec:
  ldapServerRef:
    name: demo-ldap-server
  groupName: developers
  description: Development team (POSIX group)
  groupType: posixGroup
  groupID: 1001
  organizationalUnit: groups
  members:
    - johndoe
    - janedoe
EOF

echo "   4.2. Creating groupOfNames..."
cat <<EOF | kubectl apply -f -
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: managers-group
  namespace: ${NAMESPACE}
spec:
  ldapServerRef:
    name: demo-ldap-server
  groupName: managers
  description: Team managers
  groupType: groupOfNames
  organizationalUnit: groups
  members:
    - uid=boss,ou=users,dc=example,dc=com
    - uid=supervisor,ou=users,dc=example,dc=com
EOF

echo "   4.3. Creating groupOfUniqueNames..."
cat <<EOF | kubectl apply -f -
apiVersion: openldap.guided-traffic.com/v1
kind: LDAPGroup
metadata:
  name: admins-unique
  namespace: ${NAMESPACE}
spec:
  ldapServerRef:
    name: demo-ldap-server
  groupName: admins
  description: System administrators
  groupType: groupOfUniqueNames
  organizationalUnit: groups
  members:
    - uid=admin,ou=users,dc=example,dc=com
  additionalAttributes:
    businessCategory: ["IT", "Operations"]
    description: ["High privilege users", "System maintenance"]
EOF

echo "5. Waiting for resources to be created..."
sleep 3

echo "6. Checking resource status..."
echo
echo "=== LDAP Server Status ==="
kubectl get ldapserver -n ${NAMESPACE} -o wide

echo
echo "=== LDAP Groups Status ==="
kubectl get ldapgroup -n ${NAMESPACE} -o wide

echo
echo "=== Detailed Group Status ==="
for group in developers-posix managers-group admins-unique; do
    echo "--- Group: $group ---"
    kubectl get ldapgroup $group -n ${NAMESPACE} -o jsonpath='{.status}' | jq '.' 2>/dev/null || echo "Status not available yet"
    echo
done

echo "7. Viewing operator logs (last 20 lines)..."
echo "   Note: This will show LDAPGroup controller activity"
echo
kubectl logs -n openldap-operator-system deployment/openldap-operator-controller-manager --tail=20 || echo "Operator not running in cluster"

echo
echo "=== Demo Complete ==="
echo
echo "To cleanup, run:"
echo "kubectl delete namespace ${NAMESPACE}"
echo
echo "To watch group reconciliation in real-time:"
echo "kubectl get ldapgroup -n ${NAMESPACE} -w"
echo
echo "To view detailed group status:"
echo "kubectl describe ldapgroup -n ${NAMESPACE}"

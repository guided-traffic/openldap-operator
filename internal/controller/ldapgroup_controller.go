/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

// LDAPGroupReconciler reconciles a LDAPGroup object
type LDAPGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapgroups/finalizers,verbs=update
//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapservers,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LDAPGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("ldapgroup", req.NamespacedName)
	logger.Info("Starting reconciliation for LDAPGroup")

	// Fetch the LDAPGroup instance
	ldapGroup := &openldapv1.LDAPGroup{}
	err := r.Get(ctx, req.NamespacedName, ldapGroup)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("LDAPGroup resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get LDAPGroup")
		return ctrl.Result{}, err
	}

	logger.Info("Retrieved LDAPGroup", "groupName", ldapGroup.Spec.GroupName, "groupType", ldapGroup.Spec.GroupType)

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(ldapGroup, "openldap.guided-traffic.com/finalizer") {
		logger.Info("Adding finalizer to LDAPGroup")
		controllerutil.AddFinalizer(ldapGroup, "openldap.guided-traffic.com/finalizer")
		if err := r.Update(ctx, ldapGroup); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle deletion
	if ldapGroup.DeletionTimestamp != nil {
		logger.Info("LDAPGroup is being deleted")
		return r.handleDeletion(ctx, ldapGroup)
	}

	// Get the referenced LDAP server
	ldapServer, err := r.getLDAPServer(ctx, ldapGroup)
	if err != nil {
		logger.Error(err, "Failed to get LDAP server")
		return r.updateStatus(ctx, ldapGroup, openldapv1.GroupPhaseError, fmt.Sprintf("Failed to get LDAP server: %v", err))
	}

	logger.Info("Retrieved LDAP server", "server", ldapServer.Name, "connectionStatus", ldapServer.Status.ConnectionStatus)

	// Check if LDAP server is connected
	if ldapServer.Status.ConnectionStatus != openldapv1.ConnectionStatusConnected {
		logger.Info("LDAP server is not connected, waiting")
		return r.updateStatus(ctx, ldapGroup, openldapv1.GroupPhasePending, "LDAP server is not connected")
	}

	// Connect to LDAP server
	conn, err := r.connectToLDAP(ctx, ldapServer)
	if err != nil {
		logger.Error(err, "Failed to connect to LDAP")
		return r.updateStatus(ctx, ldapGroup, openldapv1.GroupPhaseError, fmt.Sprintf("Failed to connect to LDAP: %v", err))
	}
	defer conn.Close()

	logger.Info("Successfully connected to LDAP server")

	// Create or update the group
	err = r.reconcileGroup(ctx, conn, ldapServer, ldapGroup)
	if err != nil {
		logger.Error(err, "Failed to reconcile group")
		return r.updateStatus(ctx, ldapGroup, openldapv1.GroupPhaseError, fmt.Sprintf("Failed to reconcile group: %v", err))
	}

	logger.Info("Successfully reconciled LDAPGroup", "groupName", ldapGroup.Spec.GroupName)
	return r.updateStatus(ctx, ldapGroup, openldapv1.GroupPhaseReady, "Group successfully synchronized")
}

// getLDAPServer retrieves the referenced LDAP server
func (r *LDAPGroupReconciler) getLDAPServer(ctx context.Context, ldapGroup *openldapv1.LDAPGroup) (*openldapv1.LDAPServer, error) {
	ldapServer := &openldapv1.LDAPServer{}
	namespace := ldapGroup.Namespace
	if ldapGroup.Spec.LDAPServerRef.Namespace != "" {
		namespace = ldapGroup.Spec.LDAPServerRef.Namespace
	}

	err := r.Get(ctx, types.NamespacedName{
		Name:      ldapGroup.Spec.LDAPServerRef.Name,
		Namespace: namespace,
	}, ldapServer)

	return ldapServer, err
}

// connectToLDAP establishes a connection to the LDAP server
func (r *LDAPGroupReconciler) connectToLDAP(ctx context.Context, ldapServer *openldapv1.LDAPServer) (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	address := fmt.Sprintf("%s:%d", ldapServer.Spec.Host, ldapServer.Spec.Port)

	// TLS Logic: TLS is enabled by default, only disabled if explicitly set to false
	useTLS := true // Default to TLS
	if ldapServer.Spec.TLS != nil && !ldapServer.Spec.TLS.Enabled {
		useTLS = false // Only disable if explicitly set to false
	}

	// Create connection based on TLS configuration
	var ldapURL string
	if useTLS {
		ldapURL = fmt.Sprintf("ldaps://%s", address)
	} else {
		ldapURL = fmt.Sprintf("ldap://%s", address)
	}

	// Configure TLS settings if using TLS
	if useTLS {
		tlsConfig := &tls.Config{
			ServerName: ldapServer.Spec.Host,
			MinVersion: tls.VersionTLS12, // Enforce minimum TLS 1.2
		}

		// Configure TLS settings if TLS config is provided
		if ldapServer.Spec.TLS != nil {
			tlsConfig.InsecureSkipVerify = ldapServer.Spec.TLS.InsecureSkipVerify
		} else {
			// Default TLS settings when no config is provided
			tlsConfig.InsecureSkipVerify = false
		}

		conn, err = ldap.DialURL(ldapURL, ldap.DialWithTLSConfig(tlsConfig))
	} else {
		conn, err = ldap.DialURL(ldapURL)
	}

	if err != nil {
		return nil, err
	}

	// Get bind password
	bindPassword, err := r.getSecretValue(ctx, ldapServer.Namespace, ldapServer.Spec.BindPasswordSecret)
	if err != nil {
		_ = conn.Close() // Best effort close, ignore errors
		return nil, err
	}

	// Bind to LDAP
	err = conn.Bind(ldapServer.Spec.BindDN, bindPassword)
	if err != nil {
		_ = conn.Close() // Best effort close, ignore errors
		return nil, err
	}

	return conn, nil
}

// reconcileGroup creates or updates the group in LDAP
func (r *LDAPGroupReconciler) reconcileGroup(ctx context.Context, conn *ldap.Conn, ldapServer *openldapv1.LDAPServer, ldapGroup *openldapv1.LDAPGroup) error {
	logger := log.FromContext(ctx).WithValues("ldapgroup", ldapGroup.Name)

	// Construct the group DN
	ou := ldapGroup.Spec.OrganizationalUnit
	if ou == "" {
		ou = "groups"
	}
	groupDN := fmt.Sprintf("cn=%s,ou=%s,%s", ldapGroup.Spec.GroupName, ou, ldapServer.Spec.BaseDN)

	logger.Info("Reconciling group", "dn", groupDN)

	// Check if group exists
	searchRequest := ldap.NewSearchRequest(
		groupDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1,
		30,
		false,
		"(objectClass=*)",
		[]string{"*"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	groupExists := err == nil && len(searchResult.Entries) > 0

	if groupExists {
		logger.Info("Group exists, updating")
		// Update existing group
		err = r.updateLDAPGroup(ctx, conn, groupDN, ldapGroup)
		if err != nil {
			return err
		}
	} else {
		logger.Info("Group does not exist, creating")
		// Create new group
		err = r.createLDAPGroup(ctx, conn, groupDN, ldapServer, ldapGroup)
		if err != nil {
			return err
		}
	}

	// Update status with current member information
	return r.updateGroupStatus(ctx, conn, groupDN, ldapGroup)
}

// createLDAPGroup creates a new group in LDAP
func (r *LDAPGroupReconciler) createLDAPGroup(ctx context.Context, conn *ldap.Conn, groupDN string, ldapServer *openldapv1.LDAPServer, ldapGroup *openldapv1.LDAPGroup) error {
	logger := log.FromContext(ctx).WithValues("ldapgroup", ldapGroup.Name)
	logger.Info("Creating new LDAP group", "dn", groupDN, "type", ldapGroup.Spec.GroupType)

	addRequest := ldap.NewAddRequest(groupDN, nil)

	// Set object classes based on group type
	switch ldapGroup.Spec.GroupType {
	case openldapv1.GroupTypePosix:
		addRequest.Attribute("objectClass", []string{"posixGroup"})
		if ldapGroup.Spec.GroupID != nil {
			addRequest.Attribute("gidNumber", []string{fmt.Sprintf("%d", *ldapGroup.Spec.GroupID)})
		}
	case openldapv1.GroupTypeGroupOfNames:
		addRequest.Attribute("objectClass", []string{"groupOfNames"})
		// groupOfNames requires at least one member - add dummy member
		addRequest.Attribute("member", []string{"cn=dummy"})
	case openldapv1.GroupTypeGroupOfUniqueNames:
		addRequest.Attribute("objectClass", []string{"groupOfUniqueNames"})
		// groupOfUniqueNames requires at least one uniqueMember - add dummy member
		addRequest.Attribute("uniqueMember", []string{"cn=dummy"})
	default:
		// Default to groupOfNames
		addRequest.Attribute("objectClass", []string{"groupOfNames"})
		addRequest.Attribute("member", []string{"cn=dummy"})
	}

	// Basic attributes
	addRequest.Attribute("cn", []string{ldapGroup.Spec.GroupName})

	if ldapGroup.Spec.Description != "" {
		addRequest.Attribute("description", []string{ldapGroup.Spec.Description})
	}

	// Add any additional attributes
	for attr, values := range ldapGroup.Spec.AdditionalAttributes {
		addRequest.Attribute(attr, values)
	}

	err := conn.Add(addRequest)
	if err != nil {
		logger.Error(err, "Failed to create LDAP group")
		return err
	}

	logger.Info("Successfully created LDAP group")
	return nil
}

// updateLDAPGroup updates an existing group in LDAP
func (r *LDAPGroupReconciler) updateLDAPGroup(ctx context.Context, conn *ldap.Conn, groupDN string, ldapGroup *openldapv1.LDAPGroup) error {
	logger := log.FromContext(ctx).WithValues("ldapgroup", ldapGroup.Name)
	logger.Info("Updating existing LDAP group", "dn", groupDN)

	modifyRequest := ldap.NewModifyRequest(groupDN, nil)

	// Update description
	if ldapGroup.Spec.Description != "" {
		modifyRequest.Replace("description", []string{ldapGroup.Spec.Description})
	}

	// Groups no longer manage members - members are managed by LDAPUser objects

	// Only modify if there are changes
	if len(modifyRequest.Changes) > 0 {
		err := conn.Modify(modifyRequest)
		if err != nil {
			logger.Error(err, "Failed to update LDAP group")
			return err
		}
		logger.Info("Successfully updated LDAP group")
	} else {
		logger.Info("No changes needed for LDAP group")
	}

	return nil
}

// updateGroupStatus updates the group status with current member information
func (r *LDAPGroupReconciler) updateGroupStatus(ctx context.Context, conn *ldap.Conn, groupDN string, ldapGroup *openldapv1.LDAPGroup) error {
	logger := log.FromContext(ctx).WithValues("ldapgroup", ldapGroup.Name)

	// Search for current group members
	searchRequest := ldap.NewSearchRequest(
		groupDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1,
		30,
		false,
		"(objectClass=*)",
		[]string{"member", "uniqueMember", "memberUid"},
		nil,
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		logger.Error(err, "Failed to search for group members")
		return err
	}

	var currentMembers []string
	if len(searchResult.Entries) > 0 {
		entry := searchResult.Entries[0]

		switch ldapGroup.Spec.GroupType {
		case openldapv1.GroupTypeGroupOfNames:
			currentMembers = entry.GetAttributeValues("member")
		case openldapv1.GroupTypeGroupOfUniqueNames:
			currentMembers = entry.GetAttributeValues("uniqueMember")
		case openldapv1.GroupTypePosix:
			currentMembers = entry.GetAttributeValues("memberUid")
		}

		// Filter out dummy members
		filteredMembers := make([]string, 0)
		for _, member := range currentMembers {
			if member != "cn=dummy" {
				filteredMembers = append(filteredMembers, member)
			}
		}
		currentMembers = filteredMembers
	}

	// Update status
	ldapGroup.Status.DN = groupDN
	ldapGroup.Status.Members = currentMembers
	// Safe conversion: member count is naturally bounded by practical LDAP limits
	memberCount := len(currentMembers)
	if memberCount > 2147483647 {
		return fmt.Errorf("member count exceeds int32 maximum")
	}
	ldapGroup.Status.MemberCount = int32(memberCount) // #nosec G115 - validated above

	logger.Info("Updated group status", "memberCount", ldapGroup.Status.MemberCount)
	return nil
}

// updateStatus updates the status of the LDAPGroup resource
func (r *LDAPGroupReconciler) updateStatus(ctx context.Context, ldapGroup *openldapv1.LDAPGroup, phase openldapv1.GroupPhase, message string) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("ldapgroup", ldapGroup.Name)

	ldapGroup.Status.Phase = phase
	ldapGroup.Status.Message = message
	now := metav1.Now()
	ldapGroup.Status.LastModified = &now
	ldapGroup.Status.ObservedGeneration = ldapGroup.Generation

	// Update condition
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: now,
		Reason:             string(phase),
		Message:            message,
	}

	if phase == openldapv1.GroupPhaseReady {
		condition.Status = metav1.ConditionTrue
	}

	// Update or add the condition
	updated := false
	for i, existingCondition := range ldapGroup.Status.Conditions {
		if existingCondition.Type == condition.Type {
			ldapGroup.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		ldapGroup.Status.Conditions = append(ldapGroup.Status.Conditions, condition)
	}

	logger.Info("Updating LDAPGroup status", "phase", phase, "message", message)

	// Retry status update on conflict
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get latest version of the resource
		latest := &openldapv1.LDAPGroup{}
		if err := r.Get(ctx, types.NamespacedName{Name: ldapGroup.Name, Namespace: ldapGroup.Namespace}, latest); err != nil {
			return err
		}

		// Update status fields on latest version
		latest.Status.Phase = phase
		latest.Status.Message = message
		latest.Status.ObservedGeneration = ldapGroup.Generation
		latest.Status.Conditions = ldapGroup.Status.Conditions
		latest.Status.Members = ldapGroup.Status.Members

		return r.Status().Update(ctx, latest)
	})

	if err != nil {
		logger.Error(err, "Failed to update LDAPGroup status")
		return ctrl.Result{}, err
	}

	if phase == openldapv1.GroupPhaseError || phase == openldapv1.GroupPhasePending {
		logger.Info("Requeuing due to error or pending state")
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	return ctrl.Result{}, nil
}

// getSecretValue retrieves a value from a Kubernetes secret
func (r *LDAPGroupReconciler) getSecretValue(ctx context.Context, namespace string, secretRef openldapv1.SecretReference) (string, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: namespace,
	}, secret)
	if err != nil {
		return "", err
	}

	value, exists := secret.Data[secretRef.Key]
	if !exists {
		return "", fmt.Errorf("key %s not found in secret %s", secretRef.Key, secretRef.Name)
	}

	return string(value), nil
}

// handleDeletion handles the deletion of an LDAPGroup resource
func (r *LDAPGroupReconciler) handleDeletion(ctx context.Context, ldapGroup *openldapv1.LDAPGroup) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("ldapgroup", ldapGroup.Name)
	logger.Info("Handling LDAPGroup deletion")

	// Get the referenced LDAP server
	ldapServer, err := r.getLDAPServer(ctx, ldapGroup)
	if err != nil {
		logger.Error(err, "Failed to get LDAP server during deletion, continuing with cleanup")
		// Continue with deletion even if we can't clean up LDAP
	} else {
		// Try to delete group from LDAP
		conn, err := r.connectToLDAP(ctx, ldapServer)
		if err != nil {
			logger.Error(err, "Failed to connect to LDAP during deletion, continuing with cleanup")
		} else {
			defer conn.Close()
			ou := ldapGroup.Spec.OrganizationalUnit
			if ou == "" {
				ou = "groups"
			}
			groupDN := fmt.Sprintf("cn=%s,ou=%s,%s", ldapGroup.Spec.GroupName, ou, ldapServer.Spec.BaseDN)

			logger.Info("Deleting group from LDAP", "dn", groupDN)
			delRequest := ldap.NewDelRequest(groupDN, nil)
			err = conn.Del(delRequest)
			if err != nil {
				logger.Error(err, "Failed to delete group from LDAP", "dn", groupDN)
			} else {
				logger.Info("Successfully deleted group from LDAP")
			}
		}
	}

	// Remove finalizer
	logger.Info("Removing finalizer from LDAPGroup")
	controllerutil.RemoveFinalizer(ldapGroup, "openldap.guided-traffic.com/finalizer")
	if err := r.Update(ctx, ldapGroup); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully handled LDAPGroup deletion")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LDAPGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openldapv1.LDAPGroup{}).
		Watches(
			&openldapv1.LDAPServer{},
			handler.EnqueueRequestsFromMapFunc(r.findGroupsForServer),
		).
		Complete(r)
}

// findGroupsForServer finds all LDAPGroups that reference a given LDAPServer
func (r *LDAPGroupReconciler) findGroupsForServer(ctx context.Context, server client.Object) []reconcile.Request {
	ldapServer, ok := server.(*openldapv1.LDAPServer)
	if !ok {
		return nil
	}

	// List all LDAPGroups
	groupList := &openldapv1.LDAPGroupList{}
	if err := r.List(ctx, groupList); err != nil {
		return nil
	}

	// Find groups that reference this server
	var requests []reconcile.Request
	for _, group := range groupList.Items {
		// Check if this group references the server
		if group.Spec.LDAPServerRef.Name == ldapServer.Name {
			// Check if they're in the same namespace or if namespace is not specified
			if group.Spec.LDAPServerRef.Namespace == "" && group.Namespace == ldapServer.Namespace {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      group.Name,
						Namespace: group.Namespace,
					},
				})
			} else if group.Spec.LDAPServerRef.Namespace == ldapServer.Namespace {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      group.Name,
						Namespace: group.Namespace,
					},
				})
			}
		}
	}

	return requests
}

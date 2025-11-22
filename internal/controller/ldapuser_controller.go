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
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
	ldapClient "github.com/guided-traffic/openldap-operator/internal/ldap"
)

const (
	// Default organizational unit names
	defaultUsersOU  = "users"
	defaultGroupsOU = "groups"
)

// LDAPUserReconciler reconciles a LDAPUser object
type LDAPUserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapusers/finalizers,verbs=update
//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapservers,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LDAPUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the LDAPUser instance
	ldapUser := &openldapv1.LDAPUser{}
	err := r.Get(ctx, req.NamespacedName, ldapUser)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("LDAPUser resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get LDAPUser")
		return ctrl.Result{}, err
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(ldapUser, "openldap.guided-traffic.com/finalizer") {
		controllerutil.AddFinalizer(ldapUser, "openldap.guided-traffic.com/finalizer")
		if err := r.Update(ctx, ldapUser); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle deletion
	if ldapUser.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, ldapUser)
	}

	// Get the referenced LDAP server
	ldapServer, err := r.getLDAPServer(ctx, ldapUser)
	if err != nil {
		return r.updateStatus(ctx, ldapUser, openldapv1.UserPhaseError, fmt.Sprintf("Failed to get LDAP server: %v", err))
	}

	// Check if LDAP server is connected
	if ldapServer.Status.ConnectionStatus != openldapv1.ConnectionStatusConnected {
		return r.updateStatus(ctx, ldapUser, openldapv1.UserPhasePending, "LDAP server is not connected")
	}

	// Connect to LDAP server
	conn, err := r.connectToLDAP(ctx, ldapServer)
	if err != nil {
		return r.updateStatus(ctx, ldapUser, openldapv1.UserPhaseError, fmt.Sprintf("Failed to connect to LDAP: %v", err))
	}
	defer conn.Close()

	// Create or update the user
	err = r.reconcileUser(ctx, conn, ldapServer, ldapUser)
	if err != nil {
		return r.updateStatus(ctx, ldapUser, openldapv1.UserPhaseError, fmt.Sprintf("Failed to reconcile user: %v", err))
	}

	// Reconcile user group memberships
	err = r.reconcileUserGroups(ctx, conn, ldapServer, ldapUser)
	if err != nil {
		return r.updateStatus(ctx, ldapUser, openldapv1.UserPhaseError, fmt.Sprintf("Failed to reconcile user groups: %v", err))
	}

	// Determine final status based on missing groups
	var finalPhase openldapv1.UserPhase
	var finalMessage string

	if len(ldapUser.Status.MissingGroups) > 0 {
		finalPhase = openldapv1.UserPhaseWarning
		finalMessage = fmt.Sprintf("User synchronized with warnings: %d missing groups (%s)",
			len(ldapUser.Status.MissingGroups),
			strings.Join(ldapUser.Status.MissingGroups, ", "))
	} else {
		finalPhase = openldapv1.UserPhaseReady
		finalMessage = "User successfully synchronized"
	}

	return r.updateStatus(ctx, ldapUser, finalPhase, finalMessage)
}

// getLDAPServer retrieves the referenced LDAP server
func (r *LDAPUserReconciler) getLDAPServer(ctx context.Context, ldapUser *openldapv1.LDAPUser) (*openldapv1.LDAPServer, error) {
	ldapServer := &openldapv1.LDAPServer{}
	namespace := ldapUser.Namespace
	if ldapUser.Spec.LDAPServerRef.Namespace != "" {
		namespace = ldapUser.Spec.LDAPServerRef.Namespace
	}

	err := r.Get(ctx, types.NamespacedName{
		Name:      ldapUser.Spec.LDAPServerRef.Name,
		Namespace: namespace,
	}, ldapServer)

	return ldapServer, err
}

// connectToLDAP establishes a connection to the LDAP server
func (r *LDAPUserReconciler) connectToLDAP(ctx context.Context, ldapServer *openldapv1.LDAPServer) (*ldap.Conn, error) {
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

// reconcileUser creates or updates the user in LDAP
func (r *LDAPUserReconciler) reconcileUser(ctx context.Context, conn *ldap.Conn, ldapServer *openldapv1.LDAPServer, ldapUser *openldapv1.LDAPUser) error {
	// Construct the user DN
	ou := ldapUser.Spec.OrganizationalUnit
	if ou == "" {
		ou = defaultUsersOU
	}
	userDN := fmt.Sprintf("uid=%s,ou=%s,%s", ldapUser.Spec.Username, ou, ldapServer.Spec.BaseDN)

	// Check if user exists
	searchRequest := ldap.NewSearchRequest(
		userDN,
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
	userExists := err == nil && len(searchResult.Entries) > 0

	if userExists {
		// Update existing user
		return r.updateLDAPUser(conn, userDN, ldapUser)
	} else {
		// Create new user
		return r.createLDAPUser(ctx, conn, userDN, ldapServer, ldapUser)
	}
}

// createLDAPUser creates a new user in LDAP
func (r *LDAPUserReconciler) createLDAPUser(ctx context.Context, conn *ldap.Conn, userDN string, ldapServer *openldapv1.LDAPServer, ldapUser *openldapv1.LDAPUser) error {
	addRequest := ldap.NewAddRequest(userDN, nil)

	// Basic attributes
	addRequest.Attribute("objectClass", []string{"inetOrgPerson", "posixAccount"})
	addRequest.Attribute("uid", []string{ldapUser.Spec.Username})
	addRequest.Attribute("cn", []string{ldapUser.Spec.Username})

	if ldapUser.Spec.FirstName != "" {
		addRequest.Attribute("givenName", []string{ldapUser.Spec.FirstName})
	}
	if ldapUser.Spec.LastName != "" {
		addRequest.Attribute("sn", []string{ldapUser.Spec.LastName})
	}
	if ldapUser.Spec.Email != "" {
		addRequest.Attribute("mail", []string{ldapUser.Spec.Email})
	}
	if ldapUser.Spec.DisplayName != "" {
		addRequest.Attribute("displayName", []string{ldapUser.Spec.DisplayName})
	}

	// POSIX attributes
	if ldapUser.Spec.UserID != nil {
		addRequest.Attribute("uidNumber", []string{fmt.Sprintf("%d", *ldapUser.Spec.UserID)})
	}
	if ldapUser.Spec.GroupID != nil {
		addRequest.Attribute("gidNumber", []string{fmt.Sprintf("%d", *ldapUser.Spec.GroupID)})
	}
	// Set homeDirectory - required for posixAccount, default to /home/<username> if not specified
	homeDir := ldapUser.Spec.HomeDirectory
	if homeDir == "" {
		homeDir = fmt.Sprintf("/home/%s", ldapUser.Spec.Username)
	}
	addRequest.Attribute("homeDirectory", []string{homeDir})

	// Update status with actual home directory
	ldapUser.Status.ActualHomeDirectory = homeDir
	if ldapUser.Spec.LoginShell != "" {
		addRequest.Attribute("loginShell", []string{ldapUser.Spec.LoginShell})
	}

	// Set password if provided
	if ldapUser.Spec.PasswordSecret != nil {
		password, err := r.getSecretValue(ctx, ldapUser.Namespace, *ldapUser.Spec.PasswordSecret)
		if err != nil {
			return fmt.Errorf("failed to get user password: %v", err)
		}
		addRequest.Attribute("userPassword", []string{password})
	}

	// Add any additional attributes
	for attr, values := range ldapUser.Spec.AdditionalAttributes {
		addRequest.Attribute(attr, values)
	}

	return conn.Add(addRequest)
}

// updateLDAPUser updates an existing user in LDAP
func (r *LDAPUserReconciler) updateLDAPUser(conn *ldap.Conn, userDN string, ldapUser *openldapv1.LDAPUser) error {
	modifyRequest := ldap.NewModifyRequest(userDN, nil)

	// Update basic attributes
	if ldapUser.Spec.FirstName != "" {
		modifyRequest.Replace("givenName", []string{ldapUser.Spec.FirstName})
	}
	if ldapUser.Spec.LastName != "" {
		modifyRequest.Replace("sn", []string{ldapUser.Spec.LastName})
	}
	if ldapUser.Spec.Email != "" {
		modifyRequest.Replace("mail", []string{ldapUser.Spec.Email})
	}
	if ldapUser.Spec.DisplayName != "" {
		modifyRequest.Replace("displayName", []string{ldapUser.Spec.DisplayName})
	}

	// Update home directory - default to /home/<username> if not specified
	homeDir := ldapUser.Spec.HomeDirectory
	if homeDir == "" {
		homeDir = fmt.Sprintf("/home/%s", ldapUser.Spec.Username)
	}
	modifyRequest.Replace("homeDirectory", []string{homeDir})

	// Update status with actual home directory
	ldapUser.Status.ActualHomeDirectory = homeDir

	return conn.Modify(modifyRequest)
}

// reconcileUserGroups manages the group membership for the user
func (r *LDAPUserReconciler) reconcileUserGroups(ctx context.Context, conn *ldap.Conn, ldapServer *openldapv1.LDAPServer, ldapUser *openldapv1.LDAPUser) error {
	// Get bind password to create LDAP client
	bindPassword, err := r.getSecretValue(ctx, ldapServer.Namespace, ldapServer.Spec.BindPasswordSecret)
	if err != nil {
		return fmt.Errorf("failed to get bind password: %v", err)
	}

	// Create LDAP client using the server spec
	client, err := ldapClient.NewClient(&ldapServer.Spec, bindPassword)
	if err != nil {
		return fmt.Errorf("failed to create LDAP client: %v", err)
	}
	defer client.Close()

	userOU := ldapUser.Spec.OrganizationalUnit
	if userOU == "" {
		userOU = defaultUsersOU
	}

	// Get current groups
	currentGroups := r.getCurrentGroups(ctx, client, ldapUser.Spec.Username, userOU)

	// Get desired groups from spec
	desiredGroups := ldapUser.Spec.Groups
	if desiredGroups == nil {
		desiredGroups = []string{}
	}

	// Categorize groups as existing or missing
	existingGroups, missingGroups := r.categorizeGroups(ctx, client, desiredGroups, ldapUser.Spec.Username)

	// Sync group memberships
	r.addUserToMissingGroups(ctx, client, ldapUser.Spec.Username, userOU, existingGroups, currentGroups)
	r.removeUserFromExtraGroups(ctx, client, ldapUser.Spec.Username, userOU, existingGroups, currentGroups)

	// Update status with current and missing groups
	ldapUser.Status.Groups = existingGroups
	ldapUser.Status.MissingGroups = missingGroups

	return nil
}

// getCurrentGroups retrieves current group memberships for a user
func (r *LDAPUserReconciler) getCurrentGroups(ctx context.Context, client *ldapClient.Client, username, userOU string) []string {
	logger := log.FromContext(ctx)
	currentGroups, err := client.GetUserGroups(username, userOU, defaultGroupsOU)
	if err != nil {
		logger.Error(err, "Failed to get current user groups")
		return []string{}
	}
	return currentGroups
}

// categorizeGroups separates desired groups into existing and missing
func (r *LDAPUserReconciler) categorizeGroups(ctx context.Context, client *ldapClient.Client, desiredGroups []string, username string) ([]string, []string) {
	logger := log.FromContext(ctx)
	var existingGroups, missingGroups []string

	for _, groupName := range desiredGroups {
		exists, err := client.GroupExists(groupName, defaultGroupsOU)
		if err != nil {
			logger.Error(err, "Failed to check if group exists", "group", groupName)
			continue
		}

		if exists {
			existingGroups = append(existingGroups, groupName)
		} else {
			missingGroups = append(missingGroups, groupName)
			logger.Info("Group does not exist in LDAP", "group", groupName, "user", username)
		}
	}

	return existingGroups, missingGroups
}

// addUserToMissingGroups adds user to groups they should be in but aren't
func (r *LDAPUserReconciler) addUserToMissingGroups(ctx context.Context, client *ldapClient.Client, username, userOU string, existingGroups, currentGroups []string) {
	logger := log.FromContext(ctx)

	for _, groupName := range existingGroups {
		if r.isInGroup(groupName, currentGroups) {
			continue
		}

		logger.Info("Adding user to group", "user", username, "group", groupName)
		r.tryAddUserToGroup(ctx, client, username, userOU, groupName)
	}
}

// removeUserFromExtraGroups removes user from groups they shouldn't be in
func (r *LDAPUserReconciler) removeUserFromExtraGroups(ctx context.Context, client *ldapClient.Client, username, userOU string, existingGroups, currentGroups []string) {
	logger := log.FromContext(ctx)

	for _, currentGroup := range currentGroups {
		if r.isInGroup(currentGroup, existingGroups) {
			continue
		}

		logger.Info("Removing user from group", "user", username, "group", currentGroup)
		r.tryRemoveUserFromGroup(ctx, client, username, userOU, currentGroup)
	}
}

// isInGroup checks if a group is in the list
func (r *LDAPUserReconciler) isInGroup(group string, groups []string) bool {
	for _, g := range groups {
		if g == group {
			return true
		}
	}
	return false
}

// tryAddUserToGroup attempts to add user to group with different types
func (r *LDAPUserReconciler) tryAddUserToGroup(ctx context.Context, client *ldapClient.Client, username, userOU, groupName string) {
	logger := log.FromContext(ctx)
	groupTypes := []openldapv1.GroupType{
		openldapv1.GroupTypeGroupOfNames,
		openldapv1.GroupTypePosix,
		openldapv1.GroupTypeGroupOfUniqueNames,
	}

	for _, gType := range groupTypes {
		err := client.AddUserToGroup(username, userOU, groupName, defaultGroupsOU, gType)
		if err == nil {
			logger.Info("Successfully added user to group", "user", username, "group", groupName, "type", gType)
			return
		}
	}
	logger.Error(nil, "Failed to add user to group with all types", "user", username, "group", groupName)
}

// tryRemoveUserFromGroup attempts to remove user from group with different types
func (r *LDAPUserReconciler) tryRemoveUserFromGroup(ctx context.Context, client *ldapClient.Client, username, userOU, groupName string) {
	logger := log.FromContext(ctx)
	groupTypes := []openldapv1.GroupType{
		openldapv1.GroupTypeGroupOfNames,
		openldapv1.GroupTypePosix,
		openldapv1.GroupTypeGroupOfUniqueNames,
	}

	for _, gType := range groupTypes {
		err := client.RemoveUserFromGroup(username, userOU, groupName, defaultGroupsOU, gType)
		if err == nil {
			logger.Info("Successfully removed user from group", "user", username, "group", groupName)
			return
		}
	}
}

// updateStatus updates the status of the LDAPUser resource
func (r *LDAPUserReconciler) updateStatus(ctx context.Context, ldapUser *openldapv1.LDAPUser, phase openldapv1.UserPhase, message string) (ctrl.Result, error) {
	ldapUser.Status.Phase = phase
	ldapUser.Status.Message = message
	now := metav1.Now()
	ldapUser.Status.LastModified = &now
	ldapUser.Status.ObservedGeneration = ldapUser.Generation

	// Update condition
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: now,
		Reason:             string(phase),
		Message:            message,
	}

	if phase == openldapv1.UserPhaseReady || phase == openldapv1.UserPhaseWarning {
		condition.Status = metav1.ConditionTrue
		if phase == openldapv1.UserPhaseWarning {
			condition.Type = "Ready"
			condition.Reason = "ReadyWithWarnings"
		}
	}

	// Update or add the condition
	updated := false
	for i, existingCondition := range ldapUser.Status.Conditions {
		if existingCondition.Type == condition.Type {
			ldapUser.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		ldapUser.Status.Conditions = append(ldapUser.Status.Conditions, condition)
	}

	if err := r.Status().Update(ctx, ldapUser); err != nil {
		return ctrl.Result{}, err
	}

	if phase == openldapv1.UserPhaseError || phase == openldapv1.UserPhasePending {
		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
	}

	return ctrl.Result{}, nil
}

// getSecretValue retrieves a value from a Kubernetes secret
func (r *LDAPUserReconciler) getSecretValue(ctx context.Context, namespace string, secretRef openldapv1.SecretReference) (string, error) {
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

// handleDeletion handles the deletion of an LDAPUser resource
func (r *LDAPUserReconciler) handleDeletion(ctx context.Context, ldapUser *openldapv1.LDAPUser) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the referenced LDAP server
	ldapServer, err := r.getLDAPServer(ctx, ldapUser)
	if err != nil {
		logger.Error(err, "Failed to get LDAP server during deletion")
		// Continue with deletion even if we can't clean up LDAP
	} else {
		// Try to delete user from LDAP
		conn, err := r.connectToLDAP(ctx, ldapServer)
		if err != nil {
			logger.Error(err, "Failed to connect to LDAP during deletion")
		} else {
			defer conn.Close()
			ou := ldapUser.Spec.OrganizationalUnit
			if ou == "" {
				ou = "users"
			}
			userDN := fmt.Sprintf("uid=%s,ou=%s,%s", ldapUser.Spec.Username, ou, ldapServer.Spec.BaseDN)

			delRequest := ldap.NewDelRequest(userDN, nil)
			err = conn.Del(delRequest)
			if err != nil {
				logger.Error(err, "Failed to delete user from LDAP", "dn", userDN)
			}
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(ldapUser, "openldap.guided-traffic.com/finalizer")
	if err := r.Update(ctx, ldapUser); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LDAPUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openldapv1.LDAPUser{}).
		Complete(r)
}

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

// LDAPServerReconciler reconciles a LDAPServer object
type LDAPServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapservers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapservers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=openldap.guided-traffic.com,resources=ldapservers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LDAPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the LDAPServer instance
	ldapServer := &openldapv1.LDAPServer{}
	err := r.Get(ctx, req.NamespacedName, ldapServer)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			logger.Info("LDAPServer resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get LDAPServer")
		return ctrl.Result{}, err
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(ldapServer, "openldap.guided-traffic.com/finalizer") {
		controllerutil.AddFinalizer(ldapServer, "openldap.guided-traffic.com/finalizer")
		if err := r.Update(ctx, ldapServer); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle deletion
	if ldapServer.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, ldapServer)
	}

	// Test connection to LDAP server
	connectionStatus, message, err := r.testConnection(ctx, ldapServer)
	if err != nil {
		logger.Error(err, "Failed to test LDAP connection")
	}

	// Update status
	ldapServer.Status.ConnectionStatus = connectionStatus
	ldapServer.Status.Message = message
	now := metav1.Now()
	ldapServer.Status.LastChecked = &now
	ldapServer.Status.ObservedGeneration = ldapServer.Generation

	// Update conditions
	condition := metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionFalse,
		LastTransitionTime: now,
		Reason:             "ConnectionFailed",
		Message:            message,
	}

	if connectionStatus == openldapv1.ConnectionStatusConnected {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "ConnectionSuccessful"
	}

	// Update or add the condition
	updated := false
	for i, existingCondition := range ldapServer.Status.Conditions {
		if existingCondition.Type == condition.Type {
			ldapServer.Status.Conditions[i] = condition
			updated = true
			break
		}
	}
	if !updated {
		ldapServer.Status.Conditions = append(ldapServer.Status.Conditions, condition)
	}

	if err := r.Status().Update(ctx, ldapServer); err != nil {
		logger.Error(err, "Failed to update LDAPServer status")
		return ctrl.Result{}, err
	}

	// Schedule next health check
	healthCheckInterval := 5 * time.Minute
	if ldapServer.Spec.HealthCheckInterval != nil {
		healthCheckInterval = ldapServer.Spec.HealthCheckInterval.Duration
	}

	return ctrl.Result{RequeueAfter: healthCheckInterval}, nil
}

// testConnection tests the connection to the LDAP server
func (r *LDAPServerReconciler) testConnection(ctx context.Context, ldapServer *openldapv1.LDAPServer) (openldapv1.ConnectionStatus, string, error) {
	// Get bind password from secret
	bindPassword, err := r.getSecretValue(ctx, ldapServer.Namespace, ldapServer.Spec.BindPasswordSecret)
	if err != nil {
		return openldapv1.ConnectionStatusError, fmt.Sprintf("Failed to get bind password: %v", err), err
	}

	// Create LDAP connection
	var conn *ldap.Conn
	address := fmt.Sprintf("%s:%d", ldapServer.Spec.Host, ldapServer.Spec.Port)

	if ldapServer.Spec.TLS != nil && ldapServer.Spec.TLS.Enabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: ldapServer.Spec.TLS.InsecureSkipVerify,
		}
		conn, err = ldap.DialTLS("tcp", address, tlsConfig)
	} else {
		conn, err = ldap.Dial("tcp", address)
	}

	if err != nil {
		return openldapv1.ConnectionStatusDisconnected, fmt.Sprintf("Failed to connect to LDAP server: %v", err), err
	}
	defer conn.Close()

	// Set timeout
	timeout := 30 * time.Second
	if ldapServer.Spec.ConnectionTimeout > 0 {
		timeout = time.Duration(ldapServer.Spec.ConnectionTimeout) * time.Second
	}
	conn.SetTimeout(timeout)

	// Attempt to bind
	err = conn.Bind(ldapServer.Spec.BindDN, bindPassword)
	if err != nil {
		return openldapv1.ConnectionStatusError, fmt.Sprintf("Failed to bind to LDAP server: %v", err), err
	}

	// Test search to ensure the connection is working
	searchRequest := ldap.NewSearchRequest(
		ldapServer.Spec.BaseDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1,
		int(timeout.Seconds()),
		false,
		"(objectClass=*)",
		[]string{"objectClass"},
		nil,
	)

	_, err = conn.Search(searchRequest)
	if err != nil {
		return openldapv1.ConnectionStatusError, fmt.Sprintf("Failed to perform test search: %v", err), err
	}

	return openldapv1.ConnectionStatusConnected, "Successfully connected to LDAP server", nil
}

// getSecretValue retrieves a value from a Kubernetes secret
func (r *LDAPServerReconciler) getSecretValue(ctx context.Context, namespace string, secretRef openldapv1.SecretReference) (string, error) {
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

// handleDeletion handles the deletion of an LDAPServer resource
func (r *LDAPServerReconciler) handleDeletion(ctx context.Context, ldapServer *openldapv1.LDAPServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Perform cleanup operations here if needed
	logger.Info("Cleaning up LDAPServer resource", "name", ldapServer.Name)

	// Remove finalizer
	controllerutil.RemoveFinalizer(ldapServer, "openldap.guided-traffic.com/finalizer")
	if err := r.Update(ctx, ldapServer); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LDAPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openldapv1.LDAPServer{}).
		Complete(r)
}

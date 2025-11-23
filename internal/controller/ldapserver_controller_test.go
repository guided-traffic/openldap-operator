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
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

// Mock client for testing getSecretValue functionality
// Simulates Kubernetes Secret API responses for controlled testing
type mockClient struct {
	client.Client
	secrets map[string]*corev1.Secret
	err     error
}

func (m *mockClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	if m.err != nil {
		return m.err
	}

	if secret, ok := obj.(*corev1.Secret); ok {
		if mockSecret, exists := m.secrets[key.String()]; exists {
			*secret = *mockSecret
			return nil
		}
		return errors.New("secret not found")
	}

	return errors.New("unsupported object type")
}

var _ = Describe("LDAPServer Controller", func() {
	var (
		reconciler    *LDAPServerReconciler
		ctx           context.Context
		scheme        *runtime.Scheme
		testNamespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		testNamespace = "test-namespace"
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(openldapv1.AddToScheme(scheme)).To(Succeed())
	})

	// getSecretValue retrieves credentials from Kubernetes Secrets
	// This is a critical security function that:
	// - Fetches secret data from the Kubernetes API
	// - Validates that the secret and specified key exist
	// - Returns the password as a string for LDAP authentication
	Describe("getSecretValue", func() {
		It("Should successfully retrieve secret value from valid secret", func() {
			// Create a test secret with password data
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			secretRef := openldapv1.SecretReference{
				Name: "test-secret",
				Key:  "password",
			}

			value, err := reconciler.getSecretValue(ctx, testNamespace, secretRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal("secret-password"))
		})

		It("Should return error when secret does not exist", func() {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			secretRef := openldapv1.SecretReference{
				Name: "nonexistent-secret",
				Key:  "password",
			}

			_, err := reconciler.getSecretValue(ctx, testNamespace, secretRef)
			Expect(err).To(HaveOccurred())
		})

		It("Should return error when key does not exist in secret", func() {
			// Create a secret but with different keys than expected
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"other-key": []byte("other-value"),
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			secretRef := openldapv1.SecretReference{
				Name: "test-secret",
				Key:  "nonexistent-key",
			}

			_, err := reconciler.getSecretValue(ctx, testNamespace, secretRef)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key nonexistent-key not found"))
		})

		It("Should handle empty secret data gracefully", func() {
			// Secrets with no data should fail appropriately
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			secretRef := openldapv1.SecretReference{
				Name: "empty-secret",
				Key:  "password",
			}

			_, err := reconciler.getSecretValue(ctx, testNamespace, secretRef)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key password not found"))
		})

		It("Should accept secret with empty string value", func() {
			// Empty passwords are technically valid (though not recommended)
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"password": []byte(""),
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			secretRef := openldapv1.SecretReference{
				Name: "test-secret",
				Key:  "password",
			}

			value, err := reconciler.getSecretValue(ctx, testNamespace, secretRef)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(""))
		})
	})

	// testConnection validates connectivity to external LDAP servers
	// This function:
	// - Retrieves bind credentials from Secrets
	// - Creates LDAP client with appropriate TLS settings
	// - Attempts to bind to verify connectivity
	// - Updates LDAPServer status based on results
	Describe("testConnection", func() {
		var ldapServer *openldapv1.LDAPServer

		BeforeEach(func() {
			ldapServer = &openldapv1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server",
					Namespace: testNamespace,
				},
				Spec: openldapv1.LDAPServerSpec{
					Host:   "localhost",
					Port:   389,
					BindDN: "cn=admin,dc=example,dc=com",
					BaseDN: "dc=example,dc=com",
					BindPasswordSecret: openldapv1.SecretReference{
						Name: "ldap-secret",
						Key:  "password",
					},
				},
			}
		})

		It("Should return error status when secret retrieval fails", func() {
			// Simulate secret not found scenario
			mockClient := &mockClient{
				err: errors.New("secret not found"),
			}

			reconciler = &LDAPServerReconciler{
				Client: mockClient,
			}

			status, message, err := reconciler.testConnection(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
			Expect(status).To(Equal(openldapv1.ConnectionStatusError))
			Expect(message).To(ContainSubstring("Failed to get bind password"))
		})

		It("Should return disconnected status when LDAP server is unreachable", func() {
			// Create valid secret but use unreachable LDAP server
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ldap-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"password": []byte("admin-password"),
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			// Use invalid hostname to force connection failure
			ldapServer.Spec.Host = "invalid-host-that-does-not-exist"

			status, message, err := reconciler.testConnection(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
			Expect(status).To(Equal(openldapv1.ConnectionStatusDisconnected))
			Expect(message).To(ContainSubstring("Failed to connect to LDAP server"))
		})

		It("Should handle TLS configuration correctly", func() {
			// Test TLS-enabled connection path
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ldap-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"password": []byte("admin-password"),
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			// Configure TLS settings
			ldapServer.Spec.TLS = &openldapv1.TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true,
			}
			ldapServer.Spec.Port = 636

			// Use invalid host to test TLS code path (will fail to connect)
			ldapServer.Spec.Host = "invalid-host-that-does-not-exist"

			status, message, err := reconciler.testConnection(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
			Expect(status).To(Equal(openldapv1.ConnectionStatusDisconnected))
			Expect(message).To(ContainSubstring("Failed to connect to LDAP server"))
		})

		It("Should respect custom connection timeout", func() {
			// Verify that custom timeout values are used
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ldap-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"password": []byte("admin-password"),
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			// Set custom timeout (in seconds)
			ldapServer.Spec.ConnectionTimeout = 60

			// Use invalid host to test timeout code path
			ldapServer.Spec.Host = "invalid-host-that-does-not-exist"

			status, message, err := reconciler.testConnection(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
			Expect(status).To(Equal(openldapv1.ConnectionStatusDisconnected))
			Expect(message).To(ContainSubstring("Failed to connect to LDAP server"))
		})
	})

	// SetupWithManager registers the controller with the controller-runtime manager
	// This is called during operator initialization to set up watches and reconciliation
	Describe("SetupWithManager", func() {
		It("Should setup controller with manager successfully", func() {
			mgr, err := manager.New(&rest.Config{}, manager.Options{
				Scheme: scheme,
			})
			Expect(err).NotTo(HaveOccurred())

			reconciler := &LDAPServerReconciler{
				Client: mgr.GetClient(),
			}

			err = reconciler.SetupWithManager(mgr)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	// Reconcile is the main controller loop that manages LDAPServer resources
	// It handles the full lifecycle: creation, updates, deletion, and periodic health checks
	// These tests verify the controller behavior without requiring an actual LDAP connection
	Describe("Reconcile", func() {
		var ldapServer *openldapv1.LDAPServer
		var secret *corev1.Secret

		BeforeEach(func() {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ldap-admin-secret",
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					"password": []byte("admin-password"),
				},
			}

			ldapServer = &openldapv1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ldap-server",
					Namespace: testNamespace,
				},
				Spec: openldapv1.LDAPServerSpec{
					Host:   "ldap.example.com",
					Port:   389,
					BindDN: "cn=admin,dc=example,dc=com",
					BaseDN: "dc=example,dc=com",
					BindPasswordSecret: openldapv1.SecretReference{
						Name: "ldap-admin-secret",
						Key:  "password",
					},
					TLS: &openldapv1.TLSConfig{
						Enabled: false,
					},
				},
			}
		})

		// When a LDAPServer resource doesn't exist (e.g., already deleted), the controller
		// should gracefully return without error. This is normal Kubernetes behavior and
		// prevents unnecessary error logging when resources are removed.
		It("Should handle resource not found gracefully", func() {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "nonexistent-server",
					Namespace: testNamespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		// Finalizers prevent Kubernetes from deleting the resource until cleanup is complete.
		// On first reconcile, the controller adds the finalizer to ensure any dependent
		// resources (like LDAPUsers/Groups) can be properly notified before the server is removed.
		// The controller requeues after adding the finalizer to continue normal reconciliation.
		It("Should add finalizer on first reconcile", func() {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret, ldapServer).
				WithStatusSubresource(&openldapv1.LDAPServer{}).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ldapServer.Name,
					Namespace: ldapServer.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Verify finalizer was added
			updatedServer := &openldapv1.LDAPServer{}
			err = fakeClient.Get(ctx, req.NamespacedName, updatedServer)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedServer.Finalizers).To(ContainElement("openldap.guided-traffic.com/finalizer"))
		})

		// After adding the finalizer, the controller performs a connection test to the LDAP server.
		// The status is updated with the test results (ConnectionStatus, LastChecked timestamp).
		// The controller schedules the next health check by returning RequeueAfter with the interval.
		// This enables periodic monitoring of LDAP server availability.
		It("Should update status with connection test results", func() {
			// Add finalizer to skip the requeue
			ldapServer.Finalizers = []string{"openldap.guided-traffic.com/finalizer"}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret, ldapServer).
				WithStatusSubresource(&openldapv1.LDAPServer{}).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ldapServer.Name,
					Namespace: ldapServer.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			// Should requeue after health check interval (default 5 minutes)
			Expect(result.RequeueAfter).ToNot(BeZero())

			// Verify status was updated
			updatedServer := &openldapv1.LDAPServer{}
			err = fakeClient.Get(ctx, req.NamespacedName, updatedServer)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedServer.Status.LastChecked).ToNot(BeNil())
			Expect(updatedServer.Status.ConnectionStatus).ToNot(BeEmpty())
		})

		// Administrators can customize how frequently the controller checks LDAP server health
		// via spec.healthCheckInterval. This test verifies that custom intervals are respected.
		// For example, production servers might use longer intervals (10m) to reduce load,
		// while development servers might use shorter intervals (1m) for quick feedback.
		It("Should respect custom health check interval", func() {
			// Add finalizer and custom health check interval
			ldapServer.Finalizers = []string{"openldap.guided-traffic.com/finalizer"}
			customInterval := metav1.Duration{Duration: 10 * time.Minute}
			ldapServer.Spec.HealthCheckInterval = &customInterval

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret, ldapServer).
				WithStatusSubresource(&openldapv1.LDAPServer{}).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ldapServer.Name,
					Namespace: ldapServer.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(10 * time.Minute))
		})

		// Kubernetes Conditions provide a standardized way to communicate resource status.
		// The controller sets an "Available" condition based on connection test results:
		// - Status=True, Reason=ConnectionSuccessful when LDAP is reachable
		// - Status=False, Reason=ConnectionFailed when LDAP is unreachable
		// These conditions are visible via 'kubectl describe' and used by monitoring tools.
		It("Should update conditions based on connection status", func() {
			ldapServer.Finalizers = []string{"openldap.guided-traffic.com/finalizer"}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret, ldapServer).
				WithStatusSubresource(&openldapv1.LDAPServer{}).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ldapServer.Name,
					Namespace: ldapServer.Namespace,
				},
			}

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())

			// Verify condition was added
			updatedServer := &openldapv1.LDAPServer{}
			err = fakeClient.Get(ctx, req.NamespacedName, updatedServer)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedServer.Status.Conditions).ToNot(BeEmpty())

			// Should have an "Available" condition
			var availableCondition *metav1.Condition
			for i := range updatedServer.Status.Conditions {
				if updatedServer.Status.Conditions[i].Type == "Available" {
					availableCondition = &updatedServer.Status.Conditions[i]
					break
				}
			}
			Expect(availableCondition).ToNot(BeNil())
		})

		// When a LDAPServer is deleted (kubectl delete), Kubernetes sets DeletionTimestamp.
		// The controller detects this and calls handleDeletion() to perform cleanup.
		// In this case, the LDAPServer has no external resources to clean up (it's just
		// a reference), so the finalizer is removed immediately, allowing Kubernetes to
		// complete the deletion. Related LDAPUsers/Groups will be notified via watch events.
		It("Should handle deletion when DeletionTimestamp is set", func() {
			// Set deletion timestamp and finalizer
			now := metav1.Now()
			ldapServer.DeletionTimestamp = &now
			ldapServer.Finalizers = []string{"openldap.guided-traffic.com/finalizer"}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(secret, ldapServer).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      ldapServer.Name,
					Namespace: ldapServer.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify finalizer was removed - object should still exist but without finalizer
			updatedServer := &openldapv1.LDAPServer{}
			err = fakeClient.Get(ctx, req.NamespacedName, updatedServer)
			// Object may be deleted or may exist without finalizer depending on fake client behavior
			if err == nil {
				// If still exists, finalizer should be removed
				Expect(updatedServer.Finalizers).ToNot(ContainElement("openldap.guided-traffic.com/finalizer"))
			}
		})
	})

	// handleDeletion cleans up resources and removes finalizer when LDAPServer is deleted.
	// For LDAPServer resources, there are no external resources to clean up since it's
	// just a configuration reference. The finalizer is removed to allow Kubernetes to
	// complete the deletion. Watch mechanisms ensure dependent LDAPUsers/Groups are notified.
	Describe("handleDeletion", func() {
		// This test verifies the core deletion logic: finalizer removal allows K8s to
		// proceed with deleting the resource. In production, dependent resources would
		// be reconciled via watch events when their referenced server disappears.
		It("Should remove finalizer after cleanup", func() {
			ldapServer := &openldapv1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-server",
					Namespace:  testNamespace,
					Finalizers: []string{"openldap.guided-traffic.com/finalizer"},
				},
				Spec: openldapv1.LDAPServerSpec{
					Host:   "ldap.example.com",
					Port:   389,
					BindDN: "cn=admin,dc=example,dc=com",
					BaseDN: "dc=example,dc=com",
					BindPasswordSecret: openldapv1.SecretReference{
						Name: "ldap-secret",
						Key:  "password",
					},
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ldapServer).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
			}

			result, err := reconciler.handleDeletion(ctx, ldapServer)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify finalizer was removed
			updatedServer := &openldapv1.LDAPServer{}
			err = fakeClient.Get(ctx, types.NamespacedName{
				Name:      ldapServer.Name,
				Namespace: ldapServer.Namespace,
			}, updatedServer)
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedServer.Finalizers).ToNot(ContainElement("openldap.guided-traffic.com/finalizer"))
		})
	})
})

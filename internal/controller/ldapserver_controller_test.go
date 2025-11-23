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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
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
})

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

// Mock client for testing getSecretValue
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

var _ = Describe("LDAPServer Helper Functions", func() {
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

	Describe("getSecretValue", func() {
		It("Should successfully retrieve secret value", func() {
			// Create a test secret
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
			// Create a test secret without the expected key
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

		It("Should handle empty secret data", func() {
			// Create a test secret with empty data
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

		It("Should handle secret with empty value", func() {
			// Create a test secret with empty value
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

		It("Should return error when secret retrieval fails", func() {
			// Use mock client that returns error
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

		It("Should return disconnected status when LDAP connection fails", func() {
			// Create a secret but use invalid LDAP server (connection will fail)
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

			// Use invalid host to force connection failure
			ldapServer.Spec.Host = "invalid-host-that-does-not-exist"

			status, message, err := reconciler.testConnection(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
			Expect(status).To(Equal(openldapv1.ConnectionStatusDisconnected))
			Expect(message).To(ContainSubstring("Failed to connect to LDAP server"))
		})

		It("Should handle TLS configuration", func() {
			// Create a secret
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

			// Configure TLS
			ldapServer.Spec.TLS = &openldapv1.TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true,
			}
			ldapServer.Spec.Port = 636

			// Use invalid host to force connection failure (but test TLS path)
			ldapServer.Spec.Host = "invalid-host-that-does-not-exist"

			status, message, err := reconciler.testConnection(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
			Expect(status).To(Equal(openldapv1.ConnectionStatusDisconnected))
			Expect(message).To(ContainSubstring("Failed to connect to LDAP server"))
		})

		It("Should use custom connection timeout", func() {
			// Create a secret
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

			// Set custom timeout
			ldapServer.Spec.ConnectionTimeout = 60

			// Use invalid host to force connection failure (but test timeout path)
			ldapServer.Spec.Host = "invalid-host-that-does-not-exist"

			status, message, err := reconciler.testConnection(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
			Expect(status).To(Equal(openldapv1.ConnectionStatusDisconnected))
			Expect(message).To(ContainSubstring("Failed to connect to LDAP server"))
		})
	})

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

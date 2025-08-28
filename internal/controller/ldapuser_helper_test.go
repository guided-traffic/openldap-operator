package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

var _ = Describe("LDAPUser Helper Functions", func() {
	var (
		reconciler    *LDAPUserReconciler
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

	Describe("getLDAPServer", func() {
		It("Should successfully retrieve LDAP server from same namespace", func() {
			ldapServer := &openldapv1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ldap-server",
					Namespace: testNamespace,
				},
				Spec: openldapv1.LDAPServerSpec{
					Host:   "ldap.example.com",
					Port:   389,
					BindDN: "cn=admin,dc=example,dc=com",
					BaseDN: "dc=example,dc=com",
				},
			}

			ldapUser := &openldapv1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: testNamespace,
				},
				Spec: openldapv1.LDAPUserSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: "test-ldap-server",
					},
					Username: "testuser",
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ldapServer).
				Build()

			reconciler = &LDAPUserReconciler{
				Client: fakeClient,
			}

			retrievedServer, err := reconciler.getLDAPServer(ctx, ldapUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrievedServer.Name).To(Equal("test-ldap-server"))
			Expect(retrievedServer.Spec.Host).To(Equal("ldap.example.com"))
		})

		It("Should retrieve LDAP server from different namespace", func() {
			ldapServerNamespace := "ldap-namespace"

			ldapServer := &openldapv1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ldap-server",
					Namespace: ldapServerNamespace,
				},
				Spec: openldapv1.LDAPServerSpec{
					Host:   "ldap.example.com",
					Port:   389,
					BindDN: "cn=admin,dc=example,dc=com",
					BaseDN: "dc=example,dc=com",
				},
			}

			ldapUser := &openldapv1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: testNamespace,
				},
				Spec: openldapv1.LDAPUserSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name:      "test-ldap-server",
						Namespace: ldapServerNamespace,
					},
					Username: "testuser",
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ldapServer).
				Build()

			reconciler = &LDAPUserReconciler{
				Client: fakeClient,
			}

			retrievedServer, err := reconciler.getLDAPServer(ctx, ldapUser)
			Expect(err).ToNot(HaveOccurred())
			Expect(retrievedServer.Name).To(Equal("test-ldap-server"))
			Expect(retrievedServer.Namespace).To(Equal(ldapServerNamespace))
		})

		It("Should return error when LDAP server does not exist", func() {
			ldapUser := &openldapv1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: testNamespace,
				},
				Spec: openldapv1.LDAPUserSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: "nonexistent-server",
					},
					Username: "testuser",
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler = &LDAPUserReconciler{
				Client: fakeClient,
			}

			_, err := reconciler.getLDAPServer(ctx, ldapUser)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("getSecretValue", func() {
		It("Should successfully retrieve secret value", func() {
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

			reconciler = &LDAPUserReconciler{
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

			reconciler = &LDAPUserReconciler{
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

			reconciler = &LDAPUserReconciler{
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
	})

	Describe("updateStatus", func() {
		It("Should call updateStatus and handle K8s client operations", func() {
			ldapUser := &openldapv1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-user",
					Namespace:  testNamespace,
					Generation: 1,
				},
				Spec: openldapv1.LDAPUserSpec{
					Username: "testuser",
				},
				Status: openldapv1.LDAPUserStatus{
					Phase: openldapv1.UserPhasePending,
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ldapUser).
				WithStatusSubresource(ldapUser).
				Build()

			reconciler = &LDAPUserReconciler{
				Client: fakeClient,
			}

			result, err := reconciler.updateStatus(ctx, ldapUser, openldapv1.UserPhaseReady, "User ready")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Check that status fields were updated
			Expect(ldapUser.Status.Phase).To(Equal(openldapv1.UserPhaseReady))
			Expect(ldapUser.Status.Message).To(Equal("User ready"))
			Expect(ldapUser.Status.LastModified).NotTo(BeNil())
			Expect(ldapUser.Status.ObservedGeneration).To(Equal(int64(1)))
		})

		It("Should return requeue for error and pending phases", func() {
			ldapUser := &openldapv1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: testNamespace,
				},
				Spec: openldapv1.LDAPUserSpec{
					Username: "testuser",
				},
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ldapUser).
				WithStatusSubresource(ldapUser).
				Build()

			reconciler = &LDAPUserReconciler{
				Client: fakeClient,
			}

			result, err := reconciler.updateStatus(ctx, ldapUser, openldapv1.UserPhaseError, "Test error")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute * 5))
		})
	})

	Describe("connectToLDAP", func() {
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
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler = &LDAPUserReconciler{
				Client: fakeClient,
			}

			_, err := reconciler.connectToLDAP(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Or(
				ContainSubstring("not found"),
				ContainSubstring("connect: connection refused"),
				ContainSubstring("Network Error"),
			))
		})

		It("Should return error when LDAP connection fails", func() {
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

			reconciler = &LDAPUserReconciler{
				Client: fakeClient,
			}

			// Use invalid host to force connection failure
			ldapServer.Spec.Host = "invalid-host-that-does-not-exist"

			_, err := reconciler.connectToLDAP(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("SetupWithManager", func() {
		It("Should setup controller with manager successfully", func() {
			mgr, err := manager.New(&rest.Config{}, manager.Options{
				Scheme: scheme,
			})
			Expect(err).NotTo(HaveOccurred())

			reconciler := &LDAPUserReconciler{
				Client: mgr.GetClient(),
			}

			err = reconciler.SetupWithManager(mgr)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

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
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

var _ = Describe("LDAPUser Controller", func() {
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

	// getLDAPServer retrieves the LDAPServer resource referenced by the LDAPUser
	// This function supports cross-namespace references, allowing users to reference
	// shared LDAP servers in central namespaces (e.g., infrastructure namespace)
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

		It("Should retrieve LDAP server from different namespace via cross-namespace reference", func() {
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

		It("Should return error when referenced LDAP server does not exist", func() {
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

	// getSecretValue retrieves credentials from Kubernetes Secrets
	// Used for sensitive user data like initial passwords or SSH keys
	Describe("getSecretValue", func() {
		It("Should successfully retrieve secret value from valid secret", func() {
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

	// updateStatus updates the LDAPUser status with phase, message and timestamps
	// Implements automatic requeue logic for Error and Pending phases
	Describe("updateStatus", func() {
		It("Should update status fields correctly and persist to Kubernetes API", func() {
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

			// Verify all status fields were updated correctly
			Expect(ldapUser.Status.Phase).To(Equal(openldapv1.UserPhaseReady))
			Expect(ldapUser.Status.Message).To(Equal("User ready"))
			Expect(ldapUser.Status.LastModified).NotTo(BeNil())
			Expect(ldapUser.Status.ObservedGeneration).To(Equal(int64(1)))
		})

		It("Should automatically requeue for Error and Pending phases after 5 minutes", func() {
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

	// connectToLDAP establishes connection to the external LDAP server
	// Creates an LDAP client with proper authentication and TLS configuration
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

		It("Should return error when bind password secret is not found", func() {
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

		It("Should return error when LDAP server is unreachable", func() {
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

			// Force connection failure with invalid host
			ldapServer.Spec.Host = "invalid-host-that-does-not-exist"

			_, err := reconciler.connectToLDAP(ctx, ldapServer)
			Expect(err).To(HaveOccurred())
		})
	})

	// SetupWithManager registers the controller with the controller-runtime manager
	// Configures watches for LDAPUser resources
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

// TestLDAPUserStatus_MissingGroups tests the group membership tracking in user status
// The controller tracks which groups exist and which are missing in LDAP
// This allows users to be synced even when some groups don't exist yet
func TestLDAPUserStatus_MissingGroups(t *testing.T) {
	t.Run("Should track missing groups in status", func(t *testing.T) {
		user := &openldapv1.LDAPUser{
			Spec: openldapv1.LDAPUserSpec{
				Groups: []string{"group1", "group2", "group3"},
			},
			Status: openldapv1.LDAPUserStatus{
				Groups:        []string{"group1"},           // Only group1 exists
				MissingGroups: []string{"group2", "group3"}, // group2 and group3 are missing
			},
		}

		// Verify the status correctly reflects existing and missing groups
		assert.Equal(t, []string{"group1"}, user.Status.Groups)
		assert.Equal(t, []string{"group2", "group3"}, user.Status.MissingGroups)
		assert.Len(t, user.Status.Groups, 1)
		assert.Len(t, user.Status.MissingGroups, 2)
	})

	t.Run("Should handle no missing groups", func(t *testing.T) {
		user := &openldapv1.LDAPUser{
			Spec: openldapv1.LDAPUserSpec{
				Groups: []string{"group1", "group2"},
			},
			Status: openldapv1.LDAPUserStatus{
				Groups:        []string{"group1", "group2"}, // All groups exist
				MissingGroups: []string{},                   // No missing groups
			},
		}

		assert.Equal(t, []string{"group1", "group2"}, user.Status.Groups)
		assert.Empty(t, user.Status.MissingGroups)
	})

	t.Run("Should handle all missing groups", func(t *testing.T) {
		user := &openldapv1.LDAPUser{
			Spec: openldapv1.LDAPUserSpec{
				Groups: []string{"group1", "group2"},
			},
			Status: openldapv1.LDAPUserStatus{
				Groups:        []string{},                   // No groups exist
				MissingGroups: []string{"group1", "group2"}, // All groups are missing
			},
		}

		assert.Empty(t, user.Status.Groups)
		assert.Equal(t, []string{"group1", "group2"}, user.Status.MissingGroups)
	})
}

// TestLDAPUserStatus_WarningPhase tests the Warning phase behavior
// Users enter Warning phase when they are synced but some groups are missing
// This allows partial synchronization while alerting administrators to missing groups
func TestLDAPUserStatus_WarningPhase(t *testing.T) {
	t.Run("Should set Warning phase when groups are missing", func(t *testing.T) {
		user := &openldapv1.LDAPUser{
			Spec: openldapv1.LDAPUserSpec{
				Username: "testuser",
				Groups:   []string{"existing-group", "missing-group"},
			},
			Status: openldapv1.LDAPUserStatus{
				Phase:         openldapv1.UserPhaseWarning,
				Message:       "User synchronized with warnings: 1 missing groups (missing-group)",
				Groups:        []string{"existing-group"},
				MissingGroups: []string{"missing-group"},
			},
		}

		// Verify Warning phase is correctly set with appropriate message
		assert.Equal(t, openldapv1.UserPhaseWarning, user.Status.Phase)
		assert.Contains(t, user.Status.Message, "missing groups")
		assert.Contains(t, user.Status.Message, "missing-group")
		assert.Len(t, user.Status.MissingGroups, 1)
	})

	t.Run("Should set Ready phase when no groups are missing", func(t *testing.T) {
		user := &openldapv1.LDAPUser{
			Spec: openldapv1.LDAPUserSpec{
				Username: "testuser",
				Groups:   []string{"group1", "group2"},
			},
			Status: openldapv1.LDAPUserStatus{
				Phase:         openldapv1.UserPhaseReady,
				Message:       "User successfully synchronized",
				Groups:        []string{"group1", "group2"},
				MissingGroups: []string{},
			},
		}

		// Ready phase should be used when all groups exist
		assert.Equal(t, openldapv1.UserPhaseReady, user.Status.Phase)
		assert.Equal(t, "User successfully synchronized", user.Status.Message)
		assert.Empty(t, user.Status.MissingGroups)
	})

	t.Run("Should handle Warning phase with multiple missing groups", func(t *testing.T) {
		user := &openldapv1.LDAPUser{
			Spec: openldapv1.LDAPUserSpec{
				Username: "testuser",
				Groups:   []string{"existing1", "missing1", "missing2", "existing2"},
			},
			Status: openldapv1.LDAPUserStatus{
				Phase:         openldapv1.UserPhaseWarning,
				Message:       "User synchronized with warnings: 2 missing groups (missing1, missing2)",
				Groups:        []string{"existing1", "existing2"},
				MissingGroups: []string{"missing1", "missing2"},
			},
		}

		// Multiple missing groups should be listed in the message
		assert.Equal(t, openldapv1.UserPhaseWarning, user.Status.Phase)
		assert.Contains(t, user.Status.Message, "2 missing groups")
		assert.Contains(t, user.Status.Message, "missing1, missing2")
		assert.Len(t, user.Status.Groups, 2)
		assert.Len(t, user.Status.MissingGroups, 2)
	})
}

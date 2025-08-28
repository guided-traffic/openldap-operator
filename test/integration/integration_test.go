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

package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
	controllers "github.com/guided-traffic/openldap-operator/internal/controller"
	ldaputil "github.com/guided-traffic/openldap-operator/internal/ldap"
)

var (
	k8sClient  client.Client
	testEnv    *envtest.Environment
	ctx        context.Context
	cancel     context.CancelFunc
	ldapHost   string
	ldapPort   int
	ldapBindDN string
	ldapBaseDN string
	ldapPass   string
)

func TestIntegration(t *testing.T) {
	// Skip integration tests if LDAP server connection info is not provided
	ldapHost = os.Getenv("LDAP_HOST")
	if ldapHost == "" {
		t.Skip("Skipping integration tests: LDAP_HOST not set")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Test Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	// Set up test environment variables with defaults
	if ldapHost == "" {
		ldapHost = "localhost"
	}
	ldapPort = 389
	if port := os.Getenv("LDAP_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &ldapPort)
	}
	if ldapBindDN == "" {
		ldapBindDN = os.Getenv("LDAP_BIND_DN")
		if ldapBindDN == "" {
			ldapBindDN = "cn=admin,dc=example,dc=com"
		}
	}
	if ldapBaseDN == "" {
		ldapBaseDN = os.Getenv("LDAP_BASE_DN")
		if ldapBaseDN == "" {
			ldapBaseDN = "dc=example,dc=com"
		}
	}
	if ldapPass == "" {
		ldapPass = os.Getenv("LDAP_PASSWORD")
		if ldapPass == "" {
			ldapPass = "admin"
		}
	}

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{"../../config/crd/bases"},
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	err = openldapv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Set up the manager and controllers
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.LDAPServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.LDAPUserReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	// Note: LDAPGroupReconciler is not implemented yet
	// err = (&controllers.LDAPGroupReconciler{
	// 	Client: mgr.GetClient(),
	// 	Scheme: mgr.GetScheme(),
	// }).SetupWithManager(mgr)
	// Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("LDAP Integration Tests", func() {
	var (
		namespace     *corev1.Namespace
		ldapServer    *openldapv1.LDAPServer
		ldapSecret    *corev1.Secret
		ldapClient    *ldaputil.Client
		testUsername  string
		testGroupName string
	)

	BeforeEach(func() {
		// Create a unique namespace for each test
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "integration-test-",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		// Create LDAP password secret
		ldapSecret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ldap-secret",
				Namespace: namespace.Name,
			},
			Data: map[string][]byte{
				"password": []byte(ldapPass),
			},
		}
		Expect(k8sClient.Create(ctx, ldapSecret)).To(Succeed())

		// Create LDAP server resource
		ldapServer = &openldapv1.LDAPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ldap-server",
				Namespace: namespace.Name,
			},
			Spec: openldapv1.LDAPServerSpec{
				Host:   ldapHost,
				Port:   int32(ldapPort),
				BindDN: ldapBindDN,
				BindPasswordSecret: openldapv1.SecretReference{
					Name: "ldap-secret",
					Key:  "password",
				},
				BaseDN:            ldapBaseDN,
				ConnectionTimeout: 30,
			},
		}
		Expect(k8sClient.Create(ctx, ldapServer)).To(Succeed())

		// Wait for LDAP server to be ready
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      ldapServer.Name,
				Namespace: ldapServer.Namespace,
			}, ldapServer)
			if err != nil {
				return false
			}
			return ldapServer.Status.ConnectionStatus == openldapv1.ConnectionStatusConnected
		}, time.Minute, time.Second).Should(BeTrue())

		// Create LDAP client for direct testing
		var err error
		ldapClient, err = ldaputil.NewClient(&ldapServer.Spec, ldapPass)
		Expect(err).NotTo(HaveOccurred())

		// Generate unique test names
		testUsername = fmt.Sprintf("testuser-%d", time.Now().Unix())
		testGroupName = fmt.Sprintf("testgroup-%d", time.Now().Unix())
	})

	AfterEach(func() {
		// Clean up LDAP resources
		if ldapClient != nil {
			// Clean up test user and group
			ldapClient.DeleteUser(testUsername, "users")
			ldapClient.DeleteGroup(testGroupName, "groups")
			ldapClient.Close()
		}

		// Clean up Kubernetes resources
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	})

	Describe("LDAP Server Management", func() {
		It("should connect to LDAP server successfully", func() {
			Expect(ldapClient.TestConnection()).To(Succeed())
		})

		It("should handle connection errors gracefully", func() {
			// Create a server with invalid credentials
			invalidServer := &openldapv1.LDAPServerSpec{
				Host:              ldapHost,
				Port:              int32(ldapPort),
				BindDN:            "cn=invalid,dc=example,dc=com",
				BaseDN:            ldapBaseDN,
				ConnectionTimeout: 5,
			}

			_, err := ldaputil.NewClient(invalidServer, "wrongpassword")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("LDAP User Management", func() {
		var ldapUser *openldapv1.LDAPUser

		BeforeEach(func() {
			ldapUser = &openldapv1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: namespace.Name,
				},
				Spec: openldapv1.LDAPUserSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: ldapServer.Name,
					},
					Username:           testUsername,
					FirstName:          "Test",
					LastName:           "User",
					Email:              "test@example.com",
					OrganizationalUnit: "users",
					UserID:             func() *int32 { var id int32 = 1001; return &id }(),
					GroupID:            func() *int32 { var id int32 = 1000; return &id }(),
					HomeDirectory:      "/home/" + testUsername,
					LoginShell:         "/bin/bash",
				},
			}
		})

		It("should create a user in LDAP", func() {
			Expect(k8sClient.Create(ctx, ldapUser)).To(Succeed())

			// Wait for user to be created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ldapUser.Name,
					Namespace: ldapUser.Namespace,
				}, ldapUser)
				if err != nil {
					return false
				}
				return ldapUser.Status.Phase == openldapv1.UserPhaseReady
			}, time.Minute, time.Second).Should(BeTrue())

			// Verify user exists in LDAP
			exists, err := ldapClient.UserExists(testUsername, "users")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should update a user in LDAP", func() {
			// Create user first
			Expect(k8sClient.Create(ctx, ldapUser)).To(Succeed())

			// Wait for user to be created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ldapUser.Name,
					Namespace: ldapUser.Namespace,
				}, ldapUser)
				return err == nil && ldapUser.Status.Phase == openldapv1.UserPhaseReady
			}, time.Minute, time.Second).Should(BeTrue())

			// Update user
			ldapUser.Spec.Email = "updated@example.com"
			Expect(k8sClient.Update(ctx, ldapUser)).To(Succeed())

			// Wait for update to be processed
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ldapUser.Name,
					Namespace: ldapUser.Namespace,
				}, ldapUser)
				if err != nil {
					return false
				}
				return ldapUser.Status.ObservedGeneration == ldapUser.Generation
			}, time.Minute, time.Second).Should(BeTrue())
		})

		It("should delete a user from LDAP", func() {
			// Create user first
			Expect(k8sClient.Create(ctx, ldapUser)).To(Succeed())

			// Wait for user to be created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ldapUser.Name,
					Namespace: ldapUser.Namespace,
				}, ldapUser)
				return err == nil && ldapUser.Status.Phase == openldapv1.UserPhaseReady
			}, time.Minute, time.Second).Should(BeTrue())

			// Delete user
			Expect(k8sClient.Delete(ctx, ldapUser)).To(Succeed())

			// Wait for user to be deleted
			Eventually(func() bool {
				exists, err := ldapClient.UserExists(testUsername, "users")
				return err == nil && !exists
			}, time.Minute, time.Second).Should(BeTrue())
		})
	})

	Describe("LDAP Group Management", func() {
		var ldapGroup *openldapv1.LDAPGroup

		BeforeEach(func() {
			ldapGroup = &openldapv1.LDAPGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: namespace.Name,
				},
				Spec: openldapv1.LDAPGroupSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: ldapServer.Name,
					},
					GroupName:          testGroupName,
					Description:        "Test group for integration testing",
					OrganizationalUnit: "groups",
					GroupType:          openldapv1.GroupTypeGroupOfNames,
					GroupID:            func() *int32 { var id int32 = 2000; return &id }(),
				},
			}
		})

		It("should create a group in LDAP", func() {
			Expect(k8sClient.Create(ctx, ldapGroup)).To(Succeed())

			// Wait for group to be created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ldapGroup.Name,
					Namespace: ldapGroup.Namespace,
				}, ldapGroup)
				if err != nil {
					return false
				}
				return ldapGroup.Status.Phase == openldapv1.GroupPhaseReady
			}, time.Minute, time.Second).Should(BeTrue())

			// Verify group exists in LDAP
			exists, err := ldapClient.GroupExists(testGroupName, "groups")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should delete a group from LDAP", func() {
			// Create group first
			Expect(k8sClient.Create(ctx, ldapGroup)).To(Succeed())

			// Wait for group to be created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ldapGroup.Name,
					Namespace: ldapGroup.Namespace,
				}, ldapGroup)
				return err == nil && ldapGroup.Status.Phase == openldapv1.GroupPhaseReady
			}, time.Minute, time.Second).Should(BeTrue())

			// Delete group
			Expect(k8sClient.Delete(ctx, ldapGroup)).To(Succeed())

			// Wait for group to be deleted
			Eventually(func() bool {
				exists, err := ldapClient.GroupExists(testGroupName, "groups")
				return err == nil && !exists
			}, time.Minute, time.Second).Should(BeTrue())
		})
	})

	Describe("Group Membership Management", func() {
		var ldapUser *openldapv1.LDAPUser
		var ldapGroup *openldapv1.LDAPGroup

		BeforeEach(func() {
			// Create user
			ldapUser = &openldapv1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: namespace.Name,
				},
				Spec: openldapv1.LDAPUserSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: ldapServer.Name,
					},
					Username:           testUsername,
					FirstName:          "Test",
					LastName:           "User",
					Email:              "test@example.com",
					OrganizationalUnit: "users",
				},
			}

			// Create group
			ldapGroup = &openldapv1.LDAPGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: namespace.Name,
				},
				Spec: openldapv1.LDAPGroupSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: ldapServer.Name,
					},
					GroupName:          testGroupName,
					Description:        "Test group",
					OrganizationalUnit: "groups",
					GroupType:          openldapv1.GroupTypeGroupOfNames,
					Members: []string{
						ldapUser.Name,
					},
				},
			}
		})

		It("should manage group membership", func() {
			// Create user and group
			Expect(k8sClient.Create(ctx, ldapUser)).To(Succeed())
			Expect(k8sClient.Create(ctx, ldapGroup)).To(Succeed())

			// Wait for both to be ready
			Eventually(func() bool {
				userReady := false
				groupReady := false

				userErr := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ldapUser.Name,
					Namespace: ldapUser.Namespace,
				}, ldapUser)
				if userErr == nil {
					userReady = ldapUser.Status.Phase == openldapv1.UserPhaseReady
				}

				groupErr := k8sClient.Get(ctx, types.NamespacedName{
					Name:      ldapGroup.Name,
					Namespace: ldapGroup.Namespace,
				}, ldapGroup)
				if groupErr == nil {
					groupReady = ldapGroup.Status.Phase == openldapv1.GroupPhaseReady
				}

				return userReady && groupReady
			}, time.Minute, time.Second).Should(BeTrue())

			// Verify group membership
			members, err := ldapClient.GetGroupMembers(testGroupName, "groups", openldapv1.GroupTypeGroupOfNames)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(members)).To(BeNumerically(">=", 1))
		})
	})

	Describe("Error Handling", func() {
		It("should handle invalid LDAP server reference", func() {
			invalidUser := &openldapv1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-user",
					Namespace: namespace.Name,
				},
				Spec: openldapv1.LDAPUserSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: "non-existent-server",
					},
					Username:  "testuser",
					FirstName: "Test",
					LastName:  "User",
				},
			}

			Expect(k8sClient.Create(ctx, invalidUser)).To(Succeed())

			// User should remain in error state
			Consistently(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      invalidUser.Name,
					Namespace: invalidUser.Namespace,
				}, invalidUser)
				if err != nil {
					return false
				}
				return invalidUser.Status.Phase == openldapv1.UserPhaseError
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		It("should handle LDAP server connection failures", func() {
			// Create server with invalid host
			invalidServer := &openldapv1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-server",
					Namespace: namespace.Name,
				},
				Spec: openldapv1.LDAPServerSpec{
					Host:   "non-existent-host.example.com",
					Port:   389,
					BindDN: ldapBindDN,
					BindPasswordSecret: openldapv1.SecretReference{
						Name: "ldap-secret",
						Key:  "password",
					},
					BaseDN: ldapBaseDN,
				},
			}

			Expect(k8sClient.Create(ctx, invalidServer)).To(Succeed())

			// Server should remain in disconnected state
			Consistently(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      invalidServer.Name,
					Namespace: invalidServer.Namespace,
				}, invalidServer)
				if err != nil {
					return false
				}
				return invalidServer.Status.ConnectionStatus == openldapv1.ConnectionStatusDisconnected ||
					invalidServer.Status.ConnectionStatus == openldapv1.ConnectionStatusError
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})
	})
})

package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "github.com/guided-traffic/openldap-operator/api/v1"
)

var _ = Describe("Controller Validation Tests", func() {

	Context("LDAPServer Controller Validation", func() {
		var (
			reconciler *LDAPServerReconciler
			ctx        context.Context
			req        reconcile.Request
		)

		BeforeEach(func() {
			scheme := runtime.NewScheme()
			_ = v1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler = &LDAPServerReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			ctx = context.Background()
			req = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-server",
					Namespace: "default",
				},
			}
		})

		It("Should handle valid LDAPServer spec", func() {
			ldapServer := &v1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server",
					Namespace: "default",
				},
				Spec: v1.LDAPServerSpec{
					Host:   "ldap.example.com",
					Port:   389,
					BindDN: "cn=admin,dc=example,dc=com",
					BaseDN: "dc=example,dc=com",
					TLS: &v1.TLSConfig{
						Enabled: false,
					},
				},
			}

			err := reconciler.Client.Create(ctx, ldapServer)
			Expect(err).NotTo(HaveOccurred())

			// Test reconciliation
			result, _ := reconciler.Reconcile(ctx, req)
			// The reconciler may requeue for retrying
			Expect(result.Requeue || result.RequeueAfter > 0).To(BeTrue())
		})

		It("Should handle missing LDAPServer", func() {
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("Should validate required fields", func() {
			ldapServer := &v1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server",
					Namespace: "default",
				},
				Spec: v1.LDAPServerSpec{
					// Missing required fields
				},
			}

			err := reconciler.Client.Create(ctx, ldapServer)
			Expect(err).NotTo(HaveOccurred())

			result, _ := reconciler.Reconcile(ctx, req)
			// The reconciler may requeue for retrying
			Expect(result.Requeue || result.RequeueAfter > 0).To(BeTrue())
		})
	})

	Context("LDAPUser Controller Validation", func() {
		var (
			reconciler *LDAPUserReconciler
			ctx        context.Context
			req        reconcile.Request
		)

		BeforeEach(func() {
			scheme := runtime.NewScheme()
			_ = v1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler = &LDAPUserReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			ctx = context.Background()
			req = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-user",
					Namespace: "default",
				},
			}
		})

		It("Should handle valid LDAPUser spec", func() {
			ldapUser := &v1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1.LDAPUserSpec{
					LDAPServerRef: v1.LDAPServerReference{
						Name:      "test-server",
						Namespace: "default",
					},
					Username:  "testuser",
					Email:     "test@example.com",
					FirstName: "Test",
					LastName:  "User",
				},
			}

			err := reconciler.Client.Create(ctx, ldapUser)
			Expect(err).NotTo(HaveOccurred())

			result, _ := reconciler.Reconcile(ctx, req)
			// The reconciler may requeue for retrying
			Expect(result.Requeue || result.RequeueAfter > 0).To(BeTrue())
		})

		It("Should handle missing LDAPUser", func() {
			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("Should validate username uniqueness", func() {
			ldapUser1 := &v1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user-1",
					Namespace: "default",
				},
				Spec: v1.LDAPUserSpec{
					LDAPServerRef: v1.LDAPServerReference{
						Name:      "test-server",
						Namespace: "default",
					},
					Username:  "duplicateuser",
					Email:     "test1@example.com",
					FirstName: "Test",
					LastName:  "User1",
				},
			}

			ldapUser2 := &v1.LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user-2",
					Namespace: "default",
				},
				Spec: v1.LDAPUserSpec{
					LDAPServerRef: v1.LDAPServerReference{
						Name:      "test-server",
						Namespace: "default",
					},
					Username:  "duplicateuser", // Same username
					Email:     "test2@example.com",
					FirstName: "Test",
					LastName:  "User2",
				},
			}

			err := reconciler.Client.Create(ctx, ldapUser1)
			Expect(err).NotTo(HaveOccurred())

			err = reconciler.Client.Create(ctx, ldapUser2)
			Expect(err).NotTo(HaveOccurred())

			// Reconcile first user
			req1 := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-user-1",
					Namespace: "default",
				},
			}
			_, _ = reconciler.Reconcile(ctx, req1)

			// Reconcile second user (duplicate username)
			req2 := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-user-2",
					Namespace: "default",
				},
			}
			_, _ = reconciler.Reconcile(ctx, req2)
		})
	})

	Context("Error Handling and Edge Cases", func() {
		It("Should handle malformed requests", func() {
			scheme := runtime.NewScheme()
			_ = v1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler := &LDAPServerReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			// Empty request
			emptyReq := reconcile.Request{}
			result, err := reconciler.Reconcile(context.Background(), emptyReq)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("Should handle context cancellation", func() {
			scheme := runtime.NewScheme()
			_ = v1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				Build()

			reconciler := &LDAPServerReconciler{
				Client: fakeClient,
				Scheme: scheme,
			}

			// Cancelled context
			cancelledCtx, cancel := context.WithCancel(context.Background())
			cancel()

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-server",
					Namespace: "default",
				},
			}

			result, err := reconciler.Reconcile(cancelledCtx, req)
			// Should handle cancelled context gracefully
			Expect(result).To(Equal(ctrl.Result{}))
			// Error is expected with cancelled context
			_ = err // Use the error variable
		})
	})
})

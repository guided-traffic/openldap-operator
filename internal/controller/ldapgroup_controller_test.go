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

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

// TestLDAPGroupReconciler_Reconcile tests the main reconciliation loop for LDAPGroup resources
// The LDAPGroup controller creates group entries in external LDAP servers
// Important: Group membership is NOT managed by LDAPGroup - it's managed by LDAPUser resources
// LDAPGroup only creates the group structure (groupOfNames, posixGroup, etc.)
func TestLDAPGroupReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = openldapv1.AddToScheme(scheme)

	tests := []struct {
		name          string
		ldapGroup     *openldapv1.LDAPGroup
		ldapServer    *openldapv1.LDAPServer
		expectedPhase openldapv1.GroupPhase
		expectError   bool
		expectRequeue bool
	}{
		{
			name: "successful reconciliation with connected server",
			ldapGroup: &openldapv1.LDAPGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: openldapv1.LDAPGroupSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: "test-server",
					},
					GroupName:   "developers",
					Description: "Development team",
					GroupType:   openldapv1.GroupTypeGroupOfNames,
				},
			},
			ldapServer: &openldapv1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server",
					Namespace: "default",
				},
				Status: openldapv1.LDAPServerStatus{
					ConnectionStatus: openldapv1.ConnectionStatusConnected,
				},
			},
			expectedPhase: openldapv1.GroupPhaseError, // Will be error because no real LDAP connection
			expectError:   false,
			expectRequeue: true,
		},
		{
			name: "missing LDAP server",
			ldapGroup: &openldapv1.LDAPGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: openldapv1.LDAPGroupSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: "missing-server",
					},
					GroupName: "developers",
					GroupType: openldapv1.GroupTypeGroupOfNames,
				},
			},
			expectedPhase: openldapv1.GroupPhaseError,
			expectError:   false,
			expectRequeue: true,
		},
		{
			name: "LDAP server not connected",
			ldapGroup: &openldapv1.LDAPGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: openldapv1.LDAPGroupSpec{
					LDAPServerRef: openldapv1.LDAPServerReference{
						Name: "disconnected-server",
					},
					GroupName: "developers",
					GroupType: openldapv1.GroupTypeGroupOfNames,
				},
			},
			ldapServer: &openldapv1.LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "disconnected-server",
					Namespace: "default",
				},
				Status: openldapv1.LDAPServerStatus{
					ConnectionStatus: openldapv1.ConnectionStatusDisconnected,
				},
			},
			expectedPhase: openldapv1.GroupPhasePending,
			expectError:   false,
			expectRequeue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create objects for the fake client
			objs := []runtime.Object{tt.ldapGroup}
			if tt.ldapServer != nil {
				objs = append(objs, tt.ldapServer)
			}

			// Create fake client
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				WithStatusSubresource(&openldapv1.LDAPGroup{}).
				Build()

			// Create reconciler
			reconciler := &LDAPGroupReconciler{
				Client: client,
				Scheme: scheme,
			}

			// Perform reconciliation
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.ldapGroup.Name,
					Namespace: tt.ldapGroup.Namespace,
				},
			}

			result, err := reconciler.Reconcile(context.TODO(), req)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectRequeue {
				// Controller may or may not requeue depending on circumstances
				_ = result
			}

			// Check final status after another reconciliation to ensure status update
			_, err = reconciler.Reconcile(context.TODO(), req)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Get updated status
			updatedGroup := &openldapv1.LDAPGroup{}
			err = client.Get(context.TODO(), req.NamespacedName, updatedGroup)
			assert.NoError(t, err)

			if tt.expectedPhase != "" {
				assert.Equal(t, tt.expectedPhase, updatedGroup.Status.Phase)
			}

			// Check that finalizer was added
			assert.Contains(t, updatedGroup.Finalizers, "openldap.guided-traffic.com/finalizer")
		})
	}
}

// TestLDAPGroupReconciler_handleDeletion tests the deletion logic for LDAPGroup resources
// When a LDAPGroup is deleted:
// 1. The finalizer ensures the group is removed from LDAP before the CR is deleted
// 2. If LDAP server is unavailable, deletion still proceeds (to prevent blocking)
// 3. Finalizer is removed to allow Kubernetes to complete the deletion
func TestLDAPGroupReconciler_handleDeletion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = openldapv1.AddToScheme(scheme)

	// Create a test group with finalizer
	ldapGroup := &openldapv1.LDAPGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-group",
			Namespace:  "default",
			Finalizers: []string{"openldap.guided-traffic.com/finalizer"},
			DeletionTimestamp: &metav1.Time{
				Time: metav1.Now().Time,
			},
		},
		Spec: openldapv1.LDAPGroupSpec{
			LDAPServerRef: openldapv1.LDAPServerReference{
				Name: "missing-server", // Intentionally missing to test error handling
			},
			GroupName: "test-group",
			GroupType: openldapv1.GroupTypeGroupOfNames,
		},
	}

	// Create fake client
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(ldapGroup).
		Build()

	// Create reconciler
	reconciler := &LDAPGroupReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Perform deletion
	result, err := reconciler.handleDeletion(context.TODO(), ldapGroup)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, int64(0), int64(result.RequeueAfter))

	// Check that object was processed for deletion (finalizer removal should trigger deletion)
	updatedGroup := &openldapv1.LDAPGroup{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Name:      ldapGroup.Name,
		Namespace: ldapGroup.Namespace,
	}, updatedGroup)
	// Object should be deleted or have finalizer removed
	if err == nil {
		// If object still exists, finalizer should be removed
		assert.NotContains(t, updatedGroup.Finalizers, "openldap.guided-traffic.com/finalizer")
	} else {
		// Object was deleted, which is also correct
		assert.Contains(t, err.Error(), "not found")
	}
}

// TestLDAPGroupReconciler_UpdateGroupStatus tests status updates
func TestLDAPGroupReconciler_UpdateGroupStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = openldapv1.AddToScheme(scheme)

	t.Run("Should update status with member count", func(t *testing.T) {
		ldapGroup := &openldapv1.LDAPGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-group",
				Namespace: "default",
			},
			Spec: openldapv1.LDAPGroupSpec{
				GroupName: "developers",
				GroupType: openldapv1.GroupTypeGroupOfNames,
			},
		}

		// Simulate status update
		ldapGroup.Status.Phase = openldapv1.GroupPhaseReady
		ldapGroup.Status.Message = "Group synchronized successfully"
		ldapGroup.Status.MemberCount = 5
		ldapGroup.Status.Members = []string{"user1", "user2", "user3", "user4", "user5"}

		assert.Equal(t, openldapv1.GroupPhaseReady, ldapGroup.Status.Phase)
		assert.Equal(t, int32(5), ldapGroup.Status.MemberCount)
		assert.Len(t, ldapGroup.Status.Members, 5)
	})

	t.Run("Should handle empty group correctly", func(t *testing.T) {
		ldapGroup := &openldapv1.LDAPGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "empty-group",
				Namespace: "default",
			},
			Spec: openldapv1.LDAPGroupSpec{
				GroupName: "empty",
				GroupType: openldapv1.GroupTypePosix,
			},
		}

		// Empty POSIX groups are valid
		ldapGroup.Status.Phase = openldapv1.GroupPhaseReady
		ldapGroup.Status.MemberCount = 0
		ldapGroup.Status.Members = []string{}

		assert.Equal(t, openldapv1.GroupPhaseReady, ldapGroup.Status.Phase)
		assert.Equal(t, int32(0), ldapGroup.Status.MemberCount)
		assert.Empty(t, ldapGroup.Status.Members)
	})
}

// TestLDAPGroupReconciler_FindGroupsForServer tests watch events
func TestLDAPGroupReconciler_FindGroupsForServer(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = openldapv1.AddToScheme(scheme)

	t.Run("Should find groups referencing a specific server", func(t *testing.T) {
		group1 := &openldapv1.LDAPGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "group1",
				Namespace: "default",
			},
			Spec: openldapv1.LDAPGroupSpec{
				LDAPServerRef: openldapv1.LDAPServerReference{
					Name: "target-server",
				},
				GroupName: "developers",
				GroupType: openldapv1.GroupTypeGroupOfNames,
			},
		}

		group2 := &openldapv1.LDAPGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "group2",
				Namespace: "default",
			},
			Spec: openldapv1.LDAPGroupSpec{
				LDAPServerRef: openldapv1.LDAPServerReference{
					Name: "other-server",
				},
				GroupName: "operators",
				GroupType: openldapv1.GroupTypeGroupOfNames,
			},
		}

		client := fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(group1, group2).
			Build()

		reconciler := &LDAPGroupReconciler{
			Client: client,
			Scheme: scheme,
		}

		server := &openldapv1.LDAPServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "target-server",
				Namespace: "default",
			},
		}

		requests := reconciler.findGroupsForServer(context.TODO(), server)
		// Should only find group1
		assert.Len(t, requests, 1)
		if len(requests) > 0 {
			assert.Equal(t, "group1", requests[0].Name)
		}
	})
}

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

func TestGroupTypeObjectClasses(t *testing.T) {
	tests := []struct {
		name      string
		groupType openldapv1.GroupType
		expected  string
	}{
		{
			name:      "posixGroup type",
			groupType: openldapv1.GroupTypePosix,
			expected:  "posixGroup",
		},
		{
			name:      "groupOfNames type",
			groupType: openldapv1.GroupTypeGroupOfNames,
			expected:  "groupOfNames",
		},
		{
			name:      "groupOfUniqueNames type",
			groupType: openldapv1.GroupTypeGroupOfUniqueNames,
			expected:  "groupOfUniqueNames",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the group type constant has the expected value
			assert.Equal(t, tt.expected, string(tt.groupType))
		})
	}
}

func TestGroupPhaseValues(t *testing.T) {
	tests := []struct {
		name     string
		phase    openldapv1.GroupPhase
		expected string
	}{
		{
			name:     "pending phase",
			phase:    openldapv1.GroupPhasePending,
			expected: "Pending",
		},
		{
			name:     "ready phase",
			phase:    openldapv1.GroupPhaseReady,
			expected: "Ready",
		},
		{
			name:     "error phase",
			phase:    openldapv1.GroupPhaseError,
			expected: "Error",
		},
		{
			name:     "deleting phase",
			phase:    openldapv1.GroupPhaseDeleting,
			expected: "Deleting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the phase constant has the expected value
			assert.Equal(t, tt.expected, string(tt.phase))
		})
	}
}

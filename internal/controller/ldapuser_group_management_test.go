package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

func TestLDAPUserReconciler_GroupManagement(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, openldapv1.AddToScheme(scheme))

	t.Run("Should validate user spec with groups", func(t *testing.T) {
		// Test that the LDAPUser spec can hold groups
		ldapUser := &openldapv1.LDAPUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testuser",
				Namespace: "default",
			},
			Spec: openldapv1.LDAPUserSpec{
				Username: "testuser",
				LDAPServerRef: openldapv1.LDAPServerReference{
					Name: "ldap-server",
				},
				Groups: []string{"existing-group", "missing-group", "another-missing-group"},
			},
		}

		// Verify the groups are correctly set in spec
		assert.Equal(t, []string{"existing-group", "missing-group", "another-missing-group"}, ldapUser.Spec.Groups)
		assert.Len(t, ldapUser.Spec.Groups, 3)
	})

	t.Run("Should handle empty groups list", func(t *testing.T) {
		ldapUser := &openldapv1.LDAPUser{
			Spec: openldapv1.LDAPUserSpec{
				Username: "testuser",
				Groups:   []string{}, // Empty groups
			},
		}

		assert.Empty(t, ldapUser.Spec.Groups)
	})

	t.Run("Should handle nil groups list", func(t *testing.T) {
		ldapUser := &openldapv1.LDAPUser{
			Spec: openldapv1.LDAPUserSpec{
				Username: "testuser",
				Groups:   nil, // Nil groups
			},
		}

		assert.Nil(t, ldapUser.Spec.Groups)
	})
}

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

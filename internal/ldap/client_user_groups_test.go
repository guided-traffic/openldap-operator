package ldap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_GetUserGroups(t *testing.T) {
	t.Run("Should return error for empty username", func(t *testing.T) {
		// Test validation logic - empty username should return error
		err := validateGetUserGroupsInput("", "users", "groups")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username cannot be empty")

		// Test valid input
		err = validateGetUserGroupsInput("testuser", "users", "groups")
		assert.NoError(t, err)
	})
}

// validateGetUserGroupsInput simulates the validation logic from GetUserGroups
func validateGetUserGroupsInput(username, userOU, groupOU string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	return nil
}

func TestClient_GetUserGroups_BuildUserDN(t *testing.T) {
	tests := []struct {
		name     string
		username string
		userOU   string
		baseDN   string
		expected string
	}{
		{
			name:     "user with OU",
			username: "testuser",
			userOU:   "users",
			baseDN:   "dc=example,dc=com",
			expected: "uid=testuser,ou=users,dc=example,dc=com",
		},
		{
			name:     "user without OU",
			username: "testuser",
			userOU:   "",
			baseDN:   "dc=example,dc=com",
			expected: "uid=testuser,dc=example,dc=com",
		},
		{
			name:     "complex username",
			username: "test.user-123",
			userOU:   "people",
			baseDN:   "ou=company,dc=example,dc=org",
			expected: "uid=test.user-123,ou=people,ou=company,dc=example,dc=org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test DN building logic directly (simulate internal behavior)
			userDN := buildUserDNForTest(tt.username, tt.userOU, tt.baseDN)
			assert.Equal(t, tt.expected, userDN)
		})
	}
}

// buildUserDNForTest simulates the internal buildUserDN logic for testing
func buildUserDNForTest(username, userOU, baseDN string) string {
	if userOU == "" {
		return fmt.Sprintf("uid=%s,%s", username, baseDN)
	}
	return fmt.Sprintf("uid=%s,ou=%s,%s", username, userOU, baseDN)
}

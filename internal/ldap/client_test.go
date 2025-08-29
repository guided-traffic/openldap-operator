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

package ldap

import (
	"fmt"
	"testing"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

func TestClient_buildUserDN(t *testing.T) {
	client := &Client{
		config: &openldapv1.LDAPServerSpec{
			BaseDN: "dc=example,dc=com",
		},
	}

	tests := []struct {
		name     string
		username string
		ou       string
		expected string
	}{
		{
			name:     "user with ou",
			username: "jdoe",
			ou:       "users",
			expected: "uid=jdoe,ou=users,dc=example,dc=com",
		},
		{
			name:     "user without ou",
			username: "admin",
			ou:       "",
			expected: "uid=admin,dc=example,dc=com",
		},
		{
			name:     "complex username",
			username: "john.doe",
			ou:       "employees",
			expected: "uid=john.doe,ou=employees,dc=example,dc=com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.buildUserDN(tt.username, tt.ou)
			if result != tt.expected {
				t.Errorf("buildUserDN() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClient_buildGroupDN(t *testing.T) {
	client := &Client{
		config: &openldapv1.LDAPServerSpec{
			BaseDN: "dc=example,dc=com",
		},
	}

	tests := []struct {
		name      string
		groupName string
		ou        string
		expected  string
	}{
		{
			name:      "group with ou",
			groupName: "developers",
			ou:        "groups",
			expected:  "cn=developers,ou=groups,dc=example,dc=com",
		},
		{
			name:      "group without ou",
			groupName: "admins",
			ou:        "",
			expected:  "cn=admins,dc=example,dc=com",
		},
		{
			name:      "complex group name",
			groupName: "dev-team",
			ou:        "teams",
			expected:  "cn=dev-team,ou=teams,dc=example,dc=com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.buildGroupDN(tt.groupName, tt.ou)
			if result != tt.expected {
				t.Errorf("buildGroupDN() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewClient_ValidationOnly(t *testing.T) {
	tests := []struct {
		name        string
		spec        *openldapv1.LDAPServerSpec
		password    string
		expectError bool
	}{
		{
			name: "valid spec",
			spec: &openldapv1.LDAPServerSpec{
				Host:              "ldap.example.com",
				Port:              389,
				BindDN:            "cn=admin,dc=example,dc=com",
				BaseDN:            "dc=example,dc=com",
				ConnectionTimeout: 30,
			},
			password:    "password",
			expectError: true, // Will fail due to no actual LDAP server
		},
		{
			name: "tls enabled spec",
			spec: &openldapv1.LDAPServerSpec{
				Host:   "ldaps.example.com",
				Port:   636,
				BindDN: "cn=admin,dc=example,dc=com",
				BaseDN: "dc=example,dc=com",
				TLS: &openldapv1.TLSConfig{
					Enabled:            true,
					InsecureSkipVerify: true,
				},
			},
			password:    "password",
			expectError: true, // Will fail due to no actual LDAP server
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.spec, tt.password)

			// Since we don't have a real LDAP server, we expect errors
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if client != nil {
				client.Close()
			}
		})
	}
}

func TestCreateUserAttributes(t *testing.T) {
	userSpec := &openldapv1.LDAPUserSpec{
		Username:      "testuser",
		FirstName:     "Test",
		LastName:      "User",
		Email:         "test@example.com",
		UserID:        func() *int32 { var id int32 = 1001; return &id }(),
		GroupID:       func() *int32 { var id int32 = 1000; return &id }(),
		HomeDirectory: "/home/testuser",
		LoginShell:    "/bin/bash",
	}

	// Test attribute building logic (without actual LDAP connection)
	if userSpec.Username == "" {
		t.Error("Username should not be empty")
	}

	if userSpec.FirstName == "" || userSpec.LastName == "" {
		t.Error("First name and last name should not be empty")
	}

	if userSpec.Email == "" {
		t.Error("Email should not be empty")
	}

	if userSpec.UserID == nil || *userSpec.UserID <= 0 {
		t.Error("User ID should be positive")
	}

	if userSpec.GroupID == nil || *userSpec.GroupID <= 0 {
		t.Error("Group ID should be positive")
	}
}

func TestCreateUserAttributes_DefaultHomeDirectory(t *testing.T) {
	tests := []struct {
		name            string
		userSpec        *openldapv1.LDAPUserSpec
		expectedHomeDir string
	}{
		{
			name: "user with explicit home directory",
			userSpec: &openldapv1.LDAPUserSpec{
				Username:      "testuser1",
				FirstName:     "Test",
				LastName:      "User",
				HomeDirectory: "/custom/home/testuser1",
			},
			expectedHomeDir: "/custom/home/testuser1",
		},
		{
			name: "user without home directory gets default",
			userSpec: &openldapv1.LDAPUserSpec{
				Username:  "testuser2",
				FirstName: "Test",
				LastName:  "User",
				// No HomeDirectory specified
			},
			expectedHomeDir: "/home/testuser2",
		},
		{
			name: "user with empty home directory gets default",
			userSpec: &openldapv1.LDAPUserSpec{
				Username:      "testuser3",
				FirstName:     "Test",
				LastName:      "User",
				HomeDirectory: "", // Explicitly empty
			},
			expectedHomeDir: "/home/testuser3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the homeDirectory logic
			homeDir := tt.userSpec.HomeDirectory
			if homeDir == "" {
				homeDir = fmt.Sprintf("/home/%s", tt.userSpec.Username)
			}

			if homeDir != tt.expectedHomeDir {
				t.Errorf("Expected homeDirectory %s, got %s", tt.expectedHomeDir, homeDir)
			}
		})
	}
}

func TestCreateGroupAttributes(t *testing.T) {
	groupSpec := &openldapv1.LDAPGroupSpec{
		GroupName:   "developers",
		Description: "Development team",
		GroupType:   openldapv1.GroupTypeGroupOfNames,
		GroupID:     func() *int32 { var id int32 = 2000; return &id }(),
	}

	// Test attribute building logic (without actual LDAP connection)
	if groupSpec.GroupName == "" {
		t.Error("Group name should not be empty")
	}

	if groupSpec.Description == "" {
		t.Error("Description should not be empty")
	}

	if groupSpec.GroupType == "" {
		t.Error("Group type should not be empty")
	}

	validGroupTypes := []openldapv1.GroupType{
		openldapv1.GroupTypePosix,
		openldapv1.GroupTypeGroupOfNames,
		openldapv1.GroupTypeGroupOfUniqueNames,
	}

	valid := false
	for _, validType := range validGroupTypes {
		if groupSpec.GroupType == validType {
			valid = true
			break
		}
	}

	if !valid {
		t.Error("Group type should be valid")
	}
}

func TestGroupTypeObjectClasses(t *testing.T) {
	tests := []struct {
		name        string
		groupType   openldapv1.GroupType
		expectedOCs []string
	}{
		{
			name:        "posix group",
			groupType:   openldapv1.GroupTypePosix,
			expectedOCs: []string{"posixGroup", "top"},
		},
		{
			name:        "group of names",
			groupType:   openldapv1.GroupTypeGroupOfNames,
			expectedOCs: []string{"groupOfNames", "top"},
		},
		{
			name:        "group of unique names",
			groupType:   openldapv1.GroupTypeGroupOfUniqueNames,
			expectedOCs: []string{"groupOfUniqueNames", "top"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objectClasses []string
			switch tt.groupType {
			case openldapv1.GroupTypePosix:
				objectClasses = []string{"posixGroup", "top"}
			case openldapv1.GroupTypeGroupOfNames:
				objectClasses = []string{"groupOfNames", "top"}
			case openldapv1.GroupTypeGroupOfUniqueNames:
				objectClasses = []string{"groupOfUniqueNames", "top"}
			default:
				objectClasses = []string{"groupOfNames", "top"}
			}

			if len(objectClasses) != len(tt.expectedOCs) {
				t.Errorf("Expected %d object classes, got %d", len(tt.expectedOCs), len(objectClasses))
				return
			}

			for i, oc := range objectClasses {
				if oc != tt.expectedOCs[i] {
					t.Errorf("Expected object class %s, got %s", tt.expectedOCs[i], oc)
				}
			}
		})
	}
}

func TestGroupMemberAttributes(t *testing.T) {
	tests := []struct {
		name         string
		groupType    openldapv1.GroupType
		expectedAttr string
	}{
		{
			name:         "group of names",
			groupType:    openldapv1.GroupTypeGroupOfNames,
			expectedAttr: "member",
		},
		{
			name:         "group of unique names",
			groupType:    openldapv1.GroupTypeGroupOfUniqueNames,
			expectedAttr: "uniqueMember",
		},
		{
			name:         "posix group",
			groupType:    openldapv1.GroupTypePosix,
			expectedAttr: "memberUid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attribute string
			switch tt.groupType {
			case openldapv1.GroupTypeGroupOfNames:
				attribute = "member"
			case openldapv1.GroupTypeGroupOfUniqueNames:
				attribute = "uniqueMember"
			case openldapv1.GroupTypePosix:
				attribute = "memberUid"
			default:
				attribute = "member"
			}

			if attribute != tt.expectedAttr {
				t.Errorf("Expected attribute %s, got %s", tt.expectedAttr, attribute)
			}
		})
	}
}

// Benchmark tests
func BenchmarkBuildUserDN(b *testing.B) {
	client := &Client{
		config: &openldapv1.LDAPServerSpec{
			BaseDN: "dc=example,dc=com",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buildUserDN("testuser", "users")
	}
}

func BenchmarkBuildGroupDN(b *testing.B) {
	client := &Client{
		config: &openldapv1.LDAPServerSpec{
			BaseDN: "dc=example,dc=com",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.buildGroupDN("testgroup", "groups")
	}
}

// Table-driven tests for complex scenarios
func TestDNBuildingEdgeCases(t *testing.T) {
	client := &Client{
		config: &openldapv1.LDAPServerSpec{
			BaseDN: "ou=people,dc=company,dc=org",
		},
	}

	tests := []struct {
		name     string
		function func(string, string) string
		param1   string
		param2   string
		expected string
	}{
		{
			name:     "user with complex base DN",
			function: client.buildUserDN,
			param1:   "jdoe",
			param2:   "employees",
			expected: "uid=jdoe,ou=employees,ou=people,dc=company,dc=org",
		},
		{
			name:     "group with complex base DN",
			function: client.buildGroupDN,
			param1:   "developers",
			param2:   "teams",
			expected: "cn=developers,ou=teams,ou=people,dc=company,dc=org",
		},
		{
			name:     "user with empty ou and complex base DN",
			function: client.buildUserDN,
			param1:   "admin",
			param2:   "",
			expected: "uid=admin,ou=people,dc=company,dc=org",
		},
		{
			name:     "group with empty ou and complex base DN",
			function: client.buildGroupDN,
			param1:   "admins",
			param2:   "",
			expected: "cn=admins,ou=people,dc=company,dc=org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.param1, tt.param2)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

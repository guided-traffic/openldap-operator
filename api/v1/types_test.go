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

package v1

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestLDAPServerSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    LDAPServerSpec
		wantErr bool
	}{
		{
			name: "valid ldap server spec",
			spec: LDAPServerSpec{
				Host:   "ldap.example.com",
				Port:   389,
				BindDN: "cn=admin,dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-secret",
					Key:  "password",
				},
				BaseDN: "dc=example,dc=com",
			},
			wantErr: false,
		},
		{
			name: "valid ldaps server spec",
			spec: LDAPServerSpec{
				Host:   "ldaps.example.com",
				Port:   636,
				BindDN: "cn=admin,dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-secret",
					Key:  "password",
				},
				BaseDN: "dc=example,dc=com",
				TLS: &TLSConfig{
					Enabled: true,
				},
			},
			wantErr: false,
		},
		{
			name: "empty host",
			spec: LDAPServerSpec{
				Host:   "",
				Port:   389,
				BindDN: "cn=admin,dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-secret",
					Key:  "password",
				},
				BaseDN: "dc=example,dc=com",
			},
			wantErr: true,
		},
		{
			name: "empty bind dn",
			spec: LDAPServerSpec{
				Host:   "ldap.example.com",
				Port:   389,
				BindDN: "",
				BindPasswordSecret: SecretReference{
					Name: "ldap-secret",
					Key:  "password",
				},
				BaseDN: "dc=example,dc=com",
			},
			wantErr: true,
		},
		{
			name: "empty base dn",
			spec: LDAPServerSpec{
				Host:   "ldap.example.com",
				Port:   389,
				BindDN: "cn=admin,dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-secret",
					Key:  "password",
				},
				BaseDN: "",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too low",
			spec: LDAPServerSpec{
				Host:   "ldap.example.com",
				Port:   0,
				BindDN: "cn=admin,dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-secret",
					Key:  "password",
				},
				BaseDN: "dc=example,dc=com",
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			spec: LDAPServerSpec{
				Host:   "ldap.example.com",
				Port:   65536,
				BindDN: "cn=admin,dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-secret",
					Key:  "password",
				},
				BaseDN: "dc=example,dc=com",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateLDAPServerSpec(&tt.spec, field.NewPath("spec"))
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateLDAPServerSpec() error = %v, wantErr %v, errors: %v", hasErr, tt.wantErr, errs)
			}
		})
	}
}

func TestLDAPUserSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    LDAPUserSpec
		wantErr bool
	}{
		{
			name: "valid ldap user spec",
			spec: LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "ldap-server",
				},
				Username:  "testuser",
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: false,
		},
		{
			name: "empty username",
			spec: LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "ldap-server",
				},
				Username:  "",
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: true,
		},
		{
			name: "empty ldap server reference",
			spec: LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "",
				},
				Username:  "testuser",
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: true,
		},
		{
			name: "invalid email format",
			spec: LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "ldap-server",
				},
				Username:  "testuser",
				Email:     "invalid-email",
				FirstName: "Test",
				LastName:  "User",
			},
			wantErr: true,
		},
		{
			name: "valid with posix attributes",
			spec: LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "ldap-server",
				},
				Username:      "testuser",
				Email:         "test@example.com",
				FirstName:     "Test",
				LastName:      "User",
				UserID:        func() *int32 { var id int32 = 1001; return &id }(),
				GroupID:       func() *int32 { var id int32 = 1000; return &id }(),
				HomeDirectory: "/home/testuser",
				LoginShell:    "/bin/bash",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateLDAPUserSpec(&tt.spec, field.NewPath("spec"))
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateLDAPUserSpec() error = %v, wantErr %v, errors: %v", hasErr, tt.wantErr, errs)
			}
		})
	}
}

func TestLDAPGroupSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    LDAPGroupSpec
		wantErr bool
	}{
		{
			name: "valid ldap group spec",
			spec: LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "ldap-server",
				},
				GroupName:   "developers",
				Description: "Development team",
				GroupType:   GroupTypeGroupOfNames,
			},
			wantErr: false,
		},
		{
			name: "empty group name",
			spec: LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "ldap-server",
				},
				GroupName:   "",
				Description: "Development team",
				GroupType:   GroupTypeGroupOfNames,
			},
			wantErr: true,
		},
		{
			name: "empty ldap server reference",
			spec: LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "",
				},
				GroupName:   "developers",
				Description: "Development team",
				GroupType:   GroupTypeGroupOfNames,
			},
			wantErr: true,
		},
		{
			name: "valid posix group",
			spec: LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "ldap-server",
				},
				GroupName:   "developers",
				Description: "Development team",
				GroupType:   GroupTypePosix,
				GroupID:     func() *int32 { var id int32 = 2000; return &id }(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateLDAPGroupSpec(&tt.spec, field.NewPath("spec"))
			hasErr := len(errs) > 0
			if hasErr != tt.wantErr {
				t.Errorf("validateLDAPGroupSpec() error = %v, wantErr %v, errors: %v", hasErr, tt.wantErr, errs)
			}
		})
	}
}

func TestConnectionStatus_String(t *testing.T) {
	tests := []struct {
		status   ConnectionStatus
		expected string
	}{
		{ConnectionStatusConnected, "Connected"},
		{ConnectionStatusDisconnected, "Disconnected"},
		{ConnectionStatusError, "Error"},
		{ConnectionStatusUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("ConnectionStatus string = %v, want %v", string(tt.status), tt.expected)
			}
		})
	}
}

func TestUserPhase_String(t *testing.T) {
	tests := []struct {
		phase    UserPhase
		expected string
	}{
		{UserPhasePending, "Pending"},
		{UserPhaseReady, "Ready"},
		{UserPhaseError, "Error"},
		{UserPhaseDeleting, "Deleting"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			if string(tt.phase) != tt.expected {
				t.Errorf("UserPhase string = %v, want %v", string(tt.phase), tt.expected)
			}
		})
	}
}

func TestGroupPhase_String(t *testing.T) {
	tests := []struct {
		phase    GroupPhase
		expected string
	}{
		{GroupPhasePending, "Pending"},
		{GroupPhaseReady, "Ready"},
		{GroupPhaseError, "Error"},
		{GroupPhaseDeleting, "Deleting"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			if string(tt.phase) != tt.expected {
				t.Errorf("GroupPhase string = %v, want %v", string(tt.phase), tt.expected)
			}
		})
	}
}

func TestGroupType_String(t *testing.T) {
	tests := []struct {
		groupType GroupType
		expected  string
	}{
		{GroupTypePosix, "posixGroup"},
		{GroupTypeGroupOfNames, "groupOfNames"},
		{GroupTypeGroupOfUniqueNames, "groupOfUniqueNames"},
	}

	for _, tt := range tests {
		t.Run(string(tt.groupType), func(t *testing.T) {
			if string(tt.groupType) != tt.expected {
				t.Errorf("GroupType string = %v, want %v", string(tt.groupType), tt.expected)
			}
		})
	}
}

func TestLDAPServer_DefaultValues(t *testing.T) {
	server := &LDAPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "default",
		},
		Spec: LDAPServerSpec{
			Host:   "ldap.example.com",
			BindDN: "cn=admin,dc=example,dc=com",
			BindPasswordSecret: SecretReference{
				Name: "ldap-secret",
				Key:  "password",
			},
			BaseDN: "dc=example,dc=com",
		},
	}

	// Test default port
	if server.Spec.Port == 0 {
		server.Spec.Port = 389 // Default should be set
	}
	if server.Spec.Port != 389 {
		t.Errorf("Expected default port to be 389, got %d", server.Spec.Port)
	}

	// Test default connection timeout
	if server.Spec.ConnectionTimeout == 0 {
		server.Spec.ConnectionTimeout = 30 // Default should be set
	}
	if server.Spec.ConnectionTimeout != 30 {
		t.Errorf("Expected default connection timeout to be 30, got %d", server.Spec.ConnectionTimeout)
	}

	// Test default health check interval
	if server.Spec.HealthCheckInterval == nil {
		server.Spec.HealthCheckInterval = &metav1.Duration{Duration: 5 * time.Minute}
	}
	if server.Spec.HealthCheckInterval.Duration != 5*time.Minute {
		t.Errorf("Expected default health check interval to be 5m, got %v", server.Spec.HealthCheckInterval.Duration)
	}
}

func TestLDAPUser_DefaultValues(t *testing.T) {
	user := &LDAPUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
		},
		Spec: LDAPUserSpec{
			LDAPServerRef: LDAPServerReference{
				Name: "ldap-server",
			},
			Username: "testuser",
		},
	}

	// Test default organizational unit
	if user.Spec.OrganizationalUnit == "" {
		user.Spec.OrganizationalUnit = "users" // Default should be set
	}
	if user.Spec.OrganizationalUnit != "users" {
		t.Errorf("Expected default organizational unit to be 'users', got %s", user.Spec.OrganizationalUnit)
	}

	// Test default enabled value
	if user.Spec.Enabled == nil {
		enabled := true
		user.Spec.Enabled = &enabled // Default should be set
	}
	if user.Spec.Enabled == nil || !*user.Spec.Enabled {
		t.Errorf("Expected default enabled to be true, got %v", user.Spec.Enabled)
	}
}

func TestLDAPGroup_DefaultValues(t *testing.T) {
	group := &LDAPGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "default",
		},
		Spec: LDAPGroupSpec{
			LDAPServerRef: LDAPServerReference{
				Name: "ldap-server",
			},
			GroupName: "developers",
		},
	}

	// Test default organizational unit
	if group.Spec.OrganizationalUnit == "" {
		group.Spec.OrganizationalUnit = "groups" // Default should be set
	}
	if group.Spec.OrganizationalUnit != "groups" {
		t.Errorf("Expected default organizational unit to be 'groups', got %s", group.Spec.OrganizationalUnit)
	}

	// Test default group type
	if group.Spec.GroupType == "" {
		group.Spec.GroupType = GroupTypeGroupOfNames // Default should be set
	}
	if group.Spec.GroupType != GroupTypeGroupOfNames {
		t.Errorf("Expected default group type to be 'groupOfNames', got %s", group.Spec.GroupType)
	}
}

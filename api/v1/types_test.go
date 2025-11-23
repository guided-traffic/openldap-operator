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
		{UserPhaseWarning, "Warning"},
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

// TestLDAPServer_DeepCopy tests the DeepCopy methods for LDAPServer.
//
// What: Verifies that LDAPServer can be deep-copied correctly using both DeepCopy() and DeepCopyObject().
// Why: Kubernetes controllers frequently copy objects when processing them (e.g., before modifications).
//
//	DeepCopy ensures independent copies that don't share pointers to nested structures.
//
// How: Creates an LDAPServer with all fields populated, calls DeepCopy() and DeepCopyObject(),
//
//	then verifies the copy has identical field values and correct type.
func TestLDAPServer_DeepCopy(t *testing.T) {
	original := &LDAPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "default",
		},
		Spec: LDAPServerSpec{
			Host:   "ldap.example.com",
			Port:   389,
			BindDN: "cn=admin,dc=example,dc=com",
			BaseDN: "dc=example,dc=com",
			BindPasswordSecret: SecretReference{
				Name: "ldap-secret",
				Key:  "password",
			},
		},
		Status: LDAPServerStatus{
			ConnectionStatus: ConnectionStatusConnected,
			Message:          "Connected successfully",
		},
	}

	// Test DeepCopy
	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if copied.Name != original.Name {
		t.Errorf("Expected name %s, got %s", original.Name, copied.Name)
	}
	if copied.Spec.Host != original.Spec.Host {
		t.Errorf("Expected host %s, got %s", original.Spec.Host, copied.Spec.Host)
	}

	// Test DeepCopyObject
	copiedObj := original.DeepCopyObject()
	if copiedObj == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
	copiedServer, ok := copiedObj.(*LDAPServer)
	if !ok {
		t.Fatal("DeepCopyObject did not return *LDAPServer")
	}
	if copiedServer.Name != original.Name {
		t.Errorf("Expected name %s, got %s", original.Name, copiedServer.Name)
	}
}

// TestLDAPUser_DeepCopy tests the DeepCopy methods for LDAPUser.
//
// What: Verifies that LDAPUser can be deep-copied correctly with all its fields including pointers.
// Why: LDAPUser contains pointer fields (UserID, GroupID, Enabled) and slices (Groups, SSHPublicKeys).
//
//	DeepCopy must create independent copies to prevent shared references between objects.
//
// How: Creates an LDAPUser with all optional fields set, calls DeepCopy() and DeepCopyObject(),
//
//	verifies field values match and pointer fields are independent copies (different addresses).
func TestLDAPUser_DeepCopy(t *testing.T) {
	userID := int32(1000)
	groupID := int32(1000)
	enabled := true

	original := &LDAPUser{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
		},
		Spec: LDAPUserSpec{
			LDAPServerRef: LDAPServerReference{
				Name: "ldap-server",
			},
			Username:           "testuser",
			Email:              "test@example.com",
			FirstName:          "Test",
			LastName:           "User",
			OrganizationalUnit: "users",
			UserID:             &userID,
			GroupID:            &groupID,
			Groups:             []string{"developers", "admins"},
			Enabled:            &enabled,
		},
		Status: LDAPUserStatus{
			Phase:   UserPhaseReady,
			Message: "User synchronized",
			Groups:  []string{"developers", "admins"},
		},
	}

	// Test DeepCopy
	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if copied.Name != original.Name {
		t.Errorf("Expected name %s, got %s", original.Name, copied.Name)
	}
	if copied.Spec.Username != original.Spec.Username {
		t.Errorf("Expected username %s, got %s", original.Spec.Username, copied.Spec.Username)
	}
	if *copied.Spec.UserID != *original.Spec.UserID {
		t.Errorf("Expected userID %d, got %d", *original.Spec.UserID, *copied.Spec.UserID)
	}
	if len(copied.Spec.Groups) != len(original.Spec.Groups) {
		t.Errorf("Expected %d groups, got %d", len(original.Spec.Groups), len(copied.Spec.Groups))
	}

	// Test DeepCopyObject
	copiedObj := original.DeepCopyObject()
	if copiedObj == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
	copiedUser, ok := copiedObj.(*LDAPUser)
	if !ok {
		t.Fatal("DeepCopyObject did not return *LDAPUser")
	}
	if copiedUser.Name != original.Name {
		t.Errorf("Expected name %s, got %s", original.Name, copiedUser.Name)
	}
}

// TestLDAPGroup_DeepCopy tests the DeepCopy methods for LDAPGroup.
//
// What: Verifies that LDAPGroup can be deep-copied correctly with its pointer field (GroupID).
// Why: LDAPGroup contains an optional pointer field (GroupID) that must be independently copied.
//
//	Incorrect copying could cause unintended side effects when modifying copied objects.
//
// How: Creates an LDAPGroup with GroupID set, calls DeepCopy() and DeepCopyObject(),
//
//	verifies field values match and GroupID pointer is a new copy (different address).
func TestLDAPGroup_DeepCopy(t *testing.T) {
	groupID := int32(5000)
	original := &LDAPGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-group",
			Namespace: "default",
		},
		Spec: LDAPGroupSpec{
			LDAPServerRef: LDAPServerReference{
				Name: "ldap-server",
			},
			GroupName:          "developers",
			Description:        "Development team",
			OrganizationalUnit: "groups",
			GroupID:            &groupID,
			GroupType:          GroupTypePosix,
		},
		Status: LDAPGroupStatus{
			Phase:       GroupPhaseReady,
			Message:     "Group synchronized",
			Members:     []string{"user1", "user2", "user3"},
			MemberCount: 3,
		},
	}

	// Test DeepCopy
	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if copied.Name != original.Name {
		t.Errorf("Expected name %s, got %s", original.Name, copied.Name)
	}
	if copied.Spec.GroupName != original.Spec.GroupName {
		t.Errorf("Expected groupName %s, got %s", original.Spec.GroupName, copied.Spec.GroupName)
	}
	if *copied.Spec.GroupID != *original.Spec.GroupID {
		t.Errorf("Expected GID %d, got %d", *original.Spec.GroupID, *copied.Spec.GroupID)
	}
	if len(copied.Status.Members) != len(original.Status.Members) {
		t.Errorf("Expected %d members, got %d", len(original.Status.Members), len(copied.Status.Members))
	}

	// Test DeepCopyObject
	copiedObj := original.DeepCopyObject()
	if copiedObj == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
	copiedGroup, ok := copiedObj.(*LDAPGroup)
	if !ok {
		t.Fatal("DeepCopyObject did not return *LDAPGroup")
	}
	if copiedGroup.Name != original.Name {
		t.Errorf("Expected name %s, got %s", original.Name, copiedGroup.Name)
	}
}

// TestLDAPServerList_DeepCopy tests DeepCopy for LDAPServerList.
//
// What: Verifies that LDAPServerList can be deep-copied including all items in the Items slice.
// Why: List types are used by Kubernetes client-go for list operations. DeepCopy must create
//
//	independent copies of the entire list to prevent shared references between list items.
//
// How: Creates a list with multiple LDAPServer items, calls DeepCopy() and DeepCopyObject(),
//
//	verifies the list length and individual item names match the original.
func TestLDAPServerList_DeepCopy(t *testing.T) {
	original := &LDAPServerList{
		Items: []LDAPServer{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "server1",
					Namespace: "default",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "server2",
					Namespace: "default",
				},
			},
		},
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if len(copied.Items) != len(original.Items) {
		t.Errorf("Expected %d items, got %d", len(original.Items), len(copied.Items))
	}

	copiedObj := original.DeepCopyObject()
	if copiedObj == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
}

// TestLDAPUserList_DeepCopy tests DeepCopy for LDAPUserList.
//
// What: Verifies that LDAPUserList can be deep-copied including all items with pointer fields.
// Why: LDAPUserList contains LDAPUser items with pointer fields that must be independently copied.
//
//	This ensures list operations don't create shared references between user objects.
//
// How: Creates a list with multiple LDAPUser items (each with UserID/GroupID pointers),
//
//	calls DeepCopy() and DeepCopyObject(), verifies list length and individual field values.
func TestLDAPUserList_DeepCopy(t *testing.T) {
	userID := int32(1000)
	groupID := int32(1000)

	original := &LDAPUserList{
		Items: []LDAPUser{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user1",
					Namespace: "default",
				},
				Spec: LDAPUserSpec{
					Username: "user1",
					UserID:   &userID,
					GroupID:  &groupID,
				},
			},
		},
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if len(copied.Items) != len(original.Items) {
		t.Errorf("Expected %d items, got %d", len(original.Items), len(copied.Items))
	}

	copiedObj := original.DeepCopyObject()
	if copiedObj == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
}

// TestLDAPGroupList_DeepCopy tests DeepCopy for LDAPGroupList.
//
// What: Verifies that LDAPGroupList can be deep-copied including all group items.
// Why: List types must support deep copying for Kubernetes client-go operations.
//
//	Ensures list operations create independent copies of all group items.
//
// How: Creates a list with multiple LDAPGroup items, calls DeepCopy() and DeepCopyObject(),
//
//	verifies the list length matches the original.
func TestLDAPGroupList_DeepCopy(t *testing.T) {
	original := &LDAPGroupList{
		Items: []LDAPGroup{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "group1",
					Namespace: "default",
				},
				Spec: LDAPGroupSpec{
					GroupName: "developers",
				},
			},
		},
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if len(copied.Items) != len(original.Items) {
		t.Errorf("Expected %d items, got %d", len(original.Items), len(copied.Items))
	}

	copiedObj := original.DeepCopyObject()
	if copiedObj == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
}

// TestTLSConfig_DeepCopy tests DeepCopy for TLSConfig.
//
// What: Verifies that TLSConfig embedded struct can be deep-copied correctly.
// Why: TLSConfig is embedded in LDAPServerSpec and must be independently copyable.
//
//	This ensures TLS settings aren't shared when copying LDAPServer objects.
//
// How: Creates a TLSConfig with all fields set, calls DeepCopy(), verifies field values match.
func TestTLSConfig_DeepCopy(t *testing.T) {
	original := &TLSConfig{
		Enabled:            true,
		InsecureSkipVerify: false,
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if copied.Enabled != original.Enabled {
		t.Errorf("Expected Enabled %v, got %v", original.Enabled, copied.Enabled)
	}
	if copied.InsecureSkipVerify != original.InsecureSkipVerify {
		t.Errorf("Expected InsecureSkipVerify %v, got %v", original.InsecureSkipVerify, copied.InsecureSkipVerify)
	}
}

// TestSecretReference_DeepCopy tests DeepCopy for SecretReference.
//
// What: Verifies that SecretReference embedded struct can be deep-copied correctly.
// Why: SecretReference is used in LDAPServerSpec for bind credentials and must be independently copyable.
//
//	Ensures secret references aren't shared when copying LDAPServer objects.
//
// How: Creates a SecretReference with Name/Key set, calls DeepCopy(), verifies field values match.
func TestSecretReference_DeepCopy(t *testing.T) {
	original := &SecretReference{
		Name: "my-secret",
		Key:  "password",
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if copied.Name != original.Name {
		t.Errorf("Expected Name %s, got %s", original.Name, copied.Name)
	}
	if copied.Key != original.Key {
		t.Errorf("Expected Key %s, got %s", original.Key, copied.Key)
	}
}

// TestLDAPServerReference_DeepCopy tests DeepCopy for LDAPServerReference.
//
// What: Verifies that LDAPServerReference embedded struct can be deep-copied correctly.
// Why: LDAPServerReference is used in LDAPUser/LDAPGroup specs to reference LDAPServer objects.
//
//	Must be independently copyable to prevent shared references across user/group objects.
//
// How: Creates a LDAPServerReference with Name/Namespace set, calls DeepCopy(), verifies field values.
func TestLDAPServerReference_DeepCopy(t *testing.T) {
	original := &LDAPServerReference{
		Name:      "ldap-server",
		Namespace: "infrastructure",
	}

	copied := original.DeepCopy()
	if copied == nil {
		t.Fatal("DeepCopy returned nil")
	}
	if copied.Name != original.Name {
		t.Errorf("Expected Name %s, got %s", original.Name, copied.Name)
	}
	if copied.Namespace != original.Namespace {
		t.Errorf("Expected Namespace %s, got %s", original.Namespace, copied.Namespace)
	}
}

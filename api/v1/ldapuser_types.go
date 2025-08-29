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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LDAPUserSpec defines the desired state of LDAPUser
type LDAPUserSpec struct {
	// LDAPServerRef is a reference to the LDAPServer this user belongs to
	LDAPServerRef LDAPServerReference `json:"ldapServerRef"`

	// Username is the LDAP username (uid)
	Username string `json:"username"`

	// Email is the user's email address
	Email string `json:"email,omitempty"`

	// FirstName is the user's first name (givenName)
	FirstName string `json:"firstName,omitempty"`

	// LastName is the user's last name (sn)
	LastName string `json:"lastName,omitempty"`

	// DisplayName is the user's display name (displayName)
	DisplayName string `json:"displayName,omitempty"`

	// PasswordSecret contains the reference to the secret containing the user's password
	PasswordSecret *SecretReference `json:"passwordSecret,omitempty"`

	// Groups is a list of group names this user should belong to
	Groups []string `json:"groups,omitempty"`

	// OrganizationalUnit specifies which OU the user should be placed in
	// If not specified, defaults to "users"
	// +kubebuilder:default:="users"
	OrganizationalUnit string `json:"organizationalUnit,omitempty"`

	// UserID is the numeric user ID (uidNumber)
	UserID *int32 `json:"userID,omitempty"`

	// GroupID is the primary group ID (gidNumber)
	GroupID *int32 `json:"groupID,omitempty"`

	// HomeDirectory is the user's home directory path
	HomeDirectory string `json:"homeDirectory,omitempty"`

	// LoginShell is the user's login shell
	LoginShell string `json:"loginShell,omitempty"`

	// Enabled indicates whether the user account is enabled
	// +kubebuilder:default:=true
	Enabled *bool `json:"enabled,omitempty"`

	// AdditionalAttributes allows setting custom LDAP attributes
	AdditionalAttributes map[string][]string `json:"additionalAttributes,omitempty"`
}

// LDAPServerReference represents a reference to an LDAPServer resource
type LDAPServerReference struct {
	// Name of the LDAPServer resource
	Name string `json:"name"`
	// Namespace of the LDAPServer resource (optional, defaults to same namespace)
	Namespace string `json:"namespace,omitempty"`
}

// LDAPUserStatus defines the observed state of LDAPUser
type LDAPUserStatus struct {
	// Phase represents the current lifecycle phase of the LDAP user
	// +kubebuilder:validation:Enum=Pending;Ready;Warning;Error;Deleting
	Phase UserPhase `json:"phase,omitempty"`

	// Message provides additional information about the current phase
	Message string `json:"message,omitempty"`

	// DN is the full distinguished name of the user in LDAP
	DN string `json:"dn,omitempty"`

	// ActualHomeDirectory is the home directory that was actually set in LDAP (may be auto-generated)
	ActualHomeDirectory string `json:"actualHomeDirectory,omitempty"`

	// Groups contains the list of groups the user currently belongs to
	Groups []string `json:"groups,omitempty"`

	// MissingGroups contains the list of groups that don't exist in LDAP but are specified in spec.groups
	MissingGroups []string `json:"missingGroups,omitempty"`

	// LastModified is the timestamp of the last modification
	LastModified *metav1.Time `json:"lastModified,omitempty"`

	// Conditions represent the latest available observations of the user's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration represents the .metadata.generation that the condition was set based upon
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// UserPhase represents the lifecycle phase of an LDAP user
type UserPhase string

const (
	// UserPhasePending indicates the user is being created or updated
	UserPhasePending UserPhase = "Pending"
	// UserPhaseReady indicates the user is successfully created and synchronized
	UserPhaseReady UserPhase = "Ready"
	// UserPhaseWarning indicates the user is created but some groups are missing
	UserPhaseWarning UserPhase = "Warning"
	// UserPhaseError indicates there was an error managing the user
	UserPhaseError UserPhase = "Error"
	// UserPhaseDeleting indicates the user is being deleted
	UserPhaseDeleting UserPhase = "Deleting"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.username"
//+kubebuilder:printcolumn:name="LDAP Server",type="string",JSONPath=".spec.ldapServerRef.name"
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Email",type="string",JSONPath=".spec.email"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// LDAPUser is the Schema for the ldapusers API
type LDAPUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPUserSpec   `json:"spec,omitempty"`
	Status LDAPUserStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LDAPUserList contains a list of LDAPUser
type LDAPUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LDAPUser{}, &LDAPUserList{})
}

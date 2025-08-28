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

// LDAPGroupSpec defines the desired state of LDAPGroup
type LDAPGroupSpec struct {
	// LDAPServerRef is a reference to the LDAPServer this group belongs to
	LDAPServerRef LDAPServerReference `json:"ldapServerRef"`

	// GroupName is the LDAP group name (cn)
	GroupName string `json:"groupName"`

	// Description is the group description
	Description string `json:"description,omitempty"`

	// OrganizationalUnit specifies which OU the group should be placed in
	// If not specified, defaults to "groups"
	// +kubebuilder:default:="groups"
	OrganizationalUnit string `json:"organizationalUnit,omitempty"`

	// GroupID is the numeric group ID (gidNumber)
	GroupID *int32 `json:"groupID,omitempty"`

	// GroupType specifies the type of group (e.g., posixGroup, groupOfNames)
	// +kubebuilder:validation:Enum=posixGroup;groupOfNames;groupOfUniqueNames
	// +kubebuilder:default:="groupOfNames"
	GroupType GroupType `json:"groupType,omitempty"`

	// AdditionalAttributes allows setting custom LDAP attributes
	AdditionalAttributes map[string][]string `json:"additionalAttributes,omitempty"`
}

// GroupType represents the type of LDAP group
type GroupType string

const (
	// GroupTypePosix represents a POSIX group (posixGroup)
	GroupTypePosix GroupType = "posixGroup"
	// GroupTypeGroupOfNames represents a groupOfNames
	GroupTypeGroupOfNames GroupType = "groupOfNames"
	// GroupTypeGroupOfUniqueNames represents a groupOfUniqueNames
	GroupTypeGroupOfUniqueNames GroupType = "groupOfUniqueNames"
)

// LDAPGroupStatus defines the observed state of LDAPGroup
type LDAPGroupStatus struct {
	// Phase represents the current lifecycle phase of the LDAP group
	// +kubebuilder:validation:Enum=Pending;Ready;Error;Deleting
	Phase GroupPhase `json:"phase,omitempty"`

	// Message provides additional information about the current phase
	Message string `json:"message,omitempty"`

	// DN is the full distinguished name of the group in LDAP
	DN string `json:"dn,omitempty"`

	// Members contains the list of current group members
	Members []string `json:"members,omitempty"`

	// MemberCount is the number of members in the group
	MemberCount int32 `json:"memberCount,omitempty"`

	// LastModified is the timestamp of the last modification
	LastModified *metav1.Time `json:"lastModified,omitempty"`

	// Conditions represent the latest available observations of the group's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration represents the .metadata.generation that the condition was set based upon
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// GroupPhase represents the lifecycle phase of an LDAP group
type GroupPhase string

const (
	// GroupPhasePending indicates the group is being created or updated
	GroupPhasePending GroupPhase = "Pending"
	// GroupPhaseReady indicates the group is successfully created and synchronized
	GroupPhaseReady GroupPhase = "Ready"
	// GroupPhaseError indicates there was an error managing the group
	GroupPhaseError GroupPhase = "Error"
	// GroupPhaseDeleting indicates the group is being deleted
	GroupPhaseDeleting GroupPhase = "Deleting"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Group Name",type="string",JSONPath=".spec.groupName"
//+kubebuilder:printcolumn:name="LDAP Server",type="string",JSONPath=".spec.ldapServerRef.name"
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Members",type="integer",JSONPath=".status.memberCount"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// LDAPGroup is the Schema for the ldapgroups API
type LDAPGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPGroupSpec   `json:"spec,omitempty"`
	Status LDAPGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LDAPGroupList contains a list of LDAPGroup
type LDAPGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LDAPGroup{}, &LDAPGroupList{})
}

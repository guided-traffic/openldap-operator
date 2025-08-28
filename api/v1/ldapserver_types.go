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

// LDAPServerSpec defines the desired state of LDAPServer
type LDAPServerSpec struct {
	// Host is the hostname or IP address of the LDAP server
	Host string `json:"host"`

	// Port is the port number of the LDAP server (default: 389 for LDAP, 636 for LDAPS)
	// +kubebuilder:default:=389
	Port int32 `json:"port,omitempty"`

	// BindDN is the distinguished name used to bind to the LDAP server
	BindDN string `json:"bindDN"`

	// BindPasswordSecret contains the reference to the secret containing the bind password
	BindPasswordSecret SecretReference `json:"bindPasswordSecret"`

	// BaseDN is the base distinguished name for LDAP operations
	BaseDN string `json:"baseDN"`

	// TLS configuration for secure connections
	TLS *TLSConfig `json:"tls,omitempty"`

	// ConnectionTimeout in seconds (default: 30)
	// +kubebuilder:default:=30
	ConnectionTimeout int32 `json:"connectionTimeout,omitempty"`

	// HealthCheckInterval defines how often to check the connection (default: 5m)
	// +kubebuilder:default:="5m"
	HealthCheckInterval *metav1.Duration `json:"healthCheckInterval,omitempty"`
}

// SecretReference represents a reference to a Kubernetes secret
type SecretReference struct {
	// Name of the secret
	Name string `json:"name"`
	// Key within the secret containing the value
	Key string `json:"key"`
}

// TLSConfig contains TLS-specific configuration
type TLSConfig struct {
	// Enabled indicates whether to use TLS/SSL
	Enabled bool `json:"enabled"`

	// InsecureSkipVerify controls whether the client verifies the server's certificate
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// CACertSecret contains the reference to the CA certificate secret
	CACertSecret *SecretReference `json:"caCertSecret,omitempty"`

	// ClientCertSecret contains the reference to the client certificate secret
	ClientCertSecret *SecretReference `json:"clientCertSecret,omitempty"`

	// ClientKeySecret contains the reference to the client private key secret
	ClientKeySecret *SecretReference `json:"clientKeySecret,omitempty"`
}

// LDAPServerStatus defines the observed state of LDAPServer
type LDAPServerStatus struct {
	// ConnectionStatus represents the current connection status to the LDAP server
	// +kubebuilder:validation:Enum=Connected;Disconnected;Error;Unknown
	ConnectionStatus ConnectionStatus `json:"connectionStatus,omitempty"`

	// LastChecked is the timestamp of the last connection check
	LastChecked *metav1.Time `json:"lastChecked,omitempty"`

	// Message provides additional information about the connection status
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations of the LDAP server's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration represents the .metadata.generation that the condition was set based upon
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ConnectionStatus represents the status of the LDAP connection
type ConnectionStatus string

const (
	// ConnectionStatusConnected indicates the LDAP server is reachable and authentication succeeded
	ConnectionStatusConnected ConnectionStatus = "Connected"
	// ConnectionStatusDisconnected indicates the LDAP server is not reachable
	ConnectionStatusDisconnected ConnectionStatus = "Disconnected"
	// ConnectionStatusError indicates there was an error connecting or authenticating
	ConnectionStatusError ConnectionStatus = "Error"
	// ConnectionStatusUnknown indicates the status is not yet determined
	ConnectionStatusUnknown ConnectionStatus = "Unknown"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Host",type="string",JSONPath=".spec.host"
//+kubebuilder:printcolumn:name="Port",type="integer",JSONPath=".spec.port"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.connectionStatus"
//+kubebuilder:printcolumn:name="Last Checked",type="date",JSONPath=".status.lastChecked"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// LDAPServer is the Schema for the ldapservers API
type LDAPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LDAPServerSpec   `json:"spec,omitempty"`
	Status LDAPServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LDAPServerList contains a list of LDAPServer
type LDAPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LDAPServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LDAPServer{}, &LDAPServerList{})
}

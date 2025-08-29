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
	"net/mail"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// validateLDAPServerSpec validates the LDAPServerSpec
func validateLDAPServerSpec(spec *LDAPServerSpec, fldPath *field.Path) field.ErrorList {
	var errs field.ErrorList

	// Validate host
	if spec.Host == "" {
		errs = append(errs, field.Required(fldPath.Child("host"), "host cannot be empty"))
	}

	// Validate port
	if spec.Port <= 0 || spec.Port > 65535 {
		errs = append(errs, field.Invalid(fldPath.Child("port"), spec.Port, "port must be between 1 and 65535"))
	}

	// Validate bindDN
	if spec.BindDN == "" {
		errs = append(errs, field.Required(fldPath.Child("bindDN"), "bindDN cannot be empty"))
	}

	// Validate baseDN
	if spec.BaseDN == "" {
		errs = append(errs, field.Required(fldPath.Child("baseDN"), "baseDN cannot be empty"))
	}

	// Validate bind password secret
	if spec.BindPasswordSecret.Name == "" {
		errs = append(errs, field.Required(fldPath.Child("bindPasswordSecret", "name"), "secret name cannot be empty"))
	}
	if spec.BindPasswordSecret.Key == "" {
		errs = append(errs, field.Required(fldPath.Child("bindPasswordSecret", "key"), "secret key cannot be empty"))
	}

	// Validate connection timeout
	if spec.ConnectionTimeout < 0 {
		errs = append(errs, field.Invalid(fldPath.Child("connectionTimeout"), spec.ConnectionTimeout, "connection timeout cannot be negative"))
	}

	return errs
}

// validateLDAPUserSpec validates the LDAPUserSpec
func validateLDAPUserSpec(spec *LDAPUserSpec, fldPath *field.Path) field.ErrorList {
	var errs field.ErrorList

	// Validate LDAP server reference
	if spec.LDAPServerRef.Name == "" {
		errs = append(errs, field.Required(fldPath.Child("ldapServerRef", "name"), "LDAP server reference name cannot be empty"))
	}

	// Validate username
	if spec.Username == "" {
		errs = append(errs, field.Required(fldPath.Child("username"), "username cannot be empty"))
	} else if !isValidUsername(spec.Username) {
		errs = append(errs, field.Invalid(fldPath.Child("username"), spec.Username, "username contains invalid characters"))
	}

	// Validate email if provided
	if spec.Email != "" {
		if _, err := mail.ParseAddress(spec.Email); err != nil {
			errs = append(errs, field.Invalid(fldPath.Child("email"), spec.Email, "invalid email format"))
		}
	}

	// Validate user ID if provided
	if spec.UserID != nil && *spec.UserID < 0 {
		errs = append(errs, field.Invalid(fldPath.Child("userID"), *spec.UserID, "user ID cannot be negative"))
	}

	// Validate group ID if provided
	if spec.GroupID != nil && *spec.GroupID < 0 {
		errs = append(errs, field.Invalid(fldPath.Child("groupID"), *spec.GroupID, "group ID cannot be negative"))
	}

	// Validate login shell if provided
	if spec.LoginShell != "" && !isValidShell(spec.LoginShell) {
		errs = append(errs, field.Invalid(fldPath.Child("loginShell"), spec.LoginShell, "invalid login shell"))
	}

	return errs
}

// validateLDAPGroupSpec validates the LDAPGroupSpec
func validateLDAPGroupSpec(spec *LDAPGroupSpec, fldPath *field.Path) field.ErrorList {
	var errs field.ErrorList

	// Validate LDAP server reference
	if spec.LDAPServerRef.Name == "" {
		errs = append(errs, field.Required(fldPath.Child("ldapServerRef", "name"), "LDAP server reference name cannot be empty"))
	}

	// Validate group name
	if spec.GroupName == "" {
		errs = append(errs, field.Required(fldPath.Child("groupName"), "group name cannot be empty"))
	} else if !isValidGroupName(spec.GroupName) {
		errs = append(errs, field.Invalid(fldPath.Child("groupName"), spec.GroupName, "group name contains invalid characters"))
	}

	// Validate group type
	if spec.GroupType != "" && !isValidGroupType(spec.GroupType) {
		errs = append(errs, field.Invalid(fldPath.Child("groupType"), spec.GroupType, "invalid group type"))
	}

	// Validate group ID if provided
	if spec.GroupID != nil && *spec.GroupID < 0 {
		errs = append(errs, field.Invalid(fldPath.Child("groupID"), *spec.GroupID, "group ID cannot be negative"))
	}

	return errs
}

// isValidUsername checks if the username is valid
func isValidUsername(username string) bool {
	if len(username) == 0 || len(username) > 32 {
		return false
	}
	// Username should contain only alphanumeric characters, dots, hyphens, and underscores
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '.' || char == '-' || char == '_') {
			return false
		}
	}
	return true
}

// isValidGroupName checks if the group name is valid
func isValidGroupName(groupName string) bool {
	if len(groupName) == 0 || len(groupName) > 32 {
		return false
	}
	// Group name should contain only alphanumeric characters, dots, hyphens, and underscores
	for _, char := range groupName {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '.' || char == '-' || char == '_') {
			return false
		}
	}
	return true
}

// isValidShell checks if the shell path is valid
func isValidShell(shell string) bool {
	validShells := []string{
		"/bin/bash",
		"/bin/sh",
		"/bin/zsh",
		"/bin/fish",
		"/bin/tcsh",
		"/bin/csh",
		"/usr/bin/bash",
		"/usr/bin/sh",
		"/usr/bin/zsh",
		"/usr/bin/fish",
		"/usr/bin/tcsh",
		"/usr/bin/csh",
		"/bin/false", // For disabled accounts
		"/sbin/nologin",
	}

	for _, validShell := range validShells {
		if shell == validShell {
			return true
		}
	}
	return false
}

// isValidGroupType checks if the group type is valid
func isValidGroupType(groupType GroupType) bool {
	switch groupType {
	case GroupTypePosix, GroupTypeGroupOfNames, GroupTypeGroupOfUniqueNames:
		return true
	default:
		return false
	}
}

// SetDefaults sets default values for LDAPServerSpec
func (s *LDAPServerSpec) SetDefaults() {
	// Initialize TLS config if nil (defaults to enabled)
	if s.TLS == nil {
		s.TLS = &TLSConfig{
			Enabled:            true,  // TLS enabled by default
			InsecureSkipVerify: false, // Secure by default, but can be overridden
		}
	}

	if s.Port == 0 {
		if s.TLS != nil && s.TLS.Enabled {
			s.Port = 636 // Default LDAPS port
		} else {
			s.Port = 389 // Default LDAP port
		}
	}

	if s.ConnectionTimeout == 0 {
		s.ConnectionTimeout = 30 // Default 30 seconds
	}
}

// SetDefaults sets default values for LDAPUserSpec
func (s *LDAPUserSpec) SetDefaults() {
	if s.OrganizationalUnit == "" {
		s.OrganizationalUnit = "users"
	}

	if s.Enabled == nil {
		enabled := true
		s.Enabled = &enabled
	}
}

// SetDefaults sets default values for LDAPGroupSpec
func (s *LDAPGroupSpec) SetDefaults() {
	if s.OrganizationalUnit == "" {
		s.OrganizationalUnit = "groups"
	}

	if s.GroupType == "" {
		s.GroupType = GroupTypeGroupOfNames
	}
}

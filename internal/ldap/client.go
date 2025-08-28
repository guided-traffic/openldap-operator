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
	"crypto/tls"
	"fmt"
	"strconv"
	"time"

	"github.com/go-ldap/ldap/v3"
	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

// Client represents an LDAP client wrapper
type Client struct {
	conn   *ldap.Conn
	config *openldapv1.LDAPServerSpec
}

// NewClient creates a new LDAP client
func NewClient(spec *openldapv1.LDAPServerSpec, password string) (*Client, error) {
	var conn *ldap.Conn
	var err error

	address := fmt.Sprintf("%s:%d", spec.Host, spec.Port)

	// Create connection based on TLS configuration
	if spec.TLS != nil && spec.TLS.Enabled {
		tlsConfig := &tls.Config{
			ServerName:         spec.Host,
			InsecureSkipVerify: spec.TLS.InsecureSkipVerify,
		}
		conn, err = ldap.DialTLS("tcp", address, tlsConfig)
	} else {
		conn, err = ldap.Dial("tcp", address)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	// Set connection timeout
	if spec.ConnectionTimeout > 0 {
		conn.SetTimeout(time.Duration(spec.ConnectionTimeout) * time.Second)
	}

	// Bind with provided credentials
	err = conn.Bind(spec.BindDN, password)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	return &Client{
		conn:   conn,
		config: spec,
	}, nil
}

// Close closes the LDAP connection
func (c *Client) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// TestConnection tests if the LDAP connection is working
func (c *Client) TestConnection() error {
	if c.conn == nil {
		return fmt.Errorf("no active connection")
	}

	// Perform a simple search to test the connection
	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1,
		int(time.Duration(30*time.Second).Seconds()),
		false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)

	_, err := c.conn.Search(searchRequest)
	return err
}

// CreateUser creates a new user in LDAP
func (c *Client) CreateUser(userSpec *openldapv1.LDAPUserSpec) error {
	dn := c.buildUserDN(userSpec.Username, userSpec.OrganizationalUnit)

	// Build user attributes
	attrs := []ldap.Attribute{
		{
			Type: "objectClass",
			Vals: []string{"inetOrgPerson", "posixAccount", "top"},
		},
		{
			Type: "uid",
			Vals: []string{userSpec.Username},
		},
		{
			Type: "cn",
			Vals: []string{fmt.Sprintf("%s %s", userSpec.FirstName, userSpec.LastName)},
		},
		{
			Type: "sn",
			Vals: []string{userSpec.LastName},
		},
		{
			Type: "givenName",
			Vals: []string{userSpec.FirstName},
		},
	}

	// Add optional attributes
	if userSpec.Email != "" {
		attrs = append(attrs, ldap.Attribute{
			Type: "mail",
			Vals: []string{userSpec.Email},
		})
	}

	if userSpec.UserID != nil {
		attrs = append(attrs, ldap.Attribute{
			Type: "uidNumber",
			Vals: []string{strconv.Itoa(int(*userSpec.UserID))},
		})
	}

	if userSpec.GroupID != nil {
		attrs = append(attrs, ldap.Attribute{
			Type: "gidNumber",
			Vals: []string{strconv.Itoa(int(*userSpec.GroupID))},
		})
	}

	if userSpec.HomeDirectory != "" {
		attrs = append(attrs, ldap.Attribute{
			Type: "homeDirectory",
			Vals: []string{userSpec.HomeDirectory},
		})
	}

	if userSpec.LoginShell != "" {
		attrs = append(attrs, ldap.Attribute{
			Type: "loginShell",
			Vals: []string{userSpec.LoginShell},
		})
	}

	// Create add request
	addRequest := ldap.NewAddRequest(dn, nil)
	for _, attr := range attrs {
		addRequest.Attribute(attr.Type, attr.Vals)
	}

	return c.conn.Add(addRequest)
}

// UpdateUser updates an existing user in LDAP
func (c *Client) UpdateUser(userSpec *openldapv1.LDAPUserSpec) error {
	dn := c.buildUserDN(userSpec.Username, userSpec.OrganizationalUnit)

	modifyRequest := ldap.NewModifyRequest(dn, nil)

	// Update displayName
	displayName := fmt.Sprintf("%s %s", userSpec.FirstName, userSpec.LastName)
	modifyRequest.Replace("cn", []string{displayName})
	modifyRequest.Replace("sn", []string{userSpec.LastName})
	modifyRequest.Replace("givenName", []string{userSpec.FirstName})

	// Update email if provided
	if userSpec.Email != "" {
		modifyRequest.Replace("mail", []string{userSpec.Email})
	}

	// Update home directory if provided
	if userSpec.HomeDirectory != "" {
		modifyRequest.Replace("homeDirectory", []string{userSpec.HomeDirectory})
	}

	// Update login shell if provided
	if userSpec.LoginShell != "" {
		modifyRequest.Replace("loginShell", []string{userSpec.LoginShell})
	}

	return c.conn.Modify(modifyRequest)
}

// DeleteUser deletes a user from LDAP
func (c *Client) DeleteUser(username, ou string) error {
	dn := c.buildUserDN(username, ou)
	deleteRequest := ldap.NewDelRequest(dn, nil)
	return c.conn.Del(deleteRequest)
}

// UserExists checks if a user exists in LDAP
func (c *Client) UserExists(username, ou string) (bool, error) {
	dn := c.buildUserDN(username, ou)

	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1,
		30,
		false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			return false, nil
		}
		return false, err
	}

	return len(result.Entries) > 0, nil
}

// CreateGroup creates a new group in LDAP
func (c *Client) CreateGroup(groupSpec *openldapv1.LDAPGroupSpec) error {
	dn := c.buildGroupDN(groupSpec.GroupName, groupSpec.OrganizationalUnit)

	var objectClasses []string
	switch groupSpec.GroupType {
	case openldapv1.GroupTypePosix:
		objectClasses = []string{"posixGroup", "top"}
	case openldapv1.GroupTypeGroupOfNames:
		objectClasses = []string{"groupOfNames", "top"}
	case openldapv1.GroupTypeGroupOfUniqueNames:
		objectClasses = []string{"groupOfUniqueNames", "top"}
	default:
		objectClasses = []string{"groupOfNames", "top"}
	}

	attrs := []ldap.Attribute{
		{
			Type: "objectClass",
			Vals: objectClasses,
		},
		{
			Type: "cn",
			Vals: []string{groupSpec.GroupName},
		},
	}

	// Add description if provided
	if groupSpec.Description != "" {
		attrs = append(attrs, ldap.Attribute{
			Type: "description",
			Vals: []string{groupSpec.Description},
		})
	}

	// Add group ID for posix groups
	if groupSpec.GroupType == openldapv1.GroupTypePosix && groupSpec.GroupID != nil {
		attrs = append(attrs, ldap.Attribute{
			Type: "gidNumber",
			Vals: []string{strconv.Itoa(int(*groupSpec.GroupID))},
		})
	}

	// Add initial member for groupOfNames (required)
	if groupSpec.GroupType == openldapv1.GroupTypeGroupOfNames {
		attrs = append(attrs, ldap.Attribute{
			Type: "member",
			Vals: []string{c.config.BindDN}, // Use bind DN as initial member
		})
	}

	// Create add request
	addRequest := ldap.NewAddRequest(dn, nil)
	for _, attr := range attrs {
		addRequest.Attribute(attr.Type, attr.Vals)
	}

	return c.conn.Add(addRequest)
}

// DeleteGroup deletes a group from LDAP
func (c *Client) DeleteGroup(groupName, ou string) error {
	dn := c.buildGroupDN(groupName, ou)
	deleteRequest := ldap.NewDelRequest(dn, nil)
	return c.conn.Del(deleteRequest)
}

// GroupExists checks if a group exists in LDAP
func (c *Client) GroupExists(groupName, ou string) (bool, error) {
	dn := c.buildGroupDN(groupName, ou)

	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1,
		30,
		false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			return false, nil
		}
		return false, err
	}

	return len(result.Entries) > 0, nil
}

// AddUserToGroup adds a user to a group
func (c *Client) AddUserToGroup(username, userOU, groupName, groupOU string, groupType openldapv1.GroupType) error {
	groupDN := c.buildGroupDN(groupName, groupOU)
	userDN := c.buildUserDN(username, userOU)

	modifyRequest := ldap.NewModifyRequest(groupDN, nil)

	switch groupType {
	case openldapv1.GroupTypeGroupOfNames:
		modifyRequest.Add("member", []string{userDN})
	case openldapv1.GroupTypeGroupOfUniqueNames:
		modifyRequest.Add("uniqueMember", []string{userDN})
	case openldapv1.GroupTypePosix:
		modifyRequest.Add("memberUid", []string{username})
	}

	return c.conn.Modify(modifyRequest)
}

// RemoveUserFromGroup removes a user from a group
func (c *Client) RemoveUserFromGroup(username, userOU, groupName, groupOU string, groupType openldapv1.GroupType) error {
	groupDN := c.buildGroupDN(groupName, groupOU)
	userDN := c.buildUserDN(username, userOU)

	modifyRequest := ldap.NewModifyRequest(groupDN, nil)

	switch groupType {
	case openldapv1.GroupTypeGroupOfNames:
		modifyRequest.Delete("member", []string{userDN})
	case openldapv1.GroupTypeGroupOfUniqueNames:
		modifyRequest.Delete("uniqueMember", []string{userDN})
	case openldapv1.GroupTypePosix:
		modifyRequest.Delete("memberUid", []string{username})
	}

	return c.conn.Modify(modifyRequest)
}

// GetGroupMembers retrieves all members of a group
func (c *Client) GetGroupMembers(groupName, ou string, groupType openldapv1.GroupType) ([]string, error) {
	dn := c.buildGroupDN(groupName, ou)

	var attribute string
	switch groupType {
	case openldapv1.GroupTypeGroupOfNames:
		attribute = "member"
	case openldapv1.GroupTypeGroupOfUniqueNames:
		attribute = "uniqueMember"
	case openldapv1.GroupTypePosix:
		attribute = "memberUid"
	default:
		attribute = "member"
	}

	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0,
		30,
		false,
		"(objectClass=*)",
		[]string{attribute},
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if len(result.Entries) == 0 {
		return []string{}, nil
	}

	return result.Entries[0].GetAttributeValues(attribute), nil
}

// buildUserDN builds the DN for a user
func (c *Client) buildUserDN(username, ou string) string {
	if ou == "" {
		return fmt.Sprintf("uid=%s,%s", username, c.config.BaseDN)
	}
	return fmt.Sprintf("uid=%s,ou=%s,%s", username, ou, c.config.BaseDN)
}

// buildGroupDN builds the DN for a group
func (c *Client) buildGroupDN(groupName, ou string) string {
	if ou == "" {
		return fmt.Sprintf("cn=%s,%s", groupName, c.config.BaseDN)
	}
	return fmt.Sprintf("cn=%s,ou=%s,%s", groupName, ou, c.config.BaseDN)
}

// SearchUsers searches for users in LDAP
func (c *Client) SearchUsers(filter string, attributes []string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		30,
		false,
		filter,
		attributes,
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	return result.Entries, nil
}

// SearchGroups searches for groups in LDAP
func (c *Client) SearchGroups(filter string, attributes []string) ([]*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		30,
		false,
		filter,
		attributes,
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	return result.Entries, nil
}

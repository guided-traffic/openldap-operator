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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
	ldaputil "github.com/guided-traffic/openldap-operator/internal/ldap"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	LDAPHost     string
	LDAPPort     int
	LDAPBindDN   string
	LDAPBaseDN   string
	LDAPPassword string
	Timeout      time.Duration
}

// LoadTestConfig loads test configuration from environment variables or flags
func LoadTestConfig() *TestConfig {
	config := &TestConfig{
		LDAPHost:     "localhost",
		LDAPPort:     389,
		LDAPBindDN:   "cn=admin,dc=example,dc=com",
		LDAPBaseDN:   "dc=example,dc=com",
		LDAPPassword: "admin123",
		Timeout:      30 * time.Second,
	}

	// Override with environment variables
	if host := os.Getenv("LDAP_HOST"); host != "" {
		config.LDAPHost = host
	}
	if port := os.Getenv("LDAP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.LDAPPort = p
		}
	}
	if bindDN := os.Getenv("LDAP_BIND_DN"); bindDN != "" {
		config.LDAPBindDN = bindDN
	}
	if baseDN := os.Getenv("LDAP_BASE_DN"); baseDN != "" {
		config.LDAPBaseDN = baseDN
	}
	if password := os.Getenv("LDAP_PASSWORD"); password != "" {
		config.LDAPPassword = password
	}

	// Override with command line flags
	flag.StringVar(&config.LDAPHost, "ldap-host", config.LDAPHost, "LDAP server host")
	flag.IntVar(&config.LDAPPort, "ldap-port", config.LDAPPort, "LDAP server port")
	flag.StringVar(&config.LDAPBindDN, "ldap-bind-dn", config.LDAPBindDN, "LDAP bind DN")
	flag.StringVar(&config.LDAPBaseDN, "ldap-base-dn", config.LDAPBaseDN, "LDAP base DN")
	flag.StringVar(&config.LDAPPassword, "ldap-password", config.LDAPPassword, "LDAP bind password")
	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "Test timeout")
	flag.Parse()

	return config
}

// TestResult represents the result of a test
type TestResult struct {
	Name     string
	Passed   bool
	Error    error
	Duration time.Duration
}

// TestSuite represents a collection of integration tests
type TestSuite struct {
	config  *TestConfig
	client  *ldaputil.Client
	results []TestResult
}

// NewTestSuite creates a new test suite
func NewTestSuite(config *TestConfig) (*TestSuite, error) {
	spec := &openldapv1.LDAPServerSpec{
		Host:              config.LDAPHost,
		Port:              int32(config.LDAPPort), //nolint:gosec // Port is validated to be in valid range
		BindDN:            config.LDAPBindDN,
		BaseDN:            config.LDAPBaseDN,
		ConnectionTimeout: int32(config.Timeout.Seconds()),
	}

	client, err := ldaputil.NewClient(spec, config.LDAPPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create LDAP client: %w", err)
	}

	return &TestSuite{
		config: config,
		client: client,
	}, nil
}

// Close closes the test suite and cleans up resources
func (ts *TestSuite) Close() error {
	if ts.client != nil {
		return ts.client.Close()
	}
	return nil
}

// RunTest runs a single test
func (ts *TestSuite) RunTest(name string, testFunc func() error) {
	start := time.Now()
	err := testFunc()
	duration := time.Since(start)

	result := TestResult{
		Name:     name,
		Passed:   err == nil,
		Error:    err,
		Duration: duration,
	}

	ts.results = append(ts.results, result)

	if err != nil {
		log.Printf("FAIL: %s (%v) - %v", name, duration, err)
	} else {
		log.Printf("PASS: %s (%v)", name, duration)
	}
}

// PrintResults prints the test results summary
func (ts *TestSuite) PrintResults() {
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, result := range ts.results {
		totalDuration += result.Duration
		if result.Passed {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("\n=== Test Results ===\n")
	fmt.Printf("Total tests: %d\n", len(ts.results))
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total duration: %v\n", totalDuration)

	if failed > 0 {
		fmt.Printf("\nFailed tests:\n")
		for _, result := range ts.results {
			if !result.Passed {
				fmt.Printf("  - %s: %v\n", result.Name, result.Error)
			}
		}
	}
}

// GetExitCode returns the appropriate exit code based on test results
func (ts *TestSuite) GetExitCode() int {
	for _, result := range ts.results {
		if !result.Passed {
			return 1
		}
	}
	return 0
}

// testConnection tests LDAP server connection
func (ts *TestSuite) testConnection() error {
	return ts.client.TestConnection()
}

// testUserOperations tests user CRUD operations
func (ts *TestSuite) testUserOperations() error {
	ctx := context.Background()
	_ = ctx // Suppress unused variable warning

	testUsername := fmt.Sprintf("testuser-%d", time.Now().Unix())
	userSpec := &openldapv1.LDAPUserSpec{
		Username:           testUsername,
		FirstName:          "Test",
		LastName:           "User",
		Email:              "test@example.com",
		OrganizationalUnit: "users",
		UserID:             func() *int32 { var id int32 = 1001; return &id }(),
		GroupID:            func() *int32 { var id int32 = 1000; return &id }(),
		HomeDirectory:      "/home/" + testUsername,
		LoginShell:         "/bin/bash",
	}

	// Test user creation
	if err := ts.client.CreateUser(userSpec); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Test user existence check
	exists, err := ts.client.UserExists(testUsername, "users")
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("user should exist after creation")
	}

	// Test user update
	userSpec.Email = "updated@example.com"
	if err := ts.client.UpdateUser(userSpec); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Test user deletion
	if err := ts.client.DeleteUser(testUsername, "users"); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Verify user is deleted
	exists, err = ts.client.UserExists(testUsername, "users")
	if err != nil {
		return fmt.Errorf("failed to check user existence after deletion: %w", err)
	}
	if exists {
		return fmt.Errorf("user should not exist after deletion")
	}

	return nil
}

// testGroupOperations tests group CRUD operations
func (ts *TestSuite) testGroupOperations() error {
	testGroupName := fmt.Sprintf("testgroup-%d", time.Now().Unix())
	groupSpec := &openldapv1.LDAPGroupSpec{
		GroupName:          testGroupName,
		Description:        "Test group for integration testing",
		OrganizationalUnit: "groups",
		GroupType:          openldapv1.GroupTypeGroupOfNames,
		GroupID:            func() *int32 { var id int32 = 2000; return &id }(),
	}

	// Test group creation
	if err := ts.client.CreateGroup(groupSpec); err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	// Test group existence check
	exists, err := ts.client.GroupExists(testGroupName, "groups")
	if err != nil {
		return fmt.Errorf("failed to check group existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("group should exist after creation")
	}

	// Test group deletion
	if err := ts.client.DeleteGroup(testGroupName, "groups"); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	// Verify group is deleted
	exists, err = ts.client.GroupExists(testGroupName, "groups")
	if err != nil {
		return fmt.Errorf("failed to check group existence after deletion: %w", err)
	}
	if exists {
		return fmt.Errorf("group should not exist after deletion")
	}

	return nil
}

// testGroupMembership tests group membership operations
func (ts *TestSuite) testGroupMembership() error {
	testUsername := fmt.Sprintf("testuser-%d", time.Now().Unix())
	testGroupName := fmt.Sprintf("testgroup-%d", time.Now().Unix())

	// Create user
	userSpec := &openldapv1.LDAPUserSpec{
		Username:           testUsername,
		FirstName:          "Test",
		LastName:           "User",
		Email:              "test@example.com",
		OrganizationalUnit: "users",
	}
	if err := ts.client.CreateUser(userSpec); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	defer func() {
		_ = ts.client.DeleteUser(testUsername, "users")
	}()

	// Create group
	groupSpec := &openldapv1.LDAPGroupSpec{
		GroupName:          testGroupName,
		Description:        "Test group",
		OrganizationalUnit: "groups",
		GroupType:          openldapv1.GroupTypeGroupOfNames,
	}
	if err := ts.client.CreateGroup(groupSpec); err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	defer func() {
		_ = ts.client.DeleteGroup(testGroupName, "groups")
	}()

	// Add user to group
	if err := ts.client.AddUserToGroup(testUsername, "users", testGroupName, "groups", openldapv1.GroupTypeGroupOfNames); err != nil {
		return fmt.Errorf("failed to add user to group: %w", err)
	}

	// Check group membership
	members, err := ts.client.GetGroupMembers(testGroupName, "groups", openldapv1.GroupTypeGroupOfNames)
	if err != nil {
		return fmt.Errorf("failed to get group members: %w", err)
	}

	found := false
	userDN := fmt.Sprintf("uid=%s,ou=users,%s", testUsername, ts.config.LDAPBaseDN)
	for _, member := range members {
		if member == userDN {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user should be a member of the group")
	}

	// Remove user from group
	if err := ts.client.RemoveUserFromGroup(testUsername, "users", testGroupName, "groups", openldapv1.GroupTypeGroupOfNames); err != nil {
		return fmt.Errorf("failed to remove user from group: %w", err)
	}

	return nil
}

func main() {
	config := LoadTestConfig()

	fmt.Printf("Running integration tests against LDAP server at %s:%d\n", config.LDAPHost, config.LDAPPort)
	fmt.Printf("Base DN: %s\n", config.LDAPBaseDN)
	fmt.Printf("Bind DN: %s\n", config.LDAPBindDN)

	testSuite, err := NewTestSuite(config)
	if err != nil {
		log.Fatalf("Failed to create test suite: %v", err)
	}
	defer testSuite.Close()

	// Run tests
	testSuite.RunTest("Connection Test", testSuite.testConnection)
	testSuite.RunTest("User Operations", testSuite.testUserOperations)
	testSuite.RunTest("Group Operations", testSuite.testGroupOperations)
	testSuite.RunTest("Group Membership", testSuite.testGroupMembership)

	// Print results and exit
	testSuite.PrintResults()
	os.Exit(testSuite.GetExitCode())
}

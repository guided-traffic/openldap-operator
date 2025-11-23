package ldap

import (
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "github.com/guided-traffic/openldap-operator/api/v1"
)

var _ = Describe("LDAP Client Integration Tests (Fixed)", func() {
	var (
		container     *LDAPTestContainer
		client        *Client
		spec          *v1.LDAPServerSpec
		adminPassword string
	)

	BeforeEach(func() {
		EnsureDockerAvailable()

		container = NewLDAPTestContainer()

		// Start LDAP container
		err := container.Start()
		Expect(err).NotTo(HaveOccurred())

		spec = container.GetConnectionSpec()
		adminPassword = container.GetAdminPassword()

		// Create client
		client, err = NewClient(spec, adminPassword)
		Expect(err).NotTo(HaveOccurred())

		// Setup organizational units
		setupOrganizationalUnits(client, spec.BaseDN)
	})

	AfterEach(func() {
		if client != nil {
			client.Close()
		}
		if container != nil {
			container.Stop()
		}
	})

	Context("User CRUD Operations", func() {
		It("Should create a user with POSIX attributes", func() {
			userID := int32(1001)
			groupID := int32(1001)
			userSpec := &v1.LDAPUserSpec{
				Username:           "testuser",
				Email:              "testuser@example.com",
				FirstName:          "Test",
				LastName:           "User",
				OrganizationalUnit: "users",
				UserID:             &userID,
				GroupID:            &groupID,
				HomeDirectory:      "/home/testuser",
				LoginShell:         "/bin/bash",
				Groups:             []string{},
			}

			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should check if user exists", func() {
			userID := int32(1002)
			groupID := int32(1002)
			userSpec := &v1.LDAPUserSpec{
				Username:           "testuser2",
				Email:              "testuser2@example.com",
				FirstName:          "Test",
				LastName:           "User2",
				OrganizationalUnit: "users",
				UserID:             &userID,
				GroupID:            &groupID,
				HomeDirectory:      "/home/testuser2",
				LoginShell:         "/bin/bash",
			}

			// Create user first
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Check if user exists
			exists, err := client.UserExists("testuser2", "users")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("Should search users", func() {
			userID := int32(1003)
			groupID := int32(1003)
			userSpec := &v1.LDAPUserSpec{
				Username:           "searchuser",
				Email:              "searchuser@example.com",
				FirstName:          "Search",
				LastName:           "User",
				OrganizationalUnit: "users",
				UserID:             &userID,
				GroupID:            &groupID,
				HomeDirectory:      "/home/searchuser",
				LoginShell:         "/bin/bash",
			}

			// Create user first
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Search for user
			entries, err := client.SearchUsers("(uid=searchuser)", []string{"uid", "cn", "mail"})
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).NotTo(BeNil())
			Expect(len(entries)).To(BeNumerically(">=", 1))
		})

		It("Should delete a user", func() {
			userID := int32(1004)
			groupID := int32(1004)
			userSpec := &v1.LDAPUserSpec{
				Username:           "deleteuser",
				Email:              "deleteuser@example.com",
				FirstName:          "Delete",
				LastName:           "User",
				OrganizationalUnit: "users",
				UserID:             &userID,
				GroupID:            &groupID,
				HomeDirectory:      "/home/deleteuser",
				LoginShell:         "/bin/bash",
			}

			// Create user first
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Delete user
			err = client.DeleteUser("deleteuser", "users")
			Expect(err).NotTo(HaveOccurred())

			// Verify user is deleted
			exists, err := client.UserExists("deleteuser", "users")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("Should update a user", func() {
			userID := int32(1005)
			groupID := int32(1005)
			userSpec := &v1.LDAPUserSpec{
				Username:           "updateuser",
				Email:              "updateuser@example.com",
				FirstName:          "Update",
				LastName:           "User",
				OrganizationalUnit: "users",
				UserID:             &userID,
				GroupID:            &groupID,
				HomeDirectory:      "/home/updateuser",
				LoginShell:         "/bin/bash",
			}

			// Create user first
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Update user
			userSpec.Email = "updated@example.com"
			err = client.UpdateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Group CRUD Operations", func() {
		It("Should create a group", func() {
			groupID := int32(2001)
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          "testgroup",
				Description:        "Test Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypePosix,
				GroupID:            &groupID,
			}

			err := client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should check if group exists", func() {
			groupID := int32(2002)
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          "testgroup2",
				Description:        "Test Group 2",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypePosix,
				GroupID:            &groupID,
			}

			// Create group first
			err := client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())

			// Check if group exists
			exists, err := client.GroupExists("testgroup2", "groups")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("Should search groups", func() {
			groupID := int32(2003)
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          "searchgroup",
				Description:        "Search Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypePosix,
				GroupID:            &groupID,
			}

			// Create group first
			err := client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())

			// Search for group
			entries, err := client.SearchGroups("(cn=searchgroup)", []string{"cn", "description"})
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).NotTo(BeNil())
			Expect(len(entries)).To(BeNumerically(">=", 1))
		})

		It("Should delete a group", func() {
			groupID := int32(2004)
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          "deletegroup",
				Description:        "Delete Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypePosix,
				GroupID:            &groupID,
			}

			// Create group first
			err := client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())

			// Delete group
			err = client.DeleteGroup("deletegroup", "groups")
			Expect(err).NotTo(HaveOccurred())

			// Verify group is deleted
			exists, err := client.GroupExists("deletegroup", "groups")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Context("Group Membership Operations", func() {
		var testUser, testGroup string

		BeforeEach(func() {
			testUser = "memberuser"
			testGroup = "membergroup"

			// Create test user
			userID := int32(1100)
			groupID := int32(1100)
			userSpec := &v1.LDAPUserSpec{
				Username:           testUser,
				Email:              fmt.Sprintf("%s@example.com", testUser),
				FirstName:          "Member",
				LastName:           "User",
				OrganizationalUnit: "users",
				UserID:             &userID,
				GroupID:            &groupID,
				HomeDirectory:      fmt.Sprintf("/home/%s", testUser),
				LoginShell:         "/bin/bash",
			}
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Create test group
			testGroupID := int32(2100)
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          testGroup,
				Description:        "Member Test Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypePosix,
				GroupID:            &testGroupID,
			}
			err = client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should add user to group", func() {
			err := client.AddUserToGroup(testUser, "users", testGroup, "groups", v1.GroupTypePosix)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should get group members", func() {
			// Add user to group first
			err := client.AddUserToGroup(testUser, "users", testGroup, "groups", v1.GroupTypePosix)
			Expect(err).NotTo(HaveOccurred())

			// Get group members
			members, err := client.GetGroupMembers(testGroup, "groups", v1.GroupTypePosix)
			Expect(err).NotTo(HaveOccurred())
			Expect(members).NotTo(BeNil())
			Expect(members).To(ContainElement(testUser))
		})

		It("Should remove user from group", func() {
			// Add user to group first
			err := client.AddUserToGroup(testUser, "users", testGroup, "groups", v1.GroupTypePosix)
			Expect(err).NotTo(HaveOccurred())

			// Remove user from group
			err = client.RemoveUserFromGroup(testUser, "users", testGroup, "groups", v1.GroupTypePosix)
			Expect(err).NotTo(HaveOccurred())

			// Verify user is removed
			members, err := client.GetGroupMembers(testGroup, "groups", v1.GroupTypePosix)
			Expect(err).NotTo(HaveOccurred())
			Expect(members).NotTo(ContainElement(testUser))
		})
	})

	Context("Error Handling", func() {
		It("Should handle invalid connection", func() {
			invalidSpec := &v1.LDAPServerSpec{
				Host:   "invalid-host",
				Port:   389,
				BindDN: "cn=admin,dc=example,dc=com",
				BaseDN: "dc=example,dc=com",
			}

			_, err := NewClient(invalidSpec, "wrongpassword")
			Expect(err).To(HaveOccurred())
		})

		It("Should handle duplicate user creation", func() {
			userID := int32(1999)
			groupID := int32(1999)
			userSpec := &v1.LDAPUserSpec{
				Username:           "duplicateuser",
				Email:              "duplicateuser@example.com",
				FirstName:          "Duplicate",
				LastName:           "User",
				OrganizationalUnit: "users",
				UserID:             &userID,
				GroupID:            &groupID,
				HomeDirectory:      "/home/duplicateuser",
				LoginShell:         "/bin/bash",
			}

			// Create user first time
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Try to create same user again
			err = client.CreateUser(userSpec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Or(
				ContainSubstring("Already exists"),
				ContainSubstring("Entry Already Exists"),
			))
		})
	})
})

// setupOrganizationalUnits creates the required organizational units
func setupOrganizationalUnits(client *Client, baseDN string) {
	// Create users OU
	usersOU := fmt.Sprintf("ou=users,%s", baseDN)
	req := ldap.NewAddRequest(usersOU, nil)
	req.Attribute("objectClass", []string{"organizationalUnit"})
	req.Attribute("ou", []string{"users"})
	err := client.conn.Add(req)
	if err != nil && !strings.Contains(err.Error(), "Already exists") {
		// Ignore if already exists, fail for other errors
		if !strings.Contains(err.Error(), "Already exists") {
			Expect(err).NotTo(HaveOccurred())
		}
	}

	// Create groups OU
	groupsOU := fmt.Sprintf("ou=groups,%s", baseDN)
	req = ldap.NewAddRequest(groupsOU, nil)
	req.Attribute("objectClass", []string{"organizationalUnit"})
	req.Attribute("ou", []string{"groups"})
	err = client.conn.Add(req)
	if err != nil && !strings.Contains(err.Error(), "Already exists") {
		// Ignore if already exists, fail for other errors
		if !strings.Contains(err.Error(), "Already exists") {
			Expect(err).NotTo(HaveOccurred())
		}
	}
}

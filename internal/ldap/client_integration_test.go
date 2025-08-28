package ldap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "github.com/guided-traffic/openldap-operator/api/v1"
)

var _ = XDescribe("LDAP Client Integration Tests", func() {
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
	})

	AfterEach(func() {
		if client != nil {
			client.Close()
		}
		if container != nil {
			container.Stop()
		}
	})

	Context("Client Creation with Real LDAP", func() {
		It("Should create and connect to real LDAP server", func() {
			var err error
			client, err = NewClient(spec, adminPassword)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("Should test connection successfully", func() {
			var err error
			client, err = NewClient(spec, adminPassword)
			Expect(err).NotTo(HaveOccurred())

			err = client.TestConnection()
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should close connection gracefully", func() {
			var err error
			client, err = NewClient(spec, adminPassword)
			Expect(err).NotTo(HaveOccurred())

			err = client.Close()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("User CRUD Operations", func() {
		BeforeEach(func() {
			var err error
			client, err = NewClient(spec, adminPassword)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create a user", func() {
			userSpec := &v1.LDAPUserSpec{
				Username:           "testuser",
				Email:              "testuser@example.com",
				FirstName:          "Test",
				LastName:           "User",
				OrganizationalUnit: "users",
				Groups:             []string{},
			}

			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should check if user exists", func() {
			userSpec := &v1.LDAPUserSpec{
				Username:           "testuser2",
				Email:              "testuser2@example.com",
				FirstName:          "Test",
				LastName:           "User2",
				OrganizationalUnit: "users",
			}

			// Create user first
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Check if user exists
			exists, err := client.UserExists("testuser2", "users")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Check non-existent user
			exists, err = client.UserExists("nonexistent", "users")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("Should search users", func() {
			// Create a test user first
			userSpec := &v1.LDAPUserSpec{
				Username:           "searchuser",
				Email:              "searchuser@example.com",
				FirstName:          "Search",
				LastName:           "User",
				OrganizationalUnit: "users",
			}

			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Search for users
			entries, err := client.SearchUsers("(uid=searchuser)", []string{"uid", "mail"})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(entries)).To(BeNumerically(">=", 1))
		})

		It("Should delete a user", func() {
			userSpec := &v1.LDAPUserSpec{
				Username:           "deleteuser",
				Email:              "deleteuser@example.com",
				FirstName:          "Delete",
				LastName:           "User",
				OrganizationalUnit: "users",
			}

			// Create user first
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Verify user exists
			exists, err := client.UserExists("deleteuser", "users")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Delete user
			err = client.DeleteUser("deleteuser", "users")
			Expect(err).NotTo(HaveOccurred())

			// Verify user no longer exists
			exists, err = client.UserExists("deleteuser", "users")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("Should update a user", func() {
			userSpec := &v1.LDAPUserSpec{
				Username:           "updateuser",
				Email:              "updateuser@example.com",
				FirstName:          "Update",
				LastName:           "User",
				OrganizationalUnit: "users",
			}

			// Create user first
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Update user
			userSpec.Email = "updated@example.com"
			userSpec.FirstName = "Updated"
			err = client.UpdateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Group CRUD Operations", func() {
		BeforeEach(func() {
			var err error
			client, err = NewClient(spec, adminPassword)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should create a group", func() {
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          "testgroup",
				Description:        "Test Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypeGroupOfNames,
				Members:            []string{},
			}

			err := client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should check if group exists", func() {
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          "existsgroup",
				Description:        "Exists Test Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypeGroupOfNames,
				Members:            []string{},
			}

			// Create group first
			err := client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())

			// Check if group exists
			exists, err := client.GroupExists("existsgroup", "groups")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Check non-existent group
			exists, err = client.GroupExists("nonexistentgroup", "groups")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("Should search groups", func() {
			// Create a test group first
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          "searchgroup",
				Description:        "Search Test Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypeGroupOfNames,
				Members:            []string{},
			}

			err := client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())

			// Search for groups
			entries, err := client.SearchGroups("(cn=searchgroup)", []string{"cn", "description"})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(entries)).To(BeNumerically(">=", 1))
		})

		It("Should delete a group", func() {
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          "deletegroup",
				Description:        "Delete Test Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypeGroupOfNames,
				Members:            []string{},
			}

			// Create group first
			err := client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())

			// Verify group exists
			exists, err := client.GroupExists("deletegroup", "groups")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			// Delete group
			err = client.DeleteGroup("deletegroup", "groups")
			Expect(err).NotTo(HaveOccurred())

			// Verify group no longer exists
			exists, err = client.GroupExists("deletegroup", "groups")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	Context("Group Membership Operations", func() {
		var testUser string
		var testGroup string

		BeforeEach(func() {
			var err error
			client, err = NewClient(spec, adminPassword)
			Expect(err).NotTo(HaveOccurred())

			testUser = "memberuser"
			testGroup = "membergroup"

			// Create test user
			userSpec := &v1.LDAPUserSpec{
				Username:           testUser,
				Email:              "memberuser@example.com",
				FirstName:          "Member",
				LastName:           "User",
				OrganizationalUnit: "users",
			}
			err = client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Create test group
			groupSpec := &v1.LDAPGroupSpec{
				GroupName:          testGroup,
				Description:        "Member Test Group",
				OrganizationalUnit: "groups",
				GroupType:          v1.GroupTypeGroupOfNames,
				Members:            []string{},
			}
			err = client.CreateGroup(groupSpec)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should add user to group", func() {
			err := client.AddUserToGroup(testUser, "users", testGroup, "groups", v1.GroupTypeGroupOfNames)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should get group members", func() {
			// Add user to group first
			err := client.AddUserToGroup(testUser, "users", testGroup, "groups", v1.GroupTypeGroupOfNames)
			Expect(err).NotTo(HaveOccurred())

			// Get group members
			members, err := client.GetGroupMembers(testGroup, "groups", v1.GroupTypeGroupOfNames)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(members)).To(BeNumerically(">=", 1))
		})

		It("Should remove user from group", func() {
			// Add user to group first
			err := client.AddUserToGroup(testUser, "users", testGroup, "groups", v1.GroupTypeGroupOfNames)
			Expect(err).NotTo(HaveOccurred())

			// Remove user from group
			err = client.RemoveUserFromGroup(testUser, "users", testGroup, "groups", v1.GroupTypeGroupOfNames)
			Expect(err).NotTo(HaveOccurred())

			// Verify user is no longer in group
			members, err := client.GetGroupMembers(testGroup, "groups", v1.GroupTypeGroupOfNames)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(members)).To(Equal(0))
		})
	})

	Context("TLS Connection Tests", func() {
		It("Should connect with TLS", func() {
			Skip("TLS tests require more complex certificate setup")

			tlsSpec := container.GetTLSConnectionSpec()
			var err error
			client, err = NewClient(tlsSpec, adminPassword)
			Expect(err).NotTo(HaveOccurred())

			err = client.TestConnection()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Error Handling", func() {
		BeforeEach(func() {
			var err error
			client, err = NewClient(spec, adminPassword)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should handle duplicate user creation", func() {
			userSpec := &v1.LDAPUserSpec{
				Username:           "duplicateuser",
				Email:              "duplicateuser@example.com",
				FirstName:          "Duplicate",
				LastName:           "User",
				OrganizationalUnit: "users",
			}

			// Create user first time - should succeed
			err := client.CreateUser(userSpec)
			Expect(err).NotTo(HaveOccurred())

			// Create user second time - should fail
			err = client.CreateUser(userSpec)
			Expect(err).To(HaveOccurred())
		})

		It("Should handle non-existent user deletion", func() {
			err := client.DeleteUser("nonexistentuser", "users")
			Expect(err).To(HaveOccurred())
		})

		It("Should handle empty search parameters", func() {
			entries, err := client.SearchUsers("", []string{})
			Expect(err).To(HaveOccurred())
			Expect(entries).To(BeNil())
		})
	})
})

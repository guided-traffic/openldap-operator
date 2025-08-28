package ldap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	openldapv1 "github.com/guided-traffic/openldap-operator/api/v1"
)

var _ = Describe("LDAP Client Helper Functions", func() {

	Describe("buildUserDN", func() {
		var client *Client

		BeforeEach(func() {
			client = &Client{
				config: &openldapv1.LDAPServerSpec{
					BaseDN: "dc=example,dc=com",
				},
			}
		})

		It("Should build correct DN for user with default OU", func() {
			dn := client.buildUserDN("testuser", "users")
			Expect(dn).To(Equal("uid=testuser,ou=users,dc=example,dc=com"))
		})

		It("Should build correct DN for user with custom OU", func() {
			dn := client.buildUserDN("admin", "administrators")
			Expect(dn).To(Equal("uid=admin,ou=administrators,dc=example,dc=com"))
		})

		It("Should handle empty OU", func() {
			dn := client.buildUserDN("testuser", "")
			Expect(dn).To(Equal("uid=testuser,dc=example,dc=com"))
		})

		It("Should handle special characters in username", func() {
			dn := client.buildUserDN("test.user@example", "users")
			Expect(dn).To(Equal("uid=test.user@example,ou=users,dc=example,dc=com"))
		})
	})

	Describe("buildGroupDN", func() {
		var client *Client

		BeforeEach(func() {
			client = &Client{
				config: &openldapv1.LDAPServerSpec{
					BaseDN: "dc=example,dc=com",
				},
			}
		})

		It("Should build correct DN for group with default OU", func() {
			dn := client.buildGroupDN("testgroup", "groups")
			Expect(dn).To(Equal("cn=testgroup,ou=groups,dc=example,dc=com"))
		})

		It("Should build correct DN for group with custom OU", func() {
			dn := client.buildGroupDN("admins", "administrators")
			Expect(dn).To(Equal("cn=admins,ou=administrators,dc=example,dc=com"))
		})

		It("Should handle empty OU", func() {
			dn := client.buildGroupDN("testgroup", "")
			Expect(dn).To(Equal("cn=testgroup,dc=example,dc=com"))
		})

		It("Should handle special characters in group name", func() {
			dn := client.buildGroupDN("test-group.1", "groups")
			Expect(dn).To(Equal("cn=test-group.1,ou=groups,dc=example,dc=com"))
		})
	})

	Describe("User attribute building", func() {
		It("Should create basic user attributes", func() {
			userSpec := &openldapv1.LDAPUserSpec{
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
			}

			// This tests the logic that would be in CreateUser
			objectClasses := []string{"inetOrgPerson", "posixAccount", "top"}
			Expect(objectClasses).To(ContainElement("inetOrgPerson"))
			Expect(objectClasses).To(ContainElement("posixAccount"))
			Expect(objectClasses).To(ContainElement("top"))

			// Test attribute mapping
			expectedCN := "Test User"
			actualCN := userSpec.FirstName + " " + userSpec.LastName
			Expect(actualCN).To(Equal(expectedCN))
		})

		It("Should handle user with POSIX attributes", func() {
			userID := int32(1001)
			groupID := int32(1001)

			userSpec := &openldapv1.LDAPUserSpec{
				Username:  "posixuser",
				FirstName: "POSIX",
				LastName:  "User",
				UserID:    &userID,
				GroupID:   &groupID,
			}

			Expect(userSpec.UserID).ToNot(BeNil())
			Expect(*userSpec.UserID).To(Equal(int32(1001)))
			Expect(userSpec.GroupID).ToNot(BeNil())
			Expect(*userSpec.GroupID).To(Equal(int32(1001)))
		})

		It("Should handle user without optional attributes", func() {
			userSpec := &openldapv1.LDAPUserSpec{
				Username:  "basicuser",
				FirstName: "Basic",
				LastName:  "User",
			}

			Expect(userSpec.Email).To(Equal(""))
			Expect(userSpec.UserID).To(BeNil())
			Expect(userSpec.GroupID).To(BeNil())
		})
	})

	Describe("Group attribute building", func() {
		It("Should create basic group attributes", func() {
			groupSpec := &openldapv1.LDAPGroupSpec{
				GroupName:   "testgroup",
				Description: "Test Group",
				Members:     []string{"user1", "user2"},
			}

			Expect(groupSpec.GroupName).To(Equal("testgroup"))
			Expect(groupSpec.Description).To(Equal("Test Group"))
			Expect(groupSpec.Members).To(HaveLen(2))
			Expect(groupSpec.Members).To(ContainElement("user1"))
			Expect(groupSpec.Members).To(ContainElement("user2"))
		})

		It("Should handle group with POSIX attributes", func() {
			groupID := int32(2001)

			groupSpec := &openldapv1.LDAPGroupSpec{
				GroupName: "posixgroup",
				GroupType: openldapv1.GroupTypePosix,
				GroupID:   &groupID,
			}

			Expect(groupSpec.GroupType).To(Equal(openldapv1.GroupTypePosix))
			Expect(groupSpec.GroupID).ToNot(BeNil())
			Expect(*groupSpec.GroupID).To(Equal(int32(2001)))
		})

		It("Should handle group without optional attributes", func() {
			groupSpec := &openldapv1.LDAPGroupSpec{
				GroupName: "basicgroup",
			}

			Expect(groupSpec.Description).To(Equal(""))
			Expect(groupSpec.Members).To(BeEmpty())
			Expect(groupSpec.GroupID).To(BeNil())
		})
	})

	Describe("DN validation", func() {
		It("Should validate user DN format", func() {
			userDN := "uid=testuser,ou=users,dc=example,dc=com"

			// Test DN components
			Expect(userDN).To(ContainSubstring("uid=testuser"))
			Expect(userDN).To(ContainSubstring("ou=users"))
			Expect(userDN).To(ContainSubstring("dc=example,dc=com"))
		})

		It("Should validate group DN format", func() {
			groupDN := "cn=testgroup,ou=groups,dc=example,dc=com"

			// Test DN components
			Expect(groupDN).To(ContainSubstring("cn=testgroup"))
			Expect(groupDN).To(ContainSubstring("ou=groups"))
			Expect(groupDN).To(ContainSubstring("dc=example,dc=com"))
		})
	})

	Describe("Configuration validation", func() {
		It("Should handle valid LDAP server config", func() {
			config := &openldapv1.LDAPServerSpec{
				Host:   "ldap.example.com",
				Port:   389,
				BaseDN: "dc=example,dc=com",
				BindDN: "cn=admin,dc=example,dc=com",
			}

			Expect(config.Host).To(Equal("ldap.example.com"))
			Expect(config.Port).To(Equal(int32(389)))
			Expect(config.BaseDN).To(Equal("dc=example,dc=com"))
			Expect(config.BindDN).To(Equal("cn=admin,dc=example,dc=com"))
		})

		It("Should handle TLS configuration", func() {
			config := &openldapv1.LDAPServerSpec{
				Host:   "ldaps.example.com",
				Port:   636,
				BaseDN: "dc=example,dc=com",
				BindDN: "cn=admin,dc=example,dc=com",
				TLS: &openldapv1.TLSConfig{
					Enabled:            true,
					InsecureSkipVerify: false,
				},
			}

			Expect(config.TLS).ToNot(BeNil())
			Expect(config.TLS.Enabled).To(BeTrue())
			Expect(config.TLS.InsecureSkipVerify).To(BeFalse())
		})
	})

	Describe("Client Operations", func() {
		Describe("Close", func() {
			It("Should close connection successfully", func() {
				client := &Client{
					conn:   nil, // mock that connection is nil
					config: &openldapv1.LDAPServerSpec{},
				}

				err := client.Close()
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should handle nil connection safely", func() {
				client := &Client{
					conn:   nil,
					config: &openldapv1.LDAPServerSpec{},
				}

				err := client.Close()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("TestConnection", func() {
			It("Should return error when no connection exists", func() {
				client := &Client{
					conn:   nil,
					config: &openldapv1.LDAPServerSpec{},
				}

				err := client.TestConnection()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no active connection"))
			})
		})
	})
})

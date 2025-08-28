package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Type Validation Tests", func() {

	Context("LDAPServerSpec Validation", func() {
		It("Should validate required fields", func() {
			spec := LDAPServerSpec{
				Host:   "ldap.example.com",
				Port:   389,
				BindDN: "cn=admin,dc=example,dc=com",
				BaseDN: "dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-password",
					Key:  "password",
				},
			}

			Expect(spec.Host).To(Equal("ldap.example.com"))
			Expect(spec.Port).To(Equal(int32(389)))
			Expect(spec.BindDN).To(Equal("cn=admin,dc=example,dc=com"))
			Expect(spec.BaseDN).To(Equal("dc=example,dc=com"))
			Expect(spec.BindPasswordSecret.Name).To(Equal("ldap-password"))
		})

		It("Should handle TLS configuration", func() {
			spec := LDAPServerSpec{
				Host:   "ldaps.example.com",
				Port:   636,
				BindDN: "cn=admin,dc=example,dc=com",
				BaseDN: "dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-password",
					Key:  "password",
				},
				TLS: &TLSConfig{
					Enabled:            true,
					InsecureSkipVerify: false,
					CACertSecret: &SecretReference{
						Name: "ca-cert",
						Key:  "ca.crt",
					},
				},
			}

			Expect(spec.TLS.Enabled).To(BeTrue())
			Expect(spec.TLS.InsecureSkipVerify).To(BeFalse())
			Expect(spec.TLS.CACertSecret.Name).To(Equal("ca-cert"))
		})

		It("Should handle connection timeouts", func() {
			duration := metav1.Duration{Duration: metav1.Now().Time.Sub(metav1.Now().Time)}
			spec := LDAPServerSpec{
				Host:   "ldap.example.com",
				Port:   389,
				BindDN: "cn=admin,dc=example,dc=com",
				BaseDN: "dc=example,dc=com",
				BindPasswordSecret: SecretReference{
					Name: "ldap-password",
					Key:  "password",
				},
				ConnectionTimeout:   60,
				HealthCheckInterval: &duration,
			}

			Expect(spec.ConnectionTimeout).To(Equal(int32(60)))
			Expect(spec.HealthCheckInterval).NotTo(BeNil())
		})
	})

	Context("LDAPUserSpec Validation", func() {
		It("Should validate basic user fields", func() {
			spec := LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name:      "test-server",
					Namespace: "default",
				},
				Username:  "testuser",
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
			}

			Expect(spec.LDAPServerRef.Name).To(Equal("test-server"))
			Expect(spec.Username).To(Equal("testuser"))
			Expect(spec.Email).To(Equal("test@example.com"))
			Expect(spec.FirstName).To(Equal("Test"))
			Expect(spec.LastName).To(Equal("User"))
		})

		It("Should handle POSIX attributes", func() {
			userID := int32(1001)
			groupID := int32(1001)

			spec := LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name:      "test-server",
					Namespace: "default",
				},
				Username:      "posixuser",
				UserID:        &userID,
				GroupID:       &groupID,
				HomeDirectory: "/home/posixuser",
				LoginShell:    "/bin/bash",
			}

			Expect(*spec.UserID).To(Equal(int32(1001)))
			Expect(*spec.GroupID).To(Equal(int32(1001)))
			Expect(spec.HomeDirectory).To(Equal("/home/posixuser"))
			Expect(spec.LoginShell).To(Equal("/bin/bash"))
		})

		It("Should handle groups and additional attributes", func() {
			spec := LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name:      "test-server",
					Namespace: "default",
				},
				Username: "testuser",
				Groups:   []string{"group1", "group2"},
				AdditionalAttributes: map[string][]string{
					"department": {"Engineering"},
					"title":      {"Developer"},
				},
			}

			Expect(spec.Groups).To(ContainElement("group1"))
			Expect(spec.Groups).To(ContainElement("group2"))
			Expect(spec.AdditionalAttributes["department"]).To(ContainElement("Engineering"))
			Expect(spec.AdditionalAttributes["title"]).To(ContainElement("Developer"))
		})

		It("Should handle password secret", func() {
			spec := LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name:      "test-server",
					Namespace: "default",
				},
				Username: "testuser",
				PasswordSecret: &SecretReference{
					Name: "user-password",
					Key:  "password",
				},
			}

			Expect(spec.PasswordSecret.Name).To(Equal("user-password"))
			Expect(spec.PasswordSecret.Key).To(Equal("password"))
		})
	})

	Context("LDAPGroupSpec Validation", func() {
		It("Should validate basic group fields", func() {
			spec := LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name:      "test-server",
					Namespace: "default",
				},
				GroupName:   "testgroup",
				Description: "Test group description",
			}

			Expect(spec.LDAPServerRef.Name).To(Equal("test-server"))
			Expect(spec.GroupName).To(Equal("testgroup"))
			Expect(spec.Description).To(Equal("Test group description"))
		})

		It("Should handle different group types", func() {
			testCases := []struct {
				groupType GroupType
				expected  GroupType
			}{
				{GroupTypePosix, GroupTypePosix},
				{GroupTypeGroupOfNames, GroupTypeGroupOfNames},
				{GroupTypeGroupOfUniqueNames, GroupTypeGroupOfUniqueNames},
			}

			for _, tc := range testCases {
				spec := LDAPGroupSpec{
					LDAPServerRef: LDAPServerReference{
						Name:      "test-server",
						Namespace: "default",
					},
					GroupName: "testgroup",
					GroupType: tc.groupType,
				}

				Expect(spec.GroupType).To(Equal(tc.expected))
			}
		})

		It("Should handle POSIX group with group ID", func() {
			groupID := int32(2001)

			spec := LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name:      "test-server",
					Namespace: "default",
				},
				GroupName: "posixgroup",
				GroupType: GroupTypePosix,
				GroupID:   &groupID,
				Members:   []string{"user1", "user2"},
			}

			Expect(spec.GroupType).To(Equal(GroupTypePosix))
			Expect(*spec.GroupID).To(Equal(int32(2001)))
			Expect(spec.Members).To(ContainElement("user1"))
			Expect(spec.Members).To(ContainElement("user2"))
		})
	})

	Context("Status Types", func() {
		It("Should handle LDAPServerStatus", func() {
			status := LDAPServerStatus{
				ConnectionStatus: ConnectionStatusConnected,
				LastChecked:      &metav1.Time{Time: metav1.Now().Time},
				Message:          "Connected successfully",
			}

			Expect(status.ConnectionStatus).To(Equal(ConnectionStatusConnected))
			Expect(status.Message).To(Equal("Connected successfully"))
			Expect(status.LastChecked).NotTo(BeNil())
		})

		It("Should handle LDAPUserStatus", func() {
			status := LDAPUserStatus{
				Phase:        UserPhasePending,
				LastModified: &metav1.Time{Time: metav1.Now().Time},
				Message:      "User creation pending",
				DN:           "uid=testuser,ou=users,dc=example,dc=com",
				Groups:       []string{"group1", "group2"},
			}

			Expect(status.Phase).To(Equal(UserPhasePending))
			Expect(status.Message).To(Equal("User creation pending"))
			Expect(status.DN).To(Equal("uid=testuser,ou=users,dc=example,dc=com"))
			Expect(status.Groups).To(ContainElement("group1"))
		})

		It("Should handle LDAPGroupStatus", func() {
			status := LDAPGroupStatus{
				Phase:        GroupPhasePending,
				LastModified: &metav1.Time{Time: metav1.Now().Time},
				Message:      "Group creation pending",
				DN:           "cn=testgroup,ou=groups,dc=example,dc=com",
				Members:      []string{"user1", "user2"},
				MemberCount:  2,
			}

			Expect(status.Phase).To(Equal(GroupPhasePending))
			Expect(status.Message).To(Equal("Group creation pending"))
			Expect(status.DN).To(Equal("cn=testgroup,ou=groups,dc=example,dc=com"))
			Expect(status.Members).To(ContainElement("user1"))
			Expect(status.MemberCount).To(Equal(int32(2)))
		})
	})

	Context("Reference Types", func() {
		It("Should validate LDAPServerReference", func() {
			ref := LDAPServerReference{
				Name:      "test-server",
				Namespace: "test-namespace",
			}

			Expect(ref.Name).To(Equal("test-server"))
			Expect(ref.Namespace).To(Equal("test-namespace"))
		})

		It("Should validate SecretReference", func() {
			ref := SecretReference{
				Name: "test-secret",
				Key:  "password",
			}

			Expect(ref.Name).To(Equal("test-secret"))
			Expect(ref.Key).To(Equal("password"))
		})
	})

	Context("TLS Configuration", func() {
		It("Should handle complete TLS config", func() {
			tlsConfig := TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: false,
				CACertSecret: &SecretReference{
					Name: "ca-cert",
					Key:  "ca.crt",
				},
				ClientCertSecret: &SecretReference{
					Name: "client-cert",
					Key:  "tls.crt",
				},
				ClientKeySecret: &SecretReference{
					Name: "client-key",
					Key:  "tls.key",
				},
			}

			Expect(tlsConfig.Enabled).To(BeTrue())
			Expect(tlsConfig.InsecureSkipVerify).To(BeFalse())
			Expect(tlsConfig.CACertSecret.Name).To(Equal("ca-cert"))
			Expect(tlsConfig.ClientCertSecret.Name).To(Equal("client-cert"))
			Expect(tlsConfig.ClientKeySecret.Name).To(Equal("client-key"))
		})

		It("Should handle insecure TLS config", func() {
			tlsConfig := TLSConfig{
				Enabled:            true,
				InsecureSkipVerify: true,
			}

			Expect(tlsConfig.Enabled).To(BeTrue())
			Expect(tlsConfig.InsecureSkipVerify).To(BeTrue())
			Expect(tlsConfig.CACertSecret).To(BeNil())
		})
	})

	Context("Edge Cases and Validation", func() {
		It("Should handle empty server reference", func() {
			ref := LDAPServerReference{}

			Expect(ref.Name).To(BeEmpty())
			Expect(ref.Namespace).To(BeEmpty())
		})

		It("Should handle minimal user spec", func() {
			spec := LDAPUserSpec{
				Username: "minimaluser",
			}

			Expect(spec.Username).To(Equal("minimaluser"))
			Expect(spec.Email).To(BeEmpty())
			Expect(spec.FirstName).To(BeEmpty())
			Expect(spec.LastName).To(BeEmpty())
		})

		It("Should handle minimal group spec", func() {
			spec := LDAPGroupSpec{
				GroupName: "minimalgroup",
			}

			Expect(spec.GroupName).To(Equal("minimalgroup"))
			Expect(spec.Description).To(BeEmpty())
			Expect(spec.Members).To(BeEmpty())
		})

		It("Should validate connection statuses", func() {
			statuses := []ConnectionStatus{
				ConnectionStatusConnected,
				ConnectionStatusDisconnected,
				ConnectionStatusError,
				ConnectionStatusUnknown,
			}

			for _, status := range statuses {
				Expect(string(status)).NotTo(BeEmpty())
			}
		})

		It("Should validate group types", func() {
			groupTypes := []GroupType{
				GroupTypePosix,
				GroupTypeGroupOfNames,
				GroupTypeGroupOfUniqueNames,
			}

			for _, groupType := range groupTypes {
				Expect(string(groupType)).NotTo(BeEmpty())
			}
		})
	})
})

package v1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaults(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API v1 Defaults and DeepCopy Suite")
}

var _ = Describe("SetDefaults Functions", func() {
	Context("LDAPServer SetDefaults", func() {
		It("Should set default port 389 for non-TLS connections", func() {
			spec := &LDAPServerSpec{
				Host:   "localhost",
				BindDN: "cn=admin,dc=example,dc=com",
				BaseDN: "dc=example,dc=com",
			}

			spec.SetDefaults()

			Expect(spec.Port).To(Equal(int32(389)))
			Expect(spec.ConnectionTimeout).To(Equal(int32(30)))
		})

		It("Should set default port 636 for TLS connections", func() {
			spec := &LDAPServerSpec{
				Host:   "localhost",
				BindDN: "cn=admin,dc=example,dc=com",
				BaseDN: "dc=example,dc=com",
				TLS: &TLSConfig{
					Enabled: true,
				},
			}

			spec.SetDefaults()

			Expect(spec.Port).To(Equal(int32(636)))
			Expect(spec.ConnectionTimeout).To(Equal(int32(30)))
		})

		It("Should not override existing port", func() {
			spec := &LDAPServerSpec{
				Host:   "localhost",
				Port:   1389,
				BindDN: "cn=admin,dc=example,dc=com",
				BaseDN: "dc=example,dc=com",
			}

			spec.SetDefaults()

			Expect(spec.Port).To(Equal(int32(1389)))
		})

		It("Should not override existing connection timeout", func() {
			spec := &LDAPServerSpec{
				Host:              "localhost",
				ConnectionTimeout: 60,
				BindDN:            "cn=admin,dc=example,dc=com",
				BaseDN:            "dc=example,dc=com",
			}

			spec.SetDefaults()

			Expect(spec.ConnectionTimeout).To(Equal(int32(60)))
		})
	})

	Context("LDAPUser SetDefaults", func() {
		It("Should set default organizational unit", func() {
			spec := &LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "test-server",
				},
				Username: "testuser",
			}

			spec.SetDefaults()

			Expect(spec.OrganizationalUnit).To(Equal("users"))
			Expect(spec.Enabled).ToNot(BeNil())
			Expect(*spec.Enabled).To(BeTrue())
		})

		It("Should not override existing organizational unit", func() {
			spec := &LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "test-server",
				},
				Username:           "testuser",
				OrganizationalUnit: "staff",
			}

			spec.SetDefaults()

			Expect(spec.OrganizationalUnit).To(Equal("staff"))
		})

		It("Should not override existing enabled value", func() {
			enabled := false
			spec := &LDAPUserSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "test-server",
				},
				Username: "testuser",
				Enabled:  &enabled,
			}

			spec.SetDefaults()

			Expect(spec.Enabled).ToNot(BeNil())
			Expect(*spec.Enabled).To(BeFalse())
		})
	})

	Context("LDAPGroup SetDefaults", func() {
		It("Should set default organizational unit and group type", func() {
			spec := &LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "test-server",
				},
				GroupName: "testgroup",
			}

			spec.SetDefaults()

			Expect(spec.OrganizationalUnit).To(Equal("groups"))
			Expect(spec.GroupType).To(Equal(GroupTypeGroupOfNames))
		})

		It("Should not override existing organizational unit", func() {
			spec := &LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "test-server",
				},
				GroupName:          "testgroup",
				OrganizationalUnit: "teams",
			}

			spec.SetDefaults()

			Expect(spec.OrganizationalUnit).To(Equal("teams"))
		})

		It("Should not override existing group type", func() {
			spec := &LDAPGroupSpec{
				LDAPServerRef: LDAPServerReference{
					Name: "test-server",
				},
				GroupName: "testgroup",
				GroupType: GroupTypePosix,
			}

			spec.SetDefaults()

			Expect(spec.GroupType).To(Equal(GroupTypePosix))
		})
	})
})

var _ = Describe("DeepCopy Functions", func() {
	Context("LDAPServer DeepCopy", func() {
		It("Should create a deep copy of LDAPServer", func() {
			original := &LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server",
					Namespace: "default",
					Labels: map[string]string{
						"app": "ldap",
					},
				},
				Spec: LDAPServerSpec{
					Host:   "localhost",
					Port:   389,
					BindDN: "cn=admin,dc=example,dc=com",
					BaseDN: "dc=example,dc=com",
					TLS: &TLSConfig{
						Enabled: true,
					},
				},
			}

			copy := original.DeepCopy()

			Expect(copy).NotTo(BeIdenticalTo(original))
			Expect(copy.Name).To(Equal(original.Name))
			Expect(copy.Spec.Host).To(Equal(original.Spec.Host))
			Expect(copy.Spec.TLS).NotTo(BeIdenticalTo(original.Spec.TLS))
			Expect(copy.Spec.TLS.Enabled).To(Equal(original.Spec.TLS.Enabled))

			// Modify copy and ensure original is unchanged
			copy.Spec.Host = "changed"
			copy.Spec.TLS.Enabled = false
			Expect(original.Spec.Host).To(Equal("localhost"))
			Expect(original.Spec.TLS.Enabled).To(BeTrue())
		})

		It("Should handle nil TLS config in DeepCopy", func() {
			original := &LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server",
					Namespace: "default",
				},
				Spec: LDAPServerSpec{
					Host:   "localhost",
					Port:   389,
					BindDN: "cn=admin,dc=example,dc=com",
					BaseDN: "dc=example,dc=com",
					TLS:    nil,
				},
			}

			copy := original.DeepCopy()
			Expect(copy.Spec.TLS).To(BeNil())
		})
	})

	Context("LDAPUser DeepCopy", func() {
		It("Should create a deep copy of LDAPUser", func() {
			original := &LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: LDAPUserSpec{
					LDAPServerRef: LDAPServerReference{
						Name: "test-server",
					},
					Username: "testuser",
					Groups:   []string{"group1", "group2"},
					UserID:   func(i int32) *int32 { return &i }(1001),
					GroupID:  func(i int32) *int32 { return &i }(1001),
				},
			}

			copy := original.DeepCopy()

			Expect(copy).NotTo(BeIdenticalTo(original))
			Expect(copy.Spec.Username).To(Equal(original.Spec.Username))
			Expect(copy.Spec.Groups).To(Equal(original.Spec.Groups))
			Expect(copy.Spec.Groups).NotTo(BeIdenticalTo(original.Spec.Groups))
			Expect(*copy.Spec.UserID).To(Equal(*original.Spec.UserID))
			Expect(copy.Spec.UserID).NotTo(BeIdenticalTo(original.Spec.UserID))

			// Modify copy and ensure original is unchanged
			copy.Spec.Groups[0] = "changed"
			*copy.Spec.UserID = 2001
			Expect(original.Spec.Groups[0]).To(Equal("group1"))
			Expect(*original.Spec.UserID).To(Equal(int32(1001)))
		})
	})

	Context("LDAPGroup DeepCopy", func() {
		It("Should create a deep copy of LDAPGroup", func() {
			original := &LDAPGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
				Spec: LDAPGroupSpec{
					LDAPServerRef: LDAPServerReference{
						Name: "test-server",
					},
					GroupName: "testgroup",
					Members:   []string{"user1", "user2"},
					GroupID:   func(i int32) *int32 { return &i }(2001),
				},
			}

			copy := original.DeepCopy()

			Expect(copy).NotTo(BeIdenticalTo(original))
			Expect(copy.Spec.GroupName).To(Equal(original.Spec.GroupName))
			Expect(copy.Spec.Members).To(Equal(original.Spec.Members))
			Expect(copy.Spec.Members).NotTo(BeIdenticalTo(original.Spec.Members))
			Expect(*copy.Spec.GroupID).To(Equal(*original.Spec.GroupID))
			Expect(copy.Spec.GroupID).NotTo(BeIdenticalTo(original.Spec.GroupID))

			// Modify copy and ensure original is unchanged
			copy.Spec.Members[0] = "changed"
			*copy.Spec.GroupID = 3001
			Expect(original.Spec.Members[0]).To(Equal("user1"))
			Expect(*original.Spec.GroupID).To(Equal(int32(2001)))
		})
	})

	Context("DeepCopyObject functions", func() {
		It("Should work with LDAPServer", func() {
			original := &LDAPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-server",
					Namespace: "default",
				},
			}

			copyObj := original.DeepCopyObject()
			copy, ok := copyObj.(*LDAPServer)
			Expect(ok).To(BeTrue())
			Expect(copy.Name).To(Equal(original.Name))
		})

		It("Should work with LDAPUser", func() {
			original := &LDAPUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
			}

			copyObj := original.DeepCopyObject()
			copy, ok := copyObj.(*LDAPUser)
			Expect(ok).To(BeTrue())
			Expect(copy.Name).To(Equal(original.Name))
		})

		It("Should work with LDAPGroup", func() {
			original := &LDAPGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-group",
					Namespace: "default",
				},
			}

			copyObj := original.DeepCopyObject()
			copy, ok := copyObj.(*LDAPGroup)
			Expect(ok).To(BeTrue())
			Expect(copy.Name).To(Equal(original.Name))
		})
	})
})

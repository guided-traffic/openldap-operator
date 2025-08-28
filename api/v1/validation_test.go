package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation Functions", func() {

	Describe("isValidShell", func() {
		It("Should accept standard bash shell", func() {
			Expect(isValidShell("/bin/bash")).To(BeTrue())
		})

		It("Should accept standard sh shell", func() {
			Expect(isValidShell("/bin/sh")).To(BeTrue())
		})

		It("Should accept zsh shell", func() {
			Expect(isValidShell("/bin/zsh")).To(BeTrue())
		})

		It("Should accept fish shell", func() {
			Expect(isValidShell("/bin/fish")).To(BeTrue())
		})

		It("Should accept tcsh shell", func() {
			Expect(isValidShell("/bin/tcsh")).To(BeTrue())
		})

		It("Should accept csh shell", func() {
			Expect(isValidShell("/bin/csh")).To(BeTrue())
		})

		It("Should accept usr bin shells", func() {
			Expect(isValidShell("/usr/bin/bash")).To(BeTrue())
			Expect(isValidShell("/usr/bin/sh")).To(BeTrue())
			Expect(isValidShell("/usr/bin/zsh")).To(BeTrue())
			Expect(isValidShell("/usr/bin/fish")).To(BeTrue())
			Expect(isValidShell("/usr/bin/tcsh")).To(BeTrue())
			Expect(isValidShell("/usr/bin/csh")).To(BeTrue())
		})

		It("Should accept disabled account shells", func() {
			Expect(isValidShell("/bin/false")).To(BeTrue())
			Expect(isValidShell("/sbin/nologin")).To(BeTrue())
		})

		It("Should reject invalid shells", func() {
			Expect(isValidShell("/bin/invalid")).To(BeFalse())
			Expect(isValidShell("/usr/bin/invalid")).To(BeFalse())
			Expect(isValidShell("/invalid/path")).To(BeFalse())
			Expect(isValidShell("bash")).To(BeFalse()) // Without full path
			Expect(isValidShell("")).To(BeFalse())     // Empty string
		})

		It("Should reject malicious shells", func() {
			Expect(isValidShell("/bin/bash; rm -rf /")).To(BeFalse())
			Expect(isValidShell("/bin/sh && malicious")).To(BeFalse())
			Expect(isValidShell("/bin/bash | evil")).To(BeFalse())
		})

		It("Should be case sensitive", func() {
			Expect(isValidShell("/BIN/BASH")).To(BeFalse())
			Expect(isValidShell("/Bin/Bash")).To(BeFalse())
		})
	})

	Describe("isValidGroupName", func() {
		It("Should accept valid group names", func() {
			Expect(isValidGroupName("users")).To(BeTrue())
			Expect(isValidGroupName("administrators")).To(BeTrue())
			Expect(isValidGroupName("group1")).To(BeTrue())
			Expect(isValidGroupName("test-group")).To(BeTrue())
			Expect(isValidGroupName("group_name")).To(BeTrue())
			Expect(isValidGroupName("group.name")).To(BeTrue())
		})

		It("Should reject invalid group names", func() {
			Expect(isValidGroupName("")).To(BeFalse())
			Expect(isValidGroupName("group with spaces")).To(BeFalse())
			Expect(isValidGroupName("group@invalid")).To(BeFalse())
			Expect(isValidGroupName("group#invalid")).To(BeFalse())
			Expect(isValidGroupName("group$invalid")).To(BeFalse())
		})

		It("Should handle special characters", func() {
			Expect(isValidGroupName("group!")).To(BeFalse())
			Expect(isValidGroupName("group%")).To(BeFalse())
			Expect(isValidGroupName("group^")).To(BeFalse())
			Expect(isValidGroupName("group&")).To(BeFalse())
			Expect(isValidGroupName("group*")).To(BeFalse())
		})
	})

	Describe("isValidUsername", func() {
		It("Should accept valid usernames", func() {
			Expect(isValidUsername("user")).To(BeTrue())
			Expect(isValidUsername("testuser")).To(BeTrue())
			Expect(isValidUsername("user1")).To(BeTrue())
			Expect(isValidUsername("test-user")).To(BeTrue())
			Expect(isValidUsername("user_name")).To(BeTrue())
			Expect(isValidUsername("user.name")).To(BeTrue())
		})

		It("Should reject invalid usernames", func() {
			Expect(isValidUsername("")).To(BeFalse())
			Expect(isValidUsername("user with spaces")).To(BeFalse())
			Expect(isValidUsername("user@invalid")).To(BeFalse())
			Expect(isValidUsername("user#invalid")).To(BeFalse())
			Expect(isValidUsername("user$invalid")).To(BeFalse())
		})

		It("Should handle special characters", func() {
			Expect(isValidUsername("user!")).To(BeFalse())
			Expect(isValidUsername("user%")).To(BeFalse())
			Expect(isValidUsername("user^")).To(BeFalse())
			Expect(isValidUsername("user&")).To(BeFalse())
			Expect(isValidUsername("user*")).To(BeFalse())
		})
	})

	Describe("isValidGroupType", func() {
		It("Should accept valid group types", func() {
			Expect(isValidGroupType(GroupTypePosix)).To(BeTrue())
			Expect(isValidGroupType(GroupTypeGroupOfNames)).To(BeTrue())
			Expect(isValidGroupType(GroupTypeGroupOfUniqueNames)).To(BeTrue())
		})

		It("Should reject invalid group types", func() {
			Expect(isValidGroupType(GroupType("invalid"))).To(BeFalse())
			Expect(isValidGroupType(GroupType(""))).To(BeFalse())
			Expect(isValidGroupType(GroupType("customType"))).To(BeFalse())
		})
	})
})

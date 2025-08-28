package ldap

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("LDAP Client CRUD Operations", func() {
	BeforeEach(func() {
		if !IsDockerAvailable() {
			Skip("Docker is not available, skipping LDAP CRUD tests")
		}
	})

	// Note: These tests now use the real Docker-based LDAP server
	// See client_integration_test.go for the actual implementation
})

package ldap

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"

	v1 "github.com/guided-traffic/openldap-operator/api/v1"
)

const (
	// Using osixia/openldap - faster startup for tests
	ldapImage         = "osixia/openldap:1.5.0"
	ldapContainerName = "test-openldap-container"
	ldapPort          = "1389"
	ldapTLSPort       = "1636"
	adminPassword     = "admin"
	adminUser         = "admin"
	baseDN            = "dc=example,dc=com"
)

// LDAPTestContainer manages a Docker container for LDAP testing
type LDAPTestContainer struct {
	containerName string
	running       bool
}

// NewLDAPTestContainer creates a new LDAP test container manager
func NewLDAPTestContainer() *LDAPTestContainer {
	return &LDAPTestContainer{
		containerName: ldapContainerName,
		running:       false,
	}
}

// Start starts the LDAP Docker container
func (c *LDAPTestContainer) Start() error {
	By("Starting LDAP Docker container")

	// Stop any existing container first
	c.Stop()

	// Remove any existing container
	exec.Command("docker", "rm", "-f", c.containerName).Run()

	// Start new container
	cmd := exec.Command("docker", "run", "-d",
		"--name", c.containerName,
		"-p", fmt.Sprintf("%s:389", ldapPort),
		"-p", fmt.Sprintf("%s:636", ldapTLSPort),
		"-e", "LDAP_ORGANISATION=Example Inc.",
		"-e", "LDAP_DOMAIN=example.com",
		"-e", fmt.Sprintf("LDAP_ADMIN_PASSWORD=%s", adminPassword),
		"-e", "LDAP_CONFIG_PASSWORD=config",
		"-e", "LDAP_READONLY_USER=false",
		"-e", "LDAP_RFC2307BIS_SCHEMA=false",
		"-e", "LDAP_BACKEND=mdb",
		"-e", "LDAP_TLS=true",
		"-e", "LDAP_TLS_CRT_FILENAME=ldap.crt",
		"-e", "LDAP_TLS_KEY_FILENAME=ldap.key",
		"-e", "LDAP_TLS_DH_PARAM_FILENAME=dhparam.pem",
		"-e", "LDAP_TLS_CA_CRT_FILENAME=ca.crt",
		"-e", "LDAP_TLS_ENFORCE=false",
		"-e", "LDAP_TLS_CIPHER_SUITE=SECURE256:-VERS-SSL3.0",
		"-e", "LDAP_TLS_VERIFY_CLIENT=demand",
		"-e", "LDAP_REPLICATION=false",
		"--rm",
		ldapImage,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start LDAP container: %v, output: %s", err, string(output))
	}

	c.running = true

	// Wait for container to be ready
	return c.waitForReady()
}

// Stop stops and removes the LDAP Docker container
func (c *LDAPTestContainer) Stop() error {
	if !c.running {
		return nil
	}

	By("Stopping LDAP Docker container")

	// Stop container
	cmd := exec.Command("docker", "stop", c.containerName)
	cmd.Run() // Ignore errors

	// Remove container
	cmd = exec.Command("docker", "rm", "-f", c.containerName)
	cmd.Run() // Ignore errors

	c.running = false
	return nil
}

// waitForReady waits for the LDAP container to be ready to accept connections
func (c *LDAPTestContainer) waitForReady() error {
	By("Waiting for LDAP container to be ready")

	timeout := time.After(120 * time.Second)  // Increased timeout for LDAP startup
	ticker := time.NewTicker(3 * time.Second) // Less frequent checks
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Get container logs for debugging
			cmd := exec.Command("docker", "logs", c.containerName)
			logs, _ := cmd.Output()
			return fmt.Errorf("timeout waiting for LDAP container to be ready. Container logs:\n%s", string(logs))
		case <-ticker.C:
			if c.isReady() {
				By("LDAP container is ready")
				// Wait a bit more to ensure LDAP is fully initialized
				time.Sleep(5 * time.Second)
				return nil
			}
		}
	}
}

// isReady checks if the LDAP container is ready
func (c *LDAPTestContainer) isReady() bool {
	// Check if container is running
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", c.containerName), "--format", "{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	status := strings.TrimSpace(string(output))
	if !strings.HasPrefix(status, "Up") {
		return false
	}

	// Try to connect using netcat first
	cmd = exec.Command("nc", "-z", "localhost", ldapPort)
	if cmd.Run() != nil {
		return false
	}

	// Try a simple LDAP search to verify the server is responding
	cmd = exec.Command("docker", "exec", c.containerName, "ldapsearch",
		"-x", "-H", "ldap://localhost:389",
		"-D", "cn=admin,dc=example,dc=com",
		"-w", adminPassword,
		"-b", baseDN,
		"-s", "base", "(objectclass=*)")

	return cmd.Run() == nil
}

// GetConnectionSpec returns the connection spec for the test container
func (c *LDAPTestContainer) GetConnectionSpec() *v1.LDAPServerSpec {
	return &v1.LDAPServerSpec{
		Host:   "localhost",
		Port:   1389,
		BindDN: "cn=admin,dc=example,dc=com",
		BaseDN: baseDN,
		TLS: &v1.TLSConfig{
			Enabled: false,
		},
	}
}

// GetTLSConnectionSpec returns the TLS connection spec for the test container
func (c *LDAPTestContainer) GetTLSConnectionSpec() *v1.LDAPServerSpec {
	return &v1.LDAPServerSpec{
		Host:   "localhost",
		Port:   1636,
		BindDN: "cn=admin,dc=example,dc=com",
		BaseDN: baseDN,
		TLS: &v1.TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: true, // For testing
		},
	}
}

// GetAdminPassword returns the admin password for the test container
func (c *LDAPTestContainer) GetAdminPassword() string {
	return adminPassword
}

// IsDockerAvailable checks if Docker is available
func IsDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	return err == nil
}

// EnsureDockerAvailable ensures Docker is available or skips the test
func EnsureDockerAvailable() {
	if !IsDockerAvailable() {
		Skip("Docker is not available, skipping LDAP integration tests")
	}
}

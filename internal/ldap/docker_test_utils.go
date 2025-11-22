package ldap

import (
	"fmt"
	"math/rand"
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
	port          string
	tlsPort       string
	running       bool
}

// NewLDAPTestContainer creates a new LDAP test container manager
func NewLDAPTestContainer() *LDAPTestContainer {
	// Generate unique container name with timestamp and random number
	timestamp := time.Now().UnixNano()
	random := rand.New(rand.NewSource(timestamp)).Intn(10000) // #nosec G404 -- not used for cryptographic purposes
	uniqueName := fmt.Sprintf("%s-%d-%d", ldapContainerName, timestamp, random)

	return &LDAPTestContainer{
		containerName: uniqueName,
		port:          ldapPort,
		tlsPort:       ldapTLSPort,
		running:       false,
	}
}

// Start starts the LDAP Docker container
func (c *LDAPTestContainer) Start() error {
	By("Starting LDAP Docker container")

	// Clean up any existing containers using the same ports
	c.cleanupExistingContainers()

	// Remove any existing container with the same name
	_ = exec.Command("docker", "rm", "-f", c.containerName).Run() // #nosec G204 -- containerName is managed internally

	// Wait a bit to ensure port is released
	time.Sleep(500 * time.Millisecond)

	// Start new container
	// #nosec G204 -- Using trusted container image and sanitized inputs
	cmd := exec.Command("docker", "run", "-d",
		"--name", c.containerName,
		"-p", fmt.Sprintf("%s:389", c.port),
		"-p", fmt.Sprintf("%s:636", c.tlsPort),
		"-e", "LDAP_ORGANIZATION=Example Inc.",
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

// cleanupExistingContainers removes any containers that might be using our ports
func (c *LDAPTestContainer) cleanupExistingContainers() {
	// Find containers using port 1389
	cmd := exec.Command("sh", "-c", fmt.Sprintf("docker ps -q --filter publish=%s", c.port))
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		containerIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, id := range containerIDs {
			if id != "" {
				_ = exec.Command("docker", "rm", "-f", id).Run() // #nosec G204 -- containerID from docker command
			}
		}
		// Wait for cleanup
		time.Sleep(1 * time.Second)
	}
}

// Stop stops and removes the LDAP Docker container
func (c *LDAPTestContainer) Stop() error {
	if !c.running {
		return nil
	}

	By("Stopping LDAP Docker container")

	// Force remove container (this also stops it)
	cmd := exec.Command("docker", "rm", "-f", c.containerName) // #nosec G204 -- containerName is managed internally
	_ = cmd.Run()                                              // Ignore errors

	c.running = false

	// Wait to ensure port is released
	time.Sleep(500 * time.Millisecond)

	return nil
}

// waitForReady waits for the LDAP container to be ready to accept connections
func (c *LDAPTestContainer) waitForReady() error {
	By("Waiting for LDAP container to be ready")

	timeout := time.After(180 * time.Second)  // Increased timeout for CI environments
	ticker := time.NewTicker(2 * time.Second) // Check every 2 seconds
	defer ticker.Stop()

	// First wait for "First start is done" message
	startDoneTimeout := time.After(150 * time.Second)
	startDoneTicker := time.NewTicker(2 * time.Second)
	defer startDoneTicker.Stop()

	By("Waiting for LDAP initial setup to complete")
	for {
		select {
		case <-startDoneTimeout:
			cmd := exec.Command("docker", "logs", c.containerName) // #nosec G204 -- containerName is managed internally
			logs, _ := cmd.Output()
			return fmt.Errorf("timeout waiting for LDAP initial setup. Container logs:\n%s", string(logs))
		case <-startDoneTicker.C:
			cmd := exec.Command("docker", "logs", c.containerName) // #nosec G204 -- containerName is managed internally
			logOutput, err := cmd.Output()
			if err == nil && strings.Contains(string(logOutput), "First start is done") {
				By("LDAP initial setup completed, waiting for service to be ready")
				goto waitForService
			}
		}
	}

waitForService:
	// Now wait for LDAP service to actually accept connections
	time.Sleep(5 * time.Second) // Give it a moment to start the service

	for {
		select {
		case <-timeout:
			// Get container logs for debugging
			cmd := exec.Command("docker", "logs", c.containerName) // #nosec G204 -- containerName is managed internally
			logs, _ := cmd.Output()
			return fmt.Errorf("timeout waiting for LDAP service to accept connections. Container logs:\n%s", string(logs))
		case <-ticker.C:
			if c.isServiceReady() {
				By("LDAP container is ready and accepting connections")
				return nil
			}
		}
	}
}

// isServiceReady checks if the LDAP service is accepting connections
func (c *LDAPTestContainer) isServiceReady() bool {
	// Check if container is still running
	// #nosec G204 -- containerName is managed internally
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", c.containerName), "--format", "{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	status := strings.TrimSpace(string(output))
	if !strings.HasPrefix(status, "Up") {
		return false
	}

	// Try a simple LDAP search to verify the server is responding
	// #nosec G204 -- All parameters are constants or sanitized
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
	// Parse port as int32
	var port int32
	fmt.Sscanf(c.port, "%d", &port)

	return &v1.LDAPServerSpec{
		Host:   "localhost",
		Port:   port,
		BindDN: "cn=admin,dc=example,dc=com",
		BaseDN: baseDN,
		TLS: &v1.TLSConfig{
			Enabled: false,
		},
	}
}

// GetTLSConnectionSpec returns the TLS connection spec for the test container
func (c *LDAPTestContainer) GetTLSConnectionSpec() *v1.LDAPServerSpec {
	// Parse port as int32
	var tlsPort int32
	fmt.Sscanf(c.tlsPort, "%d", &tlsPort)

	return &v1.LDAPServerSpec{
		Host:   "localhost",
		Port:   tlsPort,
		BindDN: "cn=admin,dc=example,dc=com",
		BaseDN: baseDN,
		TLS: &v1.TLSConfig{
			Enabled:            true,
			InsecureSkipVerify: true, // For testing
		},
	}
} // GetAdminPassword returns the admin password for the test container
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

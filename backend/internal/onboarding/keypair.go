package onboarding

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// KeyPair holds SSH public/private key pair
type KeyPair struct {
	PrivateKeyPEM []byte
	PublicKeySSH  []byte
	Fingerprint   string
}

// GenerateKeyPair creates a new RSA SSH key pair
func GenerateKeyPair(bitSize int) (*KeyPair, error) {
	if bitSize == 0 {
		bitSize = 2048 // default
	}

	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Validate private key
	if err := privateKey.Validate(); err != nil {
		return nil, fmt.Errorf("private key validation failed: %w", err)
	}

	// Encode private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Generate public key in SSH format
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	publicKeySSH := ssh.MarshalAuthorizedKey(publicKey)
	fingerprint := ssh.FingerprintSHA256(publicKey)

	return &KeyPair{
		PrivateKeyPEM: privateKeyPEM,
		PublicKeySSH:  publicKeySSH,
		Fingerprint:   fingerprint,
	}, nil
}

// GenerateSetupScript creates the onboarding script for Linux server
func GenerateSetupScript(serverName, publicKeySSH string) string {
	return fmt.Sprintf(`#!/bin/bash
# Archon Server Onboarding Script
# Server: %s
# Generated: automatically by Archon dashboard

set -e

echo "=== Archon Server Setup ==="
echo "Creating user 'archon' and configuring SSH access..."

# Create archon user if not exists
if ! id -u archon > /dev/null 2>&1; then
    sudo useradd -m -s /bin/bash archon
    echo "✓ User 'archon' created"
else
    echo "✓ User 'archon' already exists"
fi

# Setup SSH directory and authorized_keys
sudo mkdir -p /home/archon/.ssh
sudo chmod 700 /home/archon/.ssh

# Add public key
echo "%s" | sudo tee /home/archon/.ssh/authorized_keys > /dev/null
sudo chmod 600 /home/archon/.ssh/authorized_keys
sudo chown -R archon:archon /home/archon/.ssh
echo "✓ SSH key installed"

# Configure minimal sudo permissions for specific commands
# (Archon uses allowlisted commands, most are read-only)
sudo tee /etc/sudoers.d/archon > /dev/null << 'SUDOERS'
# Archon storage monitor - minimal sudo permissions
archon ALL=(ALL) NOPASSWD: /usr/bin/du, /usr/bin/find, /usr/bin/df
archon ALL=(ALL) NOPASSWD: /usr/bin/systemctl status *
SUDOERS

sudo chmod 440 /etc/sudoers.d/archon
echo "✓ Sudo permissions configured (read-only commands only)"

# Test SSH connectivity
echo ""
echo "=== Setup Complete ==="
echo "User 'archon' is ready for monitoring."
echo "Public key fingerprint: (check in dashboard)"
echo ""
echo "Next: Test connection from Archon dashboard."
`, serverName, publicKeySSH)
}

// DetectionScript returns script to detect OS and stack
func DetectionScript() string {
	return `#!/bin/bash
# Archon Stack Detection Script
set -e

echo "=== OS Detection ==="
if [ -f /etc/os-release ]; then
    . /etc/os-release
    echo "OS: $NAME"
    echo "VERSION: $VERSION"
elif [ -f /etc/redhat-release ]; then
    cat /etc/redhat-release
else
    uname -s
fi

echo ""
echo "=== Web Server Detection ==="
if systemctl is-active nginx >/dev/null 2>&1; then
    echo "nginx: running"
elif systemctl is-active apache2 >/dev/null 2>&1; then
    echo "apache2: running"
elif systemctl is-active httpd >/dev/null 2>&1; then
    echo "httpd: running"
else
    echo "none detected"
fi

echo ""
echo "=== Database Detection ==="
if systemctl is-active postgresql >/dev/null 2>&1; then
    echo "postgresql: running"
fi
if systemctl is-active mysql >/dev/null 2>&1; then
    echo "mysql: running"
fi
if systemctl is-active mariadb >/dev/null 2>&1; then
    echo "mariadb: running"
fi

echo ""
echo "=== Docker Detection ==="
if command -v docker >/dev/null 2>&1; then
    docker --version 2>/dev/null || echo "docker: installed (not running)"
else
    echo "docker: not installed"
fi

echo ""
echo "=== Disk Info ==="
df -h / | tail -1
`
}

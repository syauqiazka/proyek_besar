package ssh

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectionConfig holds SSH connection details
type ConnectionConfig struct {
	Host               string
	Port               int
	Username           string
	PrivateKey         []byte // decrypted private key
	HostFingerprint    string // expected host key fingerprint
	Timeout            time.Duration
	MaxOutputBytes     int64
}

// ExecuteResult holds command execution results
type ExecuteResult struct {
	Command    string
	ExitCode   int
	Stdout     string
	Stderr     string
	DurationMs int64
	Error      error
}

// SSHExecutor handles SSH connections and command execution
type SSHExecutor struct {
	config *ConnectionConfig
	client *ssh.Client
}

// NewExecutor creates a new SSH executor
func NewExecutor(config *ConnectionConfig) (*SSHExecutor, error) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second // default timeout
	}
	if config.MaxOutputBytes == 0 {
		config.MaxOutputBytes = 1024 * 1024 // 1MB default
	}

	return &SSHExecutor{config: config}, nil
}

// Connect establishes SSH connection and verifies host key
func (e *SSHExecutor) Connect() error {
	// Parse private key
	signer, err := ssh.ParsePrivateKey(e.config.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// SSH client config
	sshConfig := &ssh.ClientConfig{
		User: e.config.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		Timeout: e.config.Timeout,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Verify host key fingerprint (prinsip #5: verifikasi host key)
			fingerprint := ssh.FingerprintSHA256(key)
			
			if e.config.HostFingerprint == "" {
				// First connection: save fingerprint (caller should persist this)
				e.config.HostFingerprint = fingerprint
				return nil
			}

			// Subsequent connections: verify fingerprint matches
			if fingerprint != e.config.HostFingerprint {
				return fmt.Errorf("host key verification failed: expected %s, got %s", 
					e.config.HostFingerprint, fingerprint)
			}

			return nil
		},
	}

	// Connect
	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH connection failed to %s: %w", addr, err)
	}

	e.client = client
	return nil
}

// Execute runs a command with timeout and output limits
func (e *SSHExecutor) Execute(command string) (*ExecuteResult, error) {
	if e.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	startTime := time.Now()
	result := &ExecuteResult{
		Command: command,
	}

	// Create session
	session, err := e.client.NewSession()
	if err != nil {
		result.Error = fmt.Errorf("failed to create session: %w", err)
		return result, result.Error
	}
	defer session.Close()

	// Capture stdout/stderr with size limits
	var stdoutBuf, stderrBuf bytes.Buffer
	limitedStdout := &limitedWriter{w: &stdoutBuf, limit: e.config.MaxOutputBytes}
	limitedStderr := &limitedWriter{w: &stderrBuf, limit: e.config.MaxOutputBytes}
	
	session.Stdout = limitedStdout
	session.Stderr = limitedStderr

	// Execute with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case err := <-done:
		// Command finished
		result.Stdout = stdoutBuf.String()
		result.Stderr = stderrBuf.String()
		result.DurationMs = time.Since(startTime).Milliseconds()

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				result.Error = err
			}
		} else {
			result.ExitCode = 0
		}

	case <-time.After(e.config.Timeout):
		// Timeout
		session.Signal(ssh.SIGKILL)
		session.Close()
		result.Error = fmt.Errorf("command timeout after %v", e.config.Timeout)
		result.DurationMs = time.Since(startTime).Milliseconds()
	}

	return result, result.Error
}

// ExecuteTemplate renders and executes a template command with validated params
func (e *SSHExecutor) ExecuteTemplate(templateName string, params map[string]interface{}) (*ExecuteResult, error) {
	// Get command template from allowlist
	tmpl, err := GetCommand(templateName)
	if err != nil {
		return nil, err
	}

	// Validate and quote parameters
	quotedParams := make(map[string]string)
	for key, value := range params {
		strVal := fmt.Sprintf("%v", value)

		// Path validation
		if strings.HasSuffix(key, "path") {
			if err := ValidatePath(strVal, tmpl.AllowedRoots); err != nil {
				return nil, fmt.Errorf("invalid path parameter %s: %w", key, err)
			}
			quotedParams[key] = QuotePath(strVal)
		} else {
			// Generic parameter validation
			if err := ValidateParameter(strVal); err != nil {
				return nil, fmt.Errorf("invalid parameter %s: %w", key, err)
			}
			quotedParams[key] = strVal
		}
	}

	// Simple template rendering (replace {{.key}} with quoted values)
	command := tmpl.Template
	for key, quotedVal := range quotedParams {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		command = strings.ReplaceAll(command, placeholder, quotedVal)
	}

	// Check for unreplaced placeholders
	if strings.Contains(command, "{{.") {
		return nil, fmt.Errorf("missing required parameters in command template")
	}

	// Execute
	return e.Execute(command)
}

// Close closes the SSH connection
func (e *SSHExecutor) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

// GetHostFingerprint returns the verified host fingerprint
func (e *SSHExecutor) GetHostFingerprint() string {
	return e.config.HostFingerprint
}

// limitedWriter limits output to prevent memory exhaustion
type limitedWriter struct {
	w       *bytes.Buffer
	limit   int64
	written int64
}

func (lw *limitedWriter) Write(p []byte) (n int, err error) {
	remaining := lw.limit - lw.written
	if remaining <= 0 {
		return len(p), nil // drop output silently if limit exceeded
	}

	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err = lw.w.Write(p)
	lw.written += int64(n)
	return n, err
}

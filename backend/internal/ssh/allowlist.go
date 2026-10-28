package ssh

import (
	"fmt"
	"regexp"
	"strings"
)

// CommandTemplate defines an allowlisted SSH command
type CommandTemplate struct {
	Name        string
	Template    string
	Description string
	AllowedRoots []string // paths must be within these roots
	RequiresSudo bool
}

// Parameter validation patterns
var (
	// Path must be absolute, no "..", no special chars
	safePathPattern = regexp.MustCompile(`^/[a-zA-Z0-9/_.-]+$`)
	// No shell metacharacters
	noShellMetaPattern = regexp.MustCompile(`^[a-zA-Z0-9/_.:@-]+$`)
)

// Allowlist of commands (prinsip desain #7: hanya perintah allowlist)
var CommandAllowlist = map[string]CommandTemplate{
	"disk_usage": {
		Name:        "disk_usage",
		Template:    "df -h {{.mount_path}}",
		Description: "Get disk usage for mount path",
		AllowedRoots: []string{"/"},
		RequiresSudo: false,
	},
	"disk_scan": {
		Name:        "disk_scan",
		Template:    "nice -n 19 ionice -c3 du -sb {{.path}} 2>/dev/null | head -1",
		Description: "Scan disk usage of directory (with nice/ionice to reduce load)",
		AllowedRoots: []string{"/home", "/var", "/opt", "/tmp"},
		RequiresSudo: false,
	},
	"find_large_files": {
		Name:        "find_large_files",
		Template:    "nice -n 19 find {{.path}} -maxdepth {{.depth}} -type f -size +{{.min_size}}M -exec ls -lh {} \\; 2>/dev/null | head -{{.limit}}",
		Description: "Find large files (with depth limit to avoid overload)",
		AllowedRoots: []string{"/", "/home", "/var", "/opt", "/tmp"},
		RequiresSudo: false,
	},
	"check_backup": {
		Name:        "check_backup",
		Template:    "ls -lh {{.backup_path}} 2>/dev/null",
		Description: "Check backup file metadata",
		AllowedRoots: []string{"/", "/home", "/var/backups", "/opt/backups"},
		RequiresSudo: false,
	},
	"test_archive_integrity": {
		Name:        "test_archive_integrity",
		Template:    "tar -tzf {{.archive_path}} >/dev/null 2>&1 && echo 'OK' || echo 'FAILED'",
		Description: "Test tar.gz archive integrity without extracting",
		AllowedRoots: []string{"/", "/home", "/var/backups", "/opt/backups"},
		RequiresSudo: false,
	},
	"detect_os": {
		Name:        "detect_os",
		Template:    "cat /etc/os-release 2>/dev/null || cat /etc/redhat-release 2>/dev/null || uname -s",
		Description: "Detect OS distribution",
		AllowedRoots: []string{"/"},
		RequiresSudo: false,
	},
	"detect_webserver": {
		Name:        "detect_webserver",
		Template:    "systemctl is-active nginx 2>/dev/null || systemctl is-active apache2 2>/dev/null || echo 'none'",
		Description: "Detect running web server",
		AllowedRoots: []string{"/"},
		RequiresSudo: false,
	},
	"detect_docker": {
		Name:        "detect_docker",
		Template:    "docker --version 2>/dev/null || echo 'not_installed'",
		Description: "Detect Docker installation",
		AllowedRoots: []string{"/"},
		RequiresSudo: false,
	},
}

// ValidatePath ensures path is safe and within allowed roots
func ValidatePath(path string, allowedRoots []string) error {
	// Must be absolute
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// No ".." (path traversal)
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Must match safe pattern
	if !safePathPattern.MatchString(path) {
		return fmt.Errorf("path contains invalid characters: %s", path)
	}

	// Must be within allowed roots
	withinRoot := false
	for _, root := range allowedRoots {
		if strings.HasPrefix(path, root) {
			withinRoot = true
			break
		}
	}
	if !withinRoot {
		return fmt.Errorf("path outside allowed roots: %s (allowed: %v)", path, allowedRoots)
	}

	return nil
}

// ValidateParameter ensures parameter has no shell metacharacters
func ValidateParameter(param string) error {
	if !noShellMetaPattern.MatchString(param) {
		return fmt.Errorf("parameter contains shell metacharacters: %s", param)
	}
	return nil
}

// QuotePath safely quotes a path for shell execution
func QuotePath(path string) string {
	// Single quotes prevent all expansion, escape existing single quotes
	escaped := strings.ReplaceAll(path, "'", "'\\''")
	return fmt.Sprintf("'%s'", escaped)
}

// GetCommand retrieves a command template by name
func GetCommand(name string) (CommandTemplate, error) {
	cmd, exists := CommandAllowlist[name]
	if !exists {
		return CommandTemplate{}, fmt.Errorf("command not in allowlist: %s", name)
	}
	return cmd, nil
}

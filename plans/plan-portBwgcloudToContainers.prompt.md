# Port bwgcloud to Containers CLI

## Objective

Integrate the Bitwarden backup functionality into the `containers` CLI tool as a new subcommand, following the same pattern as `pdf-compress` and `ibgateway`. This provides a simple, non-interactive command to execute Bitwarden backups.

## Current Implementation

The bwgcloud project runs as a standalone Docker container with:
- **Credentials**: Retrieved from macOS Keychain via wrapper script (`run_backup.sh`)
- **Execution**: One-shot backup to local encrypted directory
- **Output**: Timestamped JSON files (`bitwarden-backup-YYYY-MM-DD-HHMMSS.json`)
- **Security**: tmpfs mounts, credential cleanup, non-root execution
- **Image**: `ghcr.io/vupham90/containers-bw-backup:latest` (follows containers project naming convention)
- **Platform**: macOS only

## Required Credentials

- `BW_CLIENTID` - Bitwarden API client ID
- `BW_CLIENTSECRET` - Bitwarden API client secret
- `BW_PASSWORD` - Bitwarden master password

## Implementation Steps

### 1. Add `bw-backup` Command to main.go

Add new command with flags:
- `--client-id` (alias: `-c`) - Bitwarden client ID
- `--client-secret` (alias: `-s`) - Bitwarden client secret
- `--password` (alias: `-p`) - Bitwarden master password
- `--backup-dir` (alias: `-d`) - Backup destination directory (default: `./backups`)

Credential precedence order:
1. CLI flags (if provided)
2. macOS Keychain (fallback)

### 2. Implement macOS Keychain Integration

Create helper function to retrieve credentials from macOS Keychain:
- Entry: `bitwarden_client_id` → `BW_CLIENTID`
- Entry: `bitwarden_client_secret` → `BW_CLIENTSECRET`
- Entry: `bitwarden_password` → `BW_PASSWORD`

Use `security find-generic-password -a <account> -w` command.

Credential precedence order:
1. CLI flags (if provided)
2. macOS Keychain (fallback)

### 3. Extend Docker Execution for tmpfs Support

Option A: Extend `RunContainer()` to accept optional tmpfs mounts
Option B: Create new `RunContainerWithTmpfs()` variant

Required tmpfs mounts for security:
- `--tmpfs /tmp`
- `--tmpfs /var/tmp`

### 4. Wire Up Command Execution

In the command's Action function:
1. Resolve backup directory (absolute path)
2. Validate backup directory exists
3. Retrieve credentials (flags → keychain)
4. Build Docker arguments:
   - Image: `ghcr.io/vupham90/containers-bw-backup:latest`
   - Volume: `-v <backup-dir>:/backups`
   - Environment: `-e BW_CLIENTID`, `-e BW_CLIENTSECRET`, `-e BW_PASSWORD`
   - Security: `--tmpfs /tmp --tmpfs /var/tmp`
5. Execute container with `RunContainer()` or variant

### 5. Update Documentation

Add usage examples to README.md:

**Basic usage:**
```bash
containers bw-backup --backup-dir ~/backups
```

**Explicit credentials:**
```bash
containers bw-backup \
  --client-id "..." \
  --client-secret "..." \
  --password "..." \
  --backup-dir ~/backups
```

### 6. Add Image Building to GitHub Actions

Update `.github/workflows/build-images.yml` to build and push the `containers-bw-backup` image:
- Use existing Dockerfile from `bwgcloud` project
- Build image as `ghcr.io/vupham90/containers-bw-backup:latest`
- Push on merge to main branch
- Trigger builds on changes to bitwarden-related files

## Design Decisions

### tmpfs Support Implementation

Extend `RunContainer()` with optional parameters for greater flexibility:
- Supports tmpfs mounts for security-sensitive commands
- Supports environment variables for configuration
- Backward compatible with existing commands (pdf-compress, ibgateway)
- Cleaner API than creating specialized function variants

## Implementation Examples

### 1. Keychain Package (keychain/keychain.go)

Create a dedicated package for macOS Keychain operations:

```go
package keychain

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetPassword retrieves a password from macOS Keychain
func GetPassword(account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-a", account, "-w")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve %s from Keychain: %w", account, err)
	}
	return strings.TrimSpace(string(output)), nil
}
```

### 2. Extended RunContainer() Function (docker.go)

```go
// Add optional parameters to RunContainer
func RunContainer(image, workDir string, args []string, env map[string]string, tmpfs []string) error {
	dockerArgs := []string{"run", "--rm"}
	
	// Add tmpfs mounts
	for _, mount := range tmpfs {
		dockerArgs = append(dockerArgs, "--tmpfs", mount)
	}
	
	// Add environment variables
	for key, value := range env {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", key, value))
	}
	
	dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/workspace", absWorkDir))
	dockerArgs = append(dockerArgs, "-w", "/workspace", image)
	dockerArgs = append(dockerArgs, args...)
	// ... execute command
}
```

### 2. Keychain Helper Function (main.go)

```go
// getKeychainPassword retrieves a password from macOS Keychain
func getKeychainPassword(account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-a", account, "-w")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve %s from Keychain", account)
	}
	return strings.TrimSpace(string(output)), nil
}
```

### 3. Bitwarden Backup Command Structure (main.go)

```go
{
	Name:  "bw-backup",
	Usage: "Backup Bitwarden vault to local directory",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "client-id", Aliases: []string{"c"}},
		&cli.StringFlag{Name: "client-secret", Aliases: []string{"s"}},
		&cli.StringFlag{Name: "password", Aliases: []string{"p"}},
		&cli.StringFlag{Name: "backup-dir", Aliases: []string{"d"}, Value: "./backups"},
	},
	Action: runBwBackup,
}
```

### 4. Credential Resolution Logic (main.go)

```go
func getCredential(flagValue, keychainAccount string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	
	// Fall back to Keychain
	value, err := keychain.GetPassword(keychainAccount)
	if err != nil {
		return "", fmt.Errorf("credential not provided and Keychain lookup failed: %w", err)
	}
	return value, nil
}
```

### 5. Command Action Implementation (main.go)

```go
func runBwBackup(c *cli.Context) error {
	// Get credentials (flags or Keychain)
	clientID, err := getCredential(c.String("client-id"), "bitwarden_client_id")
	if err != nil {
		return err
	}
	clientSecret, err := getCredential(c.String("client-secret"), "bitwarden_client_secret")
	if err != nil {
		return err
	}
	password, err := getCredential(c.String("password"), "bitwarden_password")
	if err != nil {
		return err
	}
	
	// Build environment and tmpfs
	env := map[string]string{
		"BW_CLIENTID": clientID,
		"BW_CLIENTSECRET": clientSecret,
		"BW_PASSWORD": password,
	}
	tmpfs := []string{"/tmp", "/var/tmp"}
	
	backupDir := c.String("backup-dir")
	return RunContainer("ghcr.io/vupham90/containers-bw-backup:latest", backupDir, []string{}, env, tmpfs)
}
```

### 6. GitHub Actions Workflow Update (.github/workflows/build-images.yml)

```yaml
# Add to the build matrix or as a separate job
- name: Build bw-backup image
  run: |
    docker build -t ghcr.io/${{ github.repository_owner }}/containers-bw-backup:latest \
      -f dockerfiles/bw-backup/Dockerfile dockerfiles/bw-backup
    docker push ghcr.io/${{ github.repository_owner }}/containers-bw-backup:latest
```

### 7. Updating pdf-compress for Backward Compatibility (main.go)

```go
// Existing pdf-compress command just passes nil/empty for new params
err := RunContainer(
	"ghcr.io/vupham90/containers-pdf-compress:latest",
	fileDir,
	gsArgs,
	nil,   // No environment variables
	nil,   // No tmpfs mounts
)
```

## Security Considerations

1. **Credential Handling**
   - Never log credentials in debug output
   - Clear credential variables after use
   - Support credential retrieval from secure stores

2. **tmpfs Mounts**
   - Essential for no-trace execution
   - Prevents sensitive data persistence
   - Must be included in Docker invocation

3. **Container Cleanup**
   - Use `--rm` flag (already in RunContainer)
   - Ensures no credential residue in stopped containers

4. **Exit Codes**
   - Preserve container exit codes for monitoring
   - `0` = Success, `1` = Auth failure, `2` = Export failure

## Testing Plan

1. **macOS with Keychain**
   - Credentials in Keychain → successful backup
   - Missing Keychain entry → clear error message

2. **CLI Flags**
   - Explicit flags → successful backup (overrides Keychain)
   - Invalid credentials → proper error handling

3. **Error Scenarios**
   - Missing backup directory → error before Docker run
   - Invalid backup directory → error before Docker run
   - Docker image not found → clear pull/build instructions

## Success Criteria

- [ ] Command executes successfully with Keychain credentials
- [ ] Command executes successfully with CLI flags
- [ ] Backup files created with correct timestamp format
- [ ] tmpfs mounts applied for security
- [ ] Clear error messages for all failure modes
- [ ] Documentation includes usage examples
- [ ] Image follows containers project naming convention (`ghcr.io/vupham90/containers-bw-backup:latest`)

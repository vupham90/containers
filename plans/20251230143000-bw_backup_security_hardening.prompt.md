## Plan: Security Hardening for Unattended bw-backup

This plan implements essential security improvements for unattended Bitwarden backups running on a home server. Focus is on preventing credential leakage in error scenarios and ensuring zero disk traces.

### Steps

1. **Implement file descriptor passing for credentials**
   Add `createSecurePipe()` function in [docker.go](docker.go) that creates OS pipes for credential injection. Update `RunContainer` to accept `removeContainer bool` parameter and detect sensitive `EnvVar` entries. For each sensitive credential: create pipe with `os.Pipe()`, write credential in goroutine, attach read end to `cmd.ExtraFiles`, and set env var with FD index (e.g., `BW_CLIENTID_FD=3`). Add bash utility function `read_credential_fd()` in [dockerfiles/bw-backup/backup.sh](dockerfiles/bw-backup/backup.sh) that reads from FD and closes it immediately. Credentials exist only in kernel pipe buffers and process memory—never touch filesystem.

```go
// createSecurePipe creates a pipe for passing credentials securely
func createSecurePipe(credential string) (*os.File, error) {
    r, w, err := os.Pipe()
    if err != nil {
        return nil, fmt.Errorf("failed to create pipe: %w", err)
    }
    
    go func() {
        defer w.Close()
        w.WriteString(credential)
    }()
    
    return r, nil
}

// Update RunContainer signature
func RunContainer(image string, args []string, workDir string, env map[string]EnvVar, removeContainer bool) error {
    dockerArgs := []string{"run"}
    
    if removeContainer {
        dockerArgs = append(dockerArgs, "--rm")
    }
    
    var extraFiles []*os.File
    var fdIndex = 3
    
    for key, envVar := range env {
        if envVar.Sensitive {
            r, err := createSecurePipe(envVar.Value)
            if err != nil {
                return fmt.Errorf("failed to create secure pipe for %s: %w", key, err)
            }
            defer r.Close()
            
            dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s_FD=%d", key, fdIndex))
            extraFiles = append(extraFiles, r)
            fdIndex++
        } else {
            dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", key, envVar.Value))
        }
    }
    
    // ... rest of docker args setup ...
    
    cmd := exec.Command("docker", dockerArgs...)
    cmd.ExtraFiles = extraFiles
    
    return cmd.Run()
}
```

```bash
# Utility function to read credential from file descriptor
read_credential_fd() {
    local fd_num="$1"
    local value
    
    if [[ -z "${fd_num}" ]]; then
        log "ERROR: FD number not provided"
        return 1
    fi
    
    # Read from FD and close immediately
    value=$(cat <&${fd_num})
    exec {fd_num}<&-
    
    echo "${value}"
}

# Usage example - read all credentials from file descriptors
BW_CLIENTID=$(read_credential_fd "${BW_CLIENTID_FD}")
BW_CLIENTSECRET=$(read_credential_fd "${BW_CLIENTSECRET_FD}")
BW_PASSWORD=$(read_credential_fd "${BW_PASSWORD_FD}")

# FDs are now closed, credentials only exist in shell variables
# No files on disk, not even in tmpfs
```

2. **Secure BW_SESSION token handling**
   Modify [dockerfiles/bw-backup/backup.sh](dockerfiles/bw-backup/backup.sh) to never export `BW_SESSION`. Instead, pass token directly to commands via `--session` flag. Unset token after use to minimize exposure window.

```bash
# Unlock vault - DO NOT export BW_SESSION
if ! BW_SESSION=$(bw unlock --passwordenv BW_PASSWORD --raw); then
    log "ERROR: Failed to unlock Bitwarden vault"
    exit 1
fi

# Pass session token directly to commands (never export)
bw sync --session "${BW_SESSION}"
bw export --format json --output "${BACKUP_PATH}" --session "${BW_SESSION}"

# Unset token immediately after use
unset BW_SESSION
```

3. **Harden backup file permissions**
   In [dockerfiles/bw-backup/backup.sh](dockerfiles/bw-backup/backup.sh), create backup file with restrictive permissions before writing: `touch "${BACKUP_PATH}" && chmod 0400 "${BACKUP_PATH}"`. This prevents accidental exposure of backup files to other users on the system.

```bash
# Generate timestamp for backup filename
TIMESTAMP=$(date -u +'%Y-%m-%d-%H%M%S')
BACKUP_FILENAME="bitwarden-${BW_PROFILE}-${TIMESTAMP}.json"
BACKUP_PATH="/workspace/${BACKUP_FILENAME}"

# Create file with restrictive permissions before writing
touch "${BACKUP_PATH}"
chmod 0400 "${BACKUP_PATH}"

# Export vault data
bw export --format json --output "${BACKUP_PATH}" --session "${BW_SESSION}"

log "Backup created: ${BACKUP_FILENAME}"
```

4. **Add comprehensive tmpfs mounts**
   Update [docker.go](docker.go) to add tmpfs mounts for all writable directories in the container. This ensures zero disk I/O for temporary files, Bitwarden CLI config, and bash history. Since `--rm` flag is used, all tmpfs content is automatically wiped when container exits.

```go
// In bw_backup.go - configure comprehensive tmpfs mounts
tmpfsMounts := map[string]string{
    "/tmp":          "rw,noexec,nosuid,size=100m",
    "/root/.config": "rw,noexec,nosuid,size=50m",
    "/root/.cache":  "rw,noexec,nosuid,size=50m",
    "/root/.local":  "rw,noexec,nosuid,size=50m",
}

// Build docker args with tmpfs mounts
for path, opts := range tmpfsMounts {
    dockerArgs = append(dockerArgs, "--tmpfs", fmt.Sprintf("%s:%s", path, opts))
}

// Call with removeContainer=true for automatic cleanup
return docker.RunContainer("bw-backup:latest", nil, absBackupDir, env, true)
```

```bash
# In backup.sh - remove manual cleanup commands (redundant with tmpfs + --rm)
# DELETE these lines if they exist:
# rm -f ~/.bash_history
# rm -rf /tmp/*
# rm -rf /var/tmp/*
# rm -rf ~/.cache/*

# Container and all tmpfs mounts automatically wiped on exit
log "Backup complete"
```

5. **Add basic audit logging**
   Update [bw_backup.go](bw_backup.go) to log when automated backups start and complete. This provides visibility for unattended runs and helps with debugging cron jobs.

```go
// At start of backup function
fmt.Fprintf(os.Stderr, "[AUDIT] Bitwarden backup started: profile=%s time=%s\n",
    profile, time.Now().Format(time.RFC3339))

// After successful backup
fmt.Fprintf(os.Stderr, "[AUDIT] Bitwarden backup completed: profile=%s duration=%s\n",
    profile, time.Since(startTime))
```

### Further Considerations

1. **Backup encryption at rest**: Backups are currently stored as unencrypted JSON. For additional security, consider encrypting before writing to disk. Options: A) Use GPG with a dedicated backup key, B) Use age encryption with SSH key, C) Rely on filesystem-level encryption (APFS encrypted volume, LUKS). Recommendation for home server: Use filesystem encryption for simplicity.

2. **SHA-256 integrity checksums**: Consider adding `sha256sum "${BACKUP_PATH}" > "${BACKUP_PATH}.sha256"` after creating backups. Pros: Detect corruption or tampering. Cons: Adds complexity. Recommendation: Optional, useful if backups are synced/copied to other systems.

3. **Credential reset confirmation**: Add interactive confirmation when using `--reset` flag to prevent accidental credential deletion. Low priority for unattended use but helpful during manual operations.

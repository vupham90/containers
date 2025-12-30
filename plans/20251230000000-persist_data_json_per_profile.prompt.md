## Plan: Persist Bitwarden CLI data.json Per Profile

Enable persistent Bitwarden CLI sessions per profile by mounting profile-specific data directories, eliminating repeated login cycles and reducing API calls while maintaining multi-account isolation.

### Steps

1. **Extend RunContainer signature in [docker.go](docker.go#L20)**
Add `volumeMounts []string` parameter to support additional volume mounts:
```go
func RunContainer(image, workDir string, args []string, env map[string]EnvVar, tmpfs []string, volumeMounts []string, removeContainer bool) error {
    // ... existing code ...
    
    // Add custom volume mounts
    for _, mount := range volumeMounts {
        dockerArgs = append(dockerArgs, "-v", mount)
    }
    
    // Add working directory mount
    dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/workspace", absWorkDir))
    // ... rest of function
}
```

2. **Update all RunContainer callers to pass empty volumeMounts**
- [main.go#L86](main.go#L86): `RunContainer(image, workDir, args, nil, nil, nil, true)`
- [bw_backup.go#L153](bw_backup.go#L153): Pass `nil` for volumeMounts initially
- [bw_backup.go#L324](bw_backup.go#L324): Pass `nil` for volumeMounts initially

3. **Add profile-specific config mount to Bitwarden backup calls in [bw_backup.go](bw_backup.go)**
In `runSingleBackup()` (line ~153) and `backupVault()` (line ~324), create config directory and pass as volume mount:
```go
// Determine profile name (empty for single-account)
profile := os.Getenv("BW_PROFILE")
configDir := filepath.Join(os.Getenv("HOME"), ".config", "Bitwarden CLI")
if profile != "" {
    configDir = filepath.Join(os.Getenv("HOME"), ".config", fmt.Sprintf("Bitwarden CLI-%s", profile))
}

// Create config directory
if err := os.MkdirAll(configDir, 0700); err != nil {
    return fmt.Errorf("failed to create config dir: %w", err)
}

// Mount config directory
volumeMounts := []string{fmt.Sprintf("%s:/root/.config/Bitwarden CLI", configDir)}
err = RunContainer(image, absBackupDir, []string{}, env, tmpfs, volumeMounts, true)
```

4. **Simplify session handling in [backup.sh](dockerfiles/bw-backup/backup.sh)**
Replace lines 77-84 with status-based authentication:
```bash
STATUS=$(bw status | jq -r '.status')

if [ "$STATUS" = "unauthenticated" ]; then
    log "Logging in to Bitwarden..."
    bw login --apikey
fi

log "Unlocking vault..."
BW_SESSION=$(bw unlock --passwordenv BW_PASSWORD --raw)
```

5. **Keep session alive in [backup.sh](dockerfiles/bw-backup/backup.sh)**
Replace lines 134-139 to lock instead of logout:
```bash
log "Locking vault..."
bw lock
unset BW_SESSION
log "Backup complete"
```

### Further Considerations

1. **Extend RunContainer function** - Add `volumeMounts []string` parameter to allow additional volume mounts beyond the working directory.

2. **Profile naming for multi-account** - When `BW_PROFILE` env var is not set (single-account mode), use empty string to mount default `.config/Bitwarden CLI` folder.

3. **Security trade-off** - Persisting `data.json` is actually MORE secure than the current approach. Weekly "new device" emails create alert fatigue, causing users to ignore real unauthorized access. Persistent sessions eliminate this noise while maintaining proper file permissions (0700). Document this decision and recommend users periodically check Bitwarden Web Vault → Settings → Active Sessions for unauthorized access.

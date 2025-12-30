## Plan: Add Password-Protected Backup Support

Add Bitwarden's native encrypted export functionality to the backup utility with proper keychain integration and edge case handling for the `--backup-password` flag.

### Steps

1. **Add `getBackupPassword()` helper function** in [bw_backup.go](../bw_backup.go)
   - Use `c.IsSet("backup-password")` to detect if flag was provided
   - If not set: return empty string (no encryption - backward compatible)
   - If set but empty: retrieve from keychain via `keychain.GetOrSetPassword()`
   - If set with value: return the value directly
   - **Always use global keychain account**: `bitwarden_backup_password` (no profile suffix)
```go
func getBackupPassword(c *cli.Context, reset bool) (string, error) {
    if !c.IsSet("backup-password") {
        return "", nil  // Flag not provided → no encryption
    }
    flagValue := c.String("backup-password")
    if flagValue != "" {
        return flagValue, nil  // Use provided value
    }
    // Empty flag → get from keychain (always global, no profile suffix)
    return keychain.GetOrSetPassword("containers-bw-backup", "bitwarden_backup_password", reset)
}
```

2. **Add `--backup-password` flag to bw-backup command** in [main.go](../main.go)
   - Add flag with alias `-b` or `-bp`
   - Usage text explains three behaviors: with value, empty for keychain, omit for no encryption
   - Place with other credential flags
```go
&cli.StringFlag{
    Name:    "backup-password",
    Aliases: []string{"bp"},
    Usage:   "Password for encrypted backup (use empty string for keychain, omit for no encryption)",
},
```

3. **Update `runSingleBackup()` function** in [bw_backup.go](../bw_backup.go)
   - Call `getBackupPassword()` to retrieve password
   - Add `BW_BACKUP_PASSWORD` to env vars only if password is not empty
   - Mark as sensitive to prevent logging
```go
backupPassword, err := getBackupPassword(c, profile, (no profile parameter)
   - Add `BW_BACKUP_PASSWORD` to env vars only if password is not empty
   - Mark as sensitive to prevent logging
```go
backupPassword, err := getBackupPassword(c
    env["BW_BACKUP_PASSWORD"] = EnvVar{Value: backupPassword, Sensitive: true}
}
```

4. **Update `runBatchBackup()` to get backup password once** in [bw_backup.go](../bw_backup.go)
   - Call `getBackupPassword()` once before processing profiles
   - Pass password to all `backupVault()` calls
   - Same password applies to all profiles in batch
```go
// In runBatchBackup, after parsing config:
backupPassword, err := getBackupPassword(c, "", reset)
   - Same global password applies to all profiles in batch
```go
// In runBatchBackup, after parsing config:
backupPassword, err := getBackupPassword(c, reset)
```

5. **Update `backupVault()` signature** in [bw_backup.go](../bw_backup.go)
   - Add `backupPassword` parameter
   - Add to env vars if not empty
```go
func backupVault(c *cli.Context, profile BackupProfile, orgID string, reset bool, backupPassword string) error {
    // ... existing code ...
    if backupPassword != "" {
        env["BW_BACKUP_PASSWORD"] = EnvVar{Value: backupPassword, Sensitive: true}
    }
}
```

5. **Update backup script to use encrypted export** in [backup.sh](../dockerfiles/bw-backup/backup.sh)
   - Detect `BW_BACKUP_PASSWORD` environment variable
   - Use `.encrypted.json` extension and `encrypted_json` format when password present
   - Add `--password "${BW_BACKUP_PASSWORD}"` flag to `bw export` command
   - Update cleanup to unset `BW_BACKUP_PASSWORD`
```bash
if [ -n "${BW_BACKUP_PASSWORD:-}" ]; then
    FILE_EXT="encrypted.json"
    bw export --format encrypted_json --password "${BW_BACKUP_PASSWORD}" \
              --output "${BACKUP_PATH}" --session "${BW_SESSION}"
else
    FILE_EXT="json"
    bw export --format json --output "${BACKUP_PATH}" --session "${BW_SESSION}"
fi
```

7. **Update README with usage examples** in [README.md](../dockerfiles/bw-backup/README.md)
   - Document three usage patterns: no flag, empty flag, flag with value
   - Explain keychain storage pattern
   - Show how to import encrypted backups using `bw import encrypted_json`

### Further Considerations

1. **Global backup password**: Backup password is ALWAYS global (stored as `bitwarden_backup_password` without profile suffix), even when using profiles. This differs from other credentials (client-id, password) which are per-profile. Rationale: backup password protects the backup file itself, not the Bitwarden account.

### Credential Storage Comparison

**Per-Profile (account-specific):**
- `bitwarden_client_id_{profile}`
- `bitwarden_client_secret_{profile}`
- `bitwarden_password_{profile}`

**Global (backup-specific):**
- `bitwarden_backup_password` ← No profile suffix, shared across all backups

This simplifies the UX - one backup password for all backups, regardless of which Bitwarden account/profile is being backed up.

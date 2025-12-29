## Plan: Multi-Account & Organization Bitwarden Backup Implementation

Implement batch backup support for multiple Bitwarden accounts and organizations using YAML configuration with sequential execution and keychain-based credential retrieval.

### Steps

1. **Refactor bw-backup code into separate file**
   Create new file `bw_backup.go` and move all bw-backup related code from [main.go](main.go) to it:
   - Move `getCredential` function
   - Move `runBwBackup` function
   - Keep bw-backup command definition in [main.go](main.go) but have it call `runBwBackup` from `bw_backup.go`

2. **Add YAML parsing dependency and define config structures in bw_backup.go**
   Add `gopkg.in/yaml.v3` to [go.mod](go.mod). In `bw_backup.go`, define `BackupProfile` struct with fields: `name` (string), `backup_dir` (string), `organizations` ([]string). Define `BackupConfig` struct with `profiles` ([]BackupProfile) field.

3. **Add new CLI flags to bw-backup command in [main.go](main.go#L152-L182)**
   Add `--profile` flag (string, optional, empty = use default keychain without suffix). Add `--organization-id` flag (string, optional, for single org backup). Add `--profiles` flag (string, optional, path to YAML config for batch mode). Update existing `--backup-dir` flag description to note it's required for single mode.

4. **Refactor credential retrieval to support profile-based keychain naming in bw_backup.go**
   Modify `getCredential` signature to accept `profile` parameter. When profile is non-empty, append `_${profile}` to keychain account name (e.g., `bitwarden_client_id_personal`). When profile is empty, use current account names without suffix for backward compatibility.
   ```go
   // Example: bitwarden_client_id_work vs bitwarden_client_id
   account := keychainAccount
   if profile != "" {
       account = fmt.Sprintf("%s_%s", keychainAccount, profile)
   }
   ```

5. **Implement batch backup orchestration in bw_backup.go**
   Create `runBatchBackup(c *cli.Context, configPath string)` that reads YAML, iterates profiles sequentially, calls `backupVault` for each. Create `backupVault(c *cli.Context, profile BackupProfile, orgID string)` helper that:
   - Retrieves credentials from keychain using profile name suffix
   - Expands `~/` in backup_dir paths
   - Creates backup directory if missing
   - Builds env map with `BW_CLIENTID`, `BW_CLIENTSECRET`, `BW_PASSWORD`, `BW_PROFILE`, and `BW_ORGANIZATIONID` (if orgID provided)
   - Calls [RunContainer](docker.go) with appropriate parameters
   
   Modify `runBwBackup` to check if `--profiles` flag is set: if yes, route to `runBatchBackup`, otherwise execute single backup mode.

5. **Update backup.sh script to support profiles and organizations**
   In [dockerfiles/bw-backup/backup.sh](dockerfiles/bw-backup/backup.sh), read `BW_PROFILE` and `BW_ORGANIZATIONID` environment variables. Update filename generation logic:
   ```bash
   # If BW_PROFILE is empty: bitwarden-backup-${TIMESTAMP}.js in bw_backup.goon
   # If BW_PROFILE set, no org: bitwarden-${BW_PROFILE}-backup-${TIMESTAMP}.json
   # If BW_PROFILE set, with org: bitwarden-${BW_PROFILE}-org-${BW_ORGANIZATIONID}-backup-${TIMESTAMP}.json
   ```
   When `BW_ORGANIZATIONID` is set, use `bw export --organizationid "${BW_ORGANIZATIONID}" --format json --output "${BACKUP_PATH}"` instead of regular export.

6. **Add error handling and progress reporting for batch mode**
   In `runBatchBackup`, collect errors in slice instead of failing immediately. Print progress for each profile: `fmt.Printf("[%d/%d] Processing profile: %s\n", currentIndex, totalProfiles, profile.Name)`. Print sub-progress for orgs: `fmt.Printf("  → Backing up organization: %s\n", orgID)`. After all profiles, print summary: `fmt.Printf("\nBatch backup completed: %d successful, %d failed\n", successCount, len(errors))`. If errors exist, list them and return aggregated error.

7. **Test implementation with multiple scenarios**
   Test backward compatibility: run `containers bw-backup --backup-dir ~/test` without profile (should use default keychain). Test single profile mode: `containers bw-backup --profile personal --backup-dir ~/test`. Test profile with org: `containers bw-backup --profile work --organization-id org-123 --backup-dir ~/test`. Create test YAML with 2 profiles (one with orgs, one without) and run batch mode. Verify keychain isolation between profiles. Test error handling with missing credentials and invalid org IDs.

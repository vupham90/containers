## Plan: Multi-Account & Organization Bitwarden Backup

Extend the `bw-backup` command to support backing up multiple Bitwarden accounts and organizations in batch mode using a YAML configuration file. Users can define multiple profiles with optional organization IDs, and the tool will automatically retrieve credentials from the macOS Keychain and execute backups sequentially.

### Product Requirements

#### Current Behavior
- Single Bitwarden account backup via CLI flags
- Credentials stored in macOS Keychain: `bitwarden_client_id`, `bitwarden_client_secret`, `bitwarden_password`
- One backup per command execution
- Manual execution for each account/organization

#### New Behavior

**Single Account Mode (Backward Compatible)**
```bash
# Default profile backup (existing - no profile name)
containers bw-backup --backup-dir ~/backups
# Uses: bitwarden_client_id, bitwarden_client_secret, bitwarden_password

# Default profile backup with organization (existing - no profile name)
containers bw-backup --backup-dir ~/backups --organization-id org-12
# Uses: bitwarden_client_id, bitwarden_client_secret, bitwarden_password

# Named profile backup (new)
containers bw-backup --profile personal --backup-dir ~/backups/personal
# Uses: bitwarden_client_id_personal, bitwarden_client_secret_personal, bitwarden_password_personal

# Organization backup with profile (new)
containers bw-backup --profile work --organization-id org-123 --backup-dir ~/backups/work-org
# Uses: bitwarden_client_id_work, bitwarden_client_secret_work, bitwarden_password_work
```

**Batch Mode (New)**
```bash
# Backup all profiles from YAML config
containers bw-backup --profiles ~/profiles.yaml
```

**YAML Configuration Format**
```yaml
profiles:
  - name: personal
    backup_dir: ~/backups/personal
    
  - name: work
    backup_dir: ~/backups/work
    organizations:
      - org-id-123
      - org-id-456
      
  - name: family
    backup_dir: ~/backups/family
    organizations:
      - family-org-id
```

**Keychain Credential Mapping**
- Default (no profile flag): `bitwarden_client_id`, `bitwarden_client_secret`, `bitwarden_password` (backward compatible)
- Named profiles: `bitwarden_client_id_personal`, `bitwarden_client_secret_personal`, `bitwarden_password_personal`
- Pattern: `bitwarden_{credential}_{profile_name}` (profile name required in YAML mode)

**Setup Workflow**
1. User runs single backup once per profile to store credentials in keychain:
   ```bash
   # Setup personal profile credentials
   containers bw-backup --profile personal --backup-dir ~/temp
   # Prompts for client-id, client-secret, password
   # Stores as: bitwarden_client_id_personal, bitwarden_client_secret_personal, bitwarden_password_personal
   
   # Setup work profile credentials
   containers bw-backup --profile work --backup-dir ~/temp
   # Stores as: bitwarden_client_id_work, bitwarden_client_secret_work, bitwarden_password_work
   ```
2. User creates YAML config with profile names and backup directories (profile name is mandatory in YAML)
3. User runs batch backup: `containers bw-backup --profiles profiles.yaml`
4. Tool iterates through profiles, pulls credentials from keychain using profile name suffix, executes backups

**Batch Execution Behavior**
- Process profiles sequentially (not parallel to avoid API rate limits)
- For each profile:
  1. Backup personal vault to `backup_dir`
  2. For each organization ID, backup to same `backup_dir`
- Continue on failure (don't stop entire batch if one fails)
- Print progress: `[2/3] Processing profile: work`
- Print summary at end: `Batch backup completed: 5 successful, 1 failed`
- List all errors at the end for review

**Backup File Naming**
- Default profile personal vault: `bitwarden-backup-YYYY-MM-DD-HHMMSS.json` (no profile name in filename)
- Named profile personal vault: `bitwarden-{profile}-backup-YYYY-MM-DD-HHMMSS.json`
- Named profile organization vault: `bitwarden-{profile}-org-{org-id}-backup-YYYY-MM-DD-HHMMSS.json`

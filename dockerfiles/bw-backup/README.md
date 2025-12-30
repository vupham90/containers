# Bitwarden Backup

Containerized Bitwarden vault backup utility.

## Features

- Automated vault export to timestamped JSON files
- No-trace execution with tmpfs mounts
- macOS Keychain integration for credentials
- Non-root container execution
- Security-first design

## Usage

Via containers CLI:

```bash
# Using Keychain credentials (unencrypted backup)
containers bw-backup --backup-dir ~/backups

# With password-protected backup (from keychain)
containers bw-backup --backup-dir ~/backups --encrypt

# With explicit backup password
containers bw-backup \
  --backup-dir ~/backups \
  --backup-password "my-secure-backup-password"

# Batch mode with encryption
containers bw-backup --profiles config.yaml --encrypt

# With explicit Bitwarden credentials
containers bw-backup \
  --client-id "your-client-id" \
  --client-secret "your-client-secret" \
  --password "your-master-password" \
  --backup-dir ~/backups
```

Direct Docker usage:

```bash
# Unencrypted backup
docker run --rm \
  --tmpfs /tmp \
  --tmpfs /var/tmp \
  -e BW_CLIENTID="your-client-id" \
  -e BW_CLIENTSECRET="your-client-secret" \
  -e BW_PASSWORD="your-master-password" \
  -v ~/backups:/workspace \
  ghcr.io/vupham90/containers-bw-backup:latest

# Encrypted backup
docker run --rm \
  --tmpfs /tmp \
  --tmpfs /var/tmp \
  -e BW_CLIENTID="your-client-id" \
  -e BW_CLIENTSECRET="your-client-secret" \
  -e BW_PASSWORD="your-master-password" \
  -e BW_BACKUP_PASSWORD="backup-encryption-password" \
  -v ~/backups:/workspace \
  ghcr.io/vupham90/containers-bw-backup:latest
```

## Credentials

The container requires three environment variables:

- `BW_CLIENTID` - Bitwarden API client ID
- `BW_CLIENTSECRET` - Bitwarden API client secret
- `BW_PASSWORD` - Bitwarden master password

Optional environment variable for encrypted backups:

- `BW_BACKUP_PASSWORD` - Password to encrypt the backup file (uses Bitwarden's encrypted_json format)

When using the containers CLI on macOS, credentials are automatically retrieved from Keychain entries:
- `bitwarden_client_id`
- `bitwarden_client_secret`
- `bitwarden_password`
- `bitwarden_backup_password` (optional, for encrypted backups)

## Backup Password Behavior

The backup encryption has three modes:

1. **No encryption (default)**: Creates unencrypted JSON backup
   ```bash
   containers bw-backup --backup-dir ~/backups
   ```

2. **Encrypt with keychain password**: Use `--encrypt` flag
   ```bash
   containers bw-backup --backup-dir ~/backups --encrypt
   ```

3. **Encrypt with explicit password**: Use `--backup-password` flag
   ```bash
   containers bw-backup --backup-dir ~/backups --backup-password "mypassword"
   ```

Note: `--backup-password` overrides `--encrypt` if both are provided.

## Output

Backups are saved as timestamped files:

**Unencrypted backups:**
```
bitwarden-backup-2025-12-29-143022.json
```

**Encrypted backups:**
```
bitwarden-backup-2025-12-29-143022.encrypted.json
```

## Security

- Runs as non-root user (uid 1000)
- Uses tmpfs mounts for temporary files (no disk traces)
- Clears bash history and cache after execution
- Designed for use with encrypted backup storage

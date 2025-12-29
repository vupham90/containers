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
# Using Keychain credentials
containers bw-backup --backup-dir ~/backups

# With explicit credentials
containers bw-backup \
  --client-id "your-client-id" \
  --client-secret "your-client-secret" \
  --password "your-master-password" \
  --backup-dir ~/backups
```

Direct Docker usage:

```bash
docker run --rm \
  --tmpfs /tmp \
  --tmpfs /var/tmp \
  -e BW_CLIENTID="your-client-id" \
  -e BW_CLIENTSECRET="your-client-secret" \
  -e BW_PASSWORD="your-master-password" \
  -v ~/backups:/workspace \
  ghcr.io/vupham90/containers-bw-backup:latest
```

## Credentials

The container requires three environment variables:

- `BW_CLIENTID` - Bitwarden API client ID
- `BW_CLIENTSECRET` - Bitwarden API client secret
- `BW_PASSWORD` - Bitwarden master password

When using the containers CLI on macOS, credentials are automatically retrieved from Keychain entries:
- `bitwarden_client_id`
- `bitwarden_client_secret`
- `bitwarden_password`

## Output

Backups are saved as timestamped JSON files:
```
bitwarden-backup-2025-12-29-143022.json
```

## Security

- Runs as non-root user (uid 1000)
- Uses tmpfs mounts for temporary files (no disk traces)
- Clears bash history and cache after execution
- Designed for use with encrypted backup storage

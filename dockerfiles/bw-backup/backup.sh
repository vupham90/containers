#!/bin/bash
set -euo pipefail

# Logging function
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S UTC')] $1"
}

log "Starting Bitwarden backup process..."

# Step 1: Validate credentials from environment
if [ -z "${BW_CLIENTID:-}" ] || [ -z "${BW_CLIENTSECRET:-}" ] || [ -z "${BW_PASSWORD:-}" ]; then
    log "ERROR: Missing credentials. Provide BW_CLIENTID, BW_CLIENTSECRET, BW_PASSWORD via environment."
    exit 1
fi

# Step 2: Set backup directory (mounted to /workspace by containers CLI)
TARGET_BACKUP_DIR="/workspace"

# Validate backup directory exists
if [ ! -d "${TARGET_BACKUP_DIR}" ]; then
    log "ERROR: Backup directory ${TARGET_BACKUP_DIR} does not exist."
    exit 1
fi

TIMESTAMP=$(date -u +'%Y-%m-%d-%H%M%S')

# Generate backup filename based on profile and organization
if [ -n "${BW_ORGANIZATIONID:-}" ]; then
    # Organization backup with profile
    if [ -n "${BW_PROFILE:-}" ]; then
        BACKUP_FILENAME="bitwarden-${BW_PROFILE}-org-${BW_ORGANIZATIONID}-backup-${TIMESTAMP}.json"
    else
        BACKUP_FILENAME="bitwarden-org-${BW_ORGANIZATIONID}-backup-${TIMESTAMP}.json"
    fi
else
    # Personal vault backup
    if [ -n "${BW_PROFILE:-}" ]; then
        BACKUP_FILENAME="bitwarden-${BW_PROFILE}-backup-${TIMESTAMP}.json"
    else
        BACKUP_FILENAME="bitwarden-backup-${TIMESTAMP}.json"
    fi
fi

BACKUP_PATH="${TARGET_BACKUP_DIR}/${BACKUP_FILENAME}"

log "Backup will be saved to: ${BACKUP_PATH}"

# Step 3: Login to Bitwarden with API key
log "Authenticating to Bitwarden..."
if ! bw login --apikey 2>&1; then
    log "ERROR: Failed to login to Bitwarden"
    exit 1
fi

# Step 4: Unlock vault
log "Unlocking Bitwarden vault..."
if ! BW_SESSION=$(bw unlock --passwordenv BW_PASSWORD --raw 2>&1); then
    log "ERROR: Failed to unlock Bitwarden vault"
    exit 1
fi

export BW_SESSION

# Step 5: Export vault (unencrypted - will be stored on encrypted drive)
if [ -n "${BW_ORGANIZATIONID:-}" ]; then
    log "Exporting organization vault (ID: ${BW_ORGANIZATIONID}) to ${BACKUP_FILENAME}..."
    log "Using unencrypted JSON export (will be stored on encrypted drive)"
    if ! bw export --organizationid "${BW_ORGANIZATIONID}" --format json --output "${BACKUP_PATH}" 2>&1; then
        log "ERROR: Failed to export organization vault"
        exit 2
    fi
else
    log "Exporting personal vault to ${BACKUP_FILENAME}..."
    log "Using unencrypted JSON export (will be stored on encrypted drive)"
    if ! bw export --format json --output "${BACKUP_PATH}" 2>&1; then
        log "ERROR: Failed to export personal vault"
        exit 2
    fi
fi

# Verify export file exists and is not empty
if [ ! -s "${BACKUP_PATH}" ]; then
    log "ERROR: Export file is empty or does not exist"
    exit 2
fi

FILE_SIZE=$(stat -c%s "${BACKUP_PATH}" 2>/dev/null || stat -f%z "${BACKUP_PATH}" 2>/dev/null)
log "Export completed successfully (${FILE_SIZE} bytes)"

# Step 6: Lock vault and logout
log "Locking Bitwarden vault..."
bw lock || true
bw logout || true

# Step 7: Cleanup traces for security
log "Cleaning up traces..."
rm -f ~/.bash_history 2>/dev/null || true
rm -rf /tmp/* 2>/dev/null || true
rm -rf /var/tmp/* 2>/dev/null || true
rm -rf ~/.cache/* 2>/dev/null || true

log "Backup process completed successfully!"
exit 0

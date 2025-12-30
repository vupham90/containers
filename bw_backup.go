package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/vupham90/containers/keychain"
	"gopkg.in/yaml.v3"
)

// BackupProfile represents a single backup profile configuration
type BackupProfile struct {
	Name          string   `yaml:"name"`
	BackupDir     string   `yaml:"backup_dir"`
	Organizations []string `yaml:"organizations,omitempty"`
}

// BackupConfig represents the YAML configuration for batch backups
type BackupConfig struct {
	Profiles []BackupProfile `yaml:"profiles"`
}

// getCredential retrieves a credential from CLI flag or macOS Keychain
func getCredential(flagValue, keychainAccount, profile string, reset bool) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	// Build keychain account name with profile suffix if provided
	account := keychainAccount
	if profile != "" {
		account = fmt.Sprintf("%s_%s", keychainAccount, profile)
	}

	// Use keychain with reset flag
	serviceName := "containers-bw-backup"
	return keychain.GetOrSetPassword(serviceName, account, reset)
}

// getBackupPassword retrieves the backup password with Option 2 logic
func getBackupPassword(c *cli.Context, reset bool) (string, error) {
	// If explicit password provided, use it
	if c.IsSet("backup-password") {
		return c.String("backup-password"), nil
	}

	// If --encrypt flag set, get from keychain
	if c.Bool("encrypt") {
		serviceName := "containers-bw-backup"
		return keychain.GetOrSetPassword(serviceName, "bitwarden_backup_password", reset)
	}

	// No encryption
	return "", nil
}

// runBwBackup executes the Bitwarden backup command
func runBwBackup(c *cli.Context) error {
	// Check if batch mode (profiles YAML file provided)
	profilesPath := c.String("profiles")
	if profilesPath != "" {
		return runBatchBackup(c, profilesPath)
	}

	// Single backup mode
	return runSingleBackup(c)
}

// runSingleBackup handles single profile/organization backup
func runSingleBackup(c *cli.Context) error {
	reset := c.Bool("reset")
	profile := c.String("profile")
	orgID := c.String("organization-id")

	// Get credentials (flags or Keychain with reset option and profile support)
	clientID, err := getCredential(c.String("client-id"), "bitwarden_client_id", profile, reset)
	if err != nil {
		return err
	}
	clientSecret, err := getCredential(c.String("client-secret"), "bitwarden_client_secret", profile, reset)
	if err != nil {
		return err
	}
	password, err := getCredential(c.String("password"), "bitwarden_password", profile, reset)
	if err != nil {
		return err
	}

	// Get backup password (optional, global)
	backupPassword, err := getBackupPassword(c, reset)
	if err != nil {
		return err
	}

	// Resolve backup directory
	backupDir := c.String("backup-dir")
	absBackupDir, err := filepath.Abs(backupDir)
	if err != nil {
		return fmt.Errorf("failed to resolve backup directory: %w", err)
	}

	// Verify backup directory exists
	if _, err := os.Stat(absBackupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup directory does not exist: %s", absBackupDir)
	}

	// Build environment variables
	env := map[string]EnvVar{
		"BW_CLIENTID":     {Value: clientID, Sensitive: true},
		"BW_CLIENTSECRET": {Value: clientSecret, Sensitive: true},
		"BW_PASSWORD":     {Value: password, Sensitive: true},
	}

	// Add backup password if provided
	if backupPassword != "" {
		env["BW_BACKUP_PASSWORD"] = EnvVar{Value: backupPassword, Sensitive: true}
	}

	// Add profile name if provided
	if profile != "" {
		env["BW_PROFILE"] = EnvVar{Value: profile, Sensitive: false}
	}

	// Add organization ID if provided
	if orgID != "" {
		env["BW_ORGANIZATIONID"] = EnvVar{Value: orgID, Sensitive: false}
	}

	// Add comprehensive tmpfs mounts for security - prevents all disk writes
	tmpfsMounts := map[string]string{
		"/tmp":          "rw,noexec,nosuid,size=100m",
		"/root/.config": "rw,noexec,nosuid,size=50m",
		"/root/.cache":  "rw,noexec,nosuid,size=50m",
		"/root/.local":  "rw,noexec,nosuid,size=50m",
	}

	var tmpfs []string
	for path, opts := range tmpfsMounts {
		tmpfs = append(tmpfs, fmt.Sprintf("%s:%s", path, opts))
	}

	// Audit logging
	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "[AUDIT] Bitwarden backup started: profile=%s time=%s\n",
		profile, startTime.Format(time.RFC3339))

	// Execute backup container
	image := "ghcr.io/vupham90/containers-bw-backup:latest"
	fmt.Println("Starting Bitwarden backup...")
	err = RunContainer(image, absBackupDir, []string{}, env, tmpfs, true)

	// Log completion
	if err == nil {
		fmt.Fprintf(os.Stderr, "[AUDIT] Bitwarden backup completed: profile=%s duration=%s\n",
			profile, time.Since(startTime))
	} else {
		fmt.Fprintf(os.Stderr, "[AUDIT] Bitwarden backup failed: profile=%s duration=%s error=%v\n",
			profile, time.Since(startTime), err)
	}

	return err
}

// runBatchBackup handles batch backup from YAML config
func runBatchBackup(c *cli.Context, configPath string) error {
	// Expand home directory if needed
	if len(configPath) > 0 && configPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, configPath[1:])
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML config
	var config BackupConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if len(config.Profiles) == 0 {
		return fmt.Errorf("no profiles found in config file")
	}

	fmt.Printf("Starting batch backup for %d profile(s)...\n\n", len(config.Profiles))

	var errors []string
	successCount := 0
	reset := c.Bool("reset")

	// Get backup password once for all profiles (global)
	backupPassword, err := getBackupPassword(c, reset)
	if err != nil {
		return err
	}

	// Process each profile sequentially
	for i, profile := range config.Profiles {
		fmt.Printf("[%d/%d] Processing profile: %s\n", i+1, len(config.Profiles), profile.Name)

		// Backup personal vault
		if err := backupVault(c, profile, "", reset, backupPassword); err != nil {
			errors = append(errors, fmt.Sprintf("Profile '%s' personal vault: %v", profile.Name, err))
			fmt.Printf("  ✗ Personal vault backup failed: %v\n", err)
		} else {
			successCount++
			fmt.Printf("  ✓ Personal vault backup completed\n")
		}

		// Backup each organization
		for _, orgID := range profile.Organizations {
			fmt.Printf("  → Backing up organization: %s\n", orgID)
			if err := backupVault(c, profile, orgID, reset, backupPassword); err != nil {
				errors = append(errors, fmt.Sprintf("Profile '%s' org '%s': %v", profile.Name, orgID, err))
				fmt.Printf("    ✗ Organization backup failed: %v\n", err)
			} else {
				successCount++
				fmt.Printf("    ✓ Organization backup completed\n")
			}
		}

		fmt.Println()
	}

	// Print summary
	fmt.Printf("Batch backup completed: %d successful, %d failed\n", successCount, len(errors))
	if len(errors) > 0 {
		fmt.Println("\nErrors:")
		for _, errMsg := range errors {
			fmt.Printf("  - %s\n", errMsg)
		}
		return fmt.Errorf("batch backup completed with %d error(s)", len(errors))
	}

	return nil
}

// backupVault performs a single vault backup (personal or organization)
func backupVault(_ *cli.Context, profile BackupProfile, orgID string, reset bool, backupPassword string) error {
	// Get credentials from keychain using profile name suffix
	clientID, err := getCredential("", "bitwarden_client_id", profile.Name, reset)
	if err != nil {
		return fmt.Errorf("failed to get client ID: %w", err)
	}

	clientSecret, err := getCredential("", "bitwarden_client_secret", profile.Name, reset)
	if err != nil {
		return fmt.Errorf("failed to get client secret: %w", err)
	}

	password, err := getCredential("", "bitwarden_password", profile.Name, reset)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Expand backup directory (handle ~/)
	backupDir := profile.BackupDir
	if len(backupDir) > 0 && backupDir[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		backupDir = filepath.Join(home, backupDir[1:])
	}

	// Resolve to absolute path
	absBackupDir, err := filepath.Abs(backupDir)
	if err != nil {
		return fmt.Errorf("failed to resolve backup directory: %w", err)
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(absBackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Build environment variables
	env := map[string]EnvVar{
		"BW_CLIENTID":     {Value: clientID, Sensitive: true},
		"BW_CLIENTSECRET": {Value: clientSecret, Sensitive: true},
		"BW_PASSWORD":     {Value: password, Sensitive: true},
		"BW_PROFILE":      {Value: profile.Name, Sensitive: false},
	}

	// Add backup password if provided
	if backupPassword != "" {
		env["BW_BACKUP_PASSWORD"] = EnvVar{Value: backupPassword, Sensitive: true}
	}

	// Add organization ID if provided
	if orgID != "" {
		env["BW_ORGANIZATIONID"] = EnvVar{Value: orgID, Sensitive: false}
	}

	// Add comprehensive tmpfs mounts for security - prevents all disk writes
	tmpfsMounts := map[string]string{
		"/tmp":          "rw,noexec,nosuid,size=100m",
		"/root/.config": "rw,noexec,nosuid,size=50m",
		"/root/.cache":  "rw,noexec,nosuid,size=50m",
		"/root/.local":  "rw,noexec,nosuid,size=50m",
	}

	var tmpfs []string
	for path, opts := range tmpfsMounts {
		tmpfs = append(tmpfs, fmt.Sprintf("%s:%s", path, opts))
	}

	// Audit logging
	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "[AUDIT] Bitwarden backup started: profile=%s organization=%s time=%s\n",
		profile.Name, orgID, startTime.Format(time.RFC3339))

	// Execute backup container
	image := "ghcr.io/vupham90/containers-bw-backup:latest"
	err = RunContainer(image, absBackupDir, []string{}, env, tmpfs, true)

	// Log completion
	if err == nil {
		fmt.Fprintf(os.Stderr, "[AUDIT] Bitwarden backup completed: profile=%s organization=%s duration=%s\n",
			profile.Name, orgID, time.Since(startTime))
	} else {
		fmt.Fprintf(os.Stderr, "[AUDIT] Bitwarden backup failed: profile=%s organization=%s duration=%s error=%v\n",
			profile.Name, orgID, time.Since(startTime), err)
	}

	return err
}

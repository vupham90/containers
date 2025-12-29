package keychain

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// getPassword retrieves a password from macOS Keychain
func getPassword(account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-a", account, "-w")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve %s from Keychain: %w", account, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// setPassword stores or updates a password in macOS Keychain
func setPassword(account, password string) error {
	cmd := exec.Command("security", "add-generic-password", "-a", account, "-w", password, "-U")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set password for %s in Keychain: %w", account, err)
	}
	return nil
}

// passwordExists checks if a password exists in Keychain for the given account
func passwordExists(account string) bool {
	cmd := exec.Command("security", "find-generic-password", "-a", account)
	return cmd.Run() == nil
}

// deletePassword removes a password from macOS Keychain
func deletePassword(account string) error {
	cmd := exec.Command("security", "delete-generic-password", "-a", account)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete password for %s from Keychain: %w", account, err)
	}
	return nil
}

// GetOrSetPassword retrieves a password from Keychain, or prompts the user to set it if it doesn't exist.
// If reset is true, it will delete the existing password and prompt for a new one.
func GetOrSetPassword(account string, reset bool) (string, error) {
	// If reset flag is set, delete existing and re-enter
	if reset {
		if passwordExists(account) {
			_ = deletePassword(account)
		}
		return updatePassword(account)
	}

	// Try to retrieve from Keychain
	if passwordExists(account) {
		return getPassword(account)
	}

	// Password doesn't exist, prompt user to set it
	fmt.Printf("Password for '%s' not found in Keychain.\n", account)
	password, err := promptPassword(fmt.Sprintf("Enter password for '%s': ", account))
	if err != nil {
		return "", err
	}

	// Store in Keychain
	if err := setPassword(account, password); err != nil {
		return "", fmt.Errorf("failed to save password to Keychain: %w", err)
	}

	fmt.Printf("Password for '%s' saved to Keychain.\n", account)
	return password, nil
}

// updatePassword updates a password in Keychain, prompting the user for a new value
func updatePassword(account string) (string, error) {
	password, err := promptPassword(fmt.Sprintf("Enter new password for '%s': ", account))
	if err != nil {
		return "", err
	}

	if err := setPassword(account, password); err != nil {
		return "", fmt.Errorf("failed to update password in Keychain: %w", err)
	}

	fmt.Printf("Password for '%s' updated in Keychain.\n", account)
	return password, nil
}

// promptPassword reads a password from stdin securely (or as secure as possible in this context)
func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return strings.TrimSpace(password), nil
}

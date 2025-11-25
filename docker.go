package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RunContainer executes a Docker container with the specified image, working directory, and arguments.
// The working directory is mounted as /workspace in the container.
func RunContainer(image, workDir string, args []string) error {
	// Resolve absolute path for volume mount
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("failed to resolve work directory: %w", err)
	}

	// Verify directory exists
	if _, err := os.Stat(absWorkDir); os.IsNotExist(err) {
		return fmt.Errorf("work directory does not exist: %s", absWorkDir)
	}

	// Build docker run command
	// Example: docker run --rm -v /host/path:/workspace ghcr.io/user/containers-pdf-compress:latest gs -sDEVICE=pdfwrite ...
	dockerArgs := []string{
		"run",
		"--rm",                                   // Remove container after execution
		"-v",                                     // Volume mount flag
		fmt.Sprintf("%s:/workspace", absWorkDir), // Mount host directory to /workspace
		"-w", "/workspace",                       // Set working directory inside container
		image, // Docker image
	}

	// Append command arguments (e.g., gs command and its flags)
	dockerArgs = append(dockerArgs, args...)

	// Execute docker command
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run failed: %w", err)
	}

	return nil
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// Debug: Print the exact command being executed
	fmt.Printf("Executing: docker %s\n", strings.Join(dockerArgs, " "))

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

// RunDaemon runs a Docker container in detached mode with the specified configuration.
// It first removes any existing container with the same name to ensure idempotency.
func RunDaemon(name, image string, ports map[string]string, env map[string]string) error {
	// Remove existing container if it exists
	removeCmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}")
	output, err := removeCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Check if container exists
	containerExists := false
	for _, line := range []string{string(output)} {
		if line == name {
			containerExists = true
			break
		}
	}

	if containerExists {
		rmCmd := exec.Command("docker", "rm", "-f", name)
		rmCmd.Stdout = os.Stdout
		rmCmd.Stderr = os.Stderr
		if err := rmCmd.Run(); err != nil {
			return fmt.Errorf("failed to remove existing container: %w", err)
		}
	}

	// Build docker run command
	dockerArgs := []string{
		"run",
		"-d",
		"--name", name,
		"--restart", "unless-stopped",
	}

	// Add port mappings
	for hostPort, containerPort := range ports {
		dockerArgs = append(dockerArgs, "-p", fmt.Sprintf("%s:%s", hostPort, containerPort))
	}

	// Add environment variables
	for key, value := range env {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add image
	dockerArgs = append(dockerArgs, image)

	// Execute docker command
	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run failed: %w", err)
	}

	return nil
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/vupham90/containers/keychain"
)

func main() {
	app := &cli.App{
		Name:  "containers",
		Usage: "Container-based utility tools",
		Commands: []*cli.Command{
			{
				Name:  "pdf-compress",
				Usage: "Compress PDF files using Ghostscript",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "quality",
						Aliases:  []string{"q"},
						Usage:    "Compression quality: ebook, screen, printer, prepress, default",
						Value:    "ebook",
						Required: false,
					},
				},
				ArgsUsage: "<file-path>",
				Action: func(c *cli.Context) error {
					if c.NArg() != 1 {
						return fmt.Errorf("expected 1 argument: file-path")
					}

					filePath := c.Args().Get(0)
					quality := c.String("quality")

					// Validate quality
					validQualities := map[string]bool{
						"ebook":    true,
						"screen":   true,
						"printer":  true,
						"prepress": true,
						"default":  true,
					}
					if !validQualities[quality] {
						return fmt.Errorf("invalid quality: %s", quality)
					}

					// Resolve absolute path
					absFilePath, err := filepath.Abs(filePath)
					if err != nil {
						return fmt.Errorf("failed to resolve file path: %w", err)
					}

					// Verify file exists
					if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
						return fmt.Errorf("file does not exist: %s", absFilePath)
					}

					/*
						docker run \
						  --rm \
						  -v ~/Downloads:/workspace \
						  -w /workspace \
						  --entrypoint sh \
						  ghcr.io/vupham90/containers-pdf-compress:latest \
						  -c "gs -sDEVICE=pdfwrite -dCompatibilityLevel=1.4 -dPDFSETTINGS=/ebook -o /workspace/out.pdf /workspace/ALPINE.pdf && ls -la /workspace/out.pdf"
					*/

					// Generate output path
					dir := filepath.Dir(absFilePath)
					base := strings.TrimSuffix(filepath.Base(absFilePath), ".pdf")
					outputFilename := fmt.Sprintf("%s_%s.pdf", base, quality)

					// Prepare Docker arguments for Ghostscript
					image := "ghcr.io/vupham90/containers-pdf-compress:latest"
					workDir := dir
					args := []string{
						"-sDEVICE=pdfwrite",
						"-dCompatibilityLevel=1.4",
						fmt.Sprintf("-dPDFSETTINGS=/%s", quality),
						"-o", "/workspace/" + outputFilename,
						"/workspace/" + filepath.Base(absFilePath),
					}
					return RunContainer(image, workDir, args, nil, nil)
				},
			},
			{
				Name:  "ibgateway",
				Usage: "Start IB Gateway container for Interactive Brokers",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "user",
						EnvVars:  []string{"TWS_USERID"},
						Usage:    "Interactive Brokers username",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "password",
						EnvVars:  []string{"TWS_PASSWORD"},
						Usage:    "Interactive Brokers password",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "mode",
						EnvVars: []string{"TRADING_MODE"},
						Usage:   "Trading mode: paper or live",
						Value:   "paper",
					},
					&cli.StringFlag{
						Name:  "image",
						Usage: "Docker image to use",
						Value: "ghcr.io/gnzsnz/ib-gateway:latest",
					},
					&cli.StringFlag{
						Name:  "name",
						Usage: "Container name",
						Value: "ibgateway",
					},
				},
				Action: func(c *cli.Context) error {
					user := c.String("user")
					password := c.String("password")
					mode := c.String("mode")
					image := c.String("image")
					name := c.String("name")

					// Validate trading mode
					if mode != "paper" && mode != "live" {
						return fmt.Errorf("invalid trading mode: %s (must be 'paper' or 'live')", mode)
					}

					// Configure port mappings
					ports := map[string]string{
						"4001": "4003",
						"4002": "4004",
					}

					// Configure environment variables
					env := map[string]string{
						"TWS_USERID":   user,
						"TWS_PASSWORD": password,
						"TRADING_MODE": mode,
					}

					fmt.Printf("Starting IB Gateway container '%s' in %s mode...\n", name, mode)
					return RunDaemon(name, image, ports, env)
				},
			},
			{
				Name:  "bw-backup",
				Usage: "Backup Bitwarden vault to local directory",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "client-id",
						Aliases: []string{"c"},
						Usage:   "Bitwarden API client ID",
					},
					&cli.StringFlag{
						Name:    "client-secret",
						Aliases: []string{"s"},
						Usage:   "Bitwarden API client secret",
					},
					&cli.StringFlag{
						Name:    "password",
						Aliases: []string{"p"},
						Usage:   "Bitwarden master password",
					},
					&cli.StringFlag{
						Name:    "backup-dir",
						Aliases: []string{"d"},
						Usage:   "Backup destination directory",
						Value:   "./backups",
					},
					&cli.BoolFlag{
						Name:    "reset",
						Aliases: []string{"r"},
						Usage:   "Reset all credentials and re-enter them",
					},
				},
				Action: runBwBackup,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// getCredential retrieves a credential from CLI flag or macOS Keychain
func getCredential(flagValue, keychainAccount string, reset bool) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	// Use keychain with reset flag
	serviceName := "containers-bw-backup"
	return keychain.GetOrSetPassword(serviceName, keychainAccount, reset)
}

// runBwBackup executes the Bitwarden backup command
func runBwBackup(c *cli.Context) error {
	reset := c.Bool("reset")

	// Get credentials (flags or Keychain with reset option)
	clientID, err := getCredential(c.String("client-id"), "bitwarden_client_id", reset)
	if err != nil {
		return err
	}
	clientSecret, err := getCredential(c.String("client-secret"), "bitwarden_client_secret", reset)
	if err != nil {
		return err
	}
	password, err := getCredential(c.String("password"), "bitwarden_password", reset)
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
	env := map[string]string{
		"BW_CLIENTID":     clientID,
		"BW_CLIENTSECRET": clientSecret,
		"BW_PASSWORD":     password,
	}

	// Add tmpfs mounts for security
	tmpfs := []string{"/tmp", "/var/tmp"}

	// Execute backup container
	image := "ghcr.io/vupham90/containers-bw-backup:latest"
	fmt.Println("Starting Bitwarden backup...")
	return RunContainer(image, absBackupDir, []string{}, env, tmpfs)
}

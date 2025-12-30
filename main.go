package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
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
					return RunContainer(image, workDir, args, nil, nil, nil, true)
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
					env := map[string]EnvVar{
						"TWS_USERID":   {Value: user, Sensitive: true},
						"TWS_PASSWORD": {Value: password, Sensitive: true},
						"TRADING_MODE": {Value: mode, Sensitive: false},
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
						Name:    "profile",
						Aliases: []string{"P"},
						Usage:   "Profile name for multi-account support (optional, uses default keychain if empty)",
					},
					&cli.StringFlag{
						Name:    "organization-id",
						Aliases: []string{"o"},
						Usage:   "Bitwarden organization ID to backup (optional)",
					},
					&cli.StringFlag{
						Name:  "profiles",
						Usage: "Path to YAML config file for batch backup mode",
					},
					&cli.StringFlag{
						Name:    "client-id",
						Aliases: []string{"c"},
						Usage:   "Bitwarden API client ID (optional, uses keychain if not provided)",
					},
					&cli.StringFlag{
						Name:    "client-secret",
						Aliases: []string{"s"},
						Usage:   "Bitwarden API client secret (optional, uses keychain if not provided)",
					},
					&cli.StringFlag{
						Name:    "password",
						Aliases: []string{"p"},
						Usage:   "Bitwarden master password (optional, uses keychain if not provided)",
					},
					&cli.BoolFlag{
						Name:    "encrypt",
						Aliases: []string{"e"},
						Usage:   "Enable backup encryption (password from keychain)",
					},
					&cli.StringFlag{
						Name:    "backup-password",
						Aliases: []string{"bp"},
						Usage:   "Password for encrypted backup (overrides keychain if provided)",
					},
					&cli.StringFlag{
						Name:    "backup-dir",
						Aliases: []string{"d"},
						Usage:   "Backup destination directory (required for single mode)",
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

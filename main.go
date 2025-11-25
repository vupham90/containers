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

					// Generate output path
					dir := filepath.Dir(absFilePath)
					base := strings.TrimSuffix(filepath.Base(absFilePath), ".pdf")
					outputFilename := fmt.Sprintf("%s_%s.pdf", base, quality)

					// Prepare Docker arguments for Ghostscript
					image := "ghcr.io/vupham90/containers-pdf-compress:latest"
					workDir := dir
					args := []string{
						"-dNODISPLAY", // Prevent Ghostscript from trying to open X display
						"-sDEVICE=pdfwrite",
						"-dCompatibilityLevel=1.4",
						fmt.Sprintf("-dPDFSETTINGS=/%s", quality),
						"-dNOPAUSE",
						"-dQUIET",
						"-dBATCH",
						fmt.Sprintf("-sOutputFile=%s", outputFilename),
						filepath.Base(absFilePath),
					}

					return RunContainer(image, workDir, args)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

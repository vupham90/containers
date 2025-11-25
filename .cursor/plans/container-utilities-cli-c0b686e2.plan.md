<!-- c0b686e2-1e64-48b1-8d0c-c502870fd9d5 8b554c2d-41e2-4ae4-8f02-680c893ba3f7 -->
# Container Utilities CLI Plan

## Repository Structure

```
containers/
├── main.go                       # Main CLI entry point using urfave/cli
├── docker.go                     # Docker execution wrapper functions
├── dockerfiles/
│   ├── pdf-compress/
│   │   ├── Dockerfile           # PDF compression utility
│   │   └── README.md            # Utility-specific docs
│   └── [other-util]/
│       ├── Dockerfile
│       └── README.md
├── .github/
│   └── workflows/
│       └── build-images.yml     # Build and push all Docker images
├── go.mod
├── go.sum
└── README.md                     # Project documentation
```

## Implementation Details

### 1. Go CLI Binary (`main.go`)

- Use urfave/cli v2 (latest stable)
- Define app with name "containers"
- Each utility becomes a subcommand:
  - `pdf-compress` - PDF file size reduction
  - Additional utilities added as new subcommands
- Each subcommand:
  - Accepts input/output paths as flags or arguments
  - Calls `internal/docker/runner.go` to execute `docker run`
  - Uses image: `ghcr.io/[username]/containers-[util-name]:latest`
  - Mounts volumes for file I/O
  - Streams container output to stdout/stderr

**Code Sample (`main.go`):**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"containers/internal/docker"
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
						"ebook": true, "screen": true, "printer": true,
						"prepress": true, "default": true,
					}
					if !validQualities[quality] {
						return fmt.Errorf("invalid quality: %s", quality)
					}
					
					// Generate output path
					dir := filepath.Dir(filePath)
					base := strings.TrimSuffix(filepath.Base(filePath), ".pdf")
					outputPath := filepath.Join(dir, fmt.Sprintf("%s_%s.pdf", base, quality))
					
					// Prepare Docker arguments
					image := "ghcr.io/[username]/containers-pdf-compress:latest"
					workDir := dir
					args := []string{
						"-sDEVICE=pdfwrite",
						"-dCompatibilityLevel=1.4",
						fmt.Sprintf("-dPDFSETTINGS=/%s", quality),
						"-dNOPAUSE",
						"-dQUIET",
						"-dBATCH",
						fmt.Sprintf("-sOutputFile=%s", filepath.Base(outputPath)),
						filepath.Base(filePath),
					}
					
					return docker.RunContainer(image, workDir, args)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

### 2. Docker Runner (`internal/docker/runner.go`)

- Function: `RunContainer(image string, workDir string, args []string) error`
- Executes `docker run` with:
  - Image from GitHub Container Registry
  - Volume mounts the working directory (where input file is located) to `/workspace` in container
  - Passes through command arguments
  - Handles errors and container exit codes
- **Volume Mount Strategy**: Two possible approaches:

**Option A: Mount directory containing input file** (current plan)

  - Mounts the directory where the input file is located
  - Example: Input `/Users/john/documents/report.pdf` → mounts `/Users/john/documents/` → `/workspace`
  - Pros: Works with absolute paths from anywhere
  - Cons: More path resolution logic

**Option B: Mount current working directory** (alternative)

  - Mounts the directory where user runs the command (os.Getwd())
  - Example: User in `/Users/john/`, runs `containers pdf-compress documents/report.pdf` → mounts `/Users/john/` → `/workspace`
  - Pros: Simpler, more intuitive for relative paths
  - Cons: Requires files to be relative to current directory or absolute paths need special handling

**Recommendation**: Option A (mount file's directory) for maximum flexibility

**Code Sample (`internal/docker/runner.go`):**

```go
package docker

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
		"--rm",                    // Remove container after execution
		"-v",                      // Volume mount flag
		fmt.Sprintf("%s:/workspace", absWorkDir), // Mount host directory to /workspace
		"-w", "/workspace",        // Set working directory inside container
		image,                     // Docker image
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
```

**Example Execution Flow:**

1. **User command**: `containers pdf-compress /Users/john/documents/report.pdf --quality ebook`

2. **main.go processing**:

   - Input: `/Users/john/documents/report.pdf`
   - Work directory: `/Users/john/documents/`
   - Output filename: `report_ebook.pdf` (same directory)
   - Ghostscript args: `["gs", "-sDEVICE=pdfwrite", "-dPDFSETTINGS=/ebook", "-sOutputFile=report_ebook.pdf", "report.pdf"]`

3. **Docker command executed**:
   ```bash
   docker run --rm \
     -v /Users/john/documents:/workspace \
     -w /workspace \
     ghcr.io/[username]/containers-pdf-compress:latest \
     gs -sDEVICE=pdfwrite \
        -dCompatibilityLevel=1.4 \
        -dPDFSETTINGS=/ebook \
        -dNOPAUSE -dQUIET -dBATCH \
        -sOutputFile=report_ebook.pdf \
        report.pdf
   ```

4. **Inside container**:

   - `/workspace/report.pdf` → input file (mounted from host)
   - `/workspace/report_ebook.pdf` → output file (written to mounted volume)
   - After container exits, output file appears in `/Users/john/documents/` on host

**Design Note - File Path Handling:**

Each subcommand is responsible for:

1. **Identifying file path arguments** - Each utility knows which args are file paths (e.g., first arg for pdf-compress)
2. **Resolving work directory** - Extract directory from file path(s) using `filepath.Dir()`
3. **Converting to container-relative paths** - Use `filepath.Base()` for filenames inside container (since we mount the directory)
4. **Passing workDir to docker runner** - The runner mounts this directory as `/workspace`

**Example for different utilities:**

- `pdf-compress <file>`: First arg is file path → extract its directory
- `image-resize <input> <output>`: Both args are file paths → use common directory or first file's directory
- `text-convert <file> --format json`: First arg is file path → extract its directory

**Optional Helper Function** (for consistency across utilities):

```go
// internal/docker/paths.go
package docker

import (
	"fmt"
	"path/filepath"
)

// GetWorkDirFromPaths extracts the directory from file path(s) to mount
// For single file: returns its directory
// For multiple files: returns common directory (or first file's directory)
func GetWorkDirFromPaths(paths ...string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("no paths provided")
	}
	
	// For single file, use its directory
	if len(paths) == 1 {
		absPath, err := filepath.Abs(paths[0])
		if err != nil {
			return "", err
		}
		return filepath.Dir(absPath), nil
	}
	
	// For multiple files, use first file's directory
	// (Could be enhanced to find common path prefix)
	absPath, err := filepath.Abs(paths[0])
	if err != nil {
		return "", err
	}
	return filepath.Dir(absPath), nil
}
```

### 3. Dockerfiles Structure (`dockerfiles/[util-name]/Dockerfile`)

- Each utility has isolated Dockerfile
- Example for pdf-compress:
  - Base image (e.g., `python:3-slim` or `alpine` with tools)
  - Install PDF compression tools (e.g., `ghostscript`, `qpdf`)
  - Entrypoint script or direct command
- Self-contained, no dependencies on other utilities

### 4. GitHub Actions Workflow (`.github/workflows/build-images.yml`)

- Matrix strategy to build all Docker images
- For each `dockerfiles/*/` directory:
  - Build Docker image
  - Tag as `ghcr.io/[username]/containers-[name]:latest`
  - Push to GitHub Container Registry
- Trigger on:
  - Push to main branch
  - Manual workflow dispatch
  - Changes to `dockerfiles/**`

### 5. Example: PDF Compress Command

- Subcommand: `containers pdf-compress [flags] <input> <output>`
- Flags:
  - `--quality` or `--level` for compression level
  - `--format` for output format
- Executes: `docker run --rm -v $(pwd):/workspace ghcr.io/[username]/containers-pdf-compress:latest [args]`

## Key Files to Create

1. **`cmd/containers/main.go`**: CLI app with subcommands
2. **`internal/docker/runner.go`**: Docker execution wrapper
3. **`dockerfiles/pdf-compress/Dockerfile`**: Ghostscript-based PDF compression utility
4. **`.github/workflows/build-images.yml`**: CI/CD for Docker images
5. **`go.mod`**: Go module with urfave/cli dependency
6. **`README.md`**: Usage instructions and development guide

## Design Decisions

- **Single binary**: Easier distribution and usage
- **Subcommands**: Clean separation of utilities
- **Docker runner abstraction**: Reusable across all utilities
- **Matrix build**: Efficient CI/CD for multiple images
- **Volume mounts**: Simple file I/O between host and containers
- **No auto-pull**: Users must ensure images are available (documented in README)

### To-dos

- [ ] Initialize Go module and add urfave/cli v2 dependency
- [ ] Implement internal/docker/runner.go with Docker execution logic
- [ ] Build cmd/containers/main.go with urfave/cli app structure and pdf-compress subcommand
- [ ] Create dockerfiles/pdf-compress/Dockerfile with PDF compression tools
- [ ] Set up .github/workflows/build-images.yml with matrix strategy for all Dockerfiles
- [ ] Write README.md with usage instructions, development guide, and contribution guidelines
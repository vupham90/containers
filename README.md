# Container Utilities CLI

A collection of containerized utility tools wrapped in a convenient Go CLI binary. Each utility runs in its own Docker container, providing isolated and reproducible execution environments.

## Features

- Single binary with subcommands for each utility
- Docker-based execution for isolation and reproducibility
- GitHub Actions automatically builds and publishes Docker images
- Easy to extend with new utilities

## Installation

### Prerequisites

- Go 1.19 or later
- Docker installed and running

### Building the CLI

```bash
git clone <repository-url>
cd containers
go build -o containers
```

The binary will be created as `containers` in the current directory.

## Usage

### PDF Compress

Compress PDF files using Ghostscript with various quality settings.

```bash
containers pdf-compress <file-path> [--quality <quality>]
```

**Quality Options:**
- `ebook` - Good quality, smaller file size (default)
- `screen` - Lower quality, smallest file size
- `printer` - Higher quality, larger file size
- `prepress` - Highest quality, largest file size
- `default` - Default Ghostscript settings

**Examples:**

```bash
# Compress with default quality (ebook)
containers pdf-compress document.pdf

# Compress with specific quality
containers pdf-compress document.pdf --quality screen

# Using short flag
containers pdf-compress document.pdf -q printer
```

**Output:**
- Input: `document.pdf`
- Output: `document_ebook.pdf` (in the same directory)

## Docker Images

Docker images are automatically built and published to GitHub Container Registry via GitHub Actions.

**Image naming format:**
```
ghcr.io/<username>/containers-<util-name>:latest
```

**Pulling images manually:**
```bash
docker pull ghcr.io/<username>/containers-pdf-compress:latest
```

**Note:** Before using the CLI, ensure the required Docker images are available. You can either:
1. Pull them manually (see above)
2. Build them locally (see Development section)
3. Wait for GitHub Actions to build them after pushing to the repository

## Development

### Project Structure

```
containers/
├── main.go                       # Main CLI entry point
├── docker.go                     # Docker execution wrapper
├── dockerfiles/
│   ├── pdf-compress/
│   │   ├── Dockerfile
│   │   └── README.md
│   └── [other-util]/
├── .github/
│   └── workflows/
│       └── build-images.yml
├── go.mod
└── README.md
```

### Adding a New Utility

1. **Create Dockerfile:**
   ```bash
   mkdir -p dockerfiles/new-util
   # Create dockerfiles/new-util/Dockerfile
   ```

2. **Add subcommand to main.go:**
   ```go
   {
       Name:  "new-util",
       Usage: "Description of the utility",
       // ... flags and action
   }
   ```

3. **Update GitHub Actions workflow:**
   Add the new utility to the matrix in `.github/workflows/build-images.yml`:
   ```yaml
   matrix:
     util:
       - pdf-compress
       - new-util
   ```

4. **Update image name in main.go:**
   Change the placeholder `[username]` to your GitHub username or organization.

### Building Docker Images Locally

```bash
# Build a specific utility
docker build -t ghcr.io/<username>/containers-pdf-compress:latest ./dockerfiles/pdf-compress

# Or build all utilities
for dir in dockerfiles/*/; do
    util=$(basename "$dir")
    docker build -t ghcr.io/<username>/containers-${util}:latest "./dockerfiles/${util}"
done
```

### Testing

```bash
# Build the CLI
go build -o containers

# Test PDF compression
./containers pdf-compress test.pdf --quality screen
```

## Configuration

### Updating Image Registry

The Docker image names are hardcoded in `main.go`. To use a different registry or naming convention:

1. Update the `image` variable in each subcommand's `Action` function
2. Update the GitHub Actions workflow if needed

**Example:**
```go
image := "ghcr.io/your-username/containers-pdf-compress:latest"
```

## How It Works

1. **CLI receives command** - User runs a subcommand with arguments
2. **Path resolution** - The CLI extracts the directory containing input files
3. **Docker execution** - The directory is mounted to `/workspace` in the container
4. **Container processing** - The utility processes files inside the container
5. **Output** - Results are written to the mounted directory, accessible on the host

**Volume Mount Strategy:**
- The directory containing the input file is mounted to `/workspace` in the container
- Files are referenced by their basename (filename only) inside the container
- Output files appear in the same directory on the host

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add your utility following the structure above
4. Submit a pull request

## License

[Add your license here]


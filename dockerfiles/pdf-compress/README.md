# PDF Compress Utility

This utility compresses PDF files using Ghostscript.

## Usage

The utility is invoked through the main CLI:

```bash
containers pdf-compress <file-path> [--quality <quality>]
```

## Quality Options

- `ebook` - Good quality, smaller file size (default)
- `screen` - Lower quality, smallest file size
- `printer` - Higher quality, larger file size
- `prepress` - Highest quality, largest file size
- `default` - Default Ghostscript settings

## Docker Image

The Docker image is built and published to GitHub Container Registry as:
`ghcr.io/[username]/containers-pdf-compress:latest`

## How It Works

1. The input PDF file path is provided
2. The directory containing the file is mounted to `/workspace` in the container
3. Ghostscript compresses the PDF using the specified quality setting
4. The output file is written to the same directory with `_{quality}` suffix
5. Example: `document.pdf` → `document_ebook.pdf`


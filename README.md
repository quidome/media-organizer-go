# media-organizer-go

A command-line tool written in Go to organize media files (photos and videos) into a structured library based on their creation dates.

## Features

- **Scan Media Files**: Recursively scans directories for supported media formats (JPG, PNG, MP4, MOV, etc.)
- **Creation Date Attribution**: Determines the best creation timestamp using a priority order:
  1. Embedded metadata (EXIF for photos, container metadata for videos)
  2. Filename parsing
  3. Filesystem modification time as fallback
- **Deduplication**: Identifies and handles exact duplicate files based on content
- **Organized Structure**: Copies files into a partitioned layout: `<dest>/YYYY/MM/DD/filename.ext`
- **Collision Resolution**: Automatically handles naming conflicts by appending suffixes (e.g., `photo_1.jpg`)
- **Safe Operations**: Never overwrites existing files; supports dry-run mode
- **Multiple Output Formats**: Human-readable text or machine-readable JSON

## Installation

### From Source

```bash
git clone https://github.com/quidome/media-organizer-go.git
cd media-organizer-go
go build -o bin/media-organizer ./cmd/media-organizer
```

### Using Go Install

```bash
go install github.com/quidome/media-organizer-go/cmd/media-organizer@latest
```

Make sure `~/go/bin` is in your PATH.

### Development Environment

This project uses [Nix](https://nixos.org/) for reproducible development environments. If you have Nix installed:

```bash
nix develop
```

Or use [direnv](https://direnv.net/) with the provided `.envrc`.

## Usage

### Scan Directory

Scan a directory for media files and print their paths:

```bash
media-organizer scan /path/to/photos
```

Options:
- `--max-depth N`: Limit recursion depth (default: unlimited)
- `--json`: Output detailed JSON records including creation date candidates
- `--verbose`: Show additional information

### Organize Media

Organize media files from source to destination directory:

```bash
media-organizer organize /source/directory /destination/library
```

By default, this performs a dry-run showing what would be copied. Use `--execute` to actually perform the operations:

```bash
media-organizer organize --execute /source/directory /destination/library
```

Options:
- `--execute`, `-x`: Execute copy operations (default: dry-run)
- `--json`: Output operations as JSON
- `--verbose`: Show progress and statistics

### Examples

**Dry-run organization:**
```bash
media-organizer organize ~/Downloads/photos ~/Pictures/organized
```

**Execute organization:**
```bash
media-organizer organize --execute ~/Downloads/photos ~/Pictures/organized
```

**Scan with JSON output:**
```bash
media-organizer scan --json ~/Pictures/raw | jq .
```

**Organize with JSON output:**
```bash
media-organizer organize --json ~/Downloads/photos ~/Pictures/organized
```

## Supported Formats

### Photo Formats
- JPG/JPEG, PNG, GIF, WebP, HEIC, TIFF, BMP

### Video Formats
- MP4, MOV, M4V, MKV, AVI, WebM, MTS, 3GP

## How It Works

The tool follows a multi-stage pipeline:

1. **Discover**: Find all media files in the source directory
2. **Attribute**: Determine creation timestamps from metadata, filename, or filesystem
3. **Deduplicate**: Identify and skip exact duplicate files
4. **Plan**: Calculate destination paths in YYYY/MM/DD structure
5. **Reconcile**: Check destination for conflicts and resolve naming collisions
6. **Materialize**: Copy files (only in execute mode)

For detailed pipeline information, see [PIPELINE.md](PIPELINE.md).

## Development

### Prerequisites

- Go 1.23+
- [golangci-lint](https://golangci-lint.run/) for linting

### Commands

- `just build`: Build the binary
- `just test`: Run tests
- `just lint`: Lint the code
- `just fmt`: Format the code
- `just test-coverage`: Run tests with coverage

See [justfile](justfile) for all available commands.

### Project Structure

- `cmd/media-organizer/`: CLI entry point
- `pkg/scan/`: Directory scanning logic
- `pkg/createdat/`: Creation timestamp attribution
- `pkg/plan/`: Destination path planning
- `pkg/reconcile/`: Conflict resolution and deduplication
- `pkg/copy/`: File copying operations

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass and code is linted
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.
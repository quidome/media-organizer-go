# Agent Guidelines for media-organizer-go

## Build & Test Commands
- Build: `go build ./...`
- Run: `go run ./cmd/media-organizer`
- Test all: `go test ./...`
- Test single: `go test ./path/to/package -run TestName`
- Test with coverage: `go test -cover ./...`
- Lint: `golangci-lint run` (or `go vet ./...`)
- Format: `gofmt -s -w .` or `goimports -w .`

## Code Style
- **Imports**: Use `goimports`, group standard library, external packages, then internal packages
- **Formatting**: Follow `gofmt` standard, tabs for indentation
- **Naming**: CamelCase for exports, camelCase for private; short receiver names (1-2 chars); avoid stuttering (pkg.PkgMethod)
- **Types**: Prefer interfaces in consuming packages; use pointer receivers for methods that modify state
- **Error Handling**: Always check errors; wrap with context using `fmt.Errorf("context: %w", err)`; return errors, don't panic
- **Comments**: Document all exported functions/types with complete sentences starting with the name
- **Structure**: Organize by feature/domain, not by layer (handlers, services, etc. together by feature)

## Project Conventions
- Use Go modules (`go.mod`)
- Entry points in `cmd/` directory, library code in `pkg/` or root
- Keep dependencies minimal and audited
- Use `just` (not `make`) for task automation - see `justfile` for available commands
- Development environment managed with Nix flake (`nix develop` or use direnv)

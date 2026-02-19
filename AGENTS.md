# AI Agent Context: Bitrise CodePush CLI Plugin

This file provides context for any AI coding assistant working on this project.

## Project Overview

CodePush CLI is a Bitrise CLI plugin and standalone tool for managing over-the-air (OTA) updates for mobile applications. It handles pushing updates, rolling back deployments, and integrating the Bitrise CodePush SDK into mobile projects.

- **Language**: Go 1.24+
- **CLI Framework**: Cobra
- **Build/Release**: GoReleaser
- **Module**: `github.com/bitrise-io/bitrise-plugins-codepush-cli`

## Verification Commands

Run all after making changes:

```bash
go build ./cmd/codepush        # Build
go test ./...                  # Test
go vet ./...                   # Static analysis
```

## Directory Structure

```
cmd/codepush/          CLI entry point, Cobra commands
internal/bitrise/      Bitrise CI environment integration
internal/codepush/     Core CodePush logic
```

## Code Conventions

- **Error handling**: `fmt.Errorf("message: %w", err)`, return errors, don't panic
- **Logging**: stderr for progress, stdout for structured output
- **File paths**: Use `filepath` package, validate with `os.Stat()`
- **Tests**: `*_test.go` colocated with source, table-driven preferred
- **Writing style**: Never use em dashes in any content

## Commands

```bash
codepush push [bundle-path]    # Push OTA update
codepush rollback              # Rollback deployment
codepush integrate             # Integrate SDK into project
codepush version               # Print version
```

## Bitrise Plugin

- Manifest: `bitrise-plugin.yml`
- Usage: `bitrise :codepush <command>`
- Auto-detects Bitrise environment via `BITRISE_BUILD_NUMBER` / `BITRISE_DEPLOY_DIR`

## Workflow

- Branch-based development, never commit to `main` directly
- Conventional commits: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`
- Run verification commands before pushing

## Release

1. Update version in `cmd/codepush/main.go` and `bitrise-plugin.yml`
2. Tag and release with GoReleaser
3. Binaries: `codepush-Darwin-arm64`, `codepush-Darwin-x86_64`, `codepush-Linux-x86_64`

## Common Pitfalls

- Version must be updated in both `main.go` and `bitrise-plugin.yml`
- Binary names in `.goreleaser.yml` must match `bitrise-plugin.yml` URLs
- Use stderr for logs, stdout for output (don't mix)
- Always use `%w` for error wrapping

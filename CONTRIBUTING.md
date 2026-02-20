# Contributing to CodePush CLI

## Getting Started

```bash
git clone https://github.com/bitrise-io/bitrise-plugins-codepush-cli.git
cd bitrise-plugins-codepush-cli
go build ./cmd/codepush
./codepush version
```

## Verification Commands

Run all before pushing:

```bash
go build ./cmd/codepush        # Build
go test ./...                  # Test
go vet ./...                   # Static analysis
```

## Workflow

1. Create a branch from `main`
2. Make changes
3. Run verification commands
4. Commit with conventional commit messages
5. Push and open a PR

## Commit Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` new feature
- `fix:` bug fix
- `chore:` maintenance (deps, configs)
- `docs:` documentation changes
- `refactor:` code restructuring
- `test:` adding or updating tests

## Code Conventions

### Go Idioms
- Error wrapping: `fmt.Errorf("message: %w", err)`
- Return errors, don't panic
- Use `filepath` for cross-platform paths
- Table-driven tests

### Logging
- `os.Stderr` for progress and log messages
- `os.Stdout` for structured output (JSON, etc.)

### Testing
- Test files: `*_test.go` next to source files
- Run: `go test ./...`
- Coverage: `go test -cover ./...`

## Project Structure

```
cmd/codepush/          CLI entry point (Cobra commands)
internal/bitrise/      Bitrise CI environment integration
internal/codepush/     Core CodePush logic
```

## Adding a Command

1. Define a `cobra.Command` in `cmd/codepush/main.go`
2. Register with `rootCmd.AddCommand()` in `init()`
3. Implement logic in `internal/` packages
4. Add tests
5. Update README.md

## Release Process

Releases are automated via Bitrise. Pushing a tag triggers the `pipeline_release` pipeline, which runs GoReleaser to build binaries and create a GitHub Release.

### Using the Claude Code skill (recommended)

```
/bump-version-and-release X.Y.Z
```

This walks you through the full flow: version bump, PR, tag, and verification.

### Manual steps

1. Update version in `cmd/codepush/main.go`
2. Update download URLs in `bitrise-plugin.yml` with the new version
3. Open a PR, wait for CI to pass, and merge to `main`
4. Create and push the tag:
   ```bash
   git pull origin main
   git tag -a X.Y.Z -m "Release X.Y.Z"
   git push origin X.Y.Z
   ```
5. Bitrise `pipeline_release` triggers automatically and runs GoReleaser
6. Verify: `gh release view X.Y.Z`

### Prerequisites

- A `GITHUB_TOKEN` Bitrise Secret with `repo` scope (or fine-grained `contents: write`)
- Tags use bare semantic versions without `v` prefix: `0.1.0`, not `v0.1.0`

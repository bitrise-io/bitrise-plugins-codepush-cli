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

1. Update version in `cmd/codepush/main.go`
2. Update URLs in `bitrise-plugin.yml` with new version
3. Commit: `git commit -m "chore: bump version to X.Y.Z"`
4. Push to `main`
5. Tag: `git tag -a X.Y.Z -m "Release X.Y.Z"`
6. Release: `GITHUB_TOKEN=$(gh auth token) goreleaser release --clean`
7. Verify: `gh release view X.Y.Z`

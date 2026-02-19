# Claude Context: Bitrise CodePush CLI Plugin

## Project Overview

CodePush CLI is a **Bitrise CLI plugin** and standalone CLI tool for managing over-the-air (OTA) updates for mobile applications using the Bitrise CodePush SDK. It handles pushing updates, rolling back deployments, and integrating the SDK into mobile projects.

**Primary Use Case**: Bitrise CI/CD workflows (OTA update deployment)
**Secondary Use Cases**: Local development and testing, SDK integration into projects

## Key Value Propositions

1. **OTA Update Management**: Push, rollback, and manage CodePush releases
2. **SDK Integration**: Automatically configure CodePush SDK in mobile projects
3. **Bitrise CI Integration**: Auto-detects Bitrise environment, exports results to deploy directory
4. **Standalone CLI**: Works outside Bitrise as a standalone tool

## Architecture

### Language & Framework
- **Language**: Go 1.24+
- **CLI Framework**: Cobra (`github.com/spf13/cobra`)
- **Build System**: Go modules + GoReleaser

### Directory Structure

```
bitrise-plugins-codepush-cli/
├── cmd/codepush/            # CLI entry point (main.go)
├── internal/
│   ├── bitrise/             # Bitrise CI integration (env detection, deploy export)
│   ├── bundler/             # JS bundle generation (detect, bundle, Hermes)
│   ├── codepush/            # Core CodePush logic
│   └── output/              # Styled terminal output (lipgloss, huh)
├── bitrise.yml              # CI pipeline (build, test, coverage, vet)
├── bitrise-plugin.yml       # Bitrise plugin manifest
├── .goreleaser.yml          # Release automation
└── go.mod                   # Go module definition
```

### Key Files

- **cmd/codepush/main.go**: CLI entry point, Cobra commands, flag parsing
- **internal/output/output.go**: Styled terminal output (Writer type, Step, Success, Error, etc.)
- **internal/bitrise/env.go**: Bitrise environment detection, build metadata, deploy directory export
- **internal/codepush/codepush.go**: Core CodePush business logic
- **bitrise-plugin.yml**: Plugin manifest with binary download URLs
- **.goreleaser.yml**: Cross-platform build and release configuration

## Verification Commands

Run **all** commands after making changes. Fix any failures before committing.

```bash
go build ./cmd/codepush                  # Build the binary
go test ./...                            # Run all tests
go vet ./...                             # Static analysis
```

CI enforces a **75% minimum test coverage** threshold. See `bitrise.yml` for the full pipeline.

## Command-Line Interface

### Commands
```bash
codepush push [bundle-path] [flags]      # Push an OTA update
codepush rollback [flags]                # Rollback to previous release
codepush integrate [flags]               # Integrate SDK into project
codepush version                         # Print version info
```

### Usage Patterns
```bash
# As Bitrise plugin
bitrise :codepush push
bitrise :codepush rollback

# As standalone CLI
./codepush push ./dist/bundle.js
./codepush rollback --deployment production
./codepush integrate
```

## Code Conventions

### Error Handling
- Use `fmt.Errorf("message: %w", err)` for wrapping errors
- Return errors, don't panic
- Use `out.Warning(...)` for non-fatal warnings, `out.Error(...)` for fatal errors

### CLI Output
- All human-readable output goes to stderr via `output.Writer`
- JSON output (`--json`) goes to stdout via `outputJSON()`
- Never use `fmt.Fprintf(os.Stderr, ...)` directly; use the `output.Writer` methods
- See "CLI Output Conventions" section below for full details

### File Paths
- Always use absolute paths when possible
- Use `filepath` package for cross-platform compatibility
- Validate paths with `os.Stat()` before use

### Testing
- Test files: `*_test.go` colocated with source
- Use `go test ./...` for all tests
- Use `go test -cover ./...` for coverage
- Table-driven tests are preferred

### Writing Style
- **Never use em dashes** (`---` or `\u2014`) in any content: titles, descriptions, metadata, UI copy, or comments. Use commas, periods, colons, or rewrite the sentence.

## CLI Output Conventions

All human-readable CLI output uses `internal/output.Writer` (Charmbracelet lipgloss + huh stack). Never write directly to `os.Stderr` for user-facing messages.

### Output Methods

| Method | When to use | Color mode | Plain mode |
|--------|-------------|-----------|------------|
| `out.Step(fmt, args)` | Progress steps ("Packaging...", "Resolving...") | Cyan `->` prefix | `-> message` |
| `out.Success(fmt, args)` | Command completion | Green bold `OK` | `OK message` |
| `out.Error(fmt, args)` | Fatal errors | Red bold `ERROR` | `ERROR message` |
| `out.Warning(fmt, args)` | Non-fatal warnings | Yellow bold `WARNING` | `WARNING message` |
| `out.Info(fmt, args)` | Supplementary details (indented under steps) | Dim, indented | `   message` |
| `out.Result([]KeyValue)` | Key-value results (push result, package info) | Bold keys, aligned | Aligned plain |
| `out.Table(headers, rows)` | Lists and history tables | Styled headers | Plain aligned |
| `out.Println(fmt, args)` | Plain text, titles | No styling | Plain |
| `out.Spinner(title, fn)` | Long operations (>500ms): upload, processing | Animated spinner | `-> title...` |
| `out.ConfirmDestructive(msg, yesFlag)` | Destructive operations (delete, clear) | Interactive y/N prompt | Error with `--yes` hint |

### Patterns

**Threading the Writer**: Pass `out *output.Writer` as a parameter to internal package functions. Do not use globals in `internal/` packages.

```go
func Push(client Client, opts *PushOptions, out *output.Writer) (*PushResult, error) {
    out.Step("Packaging bundle: %s", opts.BundlePath)
    // ...
}
```

**Spinner for long operations**:
```go
err := out.Spinner("Uploading package", func() error {
    return client.Upload(...)
})
```

**Destructive confirmation**:
```go
if err := out.ConfirmDestructive(
    fmt.Sprintf("This will permanently delete deployment %q", name),
    yesFlag,
); err != nil {
    return err
}
```

**Result display after success**:
```go
out.Success("Push successful")
out.Result([]output.KeyValue{
    {Key: "Package ID", Value: result.PackageID},
    {Key: "App version", Value: result.AppVersion},
})
```

### CI and Environment Detection

- Terminal detection: auto-detects via `term.IsTerminal()`
- CI detection: `CI` or `BITRISE_BUILD_NUMBER` env vars
- `NO_COLOR` env var: disables color output
- Non-interactive mode (CI or piped): no spinners, no prompts, plain text fallback

### Testing

- Use `output.NewTest(io.Discard)` for suppressed output in tests
- Use `output.NewTest(&buf)` when asserting on output content
- In `cmd/codepush/main_test.go`: `TestMain` sets `out = output.NewTest(io.Discard)`
- In internal packages: pass `output.NewTest(io.Discard)` as the `out` parameter

## Bitrise Integration

### Environment Variables (Read by Plugin)

| Variable | Purpose |
|----------|---------|
| `BITRISE_BUILD_NUMBER` | Build number for metadata |
| `BITRISE_DEPLOY_DIR` | Deploy directory for report export |
| `GIT_CLONE_COMMIT_HASH` | Git commit for metadata |

### Detection
```go
import "github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"

if bitrise.IsBitriseEnvironment() {
    metadata := bitrise.GetBuildMetadata()
    // metadata.BuildNumber, metadata.CommitHash, metadata.DeployDir
}
```

### Export to Deploy Directory
```go
destPath, err := bitrise.WriteToDeployDir("result.json", jsonData)
```

## Release Process

### Prerequisites
- `goreleaser` installed (`brew install goreleaser`)
- GitHub CLI authenticated (`gh auth login`)
- Clean git working directory, all tests passing

### Steps

1. Update version in `cmd/codepush/main.go`
2. Update `bitrise-plugin.yml` executable URLs with new version
3. Commit: `git commit -m "chore: bump version to X.Y.Z"`
4. Tag: `git tag -a X.Y.Z -m "Release X.Y.Z"`
5. Release: `GITHUB_TOKEN=$(gh auth token) goreleaser release --clean`
6. Verify: `gh release view X.Y.Z`

### Release Artifacts

| File | Description |
|------|-------------|
| `codepush-Darwin-arm64` | macOS Apple Silicon binary |
| `codepush-Darwin-x86_64` | macOS Intel binary |
| `codepush-Linux-x86_64` | Linux x86_64 binary |
| `checksums.txt` | SHA256 checksums |

## Common Development Tasks

### Building
```bash
go build -o codepush ./cmd/codepush
./codepush version
```

### Adding a New Command
1. Add `cobra.Command` in `cmd/codepush/main.go`
2. Register with `rootCmd.AddCommand()` in `init()`
3. Implement logic in `internal/` packages
4. Add tests
5. Update README.md

### Adding a New Flag
1. Add flag variable at package level in `main.go`
2. Register with `cmd.Flags()` in `init()`
3. Use in the command's `RunE` function
4. Update README.md

## Common Pitfalls

1. **Version sync**: When releasing, update BOTH `cmd/codepush/main.go` version AND `bitrise-plugin.yml` URLs. Missing either causes failures.
2. **Binary naming**: GoReleaser binary names must match `bitrise-plugin.yml` executable URLs exactly (`codepush-Darwin-arm64`, etc.).
3. **CGO_ENABLED=0**: Required for cross-compilation. If you add C dependencies, the goreleaser config needs updating.
4. **Stderr vs stdout**: Use `output.Writer` for all human output (goes to stderr). JSON (`--json`) goes to stdout. Never mix them.
5. **Error wrapping**: Always use `%w` verb for error wrapping to preserve error chains.

## Questions to Ask When Modifying

1. **Does this affect Bitrise integration?** Check `internal/bitrise/`
2. **Does this add a new command?** Update `main.go`, README.md
3. **Does this add a new flag?** Update `main.go`, README.md
4. **Does this need tests?** Yes, always add tests
5. **Does this need documentation?** Update README.md and/or CONTRIBUTING.md
6. **Does this affect the release?** Check `.goreleaser.yml` and `bitrise-plugin.yml`

## Workflow

- Always work on a branch, never commit directly to `main`
- Use conventional commits: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`
- Run verification commands before pushing
- Use `./codepush version` to verify builds work

## GitHub PR Reviews

### Reading comments
```bash
gh api repos/bitrise-io/bitrise-plugins-codepush-cli/pulls/<PR_NUMBER>/comments \
  --jq '.[] | "\(.id) | \(.path):\(.line // .original_line) | \(.body | split("\n")[0])"'
```

### Replying to comments
```bash
gh api repos/bitrise-io/bitrise-plugins-codepush-cli/pulls/<PR_NUMBER>/comments/<COMMENT_ID>/replies \
  -f body="Fixed."
```

### Updating PR title and body
```bash
gh api repos/bitrise-io/bitrise-plugins-codepush-cli/pulls/<PR_NUMBER> \
  --method PATCH -f title="the title" -F "body=@/tmp/pr-body.md"
```

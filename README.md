# CodePush CLI

Bitrise CLI plugin for managing CodePush OTA updates and SDK integration for mobile applications.

## Features

- **Push OTA updates** to React Native and mobile apps
- **Rollback** deployments to previous versions
- **Integrate** the CodePush SDK into mobile projects
- **Bitrise CI/CD** auto-detection and artifact export
- Works as **standalone CLI** or **Bitrise plugin**

## Quick Start

### As a Bitrise Plugin

```bash
# Install
bitrise plugin install --source https://github.com/bitrise-io/bitrise-plugins-codepush-cli.git

# Use
bitrise :codepush push ./dist/bundle.js
bitrise :codepush rollback
bitrise :codepush integrate
```

### As a Standalone CLI

Download the latest binary from [Releases](https://github.com/bitrise-io/bitrise-plugins-codepush-cli/releases), then:

```bash
./codepush push ./dist/bundle.js
./codepush rollback
./codepush integrate
./codepush version
```

## Commands

| Command | Description |
|---------|-------------|
| `push [bundle-path]` | Push an OTA update |
| `rollback` | Rollback to a previous release |
| `integrate` | Integrate CodePush SDK into a project |
| `version` | Print version information |

## Bitrise CI/CD Integration

When running in a Bitrise build, the plugin automatically:
- Detects the Bitrise environment
- Reads build metadata (build number, commit hash)
- Exports results to `$BITRISE_DEPLOY_DIR`

## Development

### Prerequisites

- Go 1.24+
- GoReleaser (for releases)

### Build

```bash
go build -o codepush ./cmd/codepush
./codepush version
```

### Test

```bash
go test ./...
go test -cover ./...
```

### Verify

```bash
go build ./cmd/codepush
go test ./...
go vet ./...
```

### Release

See the release process in [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT License. See [LICENSE](LICENSE) for details.

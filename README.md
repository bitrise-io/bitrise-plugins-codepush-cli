# CodePush CLI

Bitrise CLI plugin for managing CodePush OTA updates and SDK integration for mobile applications.

## Features

- **Bundle JavaScript** for React Native and Expo projects with auto-detection
- **Push OTA updates** to React Native and mobile apps
- **Rollback** deployments to previous versions
- **Integrate** the CodePush SDK into mobile projects
- **Hermes bytecode compilation** with automatic detection
- **Bitrise CI/CD** auto-detection and artifact export
- Works as **standalone CLI** or **Bitrise plugin**

## Quick Start

### As a Bitrise Plugin

```bash
# Install
bitrise plugin install --source https://github.com/bitrise-io/bitrise-plugins-codepush-cli.git

# Use
bitrise :codepush bundle --platform ios
bitrise :codepush push ./dist/bundle.js
bitrise :codepush push --bundle --platform ios
bitrise :codepush rollback
bitrise :codepush integrate
```

### As a Standalone CLI

Download the latest binary from [Releases](https://github.com/bitrise-io/bitrise-plugins-codepush-cli/releases), then:

```bash
./codepush bundle --platform ios
./codepush push ./dist/bundle.js
./codepush push --bundle --platform android
./codepush rollback
./codepush integrate
./codepush version
```

## Commands

| Command | Description |
|---------|-------------|
| `bundle` | Bundle JavaScript for an OTA update |
| `push [bundle-path]` | Push an OTA update |
| `rollback` | Rollback to a previous release |
| `integrate` | Integrate CodePush SDK into a project |
| `version` | Print version information |

## Bundling

The `bundle` command generates JavaScript bundles for React Native and Expo projects. It auto-detects the project type, entry file, Hermes configuration, and Metro config.

### Basic Usage

```bash
# Bundle for iOS
codepush bundle --platform ios

# Bundle for Android
codepush bundle --platform android

# Bundle and push in one step
codepush push --bundle --platform ios
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--platform` | (required) | `ios` or `android` |
| `--entry-file` | auto-detect | Path to entry JS file |
| `--output-dir` | `./codepush-bundle` | Output directory |
| `--bundle-name` | platform default | Custom bundle filename |
| `--dev` | `false` | Development mode |
| `--sourcemap` | `true` | Generate source maps |
| `--hermes` | `auto` | Hermes compilation: `auto`, `on`, `off` |
| `--extra-bundler-option` | none | Pass-through flags to bundler (repeatable) |
| `--project-dir` | CWD | Project root directory |
| `--config` | auto-detect | Metro config file path |

### Auto-Detection

The CLI automatically detects:

- **Project type**: React Native or Expo (from `package.json` dependencies)
- **Entry file**: `index.<platform>.js`, `index.js`, or `package.json` main field
- **Hermes**: From `build.gradle` (Android) or `Podfile` (iOS)
- **Metro config**: `metro.config.js` or `metro.config.ts`

### Examples

```bash
# Override Hermes detection
codepush bundle --platform android --hermes=off

# Custom output directory
codepush bundle --platform ios --output-dir ./my-bundle

# Development build
codepush bundle --platform ios --dev

# Pass extra options to the bundler
codepush bundle --platform ios --extra-bundler-option="--reset-cache"
```

## Pushing Updates

The `push` command packages a bundle directory and uploads it to the Bitrise CodePush API.

### Basic Usage

```bash
# Push an existing bundle directory
codepush push ./codepush-bundle --app-id <UUID> --deployment Production --token <TOKEN> --app-version 1.0.0

# Bundle and push in one step
codepush push --bundle --platform ios --app-id <UUID> --deployment Staging --app-version 1.0.0
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--app-id` | env: `CODEPUSH_APP_ID` | Connected app UUID |
| `--deployment` | env: `CODEPUSH_DEPLOYMENT` | Deployment name or UUID |
| `--token` | env: `BITRISE_API_TOKEN` | Bitrise API token |
| `--app-version` | (required) | Target app version (e.g. 1.0.0) |
| `--description` | "" | Update description |
| `--mandatory` | `false` | Mark update as mandatory |
| `--rollout` | `100` | Rollout percentage (1-100) |
| `--disabled` | `false` | Disable update after upload |
| `--api-url` | Bitrise API | API base URL |
| `--bundle` | `false` | Bundle JavaScript before pushing |

### Environment Variables

Flags can be set via environment variables (flag takes precedence):

| Variable | Flag |
|----------|------|
| `CODEPUSH_APP_ID` | `--app-id` |
| `CODEPUSH_DEPLOYMENT` | `--deployment` |
| `BITRISE_API_TOKEN` | `--token` |

### Examples

```bash
# Push with rollout
codepush push ./codepush-bundle --app-id $APP_ID --deployment Production --app-version 1.0.0 --rollout 25

# Push mandatory update
codepush push ./codepush-bundle --app-id $APP_ID --deployment Staging --app-version 1.0.0 --mandatory

# Push with description
codepush push ./codepush-bundle --app-id $APP_ID --deployment Staging --app-version 1.0.0 --description "Bug fix for login screen"
```

## Bitrise CI/CD Integration

When running in a Bitrise build, the plugin automatically:
- Detects the Bitrise environment
- Reads build metadata (build number, commit hash)
- Exports results to `$BITRISE_DEPLOY_DIR`
- Exports `codepush-bundle-summary.json` after bundling
- Exports `codepush-push-summary.json` after pushing

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

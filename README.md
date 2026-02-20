# CodePush CLI

[![Build Status](https://app.bitrise.io/app/7b3ab048-138e-4d17-899c-4ea776b5711f/status.svg?token=-eUGFSXpQwDpmLX18KJUeA&branch=main)](https://app.bitrise.io/app/7b3ab048-138e-4d17-899c-4ea776b5711f)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bitrise-io/bitrise-plugins-codepush-cli)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A CLI tool for managing over-the-air (OTA) updates for React Native and Expo mobile applications using the Bitrise CodePush service. Works as a standalone CLI or as a [Bitrise CLI plugin](https://devcenter.bitrise.io/en/bitrise-cli/bitrise-cli-plugins.html).

## What is CodePush?

CodePush lets you push JavaScript and asset updates directly to users' devices without going through the app store review process. When your React Native or Expo app starts, the CodePush SDK checks for available updates and downloads them in the background.

This CLI manages the server side of that workflow: bundling your JavaScript code, uploading it to the Bitrise CodePush service, and managing deployments (Staging, Production, etc.) and their release history.

**Typical flow:** Bundle your JS code, push it to a Staging deployment, verify on test devices, then promote to Production.

## Installation

### As a Bitrise Plugin

```bash
bitrise plugin install --source https://github.com/bitrise-io/bitrise-plugins-codepush-cli.git
```

Once installed, prefix all commands with `bitrise :codepush`:

```bash
bitrise :codepush push --bundle --platform ios --app-version 1.0.0
```

### As a Standalone CLI

Download the latest binary for your platform from [Releases](https://github.com/bitrise-io/bitrise-plugins-codepush-cli/releases):

| Platform | Binary |
|----------|--------|
| macOS (Apple Silicon) | `codepush-Darwin-arm64` |
| macOS (Intel) | `codepush-Darwin-x86_64` |
| Linux (x86_64) | `codepush-Linux-x86_64` |

```bash
chmod +x codepush-Darwin-arm64
mv codepush-Darwin-arm64 /usr/local/bin/codepush
codepush version
```

## Quick Start

Authenticate, bundle, and push your first OTA update:

```bash
# 1. Store your Bitrise API token
codepush auth login --token <YOUR_BITRISE_API_TOKEN>

# 2. Bundle your JavaScript for iOS
codepush bundle --platform ios

# 3. Push the bundle to Staging
codepush push ./codepush-bundle \
  --app-id <APP_UUID> \
  --deployment Staging \
  --app-version 1.0.0

# Or bundle and push in one step
codepush push --bundle --platform ios \
  --app-id <APP_UUID> \
  --deployment Staging \
  --app-version 1.0.0
```

For Bitrise CI workflows, set `BITRISE_API_TOKEN`, `CODEPUSH_APP_ID`, and `CODEPUSH_DEPLOYMENT` as environment variables and the CLI resolves them automatically:

```bash
bitrise :codepush push --bundle --platform ios --app-version 1.0.0
```

## Authentication

Commands that interact with the Bitrise API require an API token. Tokens are resolved in this order:

1. `BITRISE_API_TOKEN` environment variable (recommended for CI)
2. Stored config file from `codepush auth login` (recommended for local development)

Generate a personal access token at: https://app.bitrise.io/me/account/security

```bash
# Store token locally (interactive or via flag)
codepush auth login
codepush auth login --token <TOKEN>

# Remove stored token
codepush auth revoke
```

The token is stored in the user config directory with restricted permissions (0600):
- macOS: `~/Library/Application Support/codepush/config.json`
- Linux: `~/.config/codepush/config.json`

## Commands

### Global Flags

| Flag | Description |
|------|-------------|
| `--app-id` | Connected app UUID (env: `CODEPUSH_APP_ID`) |
| `--json` | Output results as JSON to stdout |

### Release Management

| Command | Description |
|---------|-------------|
| `bundle` | Bundle JavaScript for an OTA update |
| `push [bundle-path]` | Push an OTA update |
| `rollback` | Rollback to a previous release |
| `promote` | Promote a release from one deployment to another |
| `patch` | Update metadata on an existing release |

### Deployment Management

| Command | Description |
|---------|-------------|
| `deployment list` | List all deployments |
| `deployment add <name>` | Create a new deployment |
| `deployment info <deployment>` | Show deployment details and latest release |
| `deployment rename <deployment>` | Rename a deployment (`--name`) |
| `deployment remove <deployment>` | Delete a deployment (`--yes` to confirm) |
| `deployment history <deployment>` | Show release history (`--limit`, default 10) |
| `deployment clear <deployment>` | Delete all packages from a deployment (`--yes` to confirm) |

### Package Management

| Command | Description |
|---------|-------------|
| `package info <deployment>` | Show package details (`--label` for specific version) |
| `package status <deployment>` | Show package processing status |
| `package remove <deployment>` | Delete a package (`--label` required, `--yes` to confirm) |

### Authentication

| Command | Description |
|---------|-------------|
| `auth login` | Store a Bitrise API token locally |
| `auth revoke` | Remove the stored API token |

### Other

| Command | Description |
|---------|-------------|
| `version` | Print version information |

Run `codepush <command> --help` for detailed flags and usage of any command.

## Bundling

The `bundle` command generates JavaScript bundles for React Native and Expo projects. It auto-detects the project type, entry file, Hermes configuration, and Metro config.

```bash
codepush bundle --platform ios
codepush bundle --platform android
```

### Bundle Flags

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
- **Hermes**: From `build.gradle` (Android) or `Podfile` (iOS); defaults to enabled for React Native >= 0.70
- **Metro config**: `metro.config.js` or `metro.config.ts`

## Pushing Updates

```bash
# Push a pre-built bundle
codepush push ./codepush-bundle \
  --app-id <APP_UUID> --deployment Staging --app-version 1.0.0

# Bundle and push in one step
codepush push --bundle --platform ios \
  --app-id <APP_UUID> --deployment Staging --app-version 1.0.0
```

### Push Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--deployment` | env: `CODEPUSH_DEPLOYMENT` | Deployment name or UUID |
| `--app-version` | (required) | Target app version (e.g. 1.0.0) |
| `--description` | `""` | Update description |
| `--mandatory` | `false` | Mark update as mandatory |
| `--rollout` | `100` | Rollout percentage (1-100) |
| `--disabled` | `false` | Disable update after upload |
| `--bundle` | `false` | Bundle JavaScript before pushing |
| `--platform` | | Target platform (required with `--bundle`) |
| `--hermes` | `auto` | Hermes compilation (with `--bundle`) |
| `--output-dir` | `./codepush-bundle` | Bundle output directory (with `--bundle`) |
| `--project-dir` | CWD | Project root (with `--bundle`) |

## Promoting and Patching

### Promote

Copy a release from one deployment to another. Commonly used to promote a tested Staging release to Production.

```bash
codepush promote \
  --source-deployment Staging \
  --destination-deployment Production \
  --app-id <APP_UUID>

# Override metadata during promotion
codepush promote \
  --source-deployment Staging \
  --destination-deployment Production \
  --app-id <APP_UUID> \
  --rollout 25 --description "Gradual rollout"
```

**Promote flags:** `--source-deployment`, `--destination-deployment`, `--label`, `--app-version`, `--description`, `--mandatory`, `--disabled`, `--rollout`

### Patch

Update metadata on an existing release without re-deploying the code.

```bash
# Increase rollout on the latest release
codepush patch --deployment Production --rollout 50 --app-id <APP_UUID>

# Patch a specific release
codepush patch --deployment Production --label v5 --mandatory true --app-id <APP_UUID>
```

**Patch flags:** `--deployment`, `--label`, `--rollout`, `--mandatory`, `--disabled`, `--description`, `--app-version`

## Rollback

Rollback creates a new release that mirrors a previous version.

```bash
# Rollback to the immediately previous release
codepush rollback --deployment Production --app-id <APP_UUID>

# Rollback to a specific release
codepush rollback --deployment Production --target-release v3 --app-id <APP_UUID>
```

**Rollback flags:** `--deployment`, `--target-release`

## Deployment Management

```bash
# List all deployments
codepush deployment list --app-id <APP_UUID>

# Create a new deployment
codepush deployment add Beta --app-id <APP_UUID>

# View deployment details and latest release
codepush deployment info Staging --app-id <APP_UUID>

# View release history (default: last 10)
codepush deployment history Staging --app-id <APP_UUID>
codepush deployment history Staging --limit 25 --app-id <APP_UUID>

# Rename a deployment
codepush deployment rename OldName --name NewName --app-id <APP_UUID>

# Delete a deployment (destructive, requires --yes in CI)
codepush deployment remove Beta --app-id <APP_UUID> --yes

# Clear all releases from a deployment (destructive, requires --yes in CI)
codepush deployment clear Staging --app-id <APP_UUID> --yes
```

Destructive operations (`remove`, `clear`) require `--yes` to skip the interactive confirmation prompt. In CI environments, always pass `--yes`.

## Package Management

```bash
# View details of the latest package
codepush package info Staging --app-id <APP_UUID>

# View a specific package by label
codepush package info Staging --label v5 --app-id <APP_UUID>

# Check processing status (useful after push)
codepush package status Staging --app-id <APP_UUID>

# Delete a specific package (destructive)
codepush package remove Staging --label v3 --app-id <APP_UUID> --yes
```

## Workflow Examples

### Full Release Lifecycle

```bash
# 1. Authenticate
codepush auth login --token $BITRISE_API_TOKEN

# 2. Bundle the JavaScript
codepush bundle --platform ios

# 3. Push to Staging with limited rollout
codepush push ./codepush-bundle \
  --app-id $APP_ID --deployment Staging \
  --app-version 1.2.0 --rollout 10 --description "Fix login crash"

# 4. Check processing status
codepush package status Staging --app-id $APP_ID

# 5. Increase rollout after verifying on test devices
codepush patch --deployment Staging --rollout 100 --app-id $APP_ID

# 6. Promote to Production
codepush promote \
  --source-deployment Staging \
  --destination-deployment Production \
  --app-id $APP_ID --rollout 25

# 7. If something goes wrong, rollback
codepush rollback --deployment Production --app-id $APP_ID
```

### Bitrise CI Pipeline

Set these environment variables in your Bitrise workflow: `BITRISE_API_TOKEN`, `CODEPUSH_APP_ID`, `CODEPUSH_DEPLOYMENT`.

```bash
bitrise :codepush push --bundle --platform ios --app-version $APP_VERSION
```

The CLI automatically detects the Bitrise environment, attaches build metadata (build number, commit hash), and exports summary files to `$BITRISE_DEPLOY_DIR`.

## JSON Output

Pass `--json` to any command to get machine-readable JSON output on stdout. Human-readable output always goes to stderr, so JSON output is clean for piping.

```bash
# Get push result as JSON
codepush push ./codepush-bundle --app-id $APP_ID \
  --deployment Staging --app-version 1.0.0 --json

# List deployments as JSON
codepush deployment list --app-id $APP_ID --json

# Parse with jq
codepush package info Staging --app-id $APP_ID --json | jq '.app_version'
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BITRISE_API_TOKEN` | API token for authentication |
| `CODEPUSH_APP_ID` | Default connected app UUID (used when `--app-id` is not set) |
| `CODEPUSH_DEPLOYMENT` | Default deployment name or UUID (used when `--deployment` is not set) |
| `NO_COLOR` | Disable colored terminal output |

### Bitrise CI Variables (read automatically)

| Variable | Description |
|----------|-------------|
| `BITRISE_BUILD_NUMBER` | Attached to push metadata |
| `BITRISE_DEPLOY_DIR` | Directory for summary file export |
| `GIT_CLONE_COMMIT_HASH` | Attached to push metadata |

### Exported Variables (Bitrise CI)

After a successful push, rollback, promote, or patch, the CLI exports these via `envman` for downstream Bitrise steps:

| Variable | Description |
|----------|-------------|
| `CODEPUSH_PACKAGE_ID` | ID of the created or modified package |
| `CODEPUSH_APP_VERSION` | App version of the release |
| `CODEPUSH_LABEL` | Release label (patch command only) |

## Bitrise CI Integration

When running inside a Bitrise build (detected via `BITRISE_BUILD_NUMBER` or `BITRISE_DEPLOY_DIR`), the CLI automatically:

- Attaches build number and commit hash to push metadata
- Exports `codepush-bundle-summary.json` after bundling
- Exports `codepush-push-summary.json` after pushing
- Exports `codepush-patch-summary.json` after patching
- Exports environment variables via `envman` for downstream steps
- Disables interactive prompts and spinners

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, build commands, testing, and the release process.

## License

MIT License. See [LICENSE](LICENSE) for details.

# CodePush CLI

[![Build Status](https://app.bitrise.io/app/7b3ab048-138e-4d17-899c-4ea776b5711f/status.svg?token=-eUGFSXpQwDpmLX18KJUeA&branch=main)](https://app.bitrise.io/app/7b3ab048-138e-4d17-899c-4ea776b5711f)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A Bitrise CLI plugin for managing over-the-air (OTA) updates for React Native and Expo mobile applications using the Bitrise CodePush service. Can also be used as a standalone CLI tool outside of Bitrise.

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

Manage the plugin lifecycle with standard Bitrise CLI commands:

```bash
bitrise plugin list                 # confirm installation
bitrise plugin update codepush      # upgrade to latest version
bitrise plugin uninstall codepush   # remove the plugin
```

For standalone use outside Bitrise, see [Using as a Standalone CLI](#using-as-a-standalone-cli).

## Prerequisites

These requirements apply in both plugin and standalone mode:

- **[Node.js](https://nodejs.org/)** — required for bundling; version must satisfy your project's requirements.
- **[React Native](https://reactnative.dev/docs/environment-setup)** or **[Expo](https://docs.expo.dev/get-started/installation/)** — must be present in your project's `node_modules` (the CLI invokes `npx react-native bundle` or `npx expo export`).

Additionally, for plugin mode:

- **[Bitrise CLI](https://devcenter.bitrise.io/en/bitrise-cli.html)** >= 1.3.0

## Quick Start

> These examples use the Bitrise plugin syntax (`bitrise :codepush ...`). If using the standalone binary, omit the `bitrise :` prefix.

Authenticate, configure your project, and push your first OTA update:

```bash
# 1. Store your Bitrise API token
#    Local dev only: stores token in user config.
#    In Bitrise CI: set BITRISE_API_TOKEN as a workflow Secret instead.
bitrise :codepush auth login --token <YOUR_BITRISE_API_TOKEN>

# 2. Initialize project config (prompts for app ID, saves for all future commands)
bitrise :codepush init

# 3. Bundle and push to Staging in one step
bitrise :codepush push --bundle --platform ios \
  --deployment Staging \
  --app-version 1.0.0
```

In Bitrise CI workflows, set `BITRISE_API_TOKEN`, `CODEPUSH_APP_ID`, and `CODEPUSH_DEPLOYMENT` as environment variables and the CLI resolves them automatically:

```bash
bitrise :codepush push --bundle --platform ios --app-version 1.0.0
```

## Authentication

Commands that interact with the Bitrise API require an API token. Tokens are resolved in this order:

1. `BITRISE_API_TOKEN` environment variable (recommended for CI — Bitrise or any other)
2. Stored config file from `bitrise :codepush auth login` (recommended for local development)

Generate a personal access token at: https://app.bitrise.io/me/account/security

```bash
# Store token locally (interactive or via flag)
bitrise :codepush auth login
bitrise :codepush auth login --token <TOKEN>    # or: -t <TOKEN>

# Remove stored token
bitrise :codepush auth revoke
```

The token is stored in the user config directory with restricted permissions (0600):
- macOS: `~/Library/Application Support/codepush/config.json`
- Linux: `~/.config/codepush/config.json`

## Project Configuration

Running `bitrise :codepush init` creates a `.codepush.json` file in the current directory that stores your app ID:

```bash
bitrise :codepush init
```

The command prompts for your app ID interactively. You can also pass it via the global `--app-id` flag or `CODEPUSH_APP_ID` environment variable.

This file is safe to commit to version control so your team shares the same configuration. Once initialized, you no longer need to pass `--app-id` on every command.

The app ID is resolved in this order:

1. `--app-id` flag (highest priority)
2. `CODEPUSH_APP_ID` environment variable
3. `.codepush.json` file in current directory

Use `--force` (`-f`) to overwrite an existing `.codepush.json`.

### Custom Server URL

To target a different environment (e.g. staging), set the server base URL:

```bash
# Via flag
bitrise :codepush push --server-url https://api.staging.bitrise.io

# Via environment variable
export CODEPUSH_SERVER_URL=https://api.staging.bitrise.io

# Via .codepush.json (saved during init)
bitrise :codepush init --server-url https://api.staging.bitrise.io
```

The server URL is resolved in this order:

1. `--server-url` flag (highest priority)
2. `CODEPUSH_SERVER_URL` environment variable
3. `server_url` field in `.codepush.json`
4. Default: `https://api.bitrise.io`

## Commands

> Commands are shown without a prefix. Invoke them as `bitrise :codepush <command>` (plugin) or `codepush <command>` (standalone binary).

### Global Flags

| Flag | Description |
|------|-------------|
| `--app-id` | Release management app UUID (env: `CODEPUSH_APP_ID`) |
| `--json`, `-j` | Output results as JSON to stdout |
| `--server-url` | API server base URL (env: `CODEPUSH_SERVER_URL`) |

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
| `deployment list` | List all deployments (`--display-keys / -k` to include key column) |
| `deployment add <name>` | Create a new deployment (`--key / -k` for a custom deployment key) |
| `deployment info <deployment>` | Show deployment details and latest release |
| `deployment rename <deployment>` | Rename a deployment (`--name`, `-n`) |
| `deployment remove <deployment>` | Delete a deployment (`--yes`/`-y` to confirm) |
| `deployment history <deployment>` | Show release history (`--limit`/`-n`, default 10; `--display-author`/`-a` to include author column) |
| `deployment clear <deployment>` | Delete all updates from a deployment (`--yes`/`-y` to confirm) |

### Update Management

| Command | Description |
|---------|-------------|
| `update info <deployment>` | Show update details (`--label`/`-l` for specific version) |
| `update status <deployment>` | Show update processing status (`--label`/`-l`) |
| `update remove <deployment>` | Delete an update (`--label`/`-l` required, `--yes`/`-y` to confirm) |

### Setup

| Command | Description |
|---------|-------------|
| `init` | Initialize project config (`.codepush.json`) with app ID |
| `auth login` | Store a Bitrise API token locally |
| `auth revoke` | Remove the stored API token |

### Developer Tools

| Command | Description |
|---------|-------------|
| `debug <platform>` | Stream CodePush log output from a connected device or simulator (`android` or `ios`) |

### Other

| Command | Description |
|---------|-------------|
| `version` | Print version information |

Run `bitrise :codepush <command> --help` for detailed flags and usage of any command.

## Bundling

The `bundle` command generates JavaScript bundles for React Native and Expo projects. It auto-detects the project type, entry file, Hermes configuration, and Metro config.

```bash
bitrise :codepush bundle --platform ios
bitrise :codepush bundle --platform android
```

The `bundle` command produces a **directory** (not a zip file). This directory is what you pass to `push` as `[bundle-path]`. The CLI zips it internally before upload — you do not need to zip it manually.

### Bundle Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--platform`, `-p` | (required) | `ios` or `android` |
| `--entry-file`, `-e` | auto-detect | Path to entry JS file |
| `--output-dir`, `-o` | `./codepush-bundle` | Output directory |
| `--bundle-name`, `-b` | platform default | Custom bundle filename |
| `--dev` | `false` | Development mode |
| `--sourcemap` | `true` | Generate source maps |
| `--sourcemap-output, -s` | | Override sourcemap output path (implies `--sourcemap`). Not supported for Expo projects — Expo always writes sourcemaps next to the bundle; the path cannot be overridden. |
| `--hermes` | `auto` | Hermes compilation: `auto`, `on`, `off` |
| `--extra-bundler-option` | none | Pass-through flags to bundler/Metro (repeatable) |
| `--extra-hermes-flag` | none | Pass additional flags to `hermesc` (repeatable; no shorthand) |
| `--project-dir` | CWD | Project root directory |
| `--config`, `-c` | auto-detect | Metro config file path |
| `--gradle-file, -g` | auto-detect | Override `build.gradle` path for Android Hermes detection |
| `--pod-file` | auto-detect | Override `Podfile` path for iOS Hermes detection |

### Auto-Detection

The CLI automatically detects:

- **Project type**: React Native or Expo (from `package.json` dependencies)
- **Entry file**: `index.<platform>.js`, `index.js`, or `package.json` main field
- **Hermes**: From `build.gradle` (Android) or `Podfile` (iOS); defaults to enabled for React Native >= 0.70. Override these paths with `--gradle-file` / `--pod-file` when your project layout differs from the standard.
- **Metro config**: `metro.config.js` or `metro.config.ts`

## Pushing Updates

The `[bundle-path]` argument must be a **directory** — the output of `bitrise :codepush bundle`. The CLI zips it internally before upload.

```bash
# Push a pre-built bundle directory
bitrise :codepush push ./codepush-bundle \
  --app-id <APP_UUID> --deployment Staging --app-version 1.0.0

# Bundle and push in one step
bitrise :codepush push --bundle --platform ios \
  --app-id <APP_UUID> --deployment Staging --app-version 1.0.0
```

### Push Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--deployment`, `-d` | env: `CODEPUSH_DEPLOYMENT` | Deployment name or UUID |
| `--app-version`, `-t` | (required) | Target app version (e.g. 1.0.0) |
| `--description` | `""` | Update description |
| `--mandatory`, `-m` | `false` | Mark update as mandatory |
| `--rollout`, `-r` | `100` | Rollout percentage (0-100) |
| `--disabled`, `-x` | `false` | Disable update after upload |
| `--bundle` | `false` | Bundle JavaScript before pushing |
| `--platform`, `-p` | | Target platform (required with `--bundle`) |
| `--hermes` | `auto` | Hermes compilation (with `--bundle`) |
| `--output-dir`, `-o` | `./codepush-bundle` | Bundle output directory (with `--bundle`) |
| `--project-dir` | CWD | Project root (with `--bundle`) |
| `--gradle-file`, `-g` | auto-detect | Override `build.gradle` path for Android Hermes detection (with `--bundle`) |
| `--pod-file` | auto-detect | Override `Podfile` path for iOS Hermes detection (with `--bundle`) |

## Promoting and Patching

### Promote

Copy a release from one deployment to another. Commonly used to promote a tested Staging release to Production.

```bash
bitrise :codepush promote \
  --source-deployment Staging \
  --destination-deployment Production \
  --app-id <APP_UUID>

# Override metadata during promotion
bitrise :codepush promote \
  --source-deployment Staging \
  --destination-deployment Production \
  --app-id <APP_UUID> \
  --rollout 25 --description "Gradual rollout"
```

**Promote flags:** `--source-deployment` (`-s`), `--destination-deployment` (`-d`), `--label` (`-l`), `--app-version` (`-t`), `--description`, `--mandatory` (`-m`), `--disabled` (`-x`), `--rollout` (`-r`), `--no-duplicate-release-error`

Pass `--no-duplicate-release-error` to exit 0 with a warning instead of an error when the target deployment already contains a release with identical content. Useful in CI pipelines where re-promoting after a partial failure should be a no-op.

### Patch

Update metadata on an existing release without re-deploying the code.

```bash
# Increase rollout on the latest release
bitrise :codepush patch --deployment Production --rollout 50 --app-id <APP_UUID>

# Patch a specific release
bitrise :codepush patch --deployment Production --label v5 --mandatory true --app-id <APP_UUID>
```

**Patch flags:** `--deployment` (`-d`), `--label` (`-l`), `--rollout` (`-r`), `--mandatory` (`-m`), `--disabled` (`-x`), `--description`, `--app-version` (`-t`)

## Rollback

Rollback creates a new release that mirrors a previous version.

```bash
# Rollback to the immediately previous release
bitrise :codepush rollback --deployment Production --app-id <APP_UUID>

# Rollback to a specific release
bitrise :codepush rollback --deployment Production --target-release v3 --app-id <APP_UUID>
```

**Rollback flags:** `--deployment` (`-d`), `--target-release` (`-r`)

## Deployment Management

```bash
# List all deployments
bitrise :codepush deployment list --app-id <APP_UUID>
bitrise :codepush deployment list --display-keys --app-id <APP_UUID>

# Create a new deployment
bitrise :codepush deployment add Beta --app-id <APP_UUID>
bitrise :codepush deployment add Beta --key my-custom-key --app-id <APP_UUID>

# View deployment details and latest release
bitrise :codepush deployment info Staging --app-id <APP_UUID>

# View release history (default: last 10)
bitrise :codepush deployment history Staging --app-id <APP_UUID>
bitrise :codepush deployment history Staging --limit 25 --app-id <APP_UUID>
bitrise :codepush deployment history Staging --display-author --app-id <APP_UUID>

# Rename a deployment
bitrise :codepush deployment rename OldName --name NewName --app-id <APP_UUID>

# Delete a deployment (destructive, requires --yes in CI)
bitrise :codepush deployment remove Beta --app-id <APP_UUID> --yes

# Clear all releases from a deployment (destructive, requires --yes in CI)
bitrise :codepush deployment clear Staging --app-id <APP_UUID> --yes
```

Destructive operations (`remove`, `clear`) require `--yes` to skip the interactive confirmation prompt. In CI environments, always pass `--yes`.

## Update Management

```bash
# View details of the latest update
bitrise :codepush update info Staging --app-id <APP_UUID>

# View a specific update by label
bitrise :codepush update info Staging --label v5 --app-id <APP_UUID>

# Check processing status (useful after push)
bitrise :codepush update status Staging --app-id <APP_UUID>

# Delete a specific update (destructive)
bitrise :codepush update remove Staging --label v3 --app-id <APP_UUID> --yes
```

## Debugging

Stream real-time CodePush log output from a connected Android device or iOS simulator to help diagnose update delivery and installation issues.

```bash
# Android: stream CodePush logs (requires adb on PATH)
bitrise :codepush debug android

# iOS: stream CodePush logs (requires xcrun on PATH)
bitrise :codepush debug ios
```

Android uses `adb logcat` with a `CodePush:V *:S` tag filter (logcat-layer filtering). Each line is prefixed with a timestamp (`[HH:mm:ss.SSS]`).

iOS uses `xcrun simctl spawn booted log stream` with a predicate filter. Lines are printed as-is since the unified log format already includes native timestamps.

Press Ctrl-C to stop streaming.

## Workflow Examples

### Full Release Lifecycle

```bash
# 1. Authenticate
bitrise :codepush auth login --token $BITRISE_API_TOKEN

# 2. Bundle the JavaScript
bitrise :codepush bundle --platform ios

# 3. Push to Staging with limited rollout
bitrise :codepush push ./codepush-bundle \
  --app-id $APP_ID --deployment Staging \
  --app-version 1.2.0 --rollout 10 --description "Fix login crash"

# 4. Check processing status
bitrise :codepush update status Staging --app-id $APP_ID

# 5. Increase rollout after verifying on test devices
bitrise :codepush patch --deployment Staging --rollout 100 --app-id $APP_ID

# 6. Promote to Production
bitrise :codepush promote \
  --source-deployment Staging \
  --destination-deployment Production \
  --app-id $APP_ID --rollout 25

# 7. If something goes wrong, rollback
bitrise :codepush rollback --deployment Production --app-id $APP_ID
```

### Bitrise CI Pipeline

Set these environment variables in your Bitrise workflow: `BITRISE_API_TOKEN`, `CODEPUSH_APP_ID`, `CODEPUSH_DEPLOYMENT`.

```bash
bitrise :codepush push --bundle --platform ios --app-version $APP_VERSION
```

The CLI automatically detects the Bitrise environment, attaches build metadata (build number, commit hash), and exports summary files to `$BITRISE_DEPLOY_DIR`.

### Expo Workflow

Expo is auto-detected from `package.json` — no extra flags are needed. The CLI uses `npx expo export` under the hood instead of `react-native bundle`. All other flags (deployment, app-version, rollout, etc.) behave identically.

```bash
bitrise :codepush push --bundle --platform ios \
  --deployment Staging \
  --app-version 1.0.0
```

Note: `--sourcemap-output` is not supported for Expo projects. Expo always writes sourcemaps next to the bundle automatically. The `--sourcemap` flag (enable/disable sourcemaps) is still supported.

## JSON Output

Pass `--json` to any command to get machine-readable JSON output on stdout. Human-readable output always goes to stderr, so JSON output is clean for piping.

```bash
# Get push result as JSON
bitrise :codepush push ./codepush-bundle --app-id $APP_ID \
  --deployment Staging --app-version 1.0.0 --json

# List deployments as JSON
bitrise :codepush deployment list --app-id $APP_ID --json

# Parse with jq
bitrise :codepush update info Staging --app-id $APP_ID --json | jq '.app_version'
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (authentication failure, API error, validation error, etc.) |

A non-zero exit code from any command means the operation failed. Check stderr for the error message.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BITRISE_API_TOKEN` | API token for authentication |
| `CODEPUSH_APP_ID` | Default release management app UUID (used when `--app-id` is not set) |
| `CODEPUSH_DEPLOYMENT` | Default deployment name or UUID (used when `--deployment` is not set) |
| `CODEPUSH_SERVER_URL` | API server base URL (used when `--server-url` is not set) |
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
| `CODEPUSH_UPDATE_ID` | ID of the created or modified update |
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

## Using as a Standalone CLI

When using outside a Bitrise environment, download the binary directly from [Releases](https://github.com/bitrise-io/bitrise-plugins-codepush-cli/releases):

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

All commands work identically — replace `bitrise :codepush` with `codepush` in any example in this document:

```bash
codepush push --bundle --platform ios --deployment Staging --app-version 1.0.0
```

**Differences from plugin mode:**

- `BITRISE_BUILD_NUMBER`, `BITRISE_DEPLOY_DIR`, and `GIT_CLONE_COMMIT_HASH` are not auto-populated.
- `envman` exports (`CODEPUSH_UPDATE_ID`, `CODEPUSH_APP_VERSION`, `CODEPUSH_LABEL`) are not available for downstream steps.
- Authentication: use `codepush auth login` to store credentials locally, or set `BITRISE_API_TOKEN` as an environment variable — both work in standalone mode.

## Troubleshooting

**Authentication errors** (`token not found` / `401 Unauthorized`): Set `BITRISE_API_TOKEN` as an environment variable, or run `bitrise :codepush auth login` to store a token locally.

**App not initialized** (`app ID is required`): Run `bitrise :codepush init`, pass `--app-id`, or set `CODEPUSH_APP_ID`.

**Bundle path is not a directory**: The `push` command requires a directory path, not a zip file or individual file. Run `bitrise :codepush bundle` first, then pass the output directory to `push`.

**Bundle detection failures**: If auto-detection fails, specify flags explicitly: `--entry-file`, `--config` (Metro), `--gradle-file` (Android Hermes), `--pod-file` (iOS Hermes).

**`adb: command not found`** (`debug android`): Install [Android platform tools](https://developer.android.com/tools/releases/platform-tools) and ensure `adb` is on `PATH`.

**`xcrun: error`** (`debug ios`): Install Xcode Command Line Tools: `xcode-select --install`.

**Expo `--sourcemap-output` rejected**: Expo always writes sourcemaps next to the bundle; the output path cannot be overridden. Remove the `--sourcemap-output` flag. The `--sourcemap` flag (enable/disable sourcemaps) is still supported.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, build commands, testing, and the release process.

## License

MIT License. See [LICENSE](LICENSE) for details.

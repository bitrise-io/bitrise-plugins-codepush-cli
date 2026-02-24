---
name: bump-version-and-release
description: Bump version in code and plugin config, create a release PR, then tag to trigger the Bitrise release pipeline.
disable-model-invocation: true
allowed-tools: Bash(go *), Bash(git *), Bash(gh *), Bash(golangci-lint *), Read, Grep, Edit
---

# Bump Version and Release

Current version in main.go: !`grep 'version = ' cmd/codepush/version.go | head -1 | sed 's/.*"\(.*\)".*/\1/'`
New version: $ARGUMENTS

## Instructions

Follow these steps in order. Stop and report if any step fails.

### Phase 1: Pre-flight checks

1. Validate that `$ARGUMENTS` is a valid semantic version (X.Y.Z or X.Y.Z-suffix):
   ```bash
   echo "$ARGUMENTS" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'
   ```
2. Verify git working directory is clean: `git status --porcelain` should be empty
3. Verify on `main` branch: `git branch --show-current` should output `main`
4. Pull latest: `git pull origin main`
5. Run tests: `go test ./...`
6. Run linter: `golangci-lint run`

### Phase 2: Version bump

1. Update `cmd/codepush/version.go`: change `version = "OLD"` to `version = "NEW"`
2. Update `bitrise-plugin.yml`: replace the old version in all three download URLs with the new version
3. Verify both files have the new version:
   ```bash
   grep 'version = ' cmd/codepush/version.go
   grep 'download/' bitrise-plugin.yml
   ```
4. Run `go build ./cmd/codepush` to verify the build still works

### Phase 3: Create release PR

1. Create and switch to branch:
   ```bash
   git checkout -b release/$ARGUMENTS
   ```
2. Stage and commit:
   ```bash
   git add cmd/codepush/version.go bitrise-plugin.yml
   git commit -m "chore: bump version to $ARGUMENTS"
   ```
3. Push and create PR:
   ```bash
   git push -u origin release/$ARGUMENTS
   gh pr create --title "chore: bump version to $ARGUMENTS" --body "Automated version bump to $ARGUMENTS.

   After this PR is merged, the release tag will be created to trigger the Bitrise release pipeline."
   ```
4. Print the PR URL for the user.

### Phase 4: Tag and release (after PR merge)

Ask the user to confirm the PR has been merged before proceeding.

1. Switch back to main and pull:
   ```bash
   git checkout main
   git pull origin main
   ```
2. Create and push the tag:
   ```bash
   git tag -a $ARGUMENTS -m "Release $ARGUMENTS"
   git push origin $ARGUMENTS
   ```
3. Confirm: the tag push triggers the Bitrise `pipeline_release` automatically. Print the Bitrise dashboard link or instruct the user to check Bitrise for the release build.

### Phase 5: Verify

1. After the Bitrise release pipeline completes, verify the GitHub release:
   ```bash
   gh release view $ARGUMENTS
   ```
2. Check that all expected assets are present:
   - `codepush-Darwin-arm64`
   - `codepush-Darwin-x86_64`
   - `codepush-Linux-x86_64`
   - `checksums.txt`

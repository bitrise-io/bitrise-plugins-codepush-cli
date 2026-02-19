// Package bitrise provides integration with the Bitrise CI/CD environment.
package bitrise

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// BuildMetadata contains information about the current Bitrise build.
type BuildMetadata struct {
	DeployDir   string
	BuildNumber string
	CommitHash  string
}

// IsBitriseEnvironment returns true if running inside a Bitrise CI build.
func IsBitriseEnvironment() bool {
	return os.Getenv("BITRISE_BUILD_NUMBER") != "" || os.Getenv("BITRISE_DEPLOY_DIR") != ""
}

// GetBuildMetadata reads Bitrise environment variables and returns build context.
func GetBuildMetadata() BuildMetadata {
	return BuildMetadata{
		DeployDir:   os.Getenv("BITRISE_DEPLOY_DIR"),
		BuildNumber: os.Getenv("BITRISE_BUILD_NUMBER"),
		CommitHash:  os.Getenv("GIT_CLONE_COMMIT_HASH"),
	}
}

// WriteToDeployDir writes data to a file in the Bitrise deploy directory.
// Returns the full path of the written file.
func WriteToDeployDir(filename string, data []byte) (string, error) {
	deployDir := os.Getenv("BITRISE_DEPLOY_DIR")
	if deployDir == "" {
		return "", fmt.Errorf("BITRISE_DEPLOY_DIR is not set")
	}

	if err := os.MkdirAll(deployDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create deploy directory: %w", err)
	}

	destPath := filepath.Join(deployDir, filename)
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write to deploy directory: %w", err)
	}

	return destPath, nil
}

// ExportEnvVar exports an environment variable using envman so that
// downstream Bitrise steps can access it. Skips silently if envman
// is not available on PATH.
func ExportEnvVar(key, value string) error {
	envmanPath, err := exec.LookPath("envman")
	if err != nil {
		return nil // envman not available, skip silently
	}

	cmd := exec.Command(envmanPath, "add", "--key", key, "--value", value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("envman export %s: %w", key, err)
	}

	return nil
}

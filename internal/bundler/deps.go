package bundler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// detectPackageManager checks the project directory for lock files
// and returns the package manager name and command.
func detectPackageManager(projectDir string) (name, cmd string) {
	lockFiles := []struct {
		file string
		name string
		cmd  string
	}{
		{"yarn.lock", "yarn", "yarn"},
		{"pnpm-lock.yaml", "pnpm", "pnpm"},
		{"bun.lockb", "bun", "bun"},
		{"bun.lock", "bun", "bun"},
	}

	for _, lf := range lockFiles {
		if _, err := os.Stat(filepath.Join(projectDir, lf.file)); err == nil {
			return lf.name, lf.cmd
		}
	}

	return "npm", "npm"
}

// installDependencies detects the package manager and runs install.
func installDependencies(projectDir string, executor CommandExecutor, out *output.Writer) error {
	name, cmd := detectPackageManager(projectDir)

	return out.Spinner(fmt.Sprintf("Installing dependencies (%s)", name), func() error {
		if err := executor.Run(projectDir, os.Stderr, os.Stderr, cmd, "install"); err != nil {
			return fmt.Errorf("installing dependencies with %s failed: %w", name, err)
		}
		return nil
	})
}

package bundler

import (
	"bytes"
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

	return out.Indeterminate(fmt.Sprintf("Installing dependencies (%s)", name), func() error {
		var stderr bytes.Buffer
		if err := executor.Run(projectDir, &bytes.Buffer{}, &stderr, cmd, "install"); err != nil {
			if s := stderr.String(); s != "" {
				out.Info("%s", s)
			}
			return fmt.Errorf("installing dependencies with %s failed: %w", name, err)
		}
		return nil
	})
}

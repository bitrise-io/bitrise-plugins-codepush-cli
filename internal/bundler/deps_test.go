package bundler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name     string
		lockFile string
		wantName string
		wantCmd  string
	}{
		{
			name:     "detects yarn",
			lockFile: "yarn.lock",
			wantName: "yarn",
			wantCmd:  "yarn",
		},
		{
			name:     "detects pnpm",
			lockFile: "pnpm-lock.yaml",
			wantName: "pnpm",
			wantCmd:  "pnpm",
		},
		{
			name:     "detects bun from lockb",
			lockFile: "bun.lockb",
			wantName: "bun",
			wantCmd:  "bun",
		},
		{
			name:     "detects bun from lock",
			lockFile: "bun.lock",
			wantName: "bun",
			wantCmd:  "bun",
		},
		{
			name:     "defaults to npm",
			lockFile: "",
			wantName: "npm",
			wantCmd:  "npm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.lockFile != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, tt.lockFile), []byte{}, 0644))
			}

			name, cmd := detectPackageManager(dir)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantCmd, cmd)
		})
	}
}

func TestDetectPackageManager_PriorityOrder(t *testing.T) {
	dir := t.TempDir()

	// Create both yarn.lock and pnpm-lock.yaml; yarn should win (checked first)
	for _, f := range []string{"yarn.lock", "pnpm-lock.yaml"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte{}, 0644))
	}

	name, _ := detectPackageManager(dir)
	assert.Equal(t, "yarn", name)
}

func TestInstallDependencies(t *testing.T) {
	dir := t.TempDir()

	// Create yarn.lock so it detects yarn
	require.NoError(t, os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte{}, 0644))

	executor := &mockExecutor{}
	out := output.NewTest(io.Discard)

	err := installDependencies(dir, executor, out)
	require.NoError(t, err)

	require.Len(t, executor.commands, 1)

	cmd := executor.commands[0]
	assert.Equal(t, "yarn", cmd.name)
	assert.Equal(t, []string{"install"}, cmd.args)
	assert.Equal(t, dir, cmd.dir)
}

func TestInstallDependencies_DefaultsToNpm(t *testing.T) {
	dir := t.TempDir()
	executor := &mockExecutor{}
	out := output.NewTest(io.Discard)

	err := installDependencies(dir, executor, out)
	require.NoError(t, err)

	assert.Equal(t, "npm", executor.commands[0].name)
}

func TestInstallDependencies_Error(t *testing.T) {
	dir := t.TempDir()
	executor := &mockExecutor{err: fmt.Errorf("command failed")}
	out := output.NewTest(io.Discard)

	err := installDependencies(dir, executor, out)
	require.Error(t, err)
	assert.ErrorContains(t, err, "installing dependencies with npm failed")
	assert.ErrorContains(t, err, "command failed")
}

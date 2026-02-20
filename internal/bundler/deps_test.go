package bundler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
				if err := os.WriteFile(filepath.Join(dir, tt.lockFile), []byte{}, 0644); err != nil {
					t.Fatal(err)
				}
			}

			name, cmd := detectPackageManager(dir)
			if name != tt.wantName {
				t.Errorf("name: got %q, want %q", name, tt.wantName)
			}
			if cmd != tt.wantCmd {
				t.Errorf("cmd: got %q, want %q", cmd, tt.wantCmd)
			}
		})
	}
}

func TestDetectPackageManager_PriorityOrder(t *testing.T) {
	dir := t.TempDir()

	// Create both yarn.lock and pnpm-lock.yaml; yarn should win (checked first)
	for _, f := range []string{"yarn.lock", "pnpm-lock.yaml"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}

	name, _ := detectPackageManager(dir)
	if name != "yarn" {
		t.Errorf("expected yarn to take priority, got %q", name)
	}
}

func TestInstallDependencies(t *testing.T) {
	dir := t.TempDir()

	// Create yarn.lock so it detects yarn
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	executor := &mockExecutor{}
	out := output.NewTest(io.Discard)

	err := installDependencies(dir, executor, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(executor.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(executor.commands))
	}

	cmd := executor.commands[0]
	if cmd.name != "yarn" {
		t.Errorf("command name: got %q, want %q", cmd.name, "yarn")
	}
	if len(cmd.args) != 1 || cmd.args[0] != "install" {
		t.Errorf("command args: got %v, want [install]", cmd.args)
	}
	if cmd.dir != dir {
		t.Errorf("command dir: got %q, want %q", cmd.dir, dir)
	}
}

func TestInstallDependencies_DefaultsToNpm(t *testing.T) {
	dir := t.TempDir()
	executor := &mockExecutor{}
	out := output.NewTest(io.Discard)

	err := installDependencies(dir, executor, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if executor.commands[0].name != "npm" {
		t.Errorf("expected npm, got %q", executor.commands[0].name)
	}
}

func TestInstallDependencies_Error(t *testing.T) {
	dir := t.TempDir()
	executor := &mockExecutor{err: fmt.Errorf("command failed")}
	out := output.NewTest(io.Discard)

	err := installDependencies(dir, executor, out)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "installing dependencies with npm failed") {
		t.Errorf("error should mention install failure: %v", err)
	}
	if !strings.Contains(err.Error(), "command failed") {
		t.Errorf("error should wrap original: %v", err)
	}
}

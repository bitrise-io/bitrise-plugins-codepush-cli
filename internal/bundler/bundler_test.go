package bundler

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// mockExecutor records commands instead of executing them.
type mockExecutor struct {
	commands []executedCommand
	err      error
	// onRun is called during Run, allowing tests to create output files.
	onRun func(dir string, name string, args ...string)
}

type executedCommand struct {
	dir  string
	name string
	args []string
}

func (m *mockExecutor) Run(dir string, _ io.Writer, _ io.Writer, name string, args ...string) error {
	m.commands = append(m.commands, executedCommand{dir: dir, name: name, args: args})
	if m.onRun != nil {
		m.onRun(dir, name, args...)
	}
	return m.err
}

func TestNewBundler(t *testing.T) {
	executor := &mockExecutor{}

	tests := []struct {
		name        string
		projectType ProjectType
		wantType    string
		wantErr     bool
	}{
		{
			name:        "react native bundler",
			projectType: ProjectTypeReactNative,
			wantType:    "*bundler.ReactNativeBundler",
		},
		{
			name:        "expo bundler",
			projectType: ProjectTypeExpo,
			wantType:    "*bundler.ExpoBundler",
		},
		{
			name:        "unknown project type",
			projectType: ProjectTypeUnknown,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBundler(tt.projectType, executor, output.NewTest(io.Discard))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b == nil {
				t.Fatal("expected bundler, got nil")
			}
		})
	}
}

func TestDefaultBundleName(t *testing.T) {
	tests := []struct {
		platform Platform
		want     string
	}{
		{PlatformIOS, "main.jsbundle"},
		{PlatformAndroid, "index.android.bundle"},
		{Platform("windows"), "index.bundle"},
	}

	for _, tt := range tests {
		if got := DefaultBundleName(tt.platform); got != tt.want {
			t.Errorf("DefaultBundleName(%q) = %q, want %q", tt.platform, got, tt.want)
		}
	}
}

func TestReactNativeBundlerBundle(t *testing.T) {
	t.Run("basic iOS bundle", func(t *testing.T) {
		outputDir := t.TempDir()
		executor := &mockExecutor{}

		// Create the expected output file when "npx" is called
		executor.onRun = func(_ string, _ string, _ ...string) {
			bundlePath := filepath.Join(outputDir, "main.jsbundle")
			os.WriteFile(bundlePath, []byte("bundle"), 0o644)
			mapPath := bundlePath + ".map"
			os.WriteFile(mapPath, []byte("sourcemap"), 0o644)
		}

		bundler := &ReactNativeBundler{executor: executor, out: output.NewTest(io.Discard)}
		config := &ProjectConfig{
			ProjectDir:  "/project",
			ProjectType: ProjectTypeReactNative,
			Platform:    PlatformIOS,
			EntryFile:   "index.js",
		}
		opts := &BundleOptions{
			Platform:  PlatformIOS,
			OutputDir: outputDir,
			Sourcemap: true,
		}

		result, err := bundler.Bundle(config, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Platform != PlatformIOS {
			t.Errorf("platform: got %v, want %v", result.Platform, PlatformIOS)
		}
		if result.ProjectType != ProjectTypeReactNative {
			t.Errorf("project type: got %v, want %v", result.ProjectType, ProjectTypeReactNative)
		}

		// Verify the command was called correctly
		if len(executor.commands) != 1 {
			t.Fatalf("expected 1 command, got %d", len(executor.commands))
		}

		cmd := executor.commands[0]
		if cmd.name != "npx" {
			t.Errorf("command: got %q, want %q", cmd.name, "npx")
		}
		if cmd.args[0] != "react-native" || cmd.args[1] != "bundle" {
			t.Errorf("first args: got %v, want [react-native bundle ...]", cmd.args[:2])
		}

		// Check that key flags are present
		assertContainsArgs(t, cmd.args, "--entry-file", "index.js")
		assertContainsArgs(t, cmd.args, "--platform", "ios")
		assertContainsArgs(t, cmd.args, "--dev", "false")
		assertContainsArgs(t, cmd.args, "--sourcemap-output", result.BundlePath+".map")
	})

	t.Run("Android bundle with custom name", func(t *testing.T) {
		outputDir := t.TempDir()
		executor := &mockExecutor{}

		executor.onRun = func(_ string, _ string, _ ...string) {
			os.WriteFile(filepath.Join(outputDir, "custom.bundle"), []byte("bundle"), 0o644)
		}

		bundler := &ReactNativeBundler{executor: executor, out: output.NewTest(io.Discard)}
		config := &ProjectConfig{
			ProjectDir:  "/project",
			ProjectType: ProjectTypeReactNative,
			Platform:    PlatformAndroid,
			EntryFile:   "index.js",
			MetroConfig: "/project/metro.config.js",
		}
		opts := &BundleOptions{
			Platform:   PlatformAndroid,
			OutputDir:  outputDir,
			BundleName: "custom.bundle",
			Sourcemap:  false,
		}

		result, err := bundler.Bundle(config, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if filepath.Base(result.BundlePath) != "custom.bundle" {
			t.Errorf("bundle name: got %q, want %q", filepath.Base(result.BundlePath), "custom.bundle")
		}

		cmd := executor.commands[0]
		assertContainsArgs(t, cmd.args, "--platform", "android")
		assertContainsArgs(t, cmd.args, "--config", "/project/metro.config.js")

		// sourcemap flag should not be present
		for _, arg := range cmd.args {
			if arg == "--sourcemap-output" {
				t.Error("--sourcemap-output should not be present when Sourcemap is false")
			}
		}
	})

	t.Run("dev mode enabled", func(t *testing.T) {
		outputDir := t.TempDir()
		executor := &mockExecutor{}

		executor.onRun = func(_ string, _ string, _ ...string) {
			os.WriteFile(filepath.Join(outputDir, "main.jsbundle"), []byte("bundle"), 0o644)
		}

		bundler := &ReactNativeBundler{executor: executor, out: output.NewTest(io.Discard)}
		config := &ProjectConfig{
			ProjectDir: "/project",
			Platform:   PlatformIOS,
			EntryFile:  "index.js",
		}
		opts := &BundleOptions{
			Platform:  PlatformIOS,
			OutputDir: outputDir,
			Dev:       true,
			Sourcemap: false,
		}

		_, err := bundler.Bundle(config, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cmd := executor.commands[0]
		assertContainsArgs(t, cmd.args, "--dev", "true")
	})

	t.Run("extra bundler options", func(t *testing.T) {
		outputDir := t.TempDir()
		executor := &mockExecutor{}

		executor.onRun = func(_ string, _ string, _ ...string) {
			os.WriteFile(filepath.Join(outputDir, "main.jsbundle"), []byte("bundle"), 0o644)
		}

		bundler := &ReactNativeBundler{executor: executor, out: output.NewTest(io.Discard)}
		config := &ProjectConfig{
			ProjectDir: "/project",
			Platform:   PlatformIOS,
			EntryFile:  "index.js",
		}
		opts := &BundleOptions{
			Platform:         PlatformIOS,
			OutputDir:        outputDir,
			Sourcemap:        false,
			ExtraBundlerOpts: []string{"--reset-cache", "--verbose"},
		}

		_, err := bundler.Bundle(config, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cmd := executor.commands[0]
		args := cmd.args
		// Last two args should be the extra options
		if args[len(args)-2] != "--reset-cache" || args[len(args)-1] != "--verbose" {
			t.Errorf("extra options not appended correctly: %v", args)
		}
	})

	t.Run("bundler execution error", func(t *testing.T) {
		outputDir := t.TempDir()
		executor := &mockExecutor{err: &mockExitError{code: 1}}

		bundler := &ReactNativeBundler{executor: executor, out: output.NewTest(io.Discard)}
		config := &ProjectConfig{
			ProjectDir: "/project",
			Platform:   PlatformIOS,
			EntryFile:  "index.js",
		}
		opts := &BundleOptions{
			Platform:  PlatformIOS,
			OutputDir: outputDir,
			Sourcemap: false,
		}

		_, err := bundler.Bundle(config, opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestExpoBundlerBundle(t *testing.T) {
	t.Run("basic expo bundle", func(t *testing.T) {
		outputDir := t.TempDir()
		executor := &mockExecutor{}

		// Simulate expo export creating bundle files
		executor.onRun = func(_ string, _ string, _ ...string) {
			bundleDir := filepath.Join(outputDir, "bundles")
			os.MkdirAll(bundleDir, 0o755)
			os.WriteFile(filepath.Join(bundleDir, "ios-abc123.js"), []byte("bundle"), 0o644)
		}

		bundler := &ExpoBundler{executor: executor, out: output.NewTest(io.Discard)}
		config := &ProjectConfig{
			ProjectDir:  "/project",
			ProjectType: ProjectTypeExpo,
			Platform:    PlatformIOS,
			EntryFile:   "index.js",
		}
		opts := &BundleOptions{
			Platform:  PlatformIOS,
			OutputDir: outputDir,
		}

		result, err := bundler.Bundle(config, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProjectType != ProjectTypeExpo {
			t.Errorf("project type: got %v, want %v", result.ProjectType, ProjectTypeExpo)
		}

		cmd := executor.commands[0]
		if cmd.name != "npx" {
			t.Errorf("command: got %q, want %q", cmd.name, "npx")
		}
		if cmd.args[0] != "expo" || cmd.args[1] != "export" {
			t.Errorf("first args: got %v, want [expo export ...]", cmd.args[:2])
		}

		assertContainsArgs(t, cmd.args, "--platform", "ios")
	})

	t.Run("expo bundle with dev mode", func(t *testing.T) {
		outputDir := t.TempDir()
		executor := &mockExecutor{}

		executor.onRun = func(_ string, _ string, _ ...string) {
			os.WriteFile(filepath.Join(outputDir, "bundle.js"), []byte("bundle"), 0o644)
		}

		bundler := &ExpoBundler{executor: executor, out: output.NewTest(io.Discard)}
		config := &ProjectConfig{
			ProjectDir: "/project",
			Platform:   PlatformAndroid,
			EntryFile:  "index.js",
		}
		opts := &BundleOptions{
			Platform:  PlatformAndroid,
			OutputDir: outputDir,
			Dev:       true,
		}

		_, err := bundler.Bundle(config, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cmd := executor.commands[0]
		foundDev := false
		for _, arg := range cmd.args {
			if arg == "--dev" {
				foundDev = true
				break
			}
		}
		if !foundDev {
			t.Error("--dev flag not found in args")
		}
	})
}

// mockExitError simulates a process exit error.
type mockExitError struct {
	code int
}

func (e *mockExitError) Error() string {
	return "exit status 1"
}

// assertContainsArgs checks that the args slice contains a flag followed by its value.
func assertContainsArgs(t *testing.T, args []string, flag string, value string) {
	t.Helper()
	for i, arg := range args {
		if arg == flag && i+1 < len(args) && args[i+1] == value {
			return
		}
	}
	t.Errorf("args %v does not contain %s %s", args, flag, value)
}

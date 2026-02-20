package bundler

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestRunWithExecutor(t *testing.T) {
	t.Run("react native project without hermes", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "output")

		// Set up a minimal React Native project
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "console.log('hello')")

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			// Create the expected bundle output when npx is called
			for i, arg := range args {
				if arg == "--bundle-output" && i+1 < len(args) {
					os.MkdirAll(filepath.Dir(args[i+1]), 0o755)
					os.WriteFile(args[i+1], []byte("bundle"), 0o644)
				}
			}
		}

		opts := &BundleOptions{
			Platform:   PlatformIOS,
			ProjectDir: dir,
			OutputDir:  outputDir,
			Sourcemap:  false,
			HermesMode: HermesModeOff,
		}

		result, err := RunWithExecutor(opts, executor, output.NewTest(io.Discard))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProjectType != ProjectTypeReactNative {
			t.Errorf("project type: got %v, want %v", result.ProjectType, ProjectTypeReactNative)
		}
		if result.Platform != PlatformIOS {
			t.Errorf("platform: got %v, want %v", result.Platform, PlatformIOS)
		}
		if result.HermesApplied {
			t.Error("expected Hermes not to be applied")
		}
	})

	t.Run("expo project", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "output")

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"expo": "~49.0.0", "react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			// Simulate expo export creating a bundle file
			for i, arg := range args {
				if arg == "--output-dir" && i+1 < len(args) {
					os.MkdirAll(args[i+1], 0o755)
					os.WriteFile(filepath.Join(args[i+1], "bundle.js"), []byte("expo bundle"), 0o644)
				}
			}
		}

		opts := &BundleOptions{
			Platform:   PlatformAndroid,
			ProjectDir: dir,
			OutputDir:  outputDir,
			HermesMode: HermesModeOff,
		}

		result, err := RunWithExecutor(opts, executor, output.NewTest(io.Discard))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProjectType != ProjectTypeExpo {
			t.Errorf("project type: got %v, want %v", result.ProjectType, ProjectTypeExpo)
		}
	})

	t.Run("react native with hermes enabled but no hermesc", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "output")

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "--bundle-output" && i+1 < len(args) {
					os.MkdirAll(filepath.Dir(args[i+1]), 0o755)
					os.WriteFile(args[i+1], []byte("bundle"), 0o644)
				}
			}
		}

		opts := &BundleOptions{
			Platform:   PlatformIOS,
			ProjectDir: dir,
			OutputDir:  outputDir,
			Sourcemap:  false,
			HermesMode: HermesModeOn,
		}

		_, err := RunWithExecutor(opts, executor, output.NewTest(io.Discard))
		if err == nil {
			t.Fatal("expected error for missing hermesc, got nil")
		}
	})

	t.Run("invalid project directory", func(t *testing.T) {
		executor := &mockExecutor{}
		opts := &BundleOptions{
			Platform:   PlatformIOS,
			ProjectDir: "/nonexistent/path",
			OutputDir:  "/tmp/output",
			HermesMode: HermesModeOff,
		}

		_, err := RunWithExecutor(opts, executor, output.NewTest(io.Discard))
		if err == nil {
			t.Fatal("expected error for invalid project dir, got nil")
		}
	})

	t.Run("user overrides entry file and metro config", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "output")

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")
		writeFile(t, filepath.Join(dir, "custom.config.js"), "")

		var capturedArgs []string
		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			capturedArgs = args
			for i, arg := range args {
				if arg == "--bundle-output" && i+1 < len(args) {
					os.MkdirAll(filepath.Dir(args[i+1]), 0o755)
					os.WriteFile(args[i+1], []byte("bundle"), 0o644)
				}
			}
		}

		opts := &BundleOptions{
			Platform:   PlatformIOS,
			ProjectDir: dir,
			OutputDir:  outputDir,
			EntryFile:  "custom-entry.js",
			MetroConfig: filepath.Join(dir, "custom.config.js"),
			Sourcemap:  false,
			HermesMode: HermesModeOff,
		}

		_, err := RunWithExecutor(opts, executor, output.NewTest(io.Discard))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the overridden entry file was used
		foundEntry := false
		foundConfig := false
		for i, arg := range capturedArgs {
			if arg == "--entry-file" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "custom-entry.js" {
				foundEntry = true
			}
			if arg == "--config" && i+1 < len(capturedArgs) {
				foundConfig = true
			}
		}
		if !foundEntry {
			t.Error("custom entry file not passed to bundler")
		}
		if !foundConfig {
			t.Error("custom metro config not passed to bundler")
		}
	})

	t.Run("defaults output dir when empty", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "--bundle-output" && i+1 < len(args) {
					os.MkdirAll(filepath.Dir(args[i+1]), 0o755)
					os.WriteFile(args[i+1], []byte("bundle"), 0o644)
				}
			}
		}

		opts := &BundleOptions{
			Platform:   PlatformIOS,
			ProjectDir: dir,
			OutputDir:  "",
			Sourcemap:  false,
			HermesMode: HermesModeOff,
		}

		result, err := RunWithExecutor(opts, executor, output.NewTest(io.Discard))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.OutputDir == "" {
			t.Error("expected output dir to be set")
		}
	})

	t.Run("defaults hermes mode when empty", func(t *testing.T) {
		dir := t.TempDir()

		// Use RN < 0.70 so auto-detection defaults to Hermes off (no hermesc needed)
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.68.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "--bundle-output" && i+1 < len(args) {
					os.MkdirAll(filepath.Dir(args[i+1]), 0o755)
					os.WriteFile(args[i+1], []byte("bundle"), 0o644)
				}
			}
		}

		opts := &BundleOptions{
			Platform:   PlatformIOS,
			ProjectDir: dir,
			OutputDir:  filepath.Join(dir, "output"),
			Sourcemap:  false,
			HermesMode: "",
		}

		_, err := RunWithExecutor(opts, executor, output.NewTest(io.Discard))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("does not export bitrise summary", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "output")
		deployDir := filepath.Join(dir, "deploy")
		if err := os.MkdirAll(deployDir, 0o755); err != nil {
			t.Fatal(err)
		}

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "--bundle-output" && i+1 < len(args) {
					os.MkdirAll(filepath.Dir(args[i+1]), 0o755)
					os.WriteFile(args[i+1], []byte("bundle"), 0o644)
				}
			}
		}

		t.Setenv("BITRISE_DEPLOY_DIR", deployDir)
		t.Setenv("BITRISE_BUILD_NUMBER", "42")

		opts := &BundleOptions{
			Platform:   PlatformIOS,
			ProjectDir: dir,
			OutputDir:  outputDir,
			Sourcemap:  false,
			HermesMode: HermesModeOff,
		}

		_, err := RunWithExecutor(opts, executor, output.NewTest(io.Discard))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// RunWithExecutor no longer exports to Bitrise deploy dir; the CLI layer handles that
		summaryPath := filepath.Join(deployDir, "codepush-bundle-summary.json")
		if _, err := os.Stat(summaryPath); err == nil {
			t.Error("RunWithExecutor should not export summary; that responsibility moved to CLI layer")
		}
	})
}


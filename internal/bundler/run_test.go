package bundler

import (
	"os"
	"path/filepath"
	"testing"
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

		result, err := RunWithExecutor(opts, executor)
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

		result, err := RunWithExecutor(opts, executor)
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

		_, err := RunWithExecutor(opts, executor)
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

		_, err := RunWithExecutor(opts, executor)
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

		_, err := RunWithExecutor(opts, executor)
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

		result, err := RunWithExecutor(opts, executor)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.OutputDir == "" {
			t.Error("expected output dir to be set")
		}
	})

	t.Run("defaults hermes mode when empty", func(t *testing.T) {
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
			OutputDir:  filepath.Join(dir, "output"),
			Sourcemap:  false,
			HermesMode: "",
		}

		_, err := RunWithExecutor(opts, executor)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("bitrise environment exports summary", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "output")
		deployDir := filepath.Join(dir, "deploy")
		os.MkdirAll(deployDir, 0o755)

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

		// Simulate Bitrise environment
		t.Setenv("BITRISE_DEPLOY_DIR", deployDir)
		t.Setenv("BITRISE_BUILD_NUMBER", "42")

		opts := &BundleOptions{
			Platform:   PlatformIOS,
			ProjectDir: dir,
			OutputDir:  outputDir,
			Sourcemap:  false,
			HermesMode: HermesModeOff,
		}

		_, err := RunWithExecutor(opts, executor)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify summary file was created
		summaryPath := filepath.Join(deployDir, "codepush-bundle-summary.json")
		if _, err := os.Stat(summaryPath); err != nil {
			t.Errorf("expected bundle summary at %s: %v", summaryPath, err)
		}
	})
}

func TestExportBitriseSummary(t *testing.T) {
	t.Run("writes summary json", func(t *testing.T) {
		deployDir := t.TempDir()
		t.Setenv("BITRISE_DEPLOY_DIR", deployDir)

		result := &BundleResult{
			BundlePath:    "/path/to/bundle.js",
			AssetsDir:     "/path/to/assets",
			SourcemapPath: "/path/to/bundle.js.map",
			OutputDir:     "/path/to/output",
			HermesApplied: true,
			ProjectType:   ProjectTypeReactNative,
			Platform:      PlatformIOS,
		}

		exportBitriseSummary(result)

		summaryPath := filepath.Join(deployDir, "codepush-bundle-summary.json")
		data, err := os.ReadFile(summaryPath)
		if err != nil {
			t.Fatalf("reading summary: %v", err)
		}

		content := string(data)
		if !contains(content, `"platform": "ios"`) {
			t.Error("summary missing platform")
		}
		if !contains(content, `"hermes_applied": true`) {
			t.Error("summary missing hermes_applied")
		}
		if !contains(content, `"project_type": "react-native"`) {
			t.Error("summary missing project_type")
		}
	})

	t.Run("handles missing deploy dir gracefully", func(t *testing.T) {
		t.Setenv("BITRISE_DEPLOY_DIR", "")

		result := &BundleResult{
			Platform:    PlatformIOS,
			ProjectType: ProjectTypeReactNative,
		}

		// Should not panic
		exportBitriseSummary(result)
	})
}

func contains(s string, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s string, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

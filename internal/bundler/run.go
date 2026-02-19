package bundler

import (
	"fmt"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
	"os"
	"path/filepath"
)

// Run executes the full bundle pipeline:
// 1. Detect project configuration
// 2. Execute the appropriate bundler
// 3. Compile with Hermes if applicable
// 4. Export to Bitrise deploy directory if in Bitrise environment
func Run(opts *BundleOptions, out *output.Writer) (*BundleResult, error) {
	return RunWithExecutor(opts, &DefaultExecutor{}, out)
}

// RunWithExecutor executes the full bundle pipeline with the given executor.
// This allows tests to provide a mock executor.
func RunWithExecutor(opts *BundleOptions, executor CommandExecutor, out *output.Writer) (*BundleResult, error) {
	projectDir := opts.ProjectDir
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
		projectDir = cwd
	}

	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project directory: %w", err)
	}
	opts.ProjectDir = absProjectDir

	if opts.OutputDir == "" {
		opts.OutputDir = DefaultOutputDir
	}

	hermesMode := opts.HermesMode
	if hermesMode == "" {
		hermesMode = HermesModeAuto
	}

	// Step 1: Detect project configuration
	config, err := DetectProject(absProjectDir, opts.Platform, hermesMode)
	if err != nil {
		return nil, err
	}

	// Apply user overrides
	if opts.EntryFile != "" {
		config.EntryFile = opts.EntryFile
	}
	if opts.MetroConfig != "" {
		config.MetroConfig = opts.MetroConfig
	}

	out.Info("Project type: %s", config.ProjectType)
	out.Info("Platform: %s", config.Platform)
	out.Info("Entry file: %s", config.EntryFile)
	out.Info("Hermes: %v", config.HermesEnabled)

	// Step 2: Create and run the bundler
	bundler, err := NewBundler(config.ProjectType, executor, out)
	if err != nil {
		return nil, err
	}

	result, err := bundler.Bundle(config, opts)
	if err != nil {
		return nil, err
	}

	// Step 3: Hermes compilation (only for React Native, Expo handles it internally)
	if config.HermesEnabled && config.ProjectType == ProjectTypeReactNative {
		if config.HermescPath == "" {
			return nil, fmt.Errorf("hermes is enabled but hermesc was not found in node_modules: run 'npm install' or use --hermes=off")
		}

		compiler := NewHermesCompiler(executor, out)
		if err := compiler.Compile(config.HermescPath, result.BundlePath, result.SourcemapPath); err != nil {
			return nil, err
		}
		result.HermesApplied = true
	}

	return result, nil
}

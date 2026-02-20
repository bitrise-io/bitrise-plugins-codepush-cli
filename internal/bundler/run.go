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
	hermesMode, err := resolveRunOptions(opts)
	if err != nil {
		return nil, err
	}

	if !opts.SkipInstall {
		if err := installDependencies(opts.ProjectDir, executor, out); err != nil {
			return nil, err
		}
	}

	config, err := DetectProject(opts.ProjectDir, opts.Platform, hermesMode)
	if err != nil {
		return nil, err
	}

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

	bundler, err := NewBundler(config.ProjectType, executor, out)
	if err != nil {
		return nil, err
	}

	result, err := bundler.Bundle(config, opts)
	if err != nil {
		return nil, err
	}

	if err := compileWithHermes(config, result, executor, out); err != nil {
		return nil, err
	}

	return result, nil
}

func resolveRunOptions(opts *BundleOptions) (HermesMode, error) {
	projectDir := opts.ProjectDir
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting current directory: %w", err)
		}
		projectDir = cwd
	}

	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("resolving project directory: %w", err)
	}
	opts.ProjectDir = absProjectDir

	if opts.OutputDir == "" {
		opts.OutputDir = DefaultOutputDir
	}

	hermesMode := opts.HermesMode
	if hermesMode == "" {
		hermesMode = HermesModeAuto
	}
	return hermesMode, nil
}

func compileWithHermes(config *ProjectConfig, result *BundleResult, executor CommandExecutor, out *output.Writer) error {
	if !config.HermesEnabled || config.ProjectType != ProjectTypeReactNative {
		return nil
	}
	if config.HermescPath == "" {
		return fmt.Errorf("hermes is enabled but hermesc was not found in node_modules: run 'npm install' or use --hermes=off")
	}

	compiler := NewHermesCompiler(executor, out)
	if err := compiler.Compile(config.HermescPath, result.BundlePath, result.SourcemapPath); err != nil {
		return err
	}
	result.HermesApplied = true
	return nil
}

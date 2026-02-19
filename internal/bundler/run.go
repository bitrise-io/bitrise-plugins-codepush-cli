package bundler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
)

// bundleSummary is exported to Bitrise deploy directory as JSON.
type bundleSummary struct {
	Platform      string `json:"platform"`
	ProjectType   string `json:"project_type"`
	BundlePath    string `json:"bundle_path"`
	AssetsDir     string `json:"assets_dir"`
	SourcemapPath string `json:"sourcemap_path,omitempty"`
	HermesApplied bool   `json:"hermes_applied"`
}

// Run executes the full bundle pipeline:
// 1. Detect project configuration
// 2. Execute the appropriate bundler
// 3. Compile with Hermes if applicable
// 4. Export to Bitrise deploy directory if in Bitrise environment
func Run(opts *BundleOptions) (*BundleResult, error) {
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
		opts.OutputDir = "./codepush-bundle"
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

	fmt.Fprintf(os.Stderr, "Project type: %s\n", config.ProjectType)
	fmt.Fprintf(os.Stderr, "Platform: %s\n", config.Platform)
	fmt.Fprintf(os.Stderr, "Entry file: %s\n", config.EntryFile)
	fmt.Fprintf(os.Stderr, "Hermes: %v\n", config.HermesEnabled)
	fmt.Fprintln(os.Stderr)

	// Step 2: Create and run the bundler
	executor := &DefaultExecutor{}
	bundler, err := NewBundler(config.ProjectType, executor)
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

		compiler := NewHermesCompiler(executor)
		if err := compiler.Compile(config.HermescPath, result.BundlePath, result.SourcemapPath); err != nil {
			return nil, err
		}
		result.HermesApplied = true
		fmt.Fprintln(os.Stderr)
	}

	// Step 4: Export to Bitrise deploy directory
	if bitrise.IsBitriseEnvironment() {
		fmt.Fprintf(os.Stderr, "Bitrise environment detected, exporting bundle summary to deploy directory\n")
		exportBitriseSummary(result)
	}

	return result, nil
}

// exportBitriseSummary writes a JSON summary to the Bitrise deploy directory.
func exportBitriseSummary(result *BundleResult) {
	summary := bundleSummary{
		Platform:      string(result.Platform),
		ProjectType:   result.ProjectType.String(),
		BundlePath:    result.BundlePath,
		AssetsDir:     result.AssetsDir,
		SourcemapPath: result.SourcemapPath,
		HermesApplied: result.HermesApplied,
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal bundle summary: %v\n", err)
		return
	}

	path, err := bitrise.WriteToDeployDir("codepush-bundle-summary.json", data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to export bundle summary: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Bundle summary exported to: %s\n", path)
}

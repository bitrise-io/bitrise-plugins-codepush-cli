package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReactNativeBundler bundles using "npx react-native bundle" (Metro bundler).
type ReactNativeBundler struct {
	executor CommandExecutor
}

// Bundle implements Bundler for React Native projects.
func (b *ReactNativeBundler) Bundle(config *ProjectConfig, opts *BundleOptions) (*BundleResult, error) {
	outputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output directory: %w", err)
	}

	assetsDir := filepath.Join(outputDir, "assets")
	if err := ensureDir(assetsDir); err != nil {
		return nil, err
	}

	bundleName := opts.BundleName
	if bundleName == "" {
		bundleName = DefaultBundleName(opts.Platform)
	}

	bundlePath := filepath.Join(outputDir, bundleName)

	var sourcemapPath string
	if opts.Sourcemap {
		sourcemapPath = bundlePath + ".map"
	}

	args := b.buildArgs(config, opts, outputDir, bundlePath, assetsDir, sourcemapPath)

	fmt.Fprintf(os.Stderr, "Running: npx %s\n", strings.Join(args, " "))

	if err := b.executor.Run(config.ProjectDir, os.Stderr, os.Stderr, "npx", args...); err != nil {
		return nil, fmt.Errorf("react-native bundle failed: %w", err)
	}

	if _, err := os.Stat(bundlePath); err != nil {
		return nil, fmt.Errorf("bundle file was not created at %s", bundlePath)
	}

	result := &BundleResult{
		BundlePath:  bundlePath,
		AssetsDir:   assetsDir,
		OutputDir:   outputDir,
		ProjectType: ProjectTypeReactNative,
		Platform:    opts.Platform,
	}

	if sourcemapPath != "" {
		if _, err := os.Stat(sourcemapPath); err == nil {
			result.SourcemapPath = sourcemapPath
		}
	}

	return result, nil
}

// buildArgs constructs the argument list for "npx react-native bundle".
func (b *ReactNativeBundler) buildArgs(
	config *ProjectConfig,
	opts *BundleOptions,
	outputDir string,
	bundlePath string,
	assetsDir string,
	sourcemapPath string,
) []string {
	entryFile := opts.EntryFile
	if entryFile == "" {
		entryFile = config.EntryFile
	}

	devStr := "false"
	if opts.Dev {
		devStr = "true"
	}

	args := []string{
		"react-native", "bundle",
		"--entry-file", entryFile,
		"--platform", string(opts.Platform),
		"--dev", devStr,
		"--bundle-output", bundlePath,
		"--assets-dest", assetsDir,
	}

	if sourcemapPath != "" {
		args = append(args, "--sourcemap-output", sourcemapPath)
	}

	metroConfig := opts.MetroConfig
	if metroConfig == "" {
		metroConfig = config.MetroConfig
	}
	if metroConfig != "" {
		args = append(args, "--config", metroConfig)
	}

	args = append(args, opts.ExtraBundlerOpts...)

	return args
}

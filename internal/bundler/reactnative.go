package bundler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// bundlePaths groups the derived file paths used during bundling.
type bundlePaths struct {
	outputDir     string
	bundlePath    string
	assetsDir     string
	sourcemapPath string
}

// ReactNativeBundler bundles using "npx react-native bundle" (Metro bundler).
type ReactNativeBundler struct {
	executor CommandExecutor
	out      *output.Writer
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

	sourcemapPath, err := resolveSourcemapPath(opts, bundlePath)
	if err != nil {
		return nil, err
	}

	paths := bundlePaths{
		outputDir:     outputDir,
		bundlePath:    bundlePath,
		assetsDir:     assetsDir,
		sourcemapPath: sourcemapPath,
	}
	args := b.buildArgs(config, opts, paths)

	progress := b.out.NewProgress("Bundling " + string(opts.Platform))
	mw := output.NewMetroProgressWriter(progress)
	if err := b.runBundle(config.ProjectDir, mw, "npx", args...); err != nil {
		mw.Flush()
		progress.Done("")
		b.out.Info("%s", mw.Buffered())
		return nil, fmt.Errorf("react-native bundle failed: %w", err)
	}
	mw.Flush()
	progress.Done("")

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
func (b *ReactNativeBundler) buildArgs(config *ProjectConfig, opts *BundleOptions, paths bundlePaths) []string {
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
		"--bundle-output", paths.bundlePath,
		"--assets-dest", paths.assetsDir,
	}

	if paths.sourcemapPath != "" {
		args = append(args, "--sourcemap-output", paths.sourcemapPath)
	}

	if opts.ResetCache {
		args = append(args, "--reset-cache")
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

func (b *ReactNativeBundler) runBundle(dir string, w io.Writer, name string, args ...string) error {
	if b.out.IsInteractive() {
		return runWithPTY(dir, w, name, args...)
	}
	return b.executor.Run(dir, io.Discard, w, name, args...)
}

// resolveSourcemapPath returns the absolute sourcemap path based on bundle options.
// Returns an empty string when sourcemaps are disabled.
func resolveSourcemapPath(opts *BundleOptions, bundlePath string) (string, error) {
	if !opts.Sourcemap {
		return "", nil
	}
	if opts.SourcemapOutput == "" {
		return bundlePath + ".map", nil
	}
	absPath := opts.SourcemapOutput
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(opts.ProjectDir, absPath)
	}
	if err := ensureDir(filepath.Dir(absPath)); err != nil {
		return "", fmt.Errorf("creating sourcemap output directory: %w", err)
	}
	return absPath, nil
}

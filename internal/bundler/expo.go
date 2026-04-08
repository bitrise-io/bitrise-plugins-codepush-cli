package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// ExpoBundler bundles using "npx expo export:embed" for Expo-managed projects.
// export:embed uses the same Metro+Hermes pipeline as the native app build,
// producing a bundle the CodePush SDK can load directly.
type ExpoBundler struct {
	executor CommandExecutor
	out      *output.Writer
}

// Bundle implements Bundler for Expo projects.
func (b *ExpoBundler) Bundle(config *ProjectConfig, opts *BundleOptions) (*BundleResult, error) {
	outputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output directory: %w", err)
	}

	if err := ensureDir(outputDir); err != nil {
		return nil, err
	}

	bundleName := resolveExpoBundleName(config, opts)
	bundlePath := filepath.Join(outputDir, bundleName)

	args := b.buildArgs(config, opts, outputDir, bundlePath)

	b.out.Info("Running: npx %s", strings.Join(args, " "))

	if err := b.executor.Run(config.ProjectDir, os.Stderr, os.Stderr, "npx", args...); err != nil {
		return nil, fmt.Errorf("expo export:embed failed: %w", err)
	}

	result := &BundleResult{
		BundlePath:  bundlePath,
		AssetsDir:   outputDir,
		OutputDir:   outputDir,
		ProjectType: ProjectTypeExpo,
		Platform:    opts.Platform,
	}

	if opts.Sourcemap || opts.SourcemapOutput != "" {
		mapPath := sourcemapPath(opts, bundlePath)
		if _, err := os.Stat(mapPath); err == nil {
			result.SourcemapPath = mapPath
		}
	}

	return result, nil
}

// buildArgs constructs the argument list for "npx expo export:embed".
func (b *ExpoBundler) buildArgs(config *ProjectConfig, opts *BundleOptions, outputDir, bundlePath string) []string {
	args := []string{
		"expo", "export:embed",
		"--entry-file", config.EntryFile,
		"--platform", string(opts.Platform),
		"--bundle-output", bundlePath,
		"--assets-dest", outputDir,
		"--dev", strconv.FormatBool(opts.Dev),
		"--minify", "false",
		"--reset-cache",
	}

	if config.HermesEnabled {
		args = append(args, "--bytecode")
	}

	mapPath := sourcemapPath(opts, bundlePath)
	if opts.Sourcemap || opts.SourcemapOutput != "" {
		args = append(args, "--sourcemap-output", mapPath)
	}

	args = append(args, opts.ExtraBundlerOpts...)

	return args
}

// resolveExpoBundleName returns the bundle filename the CodePush SDK expects to find
// in the zip. Priority: opts.BundleName (--bundle-name flag) > config.BundleName
// (auto-detected from native project files).
func resolveExpoBundleName(config *ProjectConfig, opts *BundleOptions) string {
	if opts.BundleName != "" {
		return opts.BundleName
	}
	return config.BundleName
}

// sourcemapPath returns the sourcemap output path for expo export:embed.
// If SourcemapOutput is explicitly set, that path is used; otherwise the
// map is placed next to the bundle at bundlePath+".map".
func sourcemapPath(opts *BundleOptions, bundlePath string) string {
	if opts.SourcemapOutput != "" {
		return opts.SourcemapOutput
	}
	return bundlePath + ".map"
}

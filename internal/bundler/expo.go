package bundler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// ExpoBundler bundles using "npx expo export" for Expo-managed projects.
type ExpoBundler struct {
	executor CommandExecutor
	out      *output.Writer
}

// Bundle implements Bundler for Expo projects.
func (b *ExpoBundler) Bundle(config *ProjectConfig, opts *BundleOptions) (*BundleResult, error) {
	// expo export always writes the sourcemap next to the bundle; there is no
	// flag to redirect the map path, so --sourcemap-output is unsupported.
	if opts.SourcemapOutput != "" {
		return nil, errors.New("--sourcemap-output is not supported for Expo projects: expo export always writes the sourcemap next to the bundle")
	}

	outputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output directory: %w", err)
	}

	if err := ensureDir(outputDir); err != nil {
		return nil, err
	}

	args := b.buildArgs(opts, outputDir)

	b.out.Info("Running: npx %s", strings.Join(args, " "))

	if err := b.executor.Run(config.ProjectDir, os.Stderr, os.Stderr, "npx", args...); err != nil {
		return nil, fmt.Errorf("expo export failed: %w", err)
	}

	// Locate the bundle in Expo's output structure
	bundlePath, err := findExpoBundleOutput(outputDir, opts.Platform)
	if err != nil {
		return nil, err
	}

	result := &BundleResult{
		BundlePath:  bundlePath,
		AssetsDir:   filepath.Join(outputDir, "assets"),
		OutputDir:   outputDir,
		ProjectType: ProjectTypeExpo,
		Platform:    opts.Platform,
	}

	// expo export writes the sourcemap next to the bundle at bundlePath+".map".
	if opts.Sourcemap {
		mapPath := bundlePath + ".map"
		if _, err := os.Stat(mapPath); err == nil {
			result.SourcemapPath = mapPath
		}
	}

	return result, nil
}

// buildArgs constructs the argument list for "npx expo export".
func (b *ExpoBundler) buildArgs(opts *BundleOptions, outputDir string) []string {
	args := []string{
		"expo", "export",
		"--output-dir", outputDir,
		"--platform", string(opts.Platform),
	}

	if opts.Dev {
		args = append(args, "--dev")
	}

	args = append(args, opts.ExtraBundlerOpts...)

	return args
}

// findExpoBundleOutput locates the JS/HBC bundle file in the Expo export output directory.
// It prefers .hbc (Hermes bytecode) over .js when both are present.
func findExpoBundleOutput(outputDir string, platform Platform) (string, error) {
	// Expo export creates bundles in _expo/static/js directories or
	// in bundles/ directory depending on the version.
	// Check common locations, preferring .hbc (Hermes) over .js.
	candidates := []string{
		filepath.Join(outputDir, "bundles", fmt.Sprintf("%s-*.hbc", platform)),
		filepath.Join(outputDir, "bundles", fmt.Sprintf("%s-*.js", platform)),
		filepath.Join(outputDir, "_expo", "static", "js", string(platform), "*.hbc"),
		filepath.Join(outputDir, "_expo", "static", "js", string(platform), "*.js"),
	}

	for _, pattern := range candidates {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		if len(matches) > 0 {
			return matches[0], nil
		}
	}

	// Fallback: scan the output directory for .hbc or .js bundle files (not sourcemaps).
	// Collect both and prefer .hbc over .js.
	var hbcFiles, jsFiles []string
	_ = filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries during fallback scan
		}
		if info.IsDir() {
			return nil
		}
		switch {
		case strings.HasSuffix(info.Name(), ".hbc"):
			hbcFiles = append(hbcFiles, path)
		case strings.HasSuffix(info.Name(), ".js"):
			jsFiles = append(jsFiles, path)
		}
		return nil
	})

	// Prefer .hbc; fall back to .js.
	for _, candidates := range [][]string{hbcFiles, jsFiles} {
		if len(candidates) == 1 {
			return candidates[0], nil
		}
		if len(candidates) > 1 {
			ext := filepath.Ext(candidates[0])
			return "", fmt.Errorf("found %d %s files in %s but could not determine which is the bundle: expected output in bundles/ or _expo/static/js/%s/", len(candidates), ext, outputDir, platform)
		}
	}

	return "", fmt.Errorf("could not find bundle output in %s: check that expo export completed successfully", outputDir)
}

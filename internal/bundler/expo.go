package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpoBundler bundles using "npx expo export" for Expo-managed projects.
type ExpoBundler struct {
	executor CommandExecutor
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

	args := b.buildArgs(opts, outputDir)

	fmt.Fprintf(os.Stderr, "Running: npx %s\n", strings.Join(args, " "))

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

	// Check for sourcemap
	mapPath := bundlePath + ".map"
	if _, err := os.Stat(mapPath); err == nil {
		result.SourcemapPath = mapPath
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

// findExpoBundleOutput locates the JS bundle file in the Expo export output directory.
func findExpoBundleOutput(outputDir string, platform Platform) (string, error) {
	// Expo export creates bundles in _expo/static/js directories or
	// in bundles/ directory depending on the version.
	// Check common locations.
	candidates := []string{
		filepath.Join(outputDir, "bundles", fmt.Sprintf("%s-*.js", platform)),
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

	// Fallback: scan the output directory for .js bundle files (not sourcemaps)
	var jsFiles []string
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".js") && !strings.HasSuffix(info.Name(), ".js.map") {
			jsFiles = append(jsFiles, path)
		}
		return nil
	})

	if len(jsFiles) == 1 {
		return jsFiles[0], nil
	}
	if len(jsFiles) > 1 {
		return "", fmt.Errorf("found %d .js files in %s but could not determine which is the bundle: expected output in bundles/ or _expo/static/js/%s/", len(jsFiles), outputDir, platform)
	}

	return "", fmt.Errorf("could not find bundle output in %s: check that expo export completed successfully", outputDir)
}

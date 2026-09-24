package bundler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

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

	bundleName := resolveExpoBundleName(config, opts)
	bundlePath := filepath.Join(outputDir, bundleName)

	mapPath, err := resolveSourcemapPath(opts, outputDir, bundleName)
	if err != nil {
		return nil, err
	}

	if err := ensureDir(outputDir); err != nil {
		return nil, err
	}

	args := b.buildArgs(config, opts, outputDir, bundlePath, mapPath)

	progress := b.out.NewProgress("Bundling " + string(opts.Platform))
	mw := output.NewMetroProgressWriter(progress)
	err = b.runBundle(config.ProjectDir, mw, "npx", args...)
	mw.Flush()
	if err != nil {
		progress.Cancel()
		b.out.Info("%s", mw.Buffered())
		return nil, fmt.Errorf("expo export:embed failed: %w", err)
	}
	progress.Done("")

	result := &BundleResult{
		BundlePath: bundlePath,
		AssetsDir:  outputDir,
		OutputDir:  outputDir,
		// HermesApplied mirrors config.HermesEnabled: when true, --bytecode was passed
		// to expo export:embed, which manages Hermes internally (unlike the RN path where
		// hermesc runs as a separate post-bundle step).
		HermesApplied: config.HermesEnabled,
		ProjectType:   ProjectTypeExpo,
		Platform:      opts.Platform,
	}

	if mapPath != "" {
		if _, err := os.Stat(mapPath); err == nil {
			result.SourcemapPath = mapPath
		}
	}

	return result, nil
}

// buildArgs constructs the argument list for "npx expo export:embed".
func (b *ExpoBundler) runBundle(dir string, w io.Writer, name string, args ...string) error {
	if b.out.IsInteractive() {
		return runWithPTY(dir, w, name, args...)
	}
	return b.executor.Run(dir, io.Discard, w, name, args...)
}

func (b *ExpoBundler) buildArgs(config *ProjectConfig, opts *BundleOptions, outputDir, bundlePath, mapPath string) []string {
	args := []string{
		"expo", "export:embed",
		"--entry-file", config.EntryFile,
		"--platform", string(opts.Platform),
		"--bundle-output", bundlePath,
		"--assets-dest", outputDir,
		"--dev", strconv.FormatBool(opts.Dev),
		"--minify", strconv.FormatBool(opts.Minify),
	}

	if opts.ResetCache {
		args = append(args, "--reset-cache")
	}

	if config.HermesEnabled {
		args = append(args, "--bytecode")
	}

	if mapPath != "" {
		args = append(args, "--sourcemap-output", mapPath)
	}

	args = append(args, opts.ExtraBundlerOpts...)

	return args
}

// resolveExpoBundleName returns the bundle filename the CodePush SDK expects to find
// in the zip. Priority: opts.BundleName (--bundle-name flag) > config.BundleName
// (auto-detected from native project files) > DefaultBundleName.
func resolveExpoBundleName(config *ProjectConfig, opts *BundleOptions) string {
	if opts.BundleName != "" {
		return opts.BundleName
	}
	if config.BundleName != "" {
		return config.BundleName
	}
	return DefaultBundleName(config.Platform)
}

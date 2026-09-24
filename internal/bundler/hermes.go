package bundler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// HermesCompiler handles Hermes bytecode compilation of JS bundles.
type HermesCompiler struct {
	executor CommandExecutor
	out      *output.Writer
}

// NewHermesCompiler creates a new HermesCompiler.
func NewHermesCompiler(executor CommandExecutor, out *output.Writer) *HermesCompiler {
	return &HermesCompiler{executor: executor, out: out}
}

// CompileOptions holds the inputs for a Hermes compilation.
type CompileOptions struct {
	HermescPath string
	BundlePath  string
	// SourcemapPath is the Metro source map. When set, hermesc also writes a
	// source map, and the two are composed into this path.
	SourcemapPath string
	// ProjectDir is where the compose-source-maps.js script is looked up.
	ProjectDir string
	// ExtraFlags are appended to the hermesc invocation before the input file.
	ExtraFlags []string
}

// Compile compiles a JS bundle to Hermes bytecode.
// The compiled bytecode replaces the original bundle file (CodePush clients
// expect the original filename).
// With a source map, hermesc writes its outputs next to the source map, so its
// own map never lands in the bundle's directory (the update payload).
func (h *HermesCompiler) Compile(opts *CompileOptions) error {
	if _, err := os.Stat(opts.HermescPath); err != nil {
		return fmt.Errorf("hermesc binary not found at %s: %w", opts.HermescPath, err)
	}

	if _, err := os.Stat(opts.BundlePath); err != nil {
		return fmt.Errorf("bundle file not found at %s: %w", opts.BundlePath, err)
	}

	hbcPath := hermesWorkPath(opts.BundlePath, opts.SourcemapPath)

	// Compile JS to Hermes bytecode
	args := []string{"-emit-binary", "-out", hbcPath}

	if opts.SourcemapPath != "" {
		args = append(args, "-output-source-map")
	}

	args = append(args, opts.ExtraFlags...)
	args = append(args, opts.BundlePath)

	h.out.Step("Running Hermes compilation: %s %v", opts.HermescPath, args)

	if err := h.executor.Run("", os.Stderr, os.Stderr, opts.HermescPath, args...); err != nil {
		return fmt.Errorf("hermes compilation failed: %w", err)
	}

	// Replace the original JS bundle with the compiled bytecode
	if err := moveFile(hbcPath, opts.BundlePath); err != nil {
		return fmt.Errorf("replacing bundle with Hermes bytecode: %w", err)
	}

	// Compose source maps if both metro and hermes source maps exist
	if opts.SourcemapPath != "" {
		hermesMapPath := hbcPath + ".map"
		if _, err := os.Stat(hermesMapPath); err == nil {
			h.composeSourceMaps(opts.ProjectDir, opts.SourcemapPath, hermesMapPath)
		}
	}

	return nil
}

// hermesWorkPath returns the hermesc -out path. hermesc writes its source map
// to the same path plus ".map".
//
// With a source map, the path is derived from the source map path by replacing
// its ".map" extension with ".hbc" (main.jsbundle.map -> main.jsbundle.hbc and
// main.jsbundle.hbc.map). Neither output can be the source map path itself,
// whatever name the user chose, so hermesc never overwrites the Metro map.
func hermesWorkPath(bundlePath, sourcemapPath string) string {
	if sourcemapPath == "" {
		return bundlePath + ".hbc"
	}
	return strings.TrimSuffix(sourcemapPath, ".map") + ".hbc"
}

// composeSourceMaps attempts to compose Metro and Hermes source maps into
// metroMapPath. This is a best-effort operation: on failure both maps are kept
// so they can be composed by hand, and a warning is logged.
func (h *HermesCompiler) composeSourceMaps(projectDir, metroMapPath, hermesMapPath string) {
	composeScript := filepath.Join(projectDir, "node_modules", "react-native", "scripts", "compose-source-maps.js")
	manualCmd := fmt.Sprintf("node %s %s %s -o <output>", composeScript, metroMapPath, hermesMapPath)

	if _, err := os.Stat(composeScript); err != nil {
		h.warnUncomposed("compose-source-maps.js not found at "+composeScript, manualCmd)
		return
	}

	composedPath := metroMapPath + ".composed"
	err := h.executor.Run(projectDir, os.Stderr, os.Stderr, "node", composeScript, metroMapPath, hermesMapPath, "-o", composedPath)
	if err != nil {
		_ = os.Remove(composedPath)
		h.warnUncomposed("source map composition failed", manualCmd)
		return
	}

	// Replace original sourcemap with composed one
	if err := os.Rename(composedPath, metroMapPath); err != nil {
		h.warnUncomposed(fmt.Sprintf("could not replace source map with composed version: %v", err), manualCmd)
		return
	}
	if err := os.Remove(hermesMapPath); err != nil {
		h.out.Warning("could not clean up Hermes source map: %v", err)
	}
}

func (h *HermesCompiler) warnUncomposed(reason, manualCmd string) {
	h.out.Warning("%s. Kept the Metro and Hermes source maps; compose them with: %s", reason, manualCmd)
}

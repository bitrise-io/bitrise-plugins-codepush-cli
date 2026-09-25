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

type CompileOptions struct {
	HermescPath   string
	BundlePath    string
	SourcemapPath string
	ProjectDir    string
	ExtraFlags    []string
}

// Compile replaces the bundle with bytecode under the same name, because
// CodePush clients expect the original filename.
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

// With a source map, hermesc writes next to it so its own map stays out of the
// payload. Deriving the name from the source map path means neither output
// (x.hbc, x.hbc.map) can overwrite a custom --sourcemap-output.
func hermesWorkPath(bundlePath, sourcemapPath string) string {
	if sourcemapPath == "" {
		return bundlePath + ".hbc"
	}
	return strings.TrimSuffix(sourcemapPath, ".map") + ".hbc"
}

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

package bundler

import (
	"fmt"
	"os"
	"path/filepath"

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

// Compile takes a JS bundle path and compiles it to Hermes bytecode.
// The compiled bytecode replaces the original bundle file (CodePush clients
// expect the original filename).
// If sourcemapPath is non-empty, attempts to compose source maps.
func (h *HermesCompiler) Compile(hermescPath string, bundlePath string, sourcemapPath string) error {
	if _, err := os.Stat(hermescPath); err != nil {
		return fmt.Errorf("hermesc binary not found at %s: %w", hermescPath, err)
	}

	if _, err := os.Stat(bundlePath); err != nil {
		return fmt.Errorf("bundle file not found at %s: %w", bundlePath, err)
	}

	hbcPath := bundlePath + ".hbc"

	// Compile JS to Hermes bytecode
	args := []string{"-emit-binary", "-out", hbcPath}

	if sourcemapPath != "" {
		args = append(args, "-output-source-map")
	}

	args = append(args, bundlePath)

	h.out.Step("Running Hermes compilation: %s %v", hermescPath, args)

	if err := h.executor.Run("", os.Stderr, os.Stderr, hermescPath, args...); err != nil {
		return fmt.Errorf("hermes compilation failed: %w", err)
	}

	// Replace the original JS bundle with the compiled bytecode
	if err := os.Rename(hbcPath, bundlePath); err != nil {
		return fmt.Errorf("replacing bundle with Hermes bytecode: %w", err)
	}

	// Compose source maps if both metro and hermes source maps exist
	if sourcemapPath != "" {
		hermesMapPath := hbcPath + ".map"
		if _, err := os.Stat(hermesMapPath); err == nil {
			h.composeSourceMaps(bundlePath, sourcemapPath, hermesMapPath)
		}
	}

	return nil
}

// composeSourceMaps attempts to compose Metro and Hermes source maps.
// This is a best-effort operation; failures are logged but not fatal.
func (h *HermesCompiler) composeSourceMaps(bundlePath string, metroMapPath string, hermesMapPath string) {
	projectDir := filepath.Dir(bundlePath)

	// Look for the compose-source-maps script
	composeScript := filepath.Join(projectDir, "node_modules", "react-native", "scripts", "compose-source-maps.js")
	if _, err := os.Stat(composeScript); err != nil {
		h.out.Warning("compose-source-maps.js not found, using Hermes source map only")
		if err := os.Rename(hermesMapPath, metroMapPath); err != nil {
			h.out.Warning("could not rename Hermes source map: %v", err)
		}
		return
	}

	composedPath := metroMapPath + ".composed"
	err := h.executor.Run("", os.Stderr, os.Stderr, "node", composeScript, metroMapPath, hermesMapPath, "-o", composedPath)
	if err != nil {
		h.out.Warning("source map composition failed, using Hermes source map only")
		if err := os.Rename(hermesMapPath, metroMapPath); err != nil {
			h.out.Warning("could not rename Hermes source map: %v", err)
		}
		return
	}

	// Replace original sourcemap with composed one
	if err := os.Rename(composedPath, metroMapPath); err != nil {
		h.out.Warning("could not replace source map with composed version: %v", err)
	}
	if err := os.Remove(hermesMapPath); err != nil {
		h.out.Warning("could not clean up Hermes source map: %v", err)
	}
}

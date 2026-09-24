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
// If sourcemapPath is non-empty, hermesc runs next to the source map so its own
// map never lands in the bundle's directory (the update payload), then the two
// maps are composed using the compose script from projectDir.
// extraHermesFlags are appended to the hermesc invocation before the input file.
func (h *HermesCompiler) Compile(hermescPath, bundlePath, sourcemapPath, projectDir string, extraHermesFlags []string) error {
	if _, err := os.Stat(hermescPath); err != nil {
		return fmt.Errorf("hermesc binary not found at %s: %w", hermescPath, err)
	}

	if _, err := os.Stat(bundlePath); err != nil {
		return fmt.Errorf("bundle file not found at %s: %w", bundlePath, err)
	}

	hbcPath := bundlePath + ".hbc"
	if sourcemapPath != "" {
		hbcPath = filepath.Join(filepath.Dir(sourcemapPath), filepath.Base(bundlePath)+".hbc")
	}

	// Compile JS to Hermes bytecode
	args := []string{"-emit-binary", "-out", hbcPath}

	if sourcemapPath != "" {
		args = append(args, "-output-source-map")
	}

	args = append(args, extraHermesFlags...)
	args = append(args, bundlePath)

	h.out.Step("Running Hermes compilation: %s %v", hermescPath, args)

	if err := h.executor.Run("", os.Stderr, os.Stderr, hermescPath, args...); err != nil {
		return fmt.Errorf("hermes compilation failed: %w", err)
	}

	// Replace the original JS bundle with the compiled bytecode
	if err := moveFile(hbcPath, bundlePath); err != nil {
		return fmt.Errorf("replacing bundle with Hermes bytecode: %w", err)
	}

	// Compose source maps if both metro and hermes source maps exist
	if sourcemapPath != "" {
		hermesMapPath := hbcPath + ".map"
		if _, err := os.Stat(hermesMapPath); err == nil {
			h.composeSourceMaps(projectDir, sourcemapPath, hermesMapPath)
		}
	}

	return nil
}

// composeSourceMaps attempts to compose Metro and Hermes source maps into
// metroMapPath. This is a best-effort operation: on failure both maps are kept
// so they can be composed by hand, and a warning is logged.
func (h *HermesCompiler) composeSourceMaps(projectDir, metroMapPath, hermesMapPath string) {
	composeScript := filepath.Join(projectDir, "node_modules", "react-native", "scripts", "compose-source-maps.js")
	if _, err := os.Stat(composeScript); err != nil {
		h.warnUncomposed("compose-source-maps.js not found at "+composeScript, composeScript, metroMapPath, hermesMapPath)
		return
	}

	composedPath := metroMapPath + ".composed"
	err := h.executor.Run(projectDir, os.Stderr, os.Stderr, "node", composeScript, metroMapPath, hermesMapPath, "-o", composedPath)
	if err != nil {
		_ = os.Remove(composedPath)
		h.warnUncomposed("source map composition failed", composeScript, metroMapPath, hermesMapPath)
		return
	}

	// Replace original sourcemap with composed one
	if err := os.Rename(composedPath, metroMapPath); err != nil {
		h.warnUncomposed(fmt.Sprintf("could not replace source map with composed version: %v", err), composeScript, metroMapPath, hermesMapPath)
		return
	}
	if err := os.Remove(hermesMapPath); err != nil {
		h.out.Warning("could not clean up Hermes source map: %v", err)
	}
}

func (h *HermesCompiler) warnUncomposed(reason, composeScript, metroMapPath, hermesMapPath string) {
	h.out.Warning("%s. Kept the Metro source map at %s and the Hermes source map at %s; "+
		"compose them with: node %s %s %s -o <output>",
		reason, metroMapPath, hermesMapPath, composeScript, metroMapPath, hermesMapPath)
}

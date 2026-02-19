package bundler

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestHermesCompilerCompile(t *testing.T) {
	t.Run("successful compilation", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")

		executor := &mockExecutor{}
		// Simulate hermesc creating the .hbc file
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "-out" && i+1 < len(args) {
					os.WriteFile(args[i+1], []byte("bytecode"), 0o644)
				}
			}
		}

		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		err := compiler.Compile(hermescPath, bundlePath, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the command was called correctly
		if len(executor.commands) != 1 {
			t.Fatalf("expected 1 command, got %d", len(executor.commands))
		}

		cmd := executor.commands[0]
		if cmd.name != hermescPath {
			t.Errorf("command: got %q, want %q", cmd.name, hermescPath)
		}

		// Check args include -emit-binary
		foundEmitBinary := false
		for _, arg := range cmd.args {
			if arg == "-emit-binary" {
				foundEmitBinary = true
			}
		}
		if !foundEmitBinary {
			t.Error("-emit-binary flag not found in args")
		}

		// Verify the .hbc file was renamed to the original bundle path
		data, err := os.ReadFile(bundlePath)
		if err != nil {
			t.Fatalf("reading bundle: %v", err)
		}
		if string(data) != "bytecode" {
			t.Errorf("bundle content: got %q, want %q", string(data), "bytecode")
		}
	})

	t.Run("with sourcemap", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")
		sourcemapPath := filepath.Join(dir, "main.jsbundle.map")

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")
		writeFile(t, sourcemapPath, "{}")

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "-out" && i+1 < len(args) {
					os.WriteFile(args[i+1], []byte("bytecode"), 0o644)
				}
			}
		}

		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		err := compiler.Compile(hermescPath, bundlePath, sourcemapPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cmd := executor.commands[0]
		foundSourceMap := false
		for _, arg := range cmd.args {
			if arg == "-output-source-map" {
				foundSourceMap = true
			}
		}
		if !foundSourceMap {
			t.Error("-output-source-map flag not found when sourcemap path provided")
		}
	})

	t.Run("hermesc binary not found", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		writeFile(t, bundlePath, "console.log('hello')")

		executor := &mockExecutor{}
		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))

		err := compiler.Compile("/nonexistent/hermesc", bundlePath, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("bundle file not found", func(t *testing.T) {
		dir := t.TempDir()
		hermescPath := filepath.Join(dir, "hermesc")
		writeFile(t, hermescPath, "")

		executor := &mockExecutor{}
		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))

		err := compiler.Compile(hermescPath, "/nonexistent/bundle.js", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("hermesc execution fails", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")

		executor := &mockExecutor{err: &mockExitError{code: 1}}
		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))

		err := compiler.Compile(hermescPath, bundlePath, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("with sourcemap and hermes map triggers composition", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")
		sourcemapPath := filepath.Join(dir, "main.jsbundle.map")
		hbcPath := bundlePath + ".hbc"
		hermesMapPath := hbcPath + ".map"

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")
		writeFile(t, sourcemapPath, `{"version":3}`)

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "-out" && i+1 < len(args) {
					os.WriteFile(args[i+1], []byte("bytecode"), 0o644)
					// Also create the hermes source map
					os.WriteFile(args[i+1]+".map", []byte(`{"hermes":true}`), 0o644)
				}
			}
		}

		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		err := compiler.Compile(hermescPath, bundlePath, sourcemapPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The hermes map should have been renamed to the metro map path
		// since compose-source-maps.js won't be found
		data, err := os.ReadFile(sourcemapPath)
		if err != nil {
			t.Fatalf("reading sourcemap: %v", err)
		}
		if string(data) != `{"hermes":true}` {
			t.Errorf("sourcemap content: got %q, want hermes map content", string(data))
		}

		// The hermes map file should be gone
		if _, err := os.Stat(hermesMapPath); err == nil {
			t.Error("hermes map file should have been renamed away")
		}
	})
}

func TestComposeSourceMaps(t *testing.T) {
	t.Run("no compose script falls back to hermes map", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		metroMapPath := filepath.Join(dir, "metro.map")
		hermesMapPath := filepath.Join(dir, "hermes.map")

		writeFile(t, bundlePath, "bytecode")
		writeFile(t, metroMapPath, `{"metro":true}`)
		writeFile(t, hermesMapPath, `{"hermes":true}`)

		executor := &mockExecutor{}
		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		compiler.composeSourceMaps(bundlePath, metroMapPath, hermesMapPath)

		// Metro map should now contain hermes map content
		data, err := os.ReadFile(metroMapPath)
		if err != nil {
			t.Fatalf("reading map: %v", err)
		}
		if string(data) != `{"hermes":true}` {
			t.Errorf("map content: got %q, want hermes map", string(data))
		}
	})

	t.Run("compose script exists but execution fails", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		metroMapPath := filepath.Join(dir, "metro.map")
		hermesMapPath := filepath.Join(dir, "hermes.map")

		// Create the compose script path so it's found
		scriptDir := filepath.Join(dir, "node_modules", "react-native", "scripts")
		os.MkdirAll(scriptDir, 0o755)
		writeFile(t, filepath.Join(scriptDir, "compose-source-maps.js"), "")

		writeFile(t, bundlePath, "bytecode")
		writeFile(t, metroMapPath, `{"metro":true}`)
		writeFile(t, hermesMapPath, `{"hermes":true}`)

		executor := &mockExecutor{err: &mockExitError{code: 1}}
		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		compiler.composeSourceMaps(bundlePath, metroMapPath, hermesMapPath)

		// Should fall back to hermes map on failure
		data, err := os.ReadFile(metroMapPath)
		if err != nil {
			t.Fatalf("reading map: %v", err)
		}
		if string(data) != `{"hermes":true}` {
			t.Errorf("map content: got %q, want hermes map fallback", string(data))
		}
	})

	t.Run("compose script succeeds", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		metroMapPath := filepath.Join(dir, "metro.map")
		hermesMapPath := filepath.Join(dir, "hermes.map")

		// Create the compose script path
		scriptDir := filepath.Join(dir, "node_modules", "react-native", "scripts")
		os.MkdirAll(scriptDir, 0o755)
		writeFile(t, filepath.Join(scriptDir, "compose-source-maps.js"), "")

		writeFile(t, bundlePath, "bytecode")
		writeFile(t, metroMapPath, `{"metro":true}`)
		writeFile(t, hermesMapPath, `{"hermes":true}`)

		executor := &mockExecutor{}
		// Simulate compose-source-maps creating the composed file
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "-o" && i+1 < len(args) {
					os.WriteFile(args[i+1], []byte(`{"composed":true}`), 0o644)
				}
			}
		}

		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		compiler.composeSourceMaps(bundlePath, metroMapPath, hermesMapPath)

		// Metro map should have composed content
		data, err := os.ReadFile(metroMapPath)
		if err != nil {
			t.Fatalf("reading map: %v", err)
		}
		if string(data) != `{"composed":true}` {
			t.Errorf("map content: got %q, want composed map", string(data))
		}

		// Hermes map should be cleaned up
		if _, err := os.Stat(hermesMapPath); err == nil {
			t.Error("hermes map should have been removed after composition")
		}
	})
}

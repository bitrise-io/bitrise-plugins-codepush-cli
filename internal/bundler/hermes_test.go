package bundler

import (
	"os"
	"path/filepath"
	"testing"
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

		compiler := NewHermesCompiler(executor)
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

		compiler := NewHermesCompiler(executor)
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
		compiler := NewHermesCompiler(executor)

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
		compiler := NewHermesCompiler(executor)

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
		compiler := NewHermesCompiler(executor)

		err := compiler.Compile(hermescPath, bundlePath, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

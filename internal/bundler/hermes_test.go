package bundler

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		err := compiler.Compile(&CompileOptions{HermescPath: hermescPath, BundlePath: bundlePath, ProjectDir: dir})
		require.NoError(t, err)

		// Verify the command was called correctly
		require.Len(t, executor.commands, 1)

		cmd := executor.commands[0]
		assert.Equal(t, hermescPath, cmd.name)

		// Check args include -emit-binary
		assert.Contains(t, cmd.args, "-emit-binary")

		// Verify the .hbc file was renamed to the original bundle path
		data, err := os.ReadFile(bundlePath)
		require.NoError(t, err)
		assert.Equal(t, "bytecode", string(data))
	})

	t.Run("with sourcemap runs hermesc next to the source map", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "CodePush")
		mapDir := filepath.Join(dir, "CodePush.sourcemaps")
		require.NoError(t, os.MkdirAll(outputDir, 0o755))
		require.NoError(t, os.MkdirAll(mapDir, 0o755))
		bundlePath := filepath.Join(outputDir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")
		sourcemapPath := filepath.Join(mapDir, "main.jsbundle.map")

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")
		writeFile(t, sourcemapPath, "{}")

		executor := &mockExecutor{}
		executor.onRun = writeHermesOutputs

		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		err := compiler.Compile(&CompileOptions{HermescPath: hermescPath, BundlePath: bundlePath, SourcemapPath: sourcemapPath, ProjectDir: dir})
		require.NoError(t, err)

		cmd := executor.commands[0]
		assert.Contains(t, cmd.args, "-output-source-map")
		assertContainsArgs(t, cmd.args, "-out", filepath.Join(mapDir, "main.jsbundle.hbc"))

		data, err := os.ReadFile(bundlePath)
		require.NoError(t, err)
		assert.Equal(t, "bytecode", string(data))
		assertOnlyFiles(t, outputDir, "main.jsbundle")
	})

	t.Run("extra hermes flags are passed before the input file", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")

		executor := &mockExecutor{}
		executor.onRun = func(_ string, _ string, args ...string) {
			for i, arg := range args {
				if arg == "-out" && i+1 < len(args) {
					os.WriteFile(args[i+1], []byte("bytecode"), 0o644)
				}
			}
		}

		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		err := compiler.Compile(&CompileOptions{HermescPath: hermescPath, BundlePath: bundlePath, ProjectDir: dir, ExtraFlags: []string{"-O", "-w"}})
		require.NoError(t, err)

		cmd := executor.commands[0]
		// Extra flags must appear before the input file
		inputIdx := -1
		oIdx := -1
		wIdx := -1
		for i, arg := range cmd.args {
			switch arg {
			case bundlePath:
				inputIdx = i
			case "-O":
				oIdx = i
			case "-w":
				wIdx = i
			}
		}
		require.NotEqual(t, -1, oIdx, "-O flag missing")
		require.NotEqual(t, -1, wIdx, "-w flag missing")
		assert.Less(t, oIdx, inputIdx, "-O must come before input file")
		assert.Less(t, wIdx, inputIdx, "-w must come before input file")
	})

	t.Run("hermesc binary not found", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		writeFile(t, bundlePath, "console.log('hello')")

		executor := &mockExecutor{}
		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))

		err := compiler.Compile(&CompileOptions{HermescPath: "/nonexistent/hermesc", BundlePath: bundlePath, ProjectDir: dir})
		require.Error(t, err)
	})

	t.Run("bundle file not found", func(t *testing.T) {
		dir := t.TempDir()
		hermescPath := filepath.Join(dir, "hermesc")
		writeFile(t, hermescPath, "")

		executor := &mockExecutor{}
		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))

		err := compiler.Compile(&CompileOptions{HermescPath: hermescPath, BundlePath: "/nonexistent/bundle.js", ProjectDir: dir})
		require.Error(t, err)
	})

	t.Run("hermesc execution fails", func(t *testing.T) {
		dir := t.TempDir()
		bundlePath := filepath.Join(dir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")

		executor := &mockExecutor{err: &mockExitError{code: 1}}
		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))

		err := compiler.Compile(&CompileOptions{HermescPath: hermescPath, BundlePath: bundlePath, ProjectDir: dir})
		require.Error(t, err)
	})

	t.Run("without compose script keeps both maps outside the output directory", func(t *testing.T) {
		dir := t.TempDir()
		projectDir := filepath.Join(dir, "project")
		outputDir := filepath.Join(dir, "build", "CodePush")
		mapDir := filepath.Join(dir, "build", "CodePush.sourcemaps")
		require.NoError(t, os.MkdirAll(projectDir, 0o755))
		require.NoError(t, os.MkdirAll(outputDir, 0o755))
		require.NoError(t, os.MkdirAll(mapDir, 0o755))
		bundlePath := filepath.Join(outputDir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")
		sourcemapPath := filepath.Join(mapDir, "main.jsbundle.map")

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")
		writeFile(t, sourcemapPath, `{"metro":true}`)

		executor := &mockExecutor{}
		executor.onRun = writeHermesOutputs

		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		err := compiler.Compile(&CompileOptions{HermescPath: hermescPath, BundlePath: bundlePath, SourcemapPath: sourcemapPath, ProjectDir: projectDir})
		require.NoError(t, err)

		data, err := os.ReadFile(sourcemapPath)
		require.NoError(t, err)
		assert.Equal(t, `{"metro":true}`, string(data), "metro map must not be overwritten")

		data, err = os.ReadFile(filepath.Join(mapDir, "main.jsbundle.hbc.map"))
		require.NoError(t, err)
		assert.Equal(t, `{"hermes":true}`, string(data))

		assertOnlyFiles(t, outputDir, "main.jsbundle")
	})

	t.Run("composes with the script from the project directory", func(t *testing.T) {
		dir := t.TempDir()
		projectDir := filepath.Join(dir, "project")
		outputDir := filepath.Join(dir, "build", "CodePush")
		mapDir := filepath.Join(dir, "build", "CodePush.sourcemaps")
		scriptDir := filepath.Join(projectDir, "node_modules", "react-native", "scripts")
		require.NoError(t, os.MkdirAll(scriptDir, 0o755))
		require.NoError(t, os.MkdirAll(outputDir, 0o755))
		require.NoError(t, os.MkdirAll(mapDir, 0o755))
		composeScript := filepath.Join(scriptDir, "compose-source-maps.js")
		writeFile(t, composeScript, "")
		bundlePath := filepath.Join(outputDir, "main.jsbundle")
		hermescPath := filepath.Join(dir, "hermesc")
		sourcemapPath := filepath.Join(mapDir, "main.jsbundle.map")

		writeFile(t, bundlePath, "console.log('hello')")
		writeFile(t, hermescPath, "")
		writeFile(t, sourcemapPath, `{"metro":true}`)

		executor := &mockExecutor{}
		executor.onRun = func(dir string, name string, args ...string) {
			writeHermesOutputs(dir, name, args...)
			writeComposeOutput(dir, name, args...)
		}

		compiler := NewHermesCompiler(executor, output.NewTest(io.Discard))
		err := compiler.Compile(&CompileOptions{HermescPath: hermescPath, BundlePath: bundlePath, SourcemapPath: sourcemapPath, ProjectDir: projectDir})
		require.NoError(t, err)

		require.Len(t, executor.commands, 2)
		compose := executor.commands[1]
		assert.Equal(t, "node", compose.name)
		assert.Equal(t, projectDir, compose.dir)
		assert.Equal(t, composeScript, compose.args[0])

		data, err := os.ReadFile(sourcemapPath)
		require.NoError(t, err)
		assert.Equal(t, `{"composed":true}`, string(data))
		assertOnlyFiles(t, mapDir, "main.jsbundle.map")
		assertOnlyFiles(t, outputDir, "main.jsbundle")
	})
}

func writeHermesOutputs(_ string, _ string, args ...string) {
	withMap := false
	for _, arg := range args {
		if arg == "-output-source-map" {
			withMap = true
		}
	}
	for i, arg := range args {
		if arg == "-out" && i+1 < len(args) {
			os.WriteFile(args[i+1], []byte("bytecode"), 0o644)
			if withMap {
				os.WriteFile(args[i+1]+".map", []byte(`{"hermes":true}`), 0o644)
			}
		}
	}
}

func writeComposeOutput(_ string, name string, args ...string) {
	if name != "node" {
		return
	}
	for i, arg := range args {
		if arg == "-o" && i+1 < len(args) {
			os.WriteFile(args[i+1], []byte(`{"composed":true}`), 0o644)
		}
	}
}

func assertOnlyFiles(t *testing.T, dir string, names ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	assert.ElementsMatch(t, names, got)
}

func TestComposeSourceMaps(t *testing.T) {
	tests := []struct {
		name        string
		withScript  bool
		execErr     error
		wantMetro   string // content of the metro map path afterwards
		wantHermes  bool   // hermes map still exists
		wantWarning string
	}{
		{
			name:        "no compose script keeps both maps",
			wantMetro:   `{"metro":true}`,
			wantHermes:  true,
			wantWarning: "compose-source-maps.js not found",
		},
		{
			name:        "compose script fails keeps both maps",
			withScript:  true,
			execErr:     &mockExitError{code: 1},
			wantMetro:   `{"metro":true}`,
			wantHermes:  true,
			wantWarning: "source map composition failed",
		},
		{
			name:       "compose script succeeds",
			withScript: true,
			wantMetro:  `{"composed":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir := t.TempDir()
			mapDir := t.TempDir()
			metroMapPath := filepath.Join(mapDir, "main.jsbundle.map")
			hermesMapPath := filepath.Join(mapDir, "main.jsbundle.hbc.map")
			writeFile(t, metroMapPath, `{"metro":true}`)
			writeFile(t, hermesMapPath, `{"hermes":true}`)
			if tt.withScript {
				scriptDir := filepath.Join(projectDir, "node_modules", "react-native", "scripts")
				require.NoError(t, os.MkdirAll(scriptDir, 0o755))
				writeFile(t, filepath.Join(scriptDir, "compose-source-maps.js"), "")
			}

			var buf bytes.Buffer
			executor := &mockExecutor{err: tt.execErr, onRun: writeComposeOutput}
			compiler := NewHermesCompiler(executor, output.NewTest(&buf))
			compiler.composeSourceMaps(projectDir, metroMapPath, hermesMapPath)

			data, err := os.ReadFile(metroMapPath)
			require.NoError(t, err)
			assert.Equal(t, tt.wantMetro, string(data))

			_, err = os.Stat(hermesMapPath)
			assert.Equal(t, tt.wantHermes, err == nil, "hermes map exists")

			_, err = os.Stat(metroMapPath + ".composed")
			assert.True(t, os.IsNotExist(err), "no leftover composed map")

			if tt.wantWarning != "" {
				assert.Contains(t, buf.String(), tt.wantWarning)
				assert.Contains(t, buf.String(), hermesMapPath, "warning includes the manual compose command")
			} else {
				assert.Empty(t, buf.String())
			}
			for _, cmd := range executor.commands {
				assert.Equal(t, projectDir, cmd.dir, "compose runs in the project directory")
			}
		})
	}
}

func TestHermesWorkPath(t *testing.T) {
	tests := []struct {
		sourcemapPath string
		wantHbc       string
	}{
		{"", "/out/main.jsbundle.hbc"},
		{"/maps/main.jsbundle.map", "/maps/main.jsbundle.hbc"},
		// Custom names that would collide with a "<bundle>.hbc[.map]" scheme.
		{"/maps/main.jsbundle.hbc.map", "/maps/main.jsbundle.hbc.hbc"},
		{"/maps/main.jsbundle.hbc", "/maps/main.jsbundle.hbc.hbc"},
		{"/maps/app", "/maps/app.hbc"},
	}
	for _, tt := range tests {
		got := hermesWorkPath("/out/main.jsbundle", tt.sourcemapPath)
		assert.Equal(t, tt.wantHbc, got, tt.sourcemapPath)
		if tt.sourcemapPath != "" {
			assert.NotEqual(t, tt.sourcemapPath, got, "bytecode must not overwrite the source map")
			assert.NotEqual(t, tt.sourcemapPath, got+".map", "hermes map must not overwrite the source map")
		}
	}
}

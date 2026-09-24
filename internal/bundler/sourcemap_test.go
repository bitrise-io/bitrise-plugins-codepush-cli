package bundler

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestResolveSourcemapPath(t *testing.T) {
	t.Run("disabled returns empty path", func(t *testing.T) {
		dir := t.TempDir()
		path, err := resolveSourcemapPath(&BundleOptions{}, filepath.Join(dir, "CodePush"), "main.jsbundle")
		require.NoError(t, err)
		assert.Empty(t, path)

		_, err = os.Stat(filepath.Join(dir, "CodePush"+SourcemapDirSuffix))
		assert.True(t, os.IsNotExist(err), "no source map directory when source maps are disabled")
	})

	t.Run("default is next to the output directory", func(t *testing.T) {
		dir := t.TempDir()
		outputDir := filepath.Join(dir, "CodePush")

		path, err := resolveSourcemapPath(&BundleOptions{Sourcemap: true}, outputDir, "main.jsbundle")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "CodePush.sourcemaps", "main.jsbundle.map"), path)
		assert.DirExists(t, filepath.Dir(path))
	})

	t.Run("relative explicit path resolves against the project directory", func(t *testing.T) {
		projectDir := t.TempDir()
		opts := &BundleOptions{SourcemapOutput: "maps/app.map", ProjectDir: projectDir}

		path, err := resolveSourcemapPath(opts, filepath.Join(projectDir, "CodePush"), "main.jsbundle")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(projectDir, "maps", "app.map"), path)
	})

	t.Run("explicit path inside the output directory is rejected", func(t *testing.T) {
		projectDir := t.TempDir()
		outputDir := filepath.Join(projectDir, "CodePush")
		for _, output := range []string{"CodePush/main.jsbundle.map", "CodePush/maps/main.jsbundle.map"} {
			opts := &BundleOptions{SourcemapOutput: output, ProjectDir: projectDir}
			_, err := resolveSourcemapPath(opts, outputDir, "main.jsbundle")
			require.Error(t, err, output)
			assert.Contains(t, err.Error(), "inside the output directory")
		}
	})
}

func TestIsWithinDir(t *testing.T) {
	dir := filepath.Join("/project", "CodePush")
	tests := []struct {
		path string
		want bool
	}{
		{filepath.Join(dir, "main.jsbundle.map"), true},
		{filepath.Join(dir, "a", "b.map"), true},
		{dir, true},
		// Sibling with a shared name prefix: a string prefix check gets this wrong.
		{filepath.Join("/project", "CodePush.sourcemaps", "main.jsbundle.map"), false},
		{filepath.Join("/project", "main.jsbundle.map"), false},
		{filepath.Join("/elsewhere", "main.jsbundle.map"), false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isWithinDir(tt.path, dir), tt.path)
	}
}

func TestFindSourcemaps(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	writeFile(t, filepath.Join(dir, "main.jsbundle"), "bundle")
	writeFile(t, filepath.Join(dir, "main.jsbundle.map"), "{}")
	writeFile(t, filepath.Join(dir, "assets", "nested.map"), "{}")
	writeFile(t, filepath.Join(dir, "assets", "image.png"), "png")

	maps, err := FindSourcemaps(dir)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"main.jsbundle.map", filepath.Join("assets", "nested.map")}, maps)

	clean := t.TempDir()
	writeFile(t, filepath.Join(clean, "main.jsbundle"), "bundle")
	maps, err = FindSourcemaps(clean)
	require.NoError(t, err)
	assert.Empty(t, maps)
}

func TestMoveFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	writeFile(t, src, "content")

	require.NoError(t, moveFile(src, dst))

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))

	assert.Error(t, moveFile(filepath.Join(dir, "missing"), dst))
}

func TestBundleRejectsSourcemapInsideOutputDir(t *testing.T) {
	projectDir := t.TempDir()
	outputDir := filepath.Join(projectDir, "CodePush")
	opts := &BundleOptions{
		Platform:        PlatformIOS,
		OutputDir:       outputDir,
		ProjectDir:      projectDir,
		SourcemapOutput: "CodePush/main.jsbundle.map",
	}
	config := &ProjectConfig{ProjectDir: projectDir, Platform: PlatformIOS, EntryFile: "index.js"}

	for name, b := range map[string]Bundler{
		"react-native": &ReactNativeBundler{executor: &mockExecutor{}, out: output.NewTest(io.Discard)},
		"expo":         &ExpoBundler{executor: &mockExecutor{}, out: output.NewTest(io.Discard)},
	} {
		_, err := b.Bundle(config, opts)
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "inside the output directory", name)
		assert.NoDirExists(t, outputDir, "%s: nothing is created before the check", name)
	}
}

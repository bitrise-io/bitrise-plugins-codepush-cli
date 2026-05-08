package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	configDirFunc = func() (string, error) { return dir, nil }
	t.Cleanup(func() { configDirFunc = defaultConfigDir })
	return dir
}

func TestLoad(t *testing.T) {
	t.Run("returns nil when file does not exist", func(t *testing.T) {
		setupTestDir(t)

		cfg, err := Load()
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("returns config with valid JSON", func(t *testing.T) {
		dir := setupTestDir(t)
		os.WriteFile(filepath.Join(dir, FileName), []byte(`{"app_id":"abc-123"}`), 0o644)

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "abc-123", cfg.AppID)
		assert.Empty(t, cfg.ServerURL)
	})

	t.Run("returns config with server URL", func(t *testing.T) {
		dir := setupTestDir(t)
		os.WriteFile(filepath.Join(dir, FileName), []byte(`{"app_id":"abc-123","server_url":"https://api.staging.bitrise.io"}`), 0o644)

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "abc-123", cfg.AppID)
		assert.Equal(t, "https://api.staging.bitrise.io", cfg.ServerURL)
	})

	t.Run("returns error for malformed JSON", func(t *testing.T) {
		dir := setupTestDir(t)
		os.WriteFile(filepath.Join(dir, FileName), []byte(`{not json}`), 0o644)

		_, err := Load()
		require.Error(t, err)
	})

	t.Run("returns error when directory resolution fails", func(t *testing.T) {
		configDirFunc = func() (string, error) { return "", os.ErrNotExist }
		t.Cleanup(func() { configDirFunc = defaultConfigDir })

		_, err := Load()
		require.Error(t, err)
	})
}

func TestSave(t *testing.T) {
	t.Run("save and load round-trip", func(t *testing.T) {
		dir := setupTestDir(t)

		want := &ProjectConfig{AppID: "round-trip-id"}
		require.NoError(t, Save(dir, want))

		got, err := Load()
		require.NoError(t, err)
		assert.Equal(t, want.AppID, got.AppID)
	})

	t.Run("round-trip with server URL", func(t *testing.T) {
		dir := setupTestDir(t)

		want := &ProjectConfig{AppID: "my-app", ServerURL: "https://api.staging.bitrise.io"}
		require.NoError(t, Save(dir, want))

		got, err := Load()
		require.NoError(t, err)
		assert.Equal(t, want.AppID, got.AppID)
		assert.Equal(t, want.ServerURL, got.ServerURL)
	})

	t.Run("omits server URL when empty", func(t *testing.T) {
		dir := setupTestDir(t)

		require.NoError(t, Save(dir, &ProjectConfig{AppID: "my-app"}))

		data, err := os.ReadFile(filepath.Join(dir, FileName))
		require.NoError(t, err)
		assert.NotContains(t, string(data), "server_url")
	})

	t.Run("file has 0644 permissions", func(t *testing.T) {
		dir := setupTestDir(t)

		require.NoError(t, Save(dir, &ProjectConfig{AppID: "test"}))

		info, err := os.Stat(filepath.Join(dir, FileName))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	})

	t.Run("overwrites existing config", func(t *testing.T) {
		dir := setupTestDir(t)

		Save(dir, &ProjectConfig{AppID: "first"})
		Save(dir, &ProjectConfig{AppID: "second"})

		got, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "second", got.AppID)
	})
}

func TestProgressStyleRoundTrip(t *testing.T) {
	t.Run("saves and loads progress_style", func(t *testing.T) {
		dir := setupTestDir(t)

		want := &ProjectConfig{AppID: "my-app", ProgressStyle: "spinner"}
		require.NoError(t, Save(dir, want))

		got, err := Load()
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "spinner", got.ProgressStyle)
	})

	t.Run("omits progress_style when empty", func(t *testing.T) {
		dir := setupTestDir(t)

		require.NoError(t, Save(dir, &ProjectConfig{AppID: "my-app"}))

		data, err := os.ReadFile(filepath.Join(dir, FileName))
		require.NoError(t, err)
		assert.NotContains(t, string(data), "progress_style")
	})

	t.Run("loads unknown progress_style without error", func(t *testing.T) {
		dir := setupTestDir(t)
		os.WriteFile(filepath.Join(dir, FileName), []byte(`{"app_id":"x","progress_style":"rainbow"}`), 0o644)

		cfg, err := Load()
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "rainbow", cfg.ProgressStyle)
	})
}

func TestFilePath(t *testing.T) {
	dir := setupTestDir(t)

	got, err := FilePath()
	require.NoError(t, err)

	want := filepath.Join(dir, FileName)
	assert.Equal(t, want, got)
}

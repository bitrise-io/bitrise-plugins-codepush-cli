package config

import (
	"os"
	"path/filepath"
	"testing"
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg != nil {
			t.Errorf("expected nil config, got %+v", cfg)
		}
	})

	t.Run("returns config with valid JSON", func(t *testing.T) {
		dir := setupTestDir(t)
		os.WriteFile(filepath.Join(dir, FileName), []byte(`{"app_id":"abc-123"}`), 0o644)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected config, got nil")
		}
		if cfg.AppID != "abc-123" {
			t.Errorf("app_id: got %q, want %q", cfg.AppID, "abc-123")
		}
	})

	t.Run("returns error for malformed JSON", func(t *testing.T) {
		dir := setupTestDir(t)
		os.WriteFile(filepath.Join(dir, FileName), []byte(`{not json}`), 0o644)

		_, err := Load()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when directory resolution fails", func(t *testing.T) {
		configDirFunc = func() (string, error) { return "", os.ErrNotExist }
		t.Cleanup(func() { configDirFunc = defaultConfigDir })

		_, err := Load()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("save and load round-trip", func(t *testing.T) {
		dir := setupTestDir(t)

		want := &ProjectConfig{AppID: "round-trip-id"}
		if err := Save(dir, want); err != nil {
			t.Fatalf("Save: %v", err)
		}

		got, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.AppID != want.AppID {
			t.Errorf("app_id: got %q, want %q", got.AppID, want.AppID)
		}
	})

	t.Run("file has 0644 permissions", func(t *testing.T) {
		dir := setupTestDir(t)

		if err := Save(dir, &ProjectConfig{AppID: "test"}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		info, err := os.Stat(filepath.Join(dir, FileName))
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o644 {
			t.Errorf("permissions: got %o, want 644", perm)
		}
	})

	t.Run("overwrites existing config", func(t *testing.T) {
		dir := setupTestDir(t)

		Save(dir, &ProjectConfig{AppID: "first"})
		Save(dir, &ProjectConfig{AppID: "second"})

		got, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.AppID != "second" {
			t.Errorf("app_id: got %q, want %q", got.AppID, "second")
		}
	})
}

func TestFilePath(t *testing.T) {
	dir := setupTestDir(t)

	got, err := FilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(dir, FileName)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

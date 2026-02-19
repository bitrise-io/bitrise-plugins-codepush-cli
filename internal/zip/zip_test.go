package zip

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDirectory(t *testing.T) {
	t.Run("zips files correctly", func(t *testing.T) {
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "bundle")
		os.Mkdir(srcDir, 0o755)

		writeFile(t, filepath.Join(srcDir, "main.jsbundle"), "bundle content")
		writeFile(t, filepath.Join(srcDir, "main.jsbundle.map"), "sourcemap content")

		zipPath, err := Directory(srcDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(zipPath)

		if zipPath != srcDir+".zip" {
			t.Errorf("zip path: got %q, want %q", zipPath, srcDir+".zip")
		}

		entries := readZipEntries(t, zipPath)
		if len(entries) != 2 {
			t.Fatalf("zip entries: got %d, want 2", len(entries))
		}

		sort.Strings(entries)
		if entries[0] != "main.jsbundle" {
			t.Errorf("entry[0]: got %q, want %q", entries[0], "main.jsbundle")
		}
		if entries[1] != "main.jsbundle.map" {
			t.Errorf("entry[1]: got %q, want %q", entries[1], "main.jsbundle.map")
		}
	})

	t.Run("preserves nested directory structure", func(t *testing.T) {
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "bundle")
		os.MkdirAll(filepath.Join(srcDir, "assets", "images"), 0o755)

		writeFile(t, filepath.Join(srcDir, "index.js"), "code")
		writeFile(t, filepath.Join(srcDir, "assets", "images", "logo.png"), "image data")

		zipPath, err := Directory(srcDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(zipPath)

		entries := readZipEntries(t, zipPath)
		sort.Strings(entries)

		expected := []string{"assets/", "assets/images/", "assets/images/logo.png", "index.js"}
		if len(entries) != len(expected) {
			t.Fatalf("zip entries: got %v, want %v", entries, expected)
		}
		for i, e := range expected {
			if entries[i] != e {
				t.Errorf("entry[%d]: got %q, want %q", i, entries[i], e)
			}
		}
	})

	t.Run("preserves file contents", func(t *testing.T) {
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "bundle")
		os.Mkdir(srcDir, 0o755)

		content := "console.log('hello world')"
		writeFile(t, filepath.Join(srcDir, "app.js"), content)

		zipPath, err := Directory(srcDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(zipPath)

		r, err := zip.OpenReader(zipPath)
		if err != nil {
			t.Fatalf("opening zip: %v", err)
		}
		defer r.Close()

		for _, f := range r.File {
			if f.Name == "app.js" {
				rc, err := f.Open()
				if err != nil {
					t.Fatalf("opening entry: %v", err)
				}
				defer rc.Close()

				buf := make([]byte, len(content)+10)
				n, _ := rc.Read(buf)
				if string(buf[:n]) != content {
					t.Errorf("content: got %q, want %q", string(buf[:n]), content)
				}
				return
			}
		}
		t.Error("app.js not found in zip")
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := Directory("/nonexistent/path")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("source is a file not directory", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "notadir")
		writeFile(t, filePath, "content")

		_, err := Directory(filePath)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		srcDir := filepath.Join(dir, "empty")
		os.Mkdir(srcDir, 0o755)

		zipPath, err := Directory(srcDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(zipPath)

		entries := readZipEntries(t, zipPath)
		if len(entries) != 0 {
			t.Errorf("zip entries: got %d, want 0", len(entries))
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing file %s: %v", path, err)
	}
}

func readZipEntries(t *testing.T, zipPath string) []string {
	t.Helper()
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("opening zip %s: %v", zipPath, err)
	}
	defer r.Close()

	var entries []string
	for _, f := range r.File {
		entries = append(entries, f.Name)
	}
	return entries
}

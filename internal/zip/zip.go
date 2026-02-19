// Package zip provides utilities for creating zip archives from directories.
package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Directory creates a zip archive from the contents of srcDir.
// The zip file is created as a sibling to srcDir with a .zip extension.
// Returns the path to the created zip file.
func Directory(srcDir string) (string, error) {
	absDir, err := filepath.Abs(srcDir)
	if err != nil {
		return "", fmt.Errorf("resolving directory path: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("source directory does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("source path is not a directory: %s", absDir)
	}

	zipPath := absDir + ".zip"
	f, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("creating zip file: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		if relPath == "." {
			return nil
		}

		// Zip spec requires forward slashes
		zipEntryName := filepath.ToSlash(relPath)

		if info.IsDir() {
			_, err := w.Create(zipEntryName + "/")
			return err
		}

		writer, err := w.Create(zipEntryName)
		if err != nil {
			return fmt.Errorf("creating zip entry %s: %w", zipEntryName, err)
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", path, err)
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("adding files to zip: %w", err)
	}

	return zipPath, nil
}

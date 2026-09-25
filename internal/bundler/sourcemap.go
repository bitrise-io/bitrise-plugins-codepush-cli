package bundler

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SourcemapDirSuffix keeps source maps out of the output directory, which
// ships to devices as the update payload.
const SourcemapDirSuffix = ".sourcemaps"

func resolveSourcemapPath(opts *BundleOptions, outputDir, bundleName string) (string, error) {
	// A pass-through --sourcemap-output comes after the CLI's own flag, so the
	// bundler would use it instead of the path checked below.
	for _, opt := range opts.ExtraBundlerOpts {
		if strings.HasPrefix(opt, "--sourcemap-output") {
			return "", errors.New("--extra-bundler-option cannot set --sourcemap-output: " +
				"use --sourcemap-output (-s), which keeps the source map out of the update payload")
		}
	}

	if !opts.Sourcemap && opts.SourcemapOutput == "" {
		return "", nil
	}

	mapPath := filepath.Join(outputDir+SourcemapDirSuffix, bundleName+".map")
	if opts.SourcemapOutput != "" {
		mapPath = opts.SourcemapOutput
		if !filepath.IsAbs(mapPath) {
			mapPath = filepath.Join(opts.ProjectDir, mapPath)
		}
	}

	if isWithinDir(mapPath, outputDir) {
		return "", fmt.Errorf("sourcemap output %s is inside the output directory %s: "+
			"everything in the output directory ships to devices in the update payload, "+
			"choose a --sourcemap-output path outside it", mapPath, outputDir)
	}

	if err := ensureDir(filepath.Dir(mapPath)); err != nil {
		return "", fmt.Errorf("creating sourcemap output directory: %w", err)
	}
	return mapPath, nil
}

// Symlinks and case differences (on case-insensitive filesystems) must not
// hide a path inside dir.
func isWithinDir(path, dir string) bool {
	path, dir = resolveExisting(path), resolveExisting(dir)
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		path, dir = strings.ToLower(path), strings.ToLower(dir)
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Only a prefix of the path can be resolved: the output directory and the
// source map usually do not exist yet when they are checked.
func resolveExisting(path string) string {
	path = filepath.Clean(path)
	var rest []string
	for cur := path; ; cur = filepath.Dir(cur) {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(append([]string{resolved}, rest...)...)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return path
		}
		rest = append([]string{filepath.Base(cur)}, rest...)
	}
}

func FindSourcemaps(dir string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".map" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		found = append(found, rel)
		return nil
	})
	return found, err
}

// Falls back to copying when rename fails (for example across filesystems),
// through a temporary file so dst is never left partially written.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op after a successful rename

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if info, err := in.Stat(); err == nil {
		_ = os.Chmod(tmpPath, info.Mode().Perm())
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

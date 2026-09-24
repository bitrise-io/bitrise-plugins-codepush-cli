package bundler

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SourcemapDirSuffix is appended to the output directory to get the default
// source map directory. Source maps must stay out of the output directory:
// everything in it is zipped into the update payload and shipped to devices.
const SourcemapDirSuffix = ".sourcemaps"

// resolveSourcemapPath returns the absolute source map path, or an empty string
// when source maps are disabled. The default is
// <outputDir>.sourcemaps/<bundleName>.map; an explicit SourcemapOutput is
// resolved against ProjectDir. Either way, the path must be outside outputDir.
func resolveSourcemapPath(opts *BundleOptions, outputDir, bundleName string) (string, error) {
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

// isWithinDir reports whether path is dir itself or inside it.
func isWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// FindSourcemaps returns the paths of all *.map files under dir, relative to dir.
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

// moveFile renames src to dst, falling back to copy and remove when a rename is
// not possible (for example across filesystems).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}

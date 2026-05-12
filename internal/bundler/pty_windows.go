//go:build windows

package bundler

import "io"

// runWithPTY falls back to the standard executor on Windows where PTY is not available.
func runWithPTY(dir string, w io.Writer, name string, args ...string) error {
	ex := &DefaultExecutor{}
	return ex.Run(dir, io.Discard, w, name, args...)
}

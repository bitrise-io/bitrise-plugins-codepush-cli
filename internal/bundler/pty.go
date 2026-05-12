//go:build !windows

package bundler

import (
	"io"
	"os/exec"

	"github.com/creack/pty"
)

// runWithPTY starts name with a pseudo-terminal as its controlling terminal so
// that TTY-aware tools (e.g. Metro bundler) emit their interactive progress
// output. stdout and stderr of the subprocess are merged on the PTY master and
// copied to w. EIO on the master read is treated as normal EOF.
func runWithPTY(dir string, w io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 50, Cols: 200})
	if err != nil {
		return err
	}
	defer func() { _ = ptmx.Close() }()

	// Copy PTY output; EIO is expected when the slave closes — treat as EOF.
	_, _ = io.Copy(w, ptmx)

	return cmd.Wait()
}

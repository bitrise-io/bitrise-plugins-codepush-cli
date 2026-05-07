// Package output provides styled terminal output with automatic CI and
// terminal capability detection. All human-readable CLI output should use
// this package instead of writing directly to os.Stderr.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

// Writer provides styled terminal output. Create one with New() for
// production use or NewTest() for tests.
type Writer struct {
	mu          sync.Mutex
	w           io.Writer
	interactive bool     // terminal AND not CI
	color       bool     // terminal AND not NO_COLOR
	barStyle    BarStyle // default StyleBar (zero value)
}

// KeyValue is a key-value pair for Result output.
type KeyValue struct {
	Key   string
	Value string
}

// StepHandle is returned by StartStep and lets the caller mark the step
// as completed. In interactive mode Done replaces the "->" line with "OK".
type StepHandle struct {
	write       func([]byte)
	interactive bool
	color       bool
	label       string
}

// Cancel is the error-path counterpart to Done. Unlike ProgressBar.Cancel it
// is always a no-op: StartStep writes the "-> label\n" line immediately, so
// the terminal line is already complete. Cancel exists purely for call-site
// symmetry so error paths can mirror success paths.
func (sh *StepHandle) Cancel() {}

// New creates a Writer that writes to stderr with auto-detected capabilities.
func New() *Writer {
	return NewWriter(os.Stderr)
}

// NewWriter creates a Writer targeting the given writer. Terminal capability
// is detected via Fd() if the writer supports it.
func NewWriter(w io.Writer) *Writer {
	isTerm := false
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		isTerm = term.IsTerminal(int(f.Fd()))
	}

	isCI := os.Getenv("CI") != "" || os.Getenv("BITRISE_BUILD_NUMBER") != ""
	noColor := os.Getenv("NO_COLOR") != ""

	return &Writer{
		w:           w,
		interactive: isTerm && !isCI,
		color:       isTerm && !noColor,
	}
}

// NewTest creates a Writer with no color and non-interactive mode.
func NewTest(w io.Writer) *Writer {
	return &Writer{
		w:           w,
		interactive: false,
		color:       false,
	}
}

// IsInteractive returns true if the writer targets an interactive terminal
// (not CI, not piped).
func (w *Writer) IsInteractive() bool {
	return w.interactive
}

// StartStep prints a progress step and returns a StepHandle. In interactive
// mode, calling Done on the handle replaces the "->" line with "OK" using
// cursor-up. In non-interactive mode Done is a no-op.
func (w *Writer) StartStep(format string, args ...any) *StepHandle {
	label := fmt.Sprintf(format, args...)
	w.Step("%s", label)
	return &StepHandle{
		write:       w.write,
		interactive: w.interactive,
		color:       w.color,
		label:       label,
	}
}

// Done replaces the step line with a green "OK label" line using cursor-up.
// No-op in non-interactive mode.
func (sh *StepHandle) Done() {
	if !sh.interactive {
		return
	}
	var ok string
	if sh.color {
		ok = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("OK")
	} else {
		ok = "OK"
	}
	sh.write(fmt.Appendf(nil, "\033[1A\r\033[2K%s %s\n", ok, sh.label))
}

// Step prints a progress step. Color mode: "-> message" with cyan arrow.
// Plain mode: "-> message".
func (w *Writer) Step(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		arrow := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("->")
		w.write(fmt.Appendf(nil, "%s %s\n", arrow, msg))
	} else {
		w.write(fmt.Appendf(nil, "-> %s\n", msg))
	}
}

// Success prints a success message. Color mode: green bold checkmark.
// Plain mode: "OK message".
func (w *Writer) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("OK")
		w.write(fmt.Appendf(nil, "%s %s\n", prefix, msg))
	} else {
		w.write(fmt.Appendf(nil, "OK %s\n", msg))
	}
}

// Error prints an error message. Color mode: red prefix.
// Plain mode: "ERROR message".
func (w *Writer) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")).Render("ERROR")
		w.write(fmt.Appendf(nil, "%s %s\n", prefix, msg))
	} else {
		w.write(fmt.Appendf(nil, "ERROR %s\n", msg))
	}
}

// Warning prints a warning message. Color mode: yellow prefix.
// Plain mode: "WARNING message".
func (w *Writer) Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).Render("WARNING")
		w.write(fmt.Appendf(nil, "%s %s\n", prefix, msg))
	} else {
		w.write(fmt.Appendf(nil, "WARNING %s\n", msg))
	}
}

// Info prints supplementary information indented under a step.
// Color mode: dim text. Plain mode: indented text.
func (w *Writer) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		dim := lipgloss.NewStyle().Faint(true)
		w.write(fmt.Appendf(nil, "   %s\n", dim.Render(msg)))
	} else {
		w.write(fmt.Appendf(nil, "   %s\n", msg))
	}
}

// Result prints key-value pairs with aligned formatting.
func (w *Writer) Result(pairs []KeyValue) {
	if len(pairs) == 0 {
		return
	}

	maxKeyLen := 0
	for _, p := range pairs {
		if len(p.Key) > maxKeyLen {
			maxKeyLen = len(p.Key)
		}
	}

	w.write([]byte("\n"))
	for _, p := range pairs {
		padding := strings.Repeat(" ", maxKeyLen-len(p.Key))
		if w.color {
			key := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cba6f7")).Render(p.Key)
			w.write(fmt.Appendf(nil, "  %s%s  %s\n", key, padding, p.Value))
		} else {
			w.write(fmt.Appendf(nil, "  %s%s  %s\n", p.Key, padding, p.Value))
		}
	}
}

// Table renders a styled table.
func (w *Writer) Table(headers []string, rows [][]string) {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		BorderRow(false).
		BorderColumn(false).
		BorderLeft(false).
		BorderRight(false).
		BorderTop(false).
		BorderBottom(false)

	cellStyle := lipgloss.NewStyle().PaddingRight(1)
	headerStyle := cellStyle
	if w.color {
		headerStyle = headerStyle.Bold(true).Foreground(lipgloss.Color("6"))
	}
	t = t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return headerStyle
		}
		return cellStyle
	})

	w.write([]byte(t.Render() + "\n"))
}

// Println prints a plain line with no prefix or styling.
func (w *Writer) Println(format string, args ...any) {
	w.write(fmt.Appendf(nil, format+"\n", args...))
}

// SetBarStyle configures the visual style used by all progress and indeterminate
// bars created from this Writer. The default is StyleBar.
func (w *Writer) SetBarStyle(s BarStyle) {
	w.barStyle = s
}

func (w *Writer) write(b []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.w.Write(b)
}

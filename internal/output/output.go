// Package output provides styled terminal output with automatic CI and
// terminal capability detection. All human-readable CLI output should use
// this package instead of writing directly to os.Stderr.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

// Writer provides styled terminal output. Create one with New() for
// production use or NewTest() for tests.
type Writer struct {
	w           io.Writer
	interactive bool // terminal AND not CI
	color       bool // terminal AND not NO_COLOR
}

// KeyValue is a key-value pair for Result output.
type KeyValue struct {
	Key   string
	Value string
}

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

// Step prints a progress step. Color mode: "-> message" with cyan arrow.
// Plain mode: "-> message".
func (w *Writer) Step(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		arrow := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("->")
		fmt.Fprintf(w.w, "%s %s\n", arrow, msg)
	} else {
		fmt.Fprintf(w.w, "-> %s\n", msg)
	}
}

// Success prints a success message. Color mode: green bold checkmark.
// Plain mode: "OK message".
func (w *Writer) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("OK")
		fmt.Fprintf(w.w, "%s %s\n", prefix, msg)
	} else {
		fmt.Fprintf(w.w, "OK %s\n", msg)
	}
}

// Error prints an error message. Color mode: red prefix.
// Plain mode: "ERROR message".
func (w *Writer) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1")).Render("ERROR")
		fmt.Fprintf(w.w, "%s %s\n", prefix, msg)
	} else {
		fmt.Fprintf(w.w, "ERROR %s\n", msg)
	}
}

// Warning prints a warning message. Color mode: yellow prefix.
// Plain mode: "WARNING message".
func (w *Writer) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		prefix := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")).Render("WARNING")
		fmt.Fprintf(w.w, "%s %s\n", prefix, msg)
	} else {
		fmt.Fprintf(w.w, "WARNING %s\n", msg)
	}
}

// Info prints supplementary information indented under a step.
// Color mode: dim text. Plain mode: indented text.
func (w *Writer) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if w.color {
		dim := lipgloss.NewStyle().Faint(true)
		fmt.Fprintf(w.w, "   %s\n", dim.Render(msg))
	} else {
		fmt.Fprintf(w.w, "   %s\n", msg)
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

	fmt.Fprintln(w.w)
	for _, p := range pairs {
		padding := strings.Repeat(" ", maxKeyLen-len(p.Key))
		if w.color {
			key := lipgloss.NewStyle().Bold(true).Render(p.Key)
			fmt.Fprintf(w.w, "  %s%s  %s\n", key, padding, p.Value)
		} else {
			fmt.Fprintf(w.w, "  %s%s  %s\n", p.Key, padding, p.Value)
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

	if w.color {
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
		t = t.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle()
		})
	}

	fmt.Fprintln(w.w, t.Render())
}

// Println prints a plain line with no prefix or styling.
func (w *Writer) Println(format string, args ...interface{}) {
	fmt.Fprintf(w.w, format+"\n", args...)
}

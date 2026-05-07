package output

import (
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// BarStyle selects the type of progress indicator used by progress and
// indeterminate bars.
type BarStyle int

const (
	// StyleBar renders a gradient progress bar (indigo to pink). This is the default.
	StyleBar BarStyle = iota
	// StyleSpinner renders an animated braille spinner character (⠋⠙⠹…).
	StyleSpinner
	// StyleCounter shows the percentage and sub-label (X/Y or bytes) without a bar.
	StyleCounter
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ParseBarStyle converts a string name to a BarStyle. Returns StyleBar for
// unrecognised values.
func ParseBarStyle(s string) BarStyle {
	switch s {
	case "spinner":
		return StyleSpinner
	case "counter":
		return StyleCounter
	default:
		return StyleBar
	}
}

// HumanBytes formats a byte count into a human-readable string using binary SI units.
func HumanBytes(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n == 0:
		return "0 B"
	case n < kb:
		return fmt.Sprintf("%d B", n)
	case n < mb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	case n < gb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
	}
}

// ProgressBar renders a determinate progress bar to the terminal.
type ProgressBar struct {
	once        sync.Once
	write       func([]byte) // bound to Writer.write
	interactive bool
	color       bool
	barStyle    BarStyle
	label       string
	width       int // default 30 track chars
	frame       int // spinner frame index, incremented on each Update
}

// NewProgress creates a ProgressBar for the given label. In interactive mode
// it prints "-> label" without a newline so that Update can overwrite it
// in-place. In non-interactive mode it prints "-> label...\n" and the bar
// is a no-op.
func (w *Writer) NewProgress(label string) *ProgressBar {
	pb := &ProgressBar{
		write:       w.write,
		interactive: w.interactive,
		color:       w.color,
		barStyle:    w.barStyle,
		label:       label,
		width:       30,
	}
	if w.interactive {
		w.write(fmt.Appendf(nil, "%s %s", renderArrow(w.color), label))
	} else {
		w.Step("%s...", label)
	}
	return pb
}

// Update renders the progress indicator at the given percentage with an optional sub-label.
// Overwrites the current line (which NewProgress left without a newline) in-place.
// In non-interactive mode this is a no-op.
func (pb *ProgressBar) Update(pct float64, sub string) {
	if !pb.interactive {
		return
	}

	pctStr := renderPct(pct, pb.color)
	arrow := renderArrow(pb.color)

	switch pb.barStyle {
	case StyleSpinner:
		ch := spinnerFrames[pb.frame%len(spinnerFrames)]
		pb.frame++
		if pb.color {
			ch = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")).Render(ch)
		}
		if sub != "" {
			pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s  %s", arrow, pb.label, ch, pctStr, sub))
		} else {
			pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s", arrow, pb.label, ch, pctStr))
		}
	case StyleCounter:
		if sub != "" {
			pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s", arrow, pb.label, pctStr, sub))
		} else {
			pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s", arrow, pb.label, pctStr))
		}
	default: // StyleBar
		filled := max(min(int(math.Round(float64(pb.width)*pct/100)), pb.width), 0)
		bar := renderGradientBar(filled, pb.width-filled, pb.color)
		if sub != "" {
			pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s  %s", arrow, pb.label, bar, pctStr, sub))
		} else {
			pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s", arrow, pb.label, bar, pctStr))
		}
	}
}

// Done finalises the progress indicator. Overwrites the current line with
// "OK label …\n". Idempotent. No-op in non-interactive mode.
func (pb *ProgressBar) Done(sub string) {
	if !pb.interactive {
		return
	}
	pb.once.Do(func() {
		pctStr := renderPct(100, pb.color)
		var ok string
		if pb.color {
			ok = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("OK")
		} else {
			ok = "OK"
		}
		switch pb.barStyle {
		case StyleSpinner, StyleCounter:
			if sub != "" {
				pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s\n", ok, pb.label, pctStr, sub))
			} else {
				pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s\n", ok, pb.label, pctStr))
			}
		default: // StyleBar
			bar := renderGradientBar(pb.width, 0, pb.color)
			if sub != "" {
				pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s  %s\n", ok, pb.label, bar, pctStr, sub))
			} else {
				pb.write(fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s\n", ok, pb.label, bar, pctStr))
			}
		}
	})
}

// Cancel terminates the progress indicator line without marking it as OK.
// Use on error paths instead of Done to avoid a misleading "OK" state.
// Idempotent. No-op in non-interactive mode.
func (pb *ProgressBar) Cancel() {
	if !pb.interactive {
		return
	}
	pb.once.Do(func() {
		pb.write([]byte("\n"))
	})
}

// progressReader wraps an io.Reader and updates a ProgressBar on each read.
type progressReader struct {
	r     io.Reader
	total int64
	read  int64
	pb    *ProgressBar
}

// NewProgressReader wraps r so that each Read updates pb with the current
// transfer progress. total is the expected total byte count; pass 0 if unknown.
func NewProgressReader(r io.Reader, total int64, pb *ProgressBar) io.Reader {
	return &progressReader{r: r, total: total, pb: pb}
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.read += int64(n)
	if r.total > 0 {
		pct := float64(r.read) / float64(r.total) * 100
		r.pb.Update(pct, HumanBytes(r.read)+" / "+HumanBytes(r.total))
	} else {
		r.pb.Update(0, HumanBytes(r.read))
	}
	return n, err
}

// metroProgressRe matches Metro's progress lines in both pipe and PTY modes.
// .*? handles ANSI escape codes or extra text between % and (N/M).
// [^)]* allows trailing text like "modules" inside the parentheses.
var metroProgressRe = regexp.MustCompile(`(\d+\.?\d*)%.*?\((\d+)/(\d+)[^)]*\)`)

// MetroProgressWriter buffers Metro bundler stderr output into a 20-line ring
// for error display. pb may be nil (progress updates are skipped when nil).
// Write is not safe for concurrent use; callers must serialize writes (Metro
// output arrives on a single io.Copy goroutine, so this holds in practice).
type MetroProgressWriter struct {
	pb   *ProgressBar
	buf  []byte
	ring []string // fixed 20-line FIFO ring
}

// NewMetroProgressWriter creates a MetroProgressWriter backed by pb.
func NewMetroProgressWriter(pb *ProgressBar) *MetroProgressWriter {
	return &MetroProgressWriter{pb: pb}
}

// Write implements io.Writer. It parses complete lines (terminated by \r or \n)
// and updates the progress bar or ring buffer accordingly.
// Always returns len(p), nil.
func (w *MetroProgressWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		idx := -1
		for i, b := range w.buf {
			if b == '\r' || b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		// skip the terminator; if \r\n pair, skip both
		rest := w.buf[idx+1:]
		if len(rest) > 0 && w.buf[idx] == '\r' && rest[0] == '\n' {
			rest = rest[1:]
		}
		w.buf = append(w.buf[:0], rest...)

		w.processLine(line)
	}
	return len(p), nil
}

// Flush processes any remaining buffered bytes as a final line.
func (w *MetroProgressWriter) Flush() {
	if len(w.buf) > 0 {
		w.processLine(string(w.buf))
		w.buf = nil
	}
}

// Buffered returns all buffered non-progress lines joined by newlines.
func (w *MetroProgressWriter) Buffered() string {
	return strings.Join(w.ring, "\n")
}

func (w *MetroProgressWriter) processLine(line string) {
	m := metroProgressRe.FindStringSubmatch(line)
	if m != nil && w.pb != nil {
		pct, _ := strconv.ParseFloat(m[1], 64)
		sub := m[2] + "/" + m[3] + " modules"
		w.pb.Update(pct, sub)
		return
	}
	// push to ring, evict oldest if at capacity
	if len(w.ring) >= 20 {
		w.ring = w.ring[1:]
	}
	w.ring = append(w.ring, line)
}

// IndeterminateBar renders a sweeping animation for operations of unknown duration.
type IndeterminateBar struct {
	once        sync.Once
	write       func([]byte)
	interactive bool
	color       bool
	barStyle    BarStyle
	label       string
	width       int
	stop        chan struct{}
	done        chan struct{}
	doneLine    []byte // pre-rendered completion line written by Stop
}

// NewIndeterminate creates an IndeterminateBar. In interactive mode it prints
// "-> label" without a newline (the sweep goroutine overwrites it in-place).
// In non-interactive mode it prints "-> label...\n" and does nothing else.
func (w *Writer) NewIndeterminate(label string) *IndeterminateBar {
	ib := &IndeterminateBar{
		write:       w.write,
		interactive: w.interactive,
		color:       w.color,
		barStyle:    w.barStyle,
		label:       label,
		width:       30,
	}
	if !w.interactive {
		w.Step("%s...", label)
		return ib
	}
	w.write(fmt.Appendf(nil, "%s %s", renderArrow(w.color), label))
	ib.doneLine = indeterminateDoneLine(label, w.color)
	ib.stop = make(chan struct{})
	ib.done = make(chan struct{})
	go ib.sweep()
	return ib
}

// indeterminateDoneLine builds the line written by Stop: erases the sweep bar
// and replaces it with a green "OK label" success-style line.
func indeterminateDoneLine(label string, color bool) []byte {
	if color {
		ok := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")).Render("OK")
		return fmt.Appendf(nil, "\r\033[2K%s %s\n", ok, label)
	}
	return fmt.Appendf(nil, "\r\033[2KOK %s\n", label)
}

const (
	sweepWindowSize = 6
	sweepInterval   = 80 * time.Millisecond
)

// Stop finalises the indeterminate bar. Idempotent and safe to call multiple times.
// In non-interactive mode this is a no-op.
func (ib *IndeterminateBar) Stop() {
	if !ib.interactive {
		return
	}
	ib.once.Do(func() {
		close(ib.stop)
		<-ib.done
		ib.write(ib.doneLine)
	})
}

func (ib *IndeterminateBar) sweepMaxPosition() int {
	switch ib.barStyle {
	case StyleSpinner:
		return len(spinnerFrames)
	case StyleCounter:
		return 3 // animates "." / ".." / "..."
	default:
		return ib.width
	}
}

func (ib *IndeterminateBar) sweep() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	maxPosition := ib.sweepMaxPosition()
	position := 0

	for {
		select {
		case <-ib.stop:
			ib.write(ib.renderFrame(position))
			close(ib.done)
			return
		case <-ticker.C:
			ib.write(ib.renderFrame(position))
			position = (position + 1) % maxPosition
		}
	}
}

func (ib *IndeterminateBar) renderFrame(pos int) []byte {
	switch ib.barStyle {
	case StyleSpinner:
		return ib.renderSpinnerFrame(pos)
	case StyleCounter:
		return ib.renderCounterFrame(pos)
	default:
		return ib.renderSweepBarFrame(pos)
	}
}

func (ib *IndeterminateBar) renderSweepBarFrame(pos int) []byte {
	filledRune := '█'
	runes := []rune(strings.Repeat("░", ib.width))
	for i := range sweepWindowSize {
		runes[(pos+i)%ib.width] = filledRune
	}
	bar := ib.renderGradientSweep(runes, filledRune)
	arrow := renderArrow(ib.color)
	dots := ib.renderDots()
	return fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s", arrow, ib.label, bar, dots)
}

func (ib *IndeterminateBar) renderGradientSweep(runes []rune, filledRune rune) string {
	if !ib.color {
		return string(runes)
	}
	emptyStyle := lipgloss.NewStyle().Background(lipgloss.Color("#313244"))
	var sb strings.Builder
	for i, r := range runes {
		if r == filledRune {
			c := blendHex("#5A56E0", "#EE6FF8", float64(i)/float64(max(len(runes)-1, 1)))
			sb.WriteString(lipgloss.NewStyle().Background(lipgloss.Color(c)).Render(" "))
		} else {
			sb.WriteString(emptyStyle.Render(" "))
		}
	}
	return sb.String()
}

func (ib *IndeterminateBar) renderSpinnerFrame(pos int) []byte {
	ch := spinnerFrames[pos]
	if ib.color {
		ch = lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7")).Render(ch)
	}
	arrow := renderArrow(ib.color)
	dots := ib.renderDots()
	return fmt.Appendf(nil, "\r\033[2K%s %-20s  %s  %s", arrow, ib.label, ch, dots)
}

func (ib *IndeterminateBar) renderCounterFrame(pos int) []byte {
	dotsFrames := []string{".", "..", "..."}
	d := dotsFrames[pos]
	if ib.color {
		d = lipgloss.NewStyle().Faint(true).Render(d)
	}
	arrow := renderArrow(ib.color)
	return fmt.Appendf(nil, "\r\033[2K%s %-20s  %s", arrow, ib.label, d)
}

// renderPct formats a percentage value with color: lavender normally, green at 100%.
func renderPct(pct float64, color bool) string {
	if !color {
		return fmt.Sprintf("%3.0f%%", pct)
	}
	c := "#cba6f7"
	if pct >= 100 {
		c = "#a6e3a1"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(fmt.Sprintf("%3.0f%%", pct))
}

// renderArrow returns the styled "->" prefix used on progress bar lines.
func renderArrow(color bool) string {
	if color {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render("->")
	}
	return "->"
}

// renderGradientBar renders a progress bar. The filled portion uses a
// left-to-right gradient from #5A56E0 (indigo) to #EE6FF8 (pink), matching
// the Charm package-manager demo. The empty track uses faint ░ chars.
func renderGradientBar(filled, empty int, color bool) string {
	if !color {
		return strings.Repeat("█", filled) + strings.Repeat("░", empty)
	}
	emptyStyle := lipgloss.NewStyle().Background(lipgloss.Color("#313244"))
	if filled == 0 {
		return emptyStyle.Render(strings.Repeat(" ", empty))
	}
	var sb strings.Builder
	for i := range filled {
		sb.WriteString(lipgloss.NewStyle().Background(lipgloss.Color(blendHex("#5A56E0", "#EE6FF8", float64(i)/float64(max(filled-1, 1))))).Render(" "))
	}
	sb.WriteString(emptyStyle.Render(strings.Repeat(" ", empty)))
	return sb.String()
}

// blendHex linearly interpolates between two hex color strings at position t ∈ [0,1].
func blendHex(from, to string, t float64) string {
	r1, g1, b1 := parseHex(from)
	r2, g2, b2 := parseHex(to)
	r := uint8(float64(r1) + t*float64(int(r2)-int(r1)))
	g := uint8(float64(g1) + t*float64(int(g2)-int(g1)))
	b := uint8(float64(b1) + t*float64(int(b2)-int(b1)))
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// parseHex parses a "#RRGGBB" color string into its R, G, B components.
func parseHex(hex string) (uint8, uint8, uint8) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b uint8
	fmt.Sscanf(hex, "%02X%02X%02X", &r, &g, &b) //nolint:errcheck
	return r, g, b
}

func (ib *IndeterminateBar) renderDots() string {
	if ib.color {
		return lipgloss.NewStyle().Faint(true).Render("...")
	}
	return "..."
}

// Indeterminate runs action while displaying a sweeping indeterminate progress
// bar. It replaces the Spinner method with identical call semantics.
func (w *Writer) Indeterminate(label string, action func() error) error {
	bar := w.NewIndeterminate(label)
	defer bar.Stop()
	return action()
}

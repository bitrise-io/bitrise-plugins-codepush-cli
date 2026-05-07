package output

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

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
	label       string
	width       int // default 30 track chars
}

// NewProgress creates a ProgressBar for the given label. In non-interactive
// mode it prints "-> label..." immediately via Step.
func (w *Writer) NewProgress(label string) *ProgressBar {
	pb := &ProgressBar{
		write:       w.write,
		interactive: w.interactive,
		color:       w.color,
		label:       label,
		width:       30,
	}
	if !w.interactive {
		w.Step("%s...", label)
	}
	return pb
}

// Update renders the progress bar at the given percentage with an optional sub-label.
// In non-interactive mode this is a no-op.
func (pb *ProgressBar) Update(pct float64, sub string) {
	if !pb.interactive {
		return
	}

	filled := int(float64(pb.width) * pct / 100)
	if filled > pb.width {
		filled = pb.width
	}
	if filled < 0 {
		filled = 0
	}
	empty := pb.width - filled

	filledChar := "█"
	emptyChar := "░"

	var bar string
	if pb.color {
		filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
		emptyStyle := lipgloss.NewStyle().Faint(true)
		bar = filledStyle.Render(strings.Repeat(filledChar, filled)) +
			emptyStyle.Render(strings.Repeat(emptyChar, empty))
	} else {
		bar = strings.Repeat(filledChar, filled) + strings.Repeat(emptyChar, empty)
	}

	var pctStr string
	if pb.color {
		pctColor := "#cba6f7"
		if pct >= 100 {
			pctColor = "#a6e3a1"
		}
		pctStr = lipgloss.NewStyle().Foreground(lipgloss.Color(pctColor)).Render(fmt.Sprintf("%3.0f%%", pct))
	} else {
		pctStr = fmt.Sprintf("%3.0f%%", pct)
	}

	line := fmt.Sprintf("\r\033[2K\r   %-20s  %s  %s  %s", pb.label, bar, pctStr, sub)
	pb.write([]byte(line))
}

// Done finalises the progress bar, printing a 100% frame and a newline.
// Idempotent: safe to call multiple times.
// In non-interactive mode this is a no-op.
func (pb *ProgressBar) Done(sub string) {
	if !pb.interactive {
		return
	}
	pb.once.Do(func() {
		pb.Update(100, sub)
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

var metroProgressRe = regexp.MustCompile(`(\d+\.?\d*)%\s*\((\d+)/(\d+)\)`)

// MetroProgressWriter parses Metro bundler stderr output, forwards progress
// percentages to a ProgressBar, and buffers non-progress lines in a ring.
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
		w.buf = make([]byte, len(rest))
		copy(w.buf, rest)

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
	if m != nil {
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
	label       string
	width       int
	stop        chan struct{}
	done        chan struct{}
}

// NewIndeterminate creates an IndeterminateBar. In interactive mode it starts
// a background sweep goroutine. In non-interactive mode it prints "-> label..."
// via Step and does nothing else.
func (w *Writer) NewIndeterminate(label string) *IndeterminateBar {
	ib := &IndeterminateBar{
		write:       w.write,
		interactive: w.interactive,
		color:       w.color,
		label:       label,
		width:       30,
	}
	if !w.interactive {
		w.Step("%s...", label)
		return ib
	}
	ib.stop = make(chan struct{})
	ib.done = make(chan struct{})
	go ib.sweep()
	return ib
}

const (
	sweepWindowSize = 6
	sweepInterval   = 80 * time.Millisecond
)

func (ib *IndeterminateBar) renderFrame(pos int) []byte {
	filled := "█"
	empty := "░"

	track := make([]byte, ib.width)
	for i := range track {
		track[i] = []byte(empty)[0]
	}

	// build as rune slice for proper rendering
	runes := []rune(strings.Repeat(empty, ib.width))
	filledRune := []rune(filled)[0]
	for i := 0; i < sweepWindowSize; i++ {
		idx := (pos + i) % ib.width
		runes[idx] = filledRune
	}

	var bar string
	if ib.color {
		filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cba6f7"))
		emptyStyle := lipgloss.NewStyle().Faint(true)
		var sb strings.Builder
		inFilled := false
		segStart := 0
		for i, r := range runes {
			isFilled := r == filledRune
			if i == 0 {
				inFilled = isFilled
				segStart = 0
				continue
			}
			if isFilled != inFilled {
				seg := string(runes[segStart:i])
				if inFilled {
					sb.WriteString(filledStyle.Render(seg))
				} else {
					sb.WriteString(emptyStyle.Render(seg))
				}
				inFilled = isFilled
				segStart = i
			}
		}
		seg := string(runes[segStart:])
		if inFilled {
			sb.WriteString(filledStyle.Render(seg))
		} else {
			sb.WriteString(emptyStyle.Render(seg))
		}
		bar = sb.String()
	} else {
		bar = string(runes)
	}

	var dots string
	if ib.color {
		dots = lipgloss.NewStyle().Faint(true).Render("...")
	} else {
		dots = "..."
	}

	line := fmt.Sprintf("\r\033[2K\r   %-20s  %s  %s", ib.label, bar, dots)
	return []byte(line)
}

func (ib *IndeterminateBar) sweep() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	maxPosition := ib.width
	position := 0

	for {
		select {
		case <-ib.stop:
			// write final frame before signalling done
			ib.write(ib.renderFrame(position))
			close(ib.done)
			return
		case <-ticker.C:
			ib.write(ib.renderFrame(position))
			position = (position + 1) % maxPosition
		}
	}
}

// Stop finalises the indeterminate bar. Idempotent and safe to call multiple times.
// In non-interactive mode this is a no-op.
func (ib *IndeterminateBar) Stop() {
	if !ib.interactive {
		return
	}
	ib.once.Do(func() {
		close(ib.stop)
		<-ib.done
		ib.write([]byte("\n"))
	})
}

// Indeterminate runs action while displaying a sweeping indeterminate progress
// bar. It replaces the Spinner method with identical call semantics.
func (w *Writer) Indeterminate(label string, action func() error) error {
	bar := w.NewIndeterminate(label)
	defer bar.Stop()
	return action()
}

// Spinner is an alias for Indeterminate, preserved for backward compatibility.
func (w *Writer) Spinner(label string, action func() error) error {
	return w.Indeterminate(label, action)
}

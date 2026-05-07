package output

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProgressBarNonInteractive verifies that Update and Done are no-ops
// when the Writer is non-interactive (NewTest produces non-interactive writers).
func TestProgressBarNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	pb := w.NewProgress("Uploading")
	// NewProgress prints a step header; reset to isolate bar behaviour
	buf.Reset()

	pb.Update(50, "512 B / 1.0 KB")
	pb.Done("done")

	assert.Empty(t, buf.String(), "Update and Done should be no-ops in non-interactive mode")
}

// TestProgressBarDoneIdempotent verifies that calling Done twice emits only one newline.
func TestProgressBarDoneIdempotent(t *testing.T) {
	var buf bytes.Buffer

	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		label:       "Test",
		width:       30,
	}

	pb.Done("finished")
	pb.Done("finished again")

	got := buf.String()
	count := strings.Count(got, "\n")
	assert.Equal(t, 1, count, "Done called twice should produce exactly one newline, got %q", got)
}

// TestProgressReaderWithTotal verifies that progressReader calls Update
// with increasing percentages across multiple reads when total is known.
func TestProgressReaderWithTotal(t *testing.T) {
	data := make([]byte, 1000)
	r := bytes.NewReader(data)

	var pcts []float64
	pb := &ProgressBar{
		interactive: true,
		color:       false,
		label:       "Downloading",
		width:       30,
	}
	// Capture pct by intercepting the rendered line.
	// In non-color mode the format is: "\r\033[2K\r   label  ...  NNN%  sub"
	pb.write = func(b []byte) {
		s := string(b)
		idx := strings.Index(s, "%")
		if idx < 0 {
			return
		}
		// walk back over digits and dot to find the number
		start := idx
		for start > 0 {
			c := s[start-1]
			if c == ' ' {
				break
			}
			start--
		}
		numStr := strings.TrimSpace(s[start:idx])
		var f float64
		if v, ok := parseTestFloat(numStr); ok {
			f = v
		}
		if f >= 0 {
			pcts = append(pcts, f)
		}
	}

	pr := NewProgressReader(r, 1000, pb)
	chunk := make([]byte, 100)
	for {
		_, err := pr.Read(chunk)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	require.NotEmpty(t, pcts, "expected Update to be called multiple times")
	// percentages should be monotonically non-decreasing
	for i := 1; i < len(pcts); i++ {
		assert.GreaterOrEqual(t, pcts[i], pcts[i-1],
			"pcts should be non-decreasing, got %v at index %d then %v at index %d",
			pcts[i-1], i-1, pcts[i], i)
	}
	// final pct should be ~100
	assert.InDelta(t, 100.0, pcts[len(pcts)-1], 1.0, "final pct should be ~100")
}

// parseTestFloat parses a trimmed numeric string, returning the float and ok.
func parseTestFloat(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	var v float64
	var intPart int64
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		intPart = intPart*10 + int64(s[i]-'0')
		i++
	}
	v = float64(intPart)
	if i < len(s) && s[i] == '.' {
		i++
		place := 0.1
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			v += float64(s[i]-'0') * place
			place /= 10
			i++
		}
	}
	return v, true
}

// TestProgressReaderWithZeroTotal verifies no panic and Update is still called
// (with pct=0) when total is 0.
func TestProgressReaderWithZeroTotal(t *testing.T) {
	data := []byte("hello world")
	r := bytes.NewReader(data)

	updateCalled := false
	pb := &ProgressBar{
		write:       func(b []byte) { updateCalled = true },
		interactive: true,
		label:       "Reading",
		width:       30,
	}

	pr := NewProgressReader(r, 0, pb)
	buf := make([]byte, len(data))
	_, err := io.ReadFull(pr, buf)
	require.NoError(t, err)

	assert.True(t, updateCalled, "Update should have been called even with total=0")
}

// TestMetroProgressWriter verifies that Metro bundler stderr is parsed
// correctly: progress lines update the bar and non-progress lines go to the ring.
func TestMetroProgressWriter(t *testing.T) {
	var pcts []float64

	pb := &ProgressBar{
		interactive: true,
		color:       false, // plain text so we can parse percentages
		label:       "Bundle",
		width:       30,
	}
	pb.write = func(b []byte) {
		s := string(b)
		idx := strings.Index(s, "%")
		if idx < 0 {
			return
		}
		start := idx
		for start > 0 {
			c := s[start-1]
			if c == ' ' {
				break
			}
			start--
		}
		numStr := strings.TrimSpace(s[start:idx])
		if v, ok := parseTestFloat(numStr); ok {
			pcts = append(pcts, v)
		}
	}

	mw := NewMetroProgressWriter(pb)
	input := "up to date, audited 900 packages in 1s\n" +
		"99 packages are looking for funding\n" +
		"warning: Bundler cache is empty, rebuilding\n" +
		"iOS ./index.ts \r 25% (145/583)\r" +
		"iOS ./index.ts \r 50% (291/583)\r" +
		"iOS ./index.ts \r 99.9% (583/583)\n" +
		"Module not found: some error here\n"
	_, err := mw.Write([]byte(input))
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(pcts), 3, "expected at least 3 progress updates, got %v", pcts)
	assert.InDelta(t, 25.0, pcts[0], 1.0, "first pct should be ~25")
	assert.InDelta(t, 50.0, pcts[1], 1.0, "second pct should be ~50")
	assert.InDelta(t, 99.9, pcts[2], 0.5, "third pct should be ~99.9")

	buffered := mw.Buffered()
	assert.Contains(t, buffered, "Module not found", "buffered should contain error lines")
	assert.Contains(t, buffered, "up to date", "buffered should contain npm noise")
	assert.NotContains(t, buffered, "25%", "progress lines should not appear in buffered")
	assert.NotContains(t, buffered, "50%", "progress lines should not appear in buffered")
}

// TestIndeterminateBarNonInteractive verifies that NewIndeterminate prints a
// step in non-interactive mode and that Stop is a no-op afterwards.
func TestIndeterminateBarNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)

	ib := w.NewIndeterminate("test")
	assert.Contains(t, buf.String(), "-> test...")

	before := buf.String()
	ib.Stop()
	assert.Equal(t, before, buf.String(), "Stop should be a no-op in non-interactive mode")
}

// TestHumanBytes verifies HumanBytes output for various inputs.
func TestHumanBytes(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, HumanBytes(tc.n))
		})
	}
}

func TestParseBarStyle(t *testing.T) {
	assert.Equal(t, StyleBar, ParseBarStyle("bar"))
	assert.Equal(t, StyleBar, ParseBarStyle(""))
	assert.Equal(t, StyleBar, ParseBarStyle("unknown"))
	assert.Equal(t, StyleSpinner, ParseBarStyle("spinner"))
	assert.Equal(t, StyleCounter, ParseBarStyle("counter"))
}

func TestSetBarStyle(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.SetBarStyle(StyleSpinner)
	pb := w.NewProgress("test")
	assert.Equal(t, StyleSpinner, pb.barStyle)
}

func TestProgressBarCancelNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	pb := w.NewProgress("test")
	buf.Reset()
	pb.Cancel()
	assert.Empty(t, buf.String(), "Cancel should be a no-op in non-interactive mode")
}

func TestProgressBarCancelInteractive(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		label:       "Uploading",
		width:       30,
	}
	pb.Cancel()
	assert.Equal(t, "\n", buf.String(), "Cancel should write exactly one newline")
}

func TestProgressBarCancelIdempotent(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		label:       "Uploading",
		width:       30,
	}
	pb.Cancel()
	pb.Cancel()
	assert.Equal(t, "\n", buf.String(), "Cancel called twice should produce exactly one newline")
}

func TestProgressBarUpdateSpinner(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleSpinner,
		label:       "Bundling",
		width:       30,
	}
	pb.Update(42, "100/200 modules")
	got := buf.String()
	assert.Contains(t, got, "42%")
	assert.Contains(t, got, "100/200 modules")
	assert.Contains(t, got, "Bundling")
}

func TestProgressBarUpdateSpinnerNoSub(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleSpinner,
		label:       "Bundling",
		width:       30,
	}
	pb.Update(42, "")
	got := buf.String()
	assert.Contains(t, got, "42%")
	assert.NotContains(t, got, "modules")
}

func TestProgressBarUpdateSpinnerAdvancesFrame(t *testing.T) {
	var writes []string
	pb := &ProgressBar{
		write:       func(b []byte) { writes = append(writes, string(b)) },
		interactive: true,
		color:       false,
		barStyle:    StyleSpinner,
		label:       "Bundling",
		width:       30,
	}
	pb.Update(50, "")
	pb.Update(60, "")
	require.Len(t, writes, 2)
	// Each Update should use a different spinner frame
	assert.NotEqual(t, writes[0], writes[1], "successive updates should advance the spinner frame")
}

func TestProgressBarUpdateCounter(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleCounter,
		label:       "Processing",
		width:       30,
	}
	pb.Update(75, "3/4 chunks")
	got := buf.String()
	assert.Contains(t, got, "75%")
	assert.Contains(t, got, "3/4 chunks")
}

func TestProgressBarUpdateCounterNoSub(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleCounter,
		label:       "Processing",
		width:       30,
	}
	pb.Update(75, "")
	got := buf.String()
	assert.Contains(t, got, "75%")
}

func TestProgressBarUpdateBarInteractive(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleBar,
		label:       "Uploading",
		width:       10,
	}
	pb.Update(50, "500 B / 1.0 KB")
	got := buf.String()
	assert.Contains(t, got, "50%")
	assert.Contains(t, got, "█")
	assert.Contains(t, got, "░")
}

func TestProgressBarUpdateBarNoSub(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleBar,
		label:       "Uploading",
		width:       10,
	}
	pb.Update(100, "")
	got := buf.String()
	assert.Contains(t, got, "100%")
}

func TestProgressBarDoneSpinner(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleSpinner,
		label:       "Bundling",
		width:       30,
	}
	pb.Done("3.5 MB")
	got := buf.String()
	assert.Contains(t, got, "OK")
	assert.Contains(t, got, "Bundling")
	assert.Contains(t, got, "100%")
	assert.Contains(t, got, "3.5 MB")
	assert.Equal(t, 1, strings.Count(got, "\n"))
}

func TestProgressBarDoneSpinnerNoSub(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleSpinner,
		label:       "Bundling",
		width:       30,
	}
	pb.Done("")
	got := buf.String()
	assert.Contains(t, got, "OK")
	assert.Contains(t, got, "100%")
}

func TestProgressBarDoneCounter(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleCounter,
		label:       "Processing",
		width:       30,
	}
	pb.Done("done")
	got := buf.String()
	assert.Contains(t, got, "OK")
	assert.Contains(t, got, "100%")
	assert.Contains(t, got, "done")
}

func TestProgressBarDoneBarNoSub(t *testing.T) {
	var buf bytes.Buffer
	pb := &ProgressBar{
		write:       func(b []byte) { _, _ = buf.Write(b) },
		interactive: true,
		color:       false,
		barStyle:    StyleBar,
		label:       "Uploading",
		width:       10,
	}
	pb.Done("")
	got := buf.String()
	assert.Contains(t, got, "OK")
	assert.Contains(t, got, "100%")
	assert.Equal(t, 1, strings.Count(got, "\n"))
}

func TestMetroProgressWriterFlush(t *testing.T) {
	var updates []float64
	pb := &ProgressBar{
		interactive: true,
		color:       false,
		label:       "Bundle",
		width:       30,
	}
	pb.write = func(b []byte) {
		s := string(b)
		idx := strings.Index(s, "%")
		if idx < 0 {
			return
		}
		start := idx
		for start > 0 && s[start-1] != ' ' {
			start--
		}
		if v, ok := parseTestFloat(strings.TrimSpace(s[start:idx])); ok {
			updates = append(updates, v)
		}
	}

	mw := NewMetroProgressWriter(pb)
	// Write a progress line without a newline terminator (simulates Metro's final line)
	_, err := mw.Write([]byte("75% (437/583 modules)"))
	require.NoError(t, err)
	assert.Empty(t, updates, "unterminated line should not be processed before Flush")

	mw.Flush()
	require.NotEmpty(t, updates, "Flush should process the remaining buffer")
	assert.InDelta(t, 75.0, updates[0], 1.0)
}

func TestMetroProgressWriterFlushEmpty(t *testing.T) {
	pb := &ProgressBar{
		interactive: true,
		write:       func([]byte) {},
		label:       "Bundle",
		width:       30,
	}
	mw := NewMetroProgressWriter(pb)
	mw.Flush() // should not panic on empty buffer
}

func TestMetroProgressWriterNilPb(t *testing.T) {
	mw := NewMetroProgressWriter(nil)
	_, err := mw.Write([]byte("50% (291/583)\n"))
	require.NoError(t, err)
	// should not panic with nil pb
}

func TestRenderPct(t *testing.T) {
	// non-color path
	assert.Equal(t, " 50%", renderPct(50, false))
	assert.Equal(t, "100%", renderPct(100, false))
	assert.Equal(t, "  0%", renderPct(0, false))

	// color path: just verify it doesn't panic and contains the number
	got := renderPct(50, true)
	assert.Contains(t, got, "50")

	got100 := renderPct(100, true)
	assert.Contains(t, got100, "100")
}

func TestRenderGradientBar(t *testing.T) {
	// non-color: plain chars
	plain := renderGradientBar(5, 5, false)
	assert.Equal(t, "█████░░░░░", plain)

	plain0 := renderGradientBar(0, 10, false)
	assert.Equal(t, "░░░░░░░░░░", plain0)

	plainFull := renderGradientBar(10, 0, false)
	assert.Equal(t, "██████████", plainFull)

	// color path: should not panic and produce non-empty output
	colored := renderGradientBar(5, 5, true)
	assert.NotEmpty(t, colored)

	colored0 := renderGradientBar(0, 5, true)
	assert.NotEmpty(t, colored0)
}

func TestBlendHex(t *testing.T) {
	// blending at 0 should return the "from" color
	got := blendHex("#FF0000", "#0000FF", 0)
	assert.Equal(t, "#FF0000", got)

	// blending at 1 should return the "to" color
	got = blendHex("#FF0000", "#0000FF", 1)
	assert.Equal(t, "#0000FF", got)

	// midpoint should be roughly purple
	got = blendHex("#FF0000", "#0000FF", 0.5)
	assert.Equal(t, "#7F007F", got)
}

func TestParseHex(t *testing.T) {
	r, g, b := parseHex("#5A56E0")
	assert.Equal(t, uint8(0x5A), r)
	assert.Equal(t, uint8(0x56), g)
	assert.Equal(t, uint8(0xE0), b)

	// Without the # prefix
	r, g, b = parseHex("EE6FF8")
	assert.Equal(t, uint8(0xEE), r)
	assert.Equal(t, uint8(0x6F), g)
	assert.Equal(t, uint8(0xF8), b)
}

func TestIndeterminateDoneLine(t *testing.T) {
	plain := indeterminateDoneLine("Validating", false)
	assert.Contains(t, string(plain), "OK Validating")
	assert.Contains(t, string(plain), "\n")

	colored := indeterminateDoneLine("Validating", true)
	assert.Contains(t, string(colored), "Validating")
	assert.Contains(t, string(colored), "\n")
}

func TestSweepMaxPosition(t *testing.T) {
	ib := &IndeterminateBar{barStyle: StyleBar, width: 30}
	assert.Equal(t, 30, ib.sweepMaxPosition())

	ib.barStyle = StyleSpinner
	assert.Equal(t, len(spinnerFrames), ib.sweepMaxPosition())

	ib.barStyle = StyleCounter
	assert.Equal(t, 3, ib.sweepMaxPosition())
}

func TestRenderFrameSpinner(t *testing.T) {
	ib := &IndeterminateBar{barStyle: StyleSpinner, label: "test", color: false, width: 30}
	frame := ib.renderFrame(0)
	assert.NotEmpty(t, frame)
	assert.Contains(t, string(frame), spinnerFrames[0])
}

func TestRenderFrameCounter(t *testing.T) {
	ib := &IndeterminateBar{barStyle: StyleCounter, label: "test", color: false, width: 30}
	frame := ib.renderFrame(0)
	assert.NotEmpty(t, frame)
	assert.Contains(t, string(frame), ".")
}

func TestRenderFrameBar(t *testing.T) {
	ib := &IndeterminateBar{barStyle: StyleBar, label: "test", color: false, width: 10}
	frame := ib.renderFrame(0)
	assert.NotEmpty(t, frame)
	assert.Contains(t, string(frame), "test")
}

func TestRenderSpinnerFrameColor(t *testing.T) {
	ib := &IndeterminateBar{barStyle: StyleSpinner, label: "test", color: true, width: 30}
	frame := ib.renderSpinnerFrame(0)
	assert.NotEmpty(t, frame)
}

func TestRenderCounterFrameColor(t *testing.T) {
	ib := &IndeterminateBar{barStyle: StyleCounter, label: "test", color: true, width: 30}
	frame := ib.renderCounterFrame(0)
	assert.NotEmpty(t, frame)

	// All three dot positions
	for i := range 3 {
		f := ib.renderCounterFrame(i)
		assert.NotEmpty(t, f)
	}
}

func TestRenderSweepBarFrameColor(t *testing.T) {
	ib := &IndeterminateBar{barStyle: StyleBar, label: "test", color: true, width: 10}
	frame := ib.renderSweepBarFrame(0)
	assert.NotEmpty(t, frame)
}

func TestRenderGradientSweepNoColor(t *testing.T) {
	ib := &IndeterminateBar{barStyle: StyleBar, label: "test", color: false, width: 10}
	runes := []rune("██░░░░░░░░")
	result := ib.renderGradientSweep(runes, '█')
	assert.Equal(t, "██░░░░░░░░", result)
}

func TestRenderDots(t *testing.T) {
	ib := &IndeterminateBar{color: false}
	assert.Equal(t, "...", ib.renderDots())

	ibColor := &IndeterminateBar{color: true}
	got := ibColor.renderDots()
	assert.NotEmpty(t, got)
}

func TestIndeterminateBarInteractive(t *testing.T) {
	var mu bytes.Buffer
	writeFn := func(b []byte) { _, _ = mu.Write(b) }

	ib := &IndeterminateBar{
		write:       writeFn,
		interactive: true,
		color:       false,
		barStyle:    StyleBar,
		label:       "Processing",
		width:       10,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		doneLine:    indeterminateDoneLine("Processing", false),
	}
	go ib.sweep()

	// Let the goroutine tick at least once
	time.Sleep(200 * time.Millisecond)

	ib.Stop()

	got := mu.String()
	assert.Contains(t, got, "OK Processing")
}

func TestIndeterminateBarStopIdempotent(t *testing.T) {
	writeFn := func(b []byte) {}
	ib := &IndeterminateBar{
		write:       writeFn,
		interactive: true,
		color:       false,
		barStyle:    StyleSpinner,
		label:       "test",
		width:       10,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		doneLine:    indeterminateDoneLine("test", false),
	}
	go ib.sweep()
	ib.Stop()
	ib.Stop() // must not deadlock or panic
}

func TestNewIndeterminateInteractive(t *testing.T) {
	var buf bytes.Buffer
	w := &Writer{
		w:           &buf,
		interactive: true,
		color:       false,
		barStyle:    StyleBar,
	}

	ib := w.NewIndeterminate("Validating")
	require.NotNil(t, ib)

	// Let the goroutine tick
	time.Sleep(200 * time.Millisecond)
	ib.Stop()

	got := buf.String()
	assert.Contains(t, got, "Validating")
	assert.Contains(t, got, "OK Validating")
}

func TestProgressBarNewProgressInteractive(t *testing.T) {
	var buf bytes.Buffer
	w := &Writer{
		w:           &buf,
		interactive: true,
		color:       false,
		barStyle:    StyleBar,
	}

	pb := w.NewProgress("Uploading")
	require.NotNil(t, pb)
	assert.Contains(t, buf.String(), "-> Uploading")
	assert.NotContains(t, buf.String(), "\n", "interactive NewProgress should not end with newline")

	pb.Done("1.0 MB")
	assert.Contains(t, buf.String(), "OK")
}

package output

import (
	"bytes"
	"io"
	"strings"
	"testing"

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
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, HumanBytes(tc.n))
		})
	}
}

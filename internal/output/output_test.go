package output

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStep(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Step("Resolving deployment %q", "Staging")

	assert.Contains(t, buf.String(), `-> Resolving deployment "Staging"`)
}

func TestSuccess(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Success("Push successful")

	assert.Contains(t, buf.String(), "OK Push successful")
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Error("push failed: %v", "timeout")

	assert.Contains(t, buf.String(), "ERROR push failed: timeout")
}

func TestWarning(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Warning("could not load token: %v", "file not found")

	assert.Contains(t, buf.String(), "WARNING could not load token: file not found")
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Info("Resolved to abc-123")

	got := buf.String()
	assert.Contains(t, got, "Resolved to abc-123")
	// Info should be indented
	assert.True(t, len(got) >= 3 && got[:3] == "   ", "Info output should be indented, got %q", got)
}

func TestResult(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Result([]KeyValue{
		{Key: "Package ID", Value: "abc-123"},
		{Key: "App version", Value: "1.0.0"},
		{Key: "Status", Value: "done"},
	})

	got := buf.String()
	assert.Contains(t, got, "Package ID")
	assert.Contains(t, got, "abc-123")
	assert.Contains(t, got, "Status")
	assert.Contains(t, got, "done")
}

func TestResultEmpty(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Result(nil)

	assert.Empty(t, buf.String())
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Table(
		[]string{"LABEL", "APP VERSION", "STATUS"},
		[][]string{
			{"v1", "1.0.0", "done"},
			{"v2", "2.0.0", "done"},
		},
	)

	got := buf.String()
	assert.Contains(t, got, "LABEL")
	assert.Contains(t, got, "v1")
}

func TestPrintln(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Println("CodePush CLI %s", "1.0.0")

	assert.Equal(t, "CodePush CLI 1.0.0\n", buf.String())
}

func TestSpinnerNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)

	called := false
	err := w.Spinner("Processing", func() error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called)
	assert.Contains(t, buf.String(), "Processing")
}

func TestConfirmDestructiveWithYesFlag(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	err := w.ConfirmDestructive("This will delete everything", true)
	require.NoError(t, err)
}

func TestConfirmDestructiveNonInteractiveNoYes(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	err := w.ConfirmDestructive("This will delete everything", false)
	require.Error(t, err)
	assert.ErrorContains(t, err, "--yes")
}

func TestIsInteractive(t *testing.T) {
	w := NewTest(&bytes.Buffer{})
	assert.False(t, w.IsInteractive())
}

func TestSpinnerNonInteractiveError(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)

	wantErr := "upload failed"
	err := w.Spinner("Uploading", func() error {
		return fmt.Errorf("%s", wantErr)
	})

	require.Error(t, err)
	assert.Equal(t, wantErr, err.Error())
	assert.Contains(t, buf.String(), "Uploading")
}

func TestConfirmDestructiveNonInteractiveIncludesMessage(t *testing.T) {
	w := NewTest(&bytes.Buffer{})
	err := w.ConfirmDestructive("delete deployment Staging", false)
	require.Error(t, err)
	assert.ErrorContains(t, err, "delete deployment Staging")
}

func TestNewWriter(t *testing.T) {
	// NewWriter with a non-terminal writer should produce a non-interactive writer
	var buf bytes.Buffer
	w := NewWriter(&buf)
	assert.False(t, w.IsInteractive())

	// Verify it can write output
	w.Step("test step")
	assert.Contains(t, buf.String(), "test step")
}

func TestNew(t *testing.T) {
	w := New()
	require.NotNil(t, w)
	// New() targets stderr; just verify it returns a usable writer
	w.Step("smoke test")
}

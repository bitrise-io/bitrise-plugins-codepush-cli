package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestStep(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Step("Resolving deployment %q", "Staging")

	got := buf.String()
	if !strings.Contains(got, "-> Resolving deployment \"Staging\"") {
		t.Errorf("Step output = %q, want substring %q", got, "-> Resolving deployment \"Staging\"")
	}
}

func TestSuccess(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Success("Push successful")

	got := buf.String()
	if !strings.Contains(got, "OK Push successful") {
		t.Errorf("Success output = %q, want substring %q", got, "OK Push successful")
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Error("push failed: %v", "timeout")

	got := buf.String()
	if !strings.Contains(got, "ERROR push failed: timeout") {
		t.Errorf("Error output = %q, want substring %q", got, "ERROR push failed: timeout")
	}
}

func TestWarning(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Warning("could not load token: %v", "file not found")

	got := buf.String()
	if !strings.Contains(got, "WARNING could not load token: file not found") {
		t.Errorf("Warning output = %q, want substring %q", got, "WARNING could not load token: file not found")
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Info("Resolved to abc-123")

	got := buf.String()
	if !strings.Contains(got, "Resolved to abc-123") {
		t.Errorf("Info output = %q, want substring %q", got, "Resolved to abc-123")
	}
	// Info should be indented
	if !strings.HasPrefix(got, "   ") {
		t.Errorf("Info output should be indented, got %q", got)
	}
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
	if !strings.Contains(got, "Package ID") || !strings.Contains(got, "abc-123") {
		t.Errorf("Result should contain key and value, got %q", got)
	}
	if !strings.Contains(got, "Status") || !strings.Contains(got, "done") {
		t.Errorf("Result should contain all pairs, got %q", got)
	}
}

func TestResultEmpty(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Result(nil)

	if buf.Len() != 0 {
		t.Errorf("Result with nil pairs should produce no output, got %q", buf.String())
	}
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
	if !strings.Contains(got, "LABEL") || !strings.Contains(got, "v1") {
		t.Errorf("Table output = %q, want headers and rows", got)
	}
}

func TestPrintln(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	w.Println("CodePush CLI %s", "1.0.0")

	got := buf.String()
	if got != "CodePush CLI 1.0.0\n" {
		t.Errorf("Println output = %q, want %q", got, "CodePush CLI 1.0.0\n")
	}
}

func TestSpinnerNonInteractive(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)

	called := false
	err := w.Spinner("Processing", func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("action was not called")
	}

	got := buf.String()
	if !strings.Contains(got, "Processing") {
		t.Errorf("Spinner output = %q, want substring %q", got, "Processing")
	}
}

func TestConfirmDestructiveWithYesFlag(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	err := w.ConfirmDestructive("This will delete everything", true)
	if err != nil {
		t.Fatalf("unexpected error with yesFlag=true: %v", err)
	}
}

func TestConfirmDestructiveNonInteractiveNoYes(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)
	err := w.ConfirmDestructive("This will delete everything", false)
	if err == nil {
		t.Fatal("expected error for non-interactive without --yes, got nil")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("error should hint --yes, got: %v", err)
	}
}

func TestIsInteractive(t *testing.T) {
	w := NewTest(&bytes.Buffer{})
	if w.IsInteractive() {
		t.Error("NewTest writer should not be interactive")
	}
}

func TestSpinnerNonInteractiveError(t *testing.T) {
	var buf bytes.Buffer
	w := NewTest(&buf)

	wantErr := "upload failed"
	err := w.Spinner("Uploading", func() error {
		return fmt.Errorf("%s", wantErr)
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != wantErr {
		t.Errorf("error = %q, want %q", err.Error(), wantErr)
	}

	got := buf.String()
	if !strings.Contains(got, "Uploading") {
		t.Errorf("Spinner should print step before action, got %q", got)
	}
}

func TestConfirmDestructiveNonInteractiveIncludesMessage(t *testing.T) {
	w := NewTest(&bytes.Buffer{})
	err := w.ConfirmDestructive("delete deployment Staging", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "delete deployment Staging") {
		t.Errorf("error should include the message, got: %v", err)
	}
}

func TestNewWriter(t *testing.T) {
	// NewWriter with a non-terminal writer should produce a non-interactive writer
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if w.IsInteractive() {
		t.Error("NewWriter with bytes.Buffer should not be interactive")
	}

	// Verify it can write output
	w.Step("test step")
	if !strings.Contains(buf.String(), "test step") {
		t.Errorf("output should contain step text, got %q", buf.String())
	}
}

func TestNew(t *testing.T) {
	w := New()
	if w == nil {
		t.Fatal("New() returned nil")
	}
	// New() targets stderr; just verify it returns a usable writer
	w.Step("smoke test")
}

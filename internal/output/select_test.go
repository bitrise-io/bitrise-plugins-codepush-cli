package output

import (
	"io"
	"strings"
	"testing"
)

func TestSelect_NonInteractive(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		options []SelectOption
	}{
		{
			name:  "returns error with options",
			title: "Select deployment",
			options: []SelectOption{
				{Label: "Staging", Value: "staging-id"},
				{Label: "Production", Value: "prod-id"},
			},
		},
		{
			name:    "returns error with empty options",
			title:   "Select platform",
			options: []SelectOption{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := NewTest(io.Discard)

			value, err := w.Select(tc.title, tc.options)
			if err == nil {
				t.Fatal("expected error in non-interactive mode, got nil")
			}
			if value != "" {
				t.Errorf("expected empty value, got %q", value)
			}
			if !strings.Contains(err.Error(), "non-interactive") {
				t.Errorf("error should mention non-interactive mode, got: %v", err)
			}
		})
	}
}

func TestInput_NonInteractive(t *testing.T) {
	w := NewTest(io.Discard)

	value, err := w.Input("Enter value", "placeholder")
	if err == nil {
		t.Fatal("expected error in non-interactive mode, got nil")
	}
	if value != "" {
		t.Errorf("expected empty value, got %q", value)
	}
	if !strings.Contains(err.Error(), "non-interactive") {
		t.Errorf("error should mention non-interactive mode, got: %v", err)
	}
}

func TestSecureInput_NonInteractive(t *testing.T) {
	w := NewTest(io.Discard)

	value, err := w.SecureInput("Enter token", "")
	if err == nil {
		t.Fatal("expected error in non-interactive mode, got nil")
	}
	if value != "" {
		t.Errorf("expected empty value, got %q", value)
	}
	if !strings.Contains(err.Error(), "non-interactive") {
		t.Errorf("error should mention non-interactive mode, got: %v", err)
	}
}

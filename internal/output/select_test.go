package output

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			require.Error(t, err)
			assert.Empty(t, value)
			assert.ErrorContains(t, err, "non-interactive")
		})
	}
}

func TestInput_NonInteractive(t *testing.T) {
	w := NewTest(io.Discard)

	value, err := w.Input("Enter value", "placeholder")
	require.Error(t, err)
	assert.Empty(t, value)
	assert.ErrorContains(t, err, "non-interactive")
}

func TestSecureInput_NonInteractive(t *testing.T) {
	w := NewTest(io.Discard)

	value, err := w.SecureInput("Enter token", "")
	require.Error(t, err)
	assert.Empty(t, value)
	assert.ErrorContains(t, err, "non-interactive")
}

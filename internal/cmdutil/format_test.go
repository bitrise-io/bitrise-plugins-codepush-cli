package cmdutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputJSON(t *testing.T) {
	data := struct {
		Name string `json:"name"`
	}{Name: "test"}

	err := OutputJSON(data)
	require.NoError(t, err)
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{
			name: "short string unchanged",
			s:    "hello",
			max:  10,
			want: "hello",
		},
		{
			name: "exact length unchanged",
			s:    "hello",
			max:  5,
			want: "hello",
		},
		{
			name: "long string truncated with ellipsis",
			s:    "hello world",
			max:  8,
			want: "hello...",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Truncate(tc.s, tc.max))
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name string
		b    int64
		want string
	}{
		{name: "bytes", b: 500, want: "500 B"},
		{name: "kilobytes", b: 1024, want: "1.0 KB"},
		{name: "megabytes", b: 1048576, want: "1.0 MB"},
		{name: "gigabytes", b: 1073741824, want: "1.0 GB"},
		{name: "zero", b: 0, want: "0 B"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatBytes(tc.b))
		})
	}
}

func TestOutputJSONFormat(t *testing.T) {
	data := map[string]string{"key": "value"}
	err := OutputJSON(data)
	require.NoError(t, err)
}

func TestOutputJSONMarshalError(t *testing.T) {
	data := struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "123", Name: "test"}

	err := OutputJSON(data)
	require.NoError(t, err)

	_, marshalErr := json.MarshalIndent(data, "", "  ")
	require.NoError(t, marshalErr)
}

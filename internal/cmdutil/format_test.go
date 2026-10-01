package cmdutil

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
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
		{name: "exabytes caps at EB", b: 1 << 60, want: "1.0 EB"},
		{name: "max int64 caps at EB", b: math.MaxInt64, want: "8.0 EB"},
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

func TestDiffStatusPairs(t *testing.T) {
	v2Size := int64(41_000)

	tests := []struct {
		name   string
		status *codepush.UpdateStatus
		want   []output.KeyValue
	}{
		{
			name:   "servers without the field yield nothing",
			status: &codepush.UpdateStatus{Status: codepush.StatusProcessedValid},
			want:   nil,
		},
		{
			name:   "pending is a single line",
			status: &codepush.UpdateStatus{DiffGenerationStatus: codepush.DiffGenerationPending, Diffs: map[string]codepush.DiffInfo{}},
			want:   []output.KeyValue{{Key: "Diff generation", Value: "pending"}},
		},
		{
			name:   "completed without diffs says the full package is served",
			status: &codepush.UpdateStatus{DiffGenerationStatus: codepush.DiffGenerationCompleted, Diffs: map[string]codepush.DiffInfo{}},
			want:   []output.KeyValue{{Key: "Diff generation", Value: "completed (no diffs, clients download the full package)"}},
		},
		{
			name: "completed lists predecessors in hash order with v1 and v2 sizes",
			status: &codepush.UpdateStatus{
				DiffGenerationStatus: codepush.DiffGenerationCompleted,
				Diffs: map[string]codepush.DiffInfo{
					"b7c8d9e0f1a2b3c4d5e6":     {Size: 98_765},
					"a3f1c2d4e5b6978081920a1b": {Size: 123_456, V2Size: &v2Size},
				},
			},
			want: []output.KeyValue{
				{Key: "Diff generation", Value: "completed"},
				{Key: "Diff from a3f1c2d4e5b6...", Value: "v1 120.6 KB, v2 40.0 KB"},
				{Key: "Diff from b7c8d9e0f1a2...", Value: "v1 96.5 KB"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, DiffStatusPairs(tc.status))
		})
	}
}

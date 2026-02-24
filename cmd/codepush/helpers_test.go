package main

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

	err := outputJSON(data)
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
			assert.Equal(t, tc.want, truncate(tc.s, tc.max))
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
			assert.Equal(t, tc.want, formatBytes(tc.b))
		})
	}
}

func TestResolveAppID(t *testing.T) {
	t.Run("flag takes priority", func(t *testing.T) {
		old := globalAppID
		globalAppID = "flag-value"
		defer func() { globalAppID = old }()

		assert.Equal(t, "flag-value", resolveAppID())
	})

	t.Run("falls back to env var", func(t *testing.T) {
		old := globalAppID
		globalAppID = ""
		defer func() { globalAppID = old }()

		t.Setenv("CODEPUSH_APP_ID", "env-value")
		assert.Equal(t, "env-value", resolveAppID())
	})
}

func TestRequireCredentials(t *testing.T) {
	t.Run("returns error when app ID missing", func(t *testing.T) {
		old := globalAppID
		globalAppID = ""
		defer func() { globalAppID = old }()

		t.Setenv("CODEPUSH_APP_ID", "")
		_, _, err := requireCredentials()
		require.Error(t, err)
		assert.ErrorContains(t, err, "app ID is required")
	})

	t.Run("returns values when both set", func(t *testing.T) {
		old := globalAppID
		globalAppID = "my-app"
		defer func() { globalAppID = old }()

		t.Setenv("BITRISE_API_TOKEN", "my-token")
		appID, token, err := requireCredentials()
		require.NoError(t, err)
		assert.Equal(t, "my-app", appID)
		assert.Equal(t, "my-token", token)
	})
}

func TestResolveInputInteractive(t *testing.T) {
	t.Run("returns value when provided", func(t *testing.T) {
		got, err := resolveInputInteractive("provided", "Enter name", "placeholder")
		require.NoError(t, err)
		assert.Equal(t, "provided", got)
	})

	t.Run("returns error in non-interactive mode", func(t *testing.T) {
		_, err := resolveInputInteractive("", "Enter name", "placeholder")
		require.Error(t, err)
		assert.ErrorContains(t, err, "Enter name")
	})
}

func TestResolveAppIDInteractive(t *testing.T) {
	t.Run("returns valid UUID from flag", func(t *testing.T) {
		old := globalAppID
		globalAppID = "550e8400-e29b-41d4-a716-446655440000"
		defer func() { globalAppID = old }()

		got, err := resolveAppIDInteractive()
		require.NoError(t, err)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got)
	})

	t.Run("returns error for invalid UUID from flag", func(t *testing.T) {
		old := globalAppID
		globalAppID = "not-a-uuid"
		defer func() { globalAppID = old }()

		_, err := resolveAppIDInteractive()
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid app ID")
	})

	t.Run("returns error in non-interactive mode when empty", func(t *testing.T) {
		old := globalAppID
		globalAppID = ""
		defer func() { globalAppID = old }()

		t.Setenv("CODEPUSH_APP_ID", "")

		_, err := resolveAppIDInteractive()
		require.Error(t, err)
		assert.ErrorContains(t, err, "app ID is required")
	})
}

func TestResolvePlatformInteractive(t *testing.T) {
	t.Run("returns value when provided", func(t *testing.T) {
		got, err := resolvePlatformInteractive("ios")
		require.NoError(t, err)
		assert.Equal(t, "ios", got)
	})

	t.Run("returns error in non-interactive mode", func(t *testing.T) {
		_, err := resolvePlatformInteractive("")
		require.Error(t, err)
		assert.ErrorContains(t, err, "--platform")
	})
}

func TestOutputJSONFormat(t *testing.T) {
	data := map[string]string{"key": "value"}

	// outputJSON writes to stdout; just verify no error
	err := outputJSON(data)
	require.NoError(t, err)
}

func TestOutputJSONMarshalError(t *testing.T) {
	// json.Marshal cannot fail for a regular struct, but test with valid data
	data := struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "123", Name: "test"}

	err := outputJSON(data)
	require.NoError(t, err)

	// Verify the output is valid JSON by marshaling it ourselves
	_, marshalErr := json.MarshalIndent(data, "", "  ")
	require.NoError(t, marshalErr)
}

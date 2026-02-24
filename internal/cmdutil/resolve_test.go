package cmdutil

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestResolveFlag(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		envKey    string
		envValue  string
		want      string
	}{
		{
			name:      "flag value takes priority",
			flagValue: "from-flag",
			envKey:    "TEST_RESOLVE_FLAG",
			envValue:  "from-env",
			want:      "from-flag",
		},
		{
			name:      "falls back to env var",
			flagValue: "",
			envKey:    "TEST_RESOLVE_FLAG",
			envValue:  "from-env",
			want:      "from-env",
		},
		{
			name:      "returns empty when both empty",
			flagValue: "",
			envKey:    "TEST_RESOLVE_FLAG_EMPTY",
			envValue:  "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}
			assert.Equal(t, tt.want, ResolveFlag(tt.flagValue, tt.envKey))
		})
	}
}

func TestResolveToken(t *testing.T) {
	out := output.NewTest(io.Discard)

	t.Run("env var takes priority", func(t *testing.T) {
		t.Setenv("BITRISE_API_TOKEN", "env-token")
		assert.Equal(t, "env-token", ResolveToken(out))
	})

	t.Run("returns empty when nothing set", func(t *testing.T) {
		t.Setenv("BITRISE_API_TOKEN", "")
		_ = ResolveToken(out)
	})
}

func TestResolveAppID(t *testing.T) {
	out := output.NewTest(io.Discard)

	t.Run("flag takes priority", func(t *testing.T) {
		assert.Equal(t, "flag-value", ResolveAppID("flag-value", out))
	})

	t.Run("falls back to env var", func(t *testing.T) {
		t.Setenv("CODEPUSH_APP_ID", "env-value")
		assert.Equal(t, "env-value", ResolveAppID("", out))
	})
}

func TestRequireCredentials(t *testing.T) {
	out := output.NewTest(io.Discard)

	t.Run("returns error when app ID missing", func(t *testing.T) {
		t.Setenv("CODEPUSH_APP_ID", "")
		_, _, err := RequireCredentials("", out)
		require.Error(t, err)
		assert.ErrorContains(t, err, "app ID is required")
	})

	t.Run("returns values when both set", func(t *testing.T) {
		t.Setenv("BITRISE_API_TOKEN", "my-token")
		appID, token, err := RequireCredentials("my-app", out)
		require.NoError(t, err)
		assert.Equal(t, "my-app", appID)
		assert.Equal(t, "my-token", token)
	})
}

func TestResolveInputInteractive(t *testing.T) {
	out := output.NewTest(io.Discard)

	t.Run("returns value when provided", func(t *testing.T) {
		got, err := ResolveInputInteractive("provided", "Enter name", "placeholder", out)
		require.NoError(t, err)
		assert.Equal(t, "provided", got)
	})

	t.Run("returns error in non-interactive mode", func(t *testing.T) {
		_, err := ResolveInputInteractive("", "Enter name", "placeholder", out)
		require.Error(t, err)
		assert.ErrorContains(t, err, "Enter name")
	})
}

func TestResolveAppIDInteractive(t *testing.T) {
	out := output.NewTest(io.Discard)

	t.Run("returns valid UUID from flag", func(t *testing.T) {
		got, err := ResolveAppIDInteractive("550e8400-e29b-41d4-a716-446655440000", out)
		require.NoError(t, err)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", got)
	})

	t.Run("returns error for invalid UUID from flag", func(t *testing.T) {
		_, err := ResolveAppIDInteractive("not-a-uuid", out)
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid app ID")
	})

	t.Run("returns error in non-interactive mode when empty", func(t *testing.T) {
		t.Setenv("CODEPUSH_APP_ID", "")
		_, err := ResolveAppIDInteractive("", out)
		require.Error(t, err)
		assert.ErrorContains(t, err, "app ID is required")
	})
}

func TestResolvePlatformInteractive(t *testing.T) {
	out := output.NewTest(io.Discard)

	t.Run("returns value when provided", func(t *testing.T) {
		got, err := ResolvePlatformInteractive("ios", out)
		require.NoError(t, err)
		assert.Equal(t, "ios", got)
	})

	t.Run("returns error in non-interactive mode", func(t *testing.T) {
		_, err := ResolvePlatformInteractive("", out)
		require.Error(t, err)
		assert.ErrorContains(t, err, "--platform")
	})
}

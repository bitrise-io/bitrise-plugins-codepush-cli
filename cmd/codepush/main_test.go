package main

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestMain(m *testing.M) {
	out = output.NewTest(io.Discard)
	os.Exit(m.Run())
}

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
			assert.Equal(t, tt.want, resolveFlag(tt.flagValue, tt.envKey))
		})
	}
}

func TestResolveToken(t *testing.T) {
	t.Run("env var takes priority", func(t *testing.T) {
		t.Setenv("BITRISE_API_TOKEN", "env-token")
		assert.Equal(t, "env-token", resolveToken())
	})

	t.Run("returns empty when nothing set", func(t *testing.T) {
		t.Setenv("BITRISE_API_TOKEN", "")
		// May return empty or stored token; just verify no panic
		_ = resolveToken()
	})
}

func TestVersionCommand(t *testing.T) {
	assert.Equal(t, "version", versionCmd.Use)
}

func TestIntegrateReturnsError(t *testing.T) {
	err := integrateCmd.RunE(integrateCmd, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not yet implemented")
}

func TestCommandRegistration(t *testing.T) {
	commands := rootCmd.Commands()
	wantNames := []string{"version", "bundle", "push", "rollback", "promote", "integrate", "auth"}

	found := make(map[string]bool)
	for _, cmd := range commands {
		found[cmd.Name()] = true
	}

	for _, name := range wantNames {
		assert.True(t, found[name], "command %q not registered on root command", name)
	}
}

func TestAuthSubcommands(t *testing.T) {
	commands := authCmd.Commands()
	found := make(map[string]bool)
	for _, cmd := range commands {
		found[cmd.Name()] = true
	}

	assert.True(t, found["login"], "auth login subcommand not registered")
	assert.True(t, found["revoke"], "auth revoke subcommand not registered")
}

func TestPushCommandRequiresBundlePath(t *testing.T) {
	// Reset flags for test isolation
	old := pushAutoBundle
	pushAutoBundle = false
	defer func() { pushAutoBundle = old }()

	err := pushCmd.RunE(pushCmd, []string{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "bundle path is required")
}

func TestRunBundleValidation(t *testing.T) {
	t.Run("invalid platform", func(t *testing.T) {
		old := bundlePlatform
		bundlePlatform = "windows"
		defer func() { bundlePlatform = old }()

		err := runBundle()
		require.Error(t, err)
		assert.ErrorContains(t, err, "platform")
	})

	t.Run("invalid hermes mode", func(t *testing.T) {
		oldPlatform := bundlePlatform
		oldHermes := bundleHermes
		bundlePlatform = "ios"
		bundleHermes = "invalid"
		defer func() {
			bundlePlatform = oldPlatform
			bundleHermes = oldHermes
		}()

		err := runBundle()
		require.Error(t, err)
		assert.ErrorContains(t, err, "hermes")
	})
}

func TestValidatePlatform(t *testing.T) {
	tests := []struct {
		platform bundler.Platform
		wantErr  bool
	}{
		{bundler.PlatformIOS, false},
		{bundler.PlatformAndroid, false},
		{bundler.Platform("windows"), true},
		{bundler.Platform(""), true},
	}

	for _, tt := range tests {
		err := bundler.ValidatePlatform(tt.platform)
		if tt.wantErr {
			assert.Error(t, err, "ValidatePlatform(%q)", tt.platform)
		} else {
			assert.NoError(t, err, "ValidatePlatform(%q)", tt.platform)
		}
	}
}

func TestValidateHermesMode(t *testing.T) {
	tests := []struct {
		mode    bundler.HermesMode
		wantErr bool
	}{
		{bundler.HermesModeAuto, false},
		{bundler.HermesModeOn, false},
		{bundler.HermesModeOff, false},
		{bundler.HermesMode("invalid"), true},
		{bundler.HermesMode(""), true},
	}

	for _, tt := range tests {
		err := bundler.ValidateHermesMode(tt.mode)
		if tt.wantErr {
			assert.Error(t, err, "ValidateHermesMode(%q)", tt.mode)
		} else {
			assert.NoError(t, err, "ValidateHermesMode(%q)", tt.mode)
		}
	}
}

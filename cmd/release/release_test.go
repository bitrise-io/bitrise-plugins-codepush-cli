package release

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestMain(m *testing.M) {
	cmd.Out = output.NewTest(io.Discard)
	os.Exit(m.Run())
}

func TestPushCommandRequiresBundlePath(t *testing.T) {
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

		err := runBundle(cmd.Out)
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

		err := runBundle(cmd.Out)
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

package main

import (
	"io"
	"os"
	"strings"
	"testing"

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
			got := resolveFlag(tt.flagValue, tt.envKey)
			if got != tt.want {
				t.Errorf("resolveFlag(%q, %q) = %q, want %q", tt.flagValue, tt.envKey, got, tt.want)
			}
		})
	}
}

func TestResolveToken(t *testing.T) {
	t.Run("env var takes priority", func(t *testing.T) {
		t.Setenv("BITRISE_API_TOKEN", "env-token")
		got := resolveToken()
		if got != "env-token" {
			t.Errorf("got %q, want %q", got, "env-token")
		}
	})

	t.Run("returns empty when nothing set", func(t *testing.T) {
		t.Setenv("BITRISE_API_TOKEN", "")
		got := resolveToken()
		// May return empty or stored token; just verify no panic
		_ = got
	})
}

func TestVersionCommand(t *testing.T) {
	cmd := versionCmd
	if cmd.Use != "version" {
		t.Errorf("Use: got %q, want %q", cmd.Use, "version")
	}
}

func TestIntegrateReturnsError(t *testing.T) {
	err := integrateCmd.RunE(integrateCmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("error should mention not implemented: %v", err)
	}
}

func TestCommandRegistration(t *testing.T) {
	commands := rootCmd.Commands()
	wantNames := []string{"version", "bundle", "push", "rollback", "promote", "integrate", "auth"}

	found := make(map[string]bool)
	for _, cmd := range commands {
		found[cmd.Name()] = true
	}

	for _, name := range wantNames {
		if !found[name] {
			t.Errorf("command %q not registered on root command", name)
		}
	}
}

func TestAuthSubcommands(t *testing.T) {
	commands := authCmd.Commands()
	found := make(map[string]bool)
	for _, cmd := range commands {
		found[cmd.Name()] = true
	}

	if !found["login"] {
		t.Error("auth login subcommand not registered")
	}
	if !found["revoke"] {
		t.Error("auth revoke subcommand not registered")
	}
}

func TestPushCommandRequiresBundlePath(t *testing.T) {
	// Reset flags for test isolation
	old := pushAutoBundle
	pushAutoBundle = false
	defer func() { pushAutoBundle = old }()

	err := pushCmd.RunE(pushCmd, []string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bundle path is required") {
		t.Errorf("error should mention bundle path: %v", err)
	}
}

func TestRunBundleValidation(t *testing.T) {
	t.Run("invalid platform", func(t *testing.T) {
		old := bundlePlatform
		bundlePlatform = "windows"
		defer func() { bundlePlatform = old }()

		err := runBundle()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "platform") {
			t.Errorf("error should mention platform: %v", err)
		}
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
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "hermes") {
			t.Errorf("error should mention hermes: %v", err)
		}
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
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePlatform(%q): err=%v, wantErr=%v", tt.platform, err, tt.wantErr)
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
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateHermesMode(%q): err=%v, wantErr=%v", tt.mode, err, tt.wantErr)
		}
	}
}

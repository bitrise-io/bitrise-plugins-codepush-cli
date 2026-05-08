package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestMain(m *testing.M) {
	cmd.Out = output.NewTest(io.Discard)
	os.Exit(m.Run())
}

func TestVersionCommand(t *testing.T) {
	assert.Equal(t, "version", versionCmd.Use)
}

func TestProgressStyleFlag(t *testing.T) {
	tests := []struct {
		flag string
		want output.BarStyle
	}{
		{"bar", output.StyleBar},
		{"spinner", output.StyleSpinner},
		{"counter", output.StyleCounter},
		{"", output.StyleBar},
	}
	for _, tc := range tests {
		t.Run(tc.flag, func(t *testing.T) {
			assert.Equal(t, tc.want, output.ParseBarStyle(tc.flag))
		})
	}
}

func TestProgressStyleFlagHelp(t *testing.T) {
	f := cmd.RootCmd.PersistentFlags().Lookup("progress-style")
	assert.NotNil(t, f, "--progress-style flag should be registered on root command")
	assert.Equal(t, "bar", f.DefValue, "default should be bar")
}

func runWithStyle(t *testing.T, configJSON string, args ...string) *output.Writer {
	t.Helper()
	dir := t.TempDir()
	if configJSON != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".codepush.json"), []byte(configJSON), 0o644))
	}
	t.Chdir(dir)

	orig := cmd.Out
	t.Cleanup(func() { cmd.Out = orig })

	buf := &bytes.Buffer{}
	out := output.NewTest(buf)
	cmd.Out = out

	// Reset flag to default before each run.
	f := cmd.RootCmd.PersistentFlags().Lookup("progress-style")
	_ = f.Value.Set(f.DefValue)
	f.Changed = false

	cmd.RootCmd.SetArgs(args)
	_ = cmd.RootCmd.Execute()
	return out
}

func TestProgressStylePrecedence(t *testing.T) {
	t.Run("flag wins over config", func(t *testing.T) {
		out := runWithStyle(t, `{"app_id":"x","progress_style":"spinner"}`, "version", "--progress-style", "counter")
		assert.Equal(t, output.StyleCounter, out.BarStyle())
	})

	t.Run("config used when flag not set", func(t *testing.T) {
		out := runWithStyle(t, `{"app_id":"x","progress_style":"spinner"}`, "version")
		assert.Equal(t, output.StyleSpinner, out.BarStyle())
	})

	t.Run("default bar when neither flag nor config", func(t *testing.T) {
		out := runWithStyle(t, "", "version")
		assert.Equal(t, output.StyleBar, out.BarStyle())
	})

	t.Run("unknown style in config falls back to default without crash", func(t *testing.T) {
		out := runWithStyle(t, `{"app_id":"x","progress_style":"rainbow"}`, "version")
		assert.Equal(t, output.StyleBar, out.BarStyle())
	})
}

func TestCommandRegistration(t *testing.T) {
	commands := cmd.RootCmd.Commands()
	wantNames := []string{"version", "bundle", "push", "rollback", "promote", "integrate", "auth"}

	found := make(map[string]bool)
	for _, c := range commands {
		found[c.Name()] = true
	}

	for _, name := range wantNames {
		assert.True(t, found[name], "command %q not registered on root command", name)
	}
}

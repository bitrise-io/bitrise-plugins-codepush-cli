package main

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

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

package debug

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

func TestMain(m *testing.M) {
	cmd.Out = output.NewTest(io.Discard)
	os.Exit(m.Run())
}

func TestDebugUnknownPlatform(t *testing.T) {
	err := debugCmd.RunE(debugCmd, []string{"windows"})
	require.Error(t, err)
	require.ErrorContains(t, err, "unknown platform")
	require.ErrorContains(t, err, "windows")
}

func TestDebugAndroidAdbNotFound(t *testing.T) {
	// Clear PATH so adb cannot be found.
	t.Setenv("PATH", t.TempDir())

	err := debugCmd.RunE(debugCmd, []string{"android"})
	require.Error(t, err)
	require.ErrorContains(t, err, "adb not found")
}

func TestDebugIOSXcrunNotFound(t *testing.T) {
	// Clear PATH so xcrun cannot be found.
	t.Setenv("PATH", t.TempDir())

	err := debugCmd.RunE(debugCmd, []string{"ios"})
	require.Error(t, err)
	require.ErrorContains(t, err, "xcrun not found")
}

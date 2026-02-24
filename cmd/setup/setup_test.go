package setup

import (
	"io"
	"os"
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

func TestIntegrateReturnsError(t *testing.T) {
	err := integrateCmd.RunE(integrateCmd, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "not yet implemented")
}

func TestAuthSubcommands(t *testing.T) {
	commands := authCmd.Commands()
	found := make(map[string]bool)
	for _, c := range commands {
		found[c.Name()] = true
	}

	assert.True(t, found["login"], "auth login subcommand not registered")
	assert.True(t, found["revoke"], "auth revoke subcommand not registered")
}

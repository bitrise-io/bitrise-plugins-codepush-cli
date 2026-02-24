package setup

import (
	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
)

func init() {
	cmd.RootCmd.AddGroup(&cobra.Group{ID: cmd.GroupSetup, Title: "Setup:"})
}

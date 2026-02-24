package setup

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
)

var integrateCmd = &cobra.Command{
	Use:   "integrate",
	Short: "Integrate CodePush SDK into a mobile project",
	Long: `Integrate the Bitrise CodePush SDK into your mobile project.

Detects the project type (React Native, Flutter, native iOS/Android)
and configures the SDK accordingly.`,
	GroupID: cmd.GroupSetup,
	RunE: func(c *cobra.Command, args []string) error {
		return fmt.Errorf("integrate command is not yet implemented")
	},
}

func init() {
	cmd.RootCmd.AddCommand(integrateCmd)
}

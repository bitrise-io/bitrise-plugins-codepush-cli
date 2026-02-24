package main

import (
	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
)

var (
	version = "0.1.2"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(c *cobra.Command, args []string) {
		cmd.Out.Println("CodePush CLI %s", version)
		cmd.Out.Info("commit: %s", commit)
		cmd.Out.Info("built: %s", date)
	},
}

func init() {
	cmd.RootCmd.AddCommand(versionCmd)
}

// CodePush CLI - Manage CodePush OTA updates and SDK integration for mobile apps
package main

import (
	"os"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"

	_ "github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd/deployment"
	_ "github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd/packagecmd"
	_ "github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd/release"
	_ "github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd/setup"
)

func main() {
	cmd.Out = output.New()

	if err := cmd.RootCmd.Execute(); err != nil {
		cmd.Out.Error("%v", err)
		os.Exit(1)
	}
}

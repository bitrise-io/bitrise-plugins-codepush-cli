package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// GroupID is a typed alias for command group identifiers.
type GroupID = string

// Command group identifiers for organizing help output.
const (
	GroupRelease    GroupID = "release"
	GroupDeployment GroupID = "deployment"
	GroupUpdate     GroupID = "update"
	GroupSetup      GroupID = "setup"
	GroupDebug      GroupID = "debug"
)

// Out is the shared CLI output writer. Set by main() before Execute().
var Out *output.Writer

// Global flag values, bound to RootCmd's persistent flags.
var (
	AppID      string
	JSONOutput bool
	ServerURL  string
)

// RootCmd is the top-level cobra command.
var RootCmd = &cobra.Command{
	Use:   "codepush",
	Short: "Manage CodePush OTA updates and SDK integration",
	Long: `CodePush CLI manages over-the-air updates for mobile applications
and helps integrate the Bitrise CodePush SDK into your projects.

Use as a standalone CLI or as a Bitrise plugin (bitrise :codepush).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	RootCmd.PersistentFlags().StringVar(&AppID, "app-id", "", "release management app UUID (env: CODEPUSH_APP_ID)")
	RootCmd.PersistentFlags().BoolVarP(&JSONOutput, "json", "j", false, "output results as JSON to stdout")
	RootCmd.PersistentFlags().StringVar(&ServerURL, "server-url", "", "API server base URL (env: CODEPUSH_SERVER_URL)")
}

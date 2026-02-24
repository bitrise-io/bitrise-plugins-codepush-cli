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
	GroupPackage    GroupID = "package"
	GroupSetup      GroupID = "setup"
)

// DefaultAPIURL is the base URL for the CodePush API.
const DefaultAPIURL = "https://api.bitrise.io/release-management/v1"

// Out is the shared CLI output writer. Set by main() before Execute().
var Out *output.Writer

// Global flag values, bound to RootCmd's persistent flags.
var (
	AppID      string
	JSONOutput bool
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
	RootCmd.PersistentFlags().StringVar(&AppID, "app-id", "", "connected app UUID (env: CODEPUSH_APP_ID)")
	RootCmd.PersistentFlags().BoolVar(&JSONOutput, "json", false, "output results as JSON to stdout")
}

// CodePush CLI - Manage CodePush OTA updates and SDK integration for mobile apps
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var (
	version = "0.1.0"
	commit  = "none"
	date    = "unknown"
)

// out is the shared CLI output writer, initialized in main().
var out *output.Writer

// Shared API flags (persistent on root)
var (
	globalAppID string
	globalJSON  bool
)

const defaultAPIURL = "https://api.bitrise.io/release-management/v1"

func main() {
	out = output.New()

	if err := rootCmd.Execute(); err != nil {
		out.Error("%v", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "codepush",
	Short: "Manage CodePush OTA updates and SDK integration",
	Long: `CodePush CLI manages over-the-air updates for mobile applications
and helps integrate the Bitrise CodePush SDK into your projects.

Use as a standalone CLI or as a Bitrise plugin (bitrise :codepush).`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		out.Println("CodePush CLI %s", version)
		out.Info("commit: %s", commit)
		out.Info("built: %s", date)
	},
}

var integrateCmd = &cobra.Command{
	Use:   "integrate",
	Short: "Integrate CodePush SDK into a mobile project",
	Long: `Integrate the Bitrise CodePush SDK into your mobile project.

Detects the project type (React Native, Flutter, native iOS/Android)
and configures the SDK accordingly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("integrate command is not yet implemented")
	},
}

func init() {
	registerCommandTree()
	registerGlobalFlags()
	registerBundleFlags()
	registerPushFlags()
	registerRollbackFlags()
	registerPromoteFlags()
	registerPatchFlags()
	registerDeploymentFlags()
	registerPackageFlags()
	registerAuthFlags()
}

func registerCommandTree() {
	rootCmd.AddGroup(
		&cobra.Group{ID: "release", Title: "Release Management:"},
		&cobra.Group{ID: "deployment", Title: "Deployment Management:"},
		&cobra.Group{ID: "package", Title: "Package Management:"},
		&cobra.Group{ID: "setup", Title: "Setup:"},
	)

	pushCmd.GroupID = "release"
	rollbackCmd.GroupID = "release"
	promoteCmd.GroupID = "release"
	patchCmd.GroupID = "release"
	bundleCmd.GroupID = "release"
	deploymentCmd.GroupID = "deployment"
	packageCmd.GroupID = "package"
	authCmd.GroupID = "setup"
	integrateCmd.GroupID = "setup"

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(bundleCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(promoteCmd)
	rootCmd.AddCommand(patchCmd)
	rootCmd.AddCommand(integrateCmd)
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authRevokeCmd)
	rootCmd.AddCommand(deploymentCmd)
	deploymentCmd.AddCommand(deploymentListCmd)
	deploymentCmd.AddCommand(deploymentAddCmd)
	deploymentCmd.AddCommand(deploymentInfoCmd)
	deploymentCmd.AddCommand(deploymentRenameCmd)
	deploymentCmd.AddCommand(deploymentRemoveCmd)
	deploymentCmd.AddCommand(deploymentHistoryCmd)
	deploymentCmd.AddCommand(deploymentClearCmd)
	rootCmd.AddCommand(packageCmd)
	packageCmd.AddCommand(packageInfoCmd)
	packageCmd.AddCommand(packageStatusCmd)
	packageCmd.AddCommand(packageRemoveCmd)
}

func registerGlobalFlags() {
	rootCmd.PersistentFlags().StringVar(&globalAppID, "app-id", "", "connected app UUID (env: CODEPUSH_APP_ID)")
	rootCmd.PersistentFlags().BoolVar(&globalJSON, "json", false, "output results as JSON to stdout")
}

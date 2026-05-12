package cmd

import (
	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/config"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var progressStyle string

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

// Version is the CLI version string. Set by main() before Execute().
var Version string

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
	PersistentPreRunE: func(c *cobra.Command, _ []string) error {
		style := progressStyle
		if !c.Root().PersistentFlags().Changed("progress-style") {
			if cfg, err := config.Load(); err != nil {
				Out.Warning("reading %s: %s", config.FileName, err)
			} else if cfg != nil && cfg.ProgressStyle != "" {
				if !output.IsValidBarStyle(cfg.ProgressStyle) {
					Out.Warning("unknown progress_style %q in %s, using default", cfg.ProgressStyle, config.FileName)
				} else {
					style = cfg.ProgressStyle
				}
			}
		}
		Out.SetBarStyle(output.ParseBarStyle(style))
		return nil
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&AppID, "app-id", "", "release management app UUID (env: CODEPUSH_APP_ID)")
	RootCmd.PersistentFlags().BoolVarP(&JSONOutput, "json", "j", false, "output results as JSON to stdout")
	RootCmd.PersistentFlags().StringVar(&ServerURL, "server-url", "", "API server base URL (env: CODEPUSH_SERVER_URL)")
	RootCmd.PersistentFlags().StringVar(&progressStyle, "progress-style", "bar", "progress indicator style: bar, spinner, counter")
}

package setup

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/cmdutil"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/config"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration",
	Long: `Create a .codepush.json file in the current directory.

This stores the app ID so you don't need to pass --app-id on every command.
The file is safe to commit to version control.`,
	GroupID: cmd.GroupSetup,
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, err := cmdutil.ResolveAppIDInteractive(cmd.AppID, out)
		if err != nil {
			return err
		}

		return writeProjectConfig(appID)
	},
}

func writeProjectConfig(appID string) error {
	out := cmd.Out

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining working directory: %w", err)
	}

	cfgPath, err := config.FilePath()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	if !initForce {
		if _, err := os.Stat(cfgPath); err == nil {
			return fmt.Errorf("%s already exists: use --force to overwrite", config.FileName)
		}
	}

	cfg := &config.ProjectConfig{AppID: appID}
	if err := config.Save(dir, cfg); err != nil {
		return err
	}

	if cmd.JSONOutput {
		return cmdutil.OutputJSON(cfg)
	}

	out.Success("Created %s", config.FileName)
	out.Info("App ID: %s", appID)
	out.Info("Path: %s", cfgPath)
	return nil
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config file")
	cmd.RootCmd.AddCommand(initCmd)
}

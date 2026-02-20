package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/config"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration",
	Long: `Create a .codepush.json file in the current directory.

This stores the app ID so you don't need to pass --app-id on every command.
The file is safe to commit to version control.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, err := resolveAppIDInteractive()
		if err != nil {
			return err
		}

		return writeProjectConfig(appID)
	},
}

func writeProjectConfig(appID string) error {
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

	if globalJSON {
		return outputJSON(cfg)
	}

	out.Success("Created %s", config.FileName)
	out.Info("App ID: %s", appID)
	out.Info("Path: %s", cfgPath)
	return nil
}

func registerInitFlags() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config file")
}

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Rollback command flags
var (
	rollbackDeployment    string
	rollbackTargetRelease string
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to a previous release",
	Long: `Rollback the current deployment to a previous release version.

Creates a new release that mirrors a previous version. By default,
rolls back to the immediately previous release. Use --target-release
to specify a specific version label (e.g. v3).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appID := resolveFlag(globalAppID, "CODEPUSH_APP_ID")
		deployment := resolveFlag(rollbackDeployment, "CODEPUSH_DEPLOYMENT")
		token := resolveToken()

		opts := &codepush.RollbackOptions{
			AppID:        appID,
			DeploymentID: deployment,
			Token:        token,
			TargetLabel:  rollbackTargetRelease,
		}

		client := codepush.NewHTTPClient(defaultAPIURL, opts.Token)
		result, err := codepush.Rollback(cmd.Context(), client, opts, out)
		if err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		if globalJSON {
			return outputJSON(result)
		}

		out.Success("Rollback successful")
		out.Result([]output.KeyValue{
			{Key: "Package ID", Value: result.PackageID},
			{Key: "Label", Value: result.Label},
			{Key: "App version", Value: result.AppVersion},
		})

		if bitrise.IsBitriseEnvironment() {
			exportEnvVars(map[string]string{
				"CODEPUSH_PACKAGE_ID":  result.PackageID,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			})
		}

		return nil
	},
}

func registerRollbackFlags() {
	rollbackCmd.Flags().StringVar(&rollbackDeployment, "deployment", "", "deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	rollbackCmd.Flags().StringVar(&rollbackTargetRelease, "target-release", "", "specific release label to rollback to (e.g. v3)")
}

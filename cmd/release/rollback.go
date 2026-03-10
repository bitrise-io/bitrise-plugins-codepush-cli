package release

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/cmdutil"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

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
	GroupID: cmd.GroupRelease,
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		serverURL := cmdutil.ResolveServerURL(cmd.ServerURL, out)
		client := codepush.NewHTTPClient(cmdutil.APIURL(serverURL), token, cmd.Version)

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, rollbackDeployment, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		opts := &codepush.RollbackOptions{
			AppID:        appID,
			DeploymentID: deploymentID,
			Token:        token,
			TargetLabel:  rollbackTargetRelease,
		}

		result, err := codepush.Rollback(c.Context(), client, opts, out)
		if err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(result)
		}

		out.Success("Rollback successful")
		out.Result([]output.KeyValue{
			{Key: "Update ID", Value: result.UpdateID},
			{Key: "Label", Value: result.Label},
			{Key: "App version", Value: result.AppVersion},
		})

		if bitrise.IsBitriseEnvironment() {
			cmdutil.ExportEnvVars(map[string]string{
				"CODEPUSH_UPDATE_ID":   result.UpdateID,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			}, out)
		}

		return nil
	},
}

func init() {
	rollbackCmd.Flags().StringVarP(&rollbackDeployment, "deployment", "d", "", "deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	rollbackCmd.Flags().StringVarP(&rollbackTargetRelease, "target-release", "r", "", "specific release label to rollback to (e.g. v3)")
	cmd.RootCmd.AddCommand(rollbackCmd)
}

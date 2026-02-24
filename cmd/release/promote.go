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
	promoteSourceDeployment string
	promoteDestDeployment   string
	promoteLabel            string
	promoteAppVersion       string
	promoteDescription      string
	promoteMandatory        string
	promoteDisabled         string
	promoteRollout          string
)

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote a release from one deployment to another",
	Long: `Promote a release from a source deployment to a destination deployment.

Copies the latest (or specified) release from the source deployment to the
destination deployment. Override metadata like rollout percentage, mandatory
flag, or description for the promoted release.

Example: promote from Staging to Production after testing.`,
	GroupID: cmd.GroupRelease,
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

		sourceDeploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, promoteSourceDeployment, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		destDeploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, promoteDestDeployment, "", out)
		if err != nil {
			return err
		}

		opts := &codepush.PromoteOptions{
			AppID:              appID,
			SourceDeploymentID: sourceDeploymentID,
			DestDeploymentID:   destDeploymentID,
			Token:              token,
			Label:              promoteLabel,
			AppVersion:         promoteAppVersion,
			Description:        promoteDescription,
			Mandatory:          promoteMandatory,
			Disabled:           promoteDisabled,
			Rollout:            promoteRollout,
		}

		result, err := codepush.Promote(c.Context(), client, opts, out)
		if err != nil {
			return fmt.Errorf("promote failed: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(result)
		}

		out.Success("Promote successful")
		out.Result([]output.KeyValue{
			{Key: "Package ID", Value: result.PackageID},
			{Key: "Label", Value: result.Label},
			{Key: "App version", Value: result.AppVersion},
			{Key: "Destination", Value: result.DestDeployment},
		})

		if bitrise.IsBitriseEnvironment() {
			cmdutil.ExportEnvVars(map[string]string{
				"CODEPUSH_PACKAGE_ID":  result.PackageID,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			}, out)
		}

		return nil
	},
}

func init() {
	promoteCmd.Flags().StringVar(&promoteSourceDeployment, "source-deployment", "", "source deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	promoteCmd.Flags().StringVar(&promoteDestDeployment, "destination-deployment", "", "destination deployment name or UUID (required)")
	promoteCmd.Flags().StringVar(&promoteLabel, "label", "", "specific release label to promote (e.g. v5)")
	promoteCmd.Flags().StringVar(&promoteAppVersion, "app-version", "", "override target app version")
	promoteCmd.Flags().StringVar(&promoteDescription, "description", "", "override release description")
	promoteCmd.Flags().StringVar(&promoteMandatory, "mandatory", "", "override mandatory flag (true/false)")
	promoteCmd.Flags().StringVar(&promoteDisabled, "disabled", "", "override disabled flag (true/false)")
	promoteCmd.Flags().StringVar(&promoteRollout, "rollout", "", "override rollout percentage (1-100)")
	cmd.RootCmd.AddCommand(promoteCmd)
}

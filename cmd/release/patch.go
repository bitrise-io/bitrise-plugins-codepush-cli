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
	patchDeployment  string
	patchLabel       string
	patchRollout     string
	patchMandatory   string
	patchDisabled    string
	patchDescription string
	patchAppVersion  string
)

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: "Update metadata on an existing release",
	Long: `Update metadata on an existing release without re-deploying.

Adjust rollout percentage, toggle mandatory/disabled flags, update the
description, or change the target app version on a live release.

By default, patches the latest release. Use --label to target a specific version.

Examples:
  codepush patch --deployment Production --rollout 50
  codepush patch --deployment Staging --label v5 --mandatory true --disabled false`,
	GroupID: cmd.GroupRelease,
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, patchDeployment, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		opts := &codepush.PatchOptions{
			AppID:        appID,
			DeploymentID: deploymentID,
			Token:        token,
			Label:        patchLabel,
			Rollout:      patchRollout,
			Mandatory:    patchMandatory,
			Disabled:     patchDisabled,
			Description:  patchDescription,
			AppVersion:   patchAppVersion,
		}

		result, err := codepush.Patch(c.Context(), client, opts, out)
		if err != nil {
			return fmt.Errorf("patch failed: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(result)
		}

		out.Success("Patch successful")
		out.Result([]output.KeyValue{
			{Key: "Package ID", Value: result.PackageID},
			{Key: "Label", Value: result.Label},
			{Key: "App version", Value: result.AppVersion},
			{Key: "Rollout", Value: fmt.Sprintf("%d%%", result.Rollout)},
			{Key: "Mandatory", Value: fmt.Sprintf("%v", result.Mandatory)},
			{Key: "Disabled", Value: fmt.Sprintf("%v", result.Disabled)},
		})

		if bitrise.IsBitriseEnvironment() {
			cmdutil.ExportDeploySummary("codepush-patch-summary.json", result, out)
			cmdutil.ExportEnvVars(map[string]string{
				"CODEPUSH_PACKAGE_ID":  result.PackageID,
				"CODEPUSH_LABEL":       result.Label,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			}, out)
		}

		return nil
	},
}

func init() {
	patchCmd.Flags().StringVar(&patchDeployment, "deployment", "", "deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	patchCmd.Flags().StringVar(&patchLabel, "label", "", "specific release label to patch (e.g. v5, defaults to latest)")
	patchCmd.Flags().StringVar(&patchRollout, "rollout", "", "rollout percentage (1-100)")
	patchCmd.Flags().StringVar(&patchMandatory, "mandatory", "", "mark update as mandatory (true/false)")
	patchCmd.Flags().StringVar(&patchDisabled, "disabled", "", "disable update (true/false)")
	patchCmd.Flags().StringVar(&patchDescription, "description", "", "update description")
	patchCmd.Flags().StringVar(&patchAppVersion, "app-version", "", "target app version")
	cmd.RootCmd.AddCommand(patchCmd)
}

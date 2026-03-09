package release

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/cmdutil"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var (
	pushAutoBundle  bool
	pushDeployment  string
	pushAppVersion  string
	pushDescription string
	pushMandatory   bool
	pushRollout     int
	pushDisabled    bool
)

var pushCmd = &cobra.Command{
	Use:   "push [bundle-path]",
	Short: "Push an OTA update",
	Long: `Push an over-the-air update to your mobile application.

Uploads the specified bundle and deploys it to the CodePush server
for distribution to connected devices.

Use --bundle to automatically generate the JavaScript bundle before pushing.`,
	GroupID: cmd.GroupRelease,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		if pushAutoBundle {
			platform, err := cmdutil.ResolvePlatformInteractive(bundlePlatform, out)
			if err != nil {
				return err
			}
			bundlePlatform = platform

			result, err := runBundleWithOpts(out)
			if err != nil {
				return fmt.Errorf("bundling failed: %w", err)
			}

			out.Info("Bundle created at: %s", result.OutputDir)
			args = []string{result.OutputDir}
		}

		if len(args) == 0 {
			return errors.New("bundle path is required: provide as argument or use --bundle to generate one")
		}

		bundlePath, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolving bundle path: %w", err)
		}

		if bundlePrivateKeyPath != "" {
			out.Step("Signing bundle")
			if err := bundler.SignBundle(bundlePath, bundlePrivateKeyPath); err != nil {
				return fmt.Errorf("signing bundle: %w", err)
			}
			out.Info("Signed: %s/.codepushrelease", bundlePath)
		}

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		serverURL := cmdutil.ResolveServerURL(cmd.ServerURL, out)
		client := codepush.NewHTTPClient(cmdutil.APIURL(serverURL), token)

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, pushDeployment, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		appVersion, err := cmdutil.ResolveInputInteractive(pushAppVersion, "App version", "1.0.0", out)
		if err != nil {
			return err
		}

		opts := &codepush.PushOptions{
			AppID:        appID,
			DeploymentID: deploymentID,
			Token:        token,
			AppVersion:   appVersion,
			Description:  pushDescription,
			Mandatory:    pushMandatory,
			Rollout:      pushRollout,
			Disabled:     pushDisabled,
			BundlePath:   bundlePath,
		}

		result, err := codepush.Push(c.Context(), client, opts, out)
		if err != nil {
			return fmt.Errorf("push failed: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(result)
		}

		out.Success("Push successful")
		out.Result([]output.KeyValue{
			{Key: "Update ID", Value: result.UpdateID},
			{Key: "App version", Value: result.AppVersion},
			{Key: "Status", Value: result.Status},
		})

		if bitrise.IsBitriseEnvironment() {
			cmdutil.ExportDeploySummary("codepush-push-summary.json", result, out)
			cmdutil.ExportEnvVars(map[string]string{
				"CODEPUSH_UPDATE_ID":   result.UpdateID,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			}, out)
		}

		return nil
	},
}

func init() {
	pushCmd.Flags().BoolVar(&pushAutoBundle, "bundle", false, "bundle JavaScript before pushing")
	registerPushBundleFlagsOn(pushCmd)
	pushCmd.Flags().StringVarP(&pushDeployment, "deployment", "d", "", "deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	pushCmd.Flags().StringVarP(&pushAppVersion, "app-version", "t", "", "target app version (e.g. 1.0.0)")
	pushCmd.Flags().StringVar(&pushDescription, "description", "", "update description")
	pushCmd.Flags().BoolVarP(&pushMandatory, "mandatory", "m", false, "mark update as mandatory")
	pushCmd.Flags().IntVarP(&pushRollout, "rollout", "r", 100, "rollout percentage (1-100)")
	pushCmd.Flags().BoolVarP(&pushDisabled, "disabled", "x", false, "disable update after upload")
	cmd.RootCmd.AddCommand(pushCmd)
}

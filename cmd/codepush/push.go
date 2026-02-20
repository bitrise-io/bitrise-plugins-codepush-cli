package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Push command flags
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

Packages the specified bundle and deploys it to the CodePush server
for distribution to connected devices.

Use --bundle to automatically generate the JavaScript bundle before pushing.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if pushAutoBundle {
			if bundlePlatform == "" {
				return fmt.Errorf("--platform is required when using --bundle")
			}

			result, err := runBundleWithOpts()
			if err != nil {
				return fmt.Errorf("bundling failed: %w", err)
			}

			out.Info("Bundle created at: %s", result.OutputDir)
			args = []string{result.OutputDir}
		}

		if len(args) == 0 {
			return fmt.Errorf("bundle path is required: provide as argument or use --bundle to generate one")
		}

		bundlePath, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolving bundle path: %w", err)
		}

		appID := resolveAppID()
		deployment := resolveFlag(pushDeployment, "CODEPUSH_DEPLOYMENT")
		token := resolveToken()

		opts := &codepush.PushOptions{
			AppID:        appID,
			DeploymentID: deployment,
			Token:        token,
			AppVersion:   pushAppVersion,
			Description:  pushDescription,
			Mandatory:    pushMandatory,
			Rollout:      pushRollout,
			Disabled:     pushDisabled,
			BundlePath:   bundlePath,
		}

		client := codepush.NewHTTPClient(defaultAPIURL, opts.Token)
		result, err := codepush.Push(cmd.Context(), client, opts, out)
		if err != nil {
			return fmt.Errorf("push failed: %w", err)
		}

		if globalJSON {
			return outputJSON(result)
		}

		out.Success("Push successful")
		out.Result([]output.KeyValue{
			{Key: "Package ID", Value: result.PackageID},
			{Key: "App version", Value: result.AppVersion},
			{Key: "Status", Value: result.Status},
		})

		if bitrise.IsBitriseEnvironment() {
			exportDeploySummary("codepush-push-summary.json", result)
			exportEnvVars(map[string]string{
				"CODEPUSH_PACKAGE_ID":  result.PackageID,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			})
		}

		return nil
	},
}

func registerPushFlags() {
	pushCmd.Flags().BoolVar(&pushAutoBundle, "bundle", false, "bundle JavaScript before pushing")
	pushCmd.Flags().StringVar(&bundlePlatform, "platform", "", "target platform for bundling: ios or android")
	pushCmd.Flags().StringVar(&bundleOutputDir, "output-dir", bundler.DefaultOutputDir, "output directory for the bundle")
	pushCmd.Flags().StringVar(&bundleHermes, "hermes", "auto", "Hermes bytecode compilation: auto, on, or off")
	pushCmd.Flags().StringVar(&bundleProjectDir, "project-dir", "", "project root directory (defaults to current directory)")
	pushCmd.Flags().StringVar(&pushDeployment, "deployment", "", "deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	pushCmd.Flags().StringVar(&pushAppVersion, "app-version", "", "target app version (e.g. 1.0.0)")
	pushCmd.Flags().StringVar(&pushDescription, "description", "", "update description")
	pushCmd.Flags().BoolVar(&pushMandatory, "mandatory", false, "mark update as mandatory")
	pushCmd.Flags().IntVar(&pushRollout, "rollout", 100, "rollout percentage (1-100)")
	pushCmd.Flags().BoolVar(&pushDisabled, "disabled", false, "disable update after upload")
}

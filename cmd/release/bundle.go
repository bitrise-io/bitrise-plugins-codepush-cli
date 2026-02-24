package release

import (
	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/cmdutil"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Bundle JavaScript for an OTA update",
	Long: `Bundle the JavaScript code and assets for a React Native or Expo project.

Auto-detects the project type, entry file, and Hermes configuration.
Produces a directory containing the bundle, assets, and optional source maps
ready for use with 'codepush push'.`,
	GroupID: cmd.GroupRelease,
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out
		return runBundle(out)
	},
}

func init() {
	registerBundleFlagsOn(bundleCmd)
	cmd.RootCmd.AddCommand(bundleCmd)
}

func runBundle(out *output.Writer) error {
	platform, err := cmdutil.ResolvePlatformInteractive(bundlePlatform, out)
	if err != nil {
		return err
	}
	bundlePlatform = platform

	if err := bundler.ValidatePlatform(bundler.Platform(bundlePlatform)); err != nil {
		return err
	}
	if err := bundler.ValidateHermesMode(bundler.HermesMode(bundleHermes)); err != nil {
		return err
	}

	result, err := runBundleWithOpts(out)
	if err != nil {
		return err
	}

	if cmd.JSONOutput {
		summary := struct {
			Platform      string `json:"platform"`
			ProjectType   string `json:"project_type"`
			OutputDir     string `json:"output_dir"`
			BundlePath    string `json:"bundle_path"`
			AssetsDir     string `json:"assets_dir"`
			SourcemapPath string `json:"sourcemap_path,omitempty"`
			HermesApplied bool   `json:"hermes_applied"`
		}{
			Platform:      string(result.Platform),
			ProjectType:   result.ProjectType.String(),
			OutputDir:     result.OutputDir,
			BundlePath:    result.BundlePath,
			AssetsDir:     result.AssetsDir,
			SourcemapPath: result.SourcemapPath,
			HermesApplied: result.HermesApplied,
		}
		return cmdutil.OutputJSON(summary)
	}

	out.Success("Bundle created successfully")
	out.Result([]output.KeyValue{
		{Key: "Output", Value: result.OutputDir},
		{Key: "Bundle", Value: result.BundlePath},
	})
	if result.SourcemapPath != "" {
		out.Info("Sourcemap: %s", result.SourcemapPath)
	}
	if result.HermesApplied {
		out.Info("Hermes: compiled")
	}

	if bitrise.IsBitriseEnvironment() {
		cmdutil.ExportDeploySummary("codepush-bundle-summary.json", struct {
			Platform      string `json:"platform"`
			ProjectType   string `json:"project_type"`
			BundlePath    string `json:"bundle_path"`
			AssetsDir     string `json:"assets_dir"`
			SourcemapPath string `json:"sourcemap_path,omitempty"`
			HermesApplied bool   `json:"hermes_applied"`
		}{
			Platform:      string(result.Platform),
			ProjectType:   result.ProjectType.String(),
			BundlePath:    result.BundlePath,
			AssetsDir:     result.AssetsDir,
			SourcemapPath: result.SourcemapPath,
			HermesApplied: result.HermesApplied,
		}, out)
	}

	return nil
}

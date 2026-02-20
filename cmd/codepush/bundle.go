package main

import (
	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Bundle flags: shared between "bundle" and "push --bundle" commands.
// Both commands bind to the same variables so that "push --bundle" reuses
// the bundling pipeline with identical flag names.
var (
	bundlePlatform         string
	bundleEntryFile        string
	bundleOutputDir        string
	bundleBundleName       string
	bundleDev              bool
	bundleSourcemap        bool
	bundleHermes           string
	bundleExtraBundlerOpts []string
	bundleProjectDir       string
	bundleMetroConfig      string
	bundleSkipInstall      bool
)

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Bundle JavaScript for an OTA update",
	Long: `Bundle the JavaScript code and assets for a React Native or Expo project.

Auto-detects the project type, entry file, and Hermes configuration.
Produces a directory containing the bundle, assets, and optional source maps
ready for use with 'codepush push'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBundle()
	},
}

func registerBundleFlags() {
	bundleCmd.Flags().StringVar(&bundlePlatform, "platform", "", "target platform: ios or android")
	bundleCmd.Flags().StringVar(&bundleEntryFile, "entry-file", "", "path to the entry JS file (auto-detected if not set)")
	bundleCmd.Flags().StringVar(&bundleOutputDir, "output-dir", bundler.DefaultOutputDir, "output directory for the bundle")
	bundleCmd.Flags().StringVar(&bundleBundleName, "bundle-name", "", "custom bundle filename (platform default if not set)")
	bundleCmd.Flags().BoolVar(&bundleDev, "dev", false, "enable development mode")
	bundleCmd.Flags().BoolVar(&bundleSourcemap, "sourcemap", true, "generate source maps")
	bundleCmd.Flags().StringVar(&bundleHermes, "hermes", "auto", "Hermes bytecode compilation: auto, on, or off")
	bundleCmd.Flags().StringArrayVar(&bundleExtraBundlerOpts, "extra-bundler-option", nil, "additional flags passed to the bundler (repeatable)")
	bundleCmd.Flags().StringVar(&bundleProjectDir, "project-dir", "", "project root directory (defaults to current directory)")
	bundleCmd.Flags().StringVar(&bundleMetroConfig, "config", "", "path to Metro config file (auto-detected if not set)")
	bundleCmd.Flags().BoolVar(&bundleSkipInstall, "skip-install", false, "skip running package manager install before bundling")
}

func runBundle() error {
	platform, err := resolvePlatformInteractive(bundlePlatform)
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

	result, err := runBundleWithOpts()
	if err != nil {
		return err
	}

	if globalJSON {
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
		return outputJSON(summary)
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
		exportDeploySummary("codepush-bundle-summary.json", struct {
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
		})
	}

	return nil
}

func runBundleWithOpts() (*bundler.BundleResult, error) {
	opts := &bundler.BundleOptions{
		Platform:         bundler.Platform(bundlePlatform),
		EntryFile:        bundleEntryFile,
		OutputDir:        bundleOutputDir,
		BundleName:       bundleBundleName,
		Dev:              bundleDev,
		Sourcemap:        bundleSourcemap,
		HermesMode:       bundler.HermesMode(bundleHermes),
		ExtraBundlerOpts: bundleExtraBundlerOpts,
		ProjectDir:       bundleProjectDir,
		MetroConfig:      bundleMetroConfig,
		SkipInstall:      bundleSkipInstall,
	}

	return bundler.Run(opts, out)
}

package release

import (
	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Shared bundle flags: used by both "bundle" and "push --bundle" commands.
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

func init() {
	cmd.RootCmd.AddGroup(&cobra.Group{ID: cmd.GroupRelease, Title: "Release Management:"})
}

// registerBundleFlagsOn registers the full set of bundle flags on a command.
func registerBundleFlagsOn(c *cobra.Command) {
	c.Flags().StringVar(&bundlePlatform, "platform", "", "target platform: ios or android")
	c.Flags().StringVar(&bundleEntryFile, "entry-file", "", "path to the entry JS file (auto-detected if not set)")
	c.Flags().StringVar(&bundleOutputDir, "output-dir", bundler.DefaultOutputDir, "output directory for the bundle")
	c.Flags().StringVar(&bundleBundleName, "bundle-name", "", "custom bundle filename (platform default if not set)")
	c.Flags().BoolVar(&bundleDev, "dev", false, "enable development mode")
	c.Flags().BoolVar(&bundleSourcemap, "sourcemap", true, "generate source maps")
	c.Flags().StringVar(&bundleHermes, "hermes", "auto", "Hermes bytecode compilation: auto, on, or off")
	c.Flags().StringArrayVar(&bundleExtraBundlerOpts, "extra-bundler-option", nil, "additional flags passed to the bundler (repeatable)")
	c.Flags().StringVar(&bundleProjectDir, "project-dir", "", "project root directory (defaults to current directory)")
	c.Flags().StringVar(&bundleMetroConfig, "config", "", "path to Metro config file (auto-detected if not set)")
	c.Flags().BoolVar(&bundleSkipInstall, "skip-install", false, "skip running package manager install before bundling")
}

// registerPushBundleFlagsOn registers the subset of bundle flags used by push --bundle.
func registerPushBundleFlagsOn(c *cobra.Command) {
	c.Flags().StringVar(&bundlePlatform, "platform", "", "target platform for bundling: ios or android")
	c.Flags().StringVar(&bundleOutputDir, "output-dir", bundler.DefaultOutputDir, "output directory for the bundle")
	c.Flags().StringVar(&bundleHermes, "hermes", "auto", "Hermes bytecode compilation: auto, on, or off")
	c.Flags().StringVar(&bundleProjectDir, "project-dir", "", "project root directory (defaults to current directory)")
	c.Flags().BoolVar(&bundleSkipInstall, "skip-install", false, "skip running package manager install before bundling")
}

func runBundleWithOpts(out *output.Writer) (*bundler.BundleResult, error) {
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

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
	bundleMinify           bool
	bundleResetCache       bool
	bundleSourcemap        bool
	bundleSourcemapOutput  string
	bundleHermes           string
	bundleExtraBundlerOpts []string
	bundleExtraHermesFlags []string
	bundleProjectDir       string
	bundleMetroConfig      string
	bundleSkipInstall      bool
	bundleGradleFile       string
	bundlePodFile          string
)

func init() {
	cmd.RootCmd.AddGroup(&cobra.Group{ID: cmd.GroupRelease, Title: "Release Management:"})
}

// registerBundleFlagsOn registers the full set of bundle flags on a command.
func registerBundleFlagsOn(c *cobra.Command) {
	c.Flags().StringVarP(&bundlePlatform, "platform", "p", "", "target platform: ios or android")
	c.Flags().StringVarP(&bundleEntryFile, "entry-file", "e", "", "path to the entry JS file (auto-detected if not set)")
	c.Flags().StringVarP(&bundleOutputDir, "output-dir", "o", bundler.DefaultOutputDir, "output directory for the bundle")
	c.Flags().StringVarP(&bundleBundleName, "bundle-name", "b", "", "custom bundle filename (platform default if not set)")
	c.Flags().BoolVar(&bundleDev, "dev", false, "enable development mode (also controls minification on React Native: false = minified)")
	c.Flags().BoolVar(&bundleMinify, "minify", false, "minify the bundle (Expo only)")
	c.Flags().BoolVar(&bundleResetCache, "reset-cache", true, "clear Metro bundler cache before bundling")
	c.Flags().BoolVar(&bundleSourcemap, "sourcemap", true, "generate source maps")
	c.Flags().StringVarP(&bundleSourcemapOutput, "sourcemap-output", "s", "", "override sourcemap output path (implies --sourcemap)")
	c.Flags().StringVar(&bundleHermes, "hermes", "auto", "Hermes bytecode compilation: auto, on, or off")
	c.Flags().StringArrayVar(&bundleExtraBundlerOpts, "extra-bundler-option", nil, "additional flags passed to the bundler (repeatable)")
	c.Flags().StringArrayVar(&bundleExtraHermesFlags, "extra-hermes-flag", nil, "additional flags passed to hermesc (repeatable; distinct from --extra-bundler-option which targets Metro)")
	c.Flags().StringVar(&bundleProjectDir, "project-dir", "", "project root directory (defaults to current directory)")
	c.Flags().StringVarP(&bundleMetroConfig, "config", "c", "", "path to Metro config file (auto-detected if not set)")
	c.Flags().BoolVar(&bundleSkipInstall, "skip-install", false, "skip running package manager install before bundling")
	c.Flags().StringVarP(&bundleGradleFile, "gradle-file", "g", "", "override path to build.gradle used for Android Hermes auto-detection")
	c.Flags().StringVar(&bundlePodFile, "pod-file", "", "override path to Podfile used for iOS Hermes auto-detection")
}

// registerPushBundleFlagsOn registers the subset of bundle flags used by push --bundle.
func registerPushBundleFlagsOn(c *cobra.Command) {
	c.Flags().StringVarP(&bundlePlatform, "platform", "p", "", "target platform for bundling: ios or android")
	c.Flags().StringVarP(&bundleOutputDir, "output-dir", "o", bundler.DefaultOutputDir, "output directory for the bundle")
	c.Flags().StringVar(&bundleHermes, "hermes", "auto", "Hermes bytecode compilation: auto, on, or off")
	c.Flags().BoolVar(&bundleMinify, "minify", false, "minify the bundle (Expo only)")
	c.Flags().BoolVar(&bundleResetCache, "reset-cache", true, "clear Metro bundler cache before bundling")
	c.Flags().StringVar(&bundleProjectDir, "project-dir", "", "project root directory (defaults to current directory)")
	c.Flags().BoolVar(&bundleSkipInstall, "skip-install", false, "skip running package manager install before bundling")
	c.Flags().StringVarP(&bundleGradleFile, "gradle-file", "g", "", "override path to build.gradle used for Android Hermes auto-detection")
	c.Flags().StringVar(&bundlePodFile, "pod-file", "", "override path to Podfile used for iOS Hermes auto-detection")
}

func runBundleWithOpts(out *output.Writer) (*bundler.BundleResult, error) {
	opts := &bundler.BundleOptions{
		Platform:         bundler.Platform(bundlePlatform),
		EntryFile:        bundleEntryFile,
		OutputDir:        bundleOutputDir,
		BundleName:       bundleBundleName,
		Dev:              bundleDev,
		Minify:           bundleMinify,
		ResetCache:       bundleResetCache,
		Sourcemap:        bundleSourcemap,
		SourcemapOutput:  bundleSourcemapOutput,
		HermesMode:       bundler.HermesMode(bundleHermes),
		ExtraBundlerOpts: bundleExtraBundlerOpts,
		ExtraHermesFlags: bundleExtraHermesFlags,
		ProjectDir:       bundleProjectDir,
		MetroConfig:      bundleMetroConfig,
		SkipInstall:      bundleSkipInstall,
		GradleFile:       bundleGradleFile,
		PodFile:          bundlePodFile,
	}

	return bundler.Run(opts, out)
}

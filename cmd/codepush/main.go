// CodePush CLI - Manage CodePush OTA updates and SDK integration for mobile apps
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/auth"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	commit  = "none"
	date    = "unknown"
)

// Bundle command flags
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
)

// Auth command flags
var (
	authLoginToken string
)

// Push command flags
var (
	pushAutoBundle  bool
	pushAppID       string
	pushDeployment  string
	pushToken       string
	pushAppVersion  string
	pushDescription string
	pushMandatory   bool
	pushRollout     int
	pushDisabled    bool
	pushAPIURL      string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "codepush",
	Short: "Manage CodePush OTA updates and SDK integration",
	Long: `CodePush CLI manages over-the-air updates for mobile applications
and helps integrate the Bitrise CodePush SDK into your projects.

Use as a standalone CLI or as a Bitrise plugin (bitrise :codepush).`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("CodePush CLI %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built: %s\n", date)
	},
}

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

			fmt.Fprintf(os.Stderr, "Bundle created at: %s\n\n", result.OutputDir)
			args = []string{result.OutputDir}
		}

		if len(args) == 0 {
			return fmt.Errorf("bundle path is required: provide as argument or use --bundle to generate one")
		}

		bundlePath, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolving bundle path: %w", err)
		}

		appID := resolveFlag(pushAppID, "CODEPUSH_APP_ID")
		deployment := resolveFlag(pushDeployment, "CODEPUSH_DEPLOYMENT")
		token := resolveToken(pushToken)

		opts := &codepush.PushOptions{
			AppID:        appID,
			DeploymentID: deployment,
			Token:        token,
			APIURL:       pushAPIURL,
			AppVersion:   pushAppVersion,
			Description:  pushDescription,
			Mandatory:    pushMandatory,
			Rollout:      pushRollout,
			Disabled:     pushDisabled,
			BundlePath:   bundlePath,
		}

		client := codepush.NewHTTPClient(opts.APIURL, opts.Token)
		result, err := codepush.Push(client, opts)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "\nPush successful:\n")
		fmt.Fprintf(os.Stderr, "  Package ID: %s\n", result.PackageID)
		fmt.Fprintf(os.Stderr, "  App version: %s\n", result.AppVersion)
		fmt.Fprintf(os.Stderr, "  Status: %s\n", result.Status)

		return nil
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to a previous release",
	Long: `Rollback the current deployment to a previous release version.

Reverts the active OTA update so devices receive the prior version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "rollback command not yet implemented")
		return nil
	},
}

var integrateCmd = &cobra.Command{
	Use:   "integrate",
	Short: "Integrate CodePush SDK into a mobile project",
	Long: `Integrate the Bitrise CodePush SDK into your mobile project.

Detects the project type (React Native, Flutter, native iOS/Android)
and configures the SDK accordingly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "integrate command not yet implemented")
		return nil
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  `Manage the Bitrise API token used for CodePush operations.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store a Bitrise API token locally",
	Long: `Store a Bitrise API token in the local config file.

The token is saved to the config directory and used automatically
by commands that require authentication (push, rollback).

Token resolution order: --token flag > BITRISE_API_TOKEN env var > stored config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := authLoginToken
		if token == "" {
			fmt.Fprint(os.Stderr, "Enter API token: ")
			var input string
			if _, err := fmt.Scanln(&input); err != nil {
				return fmt.Errorf("reading token from stdin: %w", err)
			}
			token = input
		}

		if token == "" {
			return fmt.Errorf("token is required: provide --token or enter interactively")
		}

		fmt.Fprintf(os.Stderr, "Validating token...\n")
		if err := auth.ValidateToken(token); err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}

		if err := auth.SaveToken(token); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}

		configPath, _ := auth.ConfigFilePath()
		fmt.Fprintf(os.Stderr, "Token saved to: %s\n", configPath)
		return nil
	},
}

var authRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Remove the stored API token",
	Long: `Remove the locally stored Bitrise API token.

After revoking, commands that require authentication will need
a --token flag or BITRISE_API_TOKEN environment variable.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.RemoveToken(); err != nil {
			return fmt.Errorf("removing token: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Token revoked successfully.\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(bundleCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(integrateCmd)
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authRevokeCmd)

	// Auth login flags
	authLoginCmd.Flags().StringVar(&authLoginToken, "token", "", "Bitrise API token")

	// Bundle command flags
	bundleCmd.Flags().StringVar(&bundlePlatform, "platform", "", "target platform: ios or android (required)")
	_ = bundleCmd.MarkFlagRequired("platform")
	bundleCmd.Flags().StringVar(&bundleEntryFile, "entry-file", "", "path to the entry JS file (auto-detected if not set)")
	bundleCmd.Flags().StringVar(&bundleOutputDir, "output-dir", "./codepush-bundle", "output directory for the bundle")
	bundleCmd.Flags().StringVar(&bundleBundleName, "bundle-name", "", "custom bundle filename (platform default if not set)")
	bundleCmd.Flags().BoolVar(&bundleDev, "dev", false, "enable development mode")
	bundleCmd.Flags().BoolVar(&bundleSourcemap, "sourcemap", true, "generate source maps")
	bundleCmd.Flags().StringVar(&bundleHermes, "hermes", "auto", "Hermes bytecode compilation: auto, on, or off")
	bundleCmd.Flags().StringArrayVar(&bundleExtraBundlerOpts, "extra-bundler-option", nil, "additional flags passed to the bundler (repeatable)")
	bundleCmd.Flags().StringVar(&bundleProjectDir, "project-dir", "", "project root directory (defaults to current directory)")
	bundleCmd.Flags().StringVar(&bundleMetroConfig, "config", "", "path to Metro config file (auto-detected if not set)")

	// Push command: --bundle flag and shared bundling flags
	pushCmd.Flags().BoolVar(&pushAutoBundle, "bundle", false, "bundle JavaScript before pushing")
	pushCmd.Flags().StringVar(&bundlePlatform, "platform", "", "target platform for bundling: ios or android")
	pushCmd.Flags().StringVar(&bundleOutputDir, "output-dir", "./codepush-bundle", "output directory for the bundle")
	pushCmd.Flags().StringVar(&bundleHermes, "hermes", "auto", "Hermes bytecode compilation: auto, on, or off")
	pushCmd.Flags().StringVar(&bundleProjectDir, "project-dir", "", "project root directory (defaults to current directory)")

	// Push command: API flags
	pushCmd.Flags().StringVar(&pushAppID, "app-id", "", "connected app UUID (env: CODEPUSH_APP_ID)")
	pushCmd.Flags().StringVar(&pushDeployment, "deployment", "", "deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	pushCmd.Flags().StringVar(&pushToken, "token", "", "Bitrise API token (env: BITRISE_API_TOKEN, or use 'auth login')")
	pushCmd.Flags().StringVar(&pushAppVersion, "app-version", "", "target app version (e.g. 1.0.0)")
	pushCmd.Flags().StringVar(&pushDescription, "description", "", "update description")
	pushCmd.Flags().BoolVar(&pushMandatory, "mandatory", false, "mark update as mandatory")
	pushCmd.Flags().IntVar(&pushRollout, "rollout", 100, "rollout percentage (1-100)")
	pushCmd.Flags().BoolVar(&pushDisabled, "disabled", false, "disable update after upload")
	pushCmd.Flags().StringVar(&pushAPIURL, "api-url", "https://api.bitrise.io/release-management", "API base URL")
}

// resolveFlag returns the flag value if non-empty, otherwise falls back to the environment variable.
func resolveFlag(flagValue, envKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envKey)
}

// resolveToken returns the API token using the priority:
// 1. --token flag value
// 2. BITRISE_API_TOKEN environment variable
// 3. Stored config file token (from 'codepush auth login')
func resolveToken(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if envValue := os.Getenv("BITRISE_API_TOKEN"); envValue != "" {
		return envValue
	}
	storedToken, _ := auth.LoadToken()
	return storedToken
}

func runBundle() error {
	platform := bundler.Platform(bundlePlatform)
	if platform != bundler.PlatformIOS && platform != bundler.PlatformAndroid {
		return fmt.Errorf("--platform must be 'ios' or 'android', got %q", bundlePlatform)
	}

	hermesMode := bundler.HermesMode(bundleHermes)
	if hermesMode != bundler.HermesModeAuto && hermesMode != bundler.HermesModeOn && hermesMode != bundler.HermesModeOff {
		return fmt.Errorf("--hermes must be 'auto', 'on', or 'off', got %q", bundleHermes)
	}

	result, err := runBundleWithOpts()
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nBundle created successfully:\n")
	fmt.Fprintf(os.Stderr, "  Output: %s\n", result.OutputDir)
	fmt.Fprintf(os.Stderr, "  Bundle: %s\n", result.BundlePath)
	if result.SourcemapPath != "" {
		fmt.Fprintf(os.Stderr, "  Sourcemap: %s\n", result.SourcemapPath)
	}
	if result.HermesApplied {
		fmt.Fprintf(os.Stderr, "  Hermes: compiled\n")
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
	}

	return bundler.Run(opts)
}

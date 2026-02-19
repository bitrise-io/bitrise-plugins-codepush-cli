// CodePush CLI - Manage CodePush OTA updates and SDK integration for mobile apps
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/auth"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	commit  = "none"
	date    = "unknown"
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
)

// Push-only flags (not shared with bundle command).
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

// Auth flags.
var authLoginToken string

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
			return fmt.Errorf("push failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "\nPush successful:\n")
		fmt.Fprintf(os.Stderr, "  Package ID: %s\n", result.PackageID)
		fmt.Fprintf(os.Stderr, "  App version: %s\n", result.AppVersion)
		fmt.Fprintf(os.Stderr, "  Status: %s\n", result.Status)

		if bitrise.IsBitriseEnvironment() {
			exportPushSummary(result)
		}

		return nil
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to a previous release",
	Long: `Rollback the current deployment to a previous release version.

Reverts the active OTA update so devices receive the prior version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("rollback command is not yet implemented")
	},
}

var integrateCmd = &cobra.Command{
	Use:   "integrate",
	Short: "Integrate CodePush SDK into a mobile project",
	Long: `Integrate the Bitrise CodePush SDK into your mobile project.

Detects the project type (React Native, Flutter, native iOS/Android)
and configures the SDK accordingly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("integrate command is not yet implemented")
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

Generate a personal access token at: ` + auth.TokenGenerationURL + `

Token resolution order: --token flag > BITRISE_API_TOKEN env var > stored config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := authLoginToken
		if token == "" {
			fmt.Fprintf(os.Stderr, "\n  Generate a token at: %s\n\n", auth.TokenGenerationURL)
			fmt.Fprint(os.Stderr, "  Paste your personal access token: ")
			input, err := auth.ReadTokenSecure()
			if err != nil {
				return err
			}
			token = input
		}

		if token == "" {
			return fmt.Errorf("token is required: provide --token flag or enter interactively")
		}

		fmt.Fprintf(os.Stderr, "Validating token...")
		userInfo, err := auth.ValidateToken(token)
		if err != nil {
			fmt.Fprintf(os.Stderr, " failed\n")
			return fmt.Errorf("token validation failed: %w\n\n  Generate a new token at: %s", err, auth.TokenGenerationURL)
		}
		fmt.Fprintf(os.Stderr, " done\n")

		if err := auth.SaveToken(token); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}

		if userInfo != nil && userInfo.Username != "" {
			fmt.Fprintf(os.Stderr, "Logged in as %s", userInfo.Username)
			if userInfo.Email != "" {
				fmt.Fprintf(os.Stderr, " (%s)", userInfo.Email)
			}
			fmt.Fprintln(os.Stderr)
		}

		configPath, err := auth.ConfigFilePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not determine config path: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Token saved to: %s\n", configPath)
		}
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
	bundleCmd.Flags().StringVar(&bundleOutputDir, "output-dir", bundler.DefaultOutputDir, "output directory for the bundle")
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
	pushCmd.Flags().StringVar(&bundleOutputDir, "output-dir", bundler.DefaultOutputDir, "output directory for the bundle")
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
	storedToken, err := auth.LoadToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load stored token: %v\n", err)
	}
	return storedToken
}

func runBundle() error {
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

	fmt.Fprintf(os.Stderr, "\nBundle created successfully:\n")
	fmt.Fprintf(os.Stderr, "  Output: %s\n", result.OutputDir)
	fmt.Fprintf(os.Stderr, "  Bundle: %s\n", result.BundlePath)
	if result.SourcemapPath != "" {
		fmt.Fprintf(os.Stderr, "  Sourcemap: %s\n", result.SourcemapPath)
	}
	if result.HermesApplied {
		fmt.Fprintf(os.Stderr, "  Hermes: compiled\n")
	}

	if bitrise.IsBitriseEnvironment() {
		exportBundleSummary(result)
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

// exportPushSummary writes a JSON push summary to the Bitrise deploy directory.
func exportPushSummary(result *codepush.PushResult) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal push summary: %v\n", err)
		return
	}

	path, err := bitrise.WriteToDeployDir("codepush-push-summary.json", data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to export push summary: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Push summary exported to: %s\n", path)
}

// exportBundleSummary writes a JSON bundle summary to the Bitrise deploy directory.
func exportBundleSummary(result *bundler.BundleResult) {
	summary := struct {
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
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal bundle summary: %v\n", err)
		return
	}

	path, err := bitrise.WriteToDeployDir("codepush-bundle-summary.json", data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to export bundle summary: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Bundle summary exported to: %s\n", path)
}

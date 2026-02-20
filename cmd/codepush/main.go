// CodePush CLI - Manage CodePush OTA updates and SDK integration for mobile apps
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/auth"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bundler"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var (
	version = "0.1.0"
	commit  = "none"
	date    = "unknown"
)

// out is the shared CLI output writer, initialized in main().
var out *output.Writer

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

// Shared API flags (persistent on root)
var (
	globalAppID string
	globalJSON  bool
)

const defaultAPIURL = "https://api.bitrise.io/release-management"

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

// Rollback command flags
var (
	rollbackDeployment    string
	rollbackTargetRelease string
)

// Promote command flags
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

// Patch command flags
var (
	patchDeployment  string
	patchLabel       string
	patchRollout     string
	patchMandatory   string
	patchDisabled    string
	patchDescription string
	patchAppVersion  string
)

// Deployment command flags
var (
	deploymentRenameName string
	deploymentRemoveYes  bool
	deploymentHistoryMax int
)

// Package command flags
var (
	packageLabel     string
	packageRemoveYes bool
)

// Deployment clear flag
var deploymentClearYes bool

// Auth flags.
var authLoginToken string

func main() {
	out = output.New()

	if err := rootCmd.Execute(); err != nil {
		out.Error("%v", err)
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
		out.Println("CodePush CLI %s", version)
		out.Info("commit: %s", commit)
		out.Info("built: %s", date)
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

		appID := resolveFlag(globalAppID, "CODEPUSH_APP_ID")
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
		result, err := codepush.Push(client, opts, out)
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

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to a previous release",
	Long: `Rollback the current deployment to a previous release version.

Creates a new release that mirrors a previous version. By default,
rolls back to the immediately previous release. Use --target-release
to specify a specific version label (e.g. v3).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appID := resolveFlag(globalAppID, "CODEPUSH_APP_ID")
		deployment := resolveFlag(rollbackDeployment, "CODEPUSH_DEPLOYMENT")
		token := resolveToken()

		opts := &codepush.RollbackOptions{
			AppID:        appID,
			DeploymentID: deployment,
			Token:        token,
			TargetLabel:  rollbackTargetRelease,
		}

		client := codepush.NewHTTPClient(defaultAPIURL, opts.Token)
		result, err := codepush.Rollback(client, opts, out)
		if err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}

		if globalJSON {
			return outputJSON(result)
		}

		out.Success("Rollback successful")
		out.Result([]output.KeyValue{
			{Key: "Package ID", Value: result.PackageID},
			{Key: "Label", Value: result.Label},
			{Key: "App version", Value: result.AppVersion},
		})

		if bitrise.IsBitriseEnvironment() {
			exportEnvVars(map[string]string{
				"CODEPUSH_PACKAGE_ID":  result.PackageID,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			})
		}

		return nil
	},
}

var promoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote a release from one deployment to another",
	Long: `Promote a release from a source deployment to a destination deployment.

Copies the latest (or specified) release from the source deployment to the
destination deployment. Override metadata like rollout percentage, mandatory
flag, or description for the promoted release.

Example: promote from Staging to Production after testing.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		appID := resolveFlag(globalAppID, "CODEPUSH_APP_ID")
		sourceDeployment := resolveFlag(promoteSourceDeployment, "CODEPUSH_DEPLOYMENT")
		token := resolveToken()

		opts := &codepush.PromoteOptions{
			AppID:              appID,
			SourceDeploymentID: sourceDeployment,
			DestDeploymentID:   promoteDestDeployment,
			Token:              token,
			Label:              promoteLabel,
			AppVersion:         promoteAppVersion,
			Description:        promoteDescription,
			Mandatory:          promoteMandatory,
			Disabled:           promoteDisabled,
			Rollout:            promoteRollout,
		}

		client := codepush.NewHTTPClient(defaultAPIURL, opts.Token)
		result, err := codepush.Promote(client, opts, out)
		if err != nil {
			return fmt.Errorf("promote failed: %w", err)
		}

		if globalJSON {
			return outputJSON(result)
		}

		out.Success("Promote successful")
		out.Result([]output.KeyValue{
			{Key: "Package ID", Value: result.PackageID},
			{Key: "Label", Value: result.Label},
			{Key: "App version", Value: result.AppVersion},
			{Key: "Destination", Value: promoteDestDeployment},
		})

		if bitrise.IsBitriseEnvironment() {
			exportEnvVars(map[string]string{
				"CODEPUSH_PACKAGE_ID":  result.PackageID,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			})
		}

		return nil
	},
}

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
	RunE: func(cmd *cobra.Command, args []string) error {
		appID := resolveFlag(globalAppID, "CODEPUSH_APP_ID")
		deployment := resolveFlag(patchDeployment, "CODEPUSH_DEPLOYMENT")
		token := resolveToken()

		opts := &codepush.PatchOptions{
			AppID:        appID,
			DeploymentID: deployment,
			Token:        token,
			Label:        patchLabel,
			Rollout:      patchRollout,
			Mandatory:    patchMandatory,
			Disabled:     patchDisabled,
			Description:  patchDescription,
			AppVersion:   patchAppVersion,
		}

		client := codepush.NewHTTPClient(defaultAPIURL, opts.Token)
		result, err := codepush.Patch(client, opts, out)
		if err != nil {
			return fmt.Errorf("patch failed: %w", err)
		}

		if globalJSON {
			return outputJSON(result)
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
			exportDeploySummary("codepush-patch-summary.json", result)
			exportEnvVars(map[string]string{
				"CODEPUSH_PACKAGE_ID":  result.PackageID,
				"CODEPUSH_LABEL":       result.Label,
				"CODEPUSH_APP_VERSION": result.AppVersion,
			})
		}

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
			out.Println("")
			out.Info("Generate a token at: %s", auth.TokenGenerationURL)
			out.Println("")
			out.Println("  Paste your personal access token: ")
			input, err := auth.ReadTokenSecure()
			if err != nil {
				return fmt.Errorf("reading token: %w", err)
			}
			token = input
		}

		if token == "" {
			return fmt.Errorf("token is required: provide --token flag or enter interactively")
		}

		var userInfo *auth.UserInfo
		err := out.Spinner("Validating token", func() error {
			var valErr error
			userInfo, valErr = auth.ValidateToken(token)
			return valErr
		})
		if err != nil {
			return fmt.Errorf("token validation failed: %w\n\n  Generate a new token at: %s", err, auth.TokenGenerationURL)
		}

		if err := auth.SaveToken(token); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}

		if userInfo != nil && userInfo.Username != "" {
			if userInfo.Email != "" {
				out.Success("Logged in as %s (%s)", userInfo.Username, userInfo.Email)
			} else {
				out.Success("Logged in as %s", userInfo.Username)
			}
		}

		configPath, err := auth.ConfigFilePath()
		if err != nil {
			out.Warning("could not determine config path: %v", err)
		} else {
			out.Info("Token saved to: %s", configPath)
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

		out.Success("Token revoked successfully")
		return nil
	},
}

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Manage deployments",
	Long:  `Create, list, inspect, rename, and delete CodePush deployments.`,
}

var deploymentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployments",
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)
		deployments, err := client.ListDeployments(appID)
		if err != nil {
			return fmt.Errorf("listing deployments: %w", err)
		}

		if globalJSON {
			return outputJSON(deployments)
		}

		if len(deployments) == 0 {
			out.Info("No deployments found.")
			return nil
		}

		rows := make([][]string, len(deployments))
		for i, d := range deployments {
			rows[i] = []string{d.Name, d.ID}
		}
		out.Table([]string{"NAME", "ID"}, rows)

		return nil
	},
}

var deploymentAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)
		dep, err := client.CreateDeployment(appID, codepush.CreateDeploymentRequest{Name: args[0]})
		if err != nil {
			return fmt.Errorf("creating deployment: %w", err)
		}

		if globalJSON {
			return outputJSON(dep)
		}

		out.Success("Deployment %q created (ID: %s)", dep.Name, dep.ID)
		return nil
	},
}

var deploymentInfoCmd = &cobra.Command{
	Use:   "info <deployment>",
	Short: "Show deployment details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(client, appID, args[0], out)
		if err != nil {
			return err
		}

		dep, err := client.GetDeployment(appID, deploymentID)
		if err != nil {
			return fmt.Errorf("getting deployment: %w", err)
		}

		packages, err := client.ListPackages(appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		if globalJSON {
			info := struct {
				codepush.Deployment
				LatestPackage *codepush.Package `json:"latest_package,omitempty"`
			}{Deployment: *dep}
			if len(packages) > 0 {
				info.LatestPackage = &packages[len(packages)-1]
			}
			return outputJSON(info)
		}

		out.Step("Deployment: %s", dep.Name)
		pairs := []output.KeyValue{
			{Key: "ID", Value: dep.ID},
		}
		if dep.Key != "" {
			pairs = append(pairs, output.KeyValue{Key: "Key", Value: dep.Key})
		}
		if dep.CreatedAt != "" {
			pairs = append(pairs, output.KeyValue{Key: "Created", Value: dep.CreatedAt})
		}
		out.Result(pairs)

		if len(packages) > 0 {
			latest := packages[len(packages)-1]
			out.Step("Latest release")
			out.Result([]output.KeyValue{
				{Key: "Label", Value: latest.Label},
				{Key: "App version", Value: latest.AppVersion},
				{Key: "Mandatory", Value: fmt.Sprintf("%v", latest.Mandatory)},
				{Key: "Rollout", Value: fmt.Sprintf("%d%%", latest.Rollout)},
			})
		} else {
			out.Info("No releases.")
		}

		return nil
	},
}

var deploymentRenameCmd = &cobra.Command{
	Use:   "rename <deployment>",
	Short: "Rename a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}
		if deploymentRenameName == "" {
			return fmt.Errorf("new name is required: set --name")
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(client, appID, args[0], out)
		if err != nil {
			return err
		}

		dep, err := client.RenameDeployment(appID, deploymentID, codepush.RenameDeploymentRequest{Name: deploymentRenameName})
		if err != nil {
			return fmt.Errorf("renaming deployment: %w", err)
		}

		if globalJSON {
			return outputJSON(dep)
		}

		out.Success("Deployment renamed to %q", dep.Name)
		return nil
	},
}

var deploymentRemoveCmd = &cobra.Command{
	Use:   "remove <deployment>",
	Short: "Delete a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		if err := out.ConfirmDestructive(
			fmt.Sprintf("This will permanently delete deployment %q and all its releases", args[0]),
			deploymentRemoveYes,
		); err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(client, appID, args[0], out)
		if err != nil {
			return err
		}

		if err := client.DeleteDeployment(appID, deploymentID); err != nil {
			return fmt.Errorf("deleting deployment: %w", err)
		}

		if globalJSON {
			return outputJSON(struct {
				Deleted string `json:"deleted"`
			}{Deleted: deploymentID})
		}

		out.Success("Deployment %q deleted", args[0])
		return nil
	},
}

var deploymentHistoryCmd = &cobra.Command{
	Use:   "history <deployment>",
	Short: "Show release history for a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(client, appID, args[0], out)
		if err != nil {
			return err
		}

		packages, err := client.ListPackages(appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		// Apply limit: show the most recent entries
		if deploymentHistoryMax > 0 && len(packages) > deploymentHistoryMax {
			packages = packages[len(packages)-deploymentHistoryMax:]
		}

		if globalJSON {
			return outputJSON(packages)
		}

		if len(packages) == 0 {
			out.Info("No releases found.")
			return nil
		}

		headers := []string{"LABEL", "APP VERSION", "MANDATORY", "ROLLOUT", "DISABLED", "DESCRIPTION", "CREATED"}
		rows := make([][]string, len(packages))
		for i, p := range packages {
			rows[i] = []string{
				p.Label, p.AppVersion, fmt.Sprintf("%v", p.Mandatory),
				fmt.Sprintf("%d%%", p.Rollout), fmt.Sprintf("%v", p.Disabled),
				truncate(p.Description, 30), p.CreatedAt,
			}
		}
		out.Table(headers, rows)

		return nil
	},
}

var packageCmd = &cobra.Command{
	Use:   "package",
	Short: "Inspect packages (releases)",
	Long:  `View details and processing status of CodePush packages.`,
}

var packageInfoCmd = &cobra.Command{
	Use:   "info <deployment>",
	Short: "Show package details",
	Long: `Show details for a specific package in a deployment.

By default shows the latest package. Use --label to specify a version.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(client, appID, args[0], out)
		if err != nil {
			return err
		}

		packageID, _, err := codepush.ResolvePackageForPatch(client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		pkg, err := client.GetPackage(appID, deploymentID, packageID)
		if err != nil {
			return fmt.Errorf("getting package: %w", err)
		}

		if globalJSON {
			return outputJSON(pkg)
		}

		out.Step("Package: %s", pkg.Label)
		pairs := []output.KeyValue{
			{Key: "ID", Value: pkg.ID},
			{Key: "App version", Value: pkg.AppVersion},
			{Key: "Mandatory", Value: fmt.Sprintf("%v", pkg.Mandatory)},
			{Key: "Disabled", Value: fmt.Sprintf("%v", pkg.Disabled)},
			{Key: "Rollout", Value: fmt.Sprintf("%d%%", pkg.Rollout)},
		}
		if pkg.Description != "" {
			pairs = append(pairs, output.KeyValue{Key: "Description", Value: pkg.Description})
		}
		pairs = append(pairs, output.KeyValue{Key: "Size", Value: formatBytes(pkg.FileSizeBytes)})
		if pkg.Hash != "" {
			pairs = append(pairs, output.KeyValue{Key: "Hash", Value: pkg.Hash})
		}
		if pkg.CreatedAt != "" {
			pairs = append(pairs, output.KeyValue{Key: "Created", Value: pkg.CreatedAt})
		}
		if pkg.CreatedBy != "" {
			pairs = append(pairs, output.KeyValue{Key: "Created by", Value: pkg.CreatedBy})
		}
		out.Result(pairs)

		return nil
	},
}

var packageStatusCmd = &cobra.Command{
	Use:   "status <deployment>",
	Short: "Show package processing status",
	Long: `Show the processing status of a specific package.

By default shows the latest package. Use --label to specify a version.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(client, appID, args[0], out)
		if err != nil {
			return err
		}

		packageID, packageLabel, err := codepush.ResolvePackageForPatch(client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		status, err := client.GetPackageStatus(appID, deploymentID, packageID)
		if err != nil {
			return fmt.Errorf("getting package status: %w", err)
		}

		if globalJSON {
			return outputJSON(status)
		}

		pairs := []output.KeyValue{
			{Key: "Package", Value: packageLabel},
			{Key: "Status", Value: status.Status},
		}
		if status.StatusReason != "" {
			pairs = append(pairs, output.KeyValue{Key: "Reason", Value: status.StatusReason})
		}
		out.Result(pairs)

		return nil
	},
}

var packageRemoveCmd = &cobra.Command{
	Use:   "remove <deployment>",
	Short: "Delete a package from a deployment",
	Long: `Delete a specific package from a deployment.

Requires --label to identify the package and --yes to confirm deletion.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}
		if packageLabel == "" {
			return fmt.Errorf("label is required: set --label to identify the package to delete")
		}

		if err := out.ConfirmDestructive(
			fmt.Sprintf("This will permanently delete package %q", packageLabel),
			packageRemoveYes,
		); err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(client, appID, args[0], out)
		if err != nil {
			return err
		}

		packageID, _, err := codepush.ResolvePackageForPatch(client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		if err := client.DeletePackage(appID, deploymentID, packageID); err != nil {
			return fmt.Errorf("deleting package: %w", err)
		}

		if globalJSON {
			return outputJSON(struct {
				Deleted string `json:"deleted"`
				Label   string `json:"label"`
			}{Deleted: packageID, Label: packageLabel})
		}

		out.Success("Package %q deleted", packageLabel)
		return nil
	},
}

var deploymentClearCmd = &cobra.Command{
	Use:   "clear <deployment>",
	Short: "Delete all packages from a deployment",
	Long: `Delete all packages (releases) from a deployment.

This is a destructive operation that removes all release history.
Requires --yes to confirm.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		if err := out.ConfirmDestructive(
			fmt.Sprintf("This will permanently delete all releases from %q", args[0]),
			deploymentClearYes,
		); err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(client, appID, args[0], out)
		if err != nil {
			return err
		}

		packages, err := client.ListPackages(appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		if len(packages) == 0 {
			out.Info("No packages to delete.")
			return nil
		}

		deleted := 0
		for _, pkg := range packages {
			if err := client.DeletePackage(appID, deploymentID, pkg.ID); err != nil {
				return fmt.Errorf("deleting package %s: %w", pkg.Label, err)
			}
			deleted++
		}

		if globalJSON {
			return outputJSON(struct {
				Deployment string `json:"deployment"`
				Deleted    int    `json:"deleted"`
			}{Deployment: deploymentID, Deleted: deleted})
		}

		out.Success("Deleted %d package(s) from %q", deleted, args[0])
		return nil
	},
}

// truncate shortens a string to max length, appending "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// formatBytes returns a human-readable byte size.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return strconv.FormatInt(b, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	registerCommandTree()
	registerGlobalFlags()
	registerBundleFlags()
	registerPushFlags()
	registerRollbackFlags()
	registerPromoteFlags()
	registerPatchFlags()
	registerDeploymentFlags()
	registerPackageFlags()
	registerAuthFlags()
}

func registerCommandTree() {
	rootCmd.AddGroup(
		&cobra.Group{ID: "release", Title: "Release Management:"},
		&cobra.Group{ID: "deployment", Title: "Deployment Management:"},
		&cobra.Group{ID: "package", Title: "Package Management:"},
		&cobra.Group{ID: "setup", Title: "Setup:"},
	)

	pushCmd.GroupID = "release"
	rollbackCmd.GroupID = "release"
	promoteCmd.GroupID = "release"
	patchCmd.GroupID = "release"
	bundleCmd.GroupID = "release"
	deploymentCmd.GroupID = "deployment"
	packageCmd.GroupID = "package"
	authCmd.GroupID = "setup"
	integrateCmd.GroupID = "setup"

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(bundleCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(promoteCmd)
	rootCmd.AddCommand(patchCmd)
	rootCmd.AddCommand(integrateCmd)
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authRevokeCmd)
	rootCmd.AddCommand(deploymentCmd)
	deploymentCmd.AddCommand(deploymentListCmd)
	deploymentCmd.AddCommand(deploymentAddCmd)
	deploymentCmd.AddCommand(deploymentInfoCmd)
	deploymentCmd.AddCommand(deploymentRenameCmd)
	deploymentCmd.AddCommand(deploymentRemoveCmd)
	deploymentCmd.AddCommand(deploymentHistoryCmd)
	deploymentCmd.AddCommand(deploymentClearCmd)
	rootCmd.AddCommand(packageCmd)
	packageCmd.AddCommand(packageInfoCmd)
	packageCmd.AddCommand(packageStatusCmd)
	packageCmd.AddCommand(packageRemoveCmd)
}

func registerGlobalFlags() {
	rootCmd.PersistentFlags().StringVar(&globalAppID, "app-id", "", "connected app UUID (env: CODEPUSH_APP_ID)")
	rootCmd.PersistentFlags().BoolVar(&globalJSON, "json", false, "output results as JSON to stdout")
}

func registerBundleFlags() {
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

func registerRollbackFlags() {
	rollbackCmd.Flags().StringVar(&rollbackDeployment, "deployment", "", "deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	rollbackCmd.Flags().StringVar(&rollbackTargetRelease, "target-release", "", "specific release label to rollback to (e.g. v3)")
}

func registerPromoteFlags() {
	promoteCmd.Flags().StringVar(&promoteSourceDeployment, "source-deployment", "", "source deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	promoteCmd.Flags().StringVar(&promoteDestDeployment, "destination-deployment", "", "destination deployment name or UUID (required)")
	promoteCmd.Flags().StringVar(&promoteLabel, "label", "", "specific release label to promote (e.g. v5)")
	promoteCmd.Flags().StringVar(&promoteAppVersion, "app-version", "", "override target app version")
	promoteCmd.Flags().StringVar(&promoteDescription, "description", "", "override release description")
	promoteCmd.Flags().StringVar(&promoteMandatory, "mandatory", "", "override mandatory flag (true/false)")
	promoteCmd.Flags().StringVar(&promoteDisabled, "disabled", "", "override disabled flag (true/false)")
	promoteCmd.Flags().StringVar(&promoteRollout, "rollout", "", "override rollout percentage (1-100)")
}

func registerPatchFlags() {
	patchCmd.Flags().StringVar(&patchDeployment, "deployment", "", "deployment name or UUID (env: CODEPUSH_DEPLOYMENT)")
	patchCmd.Flags().StringVar(&patchLabel, "label", "", "specific release label to patch (e.g. v5, defaults to latest)")
	patchCmd.Flags().StringVar(&patchRollout, "rollout", "", "rollout percentage (1-100)")
	patchCmd.Flags().StringVar(&patchMandatory, "mandatory", "", "mark update as mandatory (true/false)")
	patchCmd.Flags().StringVar(&patchDisabled, "disabled", "", "disable update (true/false)")
	patchCmd.Flags().StringVar(&patchDescription, "description", "", "update description")
	patchCmd.Flags().StringVar(&patchAppVersion, "app-version", "", "target app version")
}

func registerDeploymentFlags() {
	deploymentRenameCmd.Flags().StringVar(&deploymentRenameName, "name", "", "new deployment name (required)")
	deploymentRemoveCmd.Flags().BoolVar(&deploymentRemoveYes, "yes", false, "skip confirmation prompt")
	deploymentHistoryCmd.Flags().IntVar(&deploymentHistoryMax, "limit", 10, "maximum number of releases to show")
	deploymentClearCmd.Flags().BoolVar(&deploymentClearYes, "yes", false, "skip confirmation prompt")
}

func registerPackageFlags() {
	packageInfoCmd.Flags().StringVar(&packageLabel, "label", "", "specific release label (defaults to latest)")
	packageStatusCmd.Flags().StringVar(&packageLabel, "label", "", "specific release label (defaults to latest)")
	packageRemoveCmd.Flags().StringVar(&packageLabel, "label", "", "release label to delete (required)")
	packageRemoveCmd.Flags().BoolVar(&packageRemoveYes, "yes", false, "skip confirmation prompt")
}

func registerAuthFlags() {
	authLoginCmd.Flags().StringVar(&authLoginToken, "token", "", "Bitrise API token")
}

// outputJSON marshals v as JSON to stdout. Used when --json is set.
func outputJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON output: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

// resolveFlag returns the flag value if non-empty, otherwise falls back to the environment variable.
func resolveFlag(flagValue, envKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envKey)
}

// resolveToken returns the API token using the priority:
// 1. BITRISE_API_TOKEN environment variable
// 2. Stored config file token (from 'codepush auth login')
func resolveToken() string {
	if envValue := os.Getenv("BITRISE_API_TOKEN"); envValue != "" {
		return envValue
	}
	storedToken, err := auth.LoadToken()
	if err != nil {
		if out != nil {
			out.Warning("could not load stored token: %v", err)
		}
	}
	return storedToken
}

// requireCredentials resolves and validates the app ID and API token.
func requireCredentials() (appID, token string, err error) {
	appID = resolveFlag(globalAppID, "CODEPUSH_APP_ID")
	token = resolveToken()

	if appID == "" {
		return "", "", fmt.Errorf("app ID is required: set --app-id or CODEPUSH_APP_ID")
	}
	if token == "" {
		return "", "", fmt.Errorf("API token is required: set BITRISE_API_TOKEN or run 'codepush auth login'")
	}
	return appID, token, nil
}

// exportDeploySummary writes a JSON summary to the Bitrise deploy directory.
func exportDeploySummary(filename string, v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		out.Warning("failed to marshal %s: %v", filename, err)
		return
	}

	path, err := bitrise.WriteToDeployDir(filename, data)
	if err != nil {
		out.Warning("failed to export %s: %v", filename, err)
		return
	}

	out.Info("Summary exported to: %s", path)
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
	}

	return bundler.Run(opts, out)
}

// exportEnvVars exports key-value pairs as Bitrise environment variables via envman.
func exportEnvVars(vars map[string]string) {
	for key, value := range vars {
		if err := bitrise.ExportEnvVar(key, value); err != nil {
			out.Warning("failed to export %s: %v", key, err)
		}
	}
}


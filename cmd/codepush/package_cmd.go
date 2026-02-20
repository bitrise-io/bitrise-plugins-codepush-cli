package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Package command flags
var (
	packageLabel     string
	packageRemoveYes bool
)

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

		deploymentID, err := codepush.ResolveDeployment(cmd.Context(), client, appID, args[0], out)
		if err != nil {
			return err
		}

		packageID, _, err := codepush.ResolvePackageForPatch(cmd.Context(), client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		pkg, err := client.GetPackage(cmd.Context(), appID, deploymentID, packageID)
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
			{Key: "Rollout", Value: fmt.Sprintf("%.0f%%", pkg.Rollout)},
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
		if pkg.CreatedBy != nil && pkg.CreatedBy.Email != "" {
			pairs = append(pairs, output.KeyValue{Key: "Created by", Value: pkg.CreatedBy.Email})
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

		deploymentID, err := codepush.ResolveDeployment(cmd.Context(), client, appID, args[0], out)
		if err != nil {
			return err
		}

		packageID, packageLabel, err := codepush.ResolvePackageForPatch(cmd.Context(), client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		status, err := client.GetPackageStatus(cmd.Context(), appID, deploymentID, packageID)
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

		deploymentID, err := codepush.ResolveDeployment(cmd.Context(), client, appID, args[0], out)
		if err != nil {
			return err
		}

		packageID, _, err := codepush.ResolvePackageForPatch(cmd.Context(), client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		if err := client.DeletePackage(cmd.Context(), appID, deploymentID, packageID); err != nil {
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

func registerPackageFlags() {
	packageInfoCmd.Flags().StringVar(&packageLabel, "label", "", "specific release label (defaults to latest)")
	packageStatusCmd.Flags().StringVar(&packageLabel, "label", "", "specific release label (defaults to latest)")
	packageRemoveCmd.Flags().StringVar(&packageLabel, "label", "", "release label to delete (required)")
	packageRemoveCmd.Flags().BoolVar(&packageRemoveYes, "yes", false, "skip confirmation prompt")
}

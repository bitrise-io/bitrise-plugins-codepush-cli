package packagecmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/cmdutil"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var (
	packageLabel     string
	packageRemoveYes bool
)

var packageCmd = &cobra.Command{
	Use:     "package",
	Short:   "Inspect packages (releases)",
	Long:    `View details and processing status of CodePush packages.`,
	GroupID: cmd.GroupPackage,
}

var infoCmd = &cobra.Command{
	Use:   "info [deployment]",
	Short: "Show package details",
	Long: `Show details for a specific package in a deployment.

By default shows the latest package. Use --label to specify a version.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, argValue, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		packageID, _, err := codepush.ResolvePackageForPatch(c.Context(), client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		pkg, err := client.GetPackage(c.Context(), appID, deploymentID, packageID)
		if err != nil {
			return fmt.Errorf("getting package: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(pkg)
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
		pairs = append(pairs, output.KeyValue{Key: "Size", Value: cmdutil.FormatBytes(pkg.FileSizeBytes)})
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

var statusCmd = &cobra.Command{
	Use:   "status [deployment]",
	Short: "Show package processing status",
	Long: `Show the processing status of a specific package.

By default shows the latest package. Use --label to specify a version.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, argValue, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		packageID, pkgLabel, err := codepush.ResolvePackageForPatch(c.Context(), client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		status, err := client.GetPackageStatus(c.Context(), appID, deploymentID, packageID)
		if err != nil {
			return fmt.Errorf("getting package status: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(status)
		}

		pairs := []output.KeyValue{
			{Key: "Package", Value: pkgLabel},
			{Key: "Status", Value: status.Status},
		}
		if status.StatusReason != "" {
			pairs = append(pairs, output.KeyValue{Key: "Reason", Value: status.StatusReason})
		}
		out.Result(pairs)

		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [deployment]",
	Short: "Delete a package from a deployment",
	Long: `Delete a specific package from a deployment.

Requires --label to identify the package and --yes to confirm deletion.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
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

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, argValue, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		packageID, _, err := codepush.ResolvePackageForPatch(c.Context(), client, appID, deploymentID, packageLabel, out)
		if err != nil {
			return err
		}

		if err := client.DeletePackage(c.Context(), appID, deploymentID, packageID); err != nil {
			return fmt.Errorf("deleting package: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(struct {
				Deleted string `json:"deleted"`
				Label   string `json:"label"`
			}{Deleted: packageID, Label: packageLabel})
		}

		out.Success("Package %q deleted", packageLabel)
		return nil
	},
}

func init() {
	cmd.RootCmd.AddGroup(&cobra.Group{ID: cmd.GroupPackage, Title: "Package Management:"})

	infoCmd.Flags().StringVar(&packageLabel, "label", "", "specific release label (defaults to latest)")
	statusCmd.Flags().StringVar(&packageLabel, "label", "", "specific release label (defaults to latest)")
	removeCmd.Flags().StringVar(&packageLabel, "label", "", "release label to delete (required)")
	removeCmd.Flags().BoolVar(&packageRemoveYes, "yes", false, "skip confirmation prompt")

	packageCmd.AddCommand(infoCmd, statusCmd, removeCmd)
	cmd.RootCmd.AddCommand(packageCmd)
}
